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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type outputImpl struct {
	dl     Downloader
	l2     Engine
	log    log.Logger
	Config rollup.Config
}

func (d *outputImpl) newBlock(ctx context.Context, l2Finalized eth.BlockID, l2Parent eth.BlockID, l2Safe eth.BlockID, l1Origin eth.BlockID, includeDeposits bool) (eth.BlockID, *derive.BatchData, error) {
	d.log.Info("creating new block", "l2Parent", l2Parent, "l1Origin", l1Origin, "includeDeposits", includeDeposits)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.l2.BlockByHash(fetchCtx, l2Parent.Hash)
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
	l1InfoTx, err := derive.L1InfoDepositBytes(l1Info)
	if err != nil {
		return l2Parent, nil, err
	}
	var txns []l2.Data
	txns = append(txns, l1InfoTx)
	deposits, err := derive.DeriveDeposits(l1Info.NumberU64(), receipts)
	d.log.Info("Derived deposits", "deposits", deposits, "l2Parent", l2Parent, "l1Origin", l1Origin)
	if err != nil {
		return l2Parent, nil, fmt.Errorf("failed to derive deposits: %v", err)
	}
	txns = append(txns, deposits...)

	depositStart := len(txns)

	attrs := &l2.PayloadAttributes{
		Timestamp:             hexutil.Uint64(timestamp),
		Random:                l2.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: d.Config.FeeRecipientAddress,
		Transactions:          txns,
		NoTxPool:              false,
	}
	fc := l2.ForkchoiceState{
		HeadBlockHash:      l2Parent.Hash,
		SafeBlockHash:      l2Safe.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}

	payload, err := d.addBlock(ctx, fc, attrs, false, true)
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
func (d *outputImpl) step(ctx context.Context, l2Head eth.BlockID, l2Finalized eth.BlockID, unsafeL2Head eth.BlockID, l1Input []eth.BlockID) (out eth.BlockID, err error) {
	// Sanity Checks
	if len(l1Input) == 0 {
		return l2Head, fmt.Errorf("empty L1 sequencing window on L2 %s", l2Head)
	}
	if len(l1Input) != int(d.Config.SeqWindowSize) {
		return l2Head, errors.New("Invalid sequencing window size")
	}

	logger := d.log.New("input_l1_first", l1Input[0], "input_l1_last", l1Input[len(l1Input)-1], "input_l2_parent", l2Head, "finalized_l2", l2Finalized)
	logger.Trace("Running update step on the L2 node")

	// Get inputs from L1 and L2
	epoch := rollup.Epoch(l1Input[0].Number)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.l2.BlockByHash(fetchCtx, l2Head.Hash)
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L2 block info of %s: %w", l2Head, err)
	}
	l1Info, err := d.dl.FetchL1Info(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch L1 block info of %s: %w", l1Input[0], err)
	}
	l1InfoTx, err := derive.L1InfoDepositBytes(l1Info)
	if err != nil {
		return l2Head, fmt.Errorf("failed to create l1InfoTx: %w", err)
	}
	receipts, err := d.dl.FetchReceipts(fetchCtx, l1Input[0])
	if err != nil {
		return l2Head, fmt.Errorf("failed to fetch receipts of %s: %w", l1Input[0], err)
	}
	deposits, err := derive.DeriveDeposits(uint64(epoch), receipts)
	if err != nil {
		return l2Head, fmt.Errorf("failed to derive deposits: %w", err)
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
	// Make batches contiguous
	minL2Time := l2Info.Time() + d.Config.BlockTime
	maxL2Time := l1Info.Time()
	batches = derive.FilterBatches(&d.Config, epoch, minL2Time, maxL2Time, batches)
	batches = derive.SortedAndPreparedBatches(batches, uint64(epoch), d.Config.BlockTime, minL2Time, maxL2Time)

	// Note: SafeBlockHash currently needs to be set b/c of Geth
	fc := l2.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2Head.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	updateUnsafeHead := unsafeL2Head.Hash == l2Head.Hash // If unsafe head is the same as the safe head, keep it up to date
	// Execute each L2 block in the epoch
	last := l2Head
	for i, batch := range batches {
		var txns []l2.Data
		txns = append(txns, l1InfoTx)
		if i == 0 {
			txns = append(txns, deposits...)
		}
		txns = append(txns, batch.Transactions...)
		attrs := &l2.PayloadAttributes{
			Timestamp:             hexutil.Uint64(batch.Timestamp),
			Random:                l2.Bytes32(l1Info.MixDigest()),
			SuggestedFeeRecipient: d.Config.FeeRecipientAddress,
			Transactions:          txns,
			NoTxPool:              false,
		}

		payload, err := d.addBlock(ctx, fc, attrs, true, updateUnsafeHead)
		if err != nil {
			return last, fmt.Errorf("failed to extend L2 chain at block %d/%d of epoch %d: %w", i, len(batches), epoch, err)
		}
		last = payload.ID()
		// TODO(Joshua): Update this to handle verifiers + sequencers
		fc.HeadBlockHash = last.Hash
		fc.SafeBlockHash = last.Hash
	}

	return last, nil
}

func (d *outputImpl) addBlock(ctx context.Context, fc l2.ForkchoiceState, attrs *l2.PayloadAttributes, updateSafe, updateUnsafe bool) (*l2.ExecutionPayload, error) {
	fcRes, err := d.l2.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block via forkchoice: %w", err)
	}
	id := fcRes.PayloadID
	if id == nil {
		return nil, errors.New("nil id in forkchoice result when expecting a valid ID")
	}
	payload, err := d.l2.GetPayload(ctx, *id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution payload: %w", err)
	}
	err = d.l2.ExecutePayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to insert execution payload: %w", err)
	}
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	if updateUnsafe {
		fc.HeadBlockHash = payload.BlockHash
	}
	_, err = d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err)
	}
	return payload, nil
}
