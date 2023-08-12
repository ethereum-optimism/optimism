package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

// lazily typing numbers fixme
type Transaction struct {
	Timestamp       uint64 `json:"timestamp"`
	BlockNumber     int64  `json:"number"` // viem treats me as a BigInt maybe consider just stringing me too. Not 100% necessary though
	BlockHash       string `json:"hash"`
	TransactionHash string `json:"transactionHash"`
	// TODO maybe include me
	// ParentBlockHash   string `json:"parentHash"`
}

type Extensions struct {
	OptimismBridgeAddress string `json:"OptimismBridgeAddress"`
}

type TokenInfo struct {
	// TODO lazily typing ints go through them all with fine tooth comb once api is up
	ChainId    int        `json:"chainId"`
	Address    string     `json:"address"`
	Name       string     `json:"name"`
	Symbol     string     `json:"symbol"`
	Decimals   int        `json:"decimals"`
	LogoURI    string     `json:"logoURI"`
	Extensions Extensions `json:"extensions"`
}

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
