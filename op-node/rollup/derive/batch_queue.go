package derive

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BatchesWithOrigin struct {
	Origin  eth.L1BlockRef
	Batches []*BatchData
}

type BatchQueue struct {
	log    log.Logger
	inputs []BatchesWithOrigin
	last   eth.L2BlockRef
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
func (bq *BatchQueue) DeriveL2Inputs() []*eth.PayloadAttributes {
	if len(bq.inputs) == 0 {
		return nil
	}

	// TODO implement sequencing window filtering
	batches := FilterBatches() // some refactoring to do

	// TODO: if it is time for the next batch, output it
	return nil
}

func (bq *BatchQueue) Reset(head eth.L2BlockRef) {
	bq.last = head
	bq.inputs = bq.inputs[:0]
}
