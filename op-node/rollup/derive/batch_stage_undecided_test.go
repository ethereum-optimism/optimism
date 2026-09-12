package derive

import (
	"context"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestBatchStage_UndecidedSpanBatchRetained pins the retention of Holocene span batches whose
// validity is BatchUndecided (see issue #22629): a span batch consumed from the previous stage
// must be retried once the missing L1 context or L2 fetch result becomes available, instead of
// being silently skipped. A skipped-but-valid span batch makes the local safe chain diverge from
// other nodes until a pipeline replay or restart.
func TestBatchStage_UndecidedSpanBatchRetained(t *testing.T) {
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

	safeBatch := b(cfg.L2ChainID, 22, l1[1])
	safeHead := singularBatchToBlockRef(t, safeBatch, 1)
	safePayload := singularBatchToPayload(t, safeBatch, 1)

	// Span batch starting exactly at the safe head: element at ts 22 matches the safe-head
	// block, element at ts 24 extends it.
	nextBatch := b(cfg.L2ChainID, 24, l1[1])
	span := func() *SpanBatch {
		return initializedSpanBatch([]*SingularBatch{safeBatch, nextBatch}, cfg.Genesis.L2Time, chainId)
	}

	newStage := func(lgr log.Logger, fetcher SafeBlockFetcher, spans ...Batch) (*BatchStage, *fakeBatchQueueInput) {
		input := &fakeBatchQueueInput{
			batches: spans,
			errors:  make([]error, len(spans)),
			origin:  l1[2],
		}
		stage := NewBatchStage(lgr, cfg, input, fetcher)
		_ = stage.Reset(context.Background(), l1[1], eth.SystemConfig{})
		return stage, input
	}

	t.Run("transient parent lookup failure", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		fetcher := newFakeSafeBlockFetcher()
		// The safe-head payload is present, but the parent block below the span start is not:
		// the prefix check cannot determine the span's parent yet.
		fetcher.addBlock(safeHead, &safePayload)
		stage, _ := newStage(lgr, fetcher, span())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "failed to fetch L2 block")
		logs.RequireMessageContainedOnce(t, "Undecided span batch, retaining for retry")

		// The parent becomes available; the same span must be retried and accepted, without
		// replaying the pipeline or re-reading the previous stage. The overlapped element is
		// excluded from the split, so the next batch (ts 24) is returned directly.
		fetcher.addBlock(parentRef, &parentPayload)
		batch, _, err = stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("transient overlap payload failure", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		fetcher := newFakeSafeBlockFetcher()
		// The parent ref is present so the prefix check succeeds, but the safe-head payload
		// needed by the overlap check is missing.
		fetcher.addBlock(parentRef, &parentPayload)
		fetcher.addBlock(safeHead, nil)
		stage, _ := newStage(lgr, fetcher, span())

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "failed to fetch L2 block payload")
		logs.RequireMessageContainedOnce(t, "Undecided span batch, retaining for retry")

		fetcher.addBlock(safeHead, &safePayload)
		batch, _, err = stage.NextBatch(context.Background(), safeHead)
		require.NoError(t, err)
		require.Equal(t, uint64(24), batch.Timestamp)
	})

	t.Run("permanent invalidity after undecided attempt drops the channel", func(t *testing.T) {
		lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
		fetcher := newFakeSafeBlockFetcher()
		fetcher.addBlock(safeHead, &safePayload)
		// The span's element at the safe-head height carries an extra transaction, so the
		// overlap check rejects it once it can run, but the parent lookup fails first, making
		// the first attempt undecided.
		rng := rand.New(rand.NewSource(99))
		signer := types.NewLondonSigner(chainId)
		extraTx, err2 := testutils.RandomTx(rng, common.Big1, signer).MarshalBinary()
		require.NoError(t, err2)
		conflictingElem := *safeBatch
		conflictingElem.Transactions = append(append([]hexutil.Bytes{}, safeBatch.Transactions...), extraTx)
		conflictingSpan := initializedSpanBatch([]*SingularBatch{&conflictingElem, nextBatch}, cfg.Genesis.L2Time, chainId)
		stage, _ := newStage(lgr, fetcher, conflictingSpan)

		batch, _, err := stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)

		// The parent becomes available; the overlap check now rejects the batch and flushes
		// the whole channel.
		fetcher.addBlock(parentRef, &parentPayload)
		batch, _, err = stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, NotEnoughData)
		require.Nil(t, batch)
		logs.RequireMessageContainedOnce(t, "Dropping invalid span batch, flushing channel")

		// The channel was flushed: nothing left to read, only empty-batch derivation remains.
		batch, _, err = stage.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, err, io.EOF)
		require.Nil(t, batch)
	})
}
