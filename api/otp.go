package api

import (
	"net/http"
	"time"

	"github.com/RobertChienShiba/simplebank/token"
	"github.com/RobertChienShiba/simplebank/worker"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

func (server *Server) sendOTP(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	taskPayload := &worker.PayloadSendVerifyEmail{
		Username: authPayload.Username,
	}

	opts := []asynq.Option{
		asynq.MaxRetry(10),
		asynq.ProcessIn(10 * time.Second),
		asynq.Queue(worker.QueueCritical),
	}

	err := server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "OTP sent. Please verify to complete the transfer."})
}
