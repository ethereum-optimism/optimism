package wait

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func ForBalanceChange(ctx context.Context, client *ethclient.Client, address common.Address, initial *big.Int) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return AndGet[*big.Int](
		ctx,
		100*time.Millisecond,
		func() (*big.Int, error) {
			return client.BalanceAt(ctx, address, nil)
		},
		func(b *big.Int) bool {
			return b.Cmp(initial) != 0
		},
	)
}

func ForReceiptOK(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return ForReceipt(ctx, client, hash, types.ReceiptStatusSuccessful)
}

func ForReceiptFail(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	return ForReceipt(ctx, client, hash, types.ReceiptStatusFailed)
}

func ForReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, status uint64) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
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
		if errors.Is(err, os.ErrDeadlineExceeded) {
			continue
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
	options := map[string]any{
		"enableReturnData": true,
		"tracer":           "callTracer",
		"tracerConfig":     map[string]any{},
	}
	err := client.Client().CallContext(ctx, &trace, "debug_traceTransaction", hexutil.Bytes(txHash.Bytes()), options)
	if err != nil {
		fmt.Printf("TxTrace unavailable: %v\n", err)
		return
	}
	fmt.Printf("TxTrace: %v\n", trace)
}

func For(ctx context.Context, rate time.Duration, cb func() (bool, error)) error {
	tick := time.NewTicker(rate)
	defer tick.Stop()

	for {
		// Perform the first check before any waiting.
		done, err := cb()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			// Allow loop to continue for next retry
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
