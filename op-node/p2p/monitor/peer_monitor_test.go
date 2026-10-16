package monitor

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/monitor/mocks"
	clock2 "github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

const testBanDuration = 2 * time.Hour

func peerMonitorSetup(t *testing.T) (*PeerMonitor, *clock2.DeterministicClock, *mocks.PeerManager) {
	l := testlog.Logger(t, log.LevelInfo)
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
	actionErr := make(chan error, 1)
	go monitor.background(func() error {
		stepCh <- struct{}{}
		select {
		case err := <-actionErr:
			return err
		default:
			return nil
		}
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
	actionErr <- errors.New("boom")
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
		manager.EXPECT().IsStatic(id).Return(false).Once()
		manager.EXPECT().BanPeer(id, clock.Now().Add(testBanDuration)).Return(nil).Once()

		require.NoError(t, monitor.checkNextPeer())
	})

	t.Run("Do not close protected peer when below min score", func(t *testing.T) {
		monitor, _, manager := peerMonitorSetup(t)
		id := peerIDs[0]
		manager.EXPECT().Peers().Return(peerIDs).Once()
		manager.EXPECT().GetPeerScore(id).Return(-101, nil).Once()
		manager.EXPECT().IsStatic(id).Return(true)

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

func TestCheckNextPeerBanObservability(t *testing.T) {
	banErr := errors.New("ban failure")
	for _, test := range []struct {
		name      string
		score     float64
		protected bool
		ban       bool
		banErr    error
	}{
		{name: "healthy", score: 1},
		{name: "at threshold", score: -100},
		{name: "protected", score: -101, protected: true},
		{name: "low score", score: -101, ban: true},
		{name: "ban failure", score: -101, ban: true, banErr: banErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			monitor, clock, manager := peerMonitorSetup(t)
			logger, logs := testlog.CaptureLogger(t, log.LevelDebug)
			monitor.l = logger
			id := peer.ID("test-peer")
			expiry := clock.Now().Add(testBanDuration)
			manager.EXPECT().Peers().Return([]peer.ID{id}).Once()
			manager.EXPECT().GetPeerScore(id).Return(test.score, nil).Once()
			if test.score < monitor.minScore {
				manager.EXPECT().IsStatic(id).Return(test.protected).Once()
			}
			if test.ban {
				manager.EXPECT().BanPeer(id, expiry).Return(test.banErr).Once()
			}
			err := monitor.checkNextPeer()
			if test.banErr != nil {
				require.ErrorIs(t, err, test.banErr)
			} else {
				require.NoError(t, err)
			}
			events := logs.FindLogs(testlog.NewLevelFilter(log.LevelDebug))
			if !test.ban || test.banErr != nil {
				require.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			require.Equal(t, id, events[0].AttrValue("peer"))
			require.Equal(t, test.score, events[0].AttrValue("score"))
			require.Equal(t, monitor.minScore, events[0].AttrValue("threshold"))
			require.Equal(t, expiry, events[0].AttrValue("expiry"))
		})
	}
}
