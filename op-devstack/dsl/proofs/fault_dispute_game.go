package proofs

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	fTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
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

func (g *FaultDisputeGame) Addr() *common.Address {
	// Pull To address info from a TypedCall
	call := g.game.Status()
	test := call.Test()
	addr, err := call.To()
	test.Require().NoError(err, "Failed to retrieve address from game")

	return addr
}

func (g *FaultDisputeGame) MaxDepth() fTypes.Depth {
	maxGameDepth := contract.Read(g.game.MaxGameDepth()).Uint64()
	return fTypes.Depth(maxGameDepth)
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

func (g *FaultDisputeGame) Status() gTypes.GameStatus {
	status := contract.Read(g.game.Status())
	return gTypes.GameStatus(status)
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
	if err != nil { // Avoid waiting time capturing game data when there's no error
		g.require.NoErrorf(err, "%v\n%v", errorMsg, g.GameData())
	}

	return matchClaimIdx, matchedClaim
}

func (g *FaultDisputeGame) GameData() string {
	maxDepth := g.MaxDepth()
	splitDepth := g.SplitDepth()
	claims := g.allClaims()
	info := fmt.Sprintf("Claim count: %v\n", len(claims))
	for i, claim := range claims {
		pos := claim.Position
		info = info + fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v Trace Index: %v, ClaimHash: %v, Countered By: %v, ParentIndex: %v Claimant: %v Bond: %v %v\n",
			i, claim.Position.ToGIndex(), pos.Depth(), pos.IndexAtDepth(), pos.TraceIndex(maxDepth), claim.Value.Hex(), claim.CounteredBy, claim.ParentContractIndex, claim.Claimant, claim.Bond)
	}
	l2SequenceNumber := g.L2SequenceNumber()
	status := g.Status()
	return fmt.Sprintf("Game %v - %v - L2 Sequence Number: %v - Split Depth: %v - Max Depth: %v:\n%v\n",
		g.Addr(), status, l2SequenceNumber, splitDepth, maxDepth, info)
}
