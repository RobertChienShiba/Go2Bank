package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockss "github.com/RobertChienShiba/simplebank/redis/mock"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/RobertChienShiba/simplebank/worker"
	mockwk "github.com/RobertChienShiba/simplebank/worker/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSendOTP(t *testing.T) {
	username := util.RandomOwner()

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockss.MockStore, distributor *mockwk.MockTaskDistributor)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore, distributor *mockwk.MockTaskDistributor) {
				// rate limit middleware
				store.EXPECT().ZRemRangeByScore(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().ZCard(gomock.Any()).Times(1).Return(int64(3), nil)
				store.EXPECT().ZRange(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{}, nil)
				store.EXPECT().ZAdd(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().Expire(gomock.Any(), gomock.Any()).Times(1)

				taskPayload := &worker.PayloadSendVerifyEmail{
					Username: username,
				}
				distributor.EXPECT().DistributeTaskSendVerifyEmail(
					gomock.Any(),
					taskPayload,
					gomock.Any(),
				).Times(1).Return(nil)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "Server Error",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, username, util.DepositorRole, time.Minute)
			},
			buildStubs: func(store *mockss.MockStore, distributor *mockwk.MockTaskDistributor) {
				// rate limit middleware
				store.EXPECT().ZRemRangeByScore(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().ZCard(gomock.Any()).Times(1).Return(int64(3), nil)
				store.EXPECT().ZRange(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{}, nil)
				store.EXPECT().ZAdd(gomock.Any(), gomock.Any()).Times(1).Return(nil)
				store.EXPECT().Expire(gomock.Any(), gomock.Any()).Times(1)

				distributor.EXPECT().DistributeTaskSendVerifyEmail(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Times(1).Return(errors.New("failed to enqueue task"))

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			kvCtrl := gomock.NewController(t)
			defer kvCtrl.Finish()

			distributorCtrl := gomock.NewController(t)
			defer distributorCtrl.Finish()

			kvStore := mockss.NewMockStore(kvCtrl)
			distributor := mockwk.NewMockTaskDistributor(distributorCtrl)

			tc.buildStubs(kvStore, distributor)

			server := newTestServer(t, nil, kvStore, distributor)

			url := "/transfers/sendOTP"
			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
