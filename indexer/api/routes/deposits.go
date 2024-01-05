package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// L1DepositsHandler ... Handles /api/v0/deposits/{address} GET requests
func (h Routes) L1DepositsHandler(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	cursor := r.URL.Query().Get("cursor")
	limit := r.URL.Query().Get("limit")

	params, err := h.svc.QueryParams(address, cursor, limit)
	if err != nil {
		http.Error(w, "invalid query params", http.StatusBadRequest)
		h.logger.Error("error reading request params", "err", err.Error())
		return
	}

	deposits, err := h.svc.GetDeposits(params)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		h.logger.Error("error fetching deposits", "err", err.Error())
		return
	}

	resp := h.svc.DepositResponse(deposits)
	err = jsonResponse(w, resp, http.StatusOK)
	if err != nil {
		h.logger.Error("error writing response", "err", err)
	}
}
