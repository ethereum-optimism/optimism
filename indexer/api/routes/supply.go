package routes

import (
	"net/http"
)

// SupplyView ... Handles /api/v0/supply GET requests
func (h Routes) SupplyView(w http.ResponseWriter, r *http.Request) {

	view, err := h.svc.GetSupplyInfo()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		h.logger.Error("error getting supply info", "err", err)
		return
	}

	err = jsonResponse(w, view, http.StatusOK)
	if err != nil {
		h.logger.Error("error writing response", "err", err)
	}
}
