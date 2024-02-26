package superchain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type MessageSafetyLabel int

const (
	Invalid MessageSafetyLabel = iota
	Safe
	Finalized
)

type MessageIdentifier struct {
	Origin      common.Address
	BlockNumber *big.Int
	LogIndex    uint64
	Timestamp   uint64
	ChainId     *big.Int
}

func MessagePayloadBytes(log *types.Log) []byte {
	msg := []byte{}
	for _, topic := range log.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	return append(msg, log.Data...)
}
