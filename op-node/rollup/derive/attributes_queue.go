package derive

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1ReceiptsFetcher interface {
	Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error)
}

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
	if aq.progress.Closed {
		panic("adding batch while attributes queue is closed")
	}
	aq.log.Info("Received next batch", "batch_epoch", batch.EpochNum, "batch_timestamp", batch.Timestamp, "tx_count", len(batch.Transactions))
	aq.batches = append(aq.batches, batch)
}

func (aq *AttributesQueue) Progress() Progress {
	return aq.progress
}

func (aq *AttributesQueue) Step(ctx context.Context, outer Progress) error {
	if changed, err := aq.progress.Update(outer); err != nil || changed {
		return err
	}
	attr, err := aq.DeriveL2Inputs(ctx, aq.next.SafeL2Head())
	if err != nil {
		return err
	}
	aq.next.AddSafeAttributes(attr)
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

// DeriveL2Inputs turns the next L2 batch into an Payload Attributes that builds off of the safe head
func (aq *AttributesQueue) DeriveL2Inputs(ctx context.Context, l2SafeHead eth.L2BlockRef) (*eth.PayloadAttributes, error) {
	if len(aq.batches) == 0 {
		return nil, io.EOF
	}
	batch := aq.batches[0]

	seqNumber := l2SafeHead.SequenceNumber + 1
	// Check if we need to advance an epoch & update local state
	if l2SafeHead.L1Origin != batch.Epoch() {
		aq.log.Info("advancing epoch in the attributes queue", "l2SafeHead", l2SafeHead, "l2SafeHead_origin", l2SafeHead.L1Origin, "batch_timestamp", batch.Timestamp, "batch_epoch", batch.Epoch())
		seqNumber = 0
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	l1Info, _, receipts, err := aq.dl.Fetch(fetchCtx, batch.EpochHash)
	if err != nil {
		aq.log.Error("failed to fetch L1 block info", "l1Origin", batch.Epoch(), "err", err)
		return nil, err
	}

	// Fill in deposits if we are the first block of the epoch
	var deposits []hexutil.Bytes
	if seqNumber == 0 {
		fetchCtx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		var errs []error
		deposits, errs = DeriveDeposits(receipts, aq.config.DepositContractAddress)
		for _, err := range errs {
			aq.log.Error("Failed to derive a deposit", "l1Origin", batch.Epoch(), "err", err)
		}
		if len(errs) != 0 {
			// TODO: Multierror here
			return nil, fmt.Errorf("failed to derive some deposits: %v", errs)
		}
	}

	var txns []eth.Data
	l1InfoTx, err := L1InfoDepositBytes(seqNumber, l1Info)
	if err != nil {
		return nil, fmt.Errorf("failed to create l1InfoTx: %w", err)
	}
	txns = append(txns, l1InfoTx)
	if seqNumber == 0 {
		txns = append(txns, deposits...)
	}
	txns = append(txns, batch.Transactions...)
	attrs := &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(batch.Timestamp),
		PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: aq.config.FeeRecipientAddress,
		Transactions:          txns,
		// we are verifying, not sequencing, we've got all transactions and do not pull from the tx-pool
		// (that would make the block derivation non-deterministic)
		NoTxPool: true,
	}
	aq.log.Info("generated attributes in payload queue", "tx_count", len(txns), "timestamp", batch.Timestamp)

	// Slice off the batch once we are guaranteed to succeed
	aq.batches = aq.batches[1:]
	return attrs, nil
}
