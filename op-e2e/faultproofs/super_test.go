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
	testCannonGame(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonGame_ChallengeAllZeroClaim(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	testCannonChallengeAllZeroClaim(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonDefendStep(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	testCannonDefendStep(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonPoisonedPostState(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	testCannonPoisonedPostState(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonRootBeyondProposedBlock_ValidRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGameWithCorrectRoot(ctx)
	testDisputeRootBeyondProposedBlockValidOutputRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonRootBeyondProposedBlock_InvalidRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	testDisputeRootBeyondProposedBlockInvalidOutputRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
}

func TestSuperCannonRootChangeClaimedRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	testDisputeRootChangeClaimedRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
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
