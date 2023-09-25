package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

type WithdrawalItem struct {
	Guid                 string `json:"guid"`
	From                 string `json:"from"`
	To                   string `json:"to"`
	TransactionHash      string `json:"transactionHash"`
	Timestamp            uint64 `json:"timestamp"`
	L2BlockHash          string `json:"l2BlockHash"`
	Amount               string `json:"amount"`
	ProofTransactionHash string `json:"proofTransactionHash"`
	ClaimTransactionHash string `json:"claimTransactionHash"`
	L1TokenAddress       string `json:"l1TokenAddress"`
	L2TokenAddress       string `json:"l2TokenAddress"`
}

type WithdrawalResponse struct {
	Cursor      string           `json:"cursor"`
	HasNextPage bool             `json:"hasNextPage"`
	Items       []WithdrawalItem `json:"items"`
}

// FIXME make a pure function that returns a struct instead of newWithdrawalResponse
func newWithdrawalResponse(withdrawals *database.L2BridgeWithdrawalsResponse) WithdrawalResponse {
	items := make([]WithdrawalItem, len(withdrawals.Withdrawals))
	for i, withdrawal := range withdrawals.Withdrawals {
		item := WithdrawalItem{
			Guid:                 withdrawal.L2BridgeWithdrawal.TransactionWithdrawalHash.String(),
			L2BlockHash:          withdrawal.L2BlockHash.String(),
			From:                 withdrawal.L2BridgeWithdrawal.Tx.FromAddress.String(),
			To:                   withdrawal.L2BridgeWithdrawal.Tx.ToAddress.String(),
			TransactionHash:      withdrawal.L2TransactionHash.String(),
			Amount:               withdrawal.L2BridgeWithdrawal.Tx.Amount.String(),
			ProofTransactionHash: withdrawal.ProvenL1TransactionHash.String(),
			ClaimTransactionHash: withdrawal.FinalizedL1TransactionHash.String(),
			L1TokenAddress:       withdrawal.L2BridgeWithdrawal.TokenPair.RemoteTokenAddress.String(),
			L2TokenAddress:       withdrawal.L2BridgeWithdrawal.TokenPair.LocalTokenAddress.String(),
		}
		items[i] = item
	}

	return WithdrawalResponse{
		Cursor:      withdrawals.Cursor,
		HasNextPage: withdrawals.HasNextPage,
		Items:       items,
	}
}

// L2WithdrawalsHandler ... Handles /api/v0/withdrawals/{address} GET requests
func (h Routes) L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	address := common.HexToAddress(chi.URLParam(r, "address"))
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	limit, err := h.v.ValidateLimit(limitQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid query params")
		h.logger.Error(err.Error())
		return
	}

	withdrawals, err := h.view.L2BridgeWithdrawalsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading withdrawals", http.StatusInternalServerError)
		h.logger.Error("Unable to read withdrawals from DB", "err", err.Error())
		return
	}
	response := newWithdrawalResponse(withdrawals)

	err = jsonResponse(w, response, http.StatusOK)
	if err != nil {
		h.logger.Error("Error writing response", "err", err.Error())
	}
}
