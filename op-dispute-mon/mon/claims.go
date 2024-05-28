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
	RecordClaims(status metrics.ClaimStatus, count int)
	RecordHonestActorClaims(address common.Address, data *metrics.HonestActorData)
}

type ClaimMonitor struct {
	logger       log.Logger
	clock        RClock
	honestActors map[common.Address]bool // Map for efficient lookup
	metrics      ClaimMetrics
}

func NewClaimMonitor(logger log.Logger, clock RClock, honestActors []common.Address, metrics ClaimMetrics) *ClaimMonitor {
	actors := make(map[common.Address]bool)
	for _, actor := range honestActors {
		actors[actor] = true
	}
	return &ClaimMonitor{logger, clock, actors, metrics}
}

func (c *ClaimMonitor) CheckClaims(games []*types.EnrichedGameData) {
	claimStatus := metrics.ZeroClaimStatuses()
	honest := make(map[common.Address]*metrics.HonestActorData)
	for actor := range c.honestActors {
		honest[actor] = &metrics.HonestActorData{
			PendingBonds: big.NewInt(0),
			LostBonds:    big.NewInt(0),
			WonBonds:     big.NewInt(0),
		}
	}
	for _, game := range games {
		c.checkGameClaims(game, claimStatus, honest)
	}
	for status, count := range claimStatus {
		c.metrics.RecordClaims(status, count)
	}
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
	claimStatus map[metrics.ClaimStatus]int,
	honest map[common.Address]*metrics.HonestActorData,
) {
	// Check if the game is in the first half
	duration := uint64(c.clock.Now().Unix()) - game.Timestamp
	firstHalf := duration <= game.MaxClockDuration

	// Iterate over the game's claims
	for _, claim := range game.Claims {
		c.checkUpdateHonestActorStats(game.Proxy, &claim, honest)

		// Check if the clock has expired
		if firstHalf && claim.Resolved {
			c.logger.Error("Claim resolved in the first half of the game duration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex, "id", claim.ID(), "clock", duration)
		}

		maxChessTime := time.Duration(game.MaxClockDuration) * time.Second
		var parent faultTypes.Claim
		if !claim.IsRoot() {
			parent = game.Claims[claim.ParentContractIndex].Claim
		}
		accumulatedTime := faultTypes.ChessClock(c.clock.Now(), claim.Claim, parent)
		clockExpired := accumulatedTime >= maxChessTime

		if claim.Resolved {
			if clockExpired {
				if firstHalf {
					claimStatus[metrics.FirstHalfExpiredResolved]++
				} else {
					claimStatus[metrics.SecondHalfExpiredResolved]++
				}
			} else {
				if firstHalf {
					claimStatus[metrics.FirstHalfNotExpiredResolved]++
				} else {
					claimStatus[metrics.SecondHalfNotExpiredResolved]++
				}
			}
		} else {
			if clockExpired {
				// SAFETY: accumulatedTime must be larger than or equal to maxChessTime since clockExpired
				overflow := accumulatedTime - maxChessTime
				if overflow >= MaximumResolutionResponseBuffer {
					c.logger.Warn("Claim unresolved after clock expiration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex, "delay", overflow)
				}
				if firstHalf {
					claimStatus[metrics.FirstHalfExpiredUnresolved]++
				} else {
					claimStatus[metrics.SecondHalfExpiredUnresolved]++
				}
			} else {
				if firstHalf {
					claimStatus[metrics.FirstHalfNotExpiredUnresolved]++
				} else {
					claimStatus[metrics.SecondHalfNotExpiredUnresolved]++
				}
			}
		}
	}
}
