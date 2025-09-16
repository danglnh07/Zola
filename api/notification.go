package api

import (
	"fmt"
	"net/http"

	"github.com/danglnh07/zola/service/security"
	"github.com/gin-gonic/gin"
)

// Handler for SSE, used for notification
func (server *Server) SSEHandler(ctx *gin.Context) {
	// Set header to allow SSE streaming
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")

	// Change writer to flusher or streaming
	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		server.logger.Error("SSE handler: failed to type asertion from writer to flusher")
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{"Internal server error"})
		return
	}

	// Subscribe to the hub
	claims, _ := ctx.Get(claimsKey)
	requesterID := claims.(*security.CustomClaims).ID
	subscriber := server.hub.Subscribe()
	defer server.hub.Unsubscribe(subscriber)

	// Read and send message to client
	for noti := range subscriber {
		// Filter to check if the requester is allow to get this notification
		if noti.DestID == requesterID {
			server.logger.Info(fmt.Sprintf("Notification: %s", noti.Content))
			fmt.Fprintf(ctx.Writer, "data: %s\n\n", noti.Content)
			flusher.Flush()
		} else {
			server.logger.Warn("This notification not belong to you")
		}
	}
}
