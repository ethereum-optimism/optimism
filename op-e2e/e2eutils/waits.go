package e2eutils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func WaitReceiptOK(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, client, hash, types.ReceiptStatusSuccessful)
}

func WaitReceiptFail(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return WaitReceipt(ctx, client, hash, types.ReceiptStatusFailed)
}

func WaitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, status uint64) (*types.Receipt, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if errors.Is(err, ethereum.NotFound) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				continue
			}
		}
		if err != nil {
			return nil, err
		}
		if receipt.Status != status {
			return receipt, fmt.Errorf("expected status %d, but got %d", status, receipt.Status)
		}
		return receipt, nil
	}
}

func WaitBlock(ctx context.Context, client *ethclient.Client, n uint64) error {
	for {
		height, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		if height < n {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	return nil
}

func WaitFor(ctx context.Context, rate time.Duration, cb func() (bool, error)) error {
	tick := time.NewTicker(rate)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			done, err := cb()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}
