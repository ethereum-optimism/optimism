package db

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrLogOutOfOrder  = errors.New("log out of order")
	ErrDataCorruption = errors.New("data corruption")
	ErrNotFound       = errors.New("not found")
)

type TruncatedHash [20]byte

func TruncateHash(hash common.Hash) TruncatedHash {
	var truncated TruncatedHash
	copy(truncated[:], hash[0:20])
	return truncated
}

type ExecutingMessage struct {
	Chain     uint32
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Hash      TruncatedHash
}
