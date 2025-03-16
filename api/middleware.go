package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	rds "github.com/RobertChienShiba/simplebank/redis"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/redis/go-redis/v9"
	"github.com/xlzd/gotp"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var accessToken string
		if cookie, err := ctx.Cookie("access_token"); err == nil {
			accessToken = cookie
		} else {
			authorizeHeader := ctx.GetHeader(authorizationHeaderKey)

			if len(authorizeHeader) == 0 {
				err := errors.New("authorization header is not provided")
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
				return
			}
			fields := strings.Fields(authorizeHeader)
			if len(fields) < 2 {
				err := errors.New("invalid authorization header")
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			}

			authorizationType := strings.ToLower(fields[0])
			if authorizationType != authorizationTypeBearer {
				err := fmt.Errorf("unsupported authorization type %s", authorizationType)
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
				return
			}

			accessToken = fields[1]
		}

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

type OTPRetrieve struct {
	OTP string `json:"otp" binding:"required,min=6,max=6,numeric"`
}

func verifyOTPMiddleware(otpStore rds.Store, otpDuration time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req OTPRetrieve
		body, _ := ctx.GetRawData()

		if err := json.Unmarshal(body, &req); err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			ctx.Abort()
			return
		}

		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

		redisKey := fmt.Sprintf("otp:%s", authPayload.Username)

		secret, err := otpStore.Get(redisKey)
		if err != nil {
			err := errors.New("OTP has been expired")
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			ctx.Abort()
			return
		}

		totp := gotp.NewTOTP(secret, 6, int(otpDuration.Seconds()), nil)

		if !totp.VerifyTime(req.OTP, time.Now()) {
			err := errors.New("invalid OTP")
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			ctx.Abort()
			return
		}

		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		ctx.Next()

		otpStore.Del(redisKey)
	}
}

func rateLimitMiddleware(apiName string, limiter rds.Store, maxRequests int64, windowSize time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
		ip := ctx.ClientIP()
		redisKey := fmt.Sprintf("%s:%s:%s", ip, authPayload.Username, apiName)

		currentTimestamp := time.Now().UnixMilli()
		windowStart := currentTimestamp - int64(windowSize.Milliseconds())

		// Delete request records before window start
		limiter.ZRemRangeByScore(redisKey, "0", fmt.Sprintf("%d", windowStart))

		// Count the number of requests in the current window
		count, err := limiter.ZCard(redisKey)
		if err != nil {
			err := errors.New("failed to calculate number of API requests")
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			ctx.Abort()
			return
		}

		windows, _ := limiter.ZRange(redisKey, int64(0), int64(0))
		var firstRequest int64
		if len(windows) > 0 {
			firstRequest, _ = strconv.ParseInt(windows[0], 10, 64)
		} else {
			firstRequest = currentTimestamp
		}
		nextReset := firstRequest + int64(windowSize.Milliseconds())

		ctx.Header("X-RateLimit-Limit", strconv.FormatInt(maxRequests, 10))
		ctx.Header("X-RateLimit-Remaining", strconv.FormatInt(max(int64(0), maxRequests-count), 10))
		ctx.Header("X-RateLimit-Reset", time.Unix(nextReset/1000, 0).Format(time.RFC3339))

		// If the number of requests exceeds the limit, reject the request
		if count > maxRequests {
			ctx.Header("X-RateLimit-Retry-After", strconv.FormatInt(max((nextReset-currentTimestamp)/1000, 0), 10))
			err = errors.New("API Limit Reached")
			ctx.JSON(http.StatusTooManyRequests, errorResponse(err))
			ctx.Abort()
			return
		}

		// Add the timestamp of the current request to the Sorted Set
		limiter.ZAdd(redisKey, redis.Z{
			Score:  float64(time.Now().UnixMilli()),
			Member: time.Now().UnixMilli(),
		})

		// Set expiration time
		limiter.Expire(redisKey, windowSize)

		ctx.Next()
	}
}

func csrfVerifyMiddleware() gin.HandlerFunc {
	csrfMiddleware := csrf.Protect(
		[]byte("32-byte-long-auth-key"),
		csrf.Secure(false),
		csrf.HttpOnly(true),
		// csrf.SameSite(csrf.SameSiteNoneMode),
	)
	return adapter.Wrap(csrfMiddleware)
}

func csrfTokenMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("X-CSRF-Token", csrf.Token(ctx.Request))
	}
}
