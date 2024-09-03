package types

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
)

type TruncatedHash [20]byte

func TruncateHash(hash common.Hash) TruncatedHash {
	var truncated TruncatedHash
	copy(truncated[:], hash[0:20])
	return truncated
}

func (h TruncatedHash) String() string {
	return hex.EncodeToString(h[:])
}

type ExecutingMessage struct {
	Chain     uint32
	BlockNum  uint64
	LogIdx    uint32
	Timestamp uint64
	Hash      TruncatedHash
}
