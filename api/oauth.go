package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/gin-gonic/gin"
	"github.com/google/go-querystring/query"
)

func (server *Server) googleOAuth(ctx *gin.Context) {
	code := ctx.Query("code")
	var pathUrl string = "/"

	if ctx.Query("state") != "" {
		pathUrl = ctx.Query("state")
	}

	if code == "" {
		err := errors.New("authorization code not provided")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	tokenRes, err := GetGoogleOauthToken(server.config, code)

	if err != nil {
		ctx.JSON(http.StatusBadGateway, errorResponse(err))
		return
	}

	googleUser, err := GetGoogleUser(tokenRes.Access_token, tokenRes.Id_token)

	if err != nil {
		ctx.JSON(http.StatusBadGateway, errorResponse(err))
		return
	}

	arg := db.UpsertUserParams{
		Username:       googleUser.Name + "-google",
		FullName:       googleUser.Given_name + " " + googleUser.Family_name,
		Email:          googleUser.Email,
		HashedPassword: "",
		Provider:       "Google",
	}

	user, err := server.store.UpsertUser(ctx, arg)
	if err != nil {
		errCode := db.ErrorCode(err)
		if errCode == db.UniqueViolation {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	accessToken, _, err := server.tokenMaker.CreateToken(user.Username, user.Role, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	sID := util.RandomID()
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("sid", sID, int(server.config.RefreshTokenDuration.Seconds()), "/", "", false, true)

	err1 := server.kvStore.Set("sid:"+sID, user.Username, server.config.RefreshTokenDuration)
	err2 := server.kvStore.Set("sid:"+sID+":user_agent", ctx.Request.UserAgent(), server.config.RefreshTokenDuration)
	err3 := server.kvStore.Set("sid:"+sID+":ip", ctx.ClientIP(), server.config.RefreshTokenDuration)

	if err1 != nil || err2 != nil || err3 != nil {
		err := errors.New("failed to set session in redis when logging in")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.SetCookie("access_token", accessToken, int(server.config.AccessTokenDuration.Seconds()), "/", "", false, true)

	ctx.Redirect(http.StatusTemporaryRedirect, fmt.Sprint(server.config.AllowedOrigins[0], pathUrl))
}

type GoogleOauthToken struct {
	Access_token string
	Id_token     string
}

type GoogleUserResult struct {
	Id             string
	Email          string
	Verified_email bool
	Name           string
	Given_name     string
	Family_name    string
	Picture        string
	Locale         string
}

func GetGoogleOauthToken(config util.Config, code string) (*GoogleOauthToken, error) {
	const rootURl = "https://oauth2.googleapis.com/token"

	options := struct {
		GrantType    string `url:"grant_type"`
		Code         string `url:"code"`
		ClientID     string `url:"client_id"`
		ClientSecret string `url:"client_secret"`
		RedirectURI  string `url:"redirect_uri"`
	}{
		GrantType:    "authorization_code",
		Code:         code,
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURI:  config.GoogleOAuthRedirectUrl,
	}

	values, err := query.Values(options)
	if err != nil {
		return nil, err
	}

	query := values.Encode()

	req, err := http.NewRequest("POST", rootURl, bytes.NewBufferString(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{
		Timeout: time.Second * 30,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("could not retrieve token")
	}

	var resBody bytes.Buffer
	_, err = io.Copy(&resBody, res.Body)
	if err != nil {
		return nil, err
	}

	var GoogleOauthTokenRes map[string]interface{}

	if err := json.Unmarshal(resBody.Bytes(), &GoogleOauthTokenRes); err != nil {
		return nil, err
	}

	tokenBody := &GoogleOauthToken{
		Access_token: GoogleOauthTokenRes["access_token"].(string),
		Id_token:     GoogleOauthTokenRes["id_token"].(string),
	}

	return tokenBody, nil
}

func GetGoogleUser(access_token string, id_token string) (*GoogleUserResult, error) {
	rootUrl := fmt.Sprintf("https://www.googleapis.com/oauth2/v1/userinfo?alt=json&access_token=%s", access_token)

	req, err := http.NewRequest("GET", rootUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", id_token))

	client := http.Client{
		Timeout: time.Second * 30,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("could not retrieve user")
	}

	var resBody bytes.Buffer
	_, err = io.Copy(&resBody, res.Body)
	if err != nil {
		return nil, err
	}

	var googleUserRes map[string]interface{}

	if err := json.Unmarshal(resBody.Bytes(), &googleUserRes); err != nil {
		return nil, err
	}

	fmt.Printf("%#v", googleUserRes)

	userBody := &GoogleUserResult{
		Email:       googleUserRes["email"].(string),
		Name:        googleUserRes["name"].(string),
		Given_name:  googleUserRes["given_name"].(string),
		Family_name: googleUserRes["family_name"].(string),
	}

	return userBody, nil
}

func (server *Server) testGoogleOAuth(ctx *gin.Context) {
	from := ctx.DefaultQuery("from", "/api/users/logout")
	rootUrl := "https://accounts.google.com/o/oauth2/v2/auth"

	options := struct {
		RedirectURI  string `url:"redirect_uri"`
		ClientID     string `url:"client_id"`
		AccessType   string `url:"access_type"`
		ResponseType string `url:"response_type"`
		Prompt       string `url:"prompt"`
		Scope        string `url:"scope"`
		State        string `url:"state"`
	}{
		RedirectURI:  server.config.GoogleOAuthRedirectUrl,
		ClientID:     server.config.GoogleClientID,
		AccessType:   "offline",
		ResponseType: "code",
		Prompt:       "consent",
		Scope:        "https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email",
		State:        from,
	}

	v, err := query.Values(options)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	redirectUrl := fmt.Sprintf("%s?%s", rootUrl, v.Encode())

	ctx.Redirect(http.StatusFound, redirectUrl)
}
