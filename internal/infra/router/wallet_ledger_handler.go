package router

import (
	"net/http"
	"strconv"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/utils"
)

type walletLedgerHandler struct {
	getWalletLedgerByWalletID *app.GetWalletLedgerByWalletID
}

func NewWalletLedgerHandler(getWalletLedgerByWalletID *app.GetWalletLedgerByWalletID) *walletLedgerHandler {
	return &walletLedgerHandler{
		getWalletLedgerByWalletID: getWalletLedgerByWalletID,
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
	mux.HandleFunc("GET /wallets/{walletId}/ledgers", h.handleGetWalletLedgerByWalletID)
}
