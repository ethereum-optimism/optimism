package sources

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type receiptsProviderFunc func(context.Context, eth.BlockInfo, []common.Hash) (optypes.Receipts, error)

func (f receiptsProviderFunc) FetchReceipts(ctx context.Context, block eth.BlockInfo, hashes []common.Hash) (optypes.Receipts, error) {
	return f(ctx, block, hashes)
}

func TestCachingReceiptsProvider_CanceledWaiters(t *testing.T) {
	for _, n := range []int{1, 10, 100} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			block, receipts := randomRpcBlockAndReceipts(rand.New(rand.NewSource(69)), 4)
			want := optypes.FromGethReceipts(receipts)
			entered := make(chan struct{})
			release := make(chan struct{})
			var calls atomic.Int32
			mrpc := new(mockRPC)
			mrpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByHash", []any{block.Hash, true}).
				Run(func(args mock.Arguments) {
					*args[1].(**RPCBlock) = block
				}).Return([]error{nil}).Once()
			mrpc.On("CallContext", context.Background(), mock.Anything, "eth_getBlockReceipts", []any{block.Hash}).Run(func(args mock.Arguments) {
				if calls.Add(1) == 1 {
					close(entered)
				}
				<-release
				*args[1].(*optypes.Receipts) = want
			}).Return([]error{nil}).Once()
			client := newEthClientWithCaches(nil, 1)
			client.client = mrpc
			client.trustRPC = true
			client.recProvider = NewCachingRPCReceiptsProvider(mrpc, nil, RPCReceiptsConfig{
				ProviderKind:        RPCKindStandard,
				MethodResetDuration: time.Minute,
			}, nil, 1)
			ownerDone := make(chan error, 1)
			go func() {
				_, _, err := client.FetchReceipts(context.Background(), block.Hash)
				ownerDone <- err
			}()
			<-entered
			waiterDone := make(chan error, n)
			// Always release the owner and join every goroutine, including when
			// testing an implementation that cannot cancel its lock wait.
			completed := 0
			t.Cleanup(func() {
				close(release)
				require.NoError(t, <-ownerDone)
				for ; completed < n; completed++ {
					<-waiterDone
				}
				// The successful owner still populates the cache. Cache hits keep
				// their existing behavior, even with a canceled context.
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, got, err := client.FetchReceipts(ctx, block.Hash)
				require.NoError(t, err)
				require.Equal(t, want, got)
				require.EqualValues(t, 1, calls.Load())
				mrpc.AssertExpectations(t)
			})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			for i := 0; i < n; i++ {
				go func() {
					_, _, err := client.FetchReceipts(ctx, block.Hash)
					waiterDone <- err
				}()
			}
			timeout := time.NewTimer(time.Second)
			defer timeout.Stop()
			for completed < n {
				select {
				case err := <-waiterDone:
					completed++
					require.ErrorIs(t, err, context.Canceled)
				case <-timeout.C:
					t.Fatal("canceled receipt waiters are blocked by another caller's RPC")
				}
			}
			require.EqualValues(t, 1, calls.Load(), "canceling waiters must not issue additional RPCs")
		})
	}
}

func TestCachingReceiptsProvider_WaitDeadline(t *testing.T) {
	block, receipts := randomRpcBlockAndReceipts(rand.New(rand.NewSource(69)), 4)
	info, _, err := block.Info(true, true)
	require.NoError(t, err)
	hashes := receiptTxHashes(receipts)
	provider := NewCachingReceiptsProvider(receiptsProviderFunc(func(context.Context, eth.BlockInfo, []common.Hash) (optypes.Receipts, error) {
		t.Error("a waiting request must not enter the inner provider")
		return nil, nil
	}), nil, 1)
	// Keep the block's fetching lock held throughout the deadline. A timeout
	// must release the caller without unlocking a different caller's work.
	lock := provider.getOrCreateFetchingLock(block.BlockID().Hash)
	lock.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	completed := false
	defer func() {
		lock.Unlock()
		if !completed {
			<-done
		}
	}()
	go func() {
		_, err := provider.FetchReceipts(ctx, info, hashes)
		done <- err
	}()
	select {
	case err := <-done:
		completed = true
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("receipt lock wait ignored its deadline")
	}
}

func TestCachingReceiptsProvider_RetryAfterError(t *testing.T) {
	block, receipts := randomRpcBlockAndReceipts(rand.New(rand.NewSource(69)), 4)
	info, _, err := block.Info(true, true)
	require.NoError(t, err)
	hashes := receiptTxHashes(receipts)
	want := optypes.FromGethReceipts(receipts)
	fetchErr := errors.New("receipt RPC unavailable")
	calls := 0
	provider := NewCachingReceiptsProvider(receiptsProviderFunc(func(context.Context, eth.BlockInfo, []common.Hash) (optypes.Receipts, error) {
		calls++
		if calls == 1 {
			return nil, fetchErr
		}
		return want, nil
	}), nil, 1)
	got, err := provider.FetchReceipts(context.Background(), info, hashes)
	require.ErrorIs(t, err, fetchErr)
	require.Nil(t, got)
	got, err = provider.FetchReceipts(context.Background(), info, hashes)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, 2, calls)
}
