package p2p

import (
	"fmt"
	"time"
)

var LightApplicationScoreParams = func(blockTime uint64) ApplicationScoreParams {
	slot := time.Duration(blockTime) * time.Second
	if slot == 0 {
		slot = 2 * time.Second
	}
	// We initialize an "epoch" as 6 blocks suggesting 6 blocks,
	// each taking ~ 2 seconds, is 12 seconds
	epoch := 6 * slot
	tenEpochs := 10 * epoch
	return ApplicationScoreParams{
		ValidResponseCap:      10,
		ValidResponseWeight:   1,
		ValidResponseDecay:    ScoreDecay(tenEpochs, slot),
		ErrorResponseCap:      10,
		ErrorResponseWeight:   -16,
		ErrorResponseDecay:    ScoreDecay(tenEpochs, slot),
		RejectedPayloadCap:    10,
		RejectedPayloadWeight: -50,
		RejectedPayloadDecay:  ScoreDecay(tenEpochs, slot),
		DecayToZero:           DecayToZero,
		DecayPeriod:           slot,
	}
}

var DisabledApplicationScoreParams = func(blockTime uint64) ApplicationScoreParams {
	slot := time.Duration(blockTime) * time.Second
	return ApplicationScoreParams{
		ValidResponseCap:      0,
		ValidResponseWeight:   0,
		ValidResponseDecay:    0,
		ErrorResponseCap:      0,
		ErrorResponseWeight:   0,
		ErrorResponseDecay:    0,
		RejectedPayloadCap:    0,
		RejectedPayloadWeight: 0,
		RejectedPayloadDecay:  0,
		DecayToZero:           DecayToZero,
		DecayPeriod:           slot,
	}
}

// ApplicationScoreParamsByName is a map of name to function that returns a [ApplicationScoringParams] based on the provided [rollup.Config].
var ApplicationScoreParamsByName = map[string]func(blockTime uint64) ApplicationScoreParams{
	"light": LightApplicationScoreParams,
	"none":  DisabledApplicationScoreParams,
}

// AvailableApplicationScoreParams returns a list of available application score params.
// These can be used as an input to [GetApplicationScoreParams] which returns the
// corresponding [pubsub.PeerScoreParams].
func AvailableApplicationScoreParams() []string {
	var params []string
	for name := range ApplicationScoreParamsByName {
		params = append(params, name)
	}
	return params
}

func GetApplicationScoreParams(name string, blockTime uint64) (ApplicationScoreParams, error) {
	params, ok := ApplicationScoreParamsByName[name]
	if !ok {
		return ApplicationScoreParams{}, fmt.Errorf("invalid params %s", name)
	}

	return params(blockTime), nil
}
