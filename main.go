package main

import (
	"log/slog"
	"os"

	"github.com/danglnh07/zola/api"
	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/util"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config from .env
	config := util.LoadConfig(".env")

	// Connect to database
	queries, err := db.NewQueries(config)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Run auto migration
	if err = queries.AutoMigration(); err != nil {
		logger.Error("Failed to run auto migration", "error", err)
		os.Exit(1)
	}

	// Create and start server
	server := api.NewServer(queries, config, logger)
	if err = server.Start(); err != nil {
		logger.Error("Failed to run the server or server shutdown unexpectedly", "error", err)
		os.Exit(1)
	}
}
