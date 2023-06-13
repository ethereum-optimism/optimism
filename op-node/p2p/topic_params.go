package p2p

import (
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// MeshWeight is the weight of the mesh delivery topic.
const MeshWeight = -0.7

// MaxInMeshScore is the maximum score for being in the mesh.
const MaxInMeshScore = 10

// DecayEpoch is the number of epochs to decay the score over.
const DecayEpoch = time.Duration(5)

// LightTopicScoreParams is a default instantiation of [pubsub.TopicScoreParams].
// See [TopicScoreParams] for detailed documentation.
//
// [TopicScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#TopicScoreParams
var LightTopicScoreParams = func(blockTime uint64) pubsub.TopicScoreParams {
	slot := time.Duration(blockTime) * time.Second
	if slot == 0 {
		slot = 2 * time.Second
	}
	epoch := 6 * slot
	invalidDecayPeriod := 50 * epoch
	return pubsub.TopicScoreParams{
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
	}
}

// the cap for `inMesh` time scoring.
func inMeshCap(slot time.Duration) float64 {
	return float64((3600 * time.Second) / slot)
}
