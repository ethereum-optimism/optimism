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

type FaultDisputeGameHelper struct {
	t       devtest.T
	require *require.Assertions
	game    *bindings.FaultDisputeGame
}

func NewFaultDisputeGameHelper(t devtest.T, require *require.Assertions, game *bindings.FaultDisputeGame) *FaultDisputeGameHelper {
	return &FaultDisputeGameHelper{
		t:       t,
		require: require,
		game:    game,
	}
}

func (g *FaultDisputeGameHelper) GetMaxDepth() uint64 {
	return contract.Read(g.game.MaxGameDepth()).Uint64()
}

func (g *FaultDisputeGameHelper) GetSplitDepth() uint64 {
	return contract.Read(g.game.SplitDepth()).Uint64()
}

func (g *FaultDisputeGameHelper) GetRootClaim() *ClaimHelper {
	return g.GetClaimAtIndex(int64(0))
}

func (g *FaultDisputeGameHelper) GetClaimAtIndex(claimIndex int64) *ClaimHelper {
	claim := g.getClaimAtIndex(claimIndex)
	return g.newClaimHelper(claimIndex, claim)
}

func (g *FaultDisputeGameHelper) Attack(eoa *dsl.EOA, claimIdx int64, newClaim common.Hash) {
	claim := g.getClaimAtIndex(claimIdx)
	g.t.Logf("Attacking claim %v (depth: %d) with counter-claim %v", claimIdx, claim.Position.Depth(), newClaim)

	newPosition := claim.Position.Attack().ToGIndex()
	requiredBond := contract.Read(g.game.GetRequiredBond(newPosition))

	receipt := contract.Write(eoa, g.game.Attack(claim.Value, claim.Position.ToGIndex(), newClaim), txplan.WithValue(requiredBond))
	g.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

func (g *FaultDisputeGameHelper) newClaimHelper(claimIndex int64, claim bindings.Claim) *ClaimHelper {
	return newClaimHelper(g.t, g.require, claimIndex, claim, g)
}

func (g *FaultDisputeGameHelper) getClaimAtIndex(claimIndex int64) bindings.Claim {
	return contract.Read(g.game.ClaimData(big.NewInt(claimIndex))).Decode()
}

func (g *FaultDisputeGameHelper) getAllClaims() []bindings.Claim {
	// TODO - do we need to batch these? See: op-service/sources/batching.ReadArray
	claimCount := contract.Read(g.game.ClaimDataLen())
	var claims []bindings.Claim
	for i := int64(0); i < claimCount.Int64(); i++ {
		claim := g.getClaimAtIndex(i)
		claims = append(claims, claim)
	}

	return claims
}

func (g *FaultDisputeGameHelper) waitForClaim(timeout time.Duration, errorMsg string, predicate func(claimIdx int64, claim bindings.Claim) bool) (int64, bindings.Claim) {
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), timeout)
	defer cancel()
	var matchedClaim bindings.Claim
	var matchClaimIdx int64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		claims := g.getAllClaims()
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
	// TODO - copy over gameData logic
	//if err != nil { // Avoid waiting time capturing game data when there's no error
	//	g.require.NoErrorf(err, "%v\n%v", errorMsg, g.GameData(ctx))
	//}
	return matchClaimIdx, matchedClaim
}
