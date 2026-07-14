package driver

import (
	"context"
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
	started chan struct{}
	release chan struct{}
	calls   []uint64
}

func (s *blockingUpstreamFollowSource) GetFollowStatus(ctx context.Context) (*sources.FollowStatus, error) {
	s.started <- struct{}{}
	select {
	case <-s.release:
		return s.status, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *blockingUpstreamFollowSource) L1BlockRefByNumber(_ context.Context, number uint64) (eth.L1BlockRef, error) {
	s.calls = append(s.calls, number)
	return eth.L1BlockRef{Number: number, Hash: common.Hash{byte(number)}}, nil
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
		driverCtx:            ctx,
		upstreamFollowSource: source,
		log:                  testlog.Logger(t, log.LevelError),
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
