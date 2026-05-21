package processors

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type EventDecoderFn func(*ethTypes.Log) (*messages.ExecutingMessage, error)

func MessageFromLog(l *ethTypes.Log) (*messages.Message, error) {
	if l.Address != params.InteropCrossL2InboxAddress {
		return nil, nil
	}
	if len(l.Topics) != 2 { // topics: event-id and payload-hash
		return nil, nil
	}
	if l.Topics[0] != messages.ExecutingMessageEventTopic {
		return nil, nil
	}
	var msg messages.Message
	if err := msg.DecodeEvent(l.Topics, l.Data); err != nil {
		return nil, fmt.Errorf("invalid executing message: %w", err)
	}
	return &msg, nil
}

func DecodeExecutingMessageLog(l *ethTypes.Log) (*messages.ExecutingMessage, error) {
	msg, err := MessageFromLog(l)
	if err != nil || msg == nil {
		return nil, err
	}
	return &messages.ExecutingMessage{
		ChainID:   msg.Identifier.ChainID,
		BlockNum:  msg.Identifier.BlockNumber,
		LogIdx:    msg.Identifier.LogIndex,
		Timestamp: msg.Identifier.Timestamp,
		Checksum:  msg.Checksum(),
	}, nil
}
