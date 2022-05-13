package db

import (
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum/go-ethereum/common"
)

// L1BlockLocator contains the block locator for a L1 block.
type L1BlockLocator struct {
	Number uint64      `json:"number"`
	Hash   common.Hash `json:"hash"`
}

// L2BlockLocator contains the block locator for a L2 block.
type L2BlockLocator struct {
	Number uint64        `json:"number"`
	Hash   l2common.Hash `json:"hash"`
}
