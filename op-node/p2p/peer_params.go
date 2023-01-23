package p2p

import (
	"fmt"
	"math"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// TODO: Update these parameters.
const (
	slot             = 2 * time.Second
	epoch            = 12 * time.Second
	tenEpochs        = 10 * epoch
	oneHundredEpochs = 100 * epoch
	decayToZero      = 0.01
)

// ScoreDecay returns the decay factor for a given duration.
func ScoreDecay(duration time.Duration) float64 {
	numOfTimes := duration / slot
	return math.Pow(decayToZero, 1/float64(numOfTimes))
}

// DefaultPeerScoreParams is a default instantiation of [pubsub.PeerScoreParams].
// See [PeerScoreParams] for detailed documentation.
// Default parameters are loosely based on prysm's peer scoring parameters.
// See [PrysmPeerScoringParams] for more details.
//
// [PeerScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#PeerScoreParams
// [PrysmPeerScoringParams]: https://github.com/prysmaticlabs/prysm/blob/develop/beacon-chain/p2p/gossip_scoring_params.go#L72
var DefaultPeerScoreParams = pubsub.PeerScoreParams{
	Topics:        make(map[string]*pubsub.TopicScoreParams),
	TopicScoreCap: 32.72,
	AppSpecificScore: func(p peer.ID) float64 {
		return 0
	},
	AppSpecificWeight:           1,
	IPColocationFactorWeight:    -35.11,
	IPColocationFactorThreshold: 10,
	IPColocationFactorWhitelist: nil,
	BehaviourPenaltyWeight:      -15.92,
	BehaviourPenaltyThreshold:   6,
	BehaviourPenaltyDecay:       ScoreDecay(tenEpochs),
	DecayInterval:               slot,
	DecayToZero:                 decayToZero,
	RetainScore:                 oneHundredEpochs,
}

// DisabledPeerScoreParams is an instantiation of [pubsub.PeerScoreParams] where all scoring is disabled.
// See [PeerScoreParams] for detailed documentation.
//
// [PeerScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#PeerScoreParams
var DisabledPeerScoreParams = pubsub.PeerScoreParams{
	Topics:        make(map[string]*pubsub.TopicScoreParams),
	TopicScoreCap: 0, // 0 represent no cap
	AppSpecificScore: func(p peer.ID) float64 {
		return 0
	},
	AppSpecificWeight: 1,
	// ignore colocation scoring
	IPColocationFactorWeight:    0,
	IPColocationFactorWhitelist: nil,
	// 0 disables the behaviour penalty
	BehaviourPenaltyWeight: 0,
	BehaviourPenaltyDecay:  ScoreDecay(tenEpochs),
	DecayInterval:          slot,
	DecayToZero:            decayToZero,
	RetainScore:            oneHundredEpochs,
}

// PeerScoreParamsByName is a map of name to [pubsub.PeerScoreParams].
var PeerScoreParamsByName = map[string]pubsub.PeerScoreParams{
	"default":  DefaultPeerScoreParams,
	"disabled": DisabledPeerScoreParams,
}

// AvailablePeerScoreParams returns a list of available peer score params.
// These can be used as an input to [GetPeerScoreParams] which returns the
// corresponding [pubsub.PeerScoreParams].
func AvailablePeerScoreParams() []string {
	var params []string
	for name := range PeerScoreParamsByName {
		params = append(params, name)
	}
	return params
}

// GetPeerScoreParams returns the [pubsub.PeerScoreParams] for the given name.
func GetPeerScoreParams(name string) (pubsub.PeerScoreParams, error) {
	params, ok := PeerScoreParamsByName[name]
	if !ok {
		return pubsub.PeerScoreParams{}, fmt.Errorf("invalid params %s", name)
	}

	return params, nil
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
