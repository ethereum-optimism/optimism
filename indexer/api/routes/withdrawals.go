package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/models"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/go-chi/chi/v5"
)

// FIXME make a pure function that returns a struct instead of newWithdrawalResponse
// newWithdrawalResponse ... Converts a database.L2BridgeWithdrawalsResponse to an api.WithdrawalResponse
func newWithdrawalResponse(withdrawals *database.L2BridgeWithdrawalsResponse) models.WithdrawalResponse {
	items := make([]models.WithdrawalItem, len(withdrawals.Withdrawals))
	for i, withdrawal := range withdrawals.Withdrawals {
		item := models.WithdrawalItem{
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

	return models.WithdrawalResponse{
		Cursor:      withdrawals.Cursor,
		HasNextPage: withdrawals.HasNextPage,
		Items:       items,
	}
}

// L2WithdrawalsHandler ... Handles /api/v0/withdrawals/{address} GET requests
func (h Routes) L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	addressValue := chi.URLParam(r, "address")
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	address, err := h.v.ParseValidateAddress(addressValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid address param", "param", addressValue, "err", err)
		return
	}

	err = h.v.ValidateCursor(cursor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid cursor param", "param", cursor, "err", err)
		return
	}

	limit, err := h.v.ParseValidateLimit(limitQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid query params", "err", err)
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
