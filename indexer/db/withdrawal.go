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
	GUID           string          `json:"guid"`
	FromAddress    string          `json:"from"`
	ToAddress      string          `json:"to"`
	L1Token        string          `json:"l1Token"`
	L2Token        *Token          `json:"l2Token"`
	Amount         string          `json:"amount"`
	Data           []byte          `json:"data"`
	LogIndex       uint64          `json:"logIndex"`
	BlockNumber    uint64          `json:"blockNumber"`
	BlockTimestamp string          `json:"blockTimestamp"`
	TxHash         string          `json:"transactionHash"`
	Batch          *StateBatchJSON `json:"batch"`
}
