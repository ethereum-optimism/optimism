package source

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestLogProcessor(t *testing.T) {
	ctx := context.Background()
	block1 := eth.L1BlockRef{Number: 100, Hash: common.Hash{0x11}, Time: 1111}
	t.Run("NoOutputWhenLogsAreEmpty", func(t *testing.T) {
		store := &stubLogStorage{}
		processor := newLogProcessor(store)

		err := processor.ProcessLogs(ctx, block1, types.Receipts{})
		require.NoError(t, err)
		require.Empty(t, store.logs)
	})

	t.Run("OutputLogs", func(t *testing.T) {
		rcpts := types.Receipts{
			{
				Logs: []*types.Log{
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
				Logs: []*types.Log{
					{
						Address: common.Address{0x33},
						Topics:  []common.Hash{{0xee}},
						Data:    []byte{0xff},
					},
				},
			},
		}
		store := &stubLogStorage{}
		processor := newLogProcessor(store)

		err := processor.ProcessLogs(ctx, block1, rcpts)
		require.NoError(t, err)
		expected := []storedLog{
			{
				block:     block1.ID(),
				timestamp: block1.Time,
				logIdx:    0,
				logHash:   logToHash(rcpts[0].Logs[0]),
				execMsg:   nil,
			},
			{
				block:     block1.ID(),
				timestamp: block1.Time,
				logIdx:    0,
				logHash:   logToHash(rcpts[0].Logs[1]),
				execMsg:   nil,
			},
			{
				block:     block1.ID(),
				timestamp: block1.Time,
				logIdx:    0,
				logHash:   logToHash(rcpts[1].Logs[0]),
				execMsg:   nil,
			},
		}
		require.Equal(t, expected, store.logs)
	})
}

func TestToLogHash(t *testing.T) {
	mkLog := func() *types.Log {
		return &types.Log{
			Address: common.Address{0xaa, 0xbb},
			Topics: []common.Hash{
				{0xcc},
				{0xdd},
			},
			Data:        []byte{0xee, 0xff, 0x00},
			BlockNumber: 12345,
			TxHash:      common.Hash{0x11, 0x22, 0x33},
			TxIndex:     4,
			BlockHash:   common.Hash{0x44, 0x55},
			Index:       8,
			Removed:     false,
		}
	}
	relevantMods := []func(l *types.Log){
		func(l *types.Log) { l.Address = common.Address{0xab, 0xcd} },
		func(l *types.Log) { l.Topics = append(l.Topics, common.Hash{0x12, 0x34}) },
		func(l *types.Log) { l.Topics = l.Topics[:len(l.Topics)-1] },
		func(l *types.Log) { l.Topics[0] = common.Hash{0x12, 0x34} },
		func(l *types.Log) { l.Data = append(l.Data, 0x56) },
		func(l *types.Log) { l.Data = l.Data[:len(l.Data)-1] },
		func(l *types.Log) { l.Data[0] = 0x45 },
	}
	irrelevantMods := []func(l *types.Log){
		func(l *types.Log) { l.BlockNumber = 987 },
		func(l *types.Log) { l.TxHash = common.Hash{0xab, 0xcd} },
		func(l *types.Log) { l.TxIndex = 99 },
		func(l *types.Log) { l.BlockHash = common.Hash{0xab, 0xcd} },
		func(l *types.Log) { l.Index = 98 },
		func(l *types.Log) { l.Removed = true },
	}
	refHash := logToHash(mkLog())
	// The log hash is stored in the database so test that it matches the actual value.
	// If this changes compatibility with existing databases may be affected
	expectedRefHash := db.TruncateHash(common.HexToHash("0x4e1dc08fddeb273275f787762cdfe945cf47bb4e80a1fabbc7a825801e81b73f"))
	require.Equal(t, expectedRefHash, refHash, "reference hash changed, check that database compatibility is not broken")

	// Check that the hash is changed when any data it should include changes
	for i, mod := range relevantMods {
		l := mkLog()
		mod(l)
		hash := logToHash(l)
		require.NotEqualf(t, refHash, hash, "expected relevant modification %v to affect the hash but it did not", i)
	}
	// Check that the hash is not changed when any data it should not include changes
	for i, mod := range irrelevantMods {
		l := mkLog()
		mod(l)
		hash := logToHash(l)
		require.Equal(t, refHash, hash, "expected irrelevant modification %v to not affect the hash but it did", i)
	}
}

type stubLogStorage struct {
	logs []storedLog
}

func (s *stubLogStorage) AddLog(logHash db.TruncatedHash, block eth.BlockID, timestamp uint64, logIdx uint32, execMsg *db.ExecutingMessage) error {
	s.logs = append(s.logs, storedLog{
		block:     block,
		timestamp: timestamp,
		logIdx:    logIdx,
		logHash:   logHash,
		execMsg:   execMsg,
	})
	return nil
}

type storedLog struct {
	block     eth.BlockID
	timestamp uint64
	logIdx    uint32
	logHash   db.TruncatedHash
	execMsg   *db.ExecutingMessage
}
