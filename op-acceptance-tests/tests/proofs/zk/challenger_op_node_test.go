package zk

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// newOpNodeSystem starts a single-chain interop system with no supernode, running an honest
// op-challenger whose --superroot-rpc points at the op-node's superroot_atTimestamp endpoint.
func newOpNodeSystem(t devtest.T) *presets.SingleChainInterop {
	return presets.NewSingleChainInteropNoSupernodeZKDispute(t)
}

func advanceOpNodeL1To(sys *presets.SingleChainInterop, timestamp uint64) {
	current := sys.L1EL.BlockRefByLabel(eth.Unsafe).Time
	if current < timestamp {
		sys.AdvanceTime(time.Duration(timestamp-current) * time.Second)
	}
	sys.L1EL.WaitForTime(timestamp)
}

// TestZK_HonestChallenger_ValidProposal_DefenderWins is the canonical repro: against a valid
// super-root proposal, the honest challenger recognizes it as valid and does not challenge, so the
// game resolves DEFENDER_WINS once the challenge window expires. The pre-migration actor compared a
// single-chain output root against the super-root claim, wrongly challenged, and the game resolved
// CHALLENGER_WINS.
func TestZK_HonestChallenger_ValidProposal_DefenderWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newOpNodeSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)

	game := factory.StartZKGame(proposer)

	advanceOpNodeL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}

// TestZK_HonestChallenger_InvalidProposal_ChallengerWins checks that the honest challenger detects
// an invalid super-root proposal, challenges it, and — with no prover submitting a proof — resolves
// it CHALLENGER_WINS. The challenger never proves; proving is the proposer's responsibility.
func TestZK_HonestChallenger_InvalidProposal_ChallengerWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newOpNodeSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	_, anchorSequence := registry.AnchorRoot()

	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)

	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceOpNodeL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	t.Require().Equal(common.Address{}, game.ClaimData().Prover, "challenger must not prove")
}

// TestZK_HonestChallenger_ChildOfInvalidParent_ChallengerWins checks resolution ordering: a child
// game referencing an invalid parent resolves CHALLENGER_WINS by inheritance only after the honest
// challenger has resolved the parent CHALLENGER_WINS.
func TestZK_HonestChallenger_ChildOfInvalidParent_ChallengerWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newOpNodeSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	_, anchorSequence := registry.AnchorRoot()

	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	parent := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	child := factory.StartZKGame(proposer, proofs.WithZKParent(parent.FactoryIndex()))

	parent.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceOpNodeL1To(sys, parent.ClaimData().Deadline+1)
	parent.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	child.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
}

// TestZK_HonestChallenger_ClaimsCredit checks that after winning an invalid game, the honest
// challenger closes the game and claims its credit once the finality delay elapses.
func TestZK_HonestChallenger_ClaimsCredit(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newOpNodeSystem(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	_, anchorSequence := registry.AnchorRoot()

	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)

	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceOpNodeL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)

	finalityDelay := sys.StandardBridge(sys.L2ChainA).DisputeGameFinalityDelay()
	advanceOpNodeL1To(sys, game.ResolvedAt()+uint64(finalityDelay/time.Second)+1)
	game.WaitForClaimedCredit(zkChallengerAddress(t, sys.L2ChainA.ChainID()))
}
