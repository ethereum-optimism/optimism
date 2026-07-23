package zk

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// TestZK_HonestChallenger_Supernode_ValidProposal_DefenderWins checks that the honest challenger,
// sourcing super roots from the supernode, does not challenge a valid multi-chain super-root
// proposal, so the game resolves DEFENDER_WINS once the challenge window expires.
func TestZK_HonestChallenger_Supernode_ValidProposal_DefenderWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSystemWithHonestChallenger(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)

	game := factory.StartZKGame(proposer)

	advanceL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}

// TestZK_HonestChallenger_Supernode_InvalidProposal_ChallengerWins checks that the honest
// challenger detects an invalid multi-chain super-root proposal, challenges it, and — with no
// prover — resolves it CHALLENGER_WINS without ever proving.
func TestZK_HonestChallenger_Supernode_InvalidProposal_ChallengerWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSystemWithHonestChallenger(t)
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
	advanceL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	t.Require().Equal(common.Address{}, game.ClaimData().Prover, "challenger must not prove")
}

// TestZK_HonestChallenger_Supernode_UnsafeProposal_ChallengerWins mirrors the op-node unsafe-proposal
// case against the supernode source, which reaches Data == nil through a different path than op-node
// (VerifiedResultAtTimestamp returns NotFound rather than an op-node skeleton). A proposal for a
// not-yet-safe timestamp is challenged and resolves CHALLENGER_WINS.
func TestZK_HonestChallenger_Supernode_UnsafeProposal_ChallengerWins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSystemWithHonestChallenger(t)
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	registry := sys.AnchorStateRegistry(sys.L2ChainA)
	_, anchorSequence := registry.AnchorRoot()

	// Propose a super root far beyond the current safe head; the chain never reaches it during the
	// test, so the challenger keeps seeing Data == nil and challenges on that basis.
	safeTimestamp, _ := factory.WaitForSafeSuperRootAfter(anchorSequence)
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(safeTimestamp+3600),
		proofs.WithFutureProposal(),
	)

	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	t.Require().Equal(common.Address{}, game.ClaimData().Prover, "challenger must not prove")
}
