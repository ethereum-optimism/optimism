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

type BatchQueueOutput interface {
	AddSafeAttributes(attributes *eth.PayloadAttributes)
	SafeL2Head() eth.L2BlockRef
}

type BatchesWithOrigin struct {
	Origin  eth.L1BlockRef
	Batches []*BatchData
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log       log.Logger
	inputs    []BatchesWithOrigin
	resetting bool // true if we are resetting the batch queue
	config    *rollup.Config
	dl        L1ReceiptsFetcher
	next      BatchQueueOutput
	progress  Progress
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, dl L1ReceiptsFetcher, next BatchQueueOutput) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		dl:     dl,
		next:   next,
	}
}

func (bq *BatchQueue) Progress() Progress {
	return bq.progress
}

func (bq *BatchQueue) AddBatch(batch *BatchData) error {
	if bq.progress.Closed {
		panic("write batch while closed")
	}
	bq.log.Warn("add batch", "origin", bq.progress.Origin, "tx_count", len(batch.Transactions), "timestamp", batch.Timestamp)
	if len(bq.inputs) == 0 {
		return fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp)
	}
	bq.inputs[len(bq.inputs)-1].Batches = append(bq.inputs[len(bq.inputs)-1].Batches, batch)
	return nil
}

// derive any L2 chain inputs, if we have any new batches
func (bq *BatchQueue) DeriveL2Inputs(ctx context.Context, lastL2Timestamp uint64) ([]*eth.PayloadAttributes, error) {
	// Wait for full data of the last origin, before deciding to fill with empty batches
	if !bq.progress.Closed || len(bq.inputs) == 0 {
		return nil, io.EOF
	}
	if uint64(len(bq.inputs)) < bq.config.SeqWindowSize {
		bq.log.Debug("not enough batches in batch queue, not deriving anything yet", "inputs", len(bq.inputs))
		return nil, io.EOF
	}
	if uint64(len(bq.inputs)) > bq.config.SeqWindowSize {
		return nil, fmt.Errorf("unexpectedly buffered more L1 inputs than sequencing window: %d", len(bq.inputs))
	}
	bq.log.Warn("deriving attributes")
	l1Origin := bq.inputs[0].Origin
	nextL1Block := bq.inputs[1].Origin

	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	l1Info, _, receipts, err := bq.dl.Fetch(fetchCtx, l1Origin.Hash)
	if err != nil {
		bq.log.Error("failed to fetch L1 block info", "l1Origin", l1Origin, "err", err)
		return nil, nil
	}

	deposits, errs := DeriveDeposits(receipts, bq.config.DepositContractAddress)
	for _, err := range errs {
		bq.log.Error("Failed to derive a deposit", "l1OriginHash", l1Origin.Hash, "err", err)
	}
	if len(errs) != 0 {
		return nil, fmt.Errorf("failed to derive some deposits: %v", errs)
	}

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
	batches = FilterBatches(bq.log, bq.config, epoch, minL2Time, maxL2Time, batches)
	batches = FillMissingBatches(batches, uint64(epoch), bq.config.BlockTime, minL2Time, nextL1Block.Time)
	var attributes []*eth.PayloadAttributes

	for i, batch := range batches {
		seqNr := uint64(i)
		if l1Info.Hash() == bq.config.Genesis.L1.Hash { // the genesis block is not derived, but does count as part of the first epoch: it takes seq nr 0
			seqNr += 1
		}
		var txns []eth.Data
		l1InfoTx, err := L1InfoDepositBytes(seqNr, l1Info)
		if err != nil {
			return nil, fmt.Errorf("failed to create l1InfoTx: %w", err)
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
		attributes = append(attributes, attrs) // TODO: direct assignment here
	}

	bq.inputs = bq.inputs[1:]

	return attributes, nil
}

func (bq *BatchQueue) Step(ctx context.Context, outer Progress) error {
	if changed, err := bq.progress.Update(outer); err != nil {
		return err
	} else if changed {
		if !bq.progress.Closed { // init inputs if we moved to a new open origin
			bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: bq.progress.Origin, Batches: nil})
		}
		return nil
	}

	attrs, err := bq.DeriveL2Inputs(ctx, bq.next.SafeL2Head().Time)
	if err != nil {
		return err
	}
	for _, attr := range attrs {
		if uint64(attr.Timestamp) <= bq.next.SafeL2Head().Time {
			// drop attributes if we are still progressing towards the next stage
			// (after a reset rolled us back a full sequence window)
			continue
		}
		bq.log.Warn("derived new payload attributes", "time", attr.Timestamp, "txs", len(attr.Transactions))
		bq.next.AddSafeAttributes(attr)
	}
	return nil
}

func (bq *BatchQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	// if we only just started resetting, find the origin corresponding to the safe L2 head
	if !bq.resetting {
		l2SafeHead := bq.next.SafeL2Head()
		l1SafeHead, err := l1Fetcher.L1BlockRefByHash(ctx, l2SafeHead.L1Origin.Hash)
		if err != nil {
			return fmt.Errorf("failed to find L1 reference corresponding to L1 origin %s of L2 block %s: %v", l2SafeHead.L1Origin, l2SafeHead.ID(), err)
		}
		bq.progress = Progress{
			Origin: l1SafeHead,
			Closed: false,
		}
		bq.resetting = true
		bq.log.Debug("set initial reset origin for batch queue", "origin", bq.progress.Origin)
		return nil
	}

	// we are done resetting if we have sufficient distance from the next stage to produce coherent results once we reach the origin of that stage.
	if bq.progress.Origin.Number+bq.config.SeqWindowSize < bq.next.SafeL2Head().L1Origin.Number || bq.progress.Origin.Number == 0 {
		bq.log.Debug("found reset origin for batch queue", "origin", bq.progress.Origin)
		bq.inputs = bq.inputs[:0]
		bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: bq.progress.Origin, Batches: nil})
		bq.resetting = false
		return io.EOF
	}

	bq.log.Debug("walking back to find reset origin for batch queue", "origin", bq.progress.Origin)

	// not far back enough yet, do one more step
	parent, err := l1Fetcher.L1BlockRefByHash(ctx, bq.progress.Origin.ParentHash)
	if err != nil {
		bq.log.Error("failed to fetch parent block while resetting batch queue", "err", err)
		return nil
	}
	bq.progress.Origin = parent
	return nil
}
