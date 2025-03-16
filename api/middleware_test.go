package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockss "github.com/RobertChienShiba/simplebank/redis/mock"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/xlzd/gotp"
)

func addCSRFToken(
	t *testing.T,
	csrfReq *http.Request,
	apiReq *http.Request,
	router *gin.Engine,
) {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, csrfReq)

	// require.Equal(t, http.StatusOK, w.Code)

	token := w.Header().Get("X-CSRF-Token")
	require.NotEmpty(t, token)

	apiReq.Header.Set("X-CSRF-Token", token)
	apiReq.Header.Set(authorizationHeaderKey, csrfReq.Header.Get(authorizationHeaderKey))

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "_gorilla_csrf" {
			apiReq.AddCookie(cookie)
			return
		}
	}
}

func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	role string,
	duration time.Duration,
) {
	token, payload, err := tokenMaker.CreateToken(username, role, duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}

func TestAuthMiddleware(t *testing.T) {
	username := util.RandomOwner()

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "unsupported", username, util.DepositorRole, time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationFormat",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "", username, util.DepositorRole, time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, -time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, nil, nil, nil)
			authPath := "/auth"
			server.router.GET(
				authPath,
				authMiddleware(server.tokenMaker),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	username := util.RandomOwner()

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockss.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// rate limit middleware
				store.EXPECT().ZRemRangeByScore(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().ZCard(gomock.Any()).Times(1).Return(int64(3), nil)
				store.EXPECT().ZRange(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{}, nil)
				store.EXPECT().ZAdd(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().Expire(gomock.Any(), gomock.Any()).Times(1)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "Exceed Threshold",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// rate limit middleware
				store.EXPECT().ZRemRangeByScore(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().ZCard(gomock.Any()).Times(1).Return(int64(10), nil)
				store.EXPECT().ZRange(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{}, nil)
				store.EXPECT().ZAdd(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().Expire(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusTooManyRequests, recorder.Code)
			},
		},
		{
			name: "Can't calculate number of requests",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// rate limit middleware
				store.EXPECT().ZRemRangeByScore(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().ZCard(gomock.Any()).Times(1).Return(int64(3), errors.New("failed to calculate number of API requests"))
				store.EXPECT().ZRange(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().ZAdd(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().Expire(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			kvStore := mockss.NewMockStore(ctrl)
			tc.buildStubs(kvStore)

			server := newTestServer(t, nil, kvStore, nil)
			rateLimitPath := "/middleware/test/rate_limit"
			server.router.GET(
				rateLimitPath,
				authMiddleware(server.tokenMaker),
				rateLimitMiddleware("testRateLimit", server.kvStore, 5, 1*time.Minute),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, rateLimitPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
func TestVerifyOTPMiddleware(t *testing.T) {
	username := util.RandomOwner()

	testSecret := util.RandomBase32Secret(16)
	totp := gotp.NewTOTP(testSecret, 6, 180, nil)
	testOTP := totp.Now()

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockss.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"otp": testOTP,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// verify otp middleware
				store.EXPECT().Get(gomock.Any()).Times(1).Return(testSecret, nil)

				// expire otp
				store.EXPECT().Del(gomock.Any()).Times(1)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "OTP Expired",
			body: gin.H{
				"otp": testOTP,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// verify otp middleware
				store.EXPECT().Get(gomock.Any()).Times(1).Return(testSecret, redis.Nil)

				// expire otp
				store.EXPECT().Del(gomock.Any()).Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Invalid OTP",
			body: gin.H{
				"otp": testOTP,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore) {
				// verify otp middleware
				store.EXPECT().Get(gomock.Any()).Times(1).Return(util.RandomBase32Secret(16), nil)

				// expire otp
				store.EXPECT().Del(gomock.Any()).Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			kvStore := mockss.NewMockStore(ctrl)
			tc.buildStubs(kvStore)

			server := newTestServer(t, nil, kvStore, nil)

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			verifyPath := "/middleware/test/verify_otp"
			server.router.POST(
				verifyPath,
				authMiddleware(server.tokenMaker),
				verifyOTPMiddleware(server.kvStore, 3*time.Minute),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			request, err := http.NewRequest(http.MethodPost, verifyPath, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
