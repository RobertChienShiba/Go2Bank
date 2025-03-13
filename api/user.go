package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/RobertChienShiba/simplebank/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type userResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	// Bind the request body to the req variable
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Hash the user's password
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		FullName:       req.FullName,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		errCode := db.ErrorCode(err)
		if errCode == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	customResponse := newUserResponse(user)
	ctx.JSON(http.StatusCreated, customResponse)
}

type getUserResponse struct {
	Username          string       `json:"username"`
	FullName          string       `json:"full_name"`
	Email             string       `json:"email"`
	PasswordChangedAt time.Time    `json:"password_changed_at"`
	CreatedAt         time.Time    `json:"created_at"`
	Accounts          []db.Account `json:"accounts"`
}

func (server *Server) getUser(ctx *gin.Context) {

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.store.GetUser(ctx, authPayload.Username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.ListAccountsParams{
		Owner:  user.Username,
		Limit:  int32(5),
		Offset: 0,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	customResponse := getUserResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
		Accounts:          []db.Account{},
	}

	for _, account := range accounts {
		customResponse.Accounts = append(customResponse.Accounts, db.Account{
			ID:       account.ID,
			Owner:    account.Owner,
			Currency: account.Currency,
			Balance:  account.Balance,
		})
	}

	ctx.JSON(http.StatusOK, customResponse)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	User                 userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	// Bind the request body to the req variable
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(req.Username, user.Role, server.config.AccessTokenDuration)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	sID := util.RandomID()
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("sid", sID, 3600*24, "/", "", false, true)

	err1 := server.kvStore.Set("sid:"+sID, user.Username, server.config.RefreshTokenDuration)
	err2 := server.kvStore.Set("sid:"+sID+":user_agent", ctx.Request.UserAgent(), server.config.RefreshTokenDuration)
	err3 := server.kvStore.Set("sid:"+sID+":ip", ctx.ClientIP(), server.config.RefreshTokenDuration)

	if err1 != nil || err2 != nil || err3 != nil {
		err := errors.New("failed to set session in redis when logging in")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := loginUserResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
		User:                 newUserResponse(user),
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) logoutUser(ctx *gin.Context) {
	sID, err := ctx.Cookie("sid")
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	err1 := server.kvStore.Del("sid:" + sID)
	err2 := server.kvStore.Del("sid:" + sID + ":user_agent")
	err3 := server.kvStore.Del("sid:" + sID + ":ip")
	if err1 != nil || err2 != nil || err3 != nil {
		err := errors.New("failed to delete session in redis when logging out")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.SetCookie("sid", "", -1, "/", "", false, true)
	ctx.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

type UpdateUserRequest struct {
	Username string  `json:"username" binding:"required,alphanum"`
	FullName *string `json:"full_name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

func (x *UpdateUserRequest) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *UpdateUserRequest) GetFullName() string {
	if x != nil && x.FullName != nil {
		return *x.FullName
	}
	return ""
}

func (x *UpdateUserRequest) GetEmail() string {
	if x != nil && x.Email != nil {
		return *x.Email
	}
	return ""
}

func (x *UpdateUserRequest) GetPassword() string {
	if x != nil && x.Password != nil {
		return *x.Password
	}
	return ""
}

func (server *Server) updateUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	permission := roleCheck(authPayload.Role, []string{util.BankerRole, util.DepositorRole})
	if !permission {
		err := errors.New("no permission")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}
	var req UpdateUserRequest
	// Bind the request body to the req variable
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if invalidRequest := validateUpdateUserRequest(&req); invalidRequest.Violations != nil {
		ctx.JSON(http.StatusBadRequest, invalidRequest)
		return
	}

	if authPayload.Role != util.BankerRole && authPayload.Username != req.GetUsername() {
		err := errors.New("no permission")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	arg := db.UpdateUserParams{
		Username: req.GetUsername(),
		FullName: pgtype.Text{
			String: req.GetFullName(),
			Valid:  req.FullName != nil,
		},
		Email: pgtype.Text{
			String: req.GetEmail(),
			Valid:  req.Email != nil,
		},
	}

	if req.Password != nil {
		hashedPassword, err := util.HashPassword(req.GetPassword())
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		arg.HashedPassword = pgtype.Text{
			String: hashedPassword,
			Valid:  true,
		}
		arg.PasswordChangedAt = pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	customResponse := newUserResponse(user)
	ctx.JSON(http.StatusOK, customResponse)
}

type invalidUpdateUserRequest struct {
	Violations []*FieldViolation `json:"violations"`
}

type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

func validateUpdateUserRequest(req *UpdateUserRequest) (invalidRequest invalidUpdateUserRequest) {
	if err := ValidateUsername(req.GetUsername()); err != nil {
		invalidRequest.Violations = append(invalidRequest.Violations, &FieldViolation{
			Field:       "username",
			Description: err.Error(),
		})
	}

	if req.Password != nil {
		if err := ValidatePassword(req.GetPassword()); err != nil {
			invalidRequest.Violations = append(invalidRequest.Violations, &FieldViolation{
				Field:       "password",
				Description: err.Error(),
			})
		}
	}

	if req.FullName != nil {
		if err := ValidateFullName(req.GetFullName()); err != nil {
			invalidRequest.Violations = append(invalidRequest.Violations, &FieldViolation{
				Field:       "full_name",
				Description: err.Error(),
			})
		}
	}

	if req.Email != nil {
		if err := ValidateEmail(req.GetEmail()); err != nil {
			invalidRequest.Violations = append(invalidRequest.Violations, &FieldViolation{
				Field:       "email",
				Description: err.Error(),
			})
		}
	}

	return
}
