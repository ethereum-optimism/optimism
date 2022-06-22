package derive

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var DifferentEpoch = errors.New("batch is of different epoch")

func FilterBatches(log log.Logger, config *rollup.Config, epoch eth.BlockID, minL2Time uint64, maxL2Time uint64, batches []*BatchWithL1InclusionBlock) (out []*BatchWithL1InclusionBlock) {
	uniqueTime := make(map[uint64]struct{})
	for _, batch := range batches {
		if err := ValidBatch(batch.Batch, config, epoch, minL2Time, maxL2Time); err != nil {
			if err == DifferentEpoch {
				log.Trace("ignoring batch of different epoch", "expected_epoch", epoch,
					"epoch", batch.Epoch(), "timestamp", batch.Batch.Timestamp, "txs", len(batch.Batch.Transactions))
			} else {
				log.Warn("filtered batch", "expected_epoch", epoch, "min", minL2Time, "max", maxL2Time,
					"epoch", batch.Epoch(), "timestamp", batch.Batch.Timestamp, "txs", len(batch.Batch.Transactions), "err", err)
			}
			continue
		}
		// Check if we have already seen a batch for this L2 block
		if _, ok := uniqueTime[batch.Batch.Timestamp]; ok {
			log.Warn("duplicate batch", "epoch", batch.Epoch(), "timestamp", batch.Batch.Timestamp, "txs", len(batch.Batch.Transactions))
			// block already exists, batch is duplicate (first batch persists, others are ignored)
			continue
		}
		uniqueTime[batch.Batch.Timestamp] = struct{}{}
		out = append(out, batch)
	}
	return
}

func ValidBatch(batch *BatchData, config *rollup.Config, epoch eth.BlockID, minL2Time uint64, maxL2Time uint64) error {
	if batch.EpochNum != rollup.Epoch(epoch.Number) {
		// Batch was tagged for past or future epoch,
		// i.e. it was included too late or depends on the given L1 block to be processed first.
		// This is a very common error, batches may just be buffered for a later epoch.
		return DifferentEpoch
	}
	if batch.EpochHash != epoch.Hash {
		return fmt.Errorf("batch was meant for alternative L1 chain")
	}
	if (batch.Timestamp-config.Genesis.L2Time)%config.BlockTime != 0 {
		return fmt.Errorf("bad timestamp %d, not a multiple of the block time", batch.Timestamp)
	}
	if batch.Timestamp < minL2Time {
		return fmt.Errorf("old batch: %d < %d", batch.Timestamp, minL2Time)
	}
	// limit timestamp upper bound to avoid huge amount of empty blocks
	if batch.Timestamp >= maxL2Time {
		return fmt.Errorf("batch too far into future: %d > %d", batch.Timestamp, maxL2Time)
	}
	for i, txBytes := range batch.Transactions {
		if len(txBytes) == 0 {
			return fmt.Errorf("transaction data must not be empty, but tx %d is empty", i)
		}
		if txBytes[0] == types.DepositTxType {
			return fmt.Errorf("sequencers may not embed any deposits into batch data, but tx %d has one", i)
		}
	}
	return nil
}

// FillMissingBatches turns a collection of batches to the input batches for a series of blocks
func FillMissingBatches(batches []*BatchWithL1InclusionBlock, inclusionBlock eth.L1BlockRef, epoch eth.BlockID, blockTime, minL2Time, nextL1Time uint64) []*BatchWithL1InclusionBlock {
	m := make(map[uint64]*BatchWithL1InclusionBlock)
	// The number of L2 blocks per sequencing window is variable, we do not immediately fill to maxL2Time:
	// - ensure at least 1 block
	// - fill up to the next L1 block timestamp, if higher, to keep up with L1 time
	// - fill up to the last valid batch, to keep up with L2 time
	newHeadL2Timestamp := minL2Time
	if nextL1Time > newHeadL2Timestamp+1 {
		newHeadL2Timestamp = nextL1Time - 1
	}
	for _, b := range batches {
		m[b.Batch.Timestamp] = b
		if b.Batch.Timestamp > newHeadL2Timestamp {
			newHeadL2Timestamp = b.Batch.Timestamp
		}
	}
	var out []*BatchWithL1InclusionBlock
	for t := minL2Time; t <= newHeadL2Timestamp; t += blockTime {
		b, ok := m[t]
		if ok {
			out = append(out, b)
		} else {
			out = append(out, &BatchWithL1InclusionBlock{
				Batch: &BatchData{
					BatchV1{
						EpochNum:  rollup.Epoch(epoch.Number),
						EpochHash: epoch.Hash,
						Timestamp: t,
					},
				},
				L1InclusionBlock: inclusionBlock, // TODO. Not really relevant. Maybe end of sequencing window...
			})
		}
	}
	return out
}
