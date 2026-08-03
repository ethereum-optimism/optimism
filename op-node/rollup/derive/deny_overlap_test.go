package derive

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// fakeDeniedHeights is a DeniedHeightsView backed by a static list of heights.
type fakeDeniedHeights struct {
	heights []uint64
	err     error
}

func (f *fakeDeniedHeights) DeniedHeightsInRange(minHeight, maxHeight uint64) ([]uint64, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []uint64
	for _, h := range f.heights {
		if h >= minHeight && h <= maxHeight {
			out = append(out, h)
		}
	}
	return out, nil
}

// TestBatchStage_DeniedOverlap pins the derive-level mechanism behind the interop
// block-replacement lineage bug: a span batch whose overlap section carries a block that has
// since been replaced with a deposits-only block (deny list entry at that height) must be
// dropped and its channel flushed, instead of having the denied element silently past-skipped
// and the remainder of the invalidated lineage spliced onto the canonical chain.
//
// The scenario mirrors the incident: the safe head is a deposits-only replacement at a denied
// height, and the incoming span batch is the original channel, whose element at that height
// still carries the invalidated transaction.
func TestBatchStage_DeniedOverlap(t *testing.T) {
	l1 := L1Chain([]uint64{10, 16, 22, 28})
	chainId := big.NewInt(1234)
	newConfig := func() *rollup.Config {
		cfg := &rollup.Config{
			Genesis: rollup.Genesis{
				L2Time: 20,
			},
			BlockTime:         2,
			MaxSequencerDrift: 600,
			SeqWindowSize:     1000,
			L2ChainID:         chainId,
		}
		cfg.ActivateAtGenesis(forks.Delta)
		return cfg
	}
	cfg := newConfig()

	parentBatch := b(cfg.L2ChainID, 20, l1[0])
	parentRef := singularBatchToBlockRef(t, parentBatch, 0)
	parentPayload := singularBatchToPayload(t, parentBatch, 0)

	// The canonical block at height 1 is a deposits-only replacement: no sequencer txs.
	replacementBatch := &SingularBatch{
		ParentHash: parentRef.Hash,
		Timestamp:  22,
		EpochNum:   rollup.Epoch(l1[1].Number),
		EpochHash:  l1[1].Hash,
	}
	safeHead := singularBatchToBlockRef(t, replacementBatch, 1)
	safePayload := singularBatchToPayload(t, replacementBatch, 1)

	newFetcher := func() *fakeSafeBlockFetcher {
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(safeHead, &safePayload)
		return fetcher
	}

	// The original channel's span batch: its element at the replaced height still carries the
	// invalidated transaction (b() adds one tx), followed by the remainder of that lineage.
	deniedLineageSpan := func() *SpanBatch {
		return initializedSpanBatch([]*SingularBatch{
			b(cfg.L2ChainID, 22, l1[1]), // same height as the replacement, carries the replaced tx
			b(cfg.L2ChainID, 24, l1[1]), // the lineage remainder that must not splice
		}, cfg.Genesis.L2Time, chainId)
	}
	// A batch agreeing with the canonical chain: deposits-only element at the replaced height.
	matchingLineageSpan := func() *SpanBatch {
		return initializedSpanBatch([]*SingularBatch{
			replacementBatch,
			b(cfg.L2ChainID, 24, l1[1]),
		}, cfg.Genesis.L2Time, chainId)
	}

	newStage := func(lgr log.Logger, span *SpanBatch, denied DeniedHeightsView, fetcher SafeBlockFetcher) *BatchStage {
		input := &fakeBatchQueueInput{
			batches: []Batch{span},
			errors:  []error{nil},
			origin:  l1[2],
		}
		stage := NewBatchStage(lgr, cfg, input, fetcher, denied)
		_ = stage.Reset(context.Background(), l1[1], eth.SystemConfig{})
		return stage
	}

	t.Run("conflicting overlap at denied height is dropped", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		stage := newStage(lgr, deniedLineageSpan(), &fakeDeniedHeights{heights: []uint64{1}}, newFetcher())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "overlapped block's tx count does not match")
		logs.RequireMessageContainedOnce(t, "Dropping invalid span batch, flushing channel (deny list overlap checks)")
	})

	t.Run("matching overlap at denied height is accepted", func(t *testing.T) {
		lgr := testlog.Logger(t, log.LevelCrit)
		stage := newStage(lgr, matchingLineageSpan(), &fakeDeniedHeights{heights: []uint64{1}}, newFetcher())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("no deny list entries skips re-validation", func(t *testing.T) {
		// Without deny list knowledge, the conflicting overlap is past-skipped and the tail
		// splices — this documents the content-blindness the deny-gated check exists to close.
		lgr := testlog.Logger(t, log.LevelCrit)
		stage := newStage(lgr, deniedLineageSpan(), &fakeDeniedHeights{}, newFetcher())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("nil deny list view skips re-validation", func(t *testing.T) {
		lgr := testlog.Logger(t, log.LevelCrit)
		stage := newStage(lgr, deniedLineageSpan(), nil, newFetcher())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("deny list read error fails open", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelError)
		stage := newStage(lgr, deniedLineageSpan(), &fakeDeniedHeights{err: errors.New("boom")}, newFetcher())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
		logs.RequireMessageContainedOnce(t, "Failed to read deny list heights")
	})

	t.Run("payload fetch error is undecided", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(safeHead, nil) // ref known, payload unavailable
		stage := newStage(lgr, deniedLineageSpan(), &fakeDeniedHeights{heights: []uint64{1}}, fetcher)

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "Undecided span batch (deny list overlap checks)")
	})
}
