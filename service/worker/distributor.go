package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
)

// Task distributor interface
type TaskDistributor interface {
	DistributeTaskSendEmail(ctx context.Context, payload EmailPayload, opts ...asynq.Option) (err error)
	DistributeTaskSendNotification(ctx context.Context, payload NotificationPayload, opts ...asynq.Option) (err error)
}

// Redis task distributor
type RedisTaskDistributor struct {
	client *asynq.Client
	logger *slog.Logger
}

// Constructor method for Redis task distributor
func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt, logger *slog.Logger) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
		logger: logger,
	}
}
