package zk

import (
	"math"
	"testing"
	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func TestDeploymentUsesSuperAggregationVKey(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, vkey := newSystem(t)
	factory := sys.DisputeGameFactory()

	factory.VerifyGameImplAbsent(gameTypes.SuperCannonKonaGameType)
	zk := factory.ZKGameImpl()
	t.Require().NotEqual(common.Address{}, zk.Address)
	t.Require().Equal(vkey, zk.Args.AbsolutePrestate)
	t.Require().Equal(uint64(zkChallengeDuration/time.Second), zk.Args.MaxChallengeDuration)
	t.Require().Equal(uint64(zkProveDuration/time.Second), zk.Args.MaxProveDuration)
	t.Require().Positive(zk.Args.ChallengerBond.Sign())
	t.Require().NotEqual(common.Address{}, zk.Args.AnchorStateRegistry)
	t.Require().NotEqual(common.Address{}, zk.Args.Weth)
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	code, err := sys.L1EL.EthClient().CodeAtHash(t.Ctx(), zk.Args.Verifier, l1Head.Hash)
	t.Require().NoError(err)
	t.Require().NotEmpty(code, "mock verifier must have deployed code")
}

func TestUnchallengedValidProposalAnchors(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)

	// TODO(#21463): Let the kona-sp1 proposer create the valid proposal.
	game := factory.StartZKGame(proposer)
	advanceL1To(sys, game.ClaimData().Deadline+1)

	// TODO(#21415): Let op-challenger resolve and close the unchallenged proposal.
	t.Require().Equal(gameTypes.GameStatusDefenderWon, game.Resolve(proposer))
	advanceL1To(sys, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	game.Close(proposer)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRoot(game)
}

func TestChallengedValidProposalAnchors(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newSystem(t)
	factory := sys.DisputeGameFactory()
	proposer, challenger, prover := fundedActors(sys)

	// TODO(#21463): Let the kona-sp1 proposer create the valid proposal and submit its proof.
	game := factory.StartZKGame(proposer)
	t.Require().Equal(uint32(math.MaxUint32), game.ParentIndex())
	t.Require().Equal(proofs.ZKProposalUnchallenged, game.ProposalStatus())

	challengedClaim := game.Challenge(challenger)
	t.Require().Equal(challenger.Address(), challengedClaim.Challenger)
	provedClaim := game.Prove(prover, []byte("mock-sp1-super-aggregation-proof"))
	t.Require().Equal(proofs.ZKProposalChallengedAndValidProofProvided, proofs.ZKProposalStatus(provedClaim.Status))
	t.Require().Equal(prover.Address(), provedClaim.Prover)

	// TODO(#21415): Let op-challenger resolve and close the proven proposal.
	t.Require().Equal(gameTypes.GameStatusDefenderWon, game.Resolve(proposer))
	advanceL1To(sys, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	game.Close(proposer)
	t.Require().Equal(challengerTypes.NormalDistributionMode, game.BondDistributionMode())
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRoot(game)
}

func TestChallengedInvalidProposalTimesOutWithoutAnchoring(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys, _ := newSystem(t)
	factory := sys.DisputeGameFactory()
	proposer, challenger, resolver := fundedActors(sys)
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	anchorRoot, anchorSequence := registry.AnchorRoot()

	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(
		proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	// TODO(#21415): Let the op-challenger detect, challenge, resolve, and close this invalid proposal.
	challengedClaim := game.Challenge(challenger)

	advanceL1To(sys, challengedClaim.Deadline+1)
	t.Require().True(game.GameOver())
	t.Require().Equal(gameTypes.GameStatusChallengerWon, game.Resolve(resolver))
	advanceL1To(sys, game.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)
	game.Close(resolver)
	t.Require().Equal(challengerTypes.NormalDistributionMode, game.BondDistributionMode())

	actualRoot, actualSequence := registry.AnchorRoot()
	t.Require().Equal(anchorRoot, actualRoot)
	t.Require().Equal(anchorSequence, actualSequence)
}

func fundedActors(sys *presets.SimpleInterop) (*dsl.EOA, *dsl.EOA, *dsl.EOA) {
	actors := sys.FunderL1.NewFundedEOAs(3, eth.OneEther)
	return actors[0], actors[1], actors[2]
}

func advanceL1To(sys *presets.SimpleInterop, timestamp uint64) {
	current := sys.L1EL.BlockRefByLabel(eth.Unsafe).Time
	if current < timestamp {
		sys.AdvanceTime(time.Duration(timestamp-current) * time.Second)
	}
	sys.L1EL.WaitForTime(timestamp)
}
