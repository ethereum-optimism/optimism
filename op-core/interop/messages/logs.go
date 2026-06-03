package messages

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// MessageFromLog returns the executing message decoded from a CrossL2Inbox log,
// or nil if the log is not an executing message.
func MessageFromLog(l *ethTypes.Log) (*Message, error) {
	if l.Address != params.InteropCrossL2InboxAddress {
		return nil, nil
	}
	if len(l.Topics) != 2 { // topics: event-id and payload-hash
		return nil, nil
	}
	if l.Topics[0] != ExecutingMessageEventTopic {
		return nil, nil
	}
	var msg Message
	if err := msg.DecodeEvent(l.Topics, l.Data); err != nil {
		return nil, fmt.Errorf("invalid executing message: %w", err)
	}
	return &msg, nil
}

// DecodeExecutingMessageLog returns the ExecutingMessage decoded from a CrossL2Inbox log,
// or nil if the log is not an executing message.
func DecodeExecutingMessageLog(l *ethTypes.Log) (*ExecutingMessage, error) {
	msg, err := MessageFromLog(l)
	if err != nil || msg == nil {
		return nil, err
	}
	return &ExecutingMessage{
		ChainID:   msg.Identifier.ChainID,
		BlockNum:  msg.Identifier.BlockNumber,
		LogIdx:    msg.Identifier.LogIndex,
		Timestamp: msg.Identifier.Timestamp,
		Checksum:  msg.Checksum(),
	}, nil
}

// LogToLogHash transforms a log into a hash that represents the log.
// It is the concatenation of the log's address and the hash of the log's payload,
// which is then hashed again. This is the hash that is stored in the log storage.
// The address is hashed into the payload hash to save space in the log storage,
// and because they represent paired data.
func LogToLogHash(l *ethTypes.Log) common.Hash {
	payloadHash := crypto.Keccak256Hash(LogToMessagePayload(l))
	return PayloadHashToLogHash(payloadHash, l.Address)
}
