package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/models"
	"github.com/go-chi/chi/v5"
)

// L2WithdrawalsHandler ... Handles /api/v0/withdrawals/{address} GET requests
func (h Routes) L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	addressValue := chi.URLParam(r, "address")
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	address, err := h.v.ParseValidateAddress(addressValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid address param", "param", addressValue, "err", err)
		return
	}

	err = h.v.ValidateCursor(cursor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid cursor param", "param", cursor, "err", err)
		return
	}

	limit, err := h.v.ParseValidateLimit(limitQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid query params", "err", err)
		return
	}

	withdrawals, err := h.view.L2BridgeWithdrawalsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading withdrawals", http.StatusInternalServerError)
		h.logger.Error("Unable to read withdrawals from DB", "err", err.Error())
		return
	}
	response := models.CreateWithdrawalResponse(withdrawals)

	err = jsonResponse(w, response, http.StatusOK)
	if err != nil {
		h.logger.Error("Error writing response", "err", err.Error())
	}
}
