package main

import (
	"log/slog"
	"os"

	"github.com/danglnh07/zola/api"
	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/mail"
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

	// Connect to Redis
	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddr,
	}
	distributor := worker.NewRedisTaskDistributor(redisOpt, logger)

	// Run the task processor in goroutine (since the asynq.Start will block the main thread)
	go func(opts asynq.RedisClientOpt, query *db.Queries, cfg *util.Config) {
		// Create services that will be used by the worker
		mailService := mail.NewEmailService(cfg)

		// Create the processor
		processor := worker.NewRedisTaskProcessor(opts, query, mailService, logger)

		// Start process tasks
		if err := processor.Start(); err != nil {
			logger.Error("failed to run task processor", "error", err)
			os.Exit(1)
		}
	}(redisOpt, queries, config)

	// Create and start server
	server := api.NewServer(queries, distributor, config, logger)
	if err = server.Start(); err != nil {
		logger.Error("Failed to run the server or server shutdown unexpectedly", "error", err)
		os.Exit(1)
	}
}
