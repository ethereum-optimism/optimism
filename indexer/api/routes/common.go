package routes

import (
	"encoding/json"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

func jsonResponse(w http.ResponseWriter, logger log.Logger, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Error("Failed to marshal JSON: %v", err)
		return
	}

	w.WriteHeader(statusCode)
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Error("Failed to write JSON data", err)
		return
	}
}
