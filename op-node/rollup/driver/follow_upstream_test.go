package driver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type blockingUpstreamFollowSource struct {
	status  *sources.FollowStatus
	err     error
	started chan struct{}
	release chan struct{}
	calls   []uint64
	refs    map[uint64]eth.L1BlockRef
}

func (s *blockingUpstreamFollowSource) GetFollowStatus(ctx context.Context) (*sources.FollowStatus, error) {
	s.started <- struct{}{}
	select {
	case <-s.release:
		return s.status, s.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *blockingUpstreamFollowSource) L1BlockRefByNumber(_ context.Context, number uint64) (eth.L1BlockRef, error) {
	s.calls = append(s.calls, number)
	if ref, ok := s.refs[number]; ok {
		return ref, nil
	}
	return eth.L1BlockRef{Number: number, Hash: common.Hash{byte(number)}}, nil
}

// followMetricsStub records follow-source request outcomes; all other Metrics
// methods panic via the nil embedded interface if called.
type followMetricsStub struct {
	Metrics
	results []string
}

func (m *followMetricsStub) RecordFollowSourceRequest(result string) {
	m.results = append(m.results, result)
}

func TestStartFollowUpstreamFetchIsAsync(t *testing.T) {
	ref := func(number uint64) eth.BlockID {
		return eth.BlockID{Number: number, Hash: common.Hash{byte(number)}}
	}
	source := &blockingUpstreamFollowSource{
		status: &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 3, L1Origin: ref(1)},
			SafeL2:      eth.L2BlockRef{Number: 2, L1Origin: ref(2)},
			FinalizedL2: eth.L2BlockRef{Number: 1, L1Origin: ref(3)},
			CurrentL1:   eth.L1BlockRef{Number: 4, Hash: common.Hash{4}},
		},
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	driver := &Driver{
		StatusTracker:        &followStatusTrackerStub{},
		driverCtx:            ctx,
		upstreamFollowSource: source,
		log:                  testlog.Logger(t, log.LevelError),
		metrics:              &followMetricsStub{},
	}
	resultCh := make(chan *sources.FollowStatus, 1)

	driver.startFollowUpstreamFetch(resultCh)
	select {
	case <-source.started:
	case <-time.After(time.Second):
		t.Fatal("upstream status request did not start")
	}
	select {
	case <-resultCh:
		t.Fatal("fetch completed while the upstream request was blocked")
	default:
	}

	close(source.release)
	select {
	case status := <-resultCh:
		require.Same(t, source.status, status)
	case <-time.After(time.Second):
		t.Fatal("upstream fetch did not complete")
	}
	driver.wg.Wait()
	require.Equal(t, []uint64{1, 2, 3, 4}, source.calls)
}

// TestStartFollowUpstreamFetchDeliversNilOnError pins the liveness contract the
// event loop's single-flight flag depends on: a failed fetch must still deliver
// a (nil) result, so the flag is cleared and follow-source keeps retrying.
func TestStartFollowUpstreamFetchDeliversNilOnError(t *testing.T) {
	source := &blockingUpstreamFollowSource{
		err:     errors.New("upstream unavailable"),
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	close(source.release)
	metrics := &followMetricsStub{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	driver := &Driver{
		StatusTracker:        &followStatusTrackerStub{},
		driverCtx:            ctx,
		upstreamFollowSource: source,
		log:                  testlog.Logger(t, log.LevelError),
		metrics:              metrics,
	}
	resultCh := make(chan *sources.FollowStatus, 1)

	driver.startFollowUpstreamFetch(resultCh)
	select {
	case status := <-resultCh:
		require.Nil(t, status)
	case <-time.After(time.Second):
		t.Fatal("failed fetch did not deliver a result")
	}
	driver.wg.Wait()
	require.Equal(t, []string{"error_fetch_status"}, metrics.results)
	require.Empty(t, source.calls)
}

// A restarting source can expose persisted finalized heads before derivation
// initializes. Applying those heads rewinds a live sequencer to that checkpoint.
// A real genesis reference, however, is a valid initialized source.
func TestFollowUpstreamWaitsForInitializedL1(t *testing.T) {
	genesis := eth.L1BlockRef{Hash: common.Hash{0xab}}
	source := &blockingUpstreamFollowSource{
		status: &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 30, L1Origin: genesis.ID()},
			SafeL2:      eth.L2BlockRef{Number: 30, L1Origin: genesis.ID()},
			FinalizedL2: eth.L2BlockRef{Number: 30, L1Origin: genesis.ID()},
		},
		refs:    map[uint64]eth.L1BlockRef{0: genesis},
		started: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	close(source.release)
	driver := &Driver{driverCtx: context.Background(), upstreamFollowSource: source,
		StatusTracker: &followStatusTrackerStub{},
		log:           testlog.Logger(t, log.LevelError), metrics: &followMetricsStub{}}
	require.Nil(t, driver.followUpstream(), "uninitialized source must not rewind the engine to persisted finalized heads")
	require.Empty(t, source.calls)
	source.status.CurrentL1 = genesis
	require.Same(t, source.status, driver.followUpstream(), "initialized L1 genesis must remain supported")
	require.Equal(t, []uint64{0, 0, 0, 0}, source.calls)
}

type followStatusTrackerStub struct {
	SyncStatusTracker
	value eth.SyncStatus
}

func (s *followStatusTrackerStub) SyncStatus() *eth.SyncStatus { return &s.value }

func TestFollowUpstreamDistinguishesReplayFromL1Reorg(t *testing.T) {
	for _, tc := range []struct {
		name      string
		current   uint64
		localSafe uint64
		reorg     bool
		private   bool
		accept    bool
	}{
		{"restarting source still replaying canonical L1", 5, 30, false, false, false},
		{"source reached previous L1 but not its L2 head", 10, 30, false, false, false},
		{"source caught up to previous view", 10, 80, false, false, true},
		{"source processed newer L1 and revoked L2 history", 11, 30, false, false, true},
		{"previous L1 view was reorged out", 5, 30, true, false, true},
		{"private recovery validates its lower claimed anchor separately", 10, 30, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous := eth.L1BlockRef{Number: 10, Hash: common.Hash{10}}
			source := &blockingUpstreamFollowSource{
				status: &sources.FollowStatus{
					LocalSafeL2: eth.L2BlockRef{Number: tc.localSafe, L1Origin: eth.BlockID{Number: 1, Hash: common.Hash{1}}},
					CurrentL1:   eth.L1BlockRef{Number: tc.current, Hash: common.Hash{byte(tc.current)}},
				},
				started: make(chan struct{}, 1), release: make(chan struct{}),
				refs: make(map[uint64]eth.L1BlockRef),
			}
			if tc.private {
				source.status.Recovery = &sources.FollowRecoveryStatus{}
			}
			if tc.reorg {
				source.refs[10] = eth.L1BlockRef{Number: 10, Hash: common.Hash{0xff}}
			}
			close(source.release)
			driver := &Driver{driverCtx: t.Context(), upstreamFollowSource: source,
				StatusTracker: &followStatusTrackerStub{value: eth.SyncStatus{CurrentL1: previous, LocalSafeL2: eth.L2BlockRef{Number: 80}}},
				log:           testlog.Logger(t, log.LevelError), metrics: &followMetricsStub{}}
			if tc.accept {
				require.Same(t, source.status, driver.followUpstream())
			} else {
				require.Nil(t, driver.followUpstream(), "source replay must not revoke live history on unchanged L1")
			}
		})
	}
}
