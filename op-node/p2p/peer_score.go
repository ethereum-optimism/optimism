package p2p

import (
	"math"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// AppScoring scores peers based on application-specific metrics.
func AppScoring(p peer.ID) float64 {
	return 0
}

// NewPeerScoreParams returns a default [pubsub.PeerScoreParams].
// See [PeerScoreParams] for detailed documentation.
//
// [PeerScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#PeerScoreParams
func NewPeerScoreParams() pubsub.PeerScoreParams {
	return pubsub.PeerScoreParams{
		SkipAtomicValidation:        false,
		Topics:                      make(map[string]*pubsub.TopicScoreParams),
		TopicScoreCap:               100, // Aggregate topic score cap (0 for no cap).
		AppSpecificScore:            AppScoring,
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -1,
		IPColocationFactorThreshold: 1,
		BehaviourPenaltyWeight:      -1,
		BehaviourPenaltyDecay:       0.999,
		DecayInterval:               24 * time.Hour,
		DecayToZero:                 0.001,
		RetainScore:                 math.MaxInt64, // We want to keep scores indefinitely - don't refresh on connect/disconnect
		SeenMsgTTL:                  0,             // Defaults to global TimeCacheDuration when 0
	}
}

// NewPeerScoreThresholds returns a default [pubsub.PeerScoreThresholds].
// See [PeerScoreThresholds] for detailed documentation.
//
// [PeerScoreThresholds]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#PeerScoreThresholds
func NewPeerScoreThresholds() pubsub.PeerScoreThresholds {
	return pubsub.PeerScoreThresholds{
		SkipAtomicValidation:        false,
		GossipThreshold:             -10,
		PublishThreshold:            -40,
		GraylistThreshold:           -40,
		AcceptPXThreshold:           20,
		OpportunisticGraftThreshold: 0.05,
	}
}
