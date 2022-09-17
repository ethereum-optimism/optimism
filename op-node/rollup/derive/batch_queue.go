package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
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

type BatchQueueOutput interface {
	StageProgress
	AddBatch(batch *BatchData)
	SafeL2Head() eth.L2BlockRef
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log      log.Logger
	config   *rollup.Config
	next     BatchQueueOutput
	prev     *ChannelInReader
	progress Progress

	l1Blocks []eth.L1BlockRef

	// batches in order of when we've first seen them, grouped by L2 timestamp
	batches map[uint64][]*BatchWithL1InclusionBlock
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, next BatchQueueOutput, prev *ChannelInReader) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		next:   next,
		prev:   prev,
	}
}

func (bq *BatchQueue) Progress() Progress {
	return bq.progress
}

func (bq *BatchQueue) Step(ctx context.Context, outer Progress) error {

	originBehind := bq.progress.Origin.Number < bq.next.SafeL2Head().L1Origin.Number

	// Advance origin if needed
	// Note: The entire pipeline has the same origin
	// We just don't accept batches prior to the L1 origin of the L2 safe head
	if bq.progress.Origin != bq.prev.Origin() {
		bq.progress.Closed = false
		bq.progress.Origin = bq.prev.Origin()
		if !originBehind {
			bq.l1Blocks = append(bq.l1Blocks, bq.progress.Origin)
		}
		bq.log.Info("Advancing bq origin", "origin", bq.progress.Origin)
		return nil
	}
	if !bq.progress.Closed {
		if batch, err := bq.prev.NextBatch(ctx); err == io.EOF {
			bq.log.Info("Closing batch queue origin")
			bq.progress.Closed = true
			return nil
		} else if err != nil {
			return err
		} else {
			bq.log.Info("have batch")
			if !originBehind {
				bq.AddBatch(batch)
			} else {
				bq.log.Warn("Skipping old batch")
			}
		}
	}

	// Skip adding batches / blocks to the internal state until they are from the same L1 origin
	// as the current safe head.
	if originBehind {
		if bq.progress.Closed {
			return io.EOF
		} else {
			// Immediately close the stage
			bq.progress.Closed = true
			return nil
		}
	}

	batch, err := bq.deriveNextBatch(ctx)
	if err == io.EOF {
		bq.log.Info("no more batches in deriveNextBatch")
		if bq.progress.Closed {
			return io.EOF
		} else {
			return nil
		}
	} else if err != nil {
		return err
	}
	bq.next.AddBatch(batch)
	return nil
}

func (bq *BatchQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	// Copy over the Origin from the next stage
	// It is set in the engine queue (two stages away) such that the L2 Safe Head origin is the progress
	bq.progress = bq.next.Progress()
	bq.batches = make(map[uint64][]*BatchWithL1InclusionBlock)
	// Include the new origin as an origin to build on
	bq.l1Blocks = bq.l1Blocks[:0]
	bq.l1Blocks = append(bq.l1Blocks, bq.progress.Origin)
	return io.EOF
}

func (bq *BatchQueue) AddBatch(batch *BatchData) {
	if bq.progress.Closed {
		panic("write batch while closed")
	}
	if len(bq.l1Blocks) == 0 {
		panic(fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp))
	}
	data := BatchWithL1InclusionBlock{
		L1InclusionBlock: bq.progress.Origin,
		Batch:            batch,
	}
	validity := CheckBatch(bq.config, bq.log, bq.l1Blocks, bq.next.SafeL2Head(), &data)
	if validity == BatchDrop {
		return // if we do drop the batch, CheckBatch will log the drop reason with WARN level.
	}
	bq.batches[batch.Timestamp] = append(bq.batches[batch.Timestamp], &data)
}

// deriveNextBatch derives the next batch to apply on top of the current L2 safe head,
// following the validity rules imposed on consecutive batches,
// based on currently available buffered batch and L1 origin information.
// If no batch can be derived yet, then (nil, io.EOF) is returned.
func (bq *BatchQueue) deriveNextBatch(ctx context.Context) (*BatchData, error) {
	if len(bq.l1Blocks) == 0 {
		return nil, NewCriticalError(errors.New("cannot derive next batch, no origin was prepared"))
	}
	epoch := bq.l1Blocks[0]
	l2SafeHead := bq.next.SafeL2Head()

	if l2SafeHead.L1Origin != epoch.ID() {
		return nil, NewResetError(fmt.Errorf("buffered L1 chain epoch %s in batch queue does not match safe head %s", epoch, l2SafeHead))
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
		(expiryEpoch == bq.progress.Origin.Number && bq.progress.Closed) ||
			expiryEpoch < bq.progress.Origin.Number

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
