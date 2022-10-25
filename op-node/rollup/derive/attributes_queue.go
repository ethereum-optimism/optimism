package derive

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// The attributes queue sits in between the batch queue and the engine queue
// It transforms batches into payload attributes. The outputted payload
// attributes cannot be buffered because each batch->attributes transformation
// pulls in data about the current L2 safe head.
//
// It also buffers batches that have been output because multiple batches can
// be created at once.
//
// This stage can be reset by clearing it's batch buffer.
// This stage does not need to retain any references to L1 blocks.

type AttributesQueue struct {
	log    log.Logger
	config *rollup.Config
	dl     L1ReceiptsFetcher
	eng    SystemConfigL2Fetcher
	prev   *BatchQueue
	batch  *BatchData
}

func NewAttributesQueue(log log.Logger, cfg *rollup.Config, l1Fetcher L1ReceiptsFetcher, eng SystemConfigL2Fetcher, prev *BatchQueue) *AttributesQueue {
	return &AttributesQueue{
		log:    log,
		config: cfg,
		dl:     l1Fetcher,
		eng:    eng,
		prev:   prev,
	}
}

func (aq *AttributesQueue) Origin() eth.L1BlockRef {
	return aq.prev.Origin()
}

func (aq *AttributesQueue) NextAttributes(ctx context.Context, l2SafeHead eth.L2BlockRef) (*eth.PayloadAttributes, error) {
	// Get a batch if we need it
	if aq.batch == nil {
		batch, err := aq.prev.NextBatch(ctx, l2SafeHead)
		if err != nil {
			return nil, err
		}
		aq.batch = batch
	}

	// Actually generate the next attributes
	if attrs, err := aq.createNextAttributes(ctx, aq.batch, l2SafeHead); err != nil {
		return nil, err
	} else {
		// Clear out the local state once we will succeed
		aq.batch = nil
		return attrs, nil
	}

}

// createNextAttributes transforms a batch into a payload attributes. This sets `NoTxPool` and appends the batched transactions
// to the attributes transaction list
func (aq *AttributesQueue) createNextAttributes(ctx context.Context, batch *BatchData, l2SafeHead eth.L2BlockRef) (*eth.PayloadAttributes, error) {
	// sanity check parent hash
	if batch.ParentHash != l2SafeHead.Hash {
		return nil, NewResetError(fmt.Errorf("valid batch has bad parent hash %s, expected %s", batch.ParentHash, l2SafeHead.Hash))
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	attrs, err := PreparePayloadAttributes(fetchCtx, aq.config, aq.dl, aq.eng, l2SafeHead, batch.Timestamp, batch.Epoch())
	if err != nil {
		return nil, err
	}

	// we are verifying, not sequencing, we've got all transactions and do not pull from the tx-pool
	// (that would make the block derivation non-deterministic)
	attrs.NoTxPool = true
	attrs.Transactions = append(attrs.Transactions, batch.Transactions...)

	aq.log.Info("generated attributes in payload queue", "txs", len(attrs.Transactions), "timestamp", batch.Timestamp)

	return attrs, nil
}

func (aq *AttributesQueue) Reset(ctx context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	return io.EOF
}
