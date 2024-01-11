package faultproofs

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const outputCannonTestExecutor = 0

func TestOutputCannonGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 4, common.Hash{0x01})
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	game.LogGameData(ctx)

	// Challenger should post an output root to counter claims down to the leaf level of the top game
	claim := game.RootClaim(ctx)
	for claim.IsOutputRoot(ctx) && !claim.IsOutputRootLeaf(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim agrees with the output root, expect the honest challenger to counter it
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
			claim.RequireCorrectOutputRoot(ctx)
		} else {
			// Otherwise we should counter
			claim = claim.Attack(ctx, common.Hash{0xaa})
			game.LogGameData(ctx)
		}
	}

	// Wait for the challenger to post the first claim in the cannon trace
	claim = claim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)

	// Attack the root of the cannon trace subgame
	claim = claim.Attack(ctx, common.Hash{0x00, 0xcc})
	for !claim.IsMaxDepth(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim supports the output root, wait for the honest challenger to respond
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
		} else {
			// Otherwise we need to counter the honest claim
			claim = claim.Defend(ctx, common.Hash{0x00, 0xdd})
			game.LogGameData(ctx)
		}
	}
	// Challenger should be able to call step and counter the leaf claim.
	claim.WaitForCountered(ctx)
	game.LogGameData(ctx)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
}

func TestOutputCannon_ChallengeAllZeroClaim(t *testing.T) {
	// The dishonest actor always posts claims with all zeros.
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 3, common.Hash{})
	game.LogGameData(ctx)

	claim := game.DisputeLastBlock(ctx)
	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	game.DefendClaim(ctx, claim, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		if parent.IsBottomGameRoot(ctx) {
			return parent.Attack(ctx, common.Hash{})
		}
		return parent.Defend(ctx, common.Hash{})
	})

	game.LogGameData(ctx)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}

func TestOutputCannon_PublishCannonRootClaim(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	tests := []struct {
		disputeL2BlockNumber uint64
	}{
		{7}, // Post-state output root is invalid
		{8}, // Post-state output root is valid
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Dispute_%v", test.disputeL2BlockNumber), func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
			ctx := context.Background()
			sys, _ := startFaultDisputeSystem(t)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", test.disputeL2BlockNumber, common.Hash{0x01})
			game.DisputeLastBlock(ctx)
			game.LogGameData(ctx)

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

			splitDepth := game.SplitDepth(ctx)
			game.WaitForClaimAtDepth(ctx, splitDepth+1)
		})
	}
}

func TestOutputCannonDisputeGame(t *testing.T) {
	executor := uint64(1) // Different executor to the other tests to help balance things better
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(executor))

	tests := []struct {
		name             string
		defendClaimDepth types.Depth
	}{
		{"StepFirst", 0},
		{"StepMiddle", 28},
		{"StepInExtension", 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UseExecutor(executor))

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
			require.NotNil(t, game)
			game.LogGameData(ctx)

			outputClaim := game.DisputeLastBlock(ctx)
			splitDepth := game.SplitDepth(ctx)

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

			game.DefendClaim(
				ctx,
				outputClaim,
				func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
					if claim.Depth()+1 == splitDepth+test.defendClaimDepth {
						return claim.Defend(ctx, common.Hash{byte(claim.Depth())})
					} else {
						return claim.Attack(ctx, common.Hash{byte(claim.Depth())})
					}
				})

			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.LogGameData(ctx)
			game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
		})
	}
}

func TestOutputCannonDefendStep(t *testing.T) {
	executor := uint64(1) // Different executor to the other tests to help balance things better
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(executor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	outputRootClaim := game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))

	maxDepth := game.MaxDepth(ctx)
	game.DefendClaim(ctx, outputRootClaim, func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// Post invalid claims for most steps to get down into the early part of the trace
		if claim.Depth() < maxDepth-3 {
			return claim.Attack(ctx, common.Hash{0xaa})
		} else {
			// Post our own counter but using the correct hash in low levels to force a defense step
			return correctTrace.AttackClaim(ctx, claim)
		}
	})

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForInactivity(ctx, 10, true)
	game.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
}

func TestOutputCannonProposedOutputRootValid(t *testing.T) {
	executor := uint64(1) // Different executor to the other tests to help balance things better
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(executor))
	// honestStepsFail attempts to perform both an attack and defend step using the correct trace.
	honestStepsFail := func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64) {
		// Attack step should fail
		correctTrace.StepFails(ctx, parentClaimIdx, true)
		// Defending should fail too
		correctTrace.StepFails(ctx, parentClaimIdx, false)
	}
	tests := []struct {
		// name is the name of the test
		name string

		// performMove is called to respond to each claim posted by the honest op-challenger.
		// It should either attack or defend the claim at parentClaimIdx
		performMove func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, claim *disputegame.ClaimHelper) *disputegame.ClaimHelper

		// performStep is called once the maximum game depth is reached. It should perform a step to counter the
		// claim at parentClaimIdx. Since the proposed output root is invalid, the step call should always revert.
		performStep func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64)
	}{
		{
			name: "AttackWithCorrectTrace",
			performMove: func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				// Attack everything but oddly using the correct hash.
				// Except the root of the cannon game must have an invalid VM status code.
				if claim.IsOutputRootLeaf(ctx) {
					return claim.Attack(ctx, common.Hash{0x01})
				}
				return correctTrace.AttackClaim(ctx, claim)
			},
			performStep: honestStepsFail,
		},
		{
			name: "DefendWithCorrectTrace",
			performMove: func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				// Can only attack the root claim or the first cannon claim
				if claim.IsRootClaim() {
					return correctTrace.AttackClaim(ctx, claim)
				}
				// The root of the cannon game must have an invalid VM status code
				// Attacking ensure we're running the cannon trace between two different blocks
				// instead of being in the trace extension of the output root bisection
				if claim.IsOutputRootLeaf(ctx) {
					return claim.Attack(ctx, common.Hash{0x01})
				}
				// Otherwise, defend everything using the correct hash.
				return correctTrace.DefendClaim(ctx, claim)
			},
			performStep: honestStepsFail,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UseExecutor(executor))

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
			correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

			// Now maliciously play the game and it should be impossible to win
			game.ChallengeClaim(ctx,
				game.RootClaim(ctx),
				func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
					return test.performMove(ctx, game, correctTrace, claim)
				},
				func(parentClaimIdx int64) {
					test.performStep(ctx, game, correctTrace, parentClaimIdx)
				})

			// Time travel past when the game will be resolvable.
			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.WaitForInactivity(ctx, 10, true)
			game.LogGameData(ctx)
			require.EqualValues(t, disputegame.StatusDefenderWins, game.Status(ctx))
		})
	}
}

func TestOutputCannonPoisonedPostState(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Honest first attack at "honest" level
	claim := correctTrace.AttackClaim(ctx, game.RootClaim(ctx))

	// Honest defense at "dishonest" level
	claim = correctTrace.DefendClaim(ctx, claim)

	// Dishonest attack at "honest" level - honest move would be to ignore
	claimToIgnore1 := claim.Attack(ctx, common.Hash{0x03, 0xaa})

	// Honest attack at "dishonest" level - honest move would be to ignore
	claimToIgnore2 := correctTrace.AttackClaim(ctx, claimToIgnore1)
	game.LogGameData(ctx)

	// Start the honest challenger
	game.StartChallenger(ctx, "sequencer", "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	// Start dishonest challenger that posts correct claims
	for {
		game.LogGameData(ctx)
		// Wait for the challenger to counter
		// Note that we need to ignore claimToIgnore1 which already counters this...
		claim = claim.WaitForCounterClaim(ctx, claimToIgnore1)

		// Respond with our own move
		if claim.IsBottomGameRoot(ctx) {
			// Root of the cannon game must have the right VM status code (so it can't be honest).
			// Note this occurs when there are splitDepth + 4 claims because there are multiple forks in this game.
			claim = claim.Attack(ctx, common.Hash{0x01})
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}

		// Defender moves last. If we're at max depth, then we're done
		if claim.IsMaxDepth(ctx) {
			break
		}
	}

	// Wait for the challenger to call step
	claim.WaitForCountered(ctx)
	// Verify that the challenger didn't challenge our poisoned claims
	claimToIgnore1.RequireOnlyCounteredBy(ctx, claimToIgnore2)
	claimToIgnore2.RequireOnlyCounteredBy(ctx /* nothing */)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.LogGameData(ctx)
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
}

func TestDisputeOutputRootBeyondProposedBlock_ValidOutputRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
	// Start the honest challenger
	game.StartChallenger(ctx, "sequencer", "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	claim := game.RootClaim(ctx)
	// Attack the output root
	claim = correctTrace.AttackClaim(ctx, claim)
	// Wait for the challenger to respond
	claim = claim.WaitForCounterClaim(ctx)
	// Then defend until the split depth to force the game into the extension part of the output root bisection
	// ie. the output root we wind up disputing is theoretically for a block after block number 1
	for !claim.IsOutputRootLeaf(ctx) {
		claim = correctTrace.DefendClaim(ctx, claim)
		claim = claim.WaitForCounterClaim(ctx)
	}
	game.LogGameData(ctx)
	// At this point we've reached the bottom of the output root bisection and every claim
	// will have the same, valid, output root. We now need to post a cannon trace root that claims its invalid.
	claim = claim.Defend(ctx, common.Hash{0x01, 0xaa})
	// Now defend with the correct trace
	for {
		game.LogGameData(ctx)
		claim = claim.WaitForCounterClaim(ctx)
		if claim.IsMaxDepth(ctx) {
			break
		}
		claim = correctTrace.DefendClaim(ctx, claim)
	}
	// Should not be able to step either attacking or defending
	correctTrace.StepClaimFails(ctx, claim, true)
	correctTrace.StepClaimFails(ctx, claim, false)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusDefenderWins)
	game.LogGameData(ctx)
}

func TestDisputeOutputRootBeyondProposedBlock_InvalidOutputRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Start the honest challenger
	game.StartChallenger(ctx, "sequencer", "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	claim := game.RootClaim(ctx)
	// Wait for the honest challenger to counter the root
	claim = claim.WaitForCounterClaim(ctx)
	// Then defend until the split depth to force the game into the extension part of the output root bisection
	// ie. the output root we wind up disputing is theoretically for a block after block number 1
	// The dishonest actor challenges with the correct roots
	for claim.IsOutputRoot(ctx) {
		claim = correctTrace.DefendClaim(ctx, claim)
		claim = claim.WaitForCounterClaim(ctx)
	}
	game.LogGameData(ctx)
	// Now defend with the correct trace
	for !claim.IsMaxDepth(ctx) {
		game.LogGameData(ctx)
		if claim.IsBottomGameRoot(ctx) {
			claim = correctTrace.AttackClaim(ctx, claim)
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}
		if !claim.IsMaxDepth(ctx) {
			// Have to attack the root of the cannon trace
			claim = claim.WaitForCounterClaim(ctx)
		}
	}

	// Wait for our final claim to be countered by the challenger calling step
	claim.WaitForCountered(ctx)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}

func TestDisputeOutputRoot_ChangeClaimedOutputRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Start the honest challenger
	game.StartChallenger(ctx, "sequencer", "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	claim := game.RootClaim(ctx)
	// Wait for the honest challenger to counter the root
	claim = claim.WaitForCounterClaim(ctx)

	// Then attack every claim until the leaf of output root bisection
	for {
		claim = claim.Attack(ctx, common.Hash{0xbb})
		claim = claim.WaitForCounterClaim(ctx)
		if claim.Depth() == game.SplitDepth(ctx)-1 {
			// Post the correct output root as the leaf.
			// This is for block 1 which is what the original output root was for too
			claim = correctTrace.AttackClaim(ctx, claim)
			// Challenger should post the first cannon trace
			claim = claim.WaitForCounterClaim(ctx)
			break
		}
	}

	game.LogGameData(ctx)

	// Now defend with the correct trace
	for !claim.IsMaxDepth(ctx) {
		game.LogGameData(ctx)
		if claim.IsBottomGameRoot(ctx) {
			claim = correctTrace.AttackClaim(ctx, claim)
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}
		if !claim.IsMaxDepth(ctx) {
			// Have to attack the root of the cannon trace
			claim = claim.WaitForCounterClaim(ctx)
		}
	}

	// Wait for our final claim to be countered by the challenger calling step
	claim.WaitForCountered(ctx)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}
