package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
)

type TaskDistributor interface {
	DistributeTaskSendEmail(ctx context.Context, payload Payload, opts ...asynq.Option) (err error)
}

type RedisTaskDistributor struct {
	client *asynq.Client
	logger *slog.Logger
}

func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt, logger *slog.Logger) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
		logger: logger,
	}
}
