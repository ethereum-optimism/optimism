package source

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var processorChainID = types.ChainIDFromUInt64(4)

func TestUnsafeBlocksStage(t *testing.T) {
	t.Run("IgnoreEventsAtOrPriorToStartingHead", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		processor := &stubBlockProcessor{}
		stage := NewChainProcessor(logger, client, processorChainID, eth.L1BlockRef{Number: 100}, processor, &stubRewinder{})
		stage.OnNewHead(ctx, eth.L1BlockRef{Number: 100})
		stage.OnNewHead(ctx, eth.L1BlockRef{Number: 99})

		require.Empty(t, processor.processed)
		require.Zero(t, client.calls)
	})

	t.Run("OutputNewHeadsWithNoMissedBlocks", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block1 := eth.L1BlockRef{Number: 101}
		block2 := eth.L1BlockRef{Number: 102}
		block3 := eth.L1BlockRef{Number: 103}
		processor := &stubBlockProcessor{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, &stubRewinder{})
		stage.OnNewHead(ctx, block1)
		require.Equal(t, []eth.L1BlockRef{block1}, processor.processed)
		stage.OnNewHead(ctx, block2)
		require.Equal(t, []eth.L1BlockRef{block1, block2}, processor.processed)
		stage.OnNewHead(ctx, block3)
		require.Equal(t, []eth.L1BlockRef{block1, block2, block3}, processor.processed)

		require.Zero(t, client.calls, "should not need to request block info")
	})

	t.Run("IgnoreEventsAtOrPriorToPreviousHead", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block1 := eth.L1BlockRef{Number: 101}
		processor := &stubBlockProcessor{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, &stubRewinder{})
		stage.OnNewHead(ctx, block1)
		require.NotEmpty(t, processor.processed)
		require.Equal(t, []eth.L1BlockRef{block1}, processor.processed)

		stage.OnNewHead(ctx, block0)
		stage.OnNewHead(ctx, block1)
		require.Equal(t, []eth.L1BlockRef{block1}, processor.processed)

		require.Zero(t, client.calls, "should not need to request block info")
	})

	t.Run("OutputSkippedBlocks", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block3 := eth.L1BlockRef{Number: 103}
		processor := &stubBlockProcessor{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, &stubRewinder{})

		stage.OnNewHead(ctx, block3)
		require.Equal(t, []eth.L1BlockRef{makeBlockRef(101), makeBlockRef(102), block3}, processor.processed)

		require.Equal(t, 2, client.calls, "should only request the two missing blocks")
	})

	t.Run("DoNotUpdateLastBlockOnFetchError", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{err: errors.New("boom")}
		block0 := eth.L1BlockRef{Number: 100}
		block3 := eth.L1BlockRef{Number: 103}
		processor := &stubBlockProcessor{}
		rewinder := &stubRewinder{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, rewinder)

		stage.OnNewHead(ctx, block3)
		require.Empty(t, processor.processed, "should not update any blocks because backfill failed")

		client.err = nil
		stage.OnNewHead(ctx, block3)
		require.Equal(t, []eth.L1BlockRef{makeBlockRef(101), makeBlockRef(102), block3}, processor.processed)
		require.False(t, rewinder.rewindCalled, "should not rewind because no logs could have been written")
	})

	t.Run("DoNotUpdateLastBlockOnProcessorError", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block3 := eth.L1BlockRef{Number: 103}
		processor := &stubBlockProcessor{err: errors.New("boom")}
		rewinder := &stubRewinder{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, rewinder)

		stage.OnNewHead(ctx, block3)
		require.Equal(t, []eth.L1BlockRef{makeBlockRef(101)}, processor.processed, "Attempted to process block 101")
		require.Equal(t, block0.Number, rewinder.rewoundTo, "should rewind to block before error")

		processor.err = nil
		stage.OnNewHead(ctx, block3)
		// Attempts to process block 101 again, then carries on
		require.Equal(t, []eth.L1BlockRef{makeBlockRef(101), makeBlockRef(101), makeBlockRef(102), block3}, processor.processed)
	})

	t.Run("RewindWhenNewHeadProcessingFails", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block1 := eth.L1BlockRef{Number: 101}
		processor := &stubBlockProcessor{err: errors.New("boom")}
		rewinder := &stubRewinder{}
		stage := NewChainProcessor(logger, client, processorChainID, block0, processor, rewinder)

		// No skipped blocks
		stage.OnNewHead(ctx, block1)
		require.Equal(t, []eth.L1BlockRef{block1}, processor.processed, "Attempted to process block 101")
		require.Equal(t, block0.Number, rewinder.rewoundTo, "should rewind to block before error")
	})
}

type stubBlockByNumberSource struct {
	calls int
	err   error
}

func (s *stubBlockByNumberSource) L1BlockRefByNumber(_ context.Context, number uint64) (eth.L1BlockRef, error) {
	s.calls++
	if s.err != nil {
		return eth.L1BlockRef{}, s.err
	}
	return makeBlockRef(number), nil
}

type stubBlockProcessor struct {
	processed []eth.L1BlockRef
	err       error
}

func (s *stubBlockProcessor) ProcessBlock(_ context.Context, block eth.L1BlockRef) error {
	s.processed = append(s.processed, block)
	return s.err
}

func makeBlockRef(number uint64) eth.L1BlockRef {
	return eth.L1BlockRef{
		Number:     number,
		Hash:       common.Hash{byte(number)},
		ParentHash: common.Hash{byte(number - 1)},
		Time:       number * 1000,
	}
}

type stubRewinder struct {
	rewoundTo    uint64
	rewindCalled bool
}

func (s *stubRewinder) Rewind(chainID types.ChainID, headBlockNum uint64) error {
	if chainID != processorChainID {
		return fmt.Errorf("chainID mismatch, expected %v but was %v", processorChainID, chainID)
	}
	s.rewoundTo = headBlockNum
	s.rewindCalled = true
	return nil
}
