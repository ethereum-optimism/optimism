package mon

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/metrics"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type RClock interface {
	Now() time.Time
}

type ClaimMetrics interface {
	RecordClaims(status metrics.ClaimStatus, count int)
}

type ClaimMonitor struct {
	logger  log.Logger
	clock   RClock
	metrics ClaimMetrics
}

func NewClaimMonitor(logger log.Logger, clock RClock, metrics ClaimMetrics) *ClaimMonitor {
	return &ClaimMonitor{logger, clock, metrics}
}

func (c *ClaimMonitor) CheckClaims(games []*types.EnrichedGameData) {
	claimStatus := make(map[metrics.ClaimStatus]int)
	for _, game := range games {
		c.checkGameClaims(game, claimStatus)
	}
	for status, count := range claimStatus {
		c.metrics.RecordClaims(status, count)
	}
}

func (c *ClaimMonitor) checkGameClaims(game *types.EnrichedGameData, claimStatus map[metrics.ClaimStatus]int) {
	// Check if the game is in the first half
	duration := uint64(c.clock.Now().Unix()) - game.Timestamp
	firstHalf := duration <= (game.Duration / 2)

	// Iterate over the game's claims
	for _, claim := range game.Claims {
		// Check if the clock has expired
		if firstHalf && claim.Resolved {
			c.logger.Error("Claim resolved in the first half of the game duration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex)
		}

		maxChessTime := time.Duration(game.Duration/2) * time.Second
		accumulatedTime := claim.ChessTime(c.clock.Now())
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
				c.logger.Warn("Claim unresolved after clock expiration", "game", game.Proxy, "claimContractIndex", claim.ContractIndex)
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
