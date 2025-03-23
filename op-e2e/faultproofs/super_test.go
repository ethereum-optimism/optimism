package faultproofs

import (
	"context"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateSuperCannonGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	sys.L2IDs()
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	game.LogGameData(ctx)
}

func TestSuperCannonGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))
	correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
		c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
	})
	game.LogGameData(ctx)

	invalidRoot := game.RootClaim(ctx)
	honestCounter := invalidRoot.WaitForCounterClaim(ctx)
	game.ChallengeClaim(ctx, honestCounter, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// ordered top to bottom
		switch {
		case parent.IsOutputRoot(ctx):
			parent.RequireCorrectOutputRoot(ctx)
			return parent.Attack(ctx, common.Hash{0x01, 0xaa})
		case parent.IsBottomGameRoot(ctx):
			return parent.Attack(ctx, common.Hash{0x01, 0xaa})
		default:
			return parent.Defend(ctx, common.Hash{0x01, 0xaa})
		}
	}, func(idx int64) {
		correctTrace.StepFails(ctx, idx, false)
		correctTrace.StepFails(ctx, idx, true)
	})

	game.LogGameData(ctx)

	sys.AdvanceL1Time(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func TestSuperCannonGame_HonestCallsSteps(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGameWithCorrectRoot(ctx)
	game.LogGameData(ctx)

	correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
		c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
	})
	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

	rootAttack := correctTrace.AttackClaim(ctx, game.RootClaim(ctx))
	game.DefendClaim(ctx, rootAttack, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		switch {
		case parent.IsOutputRoot(ctx):
			parent.RequireCorrectOutputRoot(ctx)
			if parent.IsOutputRootLeaf(ctx) {
				return parent.Attack(ctx, common.Hash{0x01, 0xaa})
			} else {
				return correctTrace.DefendClaim(ctx, parent)
			}
		case parent.IsBottomGameRoot(ctx):
			return correctTrace.AttackClaim(ctx, parent)
		default:
			return correctTrace.DefendClaim(ctx, parent)
		}
	})
	game.LogGameData(ctx)

	sys.AdvanceL1Time(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
}
