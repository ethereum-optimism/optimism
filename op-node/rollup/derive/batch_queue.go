package derive

import (
	"context"
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
	progress Progress

	l1Blocks []eth.L1BlockRef

	// batches in order of when we've first seen them
	batches []*BatchWithL1InclusionBlock
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, next BatchQueueOutput) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		next:   next,
	}
}

func (bq *BatchQueue) Progress() Progress {
	return bq.progress
}

func (bq *BatchQueue) Step(ctx context.Context, outer Progress) error {
	if changed, err := bq.progress.Update(outer); err != nil {
		return err
	} else if changed {
		if !bq.progress.Closed { // init inputs if we moved to a new open origin
			bq.l1Blocks = append(bq.l1Blocks, bq.progress.Origin)
		}
		return nil
	}
	batch, err := bq.deriveNextBatch(ctx)
	if err == io.EOF {
		// very noisy, commented for now, or we should bump log level from trace to debug
		// bq.log.Trace("need more L1 data before deriving next batch", "progress", bq.progress.Origin)
		return io.EOF
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
	bq.batches = bq.batches[:0]
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
	log := bq.log.New(
		"current_l1", bq.progress.Origin,
		"batch_timestamp", batch.Timestamp,
		"batch_epoch", batch.Epoch(),
		"txs", len(batch.Transactions),
	)
	log.Trace("queuing batch")

	data := BatchWithL1InclusionBlock{
		L1InclusionBlock: bq.progress.Origin,
		Batch:            batch,
	}
	validity := CheckBatch(bq.config, bq.log, bq.l1Blocks, bq.next.SafeL2Head(), &data)
	if validity == BatchDrop {
		log.Warn("ingested invalid batch from L1, dropping it instead of buffering it")
		return
	}
	bq.batches = append(bq.batches, &data)
}

// deriveNextBatch derives the next batch.
// If no batch can be derived yet, then (nil, io.EOF) is returned.
func (bq *BatchQueue) deriveNextBatch(ctx context.Context) (*BatchData, error) {
	if len(bq.l1Blocks) == 0 {
		panic("cannot derive next batch, no origin was prepared")
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
	remaining := bq.batches[:0]
batchLoop:
	for i, batch := range bq.batches {
		validity := CheckBatch(bq.config, bq.log.New("batch_index", i), bq.l1Blocks, l2SafeHead, batch)
		switch validity {
		case BatchFuture:
			remaining = append(remaining, batch)
			continue
		case BatchDrop:
			continue
		case BatchAccept:
			nextBatch = batch
			// remove the current batch since we are processing it now, and retain every batch we didn't get to yet.
			remaining = append(remaining, bq.batches[i+1:]...)
			break batchLoop
		case BatchUndecided:
			remaining = append(remaining, batch)
			bq.batches = remaining
			return nil, io.EOF
		default:
			panic(fmt.Errorf("unknown batch validity type: %d", validity))
		}
	}
	bq.batches = remaining

	if nextBatch == nil {
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
	} else {
		// advance epoch if necessary
		if nextBatch.Batch.EpochNum == rollup.Epoch(epoch.Number)+1 {
			bq.l1Blocks = bq.l1Blocks[1:]
		}
		// sanity check
		if nextBatch.Batch.EpochNum > rollup.Epoch(epoch.Number)+1 {
			return nil, NewCriticalError(fmt.Errorf("batch is advancing more than 1 epoch, from %s to %s", epoch, nextBatch.Batch.Epoch()))
		}
		return nextBatch.Batch, nil
	}
}
