package proofs

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

type FaultDisputeGame struct {
	t       devtest.T
	require *require.Assertions
	game    *bindings.FaultDisputeGame
}

func NewFaultDisputeGame(t devtest.T, require *require.Assertions, game *bindings.FaultDisputeGame) *FaultDisputeGame {
	return &FaultDisputeGame{
		t:       t,
		require: require,
		game:    game,
	}
}

func (g *FaultDisputeGame) MaxDepth() uint64 {
	return contract.Read(g.game.MaxGameDepth()).Uint64()
}

func (g *FaultDisputeGame) SplitDepth() uint64 {
	return contract.Read(g.game.SplitDepth()).Uint64()
}

func (g *FaultDisputeGame) RootClaim() *Claim {
	return g.ClaimAtIndex(int64(0))
}

func (g *FaultDisputeGame) L2SequenceNumber() *big.Int {
	return contract.Read(g.game.L2SequenceNumber())
}

func (g *FaultDisputeGame) ClaimAtIndex(claimIndex int64) *Claim {
	claim := g.claimAtIndex(claimIndex)
	return g.newClaim(claimIndex, claim)
}

func (g *FaultDisputeGame) Attack(eoa *dsl.EOA, claimIdx int64, newClaim common.Hash) {
	claim := g.claimAtIndex(claimIdx)
	g.t.Logf("Attacking claim %v (depth: %d) with counter-claim %v", claimIdx, claim.Position.Depth(), newClaim)

	newPosition := claim.Position.Attack().ToGIndex()
	requiredBond := contract.Read(g.game.GetRequiredBond((*bindings.Uint128)(newPosition)))

	attackCall := g.game.Attack(claim.Value, big.NewInt(claimIdx), newClaim)

	receipt := contract.Write(eoa, attackCall, txplan.WithValue(requiredBond), txplan.WithGasRatio(2))
	g.t.Require().Equal(receipt.Status, types.ReceiptStatusSuccessful)
}

func (g *FaultDisputeGame) newClaim(claimIndex int64, claim bindings.Claim) *Claim {
	return newClaim(g.t, g.require, claimIndex, claim, g)
}

func (g *FaultDisputeGame) claimAtIndex(claimIndex int64) bindings.Claim {
	return contract.Read(g.game.ClaimData(big.NewInt(claimIndex))).Decode()
}

func (g *FaultDisputeGame) allClaims() []bindings.Claim {
	allClaimData := contract.ReadArray(g.game.ClaimDataLen(), func(i *big.Int) bindings.TypedCall[bindings.ClaimData] {
		return g.game.ClaimData(i)
	})

	// Decode claims
	var claims []bindings.Claim
	for _, claimData := range allClaimData {
		claims = append(claims, claimData.Decode())
	}

	return claims
}

func (g *FaultDisputeGame) waitForClaim(timeout time.Duration, errorMsg string, predicate func(claimIdx int64, claim bindings.Claim) bool) (int64, bindings.Claim) {
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), timeout)
	defer cancel()
	var matchedClaim bindings.Claim
	var matchClaimIdx int64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		claims := g.allClaims()
		// Search backwards because the new claims are at the end and more likely the ones we want.
		for i := len(claims) - 1; i >= 0; i-- {
			claim := claims[i]
			if predicate(int64(i), claim) {
				matchClaimIdx = int64(i)
				matchedClaim = claim
				return true, nil
			}
		}
		return false, nil
	})
	g.require.NoError(err, errorMsg)
	// TODO(#15948) - Log GameData()
	//if err != nil { // Avoid waiting time capturing game data when there's no error
	//	g.require.NoErrorf(err, "%v\n%v", errorMsg, g.GameData(ctx))
	//}
	return matchClaimIdx, matchedClaim
}
