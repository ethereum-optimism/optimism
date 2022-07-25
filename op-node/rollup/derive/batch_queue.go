package derive

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type BatchQueueOutput interface {
	StageProgress
	AddBatch(batch *BatchData)
	SafeL2Head() eth.L2BlockRef
}

type BatchWithL1InclusionBlock struct {
	L1InclusionBlock eth.L1BlockRef
	Batch            *BatchData
}

func (b BatchWithL1InclusionBlock) Epoch() eth.BlockID {
	return b.Batch.Epoch()
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log      log.Logger
	config   *rollup.Config
	next     BatchQueueOutput
	progress Progress
	dl       L1BlockRefByNumberFetcher

	l1Blocks []eth.L1BlockRef

	// All batches with the same L2 block number. Batches are ordered by when they are seen.
	// Do a linear scan over the batches rather than deeply nested maps.
	// Note: Only a single batch with the same tuple (block number, timestamp, epoch) is allowed.
	batchesByTimestamp map[uint64][]*BatchWithL1InclusionBlock
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, dl L1BlockRefByNumberFetcher, next BatchQueueOutput) *BatchQueue {
	return &BatchQueue{
		log:                log,
		config:             cfg,
		next:               next,
		dl:                 dl,
		batchesByTimestamp: make(map[uint64][]*BatchWithL1InclusionBlock),
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
	batches, err := bq.deriveBatches(ctx, bq.next.SafeL2Head())
	if err == io.EOF {
		bq.log.Trace("Out of batches")
		return io.EOF
	} else if err != nil {
		bq.log.Error("Error deriving batches", "err", err)
		// Suppress transient errors for when reporting back to the pipeline
		return nil
	}

	for _, batch := range batches {
		if uint64(batch.Timestamp) <= bq.next.SafeL2Head().Time {
			// drop attributes if we are still progressing towards the next stage
			// (after a reset rolled us back a full sequence window)
			continue
		}
		bq.next.AddBatch(batch)
	}
	return nil
}

func (bq *BatchQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	// Copy over the Origin the from the next stage
	// It is set in the engine queue (two stages away) such that the L2 Safe Head origin is the progress
	bq.progress = bq.next.Progress()
	bq.batchesByTimestamp = make(map[uint64][]*BatchWithL1InclusionBlock)
	// Include the new origin as an origin to build off of.
	bq.l1Blocks = bq.l1Blocks[:0]
	bq.l1Blocks = append(bq.l1Blocks, bq.progress.Origin)

	return io.EOF
}

func (bq *BatchQueue) AddBatch(batch *BatchData) error {
	if bq.progress.Closed {
		panic("write batch while closed")
	}
	bq.log.Trace("queued batch", "origin", bq.progress.Origin, "tx_count", len(batch.Transactions), "timestamp", batch.Timestamp)
	if len(bq.l1Blocks) == 0 {
		return fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp)
	}

	data := BatchWithL1InclusionBlock{
		L1InclusionBlock: bq.progress.Origin,
		Batch:            batch,
	}
	batches, ok := bq.batchesByTimestamp[batch.Timestamp]
	// Filter complete duplicates. This step is not strictly needed as we always append, but it is nice to avoid lots of spam.
	if ok {
		for _, b := range batches {
			if b.Batch.Timestamp == batch.Timestamp && b.Batch.Epoch() == batch.Epoch() {
				bq.log.Warn("duplicate batch", "epoch", batch.Epoch(), "timestamp", batch.Timestamp, "txs", len(batch.Transactions))
				return nil
			}
		}
	} else {
		bq.log.Debug("First seen batch", "epoch", batch.Epoch(), "timestamp", batch.Timestamp, "txs", len(batch.Transactions))

	}
	// May have duplicate block numbers or individual fields, but have limited complete duplicates
	bq.batchesByTimestamp[batch.Timestamp] = append(batches, &data)
	return nil
}

// validExtension determines if a batch follows the previous attributes
func (bq *BatchQueue) validExtension(batch *BatchWithL1InclusionBlock, prevTime, prevEpoch uint64) bool {
	if batch.Batch.Timestamp != prevTime+bq.config.BlockTime {
		bq.log.Debug("Batch does not extend the block time properly", "time", batch.Batch.Timestamp, "prev_time", prevTime)

		return false
	}
	if batch.Batch.EpochNum != rollup.Epoch(prevEpoch) && batch.Batch.EpochNum != rollup.Epoch(prevEpoch+1) {
		bq.log.Debug("Batch does not extend the epoch properly", "epoch", batch.Batch.EpochNum, "prev_epoch", prevEpoch)

		return false
	}
	// TODO: Also check EpochHash (hard b/c maybe extension)

	// Note: `Batch.EpochNum` is an external input, but it is constrained to be a reasonable size by the
	// above equality checks.
	if uint64(batch.Batch.EpochNum)+bq.config.SeqWindowSize < batch.L1InclusionBlock.Number {
		bq.log.Debug("Batch submitted outside sequence window", "epoch", batch.Batch.EpochNum, "inclusion_block", batch.L1InclusionBlock.Number)
		return false
	}
	return true
}

// deriveBatches pulls a single batch eagerly or a collection of batches if it is the end of
// the sequencing window.
func (bq *BatchQueue) deriveBatches(ctx context.Context, l2SafeHead eth.L2BlockRef) ([]*BatchData, error) {
	if len(bq.l1Blocks) == 0 {
		return nil, io.EOF
	}
	epoch := bq.l1Blocks[0]

	// Decide if need to fill out empty batches & process an epoch at once
	// If not, just return a single batch
	// Note: can't process a full epoch until we are closed
	if bq.progress.Origin.Number >= epoch.Number+bq.config.SeqWindowSize && bq.progress.Closed {
		bq.log.Info("Advancing full epoch", "origin", epoch, "tip", bq.progress.Origin)
		// 2a. Gather all batches. First sort by timestamp and then by first seen.
		var bns []uint64
		for n := range bq.batchesByTimestamp {
			bns = append(bns, n)
		}
		sort.Slice(bns, func(i, j int) bool { return bns[i] < bns[j] })

		var batches []*BatchData
		for _, n := range bns {
			for _, batch := range bq.batchesByTimestamp[n] {
				// Filter out batches that were submitted too late.
				if uint64(batch.Batch.EpochNum)+bq.config.SeqWindowSize < batch.L1InclusionBlock.Number {
					continue
				}
				// Pre filter batches in the correct epoch
				if batch.Batch.EpochNum == rollup.Epoch(epoch.Number) {
					batches = append(batches, batch.Batch)
				}
			}
		}

		// 2b. Determine the valid time window
		l1OriginTime := bq.l1Blocks[0].Time
		nextL1BlockTime := bq.l1Blocks[1].Time // Safe b/c the epoch is the L1 Block number of the first block in L1Blocks
		minL2Time := l2SafeHead.Time + bq.config.BlockTime
		maxL2Time := l1OriginTime + bq.config.MaxSequencerDrift
		newEpoch := l2SafeHead.L1Origin != epoch.ID() // Only guarantee a L2 block if we have not already produced one for this epoch.
		if newEpoch && minL2Time+bq.config.BlockTime > maxL2Time {
			maxL2Time = minL2Time + bq.config.BlockTime
		}

		bq.log.Trace("found batches", "len", len(batches))
		// Filter + Fill batches
		batches = FilterBatches(bq.log, bq.config, epoch.ID(), minL2Time, maxL2Time, batches)
		bq.log.Trace("filtered batches", "len", len(batches), "l1Origin", bq.l1Blocks[0], "nextL1Block", bq.l1Blocks[1])
		batches = FillMissingBatches(batches, epoch.ID(), bq.config.BlockTime, minL2Time, nextL1BlockTime)
		bq.log.Trace("added missing batches", "len", len(batches), "l1OriginTime", l1OriginTime, "nextL1BlockTime", nextL1BlockTime)
		// Advance an epoch after filling all batches.
		bq.l1Blocks = bq.l1Blocks[1:]

		return batches, nil

	} else {
		bq.log.Trace("Trying to eagerly find batch")
		var ret []*BatchData
		next, err := bq.tryPopNextBatch(ctx, l2SafeHead)
		if next != nil {
			bq.log.Info("found eager batch", "batch", next.Batch)
			ret = append(ret, next.Batch)
		}
		return ret, err
	}

}

// tryPopNextBatch tries to get the next batch from the batch queue using an eager approach.
// It returns nil upon success, io.EOF if it does not have enough data, and a non-nil error
// upon a temporary processing error.
func (bq *BatchQueue) tryPopNextBatch(ctx context.Context, l2SafeHead eth.L2BlockRef) (*BatchWithL1InclusionBlock, error) {
	// We require at least 1 L1 blocks to look at.
	if len(bq.l1Blocks) == 0 {
		return nil, io.EOF
	}
	batches, ok := bq.batchesByTimestamp[l2SafeHead.Time+bq.config.BlockTime]
	// No more batches found.
	if !ok {
		return nil, io.EOF
	}

	// Find the first batch saved for this timestamp.
	// Note that we expect the number of batches for the same timestamp to be small (frequently just 1 ).
	for _, batch := range batches {
		l1OriginTime := bq.l1Blocks[0].Time

		// If this batch advances the epoch, check it's validity against the next L1 Origin
		if batch.Batch.EpochNum != rollup.Epoch(l2SafeHead.L1Origin.Number) {
			// With only 1 l1Block we cannot look at the next L1 Origin.
			// Note: This means that we are unable to determine validity of a batch
			// without more information. In this case we should bail out until we have
			// more information otherwise the eager algorithm may diverge from a non-eager
			// algorithm.
			if len(bq.l1Blocks) < 2 {
				bq.log.Warn("eager batch wants to advance epoch, but could not")
				return nil, io.EOF
			}
			l1OriginTime = bq.l1Blocks[1].Time
		}

		// Timestamp bounds
		minL2Time := l2SafeHead.Time + bq.config.BlockTime
		maxL2Time := l1OriginTime + bq.config.MaxSequencerDrift
		newEpoch := l2SafeHead.L1Origin != batch.Epoch() // Only guarantee a L2 block if we have not already produced one for this epoch.
		if newEpoch && minL2Time+bq.config.BlockTime > maxL2Time {
			maxL2Time = minL2Time + bq.config.BlockTime
		}

		// Note: Don't check epoch change here, check it in `validExtension`
		epoch, err := bq.dl.L1BlockRefByNumber(ctx, uint64(batch.Batch.EpochNum))
		if err != nil {
			bq.log.Warn("error fetching origin", "err", err)
			return nil, err
		}
		if err := ValidBatch(batch.Batch, bq.config, epoch.ID(), minL2Time, maxL2Time); err != nil {
			bq.log.Warn("Invalid batch", "err", err)
			break
		}

		// We have a valid batch, no make sure that it builds off the previous L2 block
		if bq.validExtension(batch, l2SafeHead.Time, l2SafeHead.L1Origin.Number) {
			// Advance the epoch if needed
			if l2SafeHead.L1Origin.Number != uint64(batch.Batch.EpochNum) {
				bq.l1Blocks = bq.l1Blocks[1:]
			}
			// Don't leak data in the map
			delete(bq.batchesByTimestamp, batch.Batch.Timestamp)

			bq.log.Info("Batch was valid extension")

			// We have found the fist valid batch.
			return batch, nil
		} else {
			bq.log.Info("batch was not valid extension")
		}
	}

	return nil, io.EOF
}
