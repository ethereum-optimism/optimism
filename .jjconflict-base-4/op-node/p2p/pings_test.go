package p2p

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestPingService(t *testing.T) {
	peers := []peer.ID{"a", "b", "c"}
	log, captLog := testlog.CaptureLogger(t, slog.LevelDebug)

	pingCount := &atomic.Int64{}
	pingFn := PingFn(func(ctx context.Context, peerID peer.ID) <-chan ping.Result {
		out := make(chan ping.Result, 1)
		// Atomically add, so that parallel pings don't have a 1/1000 chance
		// to increment at the same time and create a CI flake.
		newValue := pingCount.Add(1)
		switch (newValue - 1) % 3 {
		case 0:
			// success
			out <- ping.Result{
				RTT:   time.Millisecond * 10,
				Error: nil,
			}
		case 1:
			// fake timeout
		case 2:
			// error
			out <- ping.Result{
				RTT:   0,
				Error: errors.New("fake error"),
			}
		}
		close(out)
		return out
	})

	fakeClock := clock.NewDeterministicClock(time.Now())
	peersFn := PeersFn(func() []peer.ID {
		return peers
	})

	trace := make(chan string)
	srv := newTracedPingService(log, pingFn, peersFn, fakeClock, func(work string) {
		trace <- work
	})

	// wait for ping service to get online
	require.Equal(t, "started", <-trace)
	fakeClock.AdvanceTime(pingRound)
	// wait for first round to start and complete
	require.Equal(t, "pingPeers start", <-trace)
	require.Equal(t, "pingPeers end", <-trace)
	// see if client has hit all 3 cases we simulated on the server-side
	require.Equal(t, int64(3), pingCount.Load(), "pinged 3 peers")

	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("ping-pong")), "case 0")
	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("failed to ping peer, context cancelled")), "case 1")
	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("failed to ping peer, communication error")), "case 2")
	captLog.Clear()

	// advance clock again to proceed to second round, and wait for the round to start and complete
	fakeClock.AdvanceTime(pingRound)
	require.Equal(t, "pingPeers start", <-trace)
	require.Equal(t, "pingPeers end", <-trace)
	// see if client has hit all 3 cases we simulated on the server-side
	require.Equal(t, int64(6), pingCount.Load(), "pinged 3 peers again")
	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("ping-pong")), "case 0")
	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("failed to ping peer, context cancelled")), "case 1")
	require.NotNil(t, captLog.FindLog(testlog.NewMessageContainsFilter("failed to ping peer, communication error")), "case 2")
	captLog.Clear()

	srv.Close()
}
