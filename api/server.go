package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/pubsub"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Server struct {
	mux     *gin.Engine
	queries *db.Queries

	limiter     *RateLimiter
	jwtService  *security.JWTService
	oauth       OAuth
	upgrader    *websocket.Upgrader
	distributor worker.TaskDistributor
	hub         *pubsub.Hub

	config *util.Config
	logger *slog.Logger
}

func NewServer(
	queries *db.Queries,
	config *util.Config,
	hub *pubsub.Hub,
	distributor worker.TaskDistributor,
	logger *slog.Logger,
) *Server {
	logger.Info("", "Server hub", fmt.Sprintf("%p", hub))

	// Create depenency
	jwtService := security.NewJWTService(config)
	oauth := NewGoogleAuth(queries, jwtService, config, logger)

	return &Server{
		mux:     gin.Default(),
		queries: queries,

		limiter:    NewRateLimiter(config.MaxRequest, config.RefillRate),
		jwtService: jwtService,
		oauth:      oauth,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		distributor: distributor,
		hub:         hub,

		config: config,
		logger: logger,
	}
}

type ErrorResponse struct {
	Message string `json:"error"`
}

// Helper method to register handler to route
func (server *Server) RegisterHandler() {
	// Setup global middlewares
	server.mux.Use(server.CORSMiddlware(), server.RateLimitingMiddleware())

	api := server.mux.Group("/api")
	{
		// Auth routes
		api.GET("/oauth", server.oauth.HandleOAuth)

		// Send messages
		api.POST("/messages", server.AuthMiddleware(), server.HandleSendMessage)

		// Get online users
		api.GET("/users/online", server.AuthMiddleware(), server.HandleGetOnlineUsers)
	}

	// Websocket routes
	ws := server.mux.Group("/ws")
	{
		ws.GET("/messages", server.AuthMiddleware(), server.HandleWS)
	}

	// Callback URL for OAuth2
	server.mux.GET("/oauth2/callback", server.oauth.HandleCallback)
}

// Method to start the server
func (server *Server) Start() error {
	server.RegisterHandler()
	return server.mux.Run(":8080")
}
