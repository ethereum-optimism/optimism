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

type BatchesWithOrigin struct {
	Origin  eth.L1BlockRef
	Batches []*BatchData
}

// BatchQueue contains a set of batches for every L1 block.
// L1 blocks are contiguous and this does not support reorgs.
type BatchQueue struct {
	log          log.Logger
	inputs       []BatchesWithOrigin
	lastL1Origin eth.L1BlockRef
	config       *rollup.Config
	dl           L1ReceiptsFetcher
}

// NewBatchQueue creates a BatchQueue, which should be Reset(origin) before use.
func NewBatchQueue(log log.Logger, cfg *rollup.Config, dl L1ReceiptsFetcher) *BatchQueue {
	return &BatchQueue{
		log:    log,
		config: cfg,
		dl:     dl,
	}
}

func (bq *BatchQueue) LastL1Origin() eth.L1BlockRef {
	last := bq.lastL1Origin
	if len(bq.inputs) != 0 {
		last = bq.inputs[len(bq.inputs)-1].Origin
	}
	return last
}

func (bq *BatchQueue) AddOrigin(origin eth.L1BlockRef) error {
	parent := bq.LastL1Origin()
	if parent.Hash != origin.ParentHash {
		return fmt.Errorf("cannot process L1 reorg from %s to %s (parent %s)", parent, origin.ID(), origin.ParentID())
	}
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
func (bq *BatchQueue) DeriveL2Inputs(ctx context.Context, lastL2Timestamp uint64) ([]*eth.PayloadAttributes, error) {
	if len(bq.inputs) == 0 {
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
	bq.inputs = bq.inputs[:0]
	bq.inputs = append(bq.inputs, BatchesWithOrigin{Origin: l1Origin, Batches: nil})
}
