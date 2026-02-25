package pure

import (
	"fmt"
	"io"

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
	isFjord := cfg.IsFjord(cursor.Timestamp)

	readBatch, err := derive.BatchReader(r, maxRLP, isFjord)
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

// validateBatch performs simplified batch validation suitable for Karst and later.
// It checks timestamp sequencing, epoch bounds, and epoch hash consistency.
//
// This is a subset of the full validation in op-node/rollup/derive/batches.go
// (checkSingularBatch / CheckBatch). The upstream functions are unexported and
// require an l2Fetcher for L2 state lookups that we intentionally avoid.
// With Karst active, overlapping span batches are already rejected in decodeBatches,
// so the remaining checks here are sufficient for correctness.
func validateBatch(batch *derive.SingularBatch, cursor l2Cursor, l1Origins []eth.L1BlockRef, cfg *rollup.Config) bool {
	expectedTimestamp := cursor.Timestamp + cfg.BlockTime
	if batch.Timestamp != expectedTimestamp {
		return false
	}

	epochNum := uint64(batch.EpochNum)

	if epochNum < cursor.L1Origin.Number {
		return false
	}

	if len(l1Origins) == 0 {
		return false
	}
	latestOrigin := l1Origins[len(l1Origins)-1]
	if epochNum > latestOrigin.Number {
		return false
	}

	for _, origin := range l1Origins {
		if origin.Number == epochNum {
			return batch.EpochHash == origin.Hash
		}
	}

	return false
}

// makeEmptyBatch creates a batch with no transactions at the next expected
// timestamp, advancing from the current cursor position.
func makeEmptyBatch(cursor l2Cursor, cfg *rollup.Config) *derive.SingularBatch {
	return &derive.SingularBatch{
		EpochNum:  rollup.Epoch(cursor.L1Origin.Number),
		EpochHash: cursor.L1Origin.Hash,
		Timestamp: cursor.Timestamp + cfg.BlockTime,
	}
}
