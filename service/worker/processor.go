package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/mail"
	"github.com/danglnh07/zola/service/notify"
	"github.com/hibiken/asynq"
)

// Task processor interface
type TaskProcessor interface {
	Start() error
	ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) (err error)
	ProcessTaskSendNotification(ctx context.Context, task *asynq.Task) (err error)
}

// Redis task processor
type RedisTaskProcessor struct {
	server      *asynq.Server
	queries     *db.Queries
	mailService *mail.EmailService
	hub         *notify.Hub
	logger      *slog.Logger
}

// Constructor method for Redis task processor
func NewRedisTaskProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService *mail.EmailService,
	hub *notify.Hub,
	logger *slog.Logger,
) TaskProcessor {
	return &RedisTaskProcessor{
		server:      asynq.NewServer(redisOpts, asynq.Config{}),
		queries:     queries,
		mailService: mailService,
		hub:         hub,
		logger:      logger,
	}
}

// Method to start the worker server
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(SendEmail, processor.ProcessTaskSendEmail)
	mux.HandleFunc(SendNotification, processor.ProcessTaskSendNotification)

	return processor.server.Start(mux)
}
