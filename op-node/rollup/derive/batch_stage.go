package derive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// The BatchStage implements the Holocene-derivation batch stage.
//
// It pulls batches (singular or span) from the previous [ChannelInReader] stage, validates them and
// applies the strict ordering requirements of Holocene. Valid batches are forwarded to the next
// stage.
//
// It also generates empty batches when the sequencing window has passed.
//
// Upon Holocene activation, it replaces the [BatchQueue].
type BatchStage struct {
	baseBatchStage
}

var _ SingularBatchProvider = (*BatchStage)(nil)

func NewBatchStage(log log.Logger, cfg *rollup.Config, prev NextBatchProvider, l2 SafeBlockFetcher) *BatchStage {
	return &BatchStage{baseBatchStage: newBaseBatchStage(log, cfg, prev, l2)}
}

// NewBatchStageWithMetrics creates a BatchStage with derivation metrics instrumentation.
func NewBatchStageWithMetrics(log log.Logger, cfg *rollup.Config, prev NextBatchProvider, l2 SafeBlockFetcher, derivMetrics DerivationMetrics) *BatchStage {
	return &BatchStage{baseBatchStage: newBaseBatchStageWithMetrics(log, cfg, prev, l2, derivMetrics)}
}

func (bs *BatchStage) Reset(_ context.Context, base eth.L1BlockRef, _ eth.SystemConfig) error {
	bs.reset(base)
	return io.EOF
}

func (bs *BatchStage) FlushChannel() {
	bs.nextSpan = bs.nextSpan[:0]
	bs.prev.FlushChannel()
}

func (bs *BatchStage) NextBatch(ctx context.Context, parent eth.L2BlockRef) (*SingularBatch, bool, error) {
	start := time.Now()
	var result string
	defer func() {
		bs.derivMetrics.RecordStageProcessing(StageBatchQueue, time.Since(start), result)
	}()

	// Record queue depth (cached span batches)
	bs.derivMetrics.RecordStageQueueDepth(StageBatchQueue, len(bs.nextSpan))

	// with Holocene, we can always update (and prune) the origins because we don't backwards-invalidate.
	bs.updateOrigins(parent)

	// If origin behind (or at parent), we drain previous stage(s), and then return.
	// Note that a channel from the parent's L1 origin block can only contain past batches, so we
	// can just skip them.
	if bs.originBehind(parent) || parent.L1Origin.Number == bs.origin.Number {
		waitStart := time.Now()
		_, err := bs.prev.NextBatch(ctx)
		bs.derivMetrics.RecordStageWaitTime(StageBatchQueue, time.Since(waitStart))

		if err != nil {
			// includes io.EOF and NotEnoughData
			result = ResultEmpty
			return nil, false, err
		}
		// continue draining
		result = ResultEmpty
		return nil, false, NotEnoughData
	}

	if len(bs.l1Blocks) < 2 {
		// This can only happen if derivation erroneously doesn't start at a safe head.
		// By now, the L1 origin of the first safe head and the following L1 block must be in the
		// l1Blocks.
		result = ResultError
		return nil, false, NewCriticalError(fmt.Errorf(
			"unexpected low buffered origin count, origin: %v, parent: %v", bs.origin, parent))
	}

	// Note: epoch origin can now be one block ahead of the L2 Safe Head
	// This is in the case where we auto generate all batches in an epoch & advance the epoch in
	// deriveNextEmptyBatch but don't advance the L2 Safe Head's epoch
	if epoch := bs.l1Blocks[0]; parent.L1Origin != epoch.ID() && parent.L1Origin.Number != epoch.Number-1 {
		result = ResultError
		bs.derivMetrics.RecordPipelineReset(ResetReasonInvalidBatch)
		return nil, false, NewResetError(fmt.Errorf("buffered L1 chain epoch %s in batch queue does not match safe head origin %s", epoch, parent.L1Origin))
	}

	batch, err := bs.nextSingularBatchCandidate(ctx, parent)
	if err == io.EOF {
		// We only consider empty batch generation after we've drained all batches from the local
		// span batch queue and the previous stage.
		empty, err := bs.deriveNextEmptyBatch(ctx, true, parent)
		// An empty batch always advances the (local) safe head.
		if err == nil {
			result = ResultSuccess
			bs.derivMetrics.RecordStageItemProcessed(StageBatchQueue, ResultSuccess)
		} else {
			result = ResultEmpty
		}
		return empty, true, err
	} else if err != nil {
		result = ResultError
		return nil, false, err
	}

	// check candidate validity
	validity := checkSingularBatch(bs.config, bs.Log(), bs.l1Blocks, parent, batch, bs.origin)
	switch validity {
	case BatchAccept: // continue
		batch.LogContext(bs.Log()).Debug("Found next singular batch")
		// BatchStage is only used with Holocene, where blocks immediately become (local) safe
		result = ResultSuccess
		bs.derivMetrics.RecordStageItemProcessed(StageBatchQueue, ResultSuccess)
		return batch, true, nil
	case BatchPast:
		batch.LogContext(bs.Log()).Warn("Dropping past singular batch")
		// NotEnoughData to read in next batch until we're through all past batches
		result = ResultFiltered
		bs.derivMetrics.RecordStageItemProcessed(StageBatchQueue, ResultFiltered)
		return nil, false, NotEnoughData
	case BatchDrop: // drop, flush, move onto next channel
		batch.LogContext(bs.Log()).Warn("Dropping invalid singular batch, flushing channel")
		bs.FlushChannel()
		// NotEnoughData will cause derivation from previous stages until they're empty, at which
		// point empty batch derivation will happen.
		result = ResultFiltered
		bs.derivMetrics.RecordStageItemProcessed(StageBatchQueue, ResultFiltered)
		return nil, false, NotEnoughData
	case BatchUndecided: // l2 fetcher error, try again
		batch.LogContext(bs.Log()).Warn("Undecided span batch")
		result = ResultEmpty
		return nil, false, NotEnoughData
	case BatchFuture: // panic, can't happen
		result = ResultError
		return nil, false, NewCriticalError(fmt.Errorf("impossible batch validity: %v", validity))
	default:
		result = ResultError
		return nil, false, NewCriticalError(fmt.Errorf("unknown batch validity type: %d", validity))
	}
}

func (bs *BatchStage) nextSingularBatchCandidate(ctx context.Context, parent eth.L2BlockRef) (*SingularBatch, error) {
	// First check for next span-derived batch
	nextBatch, _ := bs.nextFromSpanBatch(parent)

	if nextBatch != nil {
		return nextBatch, nil
	}

	// If the next batch is a singular batch, we forward it as the candidate.
	// If it is a span batch, we check its validity and then forward its first singular batch.
	batch, err := bs.prev.NextBatch(ctx)
	if err != nil { // includes io.EOF
		return nil, err
	}
	switch typ := batch.GetBatchType(); typ {
	case SingularBatchType:
		singularBatch, ok := batch.AsSingularBatch()
		if !ok {
			return nil, NewCriticalError(errors.New("failed type assertion to SingularBatch"))
		}
		return singularBatch, nil
	case SpanBatchType:
		spanBatch, ok := batch.AsSpanBatch()
		if !ok {
			return nil, NewCriticalError(errors.New("failed type assertion to SpanBatch"))
		}

		validity, _ := checkSpanBatchPrefix(ctx, bs.config, bs.Log(), bs.l1Blocks, parent, spanBatch, bs.origin, bs.l2)
		switch validity {
		case BatchAccept: // continue
			spanBatch.LogContext(bs.Log()).Info("Found next valid span batch")
		case BatchPast:
			spanBatch.LogContext(bs.Log()).Warn("Dropping past span batch")
			// NotEnoughData to read in next batch until we're through all past batches
			return nil, NotEnoughData
		case BatchDrop: // drop, try next
			spanBatch.LogContext(bs.Log()).Warn("Dropping invalid span batch, flushing channel (span batch prefix checks)")
			bs.FlushChannel()
			return nil, NotEnoughData
		case BatchUndecided: // l2 fetcher error, try again
			spanBatch.LogContext(bs.Log()).Warn("Undecided span batch")
			return nil, NotEnoughData
		case BatchFuture: // can't happen with Holocene
			return nil, NewCriticalError(errors.New("impossible future batch validity"))
		}

		// If next batch is SpanBatch, convert it to SingularBatches.
		singularBatches, err := spanBatch.GetSingularBatches(bs.l1Blocks, parent)
		// Errors can happen here because the span batch prefix checks are not exhaustive (unlike the full span batch
		// checks) so an error must be handled like an invalid span batch (DROP).
		if err != nil {
			spanBatch.LogContext(bs.Log()).Warn("Dropping invalid span batch, flushing channel (singular batch extraction)", "error", err)
			bs.FlushChannel()
			return nil, NotEnoughData
		}
		bs.nextSpan = singularBatches
		// span-batches are non-empty, so the below pop is safe.
		return bs.popNextBatch(parent), nil
	default:
		return nil, NewCriticalError(fmt.Errorf("unrecognized batch type: %d", typ))
	}
}
