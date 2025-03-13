package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/gin-gonic/gin"
)

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {

	sID, err := ctx.Cookie("sid")
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("refresh token has been expired")))
		return
	}

	ex, err := server.kvStore.Exists("sid:" + sID)
	if err != nil || ex <= 0 {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("refresh token has been expired")))
		return
	}

	username, err := server.kvStore.Get("sid:" + sID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	storedUserAgent, _ := server.kvStore.Get("sid:" + sID + ":user_agent")
	storedIP, _ := server.kvStore.Get("sid:" + sID + ":ip")

	currentUserAgent := ctx.GetHeader("User-Agent")
	currentIP := ctx.ClientIP()
	fmt.Println("current_ip", currentIP)
	fmt.Println("store_ip", storedIP)

	if storedUserAgent != currentUserAgent {
		err := errors.New("mismatched user agent")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if storedIP != currentIP {
		err := errors.New("mismatched IP")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err1 := server.kvStore.Del("sid:" + sID)
	err2 := server.kvStore.Del("sid:" + sID + ":user_agent")
	err3 := server.kvStore.Del("sid:" + sID + ":ip")
	if err1 != nil || err2 != nil || err3 != nil {
		err := errors.New("failed to delete session in redis when renewing access token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	sID = util.RandomID()
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("sid", sID, 3600*24, "/", "", false, true)

	err1 = server.kvStore.Set("sid:"+sID, user.Username, server.config.RefreshTokenDuration)
	err2 = server.kvStore.Set("sid:"+sID+":user_agent", ctx.Request.UserAgent(), server.config.RefreshTokenDuration)
	err3 = server.kvStore.Set("sid:"+sID+":ip", ctx.ClientIP(), server.config.RefreshTokenDuration)
	if err1 != nil || err2 != nil || err3 != nil {
		err := errors.New("failed to set session in redis when renewing access token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}
