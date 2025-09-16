package worker

import (
	"context"
	"encoding/json"

	"github.com/danglnh07/zola/db"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

// Payload struct for send notification job
type NotificationPayload struct {
	SourceID uint   `json:"source_id"`
	DestID   uint   `json:"dest_id"`
	Content  string `json:"content"`
}

// Send notification key
const SendNotification = "send-notification"

// Method to distribute send notification task
func (distributor *RedisTaskDistributor) DistributeTaskSendNotification(
	ctx context.Context,
	payload NotificationPayload,
	opts ...asynq.Option,
) (err error) {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create new task
	task := asynq.NewTask(SendNotification, data, opts...)

	// Send task to Redis queue
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}

	// Log task info
	distributor.logger.Info("Task info", "task_name", SendNotification, "queue", info.Queue, "max_retry", info.MaxRetry)

	return nil
}

// Method to process send notification task
func (processor *RedisTaskProcessor) ProcessTaskSendNotification(ctx context.Context, task *asynq.Task) (err error) {
	processor.logger.Info("Start processing task", "task_name", SendNotification)

	// Unmarshal payload
	var payload NotificationPayload
	if err = json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	// Insert notification into database first
	var notification = db.Notification{
		Model:    gorm.Model{},
		SourceID: payload.SourceID,
		DestID:   payload.DestID,
		Content:  payload.Content,
		Status:   db.Unread,
	}
	result := processor.queries.DB.Create(&notification)
	if result.Error != nil {
		return result.Error
	}
	processor.logger.Info("Insert notification successfully", "content", notification.Content)

	// Publish event through hub
	processor.hub.Publish(notification)

	return nil
}
