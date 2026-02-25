package pure

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// decodeBatches reads all batches from a completed channel's compressed data
// and returns them as singular batches. Span batches are expanded into
// individual singular batches using the provided L1 origins and cursor.
//
// With Karst active, span batches must not overlap the safe chain. If the first
// batch in a span has timestamp <= cursor.Timestamp, the entire span is rejected.
// See checkSpanBatchPrefix in op-node/rollup/derive/batches.go for the full
// upstream overlap handling.
func decodeBatches(
	r io.Reader,
	cfg *rollup.Config,
	l1Origins []eth.L1BlockRef,
	cursor l2Cursor,
) ([]*derive.SingularBatch, error) {
	spec := rollup.NewChainSpec(cfg)
	maxRLP := spec.MaxRLPBytesPerChannel(cursor.Timestamp)

	readBatch, err := derive.BatchReader(r, maxRLP, true) // Fjord always active (implied by Karst)
	if err != nil {
		return nil, fmt.Errorf("creating batch reader: %w", err)
	}

	var batches []*derive.SingularBatch
	for {
		batchData, err := readBatch()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading batch: %w", err)
		}

		switch batchData.GetBatchType() {
		case derive.SingularBatchType:
			singular, err := derive.GetSingularBatch(batchData)
			if err != nil {
				return nil, fmt.Errorf("extracting singular batch: %w", err)
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
				return nil, fmt.Errorf("deriving span batch: %w", err)
			}

			// Reject overlapping span batches. Under Karst, span batches that start
			// at or before the safe head are invalid. This mirrors the overlap rejection
			// in checkSpanBatchPrefix (op-node/rollup/derive/batches.go).
			if spanBatch.GetTimestamp() <= cursor.Timestamp {
				return nil, fmt.Errorf("span batch timestamp %d overlaps safe head at %d (rejected under Karst)",
					spanBatch.GetTimestamp(), cursor.Timestamp)
			}

			l2SafeHead := eth.L2BlockRef{
				Number:         cursor.Number,
				Time:           cursor.Timestamp,
				L1Origin:       cursor.L1Origin,
				SequenceNumber: cursor.SequenceNumber,
			}
			singular, err := spanBatch.GetSingularBatches(l1Origins, l2SafeHead)
			if err != nil {
				return nil, fmt.Errorf("expanding span batch: %w", err)
			}
			batches = append(batches, singular...)

		default:
			return nil, fmt.Errorf("unknown batch type: %d", batchData.GetBatchType())
		}
	}

	return batches, nil
}

// validateBatch performs batch validation matching the checks in
// op-node/rollup/derive/batches.go (checkSingularBatch), minus checks that
// require L2 state access:
//   - Parent hash validation (deferred to post-execution via DerivedBlock.ExpectedParentHash)
//
// All other checks from checkSingularBatch are replicated here. With Karst active,
// overlapping span batches are already rejected in decodeBatches.
func validateBatch(batch *derive.SingularBatch, cursor l2Cursor, l1Origins []eth.L1BlockRef, cfg *rollup.Config, l1InclusionNum uint64) bool {
	// Timestamp must be the next expected L2 timestamp.
	expectedTimestamp := cursor.Timestamp + cfg.BlockTime
	if batch.Timestamp != expectedTimestamp {
		return false
	}

	epochNum := uint64(batch.EpochNum)

	// Sequence window: batch must be included within SeqWindowSize of its epoch.
	if epochNum+cfg.SeqWindowSize < l1InclusionNum {
		return false
	}

	// Epoch must be current or next (cannot skip epochs).
	if epochNum < cursor.L1Origin.Number {
		return false
	}
	if epochNum > cursor.L1Origin.Number+1 {
		return false
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
		return false
	}
	if batch.EpochHash != batchOrigin.Hash {
		return false
	}

	// Batch timestamp must be >= L1 origin timestamp.
	if batch.Timestamp < batchOrigin.Time {
		return false
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
							return false // should have adopted next origin
						}
						break
					}
				}
			}
		} else {
			return false
		}
	}

	// Fork activation blocks must not contain user transactions.
	if cfg.IsKarstActivationBlock(batch.Timestamp) && len(batch.Transactions) > 0 {
		return false
	}

	// Transaction validation.
	for _, txBytes := range batch.Transactions {
		if len(txBytes) == 0 {
			return false
		}
		if txBytes[0] == types.DepositTxType {
			return false
		}
	}

	return true
}
