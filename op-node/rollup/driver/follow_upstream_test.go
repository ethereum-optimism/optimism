package driver

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type blockingUpstreamFollowSource struct {
	status  *sources.FollowStatus
	err     error
	started chan struct{}
	release chan struct{}
	calls   []uint64
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
	return eth.L1BlockRef{Number: number, Hash: common.Hash{byte(number)}}, nil
}

// followMetricsStub records follow-source request outcomes; all other Metrics
// methods panic via the nil embedded interface if called.
type followMetricsStub struct {
	Metrics
	results           []string
	currentL1NotFound int
}

func (m *followMetricsStub) RecordFollowSourceRequest(result string) {
	m.results = append(m.results, result)
}

func (m *followMetricsStub) RecordFollowSourceCurrentL1NotFound() {
	m.currentL1NotFound++
}

func l1Ref(number uint64) eth.L1BlockRef {
	return eth.L1BlockRef{Number: number, Hash: common.Hash{byte(number)}}
}

// stubUpstreamFollowSource serves a fixed status and answers L1 lookups from our
// side of the chain: blocks in l1 are found, numbers in l1Errs fail, and any
// other number is not found.
type stubUpstreamFollowSource struct {
	status *sources.FollowStatus
	l1     map[uint64]eth.L1BlockRef
	l1Errs map[uint64]error
}

func (s *stubUpstreamFollowSource) GetFollowStatus(context.Context) (*sources.FollowStatus, error) {
	return s.status, nil
}

func (s *stubUpstreamFollowSource) L1BlockRefByNumber(_ context.Context, number uint64) (eth.L1BlockRef, error) {
	if err, ok := s.l1Errs[number]; ok {
		return eth.L1BlockRef{}, err
	}
	if ref, ok := s.l1[number]; ok {
		return ref, nil
	}
	return eth.L1BlockRef{}, fmt.Errorf("failed to fetch header by num %d: %w", number, ethereum.NotFound)
}

// TestFollowUpstreamCurrentL1 checks that an upstream CurrentL1 our L1 chain does
// not have yet only drops CurrentL1: the verified heads are still applied, and the
// request is not counted as an error. Any other CurrentL1 failure still rejects
// the whole status.
func TestFollowUpstreamCurrentL1(t *testing.T) {
	ourL1 := map[uint64]eth.L1BlockRef{1: l1Ref(1), 2: l1Ref(2), 3: l1Ref(3), 4: l1Ref(4)}
	newStatus := func(currentL1 eth.L1BlockRef) *sources.FollowStatus {
		return &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 3, L1Origin: l1Ref(3).ID()},
			SafeL2:      eth.L2BlockRef{Number: 2, L1Origin: l1Ref(2).ID()},
			FinalizedL2: eth.L2BlockRef{Number: 1, L1Origin: l1Ref(1).ID()},
			CurrentL1:   currentL1,
		}
	}

	tests := []struct {
		name          string
		currentL1     eth.L1BlockRef
		l1Errs        map[uint64]error
		wantRejected  string // request result if the status is rejected, empty if accepted
		wantCurrentL1 eth.L1BlockRef
		wantNotFound  int
	}{
		{
			name:          "verified",
			currentL1:     l1Ref(4),
			wantCurrentL1: l1Ref(4),
		},
		{
			name:          "not reported by upstream",
			currentL1:     eth.L1BlockRef{},
			wantCurrentL1: eth.L1BlockRef{},
		},
		{
			name:          "ahead of our L1 view",
			currentL1:     l1Ref(5),
			wantCurrentL1: eth.L1BlockRef{},
			wantNotFound:  1,
		},
		{
			name:         "L1 lookup fails",
			currentL1:    l1Ref(4),
			l1Errs:       map[uint64]error{4: errors.New("connection refused")},
			wantRejected: "error_l1_lookup",
		},
		{
			name:         "not on our L1 chain",
			currentL1:    eth.L1BlockRef{Number: 4, Hash: common.Hash{0xff}},
			wantRejected: "error_l1_mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := newStatus(tt.currentL1)
			metrics := &followMetricsStub{}
			driver := &Driver{
				driverCtx:            context.Background(),
				upstreamFollowSource: &stubUpstreamFollowSource{status: upstream, l1: ourL1, l1Errs: tt.l1Errs},
				log:                  testlog.Logger(t, log.LevelError),
				metrics:              metrics,
			}

			status := driver.followUpstream()

			require.Equal(t, tt.wantNotFound, metrics.currentL1NotFound)
			require.Equal(t, tt.currentL1, upstream.CurrentL1, "must not modify the upstream status")
			if tt.wantRejected != "" {
				require.Nil(t, status)
				require.Equal(t, []string{tt.wantRejected}, metrics.results)
				return
			}
			require.NotNil(t, status)
			require.Equal(t, upstream.LocalSafeL2, status.LocalSafeL2)
			require.Equal(t, upstream.SafeL2, status.SafeL2)
			require.Equal(t, upstream.FinalizedL2, status.FinalizedL2)
			require.Equal(t, tt.wantCurrentL1, status.CurrentL1)
			require.Empty(t, metrics.results, "the event loop records the request result")
		})
	}
}

// TestFollowUpstreamHeadOriginNotFound checks that the heads, unlike CurrentL1,
// must all be verified: if the L1 origin of one is not found, nothing is applied.
func TestFollowUpstreamHeadOriginNotFound(t *testing.T) {
	metrics := &followMetricsStub{}
	driver := &Driver{
		driverCtx: context.Background(),
		upstreamFollowSource: &stubUpstreamFollowSource{
			status: &sources.FollowStatus{
				LocalSafeL2: eth.L2BlockRef{Number: 3, L1Origin: l1Ref(3).ID()},
				SafeL2:      eth.L2BlockRef{Number: 2, L1Origin: l1Ref(2).ID()},
				FinalizedL2: eth.L2BlockRef{Number: 1, L1Origin: l1Ref(1).ID()},
				CurrentL1:   l1Ref(4),
			},
			l1: map[uint64]eth.L1BlockRef{1: l1Ref(1), 2: l1Ref(2), 4: l1Ref(4)},
		},
		log:     testlog.Logger(t, log.LevelError),
		metrics: metrics,
	}

	require.Nil(t, driver.followUpstream())
	require.Equal(t, []string{"error_l1_lookup"}, metrics.results)
	require.Zero(t, metrics.currentL1NotFound)
}

func TestStartFollowUpstreamFetchIsAsync(t *testing.T) {
	source := &blockingUpstreamFollowSource{
		status: &sources.FollowStatus{
			LocalSafeL2: eth.L2BlockRef{Number: 3, L1Origin: l1Ref(1).ID()},
			SafeL2:      eth.L2BlockRef{Number: 2, L1Origin: l1Ref(2).ID()},
			FinalizedL2: eth.L2BlockRef{Number: 1, L1Origin: l1Ref(3).ID()},
			CurrentL1:   l1Ref(4),
		},
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	driver := &Driver{
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
