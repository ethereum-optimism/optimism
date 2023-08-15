package routes

import (
	"net/http"
)

func (h Routes) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, h.Logger, "ok", http.StatusOK)
}
