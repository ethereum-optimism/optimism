package db

import (
	"github.com/ethereum/go-ethereum/common"
)

// BlockLocator contains the block number and hash. It can
// uniquely identify an Ethereum block
type BlockLocator struct {
	Number uint64      `json:"number"`
	Hash   common.Hash `json:"hash"`
}
