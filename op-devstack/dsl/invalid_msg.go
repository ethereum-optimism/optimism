package dsl

import (
	"math/big"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum/go-ethereum/common"
)

// MakeInvalidLogIndex returns a copy of the message with an incremented log index.
func MakeInvalidLogIndex(msg messages.Message) messages.Message {
	msg.Identifier.LogIndex++
	return msg
}

// MakeInvalidPayloadHash returns a copy of the message with an incremented payload hash.
func MakeInvalidPayloadHash(msg messages.Message) messages.Message {
	hash := msg.PayloadHash.Big()
	hash.Add(hash, big.NewInt(1))
	msg.PayloadHash = common.BigToHash(hash)
	return msg
}
