package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

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
	payload, err := derive.InsertHeadBlock(ctx, d.log, d.l2, fc, attrs, false)
	if err != nil {
		return l2Head, nil, fmt.Errorf("failed to extend L2 chain: %v", err)
	}

	// Generate an L2 block ref from the payload.
	ref, err := derive.PayloadToBlockRef(payload, &d.Config.Genesis)

	return ref, payload, err
}
