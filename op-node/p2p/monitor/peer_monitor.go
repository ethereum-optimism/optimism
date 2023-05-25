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
	// Time delay between checking the score of each peer to avoid
	checkInterval = 1 * time.Second
)

//go:generate mockery --name Scores --output mocks/ --with-expecter=true
type Scores interface {
	GetPeerScore(id peer.ID) (float64, error)
}

// ConnectedPeers enables querying the set of currently connected peers and disconnecting peers
//
//go:generate mockery --name ConnectedPeers --output mocks/ --with-expecter=true
type ConnectedPeers interface {
	Peers() []peer.ID
	ClosePeer(peer.ID) error
}

//go:generate mockery --name PeerProtector --output mocks/ --with-expecter=true
type PeerProtector interface {
	IsProtected(peer.ID) bool
}

//go:generate mockery --name PeerBlocker --output mocks/ --with-expecter=true
type PeerBlocker interface {
	BanPeer(peer.ID, time.Time) error
}

type PeerMonitor struct {
	ctx         context.Context
	cancelFn    context.CancelFunc
	l           log.Logger
	clock       clock.Clock
	peers       ConnectedPeers
	protector   PeerProtector
	scores      Scores
	blocker     PeerBlocker
	minScore    float64
	banDuration time.Duration

	bgTasks sync.WaitGroup

	// Used by checkNextPeer and must only be accessed from the background thread
	peerList    []peer.ID
	nextPeerIdx int
}

func NewPeerMonitor(ctx context.Context, l log.Logger, clock clock.Clock, peers ConnectedPeers, protector PeerProtector, scores Scores, blocker PeerBlocker, minScore float64, banDuration time.Duration) *PeerMonitor {
	ctx, cancelFn := context.WithCancel(ctx)
	return &PeerMonitor{
		ctx:         ctx,
		cancelFn:    cancelFn,
		l:           l,
		clock:       clock,
		peers:       peers,
		protector:   protector,
		scores:      scores,
		blocker:     blocker,
		minScore:    minScore,
		banDuration: banDuration,
	}
}
func (k *PeerMonitor) Start() {
	k.bgTasks.Add(1)
	go k.background(k.checkNextPeer)
}

func (k *PeerMonitor) Stop() {
	k.cancelFn()
	k.bgTasks.Wait()
}

// checkNextPeer checks the next peer and disconnects and bans it if its score is too low and its not protected.
// The first call gets the list of current peers and checks the first one, then each subsequent call checks the next
// peer in the list.  When the end of the list is reached, an updated list of connected peers is retrieved and the process
// starts again.
func (k *PeerMonitor) checkNextPeer() error {
	// Get a new list of peers to check if we've checked all peers in the previous list
	if k.nextPeerIdx >= len(k.peerList) {
		k.peerList = k.peers.Peers()
		k.nextPeerIdx = 0
	}
	id := k.peerList[k.nextPeerIdx]
	k.nextPeerIdx++
	score, err := k.scores.GetPeerScore(id)
	if err != nil {
		return fmt.Errorf("retrieve score for peer %v: %w", id, err)
	}
	if score > k.minScore {
		return nil
	}
	if k.protector.IsProtected(id) {
		return nil
	}
	if err := k.peers.ClosePeer(id); err != nil {
		return fmt.Errorf("disconnecting peer %v: %w", id, err)
	}
	if err := k.blocker.BanPeer(id, k.clock.Now().Add(k.banDuration)); err != nil {
		return fmt.Errorf("banning peer %v: %w", id, err)
	}

	return nil
}

// background is intended to run as a separate go routine. It will call the supplied action function every checkInterval
// until the context is done.
func (k *PeerMonitor) background(action func() error) {
	defer k.bgTasks.Done()
	ticker := k.clock.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.Ch():
			if err := action(); err != nil {
				k.l.Warn("Error while checking connected peer score", "err", err)
			}
		}
	}
}
