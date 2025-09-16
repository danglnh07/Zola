package worker

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"strings"

	"github.com/hibiken/asynq"
)

// Payload struct for send email job
type EmailPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Send email key
const SendEmail = "send-welcome-email"

//go:embed welcome.html
var welcome embed.FS

// Method to distribute send email task
func (distributor *RedisTaskDistributor) DistributeTaskSendEmail(
	ctx context.Context,
	payload EmailPayload,
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

// Method to process send email task
func (processor *RedisTaskProcessor) ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) (err error) {
	processor.logger.Info("Start processing task", "task_name", SendEmail)

	// Unmarshal payload
	var payload EmailPayload
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
