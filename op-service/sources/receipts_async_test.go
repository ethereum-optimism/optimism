package sources

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAsyncReceiptsFetcherPrefetchAndFetchShareRequest(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	fetcher := newAsyncReceiptsFetcher(func(ctx context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
		calls.Add(1)
		close(started)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-release:
			return nil, nil, nil
		}
	})
	defer fetcher.Close()

	hash := common.Hash{0x01}
	fetcher.Prefetch(hash)
	<-started

	req := fetcher.getOrStart(hash)
	require.Same(t, fetcher.requests[hash], req)
	require.Equal(t, int32(1), calls.Load())
	close(release)
	<-req.done
}

func TestAsyncReceiptsFetcherCallerCancellationDoesNotCancelRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	fetcher := newAsyncReceiptsFetcher(func(ctx context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-release:
			return nil, nil, nil
		}
	})
	defer fetcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, _, err := fetcher.Fetch(ctx, common.Hash{0x02})
		result <- err
	}()
	<-started
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)

	req := fetcher.getOrStart(common.Hash{0x02})
	close(release)
	<-req.done
}

func TestAsyncReceiptsFetcherRetriesAfterError(t *testing.T) {
	var calls atomic.Int32
	testErr := errors.New("test error")
	fetcher := newAsyncReceiptsFetcher(func(context.Context, common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
		if calls.Add(1) == 1 {
			return nil, nil, testErr
		}
		return nil, nil, nil
	})
	defer fetcher.Close()

	hash := common.Hash{0x03}
	_, _, err := fetcher.Fetch(context.Background(), hash)
	require.ErrorIs(t, err, testErr)
	_, _, err = fetcher.Fetch(context.Background(), hash)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestAsyncReceiptsFetcherCloseCancelsRequests(t *testing.T) {
	started := make(chan struct{})
	fetcher := newAsyncReceiptsFetcher(func(ctx context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
		close(started)
		<-ctx.Done()
		return nil, nil, ctx.Err()
	})

	fetcher.Prefetch(common.Hash{0x04})
	<-started

	closed := make(chan struct{})
	go func() {
		fetcher.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("fetcher did not close after cancelling its request")
	}

	_, _, err := fetcher.Fetch(context.Background(), common.Hash{0x04})
	require.ErrorIs(t, err, context.Canceled)
}
