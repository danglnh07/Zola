package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/danglnh07/zola/api"
	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/pubsub"
	"github.com/danglnh07/zola/service/worker"
	"github.com/danglnh07/zola/util"
	"github.com/hibiken/asynq"
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

	// Create the hub
	hub := pubsub.NewHub()
	logger.Info("", "Main hub", fmt.Sprintf("%p", hub))

	// Connect to Redis
	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddr,
	}
	distributor := worker.NewRedisTaskDistributor(redisOpt, logger)

	// Run the task processor in goroutine (since the asynq.Start will block the main thread)
	go StartBackgroundProcessor(redisOpt, queries, hub, logger)

	// Create and start server
	server := api.NewServer(queries, config, hub, distributor, logger)
	if err = server.Start(); err != nil {
		logger.Error("Failed to run the server or server shutdown unexpectedly", "error", err)
		os.Exit(1)
	}
}

func StartBackgroundProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	hub *pubsub.Hub,
	logger *slog.Logger,
) error {
	logger.Info("", "Start background process hub", fmt.Sprintf("%p", hub))

	// Create the processor
	processor := worker.NewRedisTaskProcessor(redisOpts, queries, hub, logger)

	// Start process tasks
	return processor.Start()
}
