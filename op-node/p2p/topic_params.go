package p2p

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// MeshWeight is the weight of the mesh delivery topic.
const MeshWeight = -0.7

// MaxInMeshScore is the maximum score for being in the mesh.
const MaxInMeshScore = 10

// DecayEpoch is the number of epochs to decay the score over.
const DecayEpoch = time.Duration(5)

// DefaultTopicScoreParams is a default instantiation of [pubsub.TopicScoreParams].
// See [TopicScoreParams] for detailed documentation.
//
// [TopicScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#TopicScoreParams
var DefaultTopicScoreParams = func(cfg *rollup.Config) pubsub.TopicScoreParams {
	slot := time.Duration(cfg.BlockTime) * time.Second
	if slot == 0 {
		slot = 2 * time.Second
	}
	// TODO: tune these params
	// TODO: we initialize an "epoch" as 6 blocks suggesting 6 blocks, each taking ~ 2 seconds, is 12 seconds
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

// DisabledTopicScoreParams is an instantiation of [pubsub.TopicScoreParams] where all scoring is disabled.
// See [TopicScoreParams] for detailed documentation.
//
// [TopicScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#TopicScoreParams
var DisabledTopicScoreParams = func(cfg *rollup.Config) pubsub.TopicScoreParams {
	slot := time.Duration(cfg.BlockTime) * time.Second
	// TODO: tune these params
	// TODO: we initialize an "epoch" as 6 blocks suggesting 6 blocks, each taking ~ 2 seconds, is 12 seconds
	epoch := 6 * slot
	invalidDecayPeriod := 50 * epoch
	return pubsub.TopicScoreParams{
		TopicWeight:                     0, // disabled
		TimeInMeshWeight:                0, // disabled
		TimeInMeshQuantum:               slot,
		TimeInMeshCap:                   inMeshCap(slot),
		FirstMessageDeliveriesWeight:    0, // disabled
		FirstMessageDeliveriesDecay:     ScoreDecay(20*epoch, slot),
		FirstMessageDeliveriesCap:       23,
		MeshMessageDeliveriesWeight:     0, // disabled
		MeshMessageDeliveriesDecay:      ScoreDecay(DecayEpoch*epoch, slot),
		MeshMessageDeliveriesCap:        float64(uint64(epoch/slot) * uint64(DecayEpoch)),
		MeshMessageDeliveriesThreshold:  float64(uint64(epoch/slot) * uint64(DecayEpoch) / 10),
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 4 * epoch,
		MeshFailurePenaltyWeight:        0, // disabled
		MeshFailurePenaltyDecay:         ScoreDecay(DecayEpoch*epoch, slot),
		InvalidMessageDeliveriesWeight:  0, // disabled
		InvalidMessageDeliveriesDecay:   ScoreDecay(invalidDecayPeriod, slot),
	}
}

// TopicScoreParamsByName is a map of name to [pubsub.TopicScoreParams].
var TopicScoreParamsByName = map[string](func(*rollup.Config) pubsub.TopicScoreParams){
	"default":  DefaultTopicScoreParams,
	"disabled": DisabledTopicScoreParams,
}

// AvailableTopicScoreParams returns a list of available topic score params.
// These can be used as an input to [GetTopicScoreParams] which returns the
// corresponding [pubsub.TopicScoreParams].
func AvailableTopicScoreParams() []string {
	var params []string
	for name := range TopicScoreParamsByName {
		params = append(params, name)
	}
	return params
}

// GetTopicScoreParams returns the [pubsub.TopicScoreParams] for the given name.
func GetTopicScoreParams(name string, cfg *rollup.Config) (pubsub.TopicScoreParams, error) {
	params, ok := TopicScoreParamsByName[name]
	if !ok {
		return pubsub.TopicScoreParams{}, fmt.Errorf("invalid topic params %s", name)
	}

	return params(cfg), nil
}
