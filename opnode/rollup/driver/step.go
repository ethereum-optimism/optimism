package driver

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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

type outputImpl struct {
	dl     Downloader
	l2     Engine
	log    log.Logger
	Config rollup.Config
}

func (d *outputImpl) createNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.BlockID) (eth.L2BlockRef, *derive.BatchData, error) {
	d.log.Info("creating new block", "l2Head", l2Head)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.l2.BlockByHash(fetchCtx, l2Head.Hash)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to fetch L2 block info of %s: %v", l2Head, err)
	}
	l2BLockRef, err := derive.BlockReferences(l2Info, &d.Config.Genesis)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to derive L2BlockRef from l2Block: %w", err)
	}

	var l1Info derive.L1Info
	var receipts types.Receipts
	// Include deposits if this is the first block of an epoch
	if l2BLockRef.L1Origin.Number != l1Origin.Number {
		l1Info, _, receipts, err = d.dl.Fetch(fetchCtx, l1Origin.Hash)
	} else {
		l1Info, err = d.dl.InfoByHash(fetchCtx, l1Origin.Hash)
		// don't fetch receipts if we do not process deposits
	}
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to fetch L1 block info of %s: %v", l1Origin, err)
	}

	timestamp := l2Info.Time() + d.Config.BlockTime
	if timestamp > l1Info.Time()+d.Config.MaxSequencerDrift {
		return l2Head, nil, errors.New("no slack left, L2 Timestamp is too large")
	}

	l1InfoTx, err := derive.L1InfoDepositBytes(l2Head.Number+1, l1Info)
	if err != nil {
		return l2Head, nil, err
	}
	var txns []l2.Data
	txns = append(txns, l1InfoTx)
	deposits, err := derive.DeriveDeposits(l2Head.Number+1, receipts)
	d.log.Info("Derived deposits", "deposits", deposits, "l2Parent", l2Head, "l1Origin", l1Origin)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to derive deposits: %v", err)
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
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}

	payload, err := d.insertHeadBlock(ctx, fc, attrs, false)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to extend L2 chain: %v", err)
	}
	batch := &derive.BatchData{
		BatchV1: derive.BatchV1{
			Epoch:        rollup.Epoch(l1Info.NumberU64()),
			Timestamp:    uint64(payload.Timestamp),
			Transactions: payload.TransactionsField[depositStart:],
		},
	}
	ref, err := derive.BlockReferences(payload, &d.Config.Genesis)
	return ref, batch, err
}

// insertEpoch creates and inserts one epoch on top of the safe head. It prefers blocks it creates to what is recorded in the unsafe chain.
// It returns the new L2 head and L2 Safe head and if there was a reorg. This function must return if there was a reorg otherwise the L2 chain must be traversed.
func (d *outputImpl) insertEpoch(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.L2BlockRef, l2Finalized eth.BlockID, l1Input []eth.BlockID) (eth.L2BlockRef, eth.L2BlockRef, bool, error) {
	// Sanity Checks
	if len(l1Input) <= 1 {
		return l2Head, l2SafeHead, false, fmt.Errorf("too small L1 sequencing window for L2 derivation on %s: %v", l2SafeHead, l1Input)
	}
	if len(l1Input) != int(d.Config.SeqWindowSize) {
		return l2Head, l2SafeHead, false, errors.New("invalid sequencing window size")
	}

	logger := d.log.New("input_l1_first", l1Input[0], "input_l1_last", l1Input[len(l1Input)-1], "input_l2_parent", l2SafeHead, "finalized_l2", l2Finalized)
	logger.Trace("Running update step on the L2 node")

	// Get inputs from L1 and L2
	epoch := rollup.Epoch(l1Input[0].Number)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.l2.BlockByHash(fetchCtx, l2SafeHead.Hash)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to fetch L2 block info of %s: %w", l2SafeHead, err)
	}
	l1Info, _, receipts, err := d.dl.Fetch(fetchCtx, l1Input[0].Hash)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to fetch L1 block info of %s: %w", l1Input[0], err)
	}
	if l2SafeHead.L1Origin.Hash != l1Info.ParentHash() {
		return l2Head, l2SafeHead, false, fmt.Errorf("l1Info %v does not extend L1 Origin (%v) of L2 Safe Head (%v)", l1Info.Hash(), l2SafeHead.L1Origin, l2SafeHead)
	}
	nextL1Block, err := d.dl.InfoByHash(ctx, l1Input[1].Hash)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to get L1 timestamp of next L1 block: %v", err)
	}
	deposits, err := derive.DeriveDeposits(l2SafeHead.Number+1, receipts)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to derive deposits: %w", err)
	}
	// TODO: with sharding the blobs may be identified in more detail than L1 block hashes
	transactions, err := d.dl.FetchAllTransactions(fetchCtx, l1Input)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to fetch transactions from %s: %v", l1Input, err)
	}
	batches, err := derive.BatchesFromEVMTransactions(&d.Config, transactions)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to fetch create batches from transactions: %w", err)
	}
	// Make batches contiguous
	minL2Time := l2Info.Time() + d.Config.BlockTime
	maxL2Time := l1Info.Time() + d.Config.MaxSequencerDrift
	if minL2Time+d.Config.BlockTime > maxL2Time {
		maxL2Time = minL2Time + d.Config.BlockTime
	}
	batches = derive.FilterBatches(&d.Config, epoch, minL2Time, maxL2Time, batches)
	batches = derive.FillMissingBatches(batches, uint64(epoch), d.Config.BlockTime, minL2Time, nextL1Block.Time())

	fc := l2.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	// Execute each L2 block in the epoch
	lastHead := l2Head
	lastSafeHead := l2SafeHead
	didReorg := false
	var payload derive.Block
	var reorg bool
	for i, batch := range batches {
		var txns []l2.Data
		l1InfoTx, err := derive.L1InfoDepositBytes(lastSafeHead.Number+1, l1Info)
		if err != nil {
			return l2Head, l2SafeHead, false, fmt.Errorf("failed to create l1InfoTx: %w", err)
		}
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

		// We are either verifying blocks (with a potential for a reorg) or inserting a safe head to the chain
		if lastHead.Hash != lastSafeHead.Hash {
			payload, reorg, err = d.verifySafeBlock(ctx, fc, attrs, lastSafeHead.ID())

		} else {
			payload, err = d.insertHeadBlock(ctx, fc, attrs, true)
		}
		if err != nil {
			return lastHead, lastSafeHead, didReorg, fmt.Errorf("failed to extend L2 chain at block %d/%d of epoch %d: %w", i, len(batches), epoch, err)
		}

		newLast, err := derive.BlockReferences(payload, &d.Config.Genesis)
		if err != nil {
			return lastHead, lastSafeHead, didReorg, fmt.Errorf("failed to derive block references: %w", err)
		}
		if reorg {
			didReorg = true
		}
		// If reorg or the L2 Head is not ahead of the safe head, bump the head block.
		if reorg || lastHead.Hash == lastSafeHead.Hash {
			lastHead = newLast
		}
		lastSafeHead = newLast

		fc.HeadBlockHash = lastHead.Hash
		fc.SafeBlockHash = lastSafeHead.Hash
	}

	return lastHead, lastSafeHead, didReorg, nil
}

// attributesMatchBlock checks if the L2 attributes pre-inputs match the output
// nil if it is a match. If err is not nil, the error contains the reason for the mismatch
func attributesMatchBlock(attrs *l2.PayloadAttributes, parentHash common.Hash, block *types.Block) error {
	if parentHash != block.ParentHash() {
		return fmt.Errorf("parent hash field does not match. expected: %v. got: %v", parentHash, block.ParentHash())
	}
	if uint64(attrs.Timestamp) != block.Time() {
		return fmt.Errorf("timestamp field does not match. expected: %v. got: %v", uint64(attrs.Timestamp), block.Time())
	}
	if attrs.Random != l2.Bytes32(block.MixDigest()) {
		return fmt.Errorf("random field does not match. expected: %v. got: %v", attrs.Random, l2.Bytes32(block.MixDigest()))
	}
	if len(attrs.Transactions) != len(block.Transactions()) {
		return fmt.Errorf("transaction count does not match. expected: %v. got: %v", len(attrs.Transactions), len(block.Transactions()))
	}
	btxs := block.Transactions()
	for i := range attrs.Transactions {
		var tx types.Transaction
		err := tx.UnmarshalBinary(attrs.Transactions[i])
		if err != nil {
			return fmt.Errorf("failed to decode transaction %d in attributes: %w", i, err)
		}

		if tx.Hash() != btxs[i].Hash() {
			return fmt.Errorf("transaction %d does not match. expected: %v. got: %v", i, tx.Hash(), btxs[i].Hash())
		}
	}
	return nil
}

// verifySafeBlock reconciles the supplied payload attributes against the actual L2 block.
// If they do not match, it inserts the new block and sets the head and safe head to the new block in the FC.
func (d *outputImpl) verifySafeBlock(ctx context.Context, fc l2.ForkchoiceState, attrs *l2.PayloadAttributes, parent eth.BlockID) (derive.Block, bool, error) {
	block, err := d.l2.BlockByNumber(ctx, new(big.Int).SetUint64(parent.Number+1))
	if err != nil {
		return nil, false, fmt.Errorf("failed to get L2 block: %w", err)
	}
	err = attributesMatchBlock(attrs, parent.Hash, block)
	if err != nil {
		// Have reorg
		d.log.Warn("Detected L2 reorg when verifying L2 safe head", "parent", parent, "prev_block", block.Hash(), "mismatch", err)
		fc.HeadBlockHash = parent.Hash
		fc.SafeBlockHash = parent.Hash
		payload, err := d.insertHeadBlock(ctx, fc, attrs, true)
		return payload, true, err
	}
	// If match, just bump the safe head
	d.log.Debug("Verified L2 block", "number", block.Number(), "hash", block.Hash())
	fc.SafeBlockHash = block.Hash()
	_, err = d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to execute ForkchoiceUpdated: %w", err)
	}
	return block, false, nil

}

// insertHeadBlock creates, executes, and inserts the specified block as the head block.
// It first uses the given FC to start the block creation process and then after the payload is executed,
// sets the FC to the same safe and finalized hashes, but updates the head hash to the new block.
// If updateSafe is true, the head block is considered to be the safe head as well as the head.
func (d *outputImpl) insertHeadBlock(ctx context.Context, fc l2.ForkchoiceState, attrs *l2.PayloadAttributes, updateSafe bool) (*l2.ExecutionPayload, error) {
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
	fc.HeadBlockHash = payload.BlockHash
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	d.log.Debug("Inserted L2 head block", "number", uint64(payload.BlockNumber), "hash", payload.BlockHash, "update_safe", updateSafe)
	_, err = d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err)
	}
	return payload, nil
}
