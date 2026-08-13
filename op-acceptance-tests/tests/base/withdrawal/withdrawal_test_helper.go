package withdrawal

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func withdrawalOpts(gameType gameTypes.GameType, extra ...presets.Option) []presets.Option {
	opts := []presets.Option{
		presets.WithTimeTravelEnabled(),
		presets.WithDeployerOptions(
			sysgo.WithFinalizationPeriodSeconds(1),
			// Satisfy OptimismPortal2 PROOF_MATURITY_DELAY_SECONDS check, avoid OptimismPortal_ProofNotOldEnough() revert
			sysgo.WithProofMaturityDelaySeconds(2),
		),
		presets.WithRespectedGameTypeOverride(gameType),
		presets.WithProposerOption(func(_ sysgo.ComponentTarget, cfg *ps.CLIConfig) {
			cfg.DisputeGameType = uint32(gameType)
		}),
	}
	if gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
		opts = append(opts, presets.WithDeployerOptions(
			sysgo.WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
			sysgo.WithDevFeatureEnabled(devfeatures.SuperRootGamesMigrationFlag),
		))
	}
	if gameType != gameTypes.SuperPermissionedGameType {
		opts = append(opts, presets.WithGameTypeAdded(gameType))
	}
	return append(opts, extra...)
}

func newSystem(t devtest.T, gameType gameTypes.GameType, extra ...presets.Option) *presets.Minimal {
	return presets.NewMinimal(t, withdrawalOpts(gameType, extra...)...)
}

type withdrawalTestEnv struct {
	bridge      *dsl.StandardBridge
	factory     *proofs.DisputeGameFactory
	anchor      *dsl.AnchorStateRegistry
	funderL1    *dsl.FunderEOA
	l2EL        *dsl.L2ELNode
	advanceTime func(time.Duration)
}

func newWithdrawalTestEnv(t devtest.T, gameType gameTypes.GameType, extra ...presets.Option) withdrawalTestEnv {
	if gameType == gameTypes.ZKDisputeGameType {
		opts := []presets.Option{
			presets.WithZK(),
			presets.WithZKProposerOption(sysgo.WithZKFastFinality()),
			presets.WithFinalizationPeriodSeconds(1),
			presets.WithProofMaturityDelaySeconds(2),
		}
		sys := presets.NewSimpleInterop(t, append(opts, extra...)...)
		return withdrawalTestEnv{
			bridge:      sys.StandardBridge(sys.L2ChainA),
			factory:     sys.DisputeGameFactory(),
			anchor:      sys.AnchorStateRegistry(sys.L2ChainA),
			funderL1:    sys.FunderL1,
			l2EL:        sys.L2ELA,
			advanceTime: sys.AdvanceTime,
		}
	}

	sys := newSystem(t, gameType, extra...)
	return withdrawalTestEnv{
		bridge:      sys.StandardBridge(),
		factory:     sys.DisputeGameFactory(),
		anchor:      sys.AnchorStateRegistry(),
		funderL1:    sys.FunderL1,
		l2EL:        sys.L2EL,
		advanceTime: sys.AdvanceTime,
	}
}

func TestWithdrawal(gt *testing.T, gameType gameTypes.GameType, extra ...presets.Option) {
	t := devtest.ParallelT(gt)
	if gameType == gameTypes.SuperPermissionedGameType || gameType == gameTypes.SuperCannonKonaGameType {
		// TODO(#21861): Enable this when kona-node supports superroot_atTimestamp
		sysgo.SkipOnKonaNode(t, "super-root proposals require op-node superroot RPC")
	}
	env := newWithdrawalTestEnv(t, gameType, extra...)
	bridge := env.bridge
	bridge.VerifyRespectedGameType(gameType)

	initialL1Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	l1User := env.funderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(env.l2EL) // Only receives funds via the deposit
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneHundredthEther

	// The max amount of withdrawal is limited to the total amount of deposit
	// We trigger deposit first to fund the L1 ETHLockbox to satisfy the invariant
	deposit := bridge.Deposit(depositAmount, l1User)
	expectedL1UserBalance := initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	expectedL2UserBalance := depositAmount
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance = expectedL2UserBalance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)
	var game *proofs.FaultDisputeGame
	if gameType != gameTypes.ZKDisputeGameType {
		game = env.factory.WaitForGame()
		t.Require().Equal(gameType, game.GameType())
	}

	withdrawal.Prove(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.ProveGasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)

	var zkGame *proofs.ZKGame
	if gameType == gameTypes.ZKDisputeGameType {
		gameIndex := withdrawal.ProvenDisputeGameIndex()
		t.Require().True(gameIndex.IsInt64(), "proven dispute game index must fit in int64: %s", gameIndex)
		t.Require().GreaterOrEqual(gameIndex.Int64(), int64(0), "proven dispute game index must not be negative")
		zkGame = env.factory.WaitForZKGameAtIndex(gameIndex.Int64())
		deadline := zkGame.ClaimData().Deadline
		zkGame.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
		claimData := zkGame.ClaimData()
		t.Require().NotEqual(common.Address{}, claimData.Prover, "fast-finality proposer must submit an accepted proof")
		t.Require().Less(zkGame.ResolvedAt(), deadline, "proven ZK game must resolve before its challenge deadline")
		env.advanceTime(max(bridge.WithdrawalDelay(), bridge.DisputeGameFinalityDelay()) + time.Second)
	} else {
		// Advance time until game is resolvable
		env.advanceTime(bridge.GameResolutionDelay())
		withdrawal.WaitForDisputeGameResolved()

		// Advance time to when game finalization and proof finalization delay has expired
		env.advanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))
	}

	if gameType == gameTypes.ZKDisputeGameType {
		t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	} else {
		t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameResolutionDelay", bridge.GameResolutionDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	}
	withdrawal.Finalize(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.FinalizeGasCost()).Add(withdrawalAmount)
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	if zkGame != nil {
		env.anchor.WaitForAnchorRootAtLeast(zkGame)
	} else {
		env.anchor.WaitForAnchorRootAtLeast(game)
	}
}

// TestWithdrawalAfterUpgrade is like TestWithdrawal but waits for the given fork to activate
// before initiating the withdrawal, exercising the upgrade path rather than genesis activation.
func TestWithdrawalAfterUpgrade(gt *testing.T, gameType gameTypes.GameType, fork opforks.Name, extra ...presets.Option) {
	t := devtest.ParallelT(gt)
	sys := newSystem(t, gameType, extra...)

	sys.L2Chain.AwaitActivation(t, fork)

	bridge := sys.StandardBridge()
	bridge.VerifyRespectedGameType(gameType)

	initialL1Balance := eth.OneThirdEther

	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL)
	depositAmount := eth.OneTenthEther
	withdrawalAmount := eth.OneHundredthEther

	deposit := bridge.Deposit(depositAmount, l1User)
	expectedL1UserBalance := initialL1Balance.Sub(depositAmount).Sub(deposit.GasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)
	expectedL2UserBalance := depositAmount
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance = expectedL2UserBalance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	withdrawal.Prove(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.ProveGasCost())
	l1User.VerifyBalanceExact(expectedL1UserBalance)

	sys.AdvanceTime(bridge.GameResolutionDelay())
	withdrawal.WaitForDisputeGameResolved()

	sys.AdvanceTime(max(bridge.WithdrawalDelay()-bridge.GameResolutionDelay(), bridge.DisputeGameFinalityDelay()))

	t.Logger().Info("Attempting to finalize", "proofMaturity", bridge.WithdrawalDelay(), "gameResolutionDelay", bridge.GameResolutionDelay(), "gameFinalityDelay", bridge.DisputeGameFinalityDelay())
	withdrawal.Finalize(l1User)
	expectedL1UserBalance = expectedL1UserBalance.Sub(withdrawal.FinalizeGasCost()).Add(withdrawalAmount)
	l1User.VerifyBalanceExact(expectedL1UserBalance)
}
