package handlers

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/middleware"
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
	Guid            string    `json:"guid"`
	BlockTimestamp  uint64    `json:"blockTimestamp"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	TransactionHash string    `json:"transactionHash"`
	Amount          string    `json:"amount"`
	BlockNumber     int       `json:"blockNumber"`
	Proof           Proof     `json:"proof"`
	Claim           Claim     `json:"claim"`
	WithdrawalState string    `json:"withdrawalState"`
	L1Token         TokenInfo `json:"l1Token"`
	L2Token         TokenInfo `json:"l2Token"`
}

type WithdrawalResponse struct {
	Cursor      string           `json:"cursor"`
	HasNextPage bool             `json:"hasNextPage"`
	Items       []WithdrawalItem `json:"items"`
}

func newWithdrawalResponse(withdrawals []*database.L2BridgeWithdrawalWithTransactionHashes) WithdrawalResponse {
	var items []WithdrawalItem
	for _, withdrawal := range withdrawals {
		item := WithdrawalItem{
			Guid:            withdrawal.L2BridgeWithdrawal.TransactionWithdrawalHash.String(),
			BlockTimestamp:  withdrawal.L2BridgeWithdrawal.Tx.Timestamp,
			From:            withdrawal.L2BridgeWithdrawal.Tx.FromAddress.String(),
			To:              withdrawal.L2BridgeWithdrawal.Tx.ToAddress.String(),
			TransactionHash: withdrawal.L2TransactionHash.String(),
			Amount:          withdrawal.L2BridgeWithdrawal.Tx.Amount.Int.String(),
			BlockNumber:     420, // TODO
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
				Address:  withdrawal.L2BridgeWithdrawal.TokenPair.L1TokenAddress.String(),
				Name:     "Example",                                              // TODO
				Symbol:   "EXAMPLE",                                              // TODO
				Decimals: 18,                                                     // TODO
				LogoURI:  "https://ethereum-optimism.github.io/data/OP/logo.svg", // TODO
				Extensions: Extensions{
					OptimismBridgeAddress: "0x636Af16bf2f682dD3109e60102b8E1A089FedAa8",
				},
			},
			L2Token: TokenInfo{
				ChainId:  10,
				Address:  withdrawal.L2BridgeWithdrawal.TokenPair.L2TokenAddress.String(),
				Name:     "Example",                                              // TODO
				Symbol:   "EXAMPLE",                                              // TODO
				Decimals: 18,                                                     // TODO
				LogoURI:  "https://ethereum-optimism.github.io/data/OP/logo.svg", // TODO
				Extensions: Extensions{
					OptimismBridgeAddress: "0x36Af16bf2f682dD3109e60102b8E1A089FedAa86",
				},
			},
		}
		items = append(items, item)
	}

	return WithdrawalResponse{
		Cursor:      "42042042-0420-4204-2042-420420420420", // TODO
		HasNextPage: true,                                   // TODO
		Items:       items,
	}
}

func L2WithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	btv := middleware.GetBridgeTransfersView(r.Context())
	logger := middleware.GetLogger(r.Context())

	address := common.HexToAddress(chi.URLParam(r, "address"))

	withdrawals, err := btv.L2BridgeWithdrawalsByAddress(address)
	if err != nil {
		http.Error(w, "Internal server error fetching withdrawals", http.StatusInternalServerError)
		logger.Error("Unable to read deposits from DB")
		logger.Error(err.Error())
		return
	}

	response := newWithdrawalResponse(withdrawals)

	jsonResponse(w, logger, response, http.StatusOK)
}
