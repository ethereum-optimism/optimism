package pure

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PureDerive is the main entry point for pure derivation. It takes an L2 safe
// head, system config, and a sequence of L1 blocks and produces the derived L2
// blocks (as payload attributes) that follow from those inputs.
//
// The function is stateless and deterministic: given the same inputs it always
// produces the same outputs. No network access, no caching, no side effects.
func PureDerive(
	cfg *rollup.Config,
	safeHead eth.L2BlockRef,
	sysConfig eth.SystemConfig,
	l1Blocks []L1Input,
) ([]DerivedBlock, error) {
	cursor := newCursor(safeHead)
	assembler := newChannelAssembler()

	l1Origins := make([]eth.L1BlockRef, len(l1Blocks))
	for i := range l1Blocks {
		l1Origins[i] = l1Blocks[i].BlockRef()
	}

	var derived []DerivedBlock

	for i := range l1Blocks {
		l1 := l1Blocks[i]
		l1Ref := l1.BlockRef()

		for _, log := range l1.ConfigLogs {
			if err := derive.ProcessSystemConfigUpdateLogEvent(&sysConfig, log, cfg, l1.Timestamp); err != nil {
				return nil, fmt.Errorf("processing system config update at L1 block %d: %w", l1.Number, err)
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

				batches, err := decodeBatches(ready.channel.Reader(), cfg, l1Origins, cursor)
				if err != nil {
					continue
				}

				for _, batch := range batches {
					if !validateBatch(batch, cursor, l1Origins, cfg) {
						continue
					}

					epochL1 := findL1Origin(l1Blocks, uint64(batch.EpochNum))
					if epochL1 == nil {
						epochL1 = &l1
					}

					block, err := buildAttributes(batch, epochL1, cursor, sysConfig, cfg)
					if err != nil {
						return nil, fmt.Errorf("building attributes at L1 block %d: %w", l1.Number, err)
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

		for needsEmptyBatch(cursor, l1Ref, cfg) {
			nextTimestamp := cursor.Timestamp + cfg.BlockTime
			newOrigin := cursor.L1Origin
			newSeqNum := cursor.SequenceNumber + 1

			// Advance epoch if the next L2 timestamp >= next L1 block's timestamp.
			nextL1 := findL1Origin(l1Blocks, cursor.L1Origin.Number+1)
			if nextL1 != nil && nextTimestamp >= nextL1.Timestamp {
				newOrigin = nextL1.BlockID()
				newSeqNum = 0
			}

			emptyBatch := &derive.SingularBatch{
				EpochNum:  rollup.Epoch(newOrigin.Number),
				EpochHash: newOrigin.Hash,
				Timestamp: nextTimestamp,
			}

			epochL1 := findL1Origin(l1Blocks, newOrigin.Number)
			if epochL1 == nil {
				epochL1 = &l1
			}
			block, err := buildAttributes(emptyBatch, epochL1, cursor, sysConfig, cfg)
			if err != nil {
				return nil, fmt.Errorf("building empty batch attributes at L1 block %d: %w", l1.Number, err)
			}
			derived = append(derived, *block)
			cursor.advance(emptyBatch.Timestamp, newOrigin, newSeqNum)
		}
	}

	return derived, nil
}

// findL1Origin looks up an L1Input by block number from the provided slice.
func findL1Origin(l1Blocks []L1Input, number uint64) *L1Input {
	for i := range l1Blocks {
		if l1Blocks[i].Number == number {
			return &l1Blocks[i]
		}
	}
	return nil
}
