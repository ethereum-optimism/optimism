package p2p

import (
	"fmt"
	"math"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DecayToZero is the decay factor for a peer's score to zero.
const DecayToZero = 0.01

// MeshWeight is the weight of the mesh delivery topic.
const MeshWeight = -0.7

// MaxInMeshScore is the maximum score for being in the mesh.
const MaxInMeshScore = 10

// DecayEpoch is the number of epochs to decay the score over.
const DecayEpoch = time.Duration(5)

// ScoreDecay returns the decay factor for a given duration.
func ScoreDecay(duration time.Duration, slot time.Duration) float64 {
	numOfTimes := duration / slot
	return math.Pow(DecayToZero, 1/float64(numOfTimes))
}

// LightPeerScoreParams is an instantiation of [pubsub.PeerScoreParams] with light penalties.
// See [PeerScoreParams] for detailed documentation.
//
// [PeerScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#PeerScoreParams
func LightPeerScoreParams(cfg *rollup.Config) pubsub.PeerScoreParams {
	slot := time.Duration(cfg.BlockTime) * time.Second
	if slot == 0 {
		slot = 2 * time.Second
	}
	// We initialize an "epoch" as 6 blocks suggesting 6 blocks,
	// each taking ~ 2 seconds, is 12 seconds
	epoch := 6 * slot
	tenEpochs := 10 * epoch
	oneHundredEpochs := 100 * epoch
	invalidDecayPeriod := 50 * epoch
	return pubsub.PeerScoreParams{
		Topics: map[string]*pubsub.TopicScoreParams{
			blocksTopicV1(cfg): {
				TopicWeight:                     0.8,
				TimeInMeshWeight:                MaxInMeshScore / inMeshCap(slot),
				TimeInMeshQuantum:               slot,
				TimeInMeshCap:                   inMeshCap(slot),
				FirstMessageDeliveriesWeight:    1,
				FirstMessageDeliveriesDecay:     ScoreDecay(20*epoch, slot),
				FirstMessageDeliveriesCap:       23,
				MeshMessageDeliveriesWeight:     MeshWeight,
				MeshMessageDeliveriesDecay:      ScoreDecay(DecayEpoch*epoch, slot),
				MeshMessageDeliveriesCap:        float64(uint64(epoch/slot) * uint64(DecayEpoch)),
				MeshMessageDeliveriesThreshold:  float64(uint64(epoch/slot) * uint64(DecayEpoch) / 10),
				MeshMessageDeliveriesWindow:     2 * time.Second,
				MeshMessageDeliveriesActivation: 4 * epoch,
				MeshFailurePenaltyWeight:        MeshWeight,
				MeshFailurePenaltyDecay:         ScoreDecay(DecayEpoch*epoch, slot),
				InvalidMessageDeliveriesWeight:  -140.4475,
				InvalidMessageDeliveriesDecay:   ScoreDecay(invalidDecayPeriod, slot),
			},
		},
		TopicScoreCap: 34,
		AppSpecificScore: func(p peer.ID) float64 {
			return 0
		},
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -35,
		IPColocationFactorThreshold: 10,
		IPColocationFactorWhitelist: nil,
		BehaviourPenaltyWeight:      -16,
		BehaviourPenaltyThreshold:   6,
		BehaviourPenaltyDecay:       ScoreDecay(tenEpochs, slot),
		DecayInterval:               slot,
		DecayToZero:                 DecayToZero,
		RetainScore:                 oneHundredEpochs,
	}
}

// the cap for `inMesh` time scoring.
func inMeshCap(slot time.Duration) float64 {
	return float64((3600 * time.Second) / slot)
}

func GetScoringParams(name string, cfg *rollup.Config) (*ScoringParams, error) {
	switch name {
	case "light":
		return &ScoringParams{
			PeerScoring:        LightPeerScoreParams(cfg),
			ApplicationScoring: LightApplicationScoreParams(cfg),
		}, nil
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown p2p scoring level: %v", name)
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
