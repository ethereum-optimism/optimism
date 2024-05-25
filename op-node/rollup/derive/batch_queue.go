package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The batch queue is responsible for ordering unordered batches & generating empty batches
// when the sequence window has passed. This is a very stateful stage.
//
// It receives batches that are tagged with the L1 Inclusion block of the batch. It only considers
// batches that are inside the sequencing window of a specific L1 Origin.
// It tries to eagerly pull batches based on the current L2 safe head.
// Otherwise it filters/creates an entire epoch's worth of batches at once.
//
// This stage tracks a range of L1 blocks with the assumption that all batches with an L1 inclusion
// block inside that range have been added to the stage by the time that it attempts to advance a
// full epoch.
//
// It is internally responsible for making sure that batches with L1 inclusions block outside it's
// working range are not considered or pruned.

type NextBatchProvider interface {
	Origin() eth.L1BlockRef
	NextBatch(ctx context.Context) (Batch, error)
}

type SafeBlockFetcher interface {
	L2BlockRefByNumber(context.Context, uint64) (eth.L2BlockRef, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayload, error)
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log    log.Logger
	config *rollup.Config
	prev   NextBatchProvider
	origin eth.L1BlockRef

	// l1Blocks contains consecutive eth.L1BlockRef sorted by time.
	// Every L1 origin of unsafe L2 blocks must be eventually included in l1Blocks.
	// Batch queue's job is to ensure below two rules:
	//  If every L2 block corresponding to single L1 block becomes safe, it will be popped from l1Blocks.
	//  If new L2 block's L1 origin is not included in l1Blocks, fetch and push to l1Blocks.
	// length of l1Blocks never exceeds SequencerWindowSize
	l1Blocks []eth.L1BlockRef

	// batches in order of when we've first seen them
	batches []*BatchWithL1InclusionBlock

	// nextSpan is cached SingularBatches derived from SpanBatch
	nextSpan []*SingularBatch

	l2 SafeBlockFetcher
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, prev NextBatchProvider, l2 SafeBlockFetcher) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		prev:   prev,
		l2:     l2,
	}
}

func (bq *BatchQueue) Origin() eth.L1BlockRef {
	return bq.prev.Origin()
}

// popNextBatch pops the next batch from the current queued up span-batch nextSpan.
// The queue must be non-empty, or the function will panic.
func (bq *BatchQueue) popNextBatch(parent eth.L2BlockRef) *SingularBatch {
	if len(bq.nextSpan) == 0 {
		panic("popping non-existent span-batch, invalid state")
	}
	nextBatch := bq.nextSpan[0]
	bq.nextSpan = bq.nextSpan[1:]
	// Must set ParentHash before return. we can use parent because the parentCheck is verified in CheckBatch().
	nextBatch.ParentHash = parent.Hash
	bq.log.Debug("pop next batch from the cached span batch")
	return nextBatch
}

// NextBatch return next valid batch upon the given safe head.
// It also returns the boolean that indicates if the batch is the last block in the batch.
func (bq *BatchQueue) NextBatch(ctx context.Context, parent eth.L2BlockRef) (*SingularBatch, bool, error) {
	if len(bq.nextSpan) > 0 {
		// There are cached singular batches derived from the span batch.
		// Check if the next cached batch matches the given parent block.
		if bq.nextSpan[0].Timestamp == parent.Time+bq.config.BlockTime {
			// Pop first one and return.
			nextBatch := bq.popNextBatch(parent)
			// len(bq.nextSpan) == 0 means it's the last batch of the span.
			return nextBatch, len(bq.nextSpan) == 0, nil
		} else {
			// Given parent block does not match the next batch. It means the previously returned batch is invalid.
			// Drop cached batches and find another batch.
			bq.log.Warn("parent block does not match the next batch. dropped cached batches", "parent", parent.ID(), "nextBatchTime", bq.nextSpan[0].GetTimestamp())
			bq.nextSpan = bq.nextSpan[:0]
		}
	}

	// If the epoch is advanced, update bq.l1Blocks
	// Advancing epoch must be done after the pipeline successfully apply the entire span batch to the chain.
	// Because the span batch can be reverted during processing the batch, then we must preserve existing l1Blocks
	// to verify the epochs of the next candidate batch.
	if len(bq.l1Blocks) > 0 && parent.L1Origin.Number > bq.l1Blocks[0].Number {
		for i, l1Block := range bq.l1Blocks {
			if parent.L1Origin.Number == l1Block.Number {
				bq.l1Blocks = bq.l1Blocks[i:]
				if len(bq.l1Blocks) > 0 {
					bq.log.Debug("Advancing internal L1 blocks", "next_epoch", bq.l1Blocks[0].ID(), "next_epoch_time", bq.l1Blocks[0].Time)
				} else {
					bq.log.Debug("Advancing internal L1 blocks. No L1 blocks left")
				}
				break
			}
		}
		// If we can't find the origin of parent block, we have to advance bq.origin.
	}

	// Note: We use the origin that we will have to determine if it's behind. This is important
	// because it's the future origin that gets saved into the l1Blocks array.
	// We always update the origin of this stage if it is not the same so after the update code
	// runs, this is consistent.
	originBehind := bq.prev.Origin().Number < parent.L1Origin.Number

	// Advance origin if needed
	// Note: The entire pipeline has the same origin
	// We just don't accept batches prior to the L1 origin of the L2 safe head
	if bq.origin != bq.prev.Origin() {
		bq.origin = bq.prev.Origin()
		if !originBehind {
			bq.l1Blocks = append(bq.l1Blocks, bq.origin)
		} else {
			// This is to handle the special case of startup. At startup we call Reset & include
			// the L1 origin. That is the only time where immediately after `Reset` is called
			// originBehind is false.
			bq.l1Blocks = bq.l1Blocks[:0]
		}
		bq.log.Info("Advancing bq origin", "origin", bq.origin, "originBehind", originBehind)
	}

	// Load more data into the batch queue
	outOfData := false
	if batch, err := bq.prev.NextBatch(ctx); err == io.EOF {
		outOfData = true
	} else if err != nil {
		return nil, false, err
	} else if !originBehind {
		bq.AddBatch(ctx, batch, parent)
	}

	// Skip adding data unless we are up to date with the origin, but do fully
	// empty the previous stages
	if originBehind {
		if outOfData {
			return nil, false, io.EOF
		} else {
			return nil, false, NotEnoughData
		}
	}

	// Finally attempt to derive more batches
	batch, err := bq.deriveNextBatch(ctx, outOfData, parent)
	if err == io.EOF && outOfData {
		return nil, false, io.EOF
	} else if err == io.EOF {
		return nil, false, NotEnoughData
	} else if err != nil {
		return nil, false, err
	}

	var nextBatch *SingularBatch
	switch batch.GetBatchType() {
	case SingularBatchType:
		singularBatch, ok := batch.(*SingularBatch)
		if !ok {
			return nil, false, NewCriticalError(errors.New("failed type assertion to SingularBatch"))
		}
		nextBatch = singularBatch
	case SpanBatchType:
		spanBatch, ok := batch.(*SpanBatch)
		if !ok {
			return nil, false, NewCriticalError(errors.New("failed type assertion to SpanBatch"))
		}
		// If next batch is SpanBatch, convert it to SingularBatches.
		singularBatches, err := spanBatch.GetSingularBatches(bq.l1Blocks, parent)
		if err != nil {
			return nil, false, NewCriticalError(err)
		}
		bq.nextSpan = singularBatches
		// span-batches are non-empty, so the below pop is safe.
		nextBatch = bq.popNextBatch(parent)
	default:
		return nil, false, NewCriticalError(fmt.Errorf("unrecognized batch type: %d", batch.GetBatchType()))
	}

	// If the nextBatch is derived from the span batch, len(bq.nextSpan) == 0 means it's the last batch of the span.
	// For singular batches, len(bq.nextSpan) == 0 is always true.
	return nextBatch, len(bq.nextSpan) == 0, nil
}

func (bq *BatchQueue) Reset(ctx context.Context, base eth.L1BlockRef, _ eth.SystemConfig) error {
	// Copy over the Origin from the next stage
	// It is set in the engine queue (two stages away) such that the L2 Safe Head origin is the progress
	bq.origin = base
	bq.batches = []*BatchWithL1InclusionBlock{}
	// Include the new origin as an origin to build on
	// Note: This is only for the initialization case. During normal resets we will later
	// throw out this block.
	bq.l1Blocks = bq.l1Blocks[:0]
	bq.l1Blocks = append(bq.l1Blocks, base)
	bq.nextSpan = bq.nextSpan[:0]
	return io.EOF
}

func (bq *BatchQueue) AddBatch(ctx context.Context, batch Batch, parent eth.L2BlockRef) {
	if len(bq.l1Blocks) == 0 {
		panic(fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.GetTimestamp()))
	}
	data := BatchWithL1InclusionBlock{
		L1InclusionBlock: bq.origin,
		Batch:            batch,
	}
	validity := CheckBatch(ctx, bq.config, bq.log, bq.l1Blocks, parent, &data, bq.l2)
	if validity == BatchDrop {
		return // if we do drop the batch, CheckBatch will log the drop reason with WARN level.
	}
	batch.LogContext(bq.log).Debug("Adding batch")
	bq.batches = append(bq.batches, &data)
}

// deriveNextBatch derives the next batch to apply on top of the current L2 safe head,
// following the validity rules imposed on consecutive batches,
// based on currently available buffered batch and L1 origin information.
// If no batch can be derived yet, then (nil, io.EOF) is returned.
func (bq *BatchQueue) deriveNextBatch(ctx context.Context, outOfData bool, parent eth.L2BlockRef) (Batch, error) {
	if len(bq.l1Blocks) == 0 {
		return nil, NewCriticalError(errors.New("cannot derive next batch, no origin was prepared"))
	}
	epoch := bq.l1Blocks[0]
	bq.log.Trace("Deriving the next batch", "epoch", epoch, "parent", parent, "outOfData", outOfData)

	// Note: epoch origin can now be one block ahead of the L2 Safe Head
	// This is in the case where we auto generate all batches in an epoch & advance the epoch
	// but don't advance the L2 Safe Head's epoch
	if parent.L1Origin != epoch.ID() && parent.L1Origin.Number != epoch.Number-1 {
		return nil, NewResetError(fmt.Errorf("buffered L1 chain epoch %s in batch queue does not match safe head origin %s", epoch, parent.L1Origin))
	}

	// Find the first-seen batch that matches all validity conditions.
	// We may not have sufficient information to proceed filtering, and then we stop.
	// There may be none: in that case we force-create an empty batch
	nextTimestamp := parent.Time + bq.config.BlockTime
	var nextBatch *BatchWithL1InclusionBlock

	// Go over all batches, in order of inclusion, and find the first batch we can accept.
	// We filter in-place by only remembering the batches that may be processed in the future, or those we are undecided on.
	var remaining []*BatchWithL1InclusionBlock
batchLoop:
	for i, batch := range bq.batches {
		validity := CheckBatch(ctx, bq.config, bq.log.New("batch_index", i), bq.l1Blocks, parent, batch, bq.l2)
		switch validity {
		case BatchFuture:
			remaining = append(remaining, batch)
			continue
		case BatchDrop:
			batch.Batch.LogContext(bq.log).Warn("Dropping batch",
				"parent", parent.ID(),
				"parent_time", parent.Time,
			)
			continue
		case BatchAccept:
			nextBatch = batch
			// don't keep the current batch in the remaining items since we are processing it now,
			// but retain every batch we didn't get to yet.
			remaining = append(remaining, bq.batches[i+1:]...)
			break batchLoop
		case BatchUndecided:
			remaining = append(remaining, bq.batches[i:]...)
			bq.batches = remaining
			return nil, io.EOF
		default:
			return nil, NewCriticalError(fmt.Errorf("unknown batch validity type: %d", validity))
		}
	}
	bq.batches = remaining

	if nextBatch != nil {
		nextBatch.Batch.LogContext(bq.log).Info("Found next batch")
		return nextBatch.Batch, nil
	}

	// If the current epoch is too old compared to the L1 block we are at,
	// i.e. if the sequence window expired, we create empty batches for the current epoch
	expiryEpoch := epoch.Number + bq.config.SeqWindowSize
	forceEmptyBatches := (expiryEpoch == bq.origin.Number && outOfData) || expiryEpoch < bq.origin.Number
	firstOfEpoch := epoch.Number == parent.L1Origin.Number+1

	bq.log.Trace("Potentially generating an empty batch",
		"expiryEpoch", expiryEpoch, "forceEmptyBatches", forceEmptyBatches, "nextTimestamp", nextTimestamp,
		"epoch_time", epoch.Time, "len_l1_blocks", len(bq.l1Blocks), "firstOfEpoch", firstOfEpoch)

	if !forceEmptyBatches {
		// sequence window did not expire yet, still room to receive batches for the current epoch,
		// no need to force-create empty batch(es) towards the next epoch yet.
		return nil, io.EOF
	}
	if len(bq.l1Blocks) < 2 {
		// need next L1 block to proceed towards
		return nil, io.EOF
	}

	nextEpoch := bq.l1Blocks[1]
	// Fill with empty L2 blocks of the same epoch until we meet the time of the next L1 origin,
	// to preserve that L2 time >= L1 time. If this is the first block of the epoch, always generate a
	// batch to ensure that we at least have one batch per epoch.
	if nextTimestamp < nextEpoch.Time || firstOfEpoch {
		bq.log.Info("Generating next batch", "epoch", epoch, "timestamp", nextTimestamp)
		return &SingularBatch{
			ParentHash:   parent.Hash,
			EpochNum:     rollup.Epoch(epoch.Number),
			EpochHash:    epoch.Hash,
			Timestamp:    nextTimestamp,
			Transactions: nil,
		}, nil
	}

	// At this point we have auto generated every batch for the current epoch
	// that we can, so we can advance to the next epoch.
	bq.log.Trace("Advancing internal L1 blocks", "next_timestamp", nextTimestamp, "next_epoch_time", nextEpoch.Time)
	bq.l1Blocks = bq.l1Blocks[1:]
	return nil, io.EOF
}
