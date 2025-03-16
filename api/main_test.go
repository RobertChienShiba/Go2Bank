package api

import (
	"os"
	"testing"
	"time"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	rds "github.com/RobertChienShiba/simplebank/redis"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/RobertChienShiba/simplebank/worker"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store, sessionStore rds.Store, taskDistributor worker.TaskDistributor) *Server {
	config := util.Config{
		AllowedOrigins:      []string{"*"},
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		APILimitBound:       int64(10),
		APILimitDuration:    5 * time.Minute,
	}

	server, err := NewServer(config, store, sessionStore, taskDistributor)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}
