package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type scorer struct {
	peerStore           Peerstore
	metricer            GossipMetricer
	log                 log.Logger
	gater               PeerGater
	bandScoreThresholds *BandScoreThresholds
}

// scorePair holds a band and its corresponding threshold.
type scorePair struct {
	band      string
	threshold float64
}

// BandScoreThresholds holds the thresholds for classifying peers
// into different score bands.
type BandScoreThresholds struct {
	bands []scorePair
}

// NewBandScorer constructs a new [BandScoreThresholds] instance.
func NewBandScorer(str string) (*BandScoreThresholds, error) {
	s := &BandScoreThresholds{
		bands: make([]scorePair, 0),
	}

	for _, band := range strings.Split(str, ";") {
		// Skip empty band strings.
		band := strings.TrimSpace(band)
		if band == "" {
			continue
		}
		split := strings.Split(band, ":")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid score band: %s", band)
		}
		threshold, err := strconv.ParseFloat(split[0], 64)
		if err != nil {
			return nil, err
		}
		s.bands = append(s.bands, scorePair{
			band:      split[1],
			threshold: threshold,
		})
	}

	// Order the bands by threshold in ascending order.
	sort.Slice(s.bands, func(i, j int) bool {
		return s.bands[i].threshold < s.bands[j].threshold
	})

	return s, nil
}

// Bucket returns the appropriate band for a given score.
func (s *BandScoreThresholds) Bucket(score float64) string {
	for _, pair := range s.bands {
		if score <= pair.threshold {
			return pair.band
		}
	}
	// If there is no band threshold higher than the score,
	// the peer must be placed in the highest bucket.
	if len(s.bands) > 0 {
		return s.bands[len(s.bands)-1].band
	}
	return ""
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

	SetScore(peer.ID, store.ScoreType, float64) error
}

// Scorer is a peer scorer that scores peers based on application-specific metrics.
type Scorer interface {
	OnConnect()
	OnDisconnect()
	SnapshotHook() pubsub.ExtendedPeerScoreInspectFn
}

// NewScorer returns a new peer scorer.
func NewScorer(peerGater PeerGater, peerStore Peerstore, metricer GossipMetricer, bandScoreThresholds *BandScoreThresholds, log log.Logger) Scorer {
	return &scorer{
		peerStore:           peerStore,
		metricer:            metricer,
		log:                 log,
		gater:               peerGater,
		bandScoreThresholds: bandScoreThresholds,
	}
}

// SnapshotHook returns a function that is called periodically by the pubsub library to inspect the peer scores.
// It is passed into the pubsub library as a [pubsub.ExtendedPeerScoreInspectFn] in the [pubsub.WithPeerScoreInspect] option.
// The returned [pubsub.ExtendedPeerScoreInspectFn] is called with a mapping of peer IDs to peer score snapshots.
func (s *scorer) SnapshotHook() pubsub.ExtendedPeerScoreInspectFn {
	return func(m map[peer.ID]*pubsub.PeerScoreSnapshot) {
		scoreMap := make(map[string]float64)
		// Zero out all bands.
		for _, b := range s.bandScoreThresholds.bands {
			scoreMap[b.band] = 0
		}
		// Now set the new scores.
		for id, snap := range m {
			scores := make(map[store.ScoreType]float64)
			scores[store.TypeGossip] = snap.Score

			if err := s.peerStore.SetScore(id, store.TypeGossip, snap.Score); err != nil {
				s.log.Warn("Unable to update peer gossip score", "err", err)
			}
			band := s.bandScoreThresholds.Bucket(snap.Score)
			scoreMap[band] += 1
			s.gater.Update(id, snap.Score)
		}
		s.metricer.SetPeerScores(scoreMap)
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
