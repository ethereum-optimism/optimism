package db

import (
	"github.com/ethereum/go-ethereum/common"
)

// IndexedL1Block contains the L1 block including the deposits in it.
type IndexedL1Block struct {
	Hash                 common.Hash
	ParentHash           common.Hash
	Number               uint64
	Timestamp            uint64
	Deposits             []Deposit
	ProvenWithdrawals    []ProvenWithdrawal
	FinalizedWithdrawals []FinalizedWithdrawal
}

// String returns the block hash for the indexed l1 block.
func (b IndexedL1Block) String() string {
	return b.Hash.String()
}

// IndexedL2Block contains the L2 block including the withdrawals in it.
type IndexedL2Block struct {
	Hash        common.Hash
	ParentHash  common.Hash
	Number      uint64
	Timestamp   uint64
	Withdrawals []Withdrawal
}

// String returns the block hash for the indexed l2 block.
func (b IndexedL2Block) String() string {
	return b.Hash.String()
}
