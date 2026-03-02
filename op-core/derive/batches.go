package derive

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
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
// Span batch prefix validation is delegated to opderive.CheckSpanBatchPrefix,
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
) []*opderive.SingularBatch {
	spec := rollup.NewChainSpec(cfg)
	maxRLP := spec.MaxRLPBytesPerChannel(cursor.Timestamp)

	readBatch, err := opderive.BatchReader(r, maxRLP, true) // Fjord always active (implied by Karst)
	if err != nil {
		lgr.Warn("failed to create batch reader", "err", err)
		return nil
	}

	var batches []*opderive.SingularBatch
	for {
		batchData, err := readBatch()
		if err == io.EOF {
			break
		} else if err != nil {
			lgr.Warn("failed to read batch", "err", err)
			return batches
		}

		switch batchData.GetBatchType() {
		case opderive.SingularBatchType:
			singular, err := opderive.GetSingularBatch(batchData)
			if err != nil {
				lgr.Warn("failed to extract singular batch", "err", err)
				return batches
			}
			batches = append(batches, singular)

		case opderive.SpanBatchType:
			spanBatch, err := opderive.DeriveSpanBatch(
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

			validity, _ := opderive.CheckSpanBatchPrefix(
				context.Background(), cfg,
				lgr,
				l1Blocks, l2SafeHead, spanBatch, l1InclusionBlock, nil,
			)
			if validity == opderive.BatchPast {
				lgr.Debug("span batch is past safe head, skipping")
				continue
			}
			if validity != opderive.BatchAccept {
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
