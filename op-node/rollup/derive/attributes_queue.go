package derive

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

type AttributesQueueOutput interface {
	AddSafeAttributes(attributes *eth.PayloadAttributes)
	SafeL2Head() eth.L2BlockRef
	StageProgress
}

type AttributesQueue struct {
	log      log.Logger
	config   *rollup.Config
	dl       L1ReceiptsFetcher
	next     AttributesQueueOutput
	progress Progress
	batches  []*BatchData
}

func NewAttributesQueue(log log.Logger, cfg *rollup.Config, l1Fetcher L1ReceiptsFetcher, next AttributesQueueOutput) *AttributesQueue {
	return &AttributesQueue{
		log:    log,
		config: cfg,
		dl:     l1Fetcher,
		next:   next,
	}
}

func (aq *AttributesQueue) AddBatch(batch *BatchData) {
	aq.log.Debug("Received next batch", "batch_epoch", batch.EpochNum, "batch_timestamp", batch.Timestamp, "tx_count", len(batch.Transactions))
	aq.batches = append(aq.batches, batch)
}

func (aq *AttributesQueue) Progress() Progress {
	return aq.progress
}

func (aq *AttributesQueue) Step(ctx context.Context, outer Progress) error {
	if changed, err := aq.progress.Update(outer); err != nil || changed {
		return err
	}
	if len(aq.batches) == 0 {
		return io.EOF
	}
	batch := aq.batches[0]

	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	attrs, crit, err := PreparePayloadAttributes(fetchCtx, aq.config, aq.dl, aq.next.SafeL2Head(), batch.Epoch())
	if err != nil {
		if crit {
			return fmt.Errorf("failed to prepare payload attributes for batch: %v", err)
		} else {
			aq.log.Error("temporarily failing to prepare payload attributes for batch", "err", err)
			return nil
		}
	}

	// we are verifying, not sequencing, we've got all transactions and do not pull from the tx-pool
	// (that would make the block derivation non-deterministic)
	attrs.NoTxPool = true
	attrs.Transactions = append(attrs.Transactions, batch.Transactions...)

	aq.log.Info("generated attributes in payload queue", "txs", len(attrs.Transactions), "timestamp", batch.Timestamp)

	// Slice off the batch once we are guaranteed to succeed
	aq.batches = aq.batches[1:]

	aq.next.AddSafeAttributes(attrs)
	return nil
}

func (aq *AttributesQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	aq.batches = aq.batches[:0]
	aq.progress = aq.next.Progress()
	return io.EOF
}

func (aq *AttributesQueue) SafeL2Head() eth.L2BlockRef {
	return aq.next.SafeL2Head()
}
