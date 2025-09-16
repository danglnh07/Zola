package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Request struct for sending friend request
type AddFriendRequest struct {
	SenderID   uint `json:"sender_id" binding:"required"`
	ReceiverID uint `json:"receiver_id" binding:"required"`
}

// Handler for sending friend request
func (server *Server) HandleAddFriend(ctx *gin.Context) {
	// Get request body and validate
	var req AddFriendRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if the sender is the requester of this request
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	if requesterID != req.SenderID {
		ctx.JSON(http.StatusForbidden, ErrorResponse{"You don't have authorization on this action"})
		return
	}

	// Check if the sender is sending a friend request to themselves
	if req.SenderID == req.ReceiverID {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Cannot send a friend request to yourself"})
		return
	}

	// Check if receiver ID exists
	var receiver db.Account
	result := server.queries.DB.Where("id = ?", req.ReceiverID).First(&receiver)
	if result.Error != nil {
		// If not found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"Receiver ID not exists"})
			return
		}

		// Other database errors
		server.logger.Error("POST /api/friends: failed to fetch receiver", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Create friendship with status pending
	var friendship = db.Friendship{
		Model:       gorm.Model{},
		RequesterID: req.SenderID,
		AddresseeID: req.ReceiverID,
		Status:      db.Pending,
	}
	result = server.queries.DB.Create(&friendship)
	if result.Error != nil {
		server.logger.Error("POST /api/friends: failed to create friend request", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return successful message back to client
	ctx.JSON(http.StatusCreated, "Send request successfully")

	// Notify the other side
	var requester db.Account
	result = server.queries.DB.Where("id = ?", friendship.RequesterID).First(&requester)
	if result.Error != nil {
		server.logger.Error("POST /api/friends: failed to get requester information", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	err := server.distributor.DistributeTaskSendNotification(context.Background(), worker.NotificationPayload{
		SourceID: friendship.RequesterID,
		DestID:   friendship.AddresseeID,
		Content:  fmt.Sprintf("%s has sent a friend request", requester.Username),
	})

	if err != nil {
		server.logger.Error("POST /api/friends: Failed to create task: send notification", "error", err)
	}
}

// Handler for updating friendship status
func (server *Server) HandleUpdateFriendshipStatus(ctx *gin.Context) {
	// Get the status in query parameter
	status := db.FriendshipStatus(ctx.Query("status"))
	if status != db.Accepted && status != db.Rejected {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid value for status"})
		return
	}

	// Get the friend request ID in path parameter
	friendRequestID := ctx.Param("id")

	// Get the friend request in database
	var friendship db.Friendship
	result := server.queries.DB.
		Where("id = ?", friendRequestID).
		Preload("Addressee").
		Preload("Requester").
		First(&friendship)
	if result.Error != nil {
		// If ID not match any record
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{"No friend request with this ID"})
			return
		}

		// Other database errors
		server.logger.Error("POST /api/friends/:id: failed to fetch friendship", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Check if the current requester has the authorization to update this status
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	if requesterID != friendship.AddresseeID {
		ctx.JSON(http.StatusForbidden, ErrorResponse{"You have no authorize to proceed with this request"})
		return
	}

	// Update the status
	friendship.Status = status
	result = server.queries.DB.Save(&friendship)
	if result.Error != nil {
		server.logger.Error("POST /api/friends/:id: failed to update friendship status", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Return successful message back to client
	ctx.JSON(http.StatusOK, "Updated successfully")

	// Notify the other side about the change
	err := server.distributor.DistributeTaskSendNotification(context.Background(), worker.NotificationPayload{
		SourceID: friendship.AddresseeID,
		DestID:   friendship.RequesterID,
		Content:  fmt.Sprintf("%s has %s your friend request", friendship.Addressee.Username, status),
	})
	if err != nil {
		server.logger.Error("POST /api/friends/:id: failed to create task: send notification", "error", err)
	}
}
