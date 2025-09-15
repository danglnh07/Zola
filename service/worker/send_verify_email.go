package worker

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"strings"

	"github.com/hibiken/asynq"
)

type Payload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

const SendEmail = "send-welcome-email"

//go:embed welcome.html
var welcome embed.FS

func (distributor *RedisTaskDistributor) DistributeTaskSendEmail(
	ctx context.Context,
	payload Payload,
	opts ...asynq.Option,
) (err error) {
	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create new task
	task := asynq.NewTask(SendEmail, data, opts...)

	// Send task to Redis queue
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}

	// Log task info
	distributor.logger.Info("Task info", "task_name", SendEmail, "queue", info.Queue, "max_retry", info.MaxRetry)

	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) (err error) {
	processor.logger.Info("Start processing task", "task_name", SendEmail)

	// Unmarshal payload
	var payload Payload
	if err = json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	// Prepare HTML email body
	tmpl, err := template.ParseFS(welcome, "welcome.html")
	if err != nil {
		return err
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, payload)
	if err != nil {
		return err
	}

	// Send email
	return processor.mailService.SendEmail(payload.Email, "Welcome to Zola", sb.String())
}
