package resolution

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

type DelayMetrics interface {
	RecordClaimResolutionDelayMax(delay float64)
	RecordClaimResolutionDelayMin(delay float64)
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

func (d *DelayCalculator) RecordResolutionDelays(games []*monTypes.EnrichedGameData) {
	var maxDelay uint64 = 0
	var minDelay uint64 = math.MaxUint64
	for _, game := range games {
		gameMin, gameMax := d.getResolutionDelays(game)
		maxDelay = max(gameMax, maxDelay)
		minDelay = min(gameMin, minDelay)
	}
	d.metrics.RecordClaimResolutionDelayMax(float64(maxDelay))
	d.metrics.RecordClaimResolutionDelayMin(float64(minDelay))
}

func (d *DelayCalculator) getResolutionDelays(game *monTypes.EnrichedGameData) (uint64, uint64) {
	var maxDelay uint64 = 0
	var minDelay uint64 = math.MaxUint64
	for _, claim := range game.Claims {
		// If the bond amount is the max uint128 value, the claim is resolved.
		// Don't include it in the delay calculation.
		if monTypes.ResolvedBondAmount.Cmp(claim.ClaimData.Bond) == 0 {
			continue
		}
		remainingTime := d.getRemainingTime(game.Duration, &claim)
		maxDelay = max(remainingTime, maxDelay)
		minDelay = min(remainingTime, minDelay)
	}
	return minDelay, maxDelay
}

func (d *DelayCalculator) getRemainingTime(maxGameDuration uint64, claim *types.Claim) uint64 {
	if claim.Clock == nil {
		return 0
	}
	maxChessTime := maxGameDuration / 2
	accumulatedTime := uint64(claim.ChessTime(d.clock.Now()))
	if accumulatedTime > maxChessTime {
		return 0
	}
	return maxChessTime - accumulatedTime
}
