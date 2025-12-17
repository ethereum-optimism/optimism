package p2p

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type scorer struct {
	peerStore Peerstore
	metricer  ScoreMetrics
	appScorer ApplicationScorer
	log       log.Logger
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
	ApplicationScore(peer.ID) float64
}

//go:generate mockery --name ScoreMetrics --output mocks/
type ScoreMetrics interface {
	SetPeerScores([]store.PeerScores)
}

// NewScorer returns a new peer scorer.
func NewScorer(peerStore Peerstore, metricer ScoreMetrics, appScorer ApplicationScorer, log log.Logger) Scorer {
	return &scorer{
		peerStore: peerStore,
		metricer:  metricer,
		appScorer: appScorer,
		log:       log,
	}
}

// SnapshotHook returns a function that is called periodically by the pubsub library to inspect the gossip peer scores.
// It is passed into the pubsub library as a [pubsub.ExtendedPeerScoreInspectFn] in the [pubsub.WithPeerScoreInspect] option.
// The returned [pubsub.ExtendedPeerScoreInspectFn] is called with a mapping of peer IDs to peer score snapshots.
// The incoming peer score snapshots only contain gossip-score components.
func (s *scorer) SnapshotHook() pubsub.ExtendedPeerScoreInspectFn {
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
			// All allow-listed topics are block-topics,
			// And the total performance is what we care about, regardless of number of past forks.
			// So add up the data. And consider the time-in-mesh of the most used topic:
			// alt CL implementations may choose to not stay on legacy topics.
			for _, topSnap := range snap.Topics {
				diff.Blocks.TimeInMesh = max(diff.Blocks.TimeInMesh, float64(topSnap.TimeInMesh)/float64(time.Second))
				diff.Blocks.MeshMessageDeliveries += topSnap.MeshMessageDeliveries
				diff.Blocks.FirstMessageDeliveries += topSnap.FirstMessageDeliveries
				diff.Blocks.InvalidMessageDeliveries += topSnap.InvalidMessageDeliveries
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

func (s *scorer) ApplicationScore(id peer.ID) float64 {
	return s.appScorer.ApplicationScore(id)
}
