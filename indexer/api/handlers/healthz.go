package handlers

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/middleware"
)

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	jsonResponse(w, logger, "ok", http.StatusOK)
}
