package p2p

import (
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type scorer struct {
	peerStore Peerstore
	metricer  GossipMetricer
	log       log.Logger
	gater     PeerGater
}

// Peerstore is a subset of the libp2p peerstore.Peerstore interface.
//
//go:generate mockery --name Peerstore --output mocks/
type Peerstore interface {
	// PeerInfo returns a peer.PeerInfo struct for given peer.ID.
	// This is a small slice of the information Peerstore has on
	// that peer, useful to other services.
	PeerInfo(peer.ID) peer.AddrInfo

	// Peers returns all of the peer IDs stored across all inner stores.
	Peers() peer.IDSlice
}

// Scorer is a peer scorer that scores peers based on application-specific metrics.
type Scorer interface {
	OnConnect()
	OnDisconnect()
	SnapshotHook() pubsub.ExtendedPeerScoreInspectFn
}

// NewScorer returns a new peer scorer.
func NewScorer(peerGater PeerGater, peerStore Peerstore, metricer GossipMetricer, log log.Logger) Scorer {
	return &scorer{
		peerStore: peerStore,
		metricer:  metricer,
		log:       log,
		gater:     peerGater,
	}
}

// SnapshotHook returns a function that is called periodically by the pubsub library to inspect the peer scores.
// It is passed into the pubsub library as a [pubsub.ExtendedPeerScoreInspectFn] in the [pubsub.WithPeerScoreInspect] option.
// The returned [pubsub.ExtendedPeerScoreInspectFn] is called with a mapping of peer IDs to peer score snapshots.
func (s *scorer) SnapshotHook() pubsub.ExtendedPeerScoreInspectFn {
	return func(m map[peer.ID]*pubsub.PeerScoreSnapshot) {
		for id, snap := range m {
			// Record peer score in the metricer
			s.metricer.RecordPeerScoring(id, snap.Score)

			// Update with the peer gater
			s.gater.Update(id, snap.Score)
		}
	}
}

// OnConnect is called when a peer connects.
// See [p2p.NotificationsMetricer] for invocation.
func (s *scorer) OnConnect() {
	// no-op
}

// OnDisconnect is called when a peer disconnects.
// See [p2p.NotificationsMetricer] for invocation.
func (s *scorer) OnDisconnect() {
	// no-op
}
