package p2p

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type ApplicationScoreParams struct {
	ValidResponseCap    float64
	ValidResponseWeight float64
	ValidResponseDecay  float64

	ErrorResponseCap    float64
	ErrorResponseWeight float64
	ErrorResponseDecay  float64

	RejectedPayloadCap    float64
	RejectedPayloadWeight float64
	RejectedPayloadDecay  float64

	DecayToZero   float64
	DecayInterval time.Duration
}

func LightApplicationScoreParams(cfg *rollup.Config) ApplicationScoreParams {
	slot := time.Duration(cfg.BlockTime) * time.Second
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
		DecayInterval:         slot,
	}
}
