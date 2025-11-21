package mon

import (
	"math/big"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const MaximumResolutionResponseBuffer = time.Minute

type RClock interface {
	Now() time.Time
}

type ClaimMetrics interface {
	RecordClaims(statuses *metrics.ClaimStatuses)
	RecordHonestActorClaims(address common.Address, data *metrics.HonestActorData)
}

type ClaimMonitor struct {
	logger       log.Logger
	clock        RClock
	honestActors types.HonestActors
	metrics      ClaimMetrics
}

func NewClaimMonitor(logger log.Logger, clock RClock, honestActors types.HonestActors, metrics ClaimMetrics) *ClaimMonitor {
	return &ClaimMonitor{logger, clock, honestActors, metrics}
}

func (c *ClaimMonitor) CheckClaims(games []*types.EnrichedGameData) {
	claimStatuses := &metrics.ClaimStatuses{}
	honest := make(map[common.Address]*metrics.HonestActorData)
	for actor := range c.honestActors {
		honest[actor] = &metrics.HonestActorData{
			PendingBonds: big.NewInt(0),
			LostBonds:    big.NewInt(0),
			WonBonds:     big.NewInt(0),
		}
	}
	for _, game := range games {
		c.checkGameClaims(game, claimStatuses, honest)
	}
	c.metrics.RecordClaims(claimStatuses)
	for actor := range c.honestActors {
		c.metrics.RecordHonestActorClaims(actor, honest[actor])
	}
}

func (c *ClaimMonitor) checkUpdateHonestActorStats(proxy common.Address, claim *types.EnrichedClaim, honest map[common.Address]*metrics.HonestActorData) {
	if !claim.Resolved {
		if c.honestActors[claim.Claimant] {
			honest[claim.Claimant].PendingClaimCount++
			honest[claim.Claimant].PendingBonds = new(big.Int).Add(honest[claim.Claimant].PendingBonds, claim.Bond)
		}
		return
	}
	if c.honestActors[claim.Claimant] {
		actor := claim.Claimant
		if claim.CounteredBy != (common.Address{}) {
			honest[actor].InvalidClaimCount++
			honest[actor].LostBonds = new(big.Int).Add(honest[actor].LostBonds, claim.Bond)
			c.logger.Error("Claim resolved against honest actor", "game", proxy, "honestActor", actor, "counteredBy", claim.CounteredBy, "claimContractIndex", claim.ContractIndex, "bondAmount", claim.Bond)
		} else {
			honest[actor].ValidClaimCount++
			// Note that we don't count refunded bonds as won
		}
	}
	if c.honestActors[claim.CounteredBy] {
		honest[claim.CounteredBy].WonBonds = new(big.Int).Add(honest[claim.CounteredBy].WonBonds, claim.Bond)
	}
}

func (c *ClaimMonitor) checkGameClaims(
	game *types.EnrichedGameData,
	claimStatuses *metrics.ClaimStatuses,
	honest map[common.Address]*metrics.HonestActorData,
) {
	// Check if the game is in the first half
	now := c.clock.Now()
	duration := uint64(now.Unix()) - game.Timestamp
	firstHalf := duration <= game.MaxClockDuration

	minDescendantAccumulatedTimeByIndex := make(map[int]time.Duration)

	// Iterate over the game's claims
	// Reverse order so we can track whether the claim has unresolvable children
	for i := len(game.Claims) - 1; i >= 0; i-- {
		claim := game.Claims[i]
		c.checkUpdateHonestActorStats(game.Proxy, &claim, honest)

		// Check if the clock has expired
		if firstHalf && claim.Resolved {
			c.logger.Error("Claim resolved in the first half of the game duration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex, "clock", duration)
		}

		maxChessTime := time.Duration(game.MaxClockDuration) * time.Second
		var parent faultTypes.Claim
		if !claim.IsRoot() {
			parent = game.Claims[claim.ParentContractIndex].Claim
		}
		accumulatedTime := faultTypes.ChessClock(now, claim.Claim, parent)

		// Calculate the minimum accumulated time of this claim or any of its descendants
		minAccumulatedTime, ok := minDescendantAccumulatedTimeByIndex[claim.ContractIndex]
		if !ok || accumulatedTime < minAccumulatedTime {
			minAccumulatedTime = accumulatedTime
		}
		// Update the minimum accumulated time for the parent claim to include this claim's time.
		curr, ok := minDescendantAccumulatedTimeByIndex[claim.ParentContractIndex]
		if !ok || minAccumulatedTime < curr {
			minDescendantAccumulatedTimeByIndex[claim.ParentContractIndex] = minAccumulatedTime
		}

		// Our clock is expired based on this claim accumulated time (can any more counter claims be posted)
		clockExpired := accumulatedTime >= maxChessTime
		// This claim is only resolvable if it and all it's descendants have expired clocks
		resolvable := minAccumulatedTime >= maxChessTime

		claimStatuses.RecordClaim(firstHalf, clockExpired, resolvable, claim.Resolved)
		if !claim.Resolved && resolvable {
			// SAFETY: minAccumulatedTime must be larger than or equal to maxChessTime since the claim is resolvable
			overflow := minAccumulatedTime - maxChessTime
			if overflow >= MaximumResolutionResponseBuffer {
				c.logger.Warn("Claim unresolved after clock expiration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex, "delay", overflow)
			}
		}
	}
}
