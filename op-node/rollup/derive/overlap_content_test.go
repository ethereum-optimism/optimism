package derive

import (
	"context"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestBatchStage_OverlapContent pins the derive-level mechanism behind the interop
// block-replacement lineage bug: a span batch whose overlap section disagrees with the safe
// chain — here, the original channel of a lineage whose block has since been replaced with a
// deposits-only block — must be dropped and its channel flushed, instead of having the
// conflicting element silently past-skipped and the remainder of the invalidated lineage
// spliced onto the canonical chain.
func TestBatchStage_OverlapContent(t *testing.T) {
	l1 := L1Chain([]uint64{10, 16, 22, 28})
	chainId := big.NewInt(1234)
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

	newStage := func(lgr log.Logger, fetcher SafeBlockFetcher, spans ...Batch) *BatchStage {
		input := &fakeBatchQueueInput{
			batches: spans,
			errors:  make([]error, len(spans)),
			origin:  l1[2],
		}
		stage := NewBatchStage(lgr, cfg, input, fetcher)
		_ = stage.Reset(context.Background(), l1[1], eth.SystemConfig{})
		return stage
	}

	t.Run("conflicting overlap is dropped", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		// The sentinel span behind the conflicting one would be accepted if it were ever read:
		// it only surviving the drop would prove the flush was logged but not performed.
		stage := newStage(lgr, newFetcher(), deniedLineageSpan(), matchingLineageSpan())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "overlapped block's tx count does not match")
		logs.RequireMessageContainedOnce(t, "Dropping invalid span batch, flushing channel (span batch overlap checks)")

		// The flush must have discarded the unread sentinel along with the rest of the channel.
		batch, _, err = stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, io.EOF)
		require.Nil(t, batch)
	})

	t.Run("matching overlap is accepted", func(t *testing.T) {
		lgr := testlog.Logger(t, log.LevelCrit)
		stage := newStage(lgr, newFetcher(), matchingLineageSpan())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("multi-block overlap: early match, late mismatch", func(t *testing.T) {
		// A two-block overlap where the first overlapped element matches the safe chain and
		// only the second diverges, exercising the comparison loop beyond its first iteration.
		safe1Batch := b(cfg.L2ChainID, 22, l1[1])
		safe1Ref := singularBatchToBlockRef(t, safe1Batch, 1)
		safe1Payload := singularBatchToPayload(t, safe1Batch, 1)
		safe2Batch := b(cfg.L2ChainID, 24, l1[1])
		safe2Ref := singularBatchToBlockRef(t, safe2Batch, 2)
		safe2Payload := singularBatchToPayload(t, safe2Batch, 2)
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(safe1Ref, &safe1Payload)
		fetcher.addBlock(safe2Ref, &safe2Payload)

		// Height 2's element carries no transactions, unlike canonical safe2Batch.
		divergent24 := &SingularBatch{
			ParentHash: safe1Ref.Hash,
			Timestamp:  24,
			EpochNum:   rollup.Epoch(l1[1].Number),
			EpochHash:  l1[1].Hash,
		}
		lateMismatchSpan := initializedSpanBatch([]*SingularBatch{
			b(cfg.L2ChainID, 22, l1[1]), // matches the safe chain at height 1
			divergent24,                 // diverges at height 2
			b(cfg.L2ChainID, 26, l1[1]), // the tail that must not splice
		}, cfg.Genesis.L2Time, chainId)
		sentinel := initializedSpanBatch([]*SingularBatch{
			b(cfg.L2ChainID, 26, l1[1]),
		}, cfg.Genesis.L2Time, chainId)

		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		stage := newStage(lgr, fetcher, lateMismatchSpan, sentinel)

		batch, _, err := stage.NextBatch(context.Background(), safe2Ref)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "overlapped block's tx count does not match")
		logs.RequireMessageContainedOnce(t, "Dropping invalid span batch, flushing channel (span batch overlap checks)")

		// The flush must have discarded the unread sentinel along with the rest of the channel.
		batch, _, err = stage.NextBatch(context.Background(), safe2Ref)
		require.ErrorIs(t, err, io.EOF)
		require.Nil(t, batch)
	})

	t.Run("overlapped origin mismatch is dropped", func(t *testing.T) {
		// A fixture where the overlapped element's transactions match the canonical block
		// (b() derives the tx from the timestamp), but its L1 origin number does not.
		txSafeBatch := b(cfg.L2ChainID, 22, l1[1])
		txSafeHead := singularBatchToBlockRef(t, txSafeBatch, 1)
		txSafePayload := singularBatchToPayload(t, txSafeBatch, 1)
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(txSafeHead, &txSafePayload)
		originMismatchSpan := initializedSpanBatch([]*SingularBatch{
			b(cfg.L2ChainID, 22, l1[0]), // same txs as the canonical block, outdated origin
			b(cfg.L2ChainID, 24, l1[1]),
		}, cfg.Genesis.L2Time, chainId)

		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		stage := newStage(lgr, fetcher, originMismatchSpan)

		batch, _, err := stage.NextBatch(context.Background(), txSafeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "overlapped block's L1 origin number does not match")
		logs.RequireMessageContainedOnce(t, "Dropping invalid span batch, flushing channel (span batch overlap checks)")
	})

	t.Run("payload fetch error is undecided", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(safeHead, nil) // ref known, payload unavailable
		stage := newStage(lgr, fetcher, deniedLineageSpan())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "Undecided span batch (span batch overlap checks)")
	})
}
