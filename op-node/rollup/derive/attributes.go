package derive

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// L1ReceiptsFetcher fetches L1 header info and receipts for the payload attributes derivation (the info tx and deposits)
type L1ReceiptsFetcher interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.L1Info, error)
	Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error)
}

// PreparePayloadAttributes prepares a PayloadAttributes template that is ready to build a L2 block with deposits only, on top of the given l2Parent, with the given epoch as L1 origin.
// The template defaults to NoTxPool=true, and no sequencer transactions: the caller has to modify the template to add transactions,
// by setting NoTxPool=false as sequencer, or by appending batch transactions as verifier.
// The severity of the error is returned; a crit=false error means there was a temporary issue, like a failed RPC or time-out.
// A crit=true error means the input arguments are inconsistent or invalid.
func PreparePayloadAttributes(ctx context.Context, cfg *rollup.Config, dl L1ReceiptsFetcher, l2Parent eth.L2BlockRef, timestamp uint64, epoch eth.BlockID) (attrs *eth.PayloadAttributes, err error) {
	var l1Info eth.L1Info
	var depositTxs []hexutil.Bytes
	var seqNumber uint64

	// If the L1 origin changed this block, then we are in the first block of the epoch. In this
	// case we need to fetch all transaction receipts from the L1 origin block so we can scan for
	// user deposits.
	if l2Parent.L1Origin.Number != epoch.Number {
		info, _, receipts, err := dl.Fetch(ctx, epoch.Hash)
		if err != nil {
			return nil, makeError(
				ErrFetchFailed,
				"failed to fetch L1 block info and receipts",
				ErrTemporary)
		}
		if l2Parent.L1Origin.Hash != info.ParentHash() {
			return nil, makeError(
				ErrL1OriginMismatch,
				fmt.Sprintf("cannot create new block with L1 origin %s (parent %s) on top of L1 origin %s",
					epoch, info.ParentHash(), l2Parent.L1Origin),
				ErrCritical)
		}
		deposits, err := DeriveDeposits(receipts, cfg.DepositContractAddress)
		if err != nil {
			return nil, makeError(
				ErrDeriveFailed,
				fmt.Sprintf("failed to derive some deposits: %v", err),
				ErrCritical)
		}
		l1Info = info
		depositTxs = deposits
		seqNumber = 0
	} else {
		if l2Parent.L1Origin.Hash != epoch.Hash {
			return nil, makeError(
				ErrEpochHashMismatch,
				fmt.Sprintf("cannot create new block with L1 origin %s in conflict with L1 origin %s",
					epoch, l2Parent.L1Origin),
				ErrCritical)
		}
		info, err := dl.InfoByHash(ctx, epoch.Hash)
		if err != nil {
			return nil, makeError(
				ErrInfoByHashFailed,
				fmt.Sprintf("failed to fetch L1 block info: %v", err),
				ErrTemporary)
		}
		l1Info = info
		depositTxs = nil
		seqNumber = l2Parent.SequenceNumber + 1
	}

	l1InfoTx, err := L1InfoDepositBytes(seqNumber, l1Info)
	if err != nil {
		return nil, makeError(
			ErrL1InfoTxFailed,
			fmt.Sprintf("failed to create l1InfoTx: %v", err),
			ErrCritical)
	}

	txs := make([]hexutil.Bytes, 0, 1+len(depositTxs))
	txs = append(txs, l1InfoTx)
	txs = append(txs, depositTxs...)

	return &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(timestamp),
		PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: cfg.FeeRecipientAddress,
		Transactions:          txs,
		NoTxPool:              true,
	}, nil
}
