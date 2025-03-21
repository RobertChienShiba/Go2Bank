package worker

import (
	"context"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	"github.com/RobertChienShiba/simplebank/mail"
	rds "github.com/RobertChienShiba/simplebank/redis"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type TaskProcessor interface {
	Start() error
	Shutdown()
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	config   util.Config
	server   *asynq.Server
	store    db.Store
	otpStore rds.Store
	mailer   mail.EmailSender
}

func NewRedisTaskProcessor(config util.Config, redisOpt asynq.RedisClientOpt, store db.Store, otpStore rds.Store, mailer mail.EmailSender) TaskProcessor {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				QueueCritical: 10,
				QueueDefault:  5,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error().Err(err).Str("type", task.Type()).
					Bytes("payload", task.Payload()).Msg("process task failed")
			}),
		},
	)

	return &RedisTaskProcessor{
		config:   config,
		server:   server,
		store:    store,
		otpStore: otpStore,
		mailer:   mailer,
	}
}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}

func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}
