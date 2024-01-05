package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// L2WithdrawalsHandler ... Handles /api/v0/withdrawals/{address} GET requests
func (h Routes) L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	cursor := r.URL.Query().Get("cursor")
	limit := r.URL.Query().Get("limit")

	params, err := h.svc.QueryParams(address, cursor, limit)
	if err != nil {
		http.Error(w, "Invalid query params", http.StatusBadRequest)
		h.logger.Error("Invalid query params", "err", err.Error())
		return
	}

	withdrawals, err := h.svc.GetWithdrawals(params)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		h.logger.Error("Error getting withdrawals", "err", err.Error())
		return
	}

	resp := h.svc.WithdrawResponse(withdrawals)
	err = jsonResponse(w, resp, http.StatusOK)
	if err != nil {
		h.logger.Error("Error writing response", "err", err.Error())
	}
}
