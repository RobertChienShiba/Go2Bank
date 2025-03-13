package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	rds "github.com/RobertChienShiba/simplebank/redis"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/RobertChienShiba/simplebank/worker"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config          util.Config
	tokenMaker      token.Maker
	store           db.Store
	kvStore         rds.Store
	taskDistributor worker.TaskDistributor
	router          *gin.Engine
}

func NewServer(config util.Config, store db.Store, kvStore rds.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker %w", err)
	}
	server := &Server{
		config:          config,
		store:           store,
		kvStore:         kvStore,
		taskDistributor: taskDistributor,
		tokenMaker:      tokenMaker,
	}
	router := gin.Default()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.GET("/tokens/renew_access", server.renewAccessToken)
	router.GET("/users/logout", server.logoutUser)

	authRoutes := router.Group("/").Use(authMiddleware(tokenMaker))

	authRoutes.PATCH("/users/update", server.updateUser)
	authRoutes.GET("/users/me", server.getUser)

	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccount)
	authRoutes.GET("/accounts", server.listAccounts)

	authRoutes.GET("/transfers/sendOTP",
		rateLimitMiddleware("sendOTP", kvStore, config.APILimitBound, config.APILimitDuration),
		server.sendOTP,
	)

	authRoutes.POST("/transfers",
		rateLimitMiddleware("verifyOTP", kvStore, config.APILimitBound, config.APILimitDuration),
		verifyOTPMiddleware(kvStore, config.APILimitDuration),
		server.createTransfer,
	)

	router.GET("/stress_test",
		func(ctx *gin.Context) {
			testPayload := &token.Payload{
				Username: "test",
			}
			ctx.Set("authorization_payload", testPayload)
			ctx.Next()
		},
		rateLimitMiddleware("test", kvStore, int64(1000), 30*time.Second),
	)

	server.router = router
	return server, nil
}

// Start runs the HTTP server on a specific address
func (server *Server) New(address string) *http.Server {

	return &http.Server{
		Handler: server.router,
		Addr:    address,
	}
	// return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
