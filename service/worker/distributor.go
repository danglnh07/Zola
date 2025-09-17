package worker

import (
	"context"
	"log/slog"

	"github.com/danglnh07/zola/db"
	"github.com/hibiken/asynq"
)

// Task distributor interface
type TaskDistributor interface {
	DistributeTaskSendMessage(ctx context.Context, payload db.Message, opts ...asynq.Option) (err error)
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
