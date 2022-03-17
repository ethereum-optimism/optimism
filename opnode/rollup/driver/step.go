package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/l2"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type Downloader interface {
	// FetchL1Info fetches the L1 header information corresponding to a L1 block ID
	FetchL1Info(ctx context.Context, id eth.BlockID) (derive.L1Info, error)
	// FetchReceipts of a L1 block
	FetchReceipts(ctx context.Context, id eth.BlockID) ([]*types.Receipt, error)
	// FetchTransactions from the given window of L1 blocks
	FetchTransactions(ctx context.Context, window []eth.BlockID) ([]*types.Transaction, error)
}

type DriverAPI interface {
	l2.EngineAPI
	l2.EthBackend
}

type outputImpl struct {
	dl     Downloader
	rpc    DriverAPI
	log    log.Logger
	Config rollup.Config
}

func (d *outputImpl) newBlock(ctx context.Context, l2Finalized eth.BlockID, l2Parent eth.BlockID, l1Origin eth.BlockID, includeDeposits bool) (eth.BlockID, *derive.BatchData, error) {
	d.log.Info("creating new block", "l2Parent", l2Parent, "l1Origin", l1Origin, "includeDeposits", includeDeposits)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.rpc.BlockByHash(fetchCtx, l2Parent.Hash)
	if err != nil {
		return l2Parent, nil, fmt.Errorf("failed to fetch L2 block info of %s: %v", l2Parent, err)
	}
	l1Info, err := d.dl.FetchL1Info(fetchCtx, l1Origin)
	if err != nil {
		return l2Parent, nil, fmt.Errorf("failed to fetch L1 block info of %s: %v", l1Origin, err)
	}

	timestamp := l2Info.Time() + d.Config.BlockTime
	if timestamp >= l1Info.Time() {
		return l2Parent, nil, errors.New("L2 Timestamp is too large")
	}

	var receipts types.Receipts
	if includeDeposits {
		receipts, err = d.dl.FetchReceipts(fetchCtx, l1Origin)
		if err != nil {
			return l2Parent, nil, fmt.Errorf("failed to fetch receipts of %s: %v", l1Origin, err)
		}

	}
	deposits, err := derive.DeriveDeposits(l1Info, receipts)
	d.log.Info("Derived deposits", "deposits", deposits, "l2Parent", l2Parent, "l1Origin", l1Origin)
	if err != nil {
		return l2Parent, nil, fmt.Errorf("failed to derive deposits: %v", err)
	}

	depositStart := len(deposits)

	attrs := &l2.PayloadAttributes{
		Timestamp:             hexutil.Uint64(timestamp),
		Random:                l2.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: d.Config.FeeRecipientAddress,
		Transactions:          deposits,
		NoTxPool:              false,
	}

	payload, err := AddBlock(ctx, d.log, d.rpc, l2Parent, l2Finalized.Hash, attrs)
	if err != nil {
		return l2Parent, nil, fmt.Errorf("failed to extend L2 chain: %v", err)
	}
	batch := &derive.BatchData{
		BatchV1: derive.BatchV1{
			Epoch:        rollup.Epoch(l1Info.NumberU64()),
			Timestamp:    uint64(payload.Timestamp),
			Transactions: payload.Transactions[depositStart:],
		},
	}

	return payload.ID(), batch, nil
}

// DriverStep derives and processes one or more L2 blocks from the given sequencing window of L1 blocks.
// An incomplete sequencing window will result in an incomplete L2 chain if so.
//
// After the step completes it returns the block ID of the last processed L2 block, even if an error occurs.
func (d *outputImpl) step(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, l1Input []eth.BlockID) (out eth.BlockID, err error) {
	if len(l1Input) == 0 {
		return l2Head, fmt.Errorf("empty L1 sequencing window on L2 %s", l2Head)
	}
	if len(l1Input) != int(d.Config.SeqWindowSize) {
		return l2Head, errors.New("Invalid sequencing window size")
	}

	logger := d.log.New("input_l1_first", l1Input[0], "input_l1_last", l1Input[len(l1Input)-1],
		"input_l2_parent", l2Head, "finalized_l2", l2Finalized)
	logger.Trace("Running update step on the L2 node")

	epoch := rollup.Epoch(l1Input[0].Number)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	l2Info, err := d.rpc.BlockByHash(fetchCtx, l2Head.Hash)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L2 block info of %s: %v", l2Head, err)
	}
	l1Info, err := d.dl.FetchL1Info(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L1 block info of %s: %v", l1Input[0], err)
	}
	receipts, err := d.dl.FetchReceipts(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch receipts of %s: %v", l1Input[0], err)
	}
	// TODO: with sharding the blobs may be identified in more detail than L1 block hashes
	transactions, err := d.dl.FetchTransactions(fetchCtx, l1Input)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch transactions from %s: %v", l1Input, err)
	}
	batches, err := derive.BatchesFromEVMTransactions(&d.Config, transactions)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch create batches from transactions: %w", err)
	}
	minL2Time := l2Info.Time() + d.Config.BlockTime
	maxL2Time := l1Info.Time()
	batches = derive.FilterBatches(&d.Config, epoch, minL2Time, maxL2Time, batches)

	attrsList, err := derive.PayloadAttributes(&d.Config, l1Info, receipts, batches, l2Info, minL2Time, maxL2Time)
	if err != nil {
		return l2Head, fmt.Errorf("failed to derive execution payload inputs: %v", err)
	}

	last := l2Head
	for i, attrs := range attrsList {
		payload, err := AddBlock(ctx, logger, d.rpc, last, l2Finalized.Hash, attrs)
		if err != nil {
			return last, fmt.Errorf("failed to extend L2 chain at block %d/%d of epoch %d: %v", i, len(attrsList), epoch, err)
		}
		last = payload.ID()
	}

	return last, nil
}

// AddBlock extends the L2 chain by deriving the full execution payload from inputs,
// and then executing and persisting it.
//
// After the step completes it returns the block ID of the last processed L2 block, even if an error occurs.
func AddBlock(ctx context.Context, logger log.Logger, rpc DriverAPI,
	l2Parent eth.BlockID, l2Finalized common.Hash, attrs *l2.PayloadAttributes) (*l2.ExecutionPayload, error) {

	payload, err := derive.ExecutionPayload(ctx, rpc, l2Parent.Hash, l2Finalized, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to derive execution payload: %v", err)
	}

	logger = logger.New("derived_l2", payload.ID())
	logger.Info("derived full block", "l2Parent", l2Parent, "attrs", attrs, "payload", payload)

	err = l2.ExecutePayload(ctx, rpc, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to apply execution payload: %v", err)
	}
	logger.Info("executed block")

	err = l2.ForkchoiceUpdate(ctx, rpc, payload.BlockHash, l2Finalized)
	if err != nil {
		return nil, fmt.Errorf("failed to persist execution payload: %v", err)
	}
	logger.Info("updated fork-choice with block")
	return payload, nil
}
