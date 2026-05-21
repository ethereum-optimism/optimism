package processors

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var logProcessorChainID = eth.ChainIDFromUInt64(4)

func TestLogProcessor(t *testing.T) {
	ctx := context.Background()
	block1 := eth.BlockRef{
		ParentHash: common.Hash{0x42},
		Number:     100,
		Hash:       common.Hash{0x11},
		Time:       1111,
	}

	t.Run("NoOutputWhenLogsAreEmpty", func(t *testing.T) {
		store := &stubLogStorage{}
		processor := NewLogProcessor(logProcessorChainID, store)

		err := processor.ProcessLogs(ctx, block1, ethTypes.Receipts{})
		require.NoError(t, err)
		require.Empty(t, store.logs)
	})

	t.Run("OutputLogs", func(t *testing.T) {
		rcpts := ethTypes.Receipts{
			{
				Logs: []*ethTypes.Log{
					{
						Address: common.Address{0x11},
						Topics:  []common.Hash{{0xaa}},
						Data:    []byte{0xbb},
					},
					{
						Address: common.Address{0x22},
						Topics:  []common.Hash{{0xcc}},
						Data:    []byte{0xdd},
					},
				},
			},
			{
				Logs: []*ethTypes.Log{
					{
						Address: common.Address{0x33},
						Topics:  []common.Hash{{0xee}},
						Data:    []byte{0xff},
					},
				},
			},
		}
		store := &stubLogStorage{}
		processor := NewLogProcessor(logProcessorChainID, store)

		err := processor.ProcessLogs(ctx, block1, rcpts)
		require.NoError(t, err)
		expectedLogs := []storedLog{
			{
				parent:  block1.ParentID(),
				logIdx:  0,
				logHash: messages.LogToLogHash(rcpts[0].Logs[0]),
				execMsg: nil,
			},
			{
				parent:  block1.ParentID(),
				logIdx:  0,
				logHash: messages.LogToLogHash(rcpts[0].Logs[1]),
				execMsg: nil,
			},
			{
				parent:  block1.ParentID(),
				logIdx:  0,
				logHash: messages.LogToLogHash(rcpts[1].Logs[0]),
				execMsg: nil,
			},
		}
		require.Equal(t, expectedLogs, store.logs)

		expectedBlocks := []storedSeal{
			{
				parent:    block1.ParentHash,
				block:     block1.ID(),
				timestamp: block1.Time,
			},
		}
		require.Equal(t, expectedBlocks, store.seals)
	})

	t.Run("IncludeExecutingMessage", func(t *testing.T) {
		rcpts := ethTypes.Receipts{
			{
				Logs: []*ethTypes.Log{
					{
						Address: predeploys.CrossL2InboxAddr,
						Topics:  []common.Hash{},
						Data:    []byte{0xff},
					},
				},
			},
		}
		execMsg := &messages.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(4),
			BlockNum:  6,
			LogIdx:    8,
			Timestamp: 10,
			Checksum:  messages.MessageChecksum{0xaa},
		}
		store := &stubLogStorage{}
		processor := NewLogProcessor(eth.ChainID{4}, store).(*logProcessor)
		processor.eventDecoder = func(l *ethTypes.Log) (*messages.ExecutingMessage, error) {
			require.Equal(t, rcpts[0].Logs[0], l)
			return execMsg, nil
		}

		err := processor.ProcessLogs(ctx, block1, rcpts)
		require.NoError(t, err)
		expected := []storedLog{
			{
				parent:  block1.ParentID(),
				logIdx:  0,
				logHash: messages.LogToLogHash(rcpts[0].Logs[0]),
				execMsg: execMsg,
			},
		}
		require.Equal(t, expected, store.logs)

		expectedBlocks := []storedSeal{
			{
				parent:    block1.ParentHash,
				block:     block1.ID(),
				timestamp: block1.Time,
			},
		}
		require.Equal(t, expectedBlocks, store.seals)
	})
}

type stubLogStorage struct {
	logs  []storedLog
	seals []storedSeal
}

func (s *stubLogStorage) SealBlock(chainID eth.ChainID, block eth.BlockRef) error {
	if logProcessorChainID != chainID {
		return fmt.Errorf("chain id mismatch, expected %v but got %v", logProcessorChainID, chainID)
	}
	s.seals = append(s.seals, storedSeal{
		parent:    block.ParentHash,
		block:     block.ID(),
		timestamp: block.Time,
	})
	return nil
}

func (s *stubLogStorage) AddLog(chainID eth.ChainID, logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *messages.ExecutingMessage) error {
	if logProcessorChainID != chainID {
		return fmt.Errorf("chain id mismatch, expected %v but got %v", logProcessorChainID, chainID)
	}
	s.logs = append(s.logs, storedLog{
		parent:  parentBlock,
		logIdx:  logIdx,
		logHash: logHash,
		execMsg: execMsg,
	})
	return nil
}

type storedSeal struct {
	parent    common.Hash
	block     eth.BlockID
	timestamp uint64
}

type storedLog struct {
	parent  eth.BlockID
	logIdx  uint32
	logHash common.Hash
	execMsg *messages.ExecutingMessage
}
