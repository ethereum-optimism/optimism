package routes

import (
	"net/http"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

type DepositItem struct {
	Guid string `json:"guid"`
	From string `json:"from"`
	To   string `json:"to"`
	// TODO could consider OriginTx to be more generic to handling L2 to L2 deposits
	// this seems more clear today though
	Tx      Transaction `json:"Tx"`
	Block   Block       `json:"Block"`
	Amount  string      `json:"amount"`
	L1Token TokenInfo   `json:"l1Token"`
	L2Token TokenInfo   `json:"l2Token"`
}

type DepositResponse struct {
	Cursor      string        `json:"cursor"`
	HasNextPage bool          `json:"hasNextPage"`
	Items       []DepositItem `json:"items"`
}

// TODO this is original spec but maybe include the l2 block info too for the relayed tx
// FIXME make a pure function that returns a struct instead of newWithdrawalResponse
func newDepositResponse(deposits *database.L1BridgeDepositsResponse) DepositResponse {
	items := make([]DepositItem, len(deposits.Deposits))
	for _, deposit := range deposits.Deposits {
		item := DepositItem{
			Guid: deposit.L1BridgeDeposit.TransactionSourceHash.String(),
			Block: Block{
				BlockNumber: 420420,  // TODO
				BlockHash:   "0x420", // TODO

			},
			Tx: Transaction{
				TransactionHash: "0x420", // TODO
				Timestamp:       deposit.L1BridgeDeposit.Tx.Timestamp,
			},
			From:   deposit.L1BridgeDeposit.Tx.FromAddress.String(),
			To:     deposit.L1BridgeDeposit.Tx.ToAddress.String(),
			Amount: deposit.L1BridgeDeposit.Tx.Amount.Int.String(),
			L1Token: TokenInfo{
				ChainId:  1,
				Address:  deposit.L1BridgeDeposit.TokenPair.LocalTokenAddress.String(),
				Name:     "TODO",
				Symbol:   "TODO",
				Decimals: 420,
				Extensions: Extensions{
					OptimismBridgeAddress: "0x420", // TODO
				},
			},
			L2Token: TokenInfo{
				ChainId:  10,
				Address:  deposit.L1BridgeDeposit.TokenPair.RemoteTokenAddress.String(),
				Name:     "TODO",
				Symbol:   "TODO",
				Decimals: 420,
				Extensions: Extensions{
					OptimismBridgeAddress: "0x420", // TODO
				},
			},
		}
		items = append(items, item)
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
