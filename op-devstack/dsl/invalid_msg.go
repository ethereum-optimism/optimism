package dsl

import (
	"math/big"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// InvalidMsgFn is a function that takes a valid message and returns an invalid copy.
type InvalidMsgFn func(messages.Message) messages.Message

// MakeInvalidBlockNumber returns a copy of the message with an incremented block number.
func MakeInvalidBlockNumber(msg messages.Message) messages.Message {
	msg.Identifier.BlockNumber++
	return msg
}

// MakeInvalidChainID returns a copy of the message with an incremented chain ID.
func MakeInvalidChainID(msg messages.Message) messages.Message {
	chainIDBig := msg.Identifier.ChainID.ToBig()
	msg.Identifier.ChainID = eth.ChainIDFromBig(chainIDBig.Add(chainIDBig, big.NewInt(1)))
	return msg
}

// MakeInvalidLogIndex returns a copy of the message with an incremented log index.
func MakeInvalidLogIndex(msg messages.Message) messages.Message {
	msg.Identifier.LogIndex++
	return msg
}

// MakeInvalidOrigin returns a copy of the message with an incremented origin address.
func MakeInvalidOrigin(msg messages.Message) messages.Message {
	originBig := msg.Identifier.Origin.Big()
	msg.Identifier.Origin = common.BigToAddress(originBig.Add(originBig, big.NewInt(1)))
	return msg
}

// MakeInvalidTimestamp returns a copy of the message with an incremented timestamp.
func MakeInvalidTimestamp(msg messages.Message) messages.Message {
	msg.Identifier.Timestamp++
	return msg
}

// MakeInvalidPayloadHash returns a copy of the message with an incremented payload hash.
func MakeInvalidPayloadHash(msg messages.Message) messages.Message {
	hash := msg.PayloadHash.Big()
	hash.Add(hash, big.NewInt(1))
	msg.PayloadHash = common.BigToHash(hash)
	return msg
}
