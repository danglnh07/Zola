package api

import (
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/notify"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
)

// Server struct, which holds the router, config and all dependencies
type Server struct {
	mux         *gin.Engine
	queries     *db.Queries
	jwtService  *security.JWTService
	distributor worker.TaskDistributor
	hub         *notify.Hub
	oauth       OAuth
	limiter     *RateLimiter
	config      *util.Config
	logger      *slog.Logger
}

// Universal error response struct
type ErrorResponse struct {
	Message string `json:"error"`
}

// Constructor method for Server struct
func NewServer(
	queries *db.Queries,
	distributor worker.TaskDistributor,
	hub *notify.Hub,
	config *util.Config,
	logger *slog.Logger,
) *Server {
	jwtService := security.NewJWTService(config)

	return &Server{
		mux:         gin.Default(),
		queries:     queries,
		jwtService:  jwtService,
		distributor: distributor,
		hub:         hub,
		oauth:       NewGoogleOAuth(queries, distributor, jwtService, config, logger),
		limiter:     NewRateLimiter(config.MaxRequest, config.RefillRate),
		config:      config,
		logger:      logger,
	}
}

// Helper method to register handler to route
func (server *Server) RegisterHandler() {
	// Setup global middlewares
	server.mux.Use(server.CORSMiddlware(), server.RateLimitingMiddleware())

	api := server.mux.Group("/api")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", server.HandleRegister)
			auth.POST("/login", server.HandleLogin)
			auth.POST("/token/refresh", server.AuthMiddleware(), server.HandleRefreshToken)
			auth.POST("/logout", server.AuthMiddleware(), server.HandleLogout)
		}
		api.GET("/oauth", server.oauth.HandleOAuth)

		// Friends routes
		friend := api.Group("/friends", server.AuthMiddleware())
		{
			friend.POST("", server.HandleAddFriend)
			friend.POST("/:id", server.HandleUpdateFriendshipStatus)
		}

		// Notification routes
		api.GET("/notification/stream", server.AuthMiddleware(), server.SSEHandler)
	}

	// Callback URL
	server.mux.GET("/oauth2/callback", server.oauth.HandleCallback)
}

// Method to start the server
func (server *Server) Start() error {
	server.RegisterHandler()
	return server.mux.Run(":8080")
}
