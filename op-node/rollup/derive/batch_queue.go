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
	log          log.Logger
	inputs       []BatchesWithOrigin
	originOpen   bool // true if the last origin expects more batches
	resetting    bool // true if we are resetting the batch queue
	lastL1Origin eth.L1BlockRef
	config       *rollup.Config
	dl           L1ReceiptsFetcher
	next         BatchQueueOutput
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

func (bq *BatchQueue) CurrentOrigin() eth.L1BlockRef {
	last := bq.lastL1Origin
	if len(bq.inputs) != 0 {
		last = bq.inputs[len(bq.inputs)-1].Origin
	}
	return last
}

func (bq *BatchQueue) OpenOrigin(origin eth.L1BlockRef) error {
	parent := bq.CurrentOrigin()
	if parent.Hash != origin.ParentHash {
		return fmt.Errorf("cannot process L1 reorg from %s to %s (parent %s)", parent, origin.ID(), origin.ParentID())
	}
	bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: origin, Batches: nil})
	bq.originOpen = true
	return nil
}

func (bq *BatchQueue) AddBatch(batch *BatchData) error {
	if len(bq.inputs) == 0 {
		return fmt.Errorf("cannot add batch with timestamp %d, no origin was prepared", batch.Timestamp)
	}
	bq.inputs[len(bq.inputs)-1].Batches = append(bq.inputs[len(bq.inputs)-1].Batches, batch)
	return nil
}

func (bq *BatchQueue) CloseOrigin() {
	bq.originOpen = false
}

func (bq *BatchQueue) IsOriginOpen() bool {
	return bq.originOpen
}

// derive any L2 chain inputs, if we have any new batches
func (bq *BatchQueue) DeriveL2Inputs(ctx context.Context, lastL2Timestamp uint64) ([]*eth.PayloadAttributes, error) {
	if bq.originOpen || len(bq.inputs) == 0 {
		return nil, io.EOF
	}
	if uint64(len(bq.inputs)) < bq.config.SeqWindowSize {
		bq.log.Debug("not enough batches in batch queue, not deriving anything yet", "inputs", len(bq.inputs))
		return nil, io.EOF
	}
	if uint64(len(bq.inputs)) > bq.config.SeqWindowSize {
		return nil, fmt.Errorf("unexpectedly buffered more L1 inputs than sequencing window: %d", len(bq.inputs))
	}
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
		var txns []eth.Data
		l1InfoTx, err := L1InfoDepositBytes(uint64(i), l1Info)
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

func (bq *BatchQueue) Reset(l1Origin eth.L1BlockRef) {
	bq.lastL1Origin = l1Origin
}

func (bq *BatchQueue) Step(ctx context.Context) error {
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
		bq.lastL1Origin = l1SafeHead
		bq.resetting = true
		return nil
	}

	// we are done resetting if we have sufficient distance from the next stage to produce coherent results once we reach the origin of that stage.
	if bq.lastL1Origin.Number+bq.config.SeqWindowSize <= bq.next.SafeL2Head().L1Origin.Number || bq.lastL1Origin.Number == 0 {
		bq.inputs = bq.inputs[:0]
		bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: bq.lastL1Origin, Batches: nil})
		bq.originOpen = true
		bq.resetting = false
		return io.EOF
	}

	// not far back enough yet, do one more step
	parent, err := l1Fetcher.L1BlockRefByHash(ctx, bq.lastL1Origin.ParentHash)
	if err != nil {
		bq.log.Error("failed to fetch parent block while resetting batch queue", "err", err)
		return nil
	}
	bq.lastL1Origin = parent
	return nil
}
