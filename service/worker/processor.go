package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/mail"
	"github.com/hibiken/asynq"
)

type TaskProcessor interface {
	Start() error
	ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) (err error)
}

type RedisTaskProcessor struct {
	server      *asynq.Server
	queries     *db.Queries
	logger      *slog.Logger
	mailService *mail.EmailService
}

func NewRedisTaskProcessor(
	redisOpts asynq.RedisClientOpt,
	queries *db.Queries,
	mailService *mail.EmailService,
	logger *slog.Logger,
) TaskProcessor {
	return &RedisTaskProcessor{
		server:      asynq.NewServer(redisOpts, asynq.Config{}),
		queries:     queries,
		mailService: mailService,
		logger:      logger,
	}
}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(SendEmail, processor.ProcessTaskSendEmail)

	return processor.server.Start(mux)
}
