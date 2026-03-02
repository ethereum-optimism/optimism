package pure

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// decodeBatches reads all batches from a completed channel's compressed data
// and returns them as singular batches. Span batches are expanded into
// individual singular batches using the provided L1 origins and cursor.
//
// Decode errors are logged and cause the function to return whatever batches
// were successfully decoded so far. Only programming errors (bugs) would
// warrant propagating errors upward; all data-dependent failures are treated
// as bad input.
//
// Span batch prefix validation is delegated to derive.CheckSpanBatchPrefix,
// which rejects overlapping span batches under Karst. If the prefix check
// returns BatchPast, the span batch is skipped. Any other non-Accept result
// causes the function to return the batches collected so far.
func decodeBatches(
	lgr log.Logger,
	r io.Reader,
	cfg *rollup.Config,
	l1Origins []eth.L1BlockRef,
	cursor l2Cursor,
	l1InclusionBlock eth.L1BlockRef,
) []*derive.SingularBatch {
	spec := rollup.NewChainSpec(cfg)
	maxRLP := spec.MaxRLPBytesPerChannel(cursor.Timestamp)

	readBatch, err := derive.BatchReader(r, maxRLP, true) // Fjord always active (implied by Karst)
	if err != nil {
		lgr.Warn("failed to create batch reader", "err", err)
		return nil
	}

	var batches []*derive.SingularBatch
	for {
		batchData, err := readBatch()
		if err == io.EOF {
			break
		} else if err != nil {
			lgr.Warn("failed to read batch", "err", err)
			return batches
		}

		switch batchData.GetBatchType() {
		case derive.SingularBatchType:
			singular, err := derive.GetSingularBatch(batchData)
			if err != nil {
				lgr.Warn("failed to extract singular batch", "err", err)
				return batches
			}
			batches = append(batches, singular)

		case derive.SpanBatchType:
			spanBatch, err := derive.DeriveSpanBatch(
				batchData,
				cfg.BlockTime,
				cfg.Genesis.L2Time,
				cfg.L2ChainID,
			)
			if err != nil {
				lgr.Warn("failed to derive span batch", "err", err)
				return batches
			}

			l2SafeHead := eth.L2BlockRef{
				Number:         cursor.Number,
				Time:           cursor.Timestamp,
				L1Origin:       cursor.L1Origin,
				SequenceNumber: cursor.SequenceNumber,
			}

			// Build l1Blocks slice starting from the cursor's epoch, as
			// CheckSpanBatchPrefix expects l1Blocks[0] to be the current epoch.
			var l1Blocks []eth.L1BlockRef
			for _, ref := range l1Origins {
				if ref.Number >= cursor.L1Origin.Number {
					l1Blocks = append(l1Blocks, ref)
				}
			}

			validity, _ := derive.CheckSpanBatchPrefix(
				context.Background(), cfg,
				lgr,
				l1Blocks, l2SafeHead, spanBatch, l1InclusionBlock, nil,
			)
			if validity == derive.BatchPast {
				lgr.Debug("span batch is past safe head, skipping")
				continue
			}
			if validity != derive.BatchAccept {
				lgr.Warn("span batch prefix check failed", "validity", validity)
				return batches
			}

			singular, err := spanBatch.GetSingularBatches(l1Origins, l2SafeHead)
			if err != nil {
				lgr.Warn("failed to expand span batch", "err", err)
				return batches
			}
			batches = append(batches, singular...)

		default:
			lgr.Warn("unknown batch type", "type", batchData.GetBatchType())
			return batches
		}
	}

	return batches
}

// validateBatch performs batch validation matching the checks in
// op-node/rollup/derive/batches.go (checkSingularBatch), minus checks that
// require L2 state access:
//   - Parent hash validation (deferred to post-execution via DerivedBlock.ExpectedParentHash)
//
// Returns BatchAccept, BatchPast, or BatchDrop. With Karst active (implying
// Holocene), past batches return BatchPast instead of BatchDrop.
//
// Overlapping span batches are already rejected in decodeBatches via
// CheckSpanBatchPrefix.
func validateBatch(lgr log.Logger, batch *derive.SingularBatch, cursor l2Cursor, l1Origins []eth.L1BlockRef, cfg *rollup.Config, l1InclusionNum uint64) derive.BatchValidity {
	expectedTimestamp := cursor.Timestamp + cfg.BlockTime

	// Holocene (implied by Karst): past batches are BatchPast, future batches are BatchDrop.
	if batch.Timestamp > expectedTimestamp {
		lgr.Warn("batch timestamp too new", "expected", expectedTimestamp, "got", batch.Timestamp)
		return derive.BatchDrop
	}
	if batch.Timestamp < expectedTimestamp {
		lgr.Debug("batch is past safe head", "expected", expectedTimestamp, "got", batch.Timestamp)
		return derive.BatchPast
	}

	epochNum := uint64(batch.EpochNum)

	// Sequence window: batch must be included within SeqWindowSize of its epoch.
	if epochNum+cfg.SeqWindowSize < l1InclusionNum {
		lgr.Warn("batch sequence window expired", "epoch", epochNum, "inclusion", l1InclusionNum, "window", cfg.SeqWindowSize)
		return derive.BatchDrop
	}

	// Epoch must be current or next (cannot skip epochs).
	if epochNum < cursor.L1Origin.Number {
		lgr.Warn("batch epoch too old", "epoch", epochNum, "cursor_origin", cursor.L1Origin.Number)
		return derive.BatchDrop
	}
	if epochNum > cursor.L1Origin.Number+1 {
		lgr.Warn("batch epoch too new", "epoch", epochNum, "cursor_origin", cursor.L1Origin.Number)
		return derive.BatchDrop
	}

	// Find the batch's L1 origin and verify epoch hash.
	var batchOrigin *eth.L1BlockRef
	for i := range l1Origins {
		if l1Origins[i].Number == epochNum {
			batchOrigin = &l1Origins[i]
			break
		}
	}
	if batchOrigin == nil {
		lgr.Warn("batch epoch L1 origin not found", "epoch", epochNum)
		return derive.BatchDrop
	}
	if batch.EpochHash != batchOrigin.Hash {
		lgr.Warn("batch epoch hash mismatch", "epoch", epochNum, "expected", batchOrigin.Hash, "got", batch.EpochHash)
		return derive.BatchDrop
	}

	// Batch timestamp must be >= L1 origin timestamp.
	if batch.Timestamp < batchOrigin.Time {
		lgr.Warn("batch timestamp before L1 origin", "batch_time", batch.Timestamp, "l1_time", batchOrigin.Time)
		return derive.BatchDrop
	}

	// Fork activation blocks must not contain user transactions.
	if (cfg.IsJovianActivationBlock(batch.Timestamp) ||
		cfg.IsKarstActivationBlock(batch.Timestamp) ||
		cfg.IsInteropActivationBlock(batch.Timestamp)) &&
		len(batch.Transactions) > 0 {
		lgr.Warn("batch has transactions at fork activation block")
		return derive.BatchDrop
	}

	// Sequencer time drift: L2 time must not exceed L1 time + MaxSequencerDrift.
	spec := rollup.NewChainSpec(cfg)
	maxDrift := batchOrigin.Time + spec.MaxSequencerDrift(batchOrigin.Time)
	if batch.Timestamp > maxDrift {
		if len(batch.Transactions) == 0 {
			// Empty batches may exceed drift to maintain L2 time >= L1 time invariant,
			// but only if they don't advance the epoch and the next origin isn't available.
			if epochNum == cursor.L1Origin.Number {
				for i := range l1Origins {
					if l1Origins[i].Number == epochNum+1 {
						if batch.Timestamp >= l1Origins[i].Time {
							lgr.Warn("empty batch exceeds drift but should have adopted next origin")
							return derive.BatchDrop
						}
						break
					}
				}
			}
		} else {
			lgr.Warn("batch exceeds sequencer drift", "batch_time", batch.Timestamp, "max_drift", maxDrift)
			return derive.BatchDrop
		}
	}

	// Transaction validation.
	isIsthmus := cfg.IsIsthmus(batch.Timestamp)
	for _, txBytes := range batch.Transactions {
		if len(txBytes) == 0 {
			lgr.Warn("batch contains empty transaction")
			return derive.BatchDrop
		}
		if txBytes[0] == types.DepositTxType {
			lgr.Warn("batch contains deposit transaction")
			return derive.BatchDrop
		}
		if !isIsthmus && txBytes[0] == types.SetCodeTxType {
			lgr.Warn("batch contains SetCode transaction before Isthmus")
			return derive.BatchDrop
		}
	}

	return derive.BatchAccept
}
