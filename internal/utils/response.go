package utils

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
)

type AppError struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Code    ErrorCode `json:"code"`
}

type appErrorMaping struct {
	target  error
	status  int
	code    ErrorCode
	message string
}

type ErrorCode string

const (
	IDEMPOTENCY_KEY_CONFLICT                          ErrorCode = "IDEMPOTENCY_KEY_CONFLICT"
	WALLET_VERSION_INVALID                            ErrorCode = "WALLET_VERSION_INVALID"
	WALLET_BALANCE_NEGATIVE                           ErrorCode = "WALLET_BALANCE_NEGATIVE"
	WALLET_MISSING_REQUIRED_FIELDS                    ErrorCode = "WALLET_MISSING_REQUIRED_FIELDS"
	LEDGER_MISSING_REQUIRED_FIELDS                    ErrorCode = "LEDGER_MISSING_REQUIRED_FIELDS"
	LEDGER_INVALID_DIRECTION                          ErrorCode = "LEDGER_INVALID_DIRECTION"
	LEDGER_NEGATIVE_AMOUNT                            ErrorCode = "LEDGER_NEGATIVE_AMOUNT"
	LEDGER_INVALID_BALANCE                            ErrorCode = "LEDGER_INVALID_BALANCE"
	LADGER_NOT_FOUND                                  ErrorCode = "LADGER_NOT_FOUND"
	WAGER_TRANSACTION_MISSING_REQUIRED_FIELDS         ErrorCode = "WAGER_TRANSACTION_MISSING_REQUIRED_FIELDS"
	WAGER_TRANSACTION_INVALID_TRANSACTION_TYPE        ErrorCode = "WAGER_TRANSACTION_INVALID_TRANSACTION_TYPE"
	WAGER_TRANSACTION_AMOUNT_ZERO_OR_NEGATIVE_ON_LOSS ErrorCode = "WAGER_TRANSACTION_AMOUNT_ZERO_OR_NEGATIVE_ON_LOSS"
	WAGER_TRANSACTION_INVALID_STATUS_TRANSITION       ErrorCode = "WAGER_TRANSACTION_INVALID_STATUS_TRANSITION"
	WAGER_TRANSACTION_NOT_FOUND                       ErrorCode = "WAGER_TRANSACTION_NOT_FOUND"
	INTERNAL_ERROR                                    ErrorCode = "INTERNAL_ERROR"
	UNHANDLED_ERROR                                   ErrorCode = "UNHANDLED_ERROR"
)

var erroMappings = []appErrorMaping{
	{app.ErrLedgerNotFound, http.StatusNotFound, LADGER_NOT_FOUND, "ledger not found"},
	{app.ErrIdempotencyKeyConflict, http.StatusConflict, IDEMPOTENCY_KEY_CONFLICT, "idempotency key already used with a different payload"},
	{app.ErrIdempotencyKeyRaceLost, http.StatusConflict, IDEMPOTENCY_KEY_CONFLICT, "idempotency key already race lost"},
	{domain.ErrWalletVersionInvalid, http.StatusUnprocessableEntity, WALLET_VERSION_INVALID, "invalid wallet version"},
	{domain.ErrWalletBalanceNegative, http.StatusUnprocessableEntity, WALLET_BALANCE_NEGATIVE, "wallet balance is negative"},
	{domain.ErrWalletMissingRequiredFields, http.StatusUnprocessableEntity, WALLET_MISSING_REQUIRED_FIELDS, "wallet is missing required fields"},
	{domain.ErrLedgerMissingRequiredFields, http.StatusUnprocessableEntity, LEDGER_MISSING_REQUIRED_FIELDS, "ledger is missing required fields"},
	{domain.ErrLedgerInvalidDirection, http.StatusUnprocessableEntity, LEDGER_INVALID_DIRECTION, "ledger direction is invalid"},
	{domain.ErrLedgerNegativeAmount, http.StatusUnprocessableEntity, LEDGER_NEGATIVE_AMOUNT, "ledger amount is negative"},
	{domain.ErrLedgerInvalidBalance, http.StatusUnprocessableEntity, LEDGER_INVALID_BALANCE, "ledger balance is invalid"},
	{domain.ErrWagerTransactionNotFound, http.StatusNotFound, WAGER_TRANSACTION_NOT_FOUND, "wager transaction not found"},
	{domain.ErrWagerMissingRequiredFields, http.StatusUnprocessableEntity, WAGER_TRANSACTION_MISSING_REQUIRED_FIELDS, "wager transaction is missing required fields"},
	{domain.ErrWagerInvalidTransactionType, http.StatusUnprocessableEntity, WAGER_TRANSACTION_INVALID_TRANSACTION_TYPE, "wager transaction type is invalid"},
	{domain.ErrWagerAmountZeroOrNegativeOnLoose, http.StatusUnprocessableEntity, WAGER_TRANSACTION_AMOUNT_ZERO_OR_NEGATIVE_ON_LOSS, "wager transaction amount is zero or negative on loss"},
	{domain.ErrWagerInvalidTransactionStatus, http.StatusUnprocessableEntity, WAGER_TRANSACTION_INVALID_STATUS_TRANSITION, "wager transaction status transition is invalid"},
}

func Success(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	code := UNHANDLED_ERROR
	message := err.Error()

	for _, m := range erroMappings {
		if errors.Is(err, m.target) {
			statusCode, code, message = m.status, m.code, m.message
			break
		}
	}

	if statusCode >= 500 {
		slog.ErrorContext(
			r.Context(),
			"unhandled error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(AppError{
		Success: false,
		Code:    code,
		Message: message,
	})
}
