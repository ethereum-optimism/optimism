package p2p

import (
	log "github.com/ethereum/go-ethereum/log"
	peer "github.com/libp2p/go-libp2p/core/peer"
	slices "golang.org/x/exp/slices"
)

// ConnectionFactor is the factor by which we multiply the connection score.
const ConnectionFactor = -10

// PeerScoreThreshold is the threshold at which we block a peer.
const PeerScoreThreshold = -100

// gater is an internal implementation of the [PeerGater] interface.
type gater struct {
	connGater  ConnectionGater
	log        log.Logger
	banEnabled bool
}

// PeerGater manages the connection gating of peers.
//
//go:generate mockery --name PeerGater --output mocks/
type PeerGater interface {
	// Update handles a peer score update and blocks/unblocks the peer if necessary.
	Update(peer.ID, float64)
}

// NewPeerGater returns a new peer gater.
func NewPeerGater(connGater ConnectionGater, log log.Logger, banEnabled bool) PeerGater {
	return &gater{
		connGater:  connGater,
		log:        log,
		banEnabled: banEnabled,
	}
}

// Update handles a peer score update and blocks/unblocks the peer if necessary.
func (s *gater) Update(id peer.ID, score float64) {
	// Check if the peer score is below the threshold
	// If so, we need to block the peer
	if score < PeerScoreThreshold && s.banEnabled {
		s.log.Warn("peer blocking enabled, blocking peer", "id", id.String(), "score", score)
		err := s.connGater.BlockPeer(id)
		if err != nil {
			s.log.Warn("connection gater failed to block peer", "id", id.String(), "err", err)
		}
	}
	// Unblock peers whose score has recovered to an acceptable level
	if (score > PeerScoreThreshold) && slices.Contains(s.connGater.ListBlockedPeers(), id) {
		err := s.connGater.UnblockPeer(id)
		if err != nil {
			s.log.Warn("connection gater failed to unblock peer", "id", id.String(), "err", err)
		}
	}
}
