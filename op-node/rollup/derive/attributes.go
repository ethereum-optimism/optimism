package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// L1ReceiptsFetcher fetches L1 header info and receipts for the payload attributes derivation (the info tx and deposits)
type L1ReceiptsFetcher interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type SystemConfigL2Fetcher interface {
	SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error)
}

// PreparePayloadAttributes prepares a PayloadAttributes template that is ready to build a L2 block with deposits only, on top of the given l2Parent, with the given epoch as L1 origin.
// The template defaults to NoTxPool=true, and no sequencer transactions: the caller has to modify the template to add transactions,
// by setting NoTxPool=false as sequencer, or by appending batch transactions as verifier.
// The severity of the error is returned; a crit=false error means there was a temporary issue, like a failed RPC or time-out.
// A crit=true error means the input arguments are inconsistent or invalid.
func PreparePayloadAttributes(ctx context.Context, cfg *rollup.Config, dl L1ReceiptsFetcher, l2 SystemConfigL2Fetcher, l2Parent eth.L2BlockRef, timestamp uint64, epoch eth.BlockID) (attrs *eth.PayloadAttributes, err error) {
	var l1Info eth.BlockInfo
	var depositTxs []hexutil.Bytes
	var seqNumber uint64

	sysConfig, err := l2.SystemConfigByL2Hash(ctx, l2Parent.Hash)
	if err != nil {
		return nil, NewTemporaryError(fmt.Errorf("failed to retrieve L2 parent block: %w", err))
	}

	// If the L1 origin changed this block, then we are in the first block of the epoch. In this
	// case we need to fetch all transaction receipts from the L1 origin block so we can scan for
	// user deposits.
	if l2Parent.L1Origin.Number != epoch.Number {
		info, receipts, err := dl.FetchReceipts(ctx, epoch.Hash)
		if err != nil {
			return nil, NewTemporaryError(fmt.Errorf("failed to fetch L1 block info and receipts: %w", err))
		}
		if l2Parent.L1Origin.Hash != info.ParentHash() {
			return nil, NewResetError(
				fmt.Errorf("cannot create new block with L1 origin %s (parent %s) on top of L1 origin %s",
					epoch, info.ParentHash(), l2Parent.L1Origin))
		}

		deposits, err := DeriveDeposits(receipts, cfg.DepositContractAddress)
		if err != nil {
			// deposits may never be ignored. Failing to process them is a critical error.
			return nil, NewCriticalError(fmt.Errorf("failed to derive some deposits: %w", err))
		}
		// apply sysCfg changes
		if err := UpdateSystemConfigWithL1Receipts(&sysConfig, receipts, cfg); err != nil {
			return nil, NewCriticalError(fmt.Errorf("failed to apply derived L1 sysCfg updates: %w", err))
		}

		l1Info = info
		depositTxs = deposits
		seqNumber = 0
	} else {
		if l2Parent.L1Origin.Hash != epoch.Hash {
			return nil, NewResetError(fmt.Errorf("cannot create new block with L1 origin %s in conflict with L1 origin %s", epoch, l2Parent.L1Origin))
		}
		info, err := dl.InfoByHash(ctx, epoch.Hash)
		if err != nil {
			return nil, NewTemporaryError(fmt.Errorf("failed to fetch L1 block info: %w", err))
		}
		l1Info = info
		depositTxs = nil
		seqNumber = l2Parent.SequenceNumber + 1
	}

	l1InfoTx, err := L1InfoDepositBytes(seqNumber, l1Info, sysConfig)
	if err != nil {
		return nil, NewCriticalError(fmt.Errorf("failed to create l1InfoTx: %w", err))
	}

	txs := make([]hexutil.Bytes, 0, 1+len(depositTxs))
	txs = append(txs, l1InfoTx)
	txs = append(txs, depositTxs...)

	return &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(timestamp),
		PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: predeploys.SequencerFeeVaultAddr,
		Transactions:          txs,
		NoTxPool:              true,
		GasLimit:              (*eth.Uint64Quantity)(&sysConfig.GasLimit),
	}, nil
}
