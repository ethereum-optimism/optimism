package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

// DepositItem ... Deposit item model for API responses
type DepositItem struct {
	Guid           string `json:"guid"`
	From           string `json:"from"`
	To             string `json:"to"`
	Timestamp      uint64 `json:"timestamp"`
	L1BlockHash    string `json:"l1BlockHash"`
	L1TxHash       string `json:"l1TxHash"`
	L2TxHash       string `json:"l2TxHash"`
	Amount         string `json:"amount"`
	L1TokenAddress string `json:"l1TokenAddress"`
	L2TokenAddress string `json:"l2TokenAddress"`
}

// DepositResponse ... Data model for API JSON response
type DepositResponse struct {
	Cursor      string        `json:"cursor"`
	HasNextPage bool          `json:"hasNextPage"`
	Items       []DepositItem `json:"items"`
}

// newDepositResponse ... Converts a database.L1BridgeDepositsResponse to an api.DepositResponse
func newDepositResponse(deposits *database.L1BridgeDepositsResponse) DepositResponse {
	items := make([]DepositItem, len(deposits.Deposits))
	for i, deposit := range deposits.Deposits {
		item := DepositItem{
			Guid:           deposit.L1BridgeDeposit.TransactionSourceHash.String(),
			L1BlockHash:    deposit.L1BlockHash.String(),
			Timestamp:      deposit.L1BridgeDeposit.Tx.Timestamp,
			L1TxHash:       deposit.L1TransactionHash.String(),
			L2TxHash:       deposit.L2TransactionHash.String(),
			From:           deposit.L1BridgeDeposit.Tx.FromAddress.String(),
			To:             deposit.L1BridgeDeposit.Tx.ToAddress.String(),
			Amount:         deposit.L1BridgeDeposit.Tx.Amount.String(),
			L1TokenAddress: deposit.L1BridgeDeposit.TokenPair.LocalTokenAddress.String(),
			L2TokenAddress: deposit.L1BridgeDeposit.TokenPair.RemoteTokenAddress.String(),
		}
		items[i] = item
	}

	return DepositResponse{
		Cursor:      deposits.Cursor,
		HasNextPage: deposits.HasNextPage,
		Items:       items,
	}
}

// L1DepositsHandler ... Handles /api/v0/deposits/{address} GET requests
func (h Routes) L1DepositsHandler(w http.ResponseWriter, r *http.Request) {
	address := common.HexToAddress(chi.URLParam(r, "address"))
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	limit, err := h.v.ParseValidateLimit(limitQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid limit param", "param", limitQuery)
		h.logger.Error(err.Error())
		return
	}

	deposits, err := h.view.L1BridgeDepositsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading deposits", http.StatusInternalServerError)
		h.logger.Error("Unable to read deposits from DB")
		h.logger.Error(err.Error())
		return
	}

	response := newDepositResponse(deposits)

	err = jsonResponse(w, response, http.StatusOK)
	if err != nil {
		h.logger.Error("Error writing response", "err", err)
	}
}
