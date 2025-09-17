package api

import (
	"errors"
	"net/http"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/pubsub"
	"github.com/danglnh07/zola/service/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler for all Web Socket endpoint
func (server *Server) HandleWS(ctx *gin.Context) {
	// Upgrade request from HTTP to Web Socket
	conn, err := server.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		server.logger.Error("failed to upgrade to Web Socket", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Create the client
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	client := pubsub.NewClient(requesterID, conn)

	// Subscribe to the server
	server.hub.Subscribe(client)
	defer server.hub.Unsubscribe(client)

	// Block until client is disconnected
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			server.logger.Info("client disconnected", "id", requesterID, "err", err)
			break
		}
	}
}

type SendMessageRequest struct {
	SenderID   uint   `json:"sender_id" binding:"required"`
	ReceiverID uint   `json:"receiver_id"` // If not provided, it would be a broadcast message
	Content    string `json:"content" binding:"required"`
}

func (server *Server) HandleSendMessage(ctx *gin.Context) {
	// Get the request body and validate
	var req SendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.logger.Error("POST /api/messages: failed to parse request body", "error", err)
		ctx.JSON(http.StatusBadRequest, ErrorResponse{"Invalid request body"})
		return
	}

	// Check if the requester is sender
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	if requesterID != req.SenderID {
		ctx.JSON(http.StatusForbidden, ErrorResponse{"You have no authorization to proceed with this request"})
		return
	}

	// Build the message model
	var message = db.Message{
		Model:    gorm.Model{},
		SenderID: req.SenderID,
		Content:  req.Content,
	}

	var sender db.Account
	result := server.queries.DB.Where("id = ?", req.SenderID).First(&sender)
	if result.Error != nil {
		server.logger.Error("POST /api/messages: failed to get sender from database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}
	message.Sender = sender

	if req.ReceiverID != 0 {
		var receiver db.Account
		result = server.queries.DB.Where("id = ?", req.ReceiverID).First(&receiver)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				ctx.JSON(http.StatusBadRequest, ErrorResponse{"receiver_id not match any account"})
				return
			}

			server.logger.Error("POST /api/messages: failed to fetch receiver from database", "error", result.Error)
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
			return
		}
		receiverID := req.ReceiverID
		message.ReceiverID = &receiverID
		message.Receiver = &receiver
		message.ChatType = db.PrivateChat
	} else {
		message.ReceiverID = nil
		message.Receiver = nil
		message.ChatType = db.PublicChat
	}

	// Add message to database
	result = server.queries.DB.Create(&message)
	if result.Error != nil {
		server.logger.Error("POST /api/messages: failed to create message in database", "error", result.Error)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Publish the send message event through hub
	err := server.distributor.DistributeTaskSendMessage(ctx, message)
	if err != nil {
		server.logger.Error("POST/api/messages: failed to create background task send message", "error", err)
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	ctx.JSON(http.StatusCreated, "Message sent successfully")
}

func (server *Server) HandleGetOnlineUsers(ctx *gin.Context) {
	var users []UserData

	for _, client := range server.hub.Clients {
		var user db.Account
		result := server.queries.DB.Select("id", "username", "email").Where("id = ?", client.AccountID).First(&user)
		if result.Error != nil {
			server.logger.Error("GET /api/users/online: failed to fetch user data from database", "error", result.Error)
			continue
		}

		users = append(users, UserData{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		})
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"total": len(server.hub.Clients),
		"users": users,
	})
}
