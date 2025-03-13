package api

import (
	"errors"
	"fmt"
	"net/http"

	db "github.com/RobertChienShiba/simplebank/db/sqlc"
	"github.com/RobertChienShiba/simplebank/token"
	"github.com/gin-gonic/gin"
)

type transferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
	OTP           string `json:"otp" binding:"required,min=6,max=6,numeric"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if !valid {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if fromAccount.Owner != authPayload.Username {
		err := errors.New("from account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// _, valid = server.validAccount(ctx, req.ToAccountID, req.Currency)
	// if !valid {
	// 	return
	// }

	Toaccount, err := server.store.GetAccount(ctx, req.ToAccountID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	afterExchange, err := server.currencyExchange(ctx, fromAccount.Currency, Toaccount.Currency, req.Amount)
	if err != nil {
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		FromAmount:    req.Amount,
		ToAmount:      afterExchange,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return account, false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return account, false
	}

	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return account, false
	}

	return account, true

}

func (server *Server) currencyExchange(ctx *gin.Context, fromCurrency string, toCurrency string, amount int64) (int64, error) {
	fromExchangeRate, errFrom := server.store.GetExchangeRate(ctx, fromCurrency)
	if errFrom != nil {
		if errors.Is(errFrom, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errFrom))
			return int64(-1), errFrom
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(errFrom))
		return int64(-1), errFrom
	}

	toExchangeRate, errTo := server.store.GetExchangeRate(ctx, toCurrency)
	if errTo != nil {
		if errors.Is(errTo, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(errTo))
			return int64(-1), errTo
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(errFrom))
		return int64(-1), errTo
	}

	exchangeRate := fromExchangeRate / toExchangeRate
	return int64(float64(amount) * exchangeRate), nil
}
