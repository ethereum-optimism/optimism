package db

import (
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum/go-ethereum/common"
)

// IndexedL1Block contains the L1 block including the deposits in it.
type IndexedL1Block struct {
	Hash       common.Hash
	ParentHash common.Hash
	Number     uint64
	Timestamp  uint64
	Deposits   []Deposit
}

// String returns the block hash for the indexed l1 block.
func (b IndexedL1Block) String() string {
	return b.Hash.String()
}

// IndexedL2Block contains the L2 block including the withdrawals in it.
type IndexedL2Block struct {
	Hash        l2common.Hash
	ParentHash  l2common.Hash
	Number      uint64
	Timestamp   uint64
	Withdrawals []Withdrawal
}

// String returns the block hash for the indexed l2 block.
func (b IndexedL2Block) String() string {
	return b.Hash.String()
}
