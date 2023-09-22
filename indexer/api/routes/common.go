package routes

import (
	"encoding/json"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

const (
	InternalServerError = "Internal server error"

	// defaultPageLimit ... Default page limit for pagination
	defaultPageLimit = 100
)

// jsonResponse ... Marshals and writes a JSON response provided arbitrary data
func jsonResponse(w http.ResponseWriter, logger log.Logger, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		logger.Error("Failed to marshal JSON: %v", err)
		return
	}

	w.WriteHeader(statusCode)
	_, err = w.Write(jsonData)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		logger.Error("Failed to write JSON data", err)
		return
	}
}
