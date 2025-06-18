package proofs

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
)

const defaultTimeout = 20 * time.Minute

type Claim struct {
	t       devtest.T
	require *require.Assertions
	Index   int64
	claim   bindings.Claim
	game    *FaultDisputeGame
}

func newClaim(t devtest.T, require *require.Assertions, claimIndex int64, claim bindings.Claim, game *FaultDisputeGame) *Claim {
	return &Claim{
		t:       t,
		require: require,
		Index:   claimIndex,
		claim:   claim,
		game:    game,
	}
}

func (c *Claim) Depth() uint64 {
	return uint64(c.claim.Depth())
}

// WaitForCounterClaim waits for the claim to be countered by another claim being posted.
// Return the new claim that counters this claim.
func (c *Claim) WaitForCounterClaim() *Claim {
	counterIdx, counterClaim := c.game.waitForClaim(defaultTimeout, fmt.Sprintf("failed to find claim with parent idx %v", c.Index), func(claimIdx int64, claim bindings.Claim) bool {
		return int64(claim.ParentContractIndex) == c.Index
	})
	return newClaim(c.t, c.require, counterIdx, counterClaim, c.game)
}

func (c *Claim) Attack(eoa *dsl.EOA, newClaim common.Hash) *Claim {
	c.game.Attack(eoa, c.Index, newClaim)
	return c.WaitForCounterClaim()
}
