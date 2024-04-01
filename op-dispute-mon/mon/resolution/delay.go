package resolution

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type DelayMetrics interface {
	RecordClaimResolutionDelayMax(delay float64)
}

type DelayCalculator struct {
	metrics DelayMetrics
	clock   clock.Clock
}

func NewDelayCalculator(metrics DelayMetrics, clock clock.Clock) *DelayCalculator {
	return &DelayCalculator{
		metrics: metrics,
		clock:   clock,
	}
}

func (d *DelayCalculator) RecordClaimResolutionDelayMax(games []*types.EnrichedGameData) {
	var maxDelay uint64 = 0
	for _, game := range games {
		maxDelay = max(d.getMaxResolutionDelay(game), maxDelay)
	}
	d.metrics.RecordClaimResolutionDelayMax(float64(maxDelay))
}

func (d *DelayCalculator) getMaxResolutionDelay(game *types.EnrichedGameData) uint64 {
	var maxDelay uint64 = 0
	for _, claim := range game.Claims {
		maxDelay = max(d.getOverflowTime(game.Duration, &claim), maxDelay)
	}
	return maxDelay
}

func (d *DelayCalculator) getOverflowTime(maxGameDuration uint64, claim *types.EnrichedClaim) uint64 {
	if claim.Resolved {
		return 0
	}
	maxChessTime := time.Duration(maxGameDuration/2) * time.Second
	accumulatedTime := claim.ChessTime(d.clock.Now())
	if accumulatedTime < maxChessTime {
		return 0
	}
	return uint64((accumulatedTime - maxChessTime).Seconds())
}
