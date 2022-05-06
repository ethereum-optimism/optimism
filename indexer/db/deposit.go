package db

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Deposit contains transaction data for deposits made via the L1 to L2 bridge.
type Deposit struct {
	GUID        string
	TxHash      common.Hash
	L1Token     common.Address
	L2Token     common.Address
	FromAddress common.Address
	ToAddress   common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
}

// String returns the tx hash for the deposit.
func (d Deposit) String() string {
	return d.TxHash.String()
}

// DepositJSON contains Deposit data suitable for JSON serialization.
type DepositJSON struct {
	GUID           string `json:"guid"`
	FromAddress    string `json:"from"`
	ToAddress      string `json:"to"`
	L1Token        *Token `json:"l1Token"`
	L2Token        string `json:"l2Token"`
	Amount         string `json:"amount"`
	Data           []byte `json:"data"`
	LogIndex       uint64 `json:"logIndex"`
	BlockNumber    uint64 `json:"blockNumber"`
	BlockTimestamp string `json:"blockTimestamp"`
	TxHash         string `json:"transactionHash"`
}
