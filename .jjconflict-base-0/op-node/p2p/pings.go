package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/clock"
)

const (
	pingRound                 = 3 * time.Minute
	pingsPerSecond rate.Limit = 1
	pingsBurst                = 10
)

type PingFn func(ctx context.Context, peerID peer.ID) <-chan ping.Result

type PeersFn func() []peer.ID

type PingService struct {
	ping  PingFn
	peers PeersFn

	clock clock.Clock

	log log.Logger

	ctx    context.Context
	cancel context.CancelFunc

	trace func(work string)

	// to signal service completion
	wg sync.WaitGroup
}

func NewPingService(
	log log.Logger,
	ping PingFn,
	peers PeersFn,
) *PingService {
	return newTracedPingService(log, ping, peers, clock.SystemClock, nil)
}

func newTracedPingService(
	log log.Logger,
	ping PingFn,
	peers PeersFn,
	clock clock.Clock,
	trace func(work string),
) *PingService {
	ctx, cancel := context.WithCancel(context.Background())
	srv := &PingService{
		ping:   ping,
		peers:  peers,
		log:    log,
		clock:  clock,
		ctx:    ctx,
		cancel: cancel,
		trace:  trace,
	}
	srv.wg.Add(1)
	go srv.pingPeersBackground()
	return srv
}

func (p *PingService) Close() {
	p.cancel()
	p.wg.Wait()
}

func (e *PingService) pingPeersBackground() {
	defer e.wg.Done()

	tick := e.clock.NewTicker(pingRound)
	defer tick.Stop()

	if e.trace != nil {
		e.trace("started")
	}

	for {
		select {
		case <-tick.Ch():
			e.pingPeers()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *PingService) pingPeers() {
	if e.trace != nil {
		e.trace("pingPeers start")
	}
	ctx, cancel := context.WithTimeout(e.ctx, pingRound)
	defer cancel()

	// Wait group to wait for all pings to complete
	var wg sync.WaitGroup
	// Rate-limiter to help schedule the ping
	// work without overwhelming ourselves.
	rl := rate.NewLimiter(pingsPerSecond, pingsBurst)

	// iterate through the connected peers
	for i, peerID := range e.peers() {
		if e.ctx.Err() != nil { // stop if the service is closing or timing out
			return
		}
		if ctx.Err() != nil {
			e.log.Warn("failed to ping all peers", "pinged", i, "err", ctx.Err())
			return
		}
		if err := rl.Wait(ctx); err != nil {
			// host may be getting closed, causing a parent ctx to close.
			return
		}
		wg.Add(1)
		go func(peerID peer.ID) {
			e.pingPeer(ctx, peerID)
			wg.Done()
		}(peerID)
	}
	wg.Wait()
	if e.trace != nil {
		e.trace("pingPeers end")
	}
}

func (e *PingService) pingPeer(ctx context.Context, peerID peer.ID) {
	results := e.ping(ctx, peerID)
	// the results channel will be closed by the ping.Ping function upon context close / completion
	res, ok := <-results
	if !ok {
		// timed out or completed before Pong
		e.log.Warn("failed to ping peer, context cancelled", "peerID", peerID, "err", ctx.Err())
	} else if res.Error != nil {
		e.log.Warn("failed to ping peer, communication error", "peerID", peerID, "err", res.Error)
	} else {
		e.log.Debug("ping-pong", "peerID", peerID, "rtt", res.RTT)
	}
}
