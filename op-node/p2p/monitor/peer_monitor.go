package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// Time delay between checking the score of each peer to avoid activity spikes
	checkInterval = 1 * time.Second
)

//go:generate mockery --name PeerManager --output mocks/ --with-expecter=true
type PeerManager interface {
	Peers() []peer.ID
	GetPeerScore(id peer.ID) (float64, error)
	IsStatic(peer.ID) bool
	// BanPeer bans the peer until the specified time and disconnects any existing connections.
	BanPeer(peer.ID, time.Time) error
}

// PeerMonitor runs a background process to periodically check for peers with scores below a minimum.
// When it finds bad peers, it disconnects and bans them.
// A delay is introduced between each peer being checked to avoid spikes in system load.
type PeerMonitor struct {
	ctx         context.Context
	cancelFn    context.CancelFunc
	l           log.Logger
	clock       clock.Clock
	manager     PeerManager
	minScore    float64
	banDuration time.Duration

	bgTasks sync.WaitGroup

	// Used by checkNextPeer and must only be accessed from the background thread
	peerList    []peer.ID
	nextPeerIdx int
}

func NewPeerMonitor(ctx context.Context, l log.Logger, clock clock.Clock, manager PeerManager, minScore float64, banDuration time.Duration) *PeerMonitor {
	ctx, cancelFn := context.WithCancel(ctx)
	return &PeerMonitor{
		ctx:         ctx,
		cancelFn:    cancelFn,
		l:           l,
		clock:       clock,
		manager:     manager,
		minScore:    minScore,
		banDuration: banDuration,
	}
}
func (p *PeerMonitor) Start() {
	p.bgTasks.Add(1)
	go p.background(p.checkNextPeer)
}

func (p *PeerMonitor) Stop() {
	p.cancelFn()
	p.bgTasks.Wait()
}

// checkNextPeer checks the next peer and disconnects and bans it if its score is too low and its not protected.
// The first call gets the list of current peers and checks the first one, then each subsequent call checks the next
// peer in the list.  When the end of the list is reached, an updated list of connected peers is retrieved and the process
// starts again.
func (p *PeerMonitor) checkNextPeer() error {
	// Get a new list of peers to check if we've checked all peers in the previous list
	if p.nextPeerIdx >= len(p.peerList) {
		p.peerList = p.manager.Peers()
		p.nextPeerIdx = 0
	}
	if len(p.peerList) == 0 {
		// No peers to check
		return nil
	}
	id := p.peerList[p.nextPeerIdx]
	p.nextPeerIdx++
	score, err := p.manager.GetPeerScore(id)
	if err != nil {
		return fmt.Errorf("retrieve score for peer %v: %w", id, err)
	}
	if score >= p.minScore {
		return nil
	}
	if p.manager.IsStatic(id) {
		return nil
	}
	if err := p.manager.BanPeer(id, p.clock.Now().Add(p.banDuration)); err != nil {
		return fmt.Errorf("banning peer %v: %w", id, err)
	}

	return nil
}

// background is intended to run as a separate go routine. It will call the supplied action function every checkInterval
// until the context is done.
func (p *PeerMonitor) background(action func() error) {
	defer p.bgTasks.Done()
	ticker := p.clock.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.Ch():
			if err := action(); err != nil {
				p.l.Warn("Error while checking connected peer score", "err", err)
			}
		}
	}
}
