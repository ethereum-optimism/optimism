package geth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// errTimeout represents a timeout
	errTimeout = errors.New("timeout")
)

func WaitForL1OriginOnL2(l1BlockNum uint64, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
	timeoutCh := time.After(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	headChan := make(chan *types.Header, 100)
	headSub, err := client.SubscribeNewHead(ctx, headChan)
	if err != nil {
		return nil, err
	}
	defer headSub.Unsubscribe()

	for {
		select {
		case head := <-headChan:
			block, err := client.BlockByNumber(ctx, head.Number)
			if err != nil {
				return nil, err
			}
			l1Info, err := derive.L1InfoDepositTxData(block.Transactions()[0].Data())
			if err != nil {
				return nil, err
			}
			if l1Info.Number >= l1BlockNum {
				return block, nil
			}

		case err := <-headSub.Err():
			return nil, fmt.Errorf("error in head subscription: %w", err)
		case <-timeoutCh:
			return nil, errTimeout
		}
	}
}

func WaitForTransaction(hash common.Hash, client *ethclient.Client, timeout time.Duration) (*types.Receipt, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if receipt != nil && err == nil {
			return receipt, nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			tip, err := client.BlockByNumber(context.Background(), nil)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("receipt for transaction %s not found. tip block number is %d: %w", hash.Hex(), tip.NumberU64(), errTimeout)
		case <-ticker.C:
		}
	}
}

func WaitForBlock(number *big.Int, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	headChan := make(chan *types.Header, 100)
	headSub, err := client.SubscribeNewHead(ctx, headChan)
	if err != nil {
		return nil, err
	}
	defer headSub.Unsubscribe()

	for {
		select {
		case head := <-headChan:
			if head.Number.Cmp(number) >= 0 {
				return client.BlockByNumber(ctx, number)
			}
		case err := <-headSub.Err():
			return nil, fmt.Errorf("error in head subscription: %w", err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func WaitForBlockToBeFinalized(number *big.Int, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
	return waitForBlockTag(number, client, timeout, rpc.FinalizedBlockNumber)
}

func WaitForBlockToBeSafe(number *big.Int, client *ethclient.Client, timeout time.Duration) (*types.Block, error) {
	return waitForBlockTag(number, client, timeout, rpc.SafeBlockNumber)
}

// waitForBlockTag polls for a block number to reach the specified tag & then returns that block at the number.
func waitForBlockTag(number *big.Int, client *ethclient.Client, timeout time.Duration, tag rpc.BlockNumber) (*types.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Wait for it to be finalized. Poll every half second.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	tagBigInt := big.NewInt(tag.Int64())

	for {
		select {
		case <-ticker.C:
			block, err := client.BlockByNumber(ctx, tagBigInt)
			if err != nil {
				return nil, err
			}
			if block != nil && block.NumberU64() >= number.Uint64() {
				return client.BlockByNumber(ctx, number)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
