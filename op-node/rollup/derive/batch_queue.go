package derive

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type BatchesWithOrigin struct {
	Origin  eth.L1BlockRef
	Batches []*BatchData
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log    log.Logger
	inputs []BatchesWithOrigin
	last   eth.L2BlockRef
	config *rollup.Config
}

func (bq *BatchQueue) lastOrigin() eth.BlockID {
	last := bq.last.L1Origin
	if len(bq.inputs) != 0 {
		last = bq.inputs[len(bq.inputs)-1].Origin.ID()
	}
	return last
}

func (bq *BatchQueue) AddOrigin(origin eth.L1BlockRef) error {
	parent := bq.lastOrigin()
	if parent.Hash != origin.ParentHash {
		return fmt.Errorf("cannot process L1 reorg from %s to %s (parent %s)", parent, origin.ID(), origin.ParentID())
	}
	// TODO: add batches to previous input, if it was empty

	bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: origin, Batches: nil})
	return nil
}

func (bq *BatchQueue) AddBatch(batch *BatchData) error {
	if len(bq.inputs) == 0 {
		return fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp)
	}
	bq.inputs[len(bq.inputs)-1].Batches = append(bq.inputs[len(bq.inputs)-1].Batches, batch)
	return nil
}

// derive any L2 chain inputs, if we have any new batches
func (bq *BatchQueue) DeriveL2Inputs(lastL2Timestamp uint64) []*eth.PayloadAttributes {
	if len(bq.inputs) == 0 {
		return nil
	}
	if uint64(len(bq.inputs)) < bq.config.SeqWindowSize {
		return nil
	}

	// TODO: pull in this data from l1Origin
	var l1Info L1Info

	// TODO: Need receipts for every block as well.
	var deposits []hexutil.Bytes
	// deposits, errs := DeriveDeposits(receipts, d.Config.DepositContractAddress)
	// for _, err := range errs {
	// 	d.log.Error("Failed to derive a deposit", "l1OriginHash", l1Input[0].Hash, "err", err)
	// }

	l1Origin := bq.inputs[0].Origin
	nextL1Block := bq.inputs[1].Origin
	epoch := rollup.Epoch(l1Origin.Number)
	minL2Time := uint64(lastL2Timestamp) + bq.config.BlockTime
	maxL2Time := l1Origin.Time + bq.config.MaxSequencerDrift
	if minL2Time+bq.config.BlockTime > maxL2Time {
		maxL2Time = minL2Time + bq.config.BlockTime
	}
	var batches []*BatchData
	for _, b := range bq.inputs {
		batches = append(batches, b.Batches...)
	}
	batches = FilterBatches(bq.config, epoch, minL2Time, maxL2Time, batches)
	batches = FillMissingBatches(batches, uint64(epoch), bq.config.BlockTime, minL2Time, nextL1Block.Time)
	var attributes []*eth.PayloadAttributes

	for i, batch := range batches {
		var txns []eth.Data
		l1InfoTx, err := L1InfoDepositBytes(uint64(i), l1Info)
		if err != nil {
			return nil // , fmt.Errorf("failed to create l1InfoTx: %w", err)
		}
		txns = append(txns, l1InfoTx)
		if i == 0 {
			txns = append(txns, deposits...)
		}
		txns = append(txns, batch.Transactions...)
		attrs := &eth.PayloadAttributes{
			Timestamp:             hexutil.Uint64(batch.Timestamp),
			PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
			SuggestedFeeRecipient: bq.config.FeeRecipientAddress,
			Transactions:          txns,
			// we are verifying, not sequencing, we've got all transactions and do not pull from the tx-pool
			// (that would make the block derivation non-deterministic)
			NoTxPool: true,
		}
		attributes = append(attributes, attrs) // TOOD: direct assignment here
	}

	return attributes
}

func (bq *BatchQueue) Reset(head eth.L2BlockRef) {
	bq.last = head
	bq.inputs = bq.inputs[:0]
}
