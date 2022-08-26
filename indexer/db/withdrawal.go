package db

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Withdrawal contains transaction data for withdrawals made via the L2 to L1 bridge.
type Withdrawal struct {
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

// String returns the tx hash for the withdrawal.
func (w Withdrawal) String() string {
	return w.TxHash.String()
}

// WithdrawalJSON contains Withdrawal data suitable for JSON serialization.
type WithdrawalJSON struct {
	GUID             string `json:"guid"`
	FromAddress      string `json:"from"`
	ToAddress        string `json:"to"`
	L1Token          string `json:"l1Token"`
	L2Token          *Token `json:"l2Token"`
	Amount           string `json:"amount"`
	Data             []byte `json:"data"`
	LogIndex         uint64 `json:"logIndex"`
	L1BlockNumber    uint64 `json:"l1BlockNumber"`
	L1BlockTimestamp string `json:"l1BlockTimestamp"`
	L2BlockNumber    uint64 `json:"l2BlockNumber"`
	L2BlockTimestamp string `json:"l2BlockTimestamp"`
	TxHash           string `json:"transactionHash"`
}
