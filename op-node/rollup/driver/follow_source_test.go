package driver

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type blockingL1FollowSource struct {
	mu      sync.Mutex
	calls   map[uint64]int
	started chan uint64
	release <-chan struct{}
}

type blockingUpstreamFollowSource struct {
	*blockingL1FollowSource
	status        *sources.FollowStatus
	statusStarted chan struct{}
	statusRelease <-chan struct{}
}

func (s *blockingUpstreamFollowSource) GetFollowStatus(ctx context.Context) (*sources.FollowStatus, error) {
	s.statusStarted <- struct{}{}
	select {
	case <-s.statusRelease:
		return s.status, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
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

func TestStartFollowUpstreamFetchDoesNotBlock(t *testing.T) {
	statusRelease := make(chan struct{})
	l1Release := make(chan struct{})
	close(l1Release)
	origin := eth.BlockID{Number: 1, Hash: common.Hash{1}}
	source := &blockingUpstreamFollowSource{
		blockingL1FollowSource: &blockingL1FollowSource{
			calls:   make(map[uint64]int),
			started: make(chan uint64, 1),
			release: l1Release,
		},
		status: &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 3, L1Origin: origin},
			SafeL2:      eth.L2BlockRef{Number: 2, L1Origin: origin},
			FinalizedL2: eth.L2BlockRef{Number: 1, L1Origin: origin},
		},
		statusStarted: make(chan struct{}, 1),
		statusRelease: statusRelease,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	driver := &Driver{driverCtx: ctx, upstreamFollowSource: source}
	resultCh := make(chan followUpstreamResult, 1)

	driver.startFollowUpstreamFetch(resultCh)
	select {
	case <-source.statusStarted:
	case <-time.After(time.Second):
		t.Fatal("upstream status fetch did not start")
	}
	select {
	case <-resultCh:
		t.Fatal("fetch completed while the upstream request was blocked")
	default:
	}

	close(statusRelease)
	select {
	case result := <-resultCh:
		require.NoError(t, result.statusErr)
		require.NoError(t, result.l1Err)
		require.Equal(t, eth.L1BlockRef{Number: 1, Hash: common.Hash{1}}, result.l1Refs[1])
	case <-time.After(time.Second):
		t.Fatal("upstream fetch did not complete")
	}
	driver.wg.Wait()
}

func TestFetchFollowUpstreamTimeoutIncludesStatusRequest(t *testing.T) {
	source := &blockingUpstreamFollowSource{
		blockingL1FollowSource: &blockingL1FollowSource{
			calls:   make(map[uint64]int),
			started: make(chan uint64, 1),
			release: make(chan struct{}),
		},
		statusStarted: make(chan struct{}, 1),
		statusRelease: make(chan struct{}),
	}
	driver := &Driver{upstreamFollowSource: source}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	result := driver.fetchFollowUpstream(ctx)
	require.ErrorIs(t, result.statusErr, context.DeadlineExceeded)
	require.Empty(t, source.calls)
}
