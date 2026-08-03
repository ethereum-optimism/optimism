package zk

import (
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/disputemon"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestZKDisputeMonitorValidInProgressProposal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	game := sys.DisputeGameFactory().StartZKGame(proposer)

	monitor := sys.StartDisputeMon(presets.WithDisputeMonHonestActors(proposer.Address()))
	monitor.VerifyCompletedCycleWithoutFailures()
	game.AwaitRootSourcePastL1Head(sys.SuperRoots)
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRoots(1),
		disputemon.CorrectDefenderAhead(1),
		disputemon.ExactNonWithdrawableCredits(0),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, game.TotalBonds().ToBig()),
		disputemon.HonestActorPendingBonds(proposer.Address(), game.TotalBonds().ToBig()),
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

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.DisagreedRoots(1),
		disputemon.CorrectChallengerAhead(1),
		disputemon.ExactNonWithdrawableCredits(0),
		disputemon.NoWithdrawalRequests(game),
		disputemon.FullyCollateralized(game, game.TotalBonds().ToBig()),
	)
}

func TestZKDisputeMonitorValidTerminalProposalAfterDeadline(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newDisputeMonitorSystem(t)
	factory := sys.DisputeGameFactory()
	proposer, resolver := fundedActors(sys)
	game := factory.StartZKGame(proposer)
	advanceL1To(&sys.SingleChainInterop, game.ClaimData().Deadline+1)
	t.Require().Equal(gameTypes.GameStatusDefenderWon, game.Resolve(resolver))

	monitor := sys.StartDisputeMon(presets.WithDisputeMonHonestActors(proposer.Address()))
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
	)

	weth := factory.DelayedWETH(game.WETHAddress())
	advanceL1To(&sys.SingleChainInterop, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	game.ClaimCredit(resolver, proposer.Address())
	withdrawal := weth.Withdrawal(game.Address, proposer.Address())
	t.Require().Equal(totalBonds, withdrawal.Amount)
	monitor.VerifyState(
		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.MatchingWithdrawalRequests(game, 1),
		disputemon.DivergentWithdrawalRequests(game, 0),
		disputemon.FullyCollateralized(game, totalBonds),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), new(big.Int)),
	)

	advanceL1To(&sys.SingleChainInterop, withdrawal.MaturesAt(weth.Delay()))
	monitor.VerifyState(
		disputemon.ExactWithdrawableCredits(1),
		disputemon.MatchingWithdrawalRequests(game, 1),
		disputemon.FullyCollateralized(game, totalBonds),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), totalBonds),
	)

	game.ClaimCredit(resolver, proposer.Address())
	t.Require().True(game.Credit(proposer.Address()).IsZero())
	t.Require().Zero(weth.Withdrawal(game.Address, proposer.Address()).Amount.Sign())
	monitor.VerifyState(
		disputemon.ExactWithdrawableCredits(1),
		disputemon.MatchingWithdrawalRequests(game, 1),
		disputemon.DivergentWithdrawalRequests(game, 0),
		disputemon.FullyCollateralized(game, new(big.Int)),
		disputemon.HonestActorPendingBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorWonBonds(proposer.Address(), new(big.Int)),
		disputemon.HonestActorPendingWithdrawals(proposer.Address(), new(big.Int)),
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
	)

	advanceL1To(&sys.SingleChainInterop, parent.ClaimData().Deadline+1)
	t.Require().Equal(gameTypes.GameStatusChallengerWon, parent.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(1),
		disputemon.IncorrectChallengerAhead(1),
		disputemon.ExactNonWithdrawableCredits(1),
		disputemon.NoWithdrawalRequests(parent),
		disputemon.FullyCollateralized(parent, totalBonds),
	)

	t.Require().Equal(gameTypes.GameStatusChallengerWon, child.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(1),
		disputemon.IncorrectChallengerWins(1),
		disputemon.ExactNonWithdrawableCredits(2),
		disputemon.NoWithdrawalRequests(parent),
		disputemon.FullyCollateralized(parent, totalBonds),
	)
}

func newDisputeMonitorSystem(t devtest.T) *presets.SimpleInterop {
	return newSystem(t, presets.WithoutHonestProposer(), presets.WithoutHonestChallenger())
}
