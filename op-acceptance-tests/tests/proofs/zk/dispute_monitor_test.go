package zk

import (
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/disputemon"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestZKDisputeMonitorSkipsLaggedGameUntilRootSourceCatchesUp(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	lagged := dsl.NewLagControlledSuperRootSource(t, sys.SuperRoots)
	game := sys.DisputeGameFactory().StartZKGame(sys.FunderL1.NewFundedEOA(eth.OneEther))
	game.AwaitRootSourcePastL1Head(sys.SuperRoots)

	monitor := sys.StartDisputeMon(presets.WithDisputeMonSupernodes(lagged))
	monitor.VerifyCompletedCycleWithoutFailures()
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(0),
		disputemon.DisagreedRoots(0),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(0),
	)

	lagged.Release()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.CorrectDefenderAhead(1),
	)
}

func TestZKDisputeMonitorValidInProgressProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	game := factory.StartZKGame(proposer)
	game.AwaitRootSourcePastL1Head(sys.SuperRoots)

	monitor := sys.StartDisputeMon(presets.WithDisputeMonHonestActors(proposer.Address()))
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.CorrectDefenderAhead(1),
		disputemon.ExactNonWithdrawableCredits(0),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, game.TotalBonds().ToBig()),
		disputemon.HonestActorPendingBonds(proposer.Address(), game.TotalBonds().ToBig()),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(0),
		disputemon.AnchorStateL2SequenceNumber(factory.ZKGameImpl().Args.AnchorStateRegistry, anchorSequence),
	)
}

func TestZKDisputeMonitorInvalidChallengedInProgressProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	factory := sys.DisputeGameFactory()
	proposer, challenger := fundedActors(sys)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(
		proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	game.Challenge(challenger)
	game.AwaitRootSourcePastL1Head(sys.SuperRoots)

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.DisagreedRoots(1),
		disputemon.CorrectChallengerAhead(1),
		disputemon.ExactNonWithdrawableCredits(0),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, game.TotalBonds().ToBig()),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(0),
	)
}

func TestZKDisputeMonitorValidTerminalProposalAfterDeadline(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	factory := sys.DisputeGameFactory()
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	_, anchorSequence := registry.AnchorRoot()
	registryAddress := factory.ZKGameImpl().Args.AnchorStateRegistry
	proposer, resolver := fundedActors(sys)
	game := factory.StartZKGame(proposer)
	game.AwaitRootSourcePastL1Head(sys.SuperRoots)

	monitor := sys.StartDisputeMon(presets.WithDisputeMonHonestActors(proposer.Address()))
	advanceL1To(&sys.SingleChainInterop, game.ClaimData().Deadline+1)
	monitor.VerifyState(
		disputemon.PendingZKResolutions(1),
		disputemon.PendingZKBondDistributions(0),
	)

	t.Require().Equal(gameTypes.GameStatusDefenderWon, game.Resolve(resolver))
	totalBonds := game.TotalBonds().ToBig()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.CorrectDefenderWins(1),
		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, totalBonds),
		disputemon.HonestActorPendingBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorWonBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), new(big.Int)),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(1),
		disputemon.AnchorStateL2SequenceNumber(registryAddress, anchorSequence),
	)

	weth := factory.DelayedWETH(game.WETHAddress())
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)
	game.ClaimCredit(resolver, proposer.Address())
	withdrawal := weth.Withdrawal(game.Address, proposer.Address())
	t.Require().Equal(totalBonds, withdrawal.Amount)
	maturity := withdrawal.MaturesAt(weth.Delay())
	monitor.VerifyState(
		disputemon.PendingZKBondDistributions(0),
		disputemon.AnchorStateL2SequenceNumber(registryAddress, game.L2SequenceNumber()),
		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.MatchingWithdrawalRequests(game, 1),
		disputemon.DivergentWithdrawalRequests(game, 0),
		disputemon.FullyCollateralized(game, totalBonds),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), new(big.Int)),
	)

	advanceL1To(&sys.SingleChainInterop, maturity)
	monitor.VerifyState(
		disputemon.ExactWithdrawableCredits(1),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), totalBonds),
	)

	game.ClaimCredit(resolver, proposer.Address())
	t.Require().True(game.Credit(proposer.Address()).IsZero())
	t.Require().Zero(weth.Withdrawal(game.Address, proposer.Address()).Amount.Sign())
	monitor.VerifyState(
		disputemon.MatchingWithdrawalRequests(game, 1),
		disputemon.DivergentWithdrawalRequests(game, 0),
		disputemon.FullyCollateralized(game, new(big.Int)),
		disputemon.HonestActorPendingBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorWonBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), new(big.Int)),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(0),
	)
}

func TestZKDisputeMonitorCanonicalChildOfInvalidParent(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	factory := sys.DisputeGameFactory()
	proposer, challenger := fundedActors(sys)
	resolver := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	parentTimestamp, parentOutputs := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(parentOutputs)
	parentOutputs[0][0] ^= 0xff
	parent := factory.StartZKGame(
		proposer,
		proofs.WithL2SequenceNumber(parentTimestamp),
		proofs.WithSuperRootFrom(parentOutputs...),
	)
	child := factory.StartZKGame(proposer, proofs.WithZKParent(parent.FactoryIndex()))
	parent.Challenge(challenger)
	parent.AwaitRootSourcePastL1Head(sys.SuperRoots)
	child.AwaitRootSourcePastL1Head(sys.SuperRoots)
	totalBonds := parent.TotalBonds().Add(child.TotalBonds()).ToBig()

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 2),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.DisagreedRoots(1),
		disputemon.CorrectDefenderAhead(1),
		disputemon.CorrectChallengerAhead(1),
		disputemon.ExactNonWithdrawableCredits(0),
		disputemon.NoWithdrawalRequests(parent),
		disputemon.FullyCollateralized(parent, totalBonds),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(0),
	)

	advanceL1To(&sys.SingleChainInterop, parent.ClaimData().Deadline+1)
	t.Require().Equal(gameTypes.GameStatusChallengerWon, parent.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(1),
		disputemon.IncorrectChallengerAhead(1),
		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.PendingZKResolutions(1),
		disputemon.PendingZKBondDistributions(1),
	)

	t.Require().Equal(gameTypes.GameStatusChallengerWon, child.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(1),
		disputemon.IncorrectChallengerWins(1),
		disputemon.ExactNonWithdrawableCredits(2),
		disputemon.NoWithdrawalRequests(parent),
		disputemon.FullyCollateralized(parent, totalBonds),
		disputemon.PendingZKResolutions(0),
		disputemon.PendingZKBondDistributions(2),
	)
}

func newDisputeMonitorSystem(t devtest.T) *presets.SimpleInterop {
	return presets.NewSimpleInterop(
		t,
		presets.WithZK(),
		presets.WithoutHonestProposer(),
		presets.WithoutHonestChallenger(),
	)
}
