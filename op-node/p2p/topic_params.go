package p2p

import (
	"fmt"
	"math"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// TODO: Update these parameters.
const (
	beaconBlockWeight  = 0.8
	meshWeight         = -0.717
	invalidDecayPeriod = 50 * epoch
	maxInMeshScore     = 10
	decayEpoch         = time.Duration(5)
)

// DefaultTopicScoreParams is a default instantiation of [pubsub.TopicScoreParams].
// See [TopicScoreParams] for detailed documentation.
// Default parameters are loosely based on prysm's default block topic scoring parameters.
// See [PrysmTopicScoringParams] for more details.
//
// [TopicScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#TopicScoreParams
// [PrysmTopicScoringParams]: https://github.com/prysmaticlabs/prysm/blob/develop/beacon-chain/p2p/gossip_scoring_params.go#L169
var DefaultTopicScoreParams = pubsub.TopicScoreParams{
	TopicWeight:                     beaconBlockWeight,
	TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
	TimeInMeshQuantum:               inMeshTime(),
	TimeInMeshCap:                   inMeshCap(),
	FirstMessageDeliveriesWeight:    1,
	FirstMessageDeliveriesDecay:     scoreDecay(20 * epoch),
	FirstMessageDeliveriesCap:       23,
	MeshMessageDeliveriesWeight:     meshWeight,
	MeshMessageDeliveriesDecay:      scoreDecay(decayEpoch * epoch),
	MeshMessageDeliveriesCap:        float64(uint64(epoch/slot) * uint64(decayEpoch)),
	MeshMessageDeliveriesThreshold:  float64(uint64(epoch/slot) * uint64(decayEpoch) / 10),
	MeshMessageDeliveriesWindow:     2 * time.Second,
	MeshMessageDeliveriesActivation: 4 * epoch,
	MeshFailurePenaltyWeight:        meshWeight,
	MeshFailurePenaltyDecay:         scoreDecay(decayEpoch * epoch),
	InvalidMessageDeliveriesWeight:  -140.4475,
	InvalidMessageDeliveriesDecay:   scoreDecay(invalidDecayPeriod),
}

// determines the decay rate from the provided time period till
// the decayToZero value. Ex: ( 1 -> 0.01)
func scoreDecay(duration time.Duration) float64 {
	numOfTimes := duration / slot
	return math.Pow(decayToZero, 1/float64(numOfTimes))
}

// denotes the unit time in mesh for scoring tallying.
func inMeshTime() time.Duration {
	return 1 * slot
}

// the cap for `inMesh` time scoring.
func inMeshCap() float64 {
	return float64((3600 * time.Second) / inMeshTime())
}

// DisabledTopicScoreParams is an instantiation of [pubsub.TopicScoreParams] where all scoring is disabled.
// See [TopicScoreParams] for detailed documentation.
//
// [TopicScoreParams]: https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub@v0.8.1#TopicScoreParams
var DisabledTopicScoreParams = pubsub.TopicScoreParams{
	TopicWeight:                     0, // disabled
	TimeInMeshWeight:                0, // disabled
	TimeInMeshQuantum:               inMeshTime(),
	TimeInMeshCap:                   inMeshCap(),
	FirstMessageDeliveriesWeight:    0, // disabled
	FirstMessageDeliveriesDecay:     scoreDecay(20 * epoch),
	FirstMessageDeliveriesCap:       23,
	MeshMessageDeliveriesWeight:     0, // disabled
	MeshMessageDeliveriesDecay:      scoreDecay(decayEpoch * epoch),
	MeshMessageDeliveriesCap:        float64(uint64(epoch/slot) * uint64(decayEpoch)),
	MeshMessageDeliveriesThreshold:  float64(uint64(epoch/slot) * uint64(decayEpoch) / 10),
	MeshMessageDeliveriesWindow:     2 * time.Second,
	MeshMessageDeliveriesActivation: 4 * epoch,
	MeshFailurePenaltyWeight:        0, // disabled
	MeshFailurePenaltyDecay:         scoreDecay(decayEpoch * epoch),
	InvalidMessageDeliveriesWeight:  0, // disabled
	InvalidMessageDeliveriesDecay:   scoreDecay(invalidDecayPeriod),
}

// TopicScoreParamsByName is a map of name to [pubsub.TopicScoreParams].
var TopicScoreParamsByName = map[string]pubsub.TopicScoreParams{
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
func GetTopicScoreParams(name string) (pubsub.TopicScoreParams, error) {
	params, ok := TopicScoreParamsByName[name]
	if !ok {
		return pubsub.TopicScoreParams{}, fmt.Errorf("invalid topic params %s", name)
	}

	return params, nil
}
