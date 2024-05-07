package mon

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ClaimCounterMetrics interface {
	RecordInvalidClaims(count int)
	RecordValidClaims(count int)
	RecordHonestActorValidClaimCount(count int)
	// Note: we don't need a non-honest actor  valid claim count since we can just
	//       subtract the honest actor valid claim count from the valid claim count.
}

type ClaimCounter struct {
	logger       log.Logger
	clock        RClock
	honestActors map[common.Address]bool // Map for efficient lookup
	validator    OutputValidator
	metrics      ClaimCounterMetrics
}

func NewClaimCounter(logger log.Logger, clock RClock, honestActors []common.Address, validator OutputValidator, metrics ClaimCounterMetrics) *ClaimCounter {
	actors := make(map[common.Address]bool)
	for _, actor := range honestActors {
		actors[actor] = true
	}
	return &ClaimCounter{logger, clock, actors, validator, metrics}
}

func (c *ClaimCounter) Count(ctx context.Context, games []*types.EnrichedGameData) {
	invalid, valid, honestValid := 0, 0, 0
	for _, game := range games {
		gameInvalid, gameValid, gameHonestValid, err := c.count(ctx, game)
		if err != nil {
			c.logger.Error("Failed to count game claims", "err", err)
		}
		invalid += gameInvalid
		valid += gameValid
		honestValid += gameHonestValid
	}
	c.metrics.RecordInvalidClaims(invalid)
	c.metrics.RecordValidClaims(valid)
	c.metrics.RecordHonestActorValidClaimCount(honestValid)
}

func (c *ClaimCounter) count(ctx context.Context, game *types.EnrichedGameData) (int, int, int, error) {
	invalid, valid, honestValid := 0, 0, 0

	agreement, _, err := c.validator.CheckRootAgreement(ctx, game.L1HeadNum, game.L2BlockNumber, game.RootClaim)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %w", ErrRootAgreement, err)
	}

	for _, claim := range game.Claims {
		if claim.Resolved {
			continue
		}
		if claim.Claim.Depth()%2 != 0 && agreement || claim.Claim.Depth()%2 == 0 && !agreement {
			invalid += 1
			if invalid >= 10 {
				c.logger.Warn("Encountered over 10 disagreed claims", "game", game.Proxy, "depth", claim.Claim.Depth(), "agreement", agreement, "honestActor", c.honestActors[claim.Claimant])
			}
			continue
		}
		if c.honestActors[claim.Claimant] {
			honestValid += 1
		}
		valid += 1
	}

	return invalid, valid, honestValid, nil
}
