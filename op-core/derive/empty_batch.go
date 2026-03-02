package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// makeEmptyBatch generates a single empty batch when the sequencing window has
// expired. It returns the batch, the L1 input for the batch's epoch (for
// buildAttributes), and the new L1 origin. Returns nil if no empty batch is needed.
//
// The epoch advances when the next L2 timestamp >= the next L1 block's timestamp.
func makeEmptyBatch(
	cursor l2Cursor,
	findL1 func(uint64) *L1Input,
	cfg *rollup.Config,
) (*opderive.SingularBatch, *L1Input, eth.BlockID) {
	nextTimestamp := cursor.Timestamp + cfg.BlockTime
	newOrigin := cursor.L1Origin
	epochL1 := findL1(cursor.L1Origin.Number)
	if epochL1 == nil {
		return nil, nil, eth.BlockID{}
	}

	nextL1 := findL1(cursor.L1Origin.Number + 1)
	if nextL1 != nil && nextTimestamp >= nextL1.Header.Time {
		newOrigin = nextL1.BlockID()
		epochL1 = nextL1
	}

	batch := &opderive.SingularBatch{
		EpochNum:  rollup.Epoch(newOrigin.Number),
		EpochHash: newOrigin.Hash,
		Timestamp: nextTimestamp,
	}

	return batch, epochL1, newOrigin
}
