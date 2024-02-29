package disputegame

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type ClaimHelper struct {
	require     *require.Assertions
	game        *OutputGameHelper
	index       int64
	parentIndex uint32
	position    types.Position
	claim       common.Hash
}

func newClaimHelper(game *OutputGameHelper, idx int64, claim ContractClaim) *ClaimHelper {
	return &ClaimHelper{
		require:     game.require,
		game:        game,
		index:       idx,
		parentIndex: claim.ParentIndex,
		position:    types.NewPositionFromGIndex(claim.Position),
		claim:       claim.Claim,
	}
}

func (c *ClaimHelper) AgreesWithOutputRoot() bool {
	return c.position.Depth()%2 == 0
}

func (c *ClaimHelper) IsRootClaim() bool {
	return c.position.IsRootPosition()
}

func (c *ClaimHelper) IsOutputRoot(ctx context.Context) bool {
	splitDepth := c.game.SplitDepth(ctx)
	return c.position.Depth() <= splitDepth
}

func (c *ClaimHelper) IsOutputRootLeaf(ctx context.Context) bool {
	splitDepth := c.game.SplitDepth(ctx)
	return c.position.Depth() == splitDepth
}

func (c *ClaimHelper) IsBottomGameRoot(ctx context.Context) bool {
	splitDepth := c.game.SplitDepth(ctx)
	return c.position.Depth() == splitDepth+1
}

func (c *ClaimHelper) IsMaxDepth(ctx context.Context) bool {
	maxDepth := c.game.MaxDepth(ctx)
	return c.position.Depth() == maxDepth
}

func (c *ClaimHelper) Depth() types.Depth {
	return c.position.Depth()
}

// WaitForCounterClaim waits for the claim to be countered by another claim being posted.
// It returns a helper for the claim that countered this one.
func (c *ClaimHelper) WaitForCounterClaim(ctx context.Context, ignoreClaims ...*ClaimHelper) *ClaimHelper {
	timeout := defaultTimeout
	if c.IsOutputRootLeaf(ctx) {
		// This is the first claim we need to run cannon on, so give it more time
		timeout = timeout * 2
	}
	counterIdx, counterClaim := c.game.waitForClaim(ctx, timeout, fmt.Sprintf("failed to find claim with parent idx %v", c.index), func(claimIdx int64, claim ContractClaim) bool {
		return int64(claim.ParentIndex) == c.index && !containsClaim(claimIdx, ignoreClaims)
	})
	return newClaimHelper(c.game, counterIdx, counterClaim)
}

// WaitForCountered waits until the claim is countered either by a child claim or by a step call.
func (c *ClaimHelper) WaitForCountered(ctx context.Context) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		latestData := c.game.getClaim(ctx, c.index)
		return latestData.CounteredBy != common.Address{}, nil
	})
	if err != nil { // Avoid waiting time capturing game data when there's no error
		c.require.NoErrorf(err, "Claim %v was not countered\n%v", c.index, c.game.gameData(ctx))
	}
}

func (c *ClaimHelper) RequireCorrectOutputRoot(ctx context.Context) {
	c.require.True(c.IsOutputRoot(ctx), "Should not expect a valid output root in the bottom game")
	expected, err := c.game.correctOutputProvider.Get(ctx, c.position)
	c.require.NoError(err, "Failed to get correct output root")
	c.require.Equalf(expected, c.claim, "Should have correct output root in claim %v and position %v", c.index, c.position)
}

func (c *ClaimHelper) Attack(ctx context.Context, value common.Hash, opts ...MoveOpt) *ClaimHelper {
	c.game.Attack(ctx, c.index, value, opts...)
	return c.WaitForCounterClaim(ctx)
}

func (c *ClaimHelper) Defend(ctx context.Context, value common.Hash, opts ...MoveOpt) *ClaimHelper {
	c.game.Defend(ctx, c.index, value, opts...)
	return c.WaitForCounterClaim(ctx)
}

func (c *ClaimHelper) RequireDifferentClaimValue(other *ClaimHelper) {
	c.require.NotEqual(c.claim, other.claim, "should have posted different claims")
}

func (c *ClaimHelper) RequireOnlyCounteredBy(ctx context.Context, expected ...*ClaimHelper) {
	claims := c.game.getAllClaims(ctx)
	for idx, claim := range claims {
		if int64(claim.ParentIndex) != c.index {
			// Doesn't counter this claim, so ignore
			continue
		}
		if !containsClaim(int64(idx), expected) {
			// Found a countering claim not in the expected list. Fail.
			c.require.FailNowf("Found unexpected countering claim", "Parent claim index: %v Game state:\n%v", c.index, c.game.gameData(ctx))
		}
	}
}

func containsClaim(claimIdx int64, haystack []*ClaimHelper) bool {
	return slices.ContainsFunc(haystack, func(candidate *ClaimHelper) bool {
		return candidate.index == claimIdx
	})
}
