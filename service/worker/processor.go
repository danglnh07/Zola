package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/pubsub"
	"github.com/hibiken/asynq"
)

// Task processor interface
type TaskProcessor interface {
	Start() error
	ProcessTaskSendMessage(ctx context.Context, task *asynq.Task) (err error)
}

// Redis task processor
type RedisTaskProcessor struct {
	server  *asynq.Server
	queries *db.Queries
	hub     *pubsub.Hub
	logger  *slog.Logger
}

// Constructor method for Redis task processor
func NewRedisTaskProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	hub *pubsub.Hub,
	logger *slog.Logger,
) TaskProcessor {
	return &RedisTaskProcessor{
		server:  asynq.NewServer(redisOpts, asynq.Config{}),
		queries: queries,
		hub:     hub,
		logger:  logger,
	}
}

// Method to start the worker server
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(SendMessage, processor.ProcessTaskSendMessage)

	return processor.server.Start(mux)
}
