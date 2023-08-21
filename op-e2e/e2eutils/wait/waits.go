package wait

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func ForReceiptOK(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return ForReceipt(ctx, client, hash, types.ReceiptStatusSuccessful)
}

func ForReceiptFail(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return ForReceipt(ctx, client, hash, types.ReceiptStatusFailed)
}

func ForReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, status uint64) (*types.Receipt, error) {
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
			return nil, fmt.Errorf("failed to get receipt: %w", err)
		}
		if receipt.Status != status {
			printDebugTrace(ctx, client, hash)
			return receipt, fmt.Errorf("expected status %d, but got %d", status, receipt.Status)
		}
		return receipt, nil
	}
}

type jsonRawString string

func (s *jsonRawString) UnmarshalJSON(input []byte) error {
	str := jsonRawString(input)
	*s = str
	return nil
}

// printDebugTrace logs debug_traceTransaction output to aid in debugging unexpected receipt statuses
func printDebugTrace(ctx context.Context, client *ethclient.Client, txHash common.Hash) {
	var trace jsonRawString
	options := map[string]string{}
	err := client.Client().CallContext(ctx, &trace, "debug_traceTransaction", hexutil.Bytes(txHash.Bytes()), options)
	if err != nil {
		fmt.Printf("TxTrace unavailable: %v\n", err)
		return
	}
	fmt.Printf("TxTrace: %v\n", trace)
}

func ForBlock(ctx context.Context, client *ethclient.Client, n uint64) error {
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

func ForNextBlock(ctx context.Context, client *ethclient.Client) error {
	current, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get starting block number: %w", err)
	}
	return ForBlock(ctx, client, current+1)
}

func For(ctx context.Context, rate time.Duration, cb func() (bool, error)) error {
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

func AndGet[T interface{}](ctx context.Context, pollRate time.Duration, get func() (T, error), predicate func(T) bool) (T, error) {
	tick := time.NewTicker(pollRate)
	defer tick.Stop()

	var nilT T
	for {
		select {
		case <-ctx.Done():
			return nilT, ctx.Err()
		case <-tick.C:
			val, err := get()
			if err != nil {
				return nilT, err
			}
			if predicate(val) {
				return val, nil
			}
		}
	}
}
