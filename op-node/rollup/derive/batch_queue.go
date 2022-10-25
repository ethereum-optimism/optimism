package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
	NextBatch(ctx context.Context) (*BatchData, error)
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log    log.Logger
	config *rollup.Config
	prev   NextBatchProvider
	origin eth.L1BlockRef

	l1Blocks []eth.L1BlockRef

	// batches in order of when we've first seen them, grouped by L2 timestamp
	batches map[uint64][]*BatchWithL1InclusionBlock
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, prev NextBatchProvider) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		prev:   prev,
	}
}

func (bq *BatchQueue) Origin() eth.L1BlockRef {
	return bq.prev.Origin()
}

func (bq *BatchQueue) NextBatch(ctx context.Context, safeL2Head eth.L2BlockRef) (*BatchData, error) {
	// Note: We use the origin that we will have to determine if it's behind. This is important
	// because it's the future origin that gets saved into the l1Blocks array.
	// We always update the origin of this stage if it is not the same so after the update code
	// runs, this is consistent.
	originBehind := bq.prev.Origin().Number < safeL2Head.L1Origin.Number

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
		bq.log.Info("Advancing bq origin", "origin", bq.origin)
	}

	// Load more data into the batch queue
	outOfData := false
	if batch, err := bq.prev.NextBatch(ctx); err == io.EOF {
		outOfData = true
	} else if err != nil {
		return nil, err
	} else if !originBehind {
		bq.AddBatch(batch, safeL2Head)
	}

	// Skip adding data unless we are up to date with the origin, but do fully
	// empty the previous stages
	if originBehind {
		if outOfData {
			return nil, io.EOF
		} else {
			return nil, NotEnoughData
		}
	}

	// Finally attempt to derive more batches
	batch, err := bq.deriveNextBatch(ctx, outOfData, safeL2Head)
	if err == io.EOF && outOfData {
		return nil, io.EOF
	} else if err == io.EOF {
		return nil, NotEnoughData
	} else if err != nil {
		return nil, err
	}
	return batch, nil
}

func (bq *BatchQueue) Reset(ctx context.Context, base eth.L1BlockRef, _ eth.SystemConfig) error {
	// Copy over the Origin from the next stage
	// It is set in the engine queue (two stages away) such that the L2 Safe Head origin is the progress
	bq.origin = base
	bq.batches = make(map[uint64][]*BatchWithL1InclusionBlock)
	// Include the new origin as an origin to build on
	// Note: This is only for the initialization case. During normal resets we will later
	// throw out this block.
	bq.l1Blocks = bq.l1Blocks[:0]
	bq.l1Blocks = append(bq.l1Blocks, base)
	return io.EOF
}

func (bq *BatchQueue) AddBatch(batch *BatchData, l2SafeHead eth.L2BlockRef) {
	if len(bq.l1Blocks) == 0 {
		panic(fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp))
	}
	data := BatchWithL1InclusionBlock{
		L1InclusionBlock: bq.origin,
		Batch:            batch,
	}
	validity := CheckBatch(bq.config, bq.log, bq.l1Blocks, l2SafeHead, &data)
	if validity == BatchDrop {
		return // if we do drop the batch, CheckBatch will log the drop reason with WARN level.
	}
	bq.batches[batch.Timestamp] = append(bq.batches[batch.Timestamp], &data)
}

// deriveNextBatch derives the next batch to apply on top of the current L2 safe head,
// following the validity rules imposed on consecutive batches,
// based on currently available buffered batch and L1 origin information.
// If no batch can be derived yet, then (nil, io.EOF) is returned.
func (bq *BatchQueue) deriveNextBatch(ctx context.Context, outOfData bool, l2SafeHead eth.L2BlockRef) (*BatchData, error) {
	if len(bq.l1Blocks) == 0 {
		return nil, NewCriticalError(errors.New("cannot derive next batch, no origin was prepared"))
	}
	epoch := bq.l1Blocks[0]

	if l2SafeHead.L1Origin != epoch.ID() {
		return nil, NewResetError(fmt.Errorf("buffered L1 chain epoch %s in batch queue does not match safe head origin %s", epoch, l2SafeHead.L1Origin))
	}

	// Find the first-seen batch that matches all validity conditions.
	// We may not have sufficient information to proceed filtering, and then we stop.
	// There may be none: in that case we force-create an empty batch
	nextTimestamp := l2SafeHead.Time + bq.config.BlockTime
	var nextBatch *BatchWithL1InclusionBlock

	// Go over all batches, in order of inclusion, and find the first batch we can accept.
	// We filter in-place by only remembering the batches that may be processed in the future, or those we are undecided on.
	var remaining []*BatchWithL1InclusionBlock
	candidates := bq.batches[nextTimestamp]
batchLoop:
	for i, batch := range candidates {
		validity := CheckBatch(bq.config, bq.log.New("batch_index", i), bq.l1Blocks, l2SafeHead, batch)
		switch validity {
		case BatchFuture:
			return nil, NewCriticalError(fmt.Errorf("found batch with timestamp %d marked as future batch, but expected timestamp %d", batch.Batch.Timestamp, nextTimestamp))
		case BatchDrop:
			bq.log.Warn("dropping batch",
				"batch_timestamp", batch.Batch.Timestamp,
				"parent_hash", batch.Batch.ParentHash,
				"batch_epoch", batch.Batch.Epoch(),
				"txs", len(batch.Batch.Transactions),
				"l2_safe_head", l2SafeHead.ID(),
				"l2_safe_head_time", l2SafeHead.Time,
			)
			continue
		case BatchAccept:
			nextBatch = batch
			// don't keep the current batch in the remaining items since we are processing it now,
			// but retain every batch we didn't get to yet.
			remaining = append(remaining, candidates[i+1:]...)
			break batchLoop
		case BatchUndecided:
			remaining = append(remaining, batch)
			bq.batches[nextTimestamp] = remaining
			return nil, io.EOF
		default:
			return nil, NewCriticalError(fmt.Errorf("unknown batch validity type: %d", validity))
		}
	}
	// clean up if we remove the final batch for this timestamp
	if len(remaining) == 0 {
		delete(bq.batches, nextTimestamp)
	} else {
		bq.batches[nextTimestamp] = remaining
	}

	if nextBatch != nil {
		// advance epoch if necessary
		if nextBatch.Batch.EpochNum == rollup.Epoch(epoch.Number)+1 {
			bq.l1Blocks = bq.l1Blocks[1:]
		}
		return nextBatch.Batch, nil
	}

	// If the current epoch is too old compared to the L1 block we are at,
	// i.e. if the sequence window expired, we create empty batches
	expiryEpoch := epoch.Number + bq.config.SeqWindowSize
	forceNextEpoch :=
		(expiryEpoch == bq.origin.Number && outOfData) ||
			expiryEpoch < bq.origin.Number

	if !forceNextEpoch {
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
	// to preserve that L2 time >= L1 time
	if nextTimestamp < nextEpoch.Time {
		return &BatchData{
			BatchV1{
				ParentHash:   l2SafeHead.Hash,
				EpochNum:     rollup.Epoch(epoch.Number),
				EpochHash:    epoch.Hash,
				Timestamp:    nextTimestamp,
				Transactions: nil,
			},
		}, nil
	}
	// As we move the safe head origin forward, we also drop the old L1 block reference
	bq.l1Blocks = bq.l1Blocks[1:]
	return &BatchData{
		BatchV1{
			ParentHash:   l2SafeHead.Hash,
			EpochNum:     rollup.Epoch(nextEpoch.Number),
			EpochHash:    nextEpoch.Hash,
			Timestamp:    nextTimestamp,
			Transactions: nil,
		},
	}, nil
}
