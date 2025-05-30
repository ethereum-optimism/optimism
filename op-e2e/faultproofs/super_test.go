package faultproofs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/batcher"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame/preimage"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/interop"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func temporarilySkip(t *testing.T) {
	// TODO(#16166) Reenable these tests
	t.Skip("Skip temporarily while #16166 is in progress")
}

func TestCreateSuperCannonGame(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		sys.L2IDs()
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		game.LogGameData(ctx)
	})
}

func TestSuperCannonGame(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testCannonGame(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	})
}

func TestSuperCannonGame_ChallengeAllZeroClaim(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testCannonChallengeAllZeroClaim(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonPublishCannonRootClaim(t *testing.T) {
	temporarilySkip(t)
	type TestCase struct {
		disputeL2SequenceNumberOffset uint64
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("Dispute_%v_%v", test.disputeL2SequenceNumberOffset, vm)
	}

	tests := []TestCase{
		{2},
		{3},
		{4},
		{5},
		{6},
	}

	vmStatusCh := make(chan byte, len(tests))
	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		ctx := context.Background()

		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		b, err := sys.L1GethClient().BlockByNumber(ctx, nil)
		require.NoError(t, err)
		disputeL2SequenceNumber := b.Time() + test.disputeL2SequenceNumberOffset

		game := disputeGameFactory.StartSuperCannonGameAtTimestamp(ctx, disputeL2SequenceNumber, common.Hash{0x01})
		game.DisputeLastBlock(ctx)
		game.LogGameData(ctx)
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		splitDepth := game.SplitDepth(ctx)
		t.Logf("Waiting for claim at depth %v (split depth %v)", splitDepth+1, splitDepth)
		game.WaitForClaimAtDepth(ctx, splitDepth+1)

		claims, err := game.Game.GetAllClaims(ctx, rpcblock.Latest)
		require.NoError(t, err)
		var bottomRootClaim types.Claim
		for _, claim := range claims {
			if claim.Depth() == splitDepth+1 {
				require.True(t, bottomRootClaim == types.Claim{}, "Found multiple bottom root claims")
				bottomRootClaim = claim
			}
		}
		require.True(t, bottomRootClaim != types.Claim{}, "Failed to find bottom root claim")
		t.Logf("Bottom root claim: %v", bottomRootClaim.Value)
		vmStatusCh <- bottomRootClaim.Value[0]
	}, WithNextVMOnly[TestCase](), WithTestName[TestCase](testName))

	// Cleanup ensures that the subtests run to completion before asserting the VM statuses
	t.Cleanup(func() {
		close(vmStatusCh)
		vmStatuses := make(map[byte]bool)
		for status := range vmStatusCh {
			vmStatuses[status] = true
		}
		require.Greater(t, len(vmStatuses), 1,
			"Invalid test setup. Expected at least 2 different VM statuses, got %v instead. Ensure that the disputeL2SequenceNumberOffsets are set correctly.",
			vmStatuses)
	})
}

func TestSuperCannonDisputeGame(t *testing.T) {
	temporarilySkip(t)
	type TestCase struct {
		name             string
		defendClaimDepth types.Depth
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{"StepFirst", 0},
		{"StepMiddle", 28},
		{"StepInExtension", 1},
	}
	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01, 0xaa})
		game.LogGameData(ctx)

		disputeClaim := game.DisputeLastBlock(ctx)
		splitDepth := game.SplitDepth(ctx)

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		game.SupportClaim(
			ctx,
			disputeClaim,
			func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				if claim.Depth()+1 == splitDepth+test.defendClaimDepth {
					return claim.Defend(ctx, common.Hash{byte(claim.Depth())})
				} else {
					return claim.Attack(ctx, common.Hash{byte(claim.Depth())})
				}
			},
			func(parentIdx int64) {
				t.Log("Calling step on challenger's claim...")
				honestActor := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
					c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
				})
				honestActor.StepFails(ctx, parentIdx, false)
				honestActor.StepFails(ctx, parentIdx, true)
			},
		)

		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.LogGameData(ctx)
		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	}, WithNextVMOnly[TestCase](), WithTestName(testName))
}

func TestSuperCannonDefendStep(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testCannonDefendStep(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonStepWithLargePreimage(t *testing.T) {
	t.Skip("Skipping large preimage test due to cross-safe stall in the supervisor")
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))

		for _, id := range sys.L2IDs() {
			require.NoError(t, sys.Batcher(id).Stop(ctx))
		}
		// Manually send a tx from the correct batcher key to the batcher input with very large (invalid) data
		// This forces op-program to load a large preimage.
		for _, id := range sys.L2IDs() {
			batcherKey := sys.L2OperatorKey(id, devkeys.BatcherRole)
			batcherHelper := batcher.NewHelper(t, &batcherKey, sys.RollupConfig(id), sys.L1GethClient())
			t.Logf("Sending large invalid batch from batcher %v", id)
			batcherHelper.SendLargeInvalidBatch(ctx)
		}
		for _, id := range sys.L2IDs() {
			require.NoError(t, sys.Batcher(id).Start(ctx))
		}

		L1Head, err := sys.L1GethClient().BlockByNumber(ctx, nil)
		require.NoError(t, err)
		t.Logf("L1 head %d (%x) timestamp: %v", L1Head.NumberU64(), L1Head.Hash(), L1Head.Time())
		l2Timestamp := L1Head.Time()
		disputeGameFactory.WaitForSuperTimestamp(l2Timestamp, &disputegame.GameCfg{})

		// Dispute any block - it will have to read the L1 batches to see if the block is reached
		game := disputeGameFactory.StartSuperCannonGameWithCorrectRootAtTimestamp(ctx, l2Timestamp)
		topGameLeaf := game.DisputeBlock(ctx, l2Timestamp)
		game.LogGameData(ctx)

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		topGameLeaf = topGameLeaf.Attack(ctx, common.Hash{0x01})

		game.LogGameData(ctx)
		// Now the honest challenger is positioned as the defender of the execution game. We then dispute it by inducing a large preimage load.
		sender := crypto.PubkeyToAddress(aliceKey(t).PublicKey)
		preimageLoadCheck := game.CreateStepLargePreimageLoadCheck(ctx, sender)
		game.ChallengeToPreimageLoad(ctx, topGameLeaf, aliceKey(t), utils.PreimageLargerThan(preimage.MinPreimageSize), preimageLoadCheck, false)
		// The above method already verified the image was uploaded and step called successfully
		// So we don't waste time resolving the game - that's tested elsewhere.
	}, WithNextVMOnly[any]())
}

func TestSuperCannonStepWithPreimage_nonExistingPreimage(t *testing.T) {
	temporarilySkip(t)
	preimageConditions := []string{"keccak", "sha256", "blob"}
	testName := func(vm string, preimageType string) string {
		return fmt.Sprintf("%v-%v", preimageType, vm)
	}

	RunTestsAcrossVmTypes(t, preimageConditions, func(t *testing.T, allocType config.AllocType, preimageType string) {
		if preimageType == "blob" || preimageType == "sha256" {
			t.Skip("TODO(#15311): Add blob preimage test case. sha256 is also used for blobs")
		}
		testSuperPreimageStep(t, utils.FirstPreimageLoadOfType(preimageType), false, allocType)
	}, WithNextVMOnly[string](), WithTestName(testName))
}

func TestSuperCannonStepWithPreimage_existingPreimage(t *testing.T) {
	temporarilySkip(t)
	// Only test pre-existing images with one type to save runtime
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		testSuperPreimageStep(t, utils.FirstKeccakPreimageLoad(), true, allocType)
	}, WithNextVMOnly[any](), WithTestNamePrefix[any]("preimage already exists"))
}

func testSuperPreimageStep(t *testing.T, preimageType utils.PreimageOpt, preloadPreimage bool, allocType config.AllocType) {
	temporarilySkip(t)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))

	status, err := sys.SupervisorClient().SyncStatus(ctx)
	require.NoError(t, err)
	l2Timestamp := status.SafeTimestamp

	game := disputeGameFactory.StartSuperCannonGameWithCorrectRootAtTimestamp(ctx, l2Timestamp)
	topGameLeaf := game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

	// This attack creates a bottom game such that we will be making the last move at the bottom. (see game depth parameters for the super DG)
	// This presents an opportunity for the challenger to step on our dishonest claim at the bottom.
	// This assumes the execution game depth is even. But if it is odd, then this test should be set up more like the FDG counter part.
	topGameLeaf = topGameLeaf.Attack(ctx, common.Hash{0x01})

	// Now the honest challenger is positioned as the defender of the execution game. We then move to challenge it to induce a preimage load
	preimageLoadCheck := game.CreateStepPreimageLoadCheck(ctx)
	game.ChallengeToPreimageLoad(ctx, topGameLeaf, aliceKey(t), preimageType, preimageLoadCheck, preloadPreimage)
	// The above method already verified the image was uploaded and step called successfully
	// So we don't waste time resolving the game - that's tested elsewhere.
}

func TestSuperCannonProposalValid_AttackWithCorrectTrace(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGameWithCorrectRoot(ctx)
		testCannonProposalValid_AttackWithCorrectTrace(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonProposalValid_DefendWithCorrectTrace(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGameWithCorrectRoot(ctx)
		testCannonProposalValid_DefendWithCorrectTrace(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonPoisonedPostState(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testCannonPoisonedPostState(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonRootBeyondProposedBlock_ValidRoot(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGameWithCorrectRoot(ctx)
		testDisputeRootBeyondProposedBlockValidOutputRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonRootBeyondProposedBlock_InvalidRoot(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testDisputeRootBeyondProposedBlockInvalidOutputRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonRootChangeClaimedRoot(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
		testDisputeRootChangeClaimedRoot(t, ctx, createSuperGameArena(t, sys, game), &game.SplitGameHelper)
	}, WithNextVMOnly[any]())
}

func TestSuperInvalidateUnsafeProposal(t *testing.T) {
	temporarilySkip(t)
	ctx := context.Background()
	type TestCase struct {
		name     string
		strategy func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{
			name: "Attack",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.AttackClaim(ctx, parent)
			},
		},
		{
			name: "Defend",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.DefendClaim(ctx, parent)
			},
		},
		{
			name: "Counter",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.CounterClaim(ctx, parent)
			},
		},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))

		client := sys.SupervisorClient()
		status, err := client.SyncStatus(ctx)
		require.NoError(t, err, "Failed to get sync status")
		// Ensure that the superchain has progressed a bit past the genesis timestamp
		disputeGameFactory.WaitForSuperTimestamp(status.SafeTimestamp+4, &disputegame.GameCfg{})
		// halt the safe chain
		for _, id := range sys.L2IDs() {
			require.NoError(t, sys.Batcher(id).Stop(ctx))
		}

		status, err = client.SyncStatus(ctx)
		require.NoError(t, err, "Failed to get sync status")

		// Wait for any client to advance its unsafe head past the safe chain. We know this head will remain unsafe since the batc
		l2Client := sys.L2GethClient(sys.L2IDs()[0], "sequencer")
		require.NoError(t, wait.ForNextBlock(ctx, l2Client))
		head, err := l2Client.BlockByNumber(ctx, nil)
		require.NoError(t, err, "Failed to get head block")
		unsafeTimestamp := head.Time()

		// Root claim is _dishonest_ because the required data is not available on L1
		unsafeSuper := createSuperRoot(t, ctx, sys, unsafeTimestamp)
		unsafeRoot := eth.SuperRoot(unsafeSuper)
		game := disputeGameFactory.StartSuperCannonGameAtTimestamp(ctx, unsafeTimestamp, common.Hash(unsafeRoot), disputegame.WithFutureProposal())

		correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
			c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
		})
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		game.SupportClaim(ctx, game.RootClaim(ctx), func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			if parent.IsBottomGameRoot(ctx) {
				return correctTrace.AttackClaim(ctx, parent)
			}
			return test.strategy(correctTrace, parent)
		},
			func(parentIdx int64) {
				t.Log("Calling step on challenger's claim...")
				correctTrace.StepFails(ctx, parentIdx, false)
				correctTrace.StepFails(ctx, parentIdx, true)
			},
		)

		// Time travel past when the game will be resolvable.
		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
		game.LogGameData(ctx)
	}, WithNextVMOnly[TestCase](), WithTestName(testName))
}

func TestSuperInvalidateProposalForFutureBlock(t *testing.T) {
	temporarilySkip(t)
	ctx := context.Background()
	type TestCase struct {
		name     string
		strategy func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{
			name: "Attack",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.AttackClaim(ctx, parent)
			},
		},
		{
			name: "Defend",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.DefendClaim(ctx, parent)
			},
		},
		{
			name: "Counter",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.CounterClaim(ctx, parent)
			},
		},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		// Root claim is _dishonest_ because the required data is not available on L1
		farFutureTimestamp := time.Now().Add(time.Second * 10_000_000).Unix()
		game := disputeGameFactory.StartSuperCannonGameAtTimestamp(ctx, uint64(farFutureTimestamp), common.Hash{0x01}, disputegame.WithFutureProposal())
		correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
			c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
		})
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		game.SupportClaim(ctx, game.RootClaim(ctx), func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			if parent.IsBottomGameRoot(ctx) {
				return correctTrace.AttackClaim(ctx, parent)
			}
			return test.strategy(correctTrace, parent)
		},
			func(parentIdx int64) {
				t.Log("Calling step on challenger's claim...")
				correctTrace.StepFails(ctx, parentIdx, false)
				correctTrace.StepFails(ctx, parentIdx, true)
			},
		)

		// Time travel past when the game will be resolvable.
		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
		game.LogGameData(ctx)
	}, WithNextVMOnly[TestCase](), WithTestName(testName))
}

func TestSuperInvalidateCorrectProposalFutureBlock(t *testing.T) {
	temporarilySkip(t)
	ctx := context.Background()
	type TestCase struct {
		name     string
		strategy func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{
			name: "Attack",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.AttackClaim(ctx, parent)
			},
		},
		{
			name: "Defend",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.DefendClaim(ctx, parent)
			},
		},
		{
			name: "Counter",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.CounterClaim(ctx, parent)
			},
		},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		client := sys.SupervisorClient()

		status, err := client.SyncStatus(ctx)
		require.NoError(t, err, "Failed to get sync status")
		superRoot, err := client.SuperRootAtTimestamp(ctx, hexutil.Uint64(status.SafeTimestamp))
		require.NoError(t, err, "Failed to get super root at safe timestamp")

		// Stop the batcher so the safe head doesn't advance
		for _, id := range sys.L2IDs() {
			require.NoError(t, sys.Batcher(id).Stop(ctx))
		}

		// Create a dispute game with a proposal that is valid at `superRoot.Timestamp`, but that claims to correspond to timestamp
		// `superRoot.Timestamp + 100000`. This is dishonest, because the superchain hasn't reached this timestamp yet.
		game := disputeGameFactory.StartSuperCannonGameAtTimestamp(ctx, superRoot.Timestamp+100_000, common.Hash(superRoot.SuperRoot), disputegame.WithFutureProposal())

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
			c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
		})

		game.SupportClaim(ctx, game.RootClaim(ctx), func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			if parent.IsBottomGameRoot(ctx) {
				return correctTrace.AttackClaim(ctx, parent)
			}
			return test.strategy(correctTrace, parent)
		},
			func(parentIdx int64) {
				t.Log("Calling step on challenger's claim...")
				correctTrace.StepFails(ctx, parentIdx, false)
				correctTrace.StepFails(ctx, parentIdx, true)
			},
		)
		game.LogGameData(ctx)

		// Time travel past when the game will be resolvable.
		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
		game.LogGameData(ctx)
	}, WithNextVMOnly[TestCase](), WithTestName(testName))
}

func TestSuperCannonHonestSafeTraceExtensionValidRoot(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()

		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		client := sys.SupervisorClient()
		// Wait for there to be there are safe L2 blocks past the claimed safe timestamp that have data available on L1 within
		// the commitment stored in the dispute game.
		status, err := client.SyncStatus(ctx)
		require.NoError(t, err, "Failed to get sync status")
		safeTimestamp := status.SafeTimestamp

		disputeGameFactory.WaitForSuperTimestamp(safeTimestamp, new(disputegame.GameCfg))

		game := disputeGameFactory.StartSuperCannonGameWithCorrectRootAtTimestamp(ctx, safeTimestamp-1)
		require.NotNil(t, game)

		// Create a correct trace actor with an honest trace extending to L2 block #5
		// Notably, L2 block #5 is a valid block within the safe chain, and the data required to reproduce it
		// will be committed to within the L1 head of the dispute game.
		correctTracePlus1 := game.CreateHonestActor(ctx,
			disputegame.WithPrivKey(malloryKey(t)),
			disputegame.WithClaimedL2BlockNumber(safeTimestamp),
			func(c *disputegame.HonestActorConfig) {
				c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
			},
		)

		// Start the honest challenger. They will defend the root claim.
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		root := game.RootClaim(ctx)
		// Have to disagree with the root claim - we're trying to invalidate a valid output root
		firstAttack := root.Attack(ctx, common.Hash{0xdd})
		game.SupportClaim(ctx, firstAttack, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			return correctTracePlus1.CounterClaim(ctx, parent)
		}, func(parentClaimIdx int64) {
			t.Logf("Waiting for bottom claim %v to be countered", parentClaimIdx)
			claim, err := game.Game.GetClaim(ctx, uint64(parentClaimIdx))
			require.NoError(t, err)
			require.Equal(t, game.MaxDepth(ctx), claim.Position.Depth())
			game.WaitForCountered(ctx, parentClaimIdx)
		})
		game.LogGameData(ctx)

		// Time travel past when the game will be resolvable.
		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonHonestSafeTraceExtensionInvalidRoot(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()

		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
		client := sys.SupervisorClient()
		// Wait for there to be there are safe L2 blocks past the claimed safe timestamp that have data available on L1 within
		// the commitment stored in the dispute game.
		status, err := client.SyncStatus(ctx)
		require.NoError(t, err, "Failed to get sync status")
		safeTimestamp := status.SafeTimestamp

		disputeGameFactory.WaitForSuperTimestamp(safeTimestamp, new(disputegame.GameCfg))

		game := disputeGameFactory.StartSuperCannonGameAtTimestamp(ctx, safeTimestamp-1, common.Hash{0xCA, 0xFE})
		require.NotNil(t, game)

		// Create a correct trace actor with an honest trace extending to safeTimestamp
		correctTrace := game.CreateHonestActor(ctx, disputegame.WithPrivKey(malloryKey(t)), func(c *disputegame.HonestActorConfig) {
			c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
		})

		// Create a correct trace actor with an honest trace extending to L2 block #5
		// Notably, L2 block #5 is a valid block within the safe chain, and the data required to reproduce it
		// will be committed to within the L1 head of the dispute game.
		correctTracePlus1 := game.CreateHonestActor(ctx,
			disputegame.WithPrivKey(malloryKey(t)),
			disputegame.WithClaimedL2BlockNumber(safeTimestamp),
			func(c *disputegame.HonestActorConfig) {
				c.ChallengerOpts = append(c.ChallengerOpts, challenger.WithDepset(t, sys.DependencySet()))
			},
		)

		// Start the honest challenger. They will defend the root claim.
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(aliceKey(t)), challenger.WithDepset(t, sys.DependencySet()))

		claim := game.RootClaim(ctx)
		game.SupportClaim(ctx, claim, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			// Have to disagree with the root claim - we're trying to invalidate a valid output root
			if parent.IsRootClaim() {
				return parent.Attack(ctx, common.Hash{0xdd})
			}
			return correctTracePlus1.CounterClaim(ctx, parent)
		}, func(parentClaimIdx int64) {
			correctTrace.StepFails(ctx, parentClaimIdx, true)
			correctTrace.StepFails(ctx, parentClaimIdx, false)
			correctTracePlus1.StepFails(ctx, parentClaimIdx, true)
			correctTracePlus1.StepFails(ctx, parentClaimIdx, false)
		})
		game.LogGameData(ctx)

		// Time travel past when the game will be resolvable.
		sys.AdvanceL1Time(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, sys.L1GethClient()))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	}, WithNextVMOnly[any]())
}

func TestSuperCannonGame_HonestCallsSteps(t *testing.T) {
	temporarilySkip(t)
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		ctx := context.Background()
		sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(allocType))
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
	}, WithNextVMOnly[any]())
}

func createSuperRoot(t *testing.T, ctx context.Context, sys interop.SuperSystem, timestamp uint64) *eth.SuperV1 {
	chains := make(map[eth.ChainID]eth.Bytes32)
	for _, id := range sys.L2IDs() {
		rollupCfg := sys.RollupConfig(id)
		blockNum, err := rollupCfg.TargetBlockNumber(timestamp)
		t.Logf("Target block number for timestamp %v (%v): %v", timestamp, rollupCfg.L2ChainID, blockNum)
		require.NoError(t, err)

		client := sys.L2RollupClient(id, "sequencer")
		output, err := client.OutputAtBlock(ctx, blockNum)
		require.NoError(t, err)
		chains[eth.ChainIDFromBig(rollupCfg.L2ChainID)] = output.OutputRoot
	}

	var output eth.SuperV1
	for _, chainID := range sys.DependencySet().Chains() {
		output.Chains = append(output.Chains, eth.ChainIDAndOutput{
			ChainID: chainID,
			Output:  chains[chainID],
		})
	}
	output.Timestamp = timestamp
	return &output
}
