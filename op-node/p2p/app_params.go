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
		// Max positive score from valid responses: 5
		ValidResponseCap:    10,
		ValidResponseWeight: 0.5,
		ValidResponseDecay:  ScoreDecay(tenEpochs, slot),

		// Takes 10 error responses to reach the default gossip threshold of -10
		// But at most we track 9. These errors include not supporting p2p sync
		// so we don't (yet) want to ignore gossip from a peer based on this measure alone.
		ErrorResponseCap:    9,
		ErrorResponseWeight: -1,
		ErrorResponseDecay:  ScoreDecay(tenEpochs, slot),

		// Takes 5 rejected payloads to reach the default ban threshold of -100
		RejectedPayloadCap:    20,
		RejectedPayloadWeight: -20,
		RejectedPayloadDecay:  ScoreDecay(tenEpochs, slot),

		DecayToZero:   DecayToZero,
		DecayInterval: slot,
	}
}
