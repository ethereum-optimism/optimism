package pure

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PureDerive is the main entry point for pure derivation. It takes an L2 safe
// head, system config, and a sequence of L1 blocks and produces the derived L2
// blocks (as payload attributes) that follow from those inputs.
//
// The function is stateless and deterministic: given the same inputs it always
// produces the same outputs. No network access, no caching, no side effects.
//
// l1Blocks should contain L1 blocks starting from safeHead.L1Origin.Number.
// The caller must ensure this range is complete.
//
// Requires the Karst fork to be active at the safe head timestamp. Before Karst,
// span batches may overlap the safe chain, which this implementation does not support.
//
// Compared to the legacy pipeline (op-node/rollup/derive), this implementation
// intentionally skips the following checks:
//   - Parent hash validation against the actual L2 chain (deferred to post-execution
//     via DerivedBlock.ExpectedParentHash)
//   - L2 block hash verification (no L2 state access)
//   - Span batch overlap comparison (rejected by Karst; overlaps are invalid)
//   - Pipeline reset / reorg handling (caller is responsible for providing correct inputs)
//
// See op-node/rollup/derive/batches.go for the full upstream validation logic.
func PureDerive(
	cfg *rollup.Config,
	safeHead eth.L2BlockRef,
	sysConfig eth.SystemConfig,
	l1Blocks []L1Input,
) ([]DerivedBlock, error) {
	if !cfg.IsKarst(safeHead.Time) {
		return nil, fmt.Errorf("pure derivation requires Karst fork (no overlapping span batches), safe head time %d is pre-Karst", safeHead.Time)
	}

	if len(l1Blocks) == 0 {
		return nil, nil
	}

	// Validate that l1Blocks start from the safe head's L1 origin.
	firstL1Num := bigs.Uint64Strict(l1Blocks[0].Header.Number)
	if firstL1Num > safeHead.L1Origin.Number {
		return nil, fmt.Errorf("l1Blocks start at %d but safe head L1 origin is %d", firstL1Num, safeHead.L1Origin.Number)
	}

	cursor := newCursor(safeHead)
	assembler := newChannelAssembler()

	// Build index for O(1) lookups by L1 block number.
	l1ByNumber := make(map[uint64]int, len(l1Blocks))
	l1Origins := make([]eth.L1BlockRef, len(l1Blocks))
	for i := range l1Blocks {
		num := bigs.Uint64Strict(l1Blocks[i].Header.Number)
		l1ByNumber[num] = i
		l1Origins[i] = l1Blocks[i].BlockRef()
	}

	findL1 := func(number uint64) *L1Input {
		if idx, ok := l1ByNumber[number]; ok {
			return &l1Blocks[idx]
		}
		return nil
	}

	var derived []DerivedBlock

	for i := range l1Blocks {
		l1 := l1Blocks[i]
		l1Ref := l1.BlockRef()

		for _, log := range l1.ConfigLogs {
			if err := derive.ProcessSystemConfigUpdateLogEvent(&sysConfig, log, cfg, l1.Header.Time); err != nil {
				return nil, fmt.Errorf("processing system config update at L1 block %d: %w", l1Ref.Number, err)
			}
		}

		assembler.checkTimeout(l1Ref, cfg.ChannelTimeoutBedrock)

		for _, txData := range l1.BatcherData {
			frames, err := derive.ParseFrames(txData)
			if err != nil {
				continue
			}

			for _, frame := range frames {
				ready := assembler.addFrame(frame, l1Ref)
				if ready == nil {
					continue
				}

				batches, err := decodeBatches(ready.channel.Reader(), cfg, l1Origins, cursor, ready.openBlock)
				if err != nil {
					continue
				}

				for _, batch := range batches {
					if !validateBatch(batch, cursor, l1Origins, cfg, l1Ref.Number) {
						continue
					}

					epochL1 := findL1(uint64(batch.EpochNum))
					if epochL1 == nil {
						return nil, fmt.Errorf("missing L1 block %d for batch epoch", batch.EpochNum)
					}

					block, err := buildAttributes(batch, epochL1, cursor, sysConfig, cfg)
					if err != nil {
						return nil, fmt.Errorf("building attributes at L1 block %d: %w", l1Ref.Number, err)
					}
					derived = append(derived, *block)

					epochID := eth.BlockID{Number: uint64(batch.EpochNum), Hash: batch.EpochHash}
					var seqNum uint64
					if epochID.Number != cursor.L1Origin.Number {
						seqNum = 0
					} else {
						seqNum = cursor.SequenceNumber + 1
					}
					cursor.advance(batch.Timestamp, epochID, seqNum)
				}
			}
		}

		for cursor.needsEmptyBatch(l1Ref, cfg) {
			nextTimestamp := cursor.Timestamp + cfg.BlockTime
			newOrigin := cursor.L1Origin
			newSeqNum := cursor.SequenceNumber + 1

			// Advance epoch if the next L2 timestamp >= next L1 block's timestamp.
			nextL1 := findL1(cursor.L1Origin.Number + 1)
			if nextL1 != nil && nextTimestamp >= nextL1.Header.Time {
				newOrigin = nextL1.BlockID()
				newSeqNum = 0
			}

			emptyBatch := &derive.SingularBatch{
				EpochNum:  rollup.Epoch(newOrigin.Number),
				EpochHash: newOrigin.Hash,
				Timestamp: nextTimestamp,
			}

			epochL1 := findL1(newOrigin.Number)
			if epochL1 == nil {
				return nil, fmt.Errorf("missing L1 block %d for empty batch epoch", newOrigin.Number)
			}
			block, err := buildAttributes(emptyBatch, epochL1, cursor, sysConfig, cfg)
			if err != nil {
				return nil, fmt.Errorf("building empty batch attributes at L1 block %d: %w", l1Ref.Number, err)
			}
			derived = append(derived, *block)
			cursor.advance(emptyBatch.Timestamp, newOrigin, newSeqNum)
		}
	}

	return derived, nil
}
