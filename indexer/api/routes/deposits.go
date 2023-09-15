package routes

import (
	"net/http"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

type DepositItem struct {
	Guid           string `json:"guid"`
	From           string `json:"from"`
	To             string `json:"to"`
	Timestamp      uint64 `json:"timestamp"`
	L1TxHash       string `json:"L1TxHash"`
	L2TxHash       string `json:"L2TxHash"`
	L1BlockHash    string `json:"Block"`
	Amount         string `json:"amount"`
	L1TokenAddress string `json:"l1Token"`
	L2TokenAddress string `json:"l2Token"`
}

type DepositResponse struct {
	Cursor      string        `json:"cursor"`
	HasNextPage bool          `json:"hasNextPage"`
	Items       []DepositItem `json:"items"`
}

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

func (h Routes) L1DepositsHandler(w http.ResponseWriter, r *http.Request) {
	address := common.HexToAddress(chi.URLParam(r, "address"))
	cursor := r.URL.Query().Get("cursor")
	limitQuery := r.URL.Query().Get("limit")

	defaultLimit := 100
	limit := defaultLimit
	if limitQuery != "" {
		parsedLimit, err := strconv.Atoi(limitQuery)
		if err != nil {
			http.Error(w, "Limit could not be parsed into a number", http.StatusBadRequest)
			h.Logger.Error("Invalid limit")
			h.Logger.Error(err.Error())
		}
		limit = parsedLimit
	}

	deposits, err := h.BridgeTransfersView.L1BridgeDepositsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading deposits", http.StatusInternalServerError)
		h.Logger.Error("Unable to read deposits from DB")
		h.Logger.Error(err.Error())
		return
	}

	response := newDepositResponse(deposits)

	jsonResponse(w, h.Logger, response, http.StatusOK)
}
