package derive

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

// state machine:
//
//  - buffer tagged data
//  - buffer batches
//  - buffer payload attributes

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) error
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, *big.Int) (*eth.ExecutionPayload, error)
}

// L2Derivation is updated with new L1 data, and the Step() function can be iterated on to keep the L2 Engine in sync.
type L2Derivation struct {
	log         log.Logger
	bank        *ChannelBank             // Where we buffer all incoming L1 data
	taggedData  []*TaggedData            // Where we buffer what we read from the bank
	batchReader *ChannelInReader         // Where we buffer tagged data to read batches from
	batchQueue  *BatchQueue              // Where we buffer all derived L2 batches
	attributes  []*eth.PayloadAttributes // Where we buffer all derived payload attributes
	engine      Engine                   // Final destination: the execution engine (EVM + chain and state DB)
}

func (l2d *L2Derivation) Input(origin eth.L1BlockRef) {

}

// Step tries to progress the buffer,
// and returns true if another step is expected to follow without new data in the buffer.
func (l2d *L2Derivation) Step() (readyNext bool) {
	taggedData := l2d.bank.Read()
	if taggedData == nil {
		return false
	}
	l2d.taggedData = append(l2d.taggedData, taggedData)

	prevOrigin := l2d.batchReader.CurrentL1Origin()

	// TODO: need to attach source of batch reader to l2 derivation buffer of tagged data
	var batch BatchData
	if err := l2d.batchReader.ReadBatch(&batch); err != nil {
		// if the stream closed, we need to reopen it, or close the rollup node
		if l2d.batchReader.Closed() {
			// TODO
		}
		// if the stream is not closed, we can recover by resetting.
		// E.g. a channel had invalid data, but a new batch submission on new channel can be read cleanly.
		l2d.batchReader.Reset()
		return true
	}

	currentOrigin := l2d.batchReader.CurrentL1Origin()
	for prevOrigin != currentOrigin {
		// TODO repeat this for each skipped origin

		l2d.batchQueue.AddOrigin()
	}

	if err := queue.AddBatch(&batch); err != nil {
		return nil, fmt.Errorf("failed to add batch to queue: %v", err)
	}

	return queue.DeriveL2Inputs(), nil
}
