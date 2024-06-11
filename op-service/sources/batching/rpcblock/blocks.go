package rpcblock

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// Block represents the block ref value in RPC calls.
// It can be either a label (e.g. latest), a block number or block hash.
type Block struct {
	value any
}

func (b Block) ArgValue() any {
	return b.value
}

var (
	Pending   = Block{"pending"}
	Latest    = Block{"latest"}
	Safe      = Block{"safe"}
	Finalized = Block{"finalized"}
)

// ByNumber references a canonical block by number.
func ByNumber(blockNum uint64) Block {
	return Block{rpc.BlockNumber(blockNum)}
}

// ByHash references a block by hash. Canonical or non-canonical blocks may be referenced.
func ByHash(hash common.Hash) Block {
	return Block{rpc.BlockNumberOrHashWithHash(hash, false)}
}
