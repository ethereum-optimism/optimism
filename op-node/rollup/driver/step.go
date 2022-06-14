package driver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

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

// isDepositTx checks an opaqueTx to determine if it is a Deposit Trransaction
// It has to return an error in the case the transaction is empty
func isDepositTx(opaqueTx eth.Data) (bool, error) {
	if len(opaqueTx) == 0 {
		return false, errors.New("empty transaction")
	}
	return opaqueTx[0] == types.DepositTxType, nil
}

// lastDeposit finds the index of last deposit at the start of the transactions.
// It walks the transactions from the start until it finds a non-deposit tx.
// An error is returned if any looked at transaction cannot be decoded
func lastDeposit(txns []eth.Data) (int, error) {
	var lastDeposit int
	for i, tx := range txns {
		deposit, err := isDepositTx(tx)
		if err != nil {
			return 0, fmt.Errorf("invalid transaction at idx %d", i)
		}
		if deposit {
			lastDeposit = i
		} else {
			break
		}
	}
	return lastDeposit, nil
}

func (d *outputImpl) processBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, payload *eth.ExecutionPayload) error {
	d.log.Info("processing new block", "parent", payload.ParentID(), "l2Head", l2Head, "id", payload.ID())
	if err := d.l2.NewPayload(ctx, payload); err != nil {
		return fmt.Errorf("failed to insert new payload: %v", err)
	}
	// now try to persist a reorg to the new payload
	fc := eth.ForkchoiceState{
		HeadBlockHash:      payload.BlockHash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	res, err := d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return fmt.Errorf("failed to update forkchoice to point to new payload: %v", err)
	}
	if res.PayloadStatus.Status != eth.ExecutionValid {
		return fmt.Errorf("failed to persist forkchoice update: %v", err)
	}
	return nil
}

func (d *outputImpl) createNewBlock(ctx context.Context, l2Head eth.L2BlockRef, l2SafeHead eth.BlockID, l2Finalized eth.BlockID, l1Origin eth.L1BlockRef) (eth.L2BlockRef, *eth.ExecutionPayload, error) {
	d.log.Info("creating new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var l1Info derive.L1Info
	var receipts types.Receipts
	var err error

	seqNumber := l2Head.SequenceNumber + 1

	// If the L1 origin changed this block, then we are in the first block of the epoch. In this
	// case we need to fetch all transaction receipts from the L1 origin block so we can scan for
	// user deposits.
	if l2Head.L1Origin.Number != l1Origin.Number {
		l1Info, _, receipts, err = d.dl.Fetch(fetchCtx, l1Origin.Hash)
		seqNumber = 0 // reset sequence number at the start of the epoch
	} else {
		l1Info, err = d.dl.InfoByHash(fetchCtx, l1Origin.Hash)
	}
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to fetch L1 block info of %s: %v", l1Origin, err)
	}

	// Start building the list of transactions to include in the new block.
	var txns []eth.Data

	// First transaction in every block is always the L1 info transaction.
	l1InfoTx, err := derive.L1InfoDepositBytes(seqNumber, l1Info)
	if err != nil {
		return l2Head, nil, err
	}
	txns = append(txns, l1InfoTx)

	// Next we append user deposits. If we're not the first block in an epoch, then receipts will
	// be empty and no deposits will be derived.
	deposits, errs := derive.DeriveDeposits(receipts, d.Config.DepositContractAddress)
	d.log.Info("Derived deposits", "deposits", deposits, "l2Parent", l2Head, "l1Origin", l1Origin)
	for _, err := range errs {
		d.log.Error("Failed to derive a deposit", "l1OriginHash", l1Origin.Hash, "err", err)
	}
	// TODO: Should we halt if len(errs) > 0? Opens up a denial of service attack, but prevents lockup of funds.
	txns = append(txns, deposits...)

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	nextL2Time := l2Head.Time + d.Config.BlockTime
	shouldProduceEmptyBlock := nextL2Time >= l1Origin.Time+d.Config.MaxSequencerDrift

	// Put together our payload attributes.
	attrs := &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(nextL2Time),
		PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: d.Config.FeeRecipientAddress,
		Transactions:          txns,
		NoTxPool:              shouldProduceEmptyBlock,
	}

	// And construct our fork choice state. This is our current fork choice state and will be
	// updated as a result of executing the block based on the attributes described above.
	fc := eth.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}

	// Actually execute the block and add it to the head of the chain.
	payload, err := d.insertHeadBlock(ctx, fc, attrs, false)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to extend L2 chain: %v", err)
	}

	// Generate an L2 block ref from the payload.
	ref, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)

	return ref, payload, err
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

	d.log.Debug("inserting epoch", "input_l1_first", l1Input[0], "input_l1_last", l1Input[len(l1Input)-1], "input_l2_parent", l2SafeHead, "finalized_l2", l2Finalized)

	// Get inputs from L1 and L2
	epoch := rollup.Epoch(l1Input[0].Number)
	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	l2Info, err := d.l2.PayloadByHash(fetchCtx, l2SafeHead.Hash)
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
	deposits, errs := derive.DeriveDeposits(receipts, d.Config.DepositContractAddress)
	for _, err := range errs {
		d.log.Error("Failed to derive a deposit", "l1OriginHash", l1Input[0].Hash, "err", err)
	}
	// TODO: Should we halt if len(errs) > 0? Opens up a denial of service attack, but prevents lockup of funds.
	// TODO: with sharding the blobs may be identified in more detail than L1 block hashes
	transactions, err := d.dl.FetchAllTransactions(fetchCtx, l1Input)
	if err != nil {
		return l2Head, l2SafeHead, false, fmt.Errorf("failed to fetch transactions from %s: %v", l1Input, err)
	}
	batches, errs := derive.BatchesFromEVMTransactions(&d.Config, transactions)
	// Some input to derive.BatchesFromEVMTransactions may be invalid and produce errors.
	// We log the errors, but keep going as this process is designed to be resilient to these errors
	// and we have defaults in case no valid (or partial) batches were submitted.
	for i, err := range errs {
		d.log.Error("Failed to decode batch", "err_idx", i, "err", err)
	}

	// Make batches contiguous
	minL2Time := uint64(l2Info.Timestamp) + d.Config.BlockTime
	maxL2Time := l1Info.Time() + d.Config.MaxSequencerDrift
	if minL2Time+d.Config.BlockTime > maxL2Time {
		maxL2Time = minL2Time + d.Config.BlockTime
	}
	batches = derive.FilterBatches(&d.Config, epoch, minL2Time, maxL2Time, batches)
	batches = derive.FillMissingBatches(batches, uint64(epoch), d.Config.BlockTime, minL2Time, nextL1Block.Time())

	fc := eth.ForkchoiceState{
		HeadBlockHash:      l2Head.Hash,
		SafeBlockHash:      l2SafeHead.Hash,
		FinalizedBlockHash: l2Finalized.Hash,
	}
	// Execute each L2 block in the epoch
	lastHead := l2Head
	lastSafeHead := l2SafeHead
	didReorg := false
	var payload *eth.ExecutionPayload
	var reorg bool
	for i, batch := range batches {
		var txns []eth.Data
		l1InfoTx, err := derive.L1InfoDepositBytes(uint64(i), l1Info)
		if err != nil {
			return l2Head, l2SafeHead, false, fmt.Errorf("failed to create l1InfoTx: %w", err)
		}
		txns = append(txns, l1InfoTx)
		if i == 0 {
			txns = append(txns, deposits...)
		}
		txns = append(txns, batch.Transactions...)
		attrs := &eth.PayloadAttributes{
			Timestamp:             hexutil.Uint64(batch.Timestamp),
			PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
			SuggestedFeeRecipient: d.Config.FeeRecipientAddress,
			Transactions:          txns,
			// we are verifying, not sequencing, we've got all transactions and do not pull from the tx-pool
			// (that would make the block derivation non-deterministic)
			NoTxPool: true,
		}

		d.log.Debug("inserting epoch batch", "safeHeadL1Origin", lastSafeHead.L1Origin, "l1Info", l1Info.ID(), "seqnr", i)

		// We are either verifying blocks (with a potential for a reorg) or inserting a safe head to the chain
		if lastHead.Hash != lastSafeHead.Hash {
			d.log.Debug("verifying derived attributes matches L2 block",
				"lastHead", lastHead, "lastSafeHead", lastSafeHead, "epoch", epoch,
				"lastSafeHead_l1origin", lastSafeHead.L1Origin, "lastHead_l1origin", lastHead.L1Origin)
			payload, reorg, err = d.verifySafeBlock(ctx, fc, attrs, lastSafeHead.ID())

		} else {
			d.log.Debug("inserting new batch after lastHead", "lastHead", lastHead.ID())
			payload, err = d.insertHeadBlock(ctx, fc, attrs, true)
		}
		if err != nil {
			return lastHead, lastSafeHead, didReorg, fmt.Errorf("failed to extend L2 chain at block %d/%d of epoch %d: %w", i, len(batches), epoch, err)
		}

		newLast, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)
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
func attributesMatchBlock(attrs *eth.PayloadAttributes, parentHash common.Hash, block *eth.ExecutionPayload) error {
	if parentHash != block.ParentHash {
		return fmt.Errorf("parent hash field does not match. expected: %v. got: %v", parentHash, block.ParentHash)
	}
	if attrs.Timestamp != block.Timestamp {
		return fmt.Errorf("timestamp field does not match. expected: %v. got: %v", uint64(attrs.Timestamp), block.Timestamp)
	}
	if attrs.PrevRandao != block.PrevRandao {
		return fmt.Errorf("random field does not match. expected: %v. got: %v", attrs.PrevRandao, block.PrevRandao)
	}
	if len(attrs.Transactions) != len(block.Transactions) {
		return fmt.Errorf("transaction count does not match. expected: %v. got: %v", len(attrs.Transactions), block.Transactions)
	}
	for i, otx := range attrs.Transactions {
		if expect := block.Transactions[i]; !bytes.Equal(otx, expect) {
			return fmt.Errorf("transaction %d does not match. expected: %v. got: %v", i, expect, otx)
		}
	}
	return nil
}

// verifySafeBlock reconciles the supplied payload attributes against the actual L2 block.
// If they do not match, it inserts the new block and sets the head and safe head to the new block in the FC.
func (d *outputImpl) verifySafeBlock(ctx context.Context, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, parent eth.BlockID) (*eth.ExecutionPayload, bool, error) {
	payload, err := d.l2.PayloadByNumber(ctx, new(big.Int).SetUint64(parent.Number+1))
	if err != nil {
		return nil, false, fmt.Errorf("failed to get L2 block: %w", err)
	}
	ref, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse block ref: %w", err)
	}
	d.log.Debug("verifySafeBlock", "parentl2", parent, "payload", payload.ID(), "payloadOrigin", ref.L1Origin, "payloadSeq", ref.SequenceNumber)
	err = attributesMatchBlock(attrs, parent.Hash, payload)
	if err != nil {
		// Have reorg
		d.log.Warn("Detected L2 reorg when verifying L2 safe head", "parent", parent, "prev_block", payload.BlockHash, "mismatch", err)
		fc.HeadBlockHash = parent.Hash
		fc.SafeBlockHash = parent.Hash
		payload, err := d.insertHeadBlock(ctx, fc, attrs, true)
		return payload, true, err
	}
	// If the attributes match, just bump the safe head
	d.log.Debug("Verified L2 block", "number", payload.BlockNumber, "hash", payload.BlockHash)
	fc.SafeBlockHash = payload.BlockHash
	_, err = d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to execute ForkchoiceUpdated: %w", err)
	}
	return payload, false, nil
}

// insertHeadBlock creates, executes, and inserts the specified block as the head block.
// It first uses the given FC to start the block creation process and then after the payload is executed,
// sets the FC to the same safe and finalized hashes, but updates the head hash to the new block.
// If updateSafe is true, the head block is considered to be the safe head as well as the head.
// It returns the payload, the count of deposits, and an error.
func (d *outputImpl) insertHeadBlock(ctx context.Context, fc eth.ForkchoiceState, attrs *eth.PayloadAttributes, updateSafe bool) (*eth.ExecutionPayload, error) {
	fcRes, err := d.l2.ForkchoiceUpdate(ctx, &fc, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new block via forkchoice: %w", err)
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("engine not ready, forkchoice pre-state is not valid: %s", fcRes.PayloadStatus.Status)
	}
	id := fcRes.PayloadID
	if id == nil {
		return nil, errors.New("nil id in forkchoice result when expecting a valid ID")
	}
	payload, err := d.l2.GetPayload(ctx, *id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution payload: %w", err)
	}
	// Sanity check payload before inserting it
	if len(payload.Transactions) == 0 {
		return nil, errors.New("no transactions in returned payload")
	}
	if payload.Transactions[0][0] != types.DepositTxType {
		return nil, fmt.Errorf("first transaction was not deposit tx. Got %v", payload.Transactions[0][0])
	}
	// Ensure that the deposits are first
	lastDeposit, err := lastDeposit(payload.Transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to find last deposit: %w", err)
	}
	// Ensure no deposits after last deposit
	for i := lastDeposit + 1; i < len(payload.Transactions); i++ {
		tx := payload.Transactions[i]
		deposit, err := isDepositTx(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction idx %d: %w", i, err)
		}
		if deposit {
			d.log.Error("Produced an invalid block where the deposit txns are not all at the start of the block", "tx_idx", i, "lastDeposit", lastDeposit)
			return nil, fmt.Errorf("deposit tx (%d) after other tx in l2 block with prev deposit at idx %d", i, lastDeposit)
		}
	}
	// If this is an unsafe block, it has deposits & transactions included from L2.
	// Record if the execution engine dropped deposits. The verification process would see a mismatch
	// between attributes and the block, but then execute the correct block.
	if !updateSafe && lastDeposit+1 != len(attrs.Transactions) {
		d.log.Error("Dropped deposits when executing L2 block")
	}

	err = d.l2.NewPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to insert execution payload: %w", err)
	}
	fc.HeadBlockHash = payload.BlockHash
	if updateSafe {
		fc.SafeBlockHash = payload.BlockHash
	}
	d.log.Debug("Inserted L2 head block", "number", uint64(payload.BlockNumber), "hash", payload.BlockHash, "update_safe", updateSafe)
	fcRes, err = d.l2.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make the new L2 block canonical via forkchoice: %w", err)
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("failed to persist forkchoice change: %s", fcRes.PayloadStatus.Status)
	}
	return payload, nil
}
