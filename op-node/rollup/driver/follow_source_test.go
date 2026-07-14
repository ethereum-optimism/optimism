package driver

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type blockingL1FollowSource struct {
	mu      sync.Mutex
	calls   map[uint64]int
	started chan uint64
	release <-chan struct{}
}

func (s *blockingL1FollowSource) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	s.mu.Lock()
	s.calls[number]++
	s.mu.Unlock()

	s.started <- number
	select {
	case <-s.release:
		return eth.L1BlockRef{Number: number, Hash: common.Hash{byte(number)}}, nil
	case <-ctx.Done():
		return eth.L1BlockRef{}, ctx.Err()
	}
}

func TestFetchL1BlockRefsDeduplicatesAndFetchesConcurrently(t *testing.T) {
	release := make(chan struct{})
	source := &blockingL1FollowSource{
		calls:   make(map[uint64]int),
		started: make(chan uint64, 3),
		release: release,
	}

	type result struct {
		refs map[uint64]eth.L1BlockRef
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		refs, err := fetchL1BlockRefs(context.Background(), source, 1, 2, 1, 3, 2)
		resultCh <- result{refs: refs, err: err}
	}()

	started := make(map[uint64]struct{})
	for range 3 {
		select {
		case number := <-source.started:
			started[number] = struct{}{}
		case <-time.After(time.Second):
			close(release)
			t.Fatal("unique L1 lookups did not start concurrently")
		}
	}
	close(release)

	res := <-resultCh
	require.NoError(t, res.err)
	require.Equal(t, map[uint64]struct{}{1: {}, 2: {}, 3: {}}, started)
	require.Equal(t, eth.L1BlockRef{Number: 1, Hash: common.Hash{1}}, res.refs[1])
	require.Equal(t, eth.L1BlockRef{Number: 2, Hash: common.Hash{2}}, res.refs[2])
	require.Equal(t, eth.L1BlockRef{Number: 3, Hash: common.Hash{3}}, res.refs[3])

	source.mu.Lock()
	defer source.mu.Unlock()
	require.Equal(t, map[uint64]int{1: 1, 2: 1, 3: 1}, source.calls)
}

func TestFetchL1BlockRefsHonorsContext(t *testing.T) {
	source := &blockingL1FollowSource{
		calls:   make(map[uint64]int),
		started: make(chan uint64, 2),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := fetchL1BlockRefs(ctx, source, 1, 2)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
