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

// ProcessLogs processes logs from a block and stores them in the log storage
// for any logs that are related to executing messages, they are decoded and stored
func (p *logProcessor) ProcessLogs(_ context.Context, block eth.L1BlockRef, rcpts ethTypes.Receipts) error {
	for _, rcpt := range rcpts {
		for _, l := range rcpt.Logs {
			// log hash represents the hash of *this* log as a potentially initiating message
			logHash := logToLogHash(l)
			var execMsg *backendTypes.ExecutingMessage
			msg, err := p.eventDecoder.DecodeExecutingMessageLog(l)
			if err != nil && !errors.Is(err, contracts.ErrEventNotFound) {
				return fmt.Errorf("failed to decode executing message log: %w", err)
			} else if err == nil {
				// if the log is an executing message, store the message
				execMsg = &msg
			}
			// executing messages have multiple entries in the database
			// they should start with the initiating message and then include the execution
			fmt.Println("p.chain", p.chain)
			err = p.logStore.AddLog(p.chain, logHash, block.ID(), block.Time, uint32(l.Index), execMsg)
			if err != nil {
				return fmt.Errorf("failed to add log %d from block %v: %w", l.Index, block.ID(), err)
			}
		}
	}
	return nil
}

// logToLogHash transforms a log into a hash that represents the log.
// it is the concatenation of the log's address and the hash of the log's payload,
// which is then hashed again. This is the hash that is stored in the log storage.
// The address is hashed into the payload hash to save space in the log storage,
// and because they represent paired data.
func logToLogHash(l *ethTypes.Log) backendTypes.TruncatedHash {
	payloadHash := crypto.Keccak256(logToMessagePayload(l))
	return payloadHashToLogHash(common.Hash(payloadHash), l.Address)
}

// logToMessagePayload is the data that is hashed to get the logHash
// it is the concatenation of the log's topics and data
// the implementation is based on the interop messaging spec
func logToMessagePayload(l *ethTypes.Log) []byte {
	msg := make([]byte, 0)
	for _, topic := range l.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	msg = append(msg, l.Data...)
	return msg
}

// payloadHashToLogHash converts the payload hash to the log hash
// it is the concatenation of the log's address and the hash of the log's payload,
// which is then hashed. This is the hash that is stored in the log storage.
// The logHash can then be used to traverse from the executing message
// to the log the referenced initiating message.
func payloadHashToLogHash(payloadHash common.Hash, addr common.Address) backendTypes.TruncatedHash {
	msg := make([]byte, 0, 2*common.HashLength)
	msg = append(msg, addr.Bytes()...)
	msg = append(msg, payloadHash.Bytes()...)
	return backendTypes.TruncateHash(crypto.Keccak256Hash(msg))
}
