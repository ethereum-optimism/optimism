package routes

import (
	"net/http"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
)

type Proof struct {
	TransactionHash string `json:"transactionHash"`
	BlockTimestamp  uint64 `json:"blockTimestamp"`
	BlockNumber     int    `json:"blockNumber"`
}

type Claim struct {
	TransactionHash string `json:"transactionHash"`
	BlockTimestamp  uint64 `json:"blockTimestamp"`
	BlockNumber     int    `json:"blockNumber"`
}

type WithdrawalItem struct {
	Guid            string      `json:"guid"`
	Tx              Transaction `json:"Tx"`
	Block           Block       `json:"Block"`
	From            string      `json:"from"`
	To              string      `json:"to"`
	TransactionHash string      `json:"transactionHash"`
	Amount          string      `json:"amount"`
	Proof           Proof       `json:"proof"`
	Claim           Claim       `json:"claim"`
	WithdrawalState string      `json:"withdrawalState"`
	L1Token         TokenInfo   `json:"l1Token"`
	L2Token         TokenInfo   `json:"l2Token"`
}

type WithdrawalResponse struct {
	Cursor      string           `json:"cursor"`
	HasNextPage bool             `json:"hasNextPage"`
	Items       []WithdrawalItem `json:"items"`
}

// FIXME make a pure function that returns a struct instead of newWithdrawalResponse
func newWithdrawalResponse(withdrawals *database.L2BridgeWithdrawalsResponse) WithdrawalResponse {
	items := make([]WithdrawalItem, len(withdrawals.Withdrawals))
	for _, withdrawal := range withdrawals.Withdrawals {
		item := WithdrawalItem{
			Guid: withdrawal.L2BridgeWithdrawal.TransactionWithdrawalHash.String(),
			Block: Block{
				BlockNumber: 420420,  // TODO
				BlockHash:   "0x420", // TODO

			},
			Tx: Transaction{
				TransactionHash: "0x420", // TODO
				Timestamp:       withdrawal.L2BridgeWithdrawal.Tx.Timestamp,
			},
			From:            withdrawal.L2BridgeWithdrawal.Tx.FromAddress.String(),
			To:              withdrawal.L2BridgeWithdrawal.Tx.ToAddress.String(),
			TransactionHash: withdrawal.L2TransactionHash.String(),
			Amount:          withdrawal.L2BridgeWithdrawal.Tx.Amount.Int.String(),
			Proof: Proof{
				TransactionHash: withdrawal.ProvenL1TransactionHash.String(),
				BlockTimestamp:  withdrawal.L2BridgeWithdrawal.Tx.Timestamp,
				BlockNumber:     420, // TODO Block struct instead
			},
			Claim: Claim{
				TransactionHash: withdrawal.FinalizedL1TransactionHash.String(),
				BlockTimestamp:  withdrawal.L2BridgeWithdrawal.Tx.Timestamp, // Using L2 timestamp for now, might need adjustment
				BlockNumber:     420,                                        // TODO block struct
			},
			WithdrawalState: "COMPLETE", // TODO
			L1Token: TokenInfo{
				ChainId:  1,
				Address:  withdrawal.L2BridgeWithdrawal.TokenPair.RemoteTokenAddress.String(),
				Name:     "Example", // TODO
				Symbol:   "EXAMPLE", // TODO
				Decimals: 18,        // TODO
				Extensions: Extensions{
					OptimismBridgeAddress: "0x636Af16bf2f682dD3109e60102b8E1A089FedAa8",
				},
			},
			L2Token: TokenInfo{
				ChainId:  10,
				Address:  withdrawal.L2BridgeWithdrawal.TokenPair.LocalTokenAddress.String(),
				Name:     "Example", // TODO
				Symbol:   "EXAMPLE", // TODO
				Decimals: 18,        // TODO
				Extensions: Extensions{
					OptimismBridgeAddress: "0x36Af16bf2f682dD3109e60102b8E1A089FedAa86",
				},
			},
		}
		items = append(items, item)
	}

	return WithdrawalResponse{
		Cursor:      withdrawals.Cursor,
		HasNextPage: withdrawals.HasNextPage,
		Items:       items,
	}
}

func (h Routes) L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
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
	withdrawals, err := h.BridgeTransfersView.L2BridgeWithdrawalsByAddress(address, cursor, limit)
	if err != nil {
		http.Error(w, "Internal server error reading withdrawals", http.StatusInternalServerError)
		h.Logger.Error("Unable to read withdrawals from DB")
		h.Logger.Error(err.Error())
	}
	response := newWithdrawalResponse(withdrawals)

	jsonResponse(w, h.Logger, response, http.StatusOK)
}
