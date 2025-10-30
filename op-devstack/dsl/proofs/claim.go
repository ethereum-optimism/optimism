package proofs

import (
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
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
	Index   uint64
	claim   bindings.Claim
	game    *FaultDisputeGame
}

func newClaim(t devtest.T, require *require.Assertions, claimIndex uint64, claim bindings.Claim, game *FaultDisputeGame) *Claim {
	return &Claim{
		t:       t,
		require: require,
		Index:   claimIndex,
		claim:   claim,
		game:    game,
	}
}

func (c *Claim) String() string {
	pos := c.claim.Position
	return fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v ClaimHash: %v, Countered By: %v, ParentIndex: %v Claimant: %v Bond: %v\n",
		c.Index, pos.ToGIndex(), pos.Depth(), pos.IndexAtDepth(), c.claim.Value.Hex(), c.claim.CounteredBy, c.claim.ParentContractIndex, c.claim.Claimant, c.claim.Bond)
}

func (c *Claim) Value() common.Hash {
	return c.claim.Value
}

func (c *Claim) Bond() *big.Int {
	return c.claim.Bond
}

func (c *Claim) Position() types.Position {
	return c.claim.Position
}

func (c *Claim) Claimant() common.Address {
	return c.claim.Claimant
}

func (c *Claim) Depth() types.Depth {
	return c.claim.Depth()
}

func (c *Claim) AsChallengerClaim() types.Claim {
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    c.claim.Value,
			Bond:     c.claim.Bond,
			Position: c.claim.Position,
		},
		CounteredBy:         c.claim.CounteredBy,
		Claimant:            c.claim.Claimant,
		Clock:               c.claim.Clock,
		ContractIndex:       int(c.Index),
		ParentContractIndex: int(c.claim.ParentContractIndex),
	}
}

// WaitForCounterClaim waits for the claim to be countered by another claim being posted.
// Return the new claim that counters this claim.
func (c *Claim) WaitForCounterClaim(ignoreClaims ...*Claim) *Claim {
	counterIdx, counterClaim := c.game.waitForClaim(defaultTimeout, fmt.Sprintf("failed to find claim with parent idx %v", c.Index), func(claimIdx uint64, claim bindings.Claim) bool {
		return uint64(claim.ParentContractIndex) == c.Index && !containsClaim(claimIdx, ignoreClaims)
	})
	return newClaim(c.t, c.require, counterIdx, counterClaim, c.game)
}

func (c *Claim) VerifyNoCounterClaim() {
	for i, claim := range c.game.allClaims() {
		c.require.NotEqualValuesf(c.Index, claim.ParentContractIndex, "Found unexpected counter-claim at index %v: %v", i, claim)
	}
}

func (c *Claim) Attack(eoa *dsl.EOA, newClaim common.Hash) *Claim {
	c.game.Attack(eoa, c.Index, newClaim)
	return c.WaitForCounterClaim()
}

func containsClaim(claimIdx uint64, haystack []*Claim) bool {
	return slices.ContainsFunc(haystack, func(candidate *Claim) bool {
		return candidate.Index == claimIdx
	})
}
