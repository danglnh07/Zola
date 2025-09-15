package api

import (
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/danglnh07/zola/service/worker"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
)

type Server struct {
	mux         *gin.Engine
	queries     *db.Queries
	jwtService  *security.JWTService
	distributor worker.TaskDistributor
	oauth       OAuth
	limiter     *RateLimiter
	config      *util.Config
	logger      *slog.Logger
}

type ErrorResponse struct {
	Message string `json:"error"`
}

func NewServer(
	queries *db.Queries,
	distributor worker.TaskDistributor,
	config *util.Config,
	logger *slog.Logger,
) *Server {
	jwtService := security.NewJWTService(config)

	return &Server{
		mux:         gin.Default(),
		queries:     queries,
		jwtService:  jwtService,
		distributor: distributor,
		oauth:       NewGoogleAuth(queries, distributor, jwtService, config, logger),
		limiter:     NewRateLimiter(config.MaxRequest, config.RefillRate),
		config:      config,
		logger:      logger,
	}
}

func (server *Server) RegisterHandler() {
	// Setup global middlewares
	server.mux.Use(server.CORSMiddlware(), server.RateLimitingMiddleware())

	api := server.mux.Group("/api")
	{
		// Auth routes
		api.POST("/auth/register", server.HandleRegister)
		api.POST("/auth/login", server.HandleLogin)
		api.POST("/auth/token/refresh", server.AuthMiddleware(), server.HandleRefreshToken)
		api.POST("/auth/logout", server.AuthMiddleware(), server.HandleLogout)
		api.GET("/oauth", server.oauth.HandleOAuth)
	}

	// Callback URL
	server.mux.GET("/oauth2/callback", server.oauth.HandleCallback)
}

func (server *Server) Start() error {
	server.RegisterHandler()
	return server.mux.Run(":8080")
}
