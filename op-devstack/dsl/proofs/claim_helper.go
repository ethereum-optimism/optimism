package proofs

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

const defaultTimeout = 20 * time.Minute

type ClaimHelper struct {
	t       devtest.T
	require *require.Assertions
	Index   int64
	claim   bindings.Claim
	game    *FaultDisputeGameHelper
}

func newClaimHelper(t devtest.T, require *require.Assertions, claimIndex int64, claim bindings.Claim, game *FaultDisputeGameHelper) *ClaimHelper {
	return &ClaimHelper{
		t:       t,
		require: require,
		Index:   claimIndex,
		claim:   claim,
		game:    game,
	}
}

// WaitForCounterClaim waits for the claim to be countered by another claim being posted.
// It returns a helper for the claim that countered this one.
func (c *ClaimHelper) WaitForCounterClaim() *ClaimHelper {
	counterIdx, counterClaim := c.game.waitForClaim(defaultTimeout, fmt.Sprintf("failed to find claim with parent idx %v", c.Index), func(claimIdx int64, claim bindings.Claim) bool {
		return int64(claim.ParentContractIndex) == c.Index
	})
	return newClaimHelper(c.t, c.require, counterIdx, counterClaim, c.game)
}
