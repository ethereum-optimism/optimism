package monitor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/monitor/mocks"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	clock2 "github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

const testBanDuration = 2 * time.Hour

func peerMonitorSetup(t *testing.T) (*PeerMonitor, *clock2.DeterministicClock, *mocks.PeerManager) {
	l := testlog.Logger(t, log.LvlInfo)
	clock := clock2.NewDeterministicClock(time.UnixMilli(10000))
	manager := mocks.NewPeerManager(t)
	monitor := NewPeerMonitor(context.Background(), l, clock, manager, -100, testBanDuration)
	return monitor, clock, manager
}

func TestPeriodicallyCheckNextPeer(t *testing.T) {
	monitor, clock, _ := peerMonitorSetup(t)
	// Each time a step is performed, it calls Done on the wait group so we can wait for it to be performed
	stepCh := make(chan struct{}, 10)
	monitor.bgTasks.Add(1)
	var actionErr error
	go monitor.background(func() error {
		stepCh <- struct{}{}
		return actionErr
	})
	defer monitor.Stop()
	// Wait for the step ticker to be started
	clock.WaitForNewPendingTaskWithTimeout(30 * time.Second)

	// Should perform another step after each interval
	for i := 0; i < 5; i++ {
		clock.AdvanceTime(checkInterval)
		waitForChan(t, stepCh, fmt.Sprintf("Did not perform step %v", i))
		require.Len(t, stepCh, 0)
	}

	// Should continue executing periodically even after an error
	actionErr = errors.New("boom")
	for i := 0; i < 5; i++ {
		clock.AdvanceTime(checkInterval)
		waitForChan(t, stepCh, fmt.Sprintf("Did not perform step %v", i))
		require.Len(t, stepCh, 0)
	}
}

func TestCheckNextPeer(t *testing.T) {
	peerIDs := []peer.ID{
		peer.ID("a"),
		peer.ID("b"),
		peer.ID("c"),
	}

	t.Run("No peers", func(t *testing.T) {
		monitor, _, manager := peerMonitorSetup(t)
		manager.EXPECT().Peers().Return(nil).Once()
		require.NoError(t, monitor.checkNextPeer())
	})

	t.Run("Check each peer then refresh list", func(t *testing.T) {
		monitor, _, manager := peerMonitorSetup(t)
		manager.EXPECT().Peers().Return(peerIDs).Once()
		for _, id := range peerIDs {
			manager.EXPECT().GetPeerScore(id).Return(1, nil).Once()

			require.NoError(t, monitor.checkNextPeer())
		}

		updatedPeers := []peer.ID{
			peer.ID("x"),
			peer.ID("y"),
			peer.ID("z"),
			peer.ID("a"),
		}
		manager.EXPECT().Peers().Return(updatedPeers).Once()
		for _, id := range updatedPeers {
			manager.EXPECT().GetPeerScore(id).Return(1, nil).Once()

			require.NoError(t, monitor.checkNextPeer())
		}
	})

	t.Run("Close and ban peer when below min score", func(t *testing.T) {
		monitor, clock, manager := peerMonitorSetup(t)
		id := peerIDs[0]
		manager.EXPECT().Peers().Return(peerIDs).Once()
		manager.EXPECT().GetPeerScore(id).Return(-101, nil).Once()
		manager.EXPECT().IsProtected(id).Return(false).Once()
		manager.EXPECT().BanPeer(id, clock.Now().Add(testBanDuration)).Return(nil).Once()

		require.NoError(t, monitor.checkNextPeer())
	})

	t.Run("Do not close protected peer when below min score", func(t *testing.T) {
		monitor, _, manager := peerMonitorSetup(t)
		id := peerIDs[0]
		manager.EXPECT().Peers().Return(peerIDs).Once()
		manager.EXPECT().GetPeerScore(id).Return(-101, nil).Once()
		manager.EXPECT().IsProtected(id).Return(true)

		require.NoError(t, monitor.checkNextPeer())
	})
}

func waitForChan(t *testing.T, ch chan struct{}, msg string) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFn()
	select {
	case <-ctx.Done():
		t.Fatal(msg)
	case <-ch:
		// Ok
	}
}
