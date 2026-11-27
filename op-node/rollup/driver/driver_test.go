package driver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type fixedSyncStatus struct {
	SyncStatusTracker
}

func (fixedSyncStatus) SyncStatus() *eth.SyncStatus {
	return &eth.SyncStatus{}
}

type panickingL2 struct {
	L2Chain
}

func (panickingL2) L2BlockRefByNumber(context.Context, uint64) (eth.L2BlockRef, error) {
	panic("test panic")
}

// TestBlockRefWithStatusReleasesEventLoopOnPanic checks that the event loop continues when the
// block lookup panics. The RPC server recovers such a panic, so the node keeps running.
func TestBlockRefWithStatusReleasesEventLoopOnPanic(t *testing.T) {
	d := &Driver{
		StatusTracker: fixedSyncStatus{},
		SyncDeriver:   &SyncDeriver{L2: panickingL2{}},
		stateReq:      make(chan chan struct{}),
	}
	released := make(chan struct{})
	go func() { // the event loop's handling of a state request
		respCh := <-d.stateReq
		respCh <- struct{}{}
		close(released)
	}()

	require.Panics(t, func() {
		_, _, _ = d.BlockRefWithStatus(context.Background(), 1)
	})
	select {
	case <-released:
	case <-time.After(10 * time.Second):
		t.Fatal("event loop is still held by the state request")
	}
}
