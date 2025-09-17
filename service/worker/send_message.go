package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/danglnh07/zola/db"
	"github.com/hibiken/asynq"
)

const SendMessage = "send-message"

func (distributor *RedisTaskDistributor) DistributeTaskSendMessage(
	ctx context.Context,
	payload db.Message,
	opts ...asynq.Option,
) (err error) {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create new task
	task := asynq.NewTask(SendMessage, data, opts...)

	// Send task to Redis queue
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}

	// Log task info
	distributor.logger.Info("Task info", "task_name", SendMessage, "queue", info.Queue, "max_retry", info.MaxRetry)

	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendMessage(ctx context.Context, task *asynq.Task) (err error) {
	processor.logger.Info("Start processing task", "task name", SendMessage)

	// Unmarshal payload
	var message db.Message
	if err := json.Unmarshal(task.Payload(), &message); err != nil {
		return err
	}

	processor.logger.Info("", "Clients size", len(processor.hub.Clients))
	processor.logger.Info("", "Processor hub", fmt.Sprintf("%p", processor.hub))

	// Check if this is a broadcast message or a private message
	var success int
	switch message.ChatType {
	case db.PublicChat:
		// Send the message to all online client
		for _, client := range processor.hub.Clients {
			if err := client.WriteMessage(message); err != nil {
				processor.logger.Error(fmt.Sprintf("Failed to send message %d to client %d", message.ID, client.AccountID), "error", err)
				continue
			}
			processor.logger.Info(fmt.Sprintf("Message %d sent to client %d successfully", message.ID, client.AccountID), "error", err)
			success++
		}
		processor.logger.Info(fmt.Sprintf("%d / %d message sent success", success, len(processor.hub.Clients)))
	case db.PrivateChat:
		if client, ok := processor.hub.Clients[*message.ReceiverID]; ok {
			return client.WriteMessage(message)
		}

		processor.logger.Info(fmt.Sprintf("Receiver %d currently offline, changed to send notification", *message.ReceiverID))
		// Process with notification
	}

	processor.logger.Info("Task completed successfully", "task name", SendMessage)

	return nil
}
