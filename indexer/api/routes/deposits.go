package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/models"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/go-chi/chi/v5"
)

// newDepositResponse ... Converts a database.L1BridgeDepositsResponse to an api.DepositResponse
func newDepositResponse(deposits *database.L1BridgeDepositsResponse) models.DepositResponse {
	items := make([]models.DepositItem, len(deposits.Deposits))
	for i, deposit := range deposits.Deposits {
		item := models.DepositItem{
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

	return models.DepositResponse{
		Cursor:      deposits.Cursor,
		HasNextPage: deposits.HasNextPage,
		Items:       items,
	}
}

// L1DepositsHandler ... Handles /api/v0/deposits/{address} GET requests
func (h Routes) L1DepositsHandler(w http.ResponseWriter, r *http.Request) {
	addressValue := chi.URLParam(r, "address")
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	address, err := h.v.ParseValidateAddress(addressValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid address param", "param", addressValue)
		h.logger.Error(err.Error())
		return
	}

	err = h.v.ValidateCursor(cursor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid cursor param", "param", cursor, "err", err.Error())
	}

	limit, err := h.v.ParseValidateLimit(limitQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Invalid limit param", "param", limitQuery, "err", err.Error())
		return
	}

	deposits, err := h.view.L1BridgeDepositsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading deposits", http.StatusInternalServerError)
		h.logger.Error("Unable to read deposits from DB", "err", err.Error())
		return
	}

	response := newDepositResponse(deposits)

	err = jsonResponse(w, response, http.StatusOK)
	if err != nil {
		h.logger.Error("Error writing response", "err", err)
	}
}
