package wait

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockCaller is a subset of the [ethclient.Client] interface
// encompassing methods that query for block information.
type BlockCaller interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

func ForBlock(ctx context.Context, client BlockCaller, n uint64) error {
	for {
		if ctx.Done() != nil {
			return ctx.Err()
		}
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

func ForProcessingFullBatch(ctx context.Context, rollupCl *sources.RollupClient) error {
	_, err := AndGet(ctx, time.Second, func() (*eth.SyncStatus, error) {
		return rollupCl.SyncStatus(ctx)
	}, func(syncStatus *eth.SyncStatus) bool {
		return syncStatus.PendingSafeL2 == syncStatus.SafeL2
	})
	return err
}
