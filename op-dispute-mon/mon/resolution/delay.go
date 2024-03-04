package resolution

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
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

func (d *DelayCalculator) RecordClaimResolutionDelayMax(games []*monTypes.EnrichedGameData) {
	var maxDelay uint64 = 0
	for _, game := range games {
		maxDelay = max(d.getMaxResolutionDelay(game), maxDelay)
	}
	d.metrics.RecordClaimResolutionDelayMax(float64(maxDelay))
}

func (d *DelayCalculator) getMaxResolutionDelay(game *monTypes.EnrichedGameData) uint64 {
	var maxDelay uint64 = 0
	for _, claim := range game.Claims {
		maxDelay = max(d.getOverflowTime(game.Duration, &claim), maxDelay)
	}
	return maxDelay
}

func (d *DelayCalculator) getOverflowTime(maxGameDuration uint64, claim *types.Claim) uint64 {
	// If the bond amount is the max uint128 value, the claim is resolved.
	if monTypes.ResolvedBondAmount.Cmp(claim.ClaimData.Bond) == 0 {
		return 0
	}
	maxChessTime := maxGameDuration / 2
	accumulatedTime := uint64(claim.ChessTime(d.clock.Now()))
	if accumulatedTime < maxChessTime {
		return 0
	}
	return accumulatedTime - maxChessTime
}
