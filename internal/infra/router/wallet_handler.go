package router

import (
	"encoding/json"
	"net/http"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/cleidison-barradas/betwallet.api/internal/utils"
)

type walletHandler struct {
	openWallet          *app.OpenWallet
	getWalletByWalletID *app.GetWalletByWalletID
}

func NewWalletHandler(openWallet *app.OpenWallet, getWalletByWalletID *app.GetWalletByWalletID) *walletHandler {
	return &walletHandler{
		openWallet:          openWallet,
		getWalletByWalletID: getWalletByWalletID,
	}
}

type openWalletRequest struct {
	PlayerID       string       `json:"playerId"`
	InitialBalance domain.Money `json:"initialBalance"`
}

type walletResponse struct {
	ID       string       `json:"id"`
	PlayerID string       `json:"playerId"`
	Balance  domain.Money `json:"balance"`
	Version  int64        `json:"version"`
}

func (h *walletHandler) handleOpenWallet(w http.ResponseWriter, r *http.Request) {
	var body openWalletRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	if body.InitialBalance.Currency() == "" {
		utils.Error(w, r, http.StatusBadRequest, domain.ErrInvalidCurrency)
		return
	}

	result, err := h.openWallet.Execute(r.Context(), app.OpenWalletCommand{
		PlayerID:       body.PlayerID,
		InitialBalance: body.InitialBalance,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusCreated, walletResponse{
		ID:       result.WalletID,
		PlayerID: result.PlayerID,
		Balance:  result.Balance,
		Version:  result.Version,
	})
}

func (h *walletHandler) handleGetWalletByWalletID(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("walletId")

	result, err := h.getWalletByWalletID.Execute(r.Context(), app.GetWalletByWalletIDCommand{
		WalletID: walletID,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, walletResponse{
		ID:       result.WalletID,
		PlayerID: result.PlayerID,
		Balance:  result.Balance,
		Version:  result.Version,
	})
}

func (h *walletHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /wallets", h.handleOpenWallet)
	mux.HandleFunc("GET /wallets/{walletId}", h.handleGetWalletByWalletID)
}
