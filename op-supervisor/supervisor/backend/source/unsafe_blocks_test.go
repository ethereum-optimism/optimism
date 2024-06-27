package source

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestUnsafeBlocksStage(t *testing.T) {
	t.Run("IgnoreEventsAtOrPriorToStartingHead", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		out := make(chan PipelineEvent, 10)
		client := &stubBlockByNumberSource{}
		stage := NewUnsafeBlocksStage(logger, client, eth.L1BlockRef{Number: 100})
		stage.Handle(ctx, UnsafeHeadEvent{Block: eth.L1BlockRef{Number: 100}}, out)
		stage.Handle(ctx, UnsafeHeadEvent{Block: eth.L1BlockRef{Number: 99}}, out)

		require.Empty(t, out)
		require.Zero(t, client.calls)
	})

	t.Run("OutputNewHeadsWithNoMissedBlocks", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		out := make(chan PipelineEvent, 10)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block1 := eth.L1BlockRef{Number: 101}
		block2 := eth.L1BlockRef{Number: 102}
		block3 := eth.L1BlockRef{Number: 103}
		stage := NewUnsafeBlocksStage(logger, client, block0)
		stage.Handle(ctx, UnsafeHeadEvent{Block: block1}, out)
		require.NotEmpty(t, out)
		require.Equal(t, UnsafeBlockEvent{block1}, <-out)
		stage.Handle(ctx, UnsafeHeadEvent{Block: block2}, out)
		require.Equal(t, UnsafeBlockEvent{block2}, <-out)
		stage.Handle(ctx, UnsafeHeadEvent{Block: block3}, out)
		require.Equal(t, UnsafeBlockEvent{block3}, <-out)

		require.Zero(t, client.calls, "should not need to request block info")
	})

	t.Run("IgnoreEventsAtOrPriorToPreviousHead", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		out := make(chan PipelineEvent, 10)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block1 := eth.L1BlockRef{Number: 101}
		stage := NewUnsafeBlocksStage(logger, client, block0)
		stage.Handle(ctx, UnsafeHeadEvent{Block: block1}, out)
		require.NotEmpty(t, out)
		require.Equal(t, UnsafeBlockEvent{block1}, <-out)

		stage.Handle(ctx, UnsafeHeadEvent{Block: block0}, out)
		stage.Handle(ctx, UnsafeHeadEvent{Block: block1}, out)
		require.Empty(t, out)

		require.Zero(t, client.calls, "should not need to request block info")
	})

	t.Run("OutputSkippedBlocks", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		out := make(chan PipelineEvent, 10)
		client := &stubBlockByNumberSource{}
		block0 := eth.L1BlockRef{Number: 100}
		block3 := eth.L1BlockRef{Number: 103}
		stage := NewUnsafeBlocksStage(logger, client, block0)

		stage.Handle(ctx, UnsafeHeadEvent{Block: block3}, out)
		// should output block 1, 2 and 3
		require.Len(t, out, 3)
		require.Equal(t, UnsafeBlockEvent{makeBlockRef(101)}, <-out)
		require.Equal(t, UnsafeBlockEvent{makeBlockRef(102)}, <-out)
		require.Equal(t, UnsafeBlockEvent{block3}, <-out)

		require.Equal(t, 2, client.calls, "should only request the two missing blocks")
	})

	t.Run("DoNotUpdateLastBlockOnError", func(t *testing.T) {
		ctx := context.Background()
		logger := testlog.Logger(t, log.LvlInfo)
		out := make(chan PipelineEvent, 10)
		client := &stubBlockByNumberSource{err: errors.New("boom")}
		block0 := eth.L1BlockRef{Number: 100}
		block3 := eth.L1BlockRef{Number: 103}
		stage := NewUnsafeBlocksStage(logger, client, block0)

		stage.Handle(ctx, UnsafeHeadEvent{Block: block3}, out)
		require.Empty(t, out, "should not update any blocks because backfill errored")

		client.err = nil
		stage.Handle(ctx, UnsafeHeadEvent{Block: block3}, out)
		// should output block 1, 2 and 3
		require.Len(t, out, 3)
		require.Equal(t, UnsafeBlockEvent{makeBlockRef(101)}, <-out)
		require.Equal(t, UnsafeBlockEvent{makeBlockRef(102)}, <-out)
		require.Equal(t, UnsafeBlockEvent{block3}, <-out)
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

func makeBlockRef(number uint64) eth.L1BlockRef {
	return eth.L1BlockRef{
		Number:     number,
		Hash:       common.Hash{byte(number)},
		ParentHash: common.Hash{byte(number - 1)},
		Time:       number * 1000,
	}
}
