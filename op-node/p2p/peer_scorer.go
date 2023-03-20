package p2p

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type scorer struct {
	peerStore           Peerstore
	metricer            GossipMetricer
	log                 log.Logger
	gater               PeerGater
	bandScoreThresholds BandScorer
}

// bandScoreThresholds holds the thresholds for classifying peers
// into different score bands.
type bandScoreThresholds struct {
	bands      map[string]float64
	lowestBand string
}

// BandScorer is an interface for placing peer scores
// into various bands.
//
// Implementations are expected to construct internals using the
// [Parse] function and then expose the [Bucket] function for
// downstream [BandScorer] consumers.
//
//go:generate mockery --name BandScorer --output mocks/
type BandScorer interface {
	Parse(str string) error
	Bucket(score float64) string
	Reset()
}

// NewBandScorer constructs a new [BandScorer] instance.
func NewBandScorer() BandScorer {
	return &bandScoreThresholds{
		bands: make(map[string]float64),
	}
}

// Reset wipes the internal state of the [BandScorer].
func (s *bandScoreThresholds) Reset() {
	s.bands = make(map[string]float64)
}

// Parse creates a [BandScorer] from a given string.
func (s *bandScoreThresholds) Parse(str string) error {
	var lowestThreshold float64
	for i, band := range strings.Split(str, ";") {
		// Skip empty band strings.
		band := strings.TrimSpace(band)
		if band == "" {
			continue
		}
		split := strings.Split(band, ":")
		if len(split) != 2 {
			return fmt.Errorf("invalid score band: %s", band)
		}
		threshold, err := strconv.ParseFloat(split[0], 64)
		if err != nil {
			return err
		}
		s.bands[split[1]] = threshold
		if threshold < lowestThreshold || i == 0 {
			s.lowestBand = split[1]
			lowestThreshold = threshold
		}
	}
	return nil
}

// Bucket returns the appropriate band for a given score.
func (s *bandScoreThresholds) Bucket(score float64) string {
	for band, threshold := range s.bands {
		if score >= threshold {
			return band
		}
	}
	// If there is no band threshold lower than the score,
	// the peer must be placed in the lowest bucket.
	return s.lowestBand
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
func NewScorer(peerGater PeerGater, peerStore Peerstore, metricer GossipMetricer, bandScoreThresholds BandScorer, log log.Logger) Scorer {
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
		// Reset the score bands
		s.bandScoreThresholds.Reset()

		// First clear the peer score bands
		scoreMap := make(map[string]float64)
		for id, snap := range m {
			// Increment the bucket for the peer's score
			band := s.bandScoreThresholds.Bucket(snap.Score)
			scoreMap[band] += 1

			// Update with the peer gater
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
