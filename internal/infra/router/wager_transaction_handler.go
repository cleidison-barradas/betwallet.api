package router

import (
	"net/http"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/cleidison-barradas/betwallet.api/internal/utils"
)

type wagerTransactionHandler struct {
	getWagerTransaction           *app.GetWagerTransaction
	getWagerTransactionByProvider *app.GetWagerTransactionByProvider
}

func NewWagerTransactionHandler(
	getWagerTransaction *app.GetWagerTransaction,
	getWagerTransactionByProvider *app.GetWagerTransactionByProvider,
) *wagerTransactionHandler {
	return &wagerTransactionHandler{
		getWagerTransaction:           getWagerTransaction,
		getWagerTransactionByProvider: getWagerTransactionByProvider,
	}
}

func (h *wagerTransactionHandler) handleGetWagerTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := domain.TransactionID(r.PathValue("transactionId"))

	result, err := h.getWagerTransaction.Execute(r.Context(), app.GetWagerTransactionByIDCommand{
		TransactionID: transactionID,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, result)
}

func (h *wagerTransactionHandler) handleGetWagerTransactionByProvider(w http.ResponseWriter, r *http.Request) {
	providerID := domain.ProviderID(r.PathValue("providerId"))
	externalTxID := r.PathValue("externalTransactionId")

	result, err := h.getWagerTransactionByProvider.Execute(r.Context(), app.GetWagerTransactionByProviderCommand{
		ProviderID:   providerID,
		ExternalTxID: externalTxID,
	})

	if err != nil {
		utils.Error(w, r, http.StatusBadRequest, err)
		return
	}

	utils.Success(w, r, http.StatusOK, result)
}

func (h *wagerTransactionHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /wagering/transactions/{transactionId}", h.handleGetWagerTransaction)
	mux.HandleFunc("GET /providers/{providerId}/wagering/transactions/{externalTransactionId}", h.handleGetWagerTransactionByProvider)
}
