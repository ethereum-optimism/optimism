package proofs

import (
	"context"
	"fmt"
	"math/big"
	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

type gameHelperProvider func(deployer *dsl.EOA) *GameHelper

type FaultDisputeGame struct {
	t              devtest.T
	require        *require.Assertions
	game           *bindings.FaultDisputeGame
	Address        common.Address
	helperProvider gameHelperProvider
}

func NewFaultDisputeGame(t devtest.T, require *require.Assertions, addr common.Address, helperProvider gameHelperProvider, game *bindings.FaultDisputeGame) *FaultDisputeGame {
	return &FaultDisputeGame{
		t:              t,
		require:        require,
		game:           game,
		Address:        addr,
		helperProvider: helperProvider,
	}
}

func (g *FaultDisputeGame) MaxDepth() challengerTypes.Depth {
	return challengerTypes.Depth(contract.Read(g.game.MaxGameDepth()).Uint64())
}

func (g *FaultDisputeGame) SplitDepth() uint64 {
	return contract.Read(g.game.SplitDepth()).Uint64()
}

func (g *FaultDisputeGame) RootClaim() *Claim {
	return g.ClaimAtIndex(0)
}

func (g *FaultDisputeGame) L2SequenceNumber() *big.Int {
	return contract.Read(g.game.L2SequenceNumber())
}

func (g *FaultDisputeGame) ClaimAtIndex(claimIndex uint64) *Claim {
	claim := g.claimAtIndex(claimIndex)
	return g.newClaim(claimIndex, claim)
}

func (g *FaultDisputeGame) Attack(eoa *dsl.EOA, claimIdx uint64, newClaim common.Hash) {
	claim := g.claimAtIndex(claimIdx)
	g.t.Logf("Attacking claim %v (depth: %d) with counter-claim %v", claimIdx, claim.Position.Depth(), newClaim)

	requiredBond := g.requiredBond(claim.Position.Attack())

	attackCall := g.game.Attack(claim.Value, new(big.Int).SetUint64(claimIdx), newClaim)

	receipt := contract.Write(eoa, attackCall, txplan.WithValue(requiredBond), txplan.WithGasRatio(2))
	g.t.Require().Equal(receipt.Status, types.ReceiptStatusSuccessful)
}

func (g *FaultDisputeGame) PerformMoves(eoa *dsl.EOA, moves ...GameHelperMove) []*Claim {
	return g.helperProvider(eoa).PerformMoves(eoa, g, moves)
}

func (g *FaultDisputeGame) requiredBond(pos challengerTypes.Position) eth.ETH {
	return eth.WeiBig(contract.Read(g.game.GetRequiredBond((*bindings.Uint128)(pos.ToGIndex()))))
}

func (g *FaultDisputeGame) status() gameTypes.GameStatus {
	status := contract.Read(g.game.Status())
	return gameTypes.GameStatus(status)
}

func (g *FaultDisputeGame) newClaim(claimIndex uint64, claim bindings.Claim) *Claim {
	return newClaim(g.t, g.require, claimIndex, claim, g)
}

func (g *FaultDisputeGame) claimAtIndex(claimIndex uint64) bindings.Claim {
	return contract.Read(g.game.ClaimData(new(big.Int).SetUint64(claimIndex))).Decode()
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

func (g *FaultDisputeGame) claimCount() uint64 {
	return contract.Read(g.game.ClaimDataLen()).Uint64()
}

func (g *FaultDisputeGame) waitForClaim(timeout time.Duration, errorMsg string, predicate func(claimIdx uint64, claim bindings.Claim) bool) (uint64, bindings.Claim) {
	timedCtx, cancel := context.WithTimeout(g.t.Ctx(), timeout)
	defer cancel()
	var matchedClaim bindings.Claim
	var matchClaimIdx uint64
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		claims := g.allClaims()
		// Search backwards because the new claims are at the end and more likely the ones we want.
		for i := len(claims) - 1; i >= 0; i-- {
			claim := claims[i]
			if predicate(uint64(i), claim) {
				matchClaimIdx = uint64(i)
				matchedClaim = claim
				return true, nil
			}
		}
		return false, nil
	})
	g.require.NoError(err, errorMsg)
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
		info = info + fmt.Sprintf("%v - Position: %v, Depth: %v, IndexAtDepth: %v Trace Index: %v, ClaimHash: %v, Countered By: %v, ParentIndex: %v Claimant: %v Bond: %v\n",
			i, claim.Position.ToGIndex(), pos.Depth(), pos.IndexAtDepth(), pos.TraceIndex(maxDepth), claim.Value.Hex(), claim.CounteredBy, claim.ParentContractIndex, claim.Claimant, claim.Bond)
	}
	seqNum := g.L2SequenceNumber()
	status := g.status()
	return fmt.Sprintf("Game %v - %v - L2 Block: %v - Split Depth: %v - Max Depth: %v:\n%v\n",
		g.Address, status, seqNum, splitDepth, maxDepth, info)
}

func (g *FaultDisputeGame) LogGameData() {
	g.t.Log(g.GameData())
}
