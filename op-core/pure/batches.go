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
			l2SafeHead := eth.L2BlockRef{
				Number:     cursor.Number,
				Time:       cursor.Timestamp,
				L1Origin:   cursor.L1Origin,
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

// validateBatch checks whether a singular batch is valid given the current
// derivation cursor and known L1 origins.
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

// needsEmptyBatch returns true when the sequencing window has expired,
// meaning the cursor's L1 origin is more than SeqWindowSize blocks behind
// the current L1 block.
func needsEmptyBatch(cursor l2Cursor, currentL1 eth.L1BlockRef, cfg *rollup.Config) bool {
	return currentL1.Number > cursor.L1Origin.Number+cfg.SeqWindowSize
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
