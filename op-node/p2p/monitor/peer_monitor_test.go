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

const kickerBanDuration = 10 * time.Minute

func peerKickerSetup(t *testing.T) (*PeerMonitor, *clock2.DeterministicClock, *mocks.ConnectedPeers, *mocks.PeerProtector, *mocks.Scores, *mocks.PeerBlocker) {
	l := testlog.Logger(t, log.LvlInfo)
	clock := clock2.NewDeterministicClock(time.UnixMilli(10000))
	peers := mocks.NewConnectedPeers(t)
	protector := mocks.NewPeerProtector(t)
	blocker := mocks.NewPeerBlocker(t)
	scores := mocks.NewScores(t)
	kicker := NewPeerMonitor(context.Background(), l, clock, peers, protector, scores, blocker, -100, kickerBanDuration)
	return kicker, clock, peers, protector, scores, blocker
}

func TestPeriodicallyCheckNextPeer(t *testing.T) {
	kicker, clock, _, _, _, _ := peerKickerSetup(t)
	// Each time a step is performed, it calls Done on the wait group so we can wait for it to be performed
	stepCh := make(chan struct{}, 10)
	kicker.bgTasks.Add(1)
	var actionErr error
	go kicker.background(func() error {
		stepCh <- struct{}{}
		return actionErr
	})
	defer kicker.Stop()
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

	t.Run("Check each peer then refresh list", func(t *testing.T) {
		kicker, _, peers, _, scores, _ := peerKickerSetup(t)
		peers.EXPECT().Peers().Return(peerIDs).Once()
		for _, id := range peerIDs {
			scores.EXPECT().GetPeerScore(id).Return(1, nil).Once()

			require.NoError(t, kicker.checkNextPeer())
		}

		updatedPeers := []peer.ID{
			peer.ID("x"),
			peer.ID("y"),
			peer.ID("z"),
			peer.ID("a"),
		}
		peers.EXPECT().Peers().Return(updatedPeers).Once()
		for _, id := range updatedPeers {
			scores.EXPECT().GetPeerScore(id).Return(1, nil).Once()

			require.NoError(t, kicker.checkNextPeer())
		}
	})

	t.Run("Close and ban peer when below min score", func(t *testing.T) {
		kicker, clock, peers, protector, scores, blocker := peerKickerSetup(t)
		id := peerIDs[0]
		peers.EXPECT().Peers().Return(peerIDs).Once()
		scores.EXPECT().GetPeerScore(id).Return(-101, nil).Once()
		protector.EXPECT().IsProtected(id).Return(false).Once()
		peers.EXPECT().ClosePeer(id).Return(nil).Once()
		blocker.EXPECT().BanPeer(id, clock.Now().Add(kickerBanDuration)).Return(nil).Once()

		require.NoError(t, kicker.checkNextPeer())
	})

	t.Run("Do not close protected peer when below min score", func(t *testing.T) {
		kicker, _, peers, protector, scores, _ := peerKickerSetup(t)
		id := peerIDs[0]
		peers.EXPECT().Peers().Return(peerIDs).Once()
		scores.EXPECT().GetPeerScore(id).Return(-101, nil).Once()
		protector.EXPECT().IsProtected(id).Return(true)

		require.NoError(t, kicker.checkNextPeer())
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
