package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/source/contracts"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	supTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type LogStorage interface {
	AddLog(chain supTypes.ChainID, logHash backendTypes.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *backendTypes.ExecutingMessage) error
}

type EventDecoder interface {
	DecodeExecutingMessageLog(log *ethTypes.Log) (backendTypes.ExecutingMessage, error)
}

type logProcessor struct {
	chain        supTypes.ChainID
	logStore     LogStorage
	eventDecoder EventDecoder
}

func newLogProcessor(chain supTypes.ChainID, logStore LogStorage) *logProcessor {
	return &logProcessor{
		chain:        chain,
		logStore:     logStore,
		eventDecoder: contracts.NewCrossL2Inbox(),
	}
}

func (p *logProcessor) ProcessLogs(_ context.Context, block eth.L1BlockRef, rcpts ethTypes.Receipts) error {
	for _, rcpt := range rcpts {
		for _, l := range rcpt.Logs {
			logHash := logToHash(l)
			var execMsg *backendTypes.ExecutingMessage
			msg, err := p.eventDecoder.DecodeExecutingMessageLog(l)
			if err != nil && !errors.Is(err, contracts.ErrEventNotFound) {
				return fmt.Errorf("failed to decode executing message log: %w", err)
			} else if err == nil {
				execMsg = &msg
			}
			err = p.logStore.AddLog(p.chain, logHash, block.ID(), block.Time, uint32(l.Index), execMsg)
			if err != nil {
				return fmt.Errorf("failed to add log %d from block %v: %w", l.Index, block.ID(), err)
			}
		}
	}
	return nil
}

func logToHash(l *ethTypes.Log) backendTypes.TruncatedHash {
	payloadHash := crypto.Keccak256(logToPayload(l))
	msg := make([]byte, 0, 2*common.HashLength)
	msg = append(msg, l.Address.Bytes()...)
	msg = append(msg, payloadHash...)
	return backendTypes.TruncateHash(crypto.Keccak256Hash(msg))
}

func logToPayload(l *ethTypes.Log) []byte {
	msg := make([]byte, 0)
	for _, topic := range l.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	msg = append(msg, l.Data...)
	return msg
}
