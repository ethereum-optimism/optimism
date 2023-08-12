package handlers

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/middleware"
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
	Tx      Transaction `json:"Block"`
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
func newDepositResponse(deposits []*database.L1BridgeDepositWithTransactionHashes) DepositResponse {
	var items []DepositItem
	for _, deposit := range deposits {
		item := DepositItem{
			Guid: deposit.L1BridgeDeposit.TransactionSourceHash.String(),
			Tx: Transaction{
				BlockNumber:     420420,  // TODO
				BlockHash:       "0x420", // TODO
				TransactionHash: "0x420", // TODO
				Timestamp:       deposit.L1BridgeDeposit.Tx.Timestamp,
			},
			From:   deposit.L1BridgeDeposit.Tx.FromAddress.String(),
			To:     deposit.L1BridgeDeposit.Tx.ToAddress.String(),
			Amount: deposit.L1BridgeDeposit.Tx.Amount.Int.String(),
			L1Token: TokenInfo{
				ChainId:  1,
				Address:  deposit.L1BridgeDeposit.TokenPair.L1TokenAddress.String(),
				Name:     "TODO",
				Symbol:   "TODO",
				Decimals: 420,
				LogoURI:  "TODO",
				Extensions: Extensions{
					OptimismBridgeAddress: "0x420", // TODO
				},
			},
			L2Token: TokenInfo{
				ChainId:  10,
				Address:  deposit.L1BridgeDeposit.TokenPair.L2TokenAddress.String(),
				Name:     "TODO",
				Symbol:   "TODO",
				Decimals: 420,
				LogoURI:  "TODO",
				Extensions: Extensions{
					OptimismBridgeAddress: "0x420", // TODO
				},
			},
		}
		items = append(items, item)
	}

	return DepositResponse{
		Cursor:      "42042042-4204-4204-4204-420420420420", // TODO
		HasNextPage: false,                                  // TODO
		Items:       items,
	}
}

func L1DepositsHandler(w http.ResponseWriter, r *http.Request) {
	btv := middleware.GetBridgeTransfersView(r.Context())
	logger := middleware.GetLogger(r.Context())

	address := common.HexToAddress(chi.URLParam(r, "address"))

	deposits, err := btv.L1BridgeDepositsByAddress(address)
	if err != nil {
		http.Error(w, "Internal server error reading deposits", http.StatusInternalServerError)
		logger.Error("Unable to read deposits from DB")
		logger.Error(err.Error())
	}

	response := newDepositResponse(deposits)

	jsonResponse(w, logger, response, http.StatusOK)
}
