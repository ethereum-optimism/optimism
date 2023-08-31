package bridge

import "github.com/ethereum/go-ethereum/common"

type logKey struct {
	BlockHash common.Hash
	LogIndex  uint64
}
