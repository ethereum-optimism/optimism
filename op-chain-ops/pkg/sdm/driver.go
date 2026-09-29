package sdm

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// DeployStateBloat deploys StateBloat and returns its successful receipt.
func DeployStateBloat(ctx context.Context, sender *TxSender, gasLimit uint64, pollInterval time.Duration) (*RPCReceipt, error) {
	nonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return nil, fmt.Errorf("pending nonce for deployer %s: %w", sender.From, err)
	}
	bytecode, err := DecodeHexBytes(StateBloatBin)
	if err != nil {
		return nil, err
	}
	tx, err := sender.SendContractCreation(ctx, nonce, bytecode, gasLimit)
	if err != nil {
		return nil, err
	}
	receipt, err := WaitRPCReceipt(ctx, sender.RPC, tx.Hash(), pollInterval)
	if err != nil {
		return nil, fmt.Errorf("StateBloat deploy from %s nonce %d: %w", sender.From, nonce, err)
	}
	if uint64(receipt.Status) != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("StateBloat deploy tx %s failed with status %d", tx.Hash(), receipt.Status)
	}
	if receipt.ContractAddress == nil {
		return nil, fmt.Errorf("StateBloat deploy tx %s receipt missing contractAddress", tx.Hash())
	}
	return receipt, nil
}

// SubmitWorkload sends count StateBloat.run(slotCount) calls from the pending
// nonce, returning every transaction sent before any error.
func SubmitWorkload(
	ctx context.Context,
	sender *TxSender,
	contract common.Address,
	count int,
	slotCount uint64,
	gasLimit uint64,
	spacing time.Duration,
) ([]*types.Transaction, error) {
	startNonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return nil, fmt.Errorf("pending nonce: %w", err)
	}
	calldata := EncodeRun(slotCount)
	txs := make([]*types.Transaction, 0, count)
	for i := 0; i < count; i++ {
		if i > 0 && spacing > 0 {
			timer := time.NewTimer(spacing)
			select {
			case <-ctx.Done():
				timer.Stop()
				return txs, ctx.Err()
			case <-timer.C:
			}
		}
		tx, err := sender.SendCall(ctx, startNonce+uint64(i), contract, calldata, gasLimit)
		if err != nil {
			return txs, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

// CollectWorkloadReceipts waits for each successful receipt, grouped by block.
func CollectWorkloadReceipts(
	ctx context.Context,
	sender *TxSender,
	txs []*types.Transaction,
	receiptTimeout time.Duration,
	pollInterval time.Duration,
) (map[uint64][]*RPCReceipt, error) {
	byBlock := make(map[uint64][]*RPCReceipt)
	for i, tx := range txs {
		receiptCtx, cancel := context.WithTimeout(ctx, receiptTimeout)
		receipt, err := WaitRPCReceipt(receiptCtx, sender.RPC, tx.Hash(), pollInterval)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("tx %d %s receipt: %w", i, tx.Hash(), err)
		}
		if uint64(receipt.Status) != types.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("tx %d %s failed with status %d", i, tx.Hash(), receipt.Status)
		}
		blockNum := receipt.BlockNum()
		byBlock[blockNum] = append(byBlock[blockNum], receipt)
	}
	return byBlock, nil
}

// CountByBlock returns how many receipts landed in each block.
func CountByBlock(byBlock map[uint64][]*RPCReceipt) map[uint64]int {
	counts := make(map[uint64]int, len(byBlock))
	for blockNum, receipts := range byBlock {
		counts[blockNum] = len(receipts)
	}
	return counts
}
