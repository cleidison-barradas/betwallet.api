package router

import (
	"net/http"
	"strconv"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/auth"
	"github.com/cleidison-barradas/betwallet.api/internal/utils"
)

type walletLedgerHandler struct {
	getWalletLedgerByWalletID *app.GetWalletLedgerByWalletID
	auth                      *auth.Middleware
}

func NewWalletLedgerHandler(
	getWalletLedgerByWalletID *app.GetWalletLedgerByWalletID,
	auth *auth.Middleware,
) *walletLedgerHandler {
	return &walletLedgerHandler{
		getWalletLedgerByWalletID: getWalletLedgerByWalletID,
		auth:                      auth,
	}
}
func (h *walletLedgerHandler) handleGetWalletLedgerByWalletID(w http.ResponseWriter, r *http.Request) {
	limit := 20
	walletID := r.PathValue("walletId")
	cursor := r.URL.Query().Get("cursor")

	if value := r.URL.Query().Get("limit"); value != "" {
		limit, _ = strconv.Atoi(value)
	}

	result, err := h.getWalletLedgerByWalletID.Execute(r.Context(), app.GetWalletLedgerByWalletIDCommand{
		WalletID: walletID,
		Limit:    limit,
		Cursor:   cursor,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, result)
}

func (h *walletLedgerHandler) Register(mux *http.ServeMux) {
	mux.Handle("GET /wallets/{walletId}/ledgers", h.auth.RequireAuth(h.auth.RequireInternal(http.HandlerFunc(h.handleGetWalletLedgerByWalletID))))
}
