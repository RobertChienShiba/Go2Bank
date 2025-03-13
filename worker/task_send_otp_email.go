package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RobertChienShiba/simplebank/util"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"github.com/xlzd/gotp"
)

const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	Username string `json:"username"`
}

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
		Str("queue", info.Queue).Int("max_retry", info.MaxRetry).Msg("enqueued task")
	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	secret := util.RandomBase32Secret(16)

	totp := gotp.NewTOTP(secret, 6, int(processor.config.APILimitDuration.Seconds()), nil) // Assuming the duration is 30 seconds
	otp := totp.Now()

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	redisKey := fmt.Sprintf("otp:%s", user.Username)

	err = processor.otpStore.Set(redisKey, secret, processor.config.APILimitDuration)
	if err != nil {
		return fmt.Errorf("failed to set otp in redis: %w", err)
	}

	subject := "Transfer OTP"
	content := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>OTP Code</title>
		</head>
		<body>
			<p>Your OTP is: <b>%s</b>. It will expire in <b>%d</b> seconds.</p>
		</body>
		</html>`, otp, int(processor.config.APILimitDuration.Seconds()))
	to := []string{user.Email}

	err = processor.mailer.SendEmail(subject, content, to, nil)
	if err != nil {
		return fmt.Errorf("failed to send verify email: %w", err)
	}

	log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
		Str("email", user.Email).Msg("processed task")
	return nil
}
