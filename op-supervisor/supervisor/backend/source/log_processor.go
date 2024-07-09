package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/source/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type LogStorage interface {
	AddLog(logHash db.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *db.ExecutingMessage) error
}

type EventDecoder interface {
	DecodeExecutingMessageLog(log *types.Log) (db.ExecutingMessage, error)
}

type logProcessor struct {
	logStore     LogStorage
	eventDecoder EventDecoder
}

func newLogProcessor(logStore LogStorage) *logProcessor {
	return &logProcessor{
		logStore:     logStore,
		eventDecoder: contracts.NewCrossL2Inbox(),
	}
}

func (p *logProcessor) ProcessLogs(_ context.Context, block eth.L1BlockRef, rcpts types.Receipts) error {
	for _, rcpt := range rcpts {
		for _, l := range rcpt.Logs {
			logHash := logToHash(l)
			var execMsg *db.ExecutingMessage
			msg, err := p.eventDecoder.DecodeExecutingMessageLog(l)
			if err != nil && !errors.Is(err, contracts.ErrEventNotFound) {
				return fmt.Errorf("failed to decode executing message log: %w", err)
			} else if err == nil {
				execMsg = &msg
			}
			err = p.logStore.AddLog(logHash, block.ID(), block.Time, uint32(l.Index), execMsg)
			if err != nil {
				// TODO(optimism#11044): Need to roll back to the start of the block....
				return fmt.Errorf("failed to add log %d from block %v: %w", l.Index, block.ID(), err)
			}
		}
	}
	return nil
}

func logToHash(l *types.Log) db.TruncatedHash {
	payloadHash := crypto.Keccak256(logToPayload(l))
	msg := make([]byte, 0, 2*common.HashLength)
	msg = append(msg, l.Address.Bytes()...)
	msg = append(msg, payloadHash...)
	return db.TruncateHash(crypto.Keccak256Hash(msg))
}

func logToPayload(l *types.Log) []byte {
	msg := make([]byte, 0)
	for _, topic := range l.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	msg = append(msg, l.Data...)
	return msg
}
