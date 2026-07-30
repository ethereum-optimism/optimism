package zk

import (
	"testing"

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
	game := sys.DisputeGameFactory().StartZKGame(sys.FunderL1.NewFundedEOA(eth.OneEther))
	awaitDisputeMonitorSource(sys, game)

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRootsForGameType(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectDefenderAhead(gameTypes.ZKDisputeGameType, 1),
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
	awaitDisputeMonitorSource(sys, game)

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.DisagreedRootsForGameType(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectChallengerAhead(gameTypes.ZKDisputeGameType, 1),
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
	awaitDisputeMonitorSource(sys, game)

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 1),
		disputemon.FailedGames(0),
		disputemon.AgreedRootsForGameType(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectDefenderWins(gameTypes.ZKDisputeGameType, 1),
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
	awaitDisputeMonitorSource(sys, parent, child)

	monitor := sys.StartDisputeMon()
	monitor.VerifyState(
		disputemon.GameCount(gameTypes.ZKDisputeGameType, 2),
		disputemon.FailedGames(0),
		disputemon.AgreedRootsForGameType(gameTypes.ZKDisputeGameType, 1),
		disputemon.DisagreedRootsForGameType(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectDefenderAhead(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectChallengerAhead(gameTypes.ZKDisputeGameType, 1),
	)

	advanceL1To(&sys.SingleChainInterop, parent.ClaimData().Deadline+1)
	t.Require().Equal(gameTypes.GameStatusChallengerWon, parent.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectAgreeChallengerAhead(gameTypes.ZKDisputeGameType, 1),
	)

	t.Require().Equal(gameTypes.GameStatusChallengerWon, child.Resolve(resolver))
	monitor.VerifyState(
		disputemon.FailedGames(0),
		disputemon.CorrectChallengerWins(gameTypes.ZKDisputeGameType, 1),
		disputemon.CorrectAgreeChallengerWins(gameTypes.ZKDisputeGameType, 1),
	)
}

func newDisputeMonitorSystem(t devtest.T) *presets.SimpleInterop {
	return newSystem(t, presets.WithoutHonestProposer(), presets.WithoutHonestChallenger())
}

func awaitDisputeMonitorSource(sys *presets.SimpleInterop, games ...*proofs.ZKGame) {
	for _, game := range games {
		game.AwaitRootSourcePastL1Head(sys.SuperRoots)
	}
}
