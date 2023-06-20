package p2p

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type scorer struct {
	peerStore Peerstore
	metricer  ScoreMetrics
	log       log.Logger
	cfg       *rollup.Config
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

	SetScore(id peer.ID, diff store.ScoreDiff) (store.PeerScores, error)
}

// Scorer is a peer scorer that scores peers based on application-specific metrics.
type Scorer interface {
	SnapshotHook() pubsub.ExtendedPeerScoreInspectFn
}

//go:generate mockery --name ScoreMetrics --output mocks/
type ScoreMetrics interface {
	SetPeerScores([]store.PeerScores)
}

// NewScorer returns a new peer scorer.
func NewScorer(cfg *rollup.Config, peerStore Peerstore, metricer ScoreMetrics, log log.Logger) Scorer {
	return &scorer{
		peerStore: peerStore,
		metricer:  metricer,
		log:       log,
		cfg:       cfg,
	}
}

// SnapshotHook returns a function that is called periodically by the pubsub library to inspect the gossip peer scores.
// It is passed into the pubsub library as a [pubsub.ExtendedPeerScoreInspectFn] in the [pubsub.WithPeerScoreInspect] option.
// The returned [pubsub.ExtendedPeerScoreInspectFn] is called with a mapping of peer IDs to peer score snapshots.
// The incoming peer score snapshots only contain gossip-score components.
func (s *scorer) SnapshotHook() pubsub.ExtendedPeerScoreInspectFn {
	blocksTopicName := blocksTopicV1(s.cfg)
	return func(m map[peer.ID]*pubsub.PeerScoreSnapshot) {
		allScores := make([]store.PeerScores, 0, len(m))
		// Now set the new scores.
		for id, snap := range m {
			diff := store.GossipScores{
				Total:              snap.Score,
				Blocks:             store.TopicScores{},
				IPColocationFactor: snap.IPColocationFactor,
				BehavioralPenalty:  snap.BehaviourPenalty,
			}
			if topSnap, ok := snap.Topics[blocksTopicName]; ok {
				diff.Blocks.TimeInMesh = float64(topSnap.TimeInMesh) / float64(time.Second)
				diff.Blocks.MeshMessageDeliveries = topSnap.MeshMessageDeliveries
				diff.Blocks.FirstMessageDeliveries = topSnap.FirstMessageDeliveries
				diff.Blocks.InvalidMessageDeliveries = topSnap.InvalidMessageDeliveries
			}
			if peerScores, err := s.peerStore.SetScore(id, &diff); err != nil {
				s.log.Warn("Unable to update peer gossip score", "err", err)
			} else {
				allScores = append(allScores, peerScores)
			}
		}
		s.metricer.SetPeerScores(allScores)
	}
}
