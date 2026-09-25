package router

import (
	"encoding/json"
	"net/http"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/auth"
	"github.com/cleidison-barradas/betwallet.api/internal/utils"
)

type wagerTransactionHandler struct {
	getWagerTransaction           *app.GetWagerTransaction
	getWagerTransactionByProvider *app.GetWagerTransactionByProvider
	processWagerTransaction       *app.ProcessWagerTransaction
	auth                          *auth.Middleware
}

func NewWagerTransactionHandler(
	getWagerTransaction *app.GetWagerTransaction,
	getWagerTransactionByProvider *app.GetWagerTransactionByProvider,
	processWagerTransaction *app.ProcessWagerTransaction,
	auth *auth.Middleware,
) *wagerTransactionHandler {
	return &wagerTransactionHandler{
		getWagerTransaction:           getWagerTransaction,
		getWagerTransactionByProvider: getWagerTransactionByProvider,
		processWagerTransaction:       processWagerTransaction,
		auth:                          auth,
	}
}

type processWagerTransactionRequest struct {
	ProviderID            string       `json:"providerId"`
	ExternalTransactionID string       `json:"externalTransactionId"`
	PlayerID              string       `json:"playerId"`
	WalletID              string       `json:"walletId"`
	RoundID               string       `json:"roundId"`
	GameID                string       `json:"gameId"`
	Kind                  string       `json:"kind"`
	Money                 domain.Money `json:"money"`
}

type processWagerTransactionResponse struct {
	TransactionID    string       `json:"transactionId"`
	Status           string       `json:"status"`
	Balance          domain.Money `json:"balance"`
	IdempotentReplay bool         `json:"idempotentReplay"`
}

func (h *wagerTransactionHandler) handleGetWagerTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := domain.TransactionID(r.PathValue("transactionId"))
	providerID := auth.ProviderIDFromContext(r.Context())

	result, err := h.getWagerTransaction.Execute(r.Context(), app.GetWagerTransactionByIDCommand{
		TransactionID: transactionID,
		ProviderID:    domain.ProviderID(providerID),
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, result)
}

func (h *wagerTransactionHandler) handleGetWagerTransactionByProvider(w http.ResponseWriter, r *http.Request) {
	providerID := auth.ProviderIDFromContext(r.Context())
	externalTxID := r.PathValue("externalTransactionId")

	result, err := h.getWagerTransactionByProvider.Execute(r.Context(), app.GetWagerTransactionByProviderCommand{
		ProviderID:   domain.ProviderID(providerID),
		ExternalTxID: externalTxID,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, result)
}

func (h *wagerTransactionHandler) handleProcessWagerTransaction(w http.ResponseWriter, r *http.Request) {
	idempotenceKey := r.Header.Get("Idempotence-Key")
	providerID := auth.ProviderIDFromContext(r.Context())

	if idempotenceKey == "" {
		utils.Error(w, r, http.StatusBadRequest, domain.ErrWagerIdempotencyKeyMissing)
		return
	}

	var body processWagerTransactionRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	result, err := h.processWagerTransaction.Execute(r.Context(), app.ProcessWagerTransactionCommand{
		ProviderID:            providerID,
		ExternalTransactionID: body.ExternalTransactionID,
		IdempotencyKey:        idempotenceKey,
		PlayerID:              body.PlayerID,
		WalletID:              body.WalletID,
		RoundID:               body.RoundID,
		GameID:                body.GameID,
		Kind:                  domain.WagerKind(body.Kind),
		Money:                 body.Money,
	})
	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusCreated, processWagerTransactionResponse{
		TransactionID:    result.TransactionID,
		Status:           result.Status,
		Balance:          result.Balance,
		IdempotentReplay: false,
	})
}

func (h *wagerTransactionHandler) Register(mux *http.ServeMux) {
	mux.Handle("GET /wagering/transactions/{transactionId}", h.auth.RequireAuth(h.auth.RequireProvider(http.HandlerFunc(h.handleGetWagerTransaction))))
	mux.Handle("GET /providers/{providerId}/wagering/transactions/{externalTransactionId}", h.auth.RequireAuth(h.auth.RequireProvider(http.HandlerFunc(h.handleGetWagerTransactionByProvider))))
	mux.Handle("POST /wagering/transactions", h.auth.RequireAuth(h.auth.RequireProvider(http.HandlerFunc(h.handleProcessWagerTransaction))))
}
