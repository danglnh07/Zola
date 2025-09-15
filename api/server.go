package api

import (
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/util"
	"github.com/gin-gonic/gin"
)

type Server struct {
	mux     *gin.Engine
	queries *db.Queries
	config  *util.Config
	logger  *slog.Logger
}

type ErrorResponse struct {
	Message string `json:"error"`
}

func NewServer(queries *db.Queries, config *util.Config, logger *slog.Logger) *Server {
	return &Server{
		mux:     gin.Default(),
		queries: queries,
		config:  config,
		logger:  logger,
	}
}

func (server *Server) RegisterHandler() {

}

func (server *Server) Start() error {
	server.RegisterHandler()
	return server.mux.Run(":8080")
}
