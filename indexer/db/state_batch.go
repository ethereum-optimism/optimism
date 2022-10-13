package db

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// StateBatch is the state batch containing merkle root of the withdrawals
// periodically written to L1.
type StateBatch struct {
	Index     *big.Int
	Root      common.Hash
	Size      *big.Int
	PrevTotal *big.Int
	ExtraData []byte
	BlockHash common.Hash
}

// StateBatchJSON contains StateBatch data suitable for JSON serialization.
type StateBatchJSON struct {
	Index          uint64 `json:"index"`
	Root           string `json:"root"`
	Size           uint64 `json:"size"`
	PrevTotal      uint64 `json:"prevTotal"`
	ExtraData      []byte `json:"extraData"`
	BlockHash      string `json:"blockHash"`
	BlockNumber    uint64 `json:"blockNumber"`
	BlockTimestamp uint64 `json:"blockTimestamp"`
}
