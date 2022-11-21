package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type BatchWithL1InclusionBlock struct {
	L1InclusionBlock eth.L1BlockRef
	Batch            *BatchData
}

type BatchValidity uint8

const (
	// BatchDrop indicates that the batch is invalid, and will always be in the future, unless we reorg
	BatchDrop = iota
	// BatchAccept indicates that the batch is valid and should be processed
	BatchAccept
	// BatchUndecided indicates we are lacking L1 information until we can proceed batch filtering
	BatchUndecided
	// BatchFuture indicates that the batch may be valid, but cannot be processed yet and should be checked again later
	BatchFuture
)

// CheckBatch checks if the given batch can be applied on top of the given l2SafeHead, given the contextual L1 blocks the batch was included in.
// The first entry of the l1Blocks should match the origin of the l2SafeHead. One or more consecutive l1Blocks should be provided.
// In case of only a single L1 block, the decision whether a batch is valid may have to stay undecided.
func CheckBatch(cfg *rollup.Config, log log.Logger, l1Blocks []eth.L1BlockRef, l2SafeHead eth.L2BlockRef, batch *BatchWithL1InclusionBlock) BatchValidity {
	// add details to the log
	log = log.New(
		"batch_timestamp", batch.Batch.Timestamp,
		"parent_hash", batch.Batch.ParentHash,
		"batch_epoch", batch.Batch.Epoch(),
		"txs", len(batch.Batch.Transactions),
	)

	// sanity check we have consistent inputs
	if len(l1Blocks) == 0 {
		log.Warn("missing L1 block input, cannot proceed with batch checking")
		return BatchUndecided
	}
	epoch := l1Blocks[0]
	if epoch.Hash != l2SafeHead.L1Origin.Hash {
		log.Warn("safe L2 head L1 origin does not match batch first l1 block (current epoch)",
			"safe_l2", l2SafeHead, "safe_origin", l2SafeHead.L1Origin, "epoch", epoch)
		return BatchUndecided
	}

	nextTimestamp := l2SafeHead.Time + cfg.BlockTime
	if batch.Batch.Timestamp > nextTimestamp {
		log.Trace("received out-of-order batch for future processing after next batch", "next_timestamp", nextTimestamp)
		return BatchFuture
	}
	if batch.Batch.Timestamp < nextTimestamp {
		log.Warn("dropping batch with old timestamp", "min_timestamp", nextTimestamp)
		return BatchDrop
	}

	// dependent on above timestamp check. If the timestamp is correct, then it must build on top of the safe head.
	if batch.Batch.ParentHash != l2SafeHead.Hash {
		log.Warn("ignoring batch with mismatching parent hash", "current_safe_head", l2SafeHead.Hash)
		return BatchDrop
	}

	// Filter out batches that were included too late.
	if uint64(batch.Batch.EpochNum)+cfg.SeqWindowSize < batch.L1InclusionBlock.Number {
		log.Warn("batch was included too late, sequence window expired")
		return BatchDrop
	}

	// Check the L1 origin of the batch
	batchOrigin := epoch
	if uint64(batch.Batch.EpochNum) < epoch.Number {
		log.Warn("dropped batch, epoch is too old", "minimum", epoch.ID())
		// batch epoch too old
		return BatchDrop
	} else if uint64(batch.Batch.EpochNum) == epoch.Number {
		// Batch is sticking to the current epoch, continue.
	} else if uint64(batch.Batch.EpochNum) == epoch.Number+1 {
		// With only 1 l1Block we cannot look at the next L1 Origin.
		// Note: This means that we are unable to determine validity of a batch
		// without more information. In this case we should bail out until we have
		// more information otherwise the eager algorithm may diverge from a non-eager
		// algorithm.
		if len(l1Blocks) < 2 {
			log.Info("eager batch wants to advance epoch, but could not without more L1 blocks", "current_epoch", epoch.ID())
			return BatchUndecided
		}
		batchOrigin = l1Blocks[1]
	} else {
		log.Warn("batch is for future epoch too far ahead, while it has the next timestamp, so it must be invalid", "current_epoch", epoch.ID())
		return BatchDrop
	}

	if batch.Batch.EpochHash != batchOrigin.Hash {
		log.Warn("batch is for different L1 chain, epoch hash does not match", "expected", batchOrigin.ID())
		return BatchDrop
	}

	// If we ran out of sequencer time drift, then we drop the batch and produce an empty batch instead,
	// as the sequencer is not allowed to include anything past this point without moving to the next epoch.
	if max := batchOrigin.Time + cfg.MaxSequencerDrift; batch.Batch.Timestamp > max {
		log.Warn("batch exceeded sequencer time drift, sequencer must adopt new L1 origin to include transactions again", "max_time", max)
		return BatchDrop
	}

	// We can do this check earlier, but it's a more intensive one, so we do this last.
	for i, txBytes := range batch.Batch.Transactions {
		if len(txBytes) == 0 {
			log.Warn("transaction data must not be empty, but found empty tx", "tx_index", i)
			return BatchDrop
		}
		if txBytes[0] == types.DepositTxType {
			log.Warn("sequencers may not embed any deposits into batch data, but found tx that has one", "tx_index", i)
			return BatchDrop
		}
	}

	return BatchAccept
}
