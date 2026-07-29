package wait

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BlockCaller is a subset of the [ethclient.Client] interface
// encompassing methods that query for block information.
type BlockCaller interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

type OutputAtBlockCaller interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

func ForBlock(ctx context.Context, client BlockCaller, n uint64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			height, err := client.BlockNumber(ctx)
			if err != nil {
				return err
			}
			if height < n {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return nil
		}
	}
}

func ForBlockWithTimestamp(ctx context.Context, client BlockCaller, target uint64) error {
	_, err := AndGet(ctx, time.Second, func() (uint64, error) {
		head, err := client.BlockByNumber(ctx, nil)
		if err != nil {
			return 0, err
		}
		return head.Time(), nil
	}, func(actual uint64) bool {
		return actual >= target
	})
	return err
}

func ForNextBlock(ctx context.Context, client BlockCaller) error {
	current, err := client.BlockNumber(ctx)
	// Long timeout so we don't have to care what the block time is. If the test passes this will complete early anyway.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err != nil {
		return fmt.Errorf("get starting block number: %w", err)
	}
	return ForBlock(ctx, client, current+1)
}

func ForProcessingFullBatch(ctx context.Context, rollupCl dial.RollupClientInterface) error {
	_, err := AndGet(ctx, time.Second, func() (*eth.SyncStatus, error) {
		return rollupCl.SyncStatus(ctx)
	}, func(syncStatus *eth.SyncStatus) bool {
		return syncStatus.PendingSafeL2 == syncStatus.SafeL2
	})
	return err
}

func ForUnsafeBlock(ctx context.Context, rollupCl dial.RollupClientInterface, n uint64) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	_, err := AndGet(ctx, time.Second, func() (*eth.SyncStatus, error) {
		return rollupCl.SyncStatus(ctx)
	}, func(syncStatus *eth.SyncStatus) bool {
		return syncStatus.UnsafeL2.Number >= n
	})
	return err
}

func ForSafeBlock(ctx context.Context, rollupClient dial.RollupClientInterface, n uint64) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	_, err := AndGet(ctx, time.Second, func() (*eth.SyncStatus, error) {
		return rollupClient.SyncStatus(ctx)
	}, func(syncStatus *eth.SyncStatus) bool {
		return syncStatus.SafeL2.Number >= n
	})
	return err
}

func ForOutputAtBlock(ctx context.Context, rollupClient OutputAtBlockCaller, n uint64) (*eth.OutputResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var lastErr error
	output, err := AndGet(ctx, 500*time.Millisecond, func() (*eth.OutputResponse, error) {
		output, err := rollupClient.OutputAtBlock(ctx, n)
		if err != nil {
			lastErr = err
			return nil, nil
		}
		return output, nil
	}, func(output *eth.OutputResponse) bool {
		return output != nil
	})
	if err != nil && lastErr != nil {
		return nil, fmt.Errorf("timed out waiting for L2 output at block %d: %w: %w", n, lastErr, err)
	}
	return output, err
}

func ForOutputAtBlockRPC(ctx context.Context, client *rpc.Client, blockNum any) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var lastErr error
	_, err := AndGet(ctx, 500*time.Millisecond, func() (bool, error) {
		var output eth.OutputResponse
		if err := client.CallContext(ctx, &output, "optimism_outputAtBlock", blockNum); err != nil {
			lastErr = err
			return false, nil
		}
		return true, nil
	}, func(success bool) bool {
		return success
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("timed out waiting for L2 output at block %v: %w: %w", blockNum, lastErr, err)
	}
	return err
}

func ForNextSafeBlock(ctx context.Context, client BlockCaller) (*types.Block, error) {
	safeBlockNumber := big.NewInt(rpc.SafeBlockNumber.Int64())
	var current *types.Block
	var err error
	for {
		current, err = client.BlockByNumber(ctx, safeBlockNumber)
		if err != nil {
			// If block is not found (e.g. upon startup of chain, when there is no "safe block" yet)
			// then it may be found later. Keep wait loop running.
			if strings.Contains(err.Error(), "block not found") {
				continue
			}
			return nil, err
		}
		break
	}

	// Long timeout so we don't have to care what the block time is. If the test passes this will complete early anyway.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			next, err := client.BlockByNumber(ctx, safeBlockNumber)
			if err != nil {
				// If block is not found (e.g. upon startup of chain, when there is no "safe block" yet)
				// then it may be found later. Keep wait loop running.
				if strings.Contains(err.Error(), "block not found") {
					continue
				}
				return nil, err
			}
			if next.NumberU64() > current.NumberU64() {
				return next, nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
