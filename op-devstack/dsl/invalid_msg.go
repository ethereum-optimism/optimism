package dsl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

// InvalidMsgFn is a function that takes a valid message and returns an invalid copy.
type InvalidMsgFn func(suptypes.Message) suptypes.Message

// MakeInvalidBlockNumber returns a copy of the message with an incremented block number.
func MakeInvalidBlockNumber(msg suptypes.Message) suptypes.Message {
	msg.Identifier.BlockNumber++
	return msg
}

// MakeInvalidChainID returns a copy of the message with an incremented chain ID.
func MakeInvalidChainID(msg suptypes.Message) suptypes.Message {
	chainIDBig := msg.Identifier.ChainID.ToBig()
	msg.Identifier.ChainID = eth.ChainIDFromBig(chainIDBig.Add(chainIDBig, big.NewInt(1)))
	return msg
}

// MakeInvalidLogIndex returns a copy of the message with an incremented log index.
func MakeInvalidLogIndex(msg suptypes.Message) suptypes.Message {
	msg.Identifier.LogIndex++
	return msg
}

// MakeInvalidOrigin returns a copy of the message with an incremented origin address.
func MakeInvalidOrigin(msg suptypes.Message) suptypes.Message {
	originBig := msg.Identifier.Origin.Big()
	msg.Identifier.Origin = common.BigToAddress(originBig.Add(originBig, big.NewInt(1)))
	return msg
}

// MakeInvalidTimestamp returns a copy of the message with an incremented timestamp.
func MakeInvalidTimestamp(msg suptypes.Message) suptypes.Message {
	msg.Identifier.Timestamp++
	return msg
}

// MakeInvalidPayloadHash returns a copy of the message with an incremented payload hash.
func MakeInvalidPayloadHash(msg suptypes.Message) suptypes.Message {
	hash := msg.PayloadHash.Big()
	hash.Add(hash, big.NewInt(1))
	msg.PayloadHash = common.BigToHash(hash)
	return msg
}
