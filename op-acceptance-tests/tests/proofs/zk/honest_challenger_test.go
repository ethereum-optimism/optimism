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

// newSupernodeSystem starts a supernode-backed interop system whose honest op-challenger sources
// super roots from the supernode (with the honest kona-sp1-proposer also running unless disabled
// via extra options). It is returned as the shared *SingleChainInterop base so the same scenario
// bodies run against either super-root source.
func newSupernodeSystem(t devtest.T, extra ...presets.Option) *presets.SingleChainInterop {
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.Combine(extra...),
	)
	return &sys.SingleChainInterop
}

// honestChallengerResolvesValidProposal takes a valid super-root proposal and lets the challenge
// window expire. The honest challenger must recognize it as valid and not challenge, so the game
// resolves DEFENDER_WINS. It returns the resolved game so lifecycle tests can assert what follows
// (e.g. anchoring).
func honestChallengerResolvesValidProposal(t devtest.T, sys *presets.SingleChainInterop, game *proofs.ZKGame) *proofs.ZKGame {
	advanceL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	return game
}

// honestChallengerBeatsInvalidProposal seeds a super-root proposal with a corrupted output root at an
// already-safe timestamp. The honest challenger must detect the mismatch, challenge it, and — with no
// prover submitting a proof — win once the window expires without ever proving. It returns the
// resolved game.
func honestChallengerBeatsInvalidProposal(t devtest.T, sys *presets.SingleChainInterop) *proofs.ZKGame {
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

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
	return game
}

// honestChallengerBeatsUnsafeProposal seeds a proposal for a timestamp the node has not yet made
// safe. It has no canonical super root (Data == nil), which the honest challenger treats as invalid:
// it challenges and, with no proof submitted, wins once the window expires without ever proving. The
// op-node source reaches Data == nil via a skeleton result, the supernode source via a NotFound from
// VerifiedResultAtTimestamp — distinct paths that must reach the same outcome.
func honestChallengerBeatsUnsafeProposal(t devtest.T, sys *presets.SingleChainInterop) {
	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	// Propose a super root far beyond the current safe head; the chain never reaches it during the
	// test, so the challenger keeps seeing Data == nil and challenges on that basis.
	safeTimestamp, _ := factory.WaitForSafeSuperRootAfter(anchorSequence)
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(safeTimestamp+uint64(zkUnsafeProposalLead/time.Second)),
		proofs.WithFutureProposal(),
	)

	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
	advanceL1To(sys, game.ClaimData().Deadline+1)
	game.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	t.Require().Equal(common.Address{}, game.ClaimData().Prover, "challenger must not prove")
}

// TestZK_HonestChallenger_ValidProposal_DefenderWins is the canonical migration repro: against a
// valid super-root proposal, the honest challenger recognizes it as valid and does not challenge, so
// the game resolves DEFENDER_WINS. The pre-migration actor compared a single-chain output root
// against the super-root claim, wrongly challenged, and the game resolved CHALLENGER_WINS. The two
// super-root sources share the decision body; the supernode subtest additionally proves the
// unchallenged valid proposal anchors (source-agnostic lifecycle behavior tested once).
func TestZK_HonestChallenger_ValidProposal_DefenderWins(gt *testing.T) {
	gt.Run("op-node", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		sys := newOpNodeSystem(t)
		// The op-node preset runs no kona-sp1-proposer; seed the proposal manually.
		game := sys.DisputeGameFactory().StartZKGame(sys.FunderL1.NewFundedEOA(eth.OneEther))
		honestChallengerResolvesValidProposal(t, sys, game)
	})
	gt.Run("supernode", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		sys := newSupernodeSystem(t)
		// The honest proposer creates the valid root proposal.
		game := honestChallengerResolvesValidProposal(t, sys, sys.DisputeGameFactory().WaitForZKGameAtIndex(0))
		// The unchallenged valid proposal anchors once the finality delay
		// elapses. The live proposer keeps anchoring later games, so assert
		// the anchor reached at least this game's sequence.
		advanceL1To(sys, game.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)
		sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRootAtLeast(game)
	})
}

// TestZK_HonestChallenger_InvalidProposal_ChallengerWins checks that the honest challenger detects an
// invalid super-root proposal, challenges it, and — with no prover submitting a proof — resolves it
// CHALLENGER_WINS. The challenger never proves; proving is the proposer's responsibility. The two
// super-root sources share the decision body; the supernode subtest additionally proves the
// challenger claims its credit and the invalid proposal never advances the anchor state.
func TestZK_HonestChallenger_InvalidProposal_ChallengerWins(gt *testing.T) {
	gt.Run("op-node", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		honestChallengerBeatsInvalidProposal(t, newOpNodeSystem(t))
	})
	gt.Run("supernode", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		// This subtest asserts the anchor never moves, so no honest proposer:
		// its valid proposals would legitimately advance the anchor.
		sys := newSupernodeSystem(t, presets.WithoutHonestProposer())
		registry := sys.AnchorStateRegistry(sys.L2ChainA)
		anchorRoot, anchorSequence := registry.AnchorRoot()

		game := honestChallengerBeatsInvalidProposal(t, sys)
		// The challenger closes the invalid game and claims its credit once finality elapses.
		advanceL1To(sys, game.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)
		game.WaitForClaimedCredit(zkChallengerAddress(t, sys.L2ChainA.ChainID()))

		// The invalid proposal must not advance the anchor state.
		actualRoot, actualSequence := registry.AnchorRoot()
		t.Require().Equal(anchorRoot, actualRoot)
		t.Require().Equal(anchorSequence, actualSequence)
	})
}

// TestZK_HonestChallenger_SourceRecovers_ChallengesInvalidProposal checks the live challenger's
// recovery path when its super-root RPC is temporarily unresponsive. Two stalled requests prove the
// first call timed out and the challenger retried; after the source resumes, the same challenger must
// submit the on-chain challenge before the challenge window expires.
func TestZK_HonestChallenger_SourceRecovers_ChallengesInvalidProposal(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newOpNodeSystem(t)
	proxy := sys.ZKChallengerSuperRootRPCProxy()
	proxy.Stall()

	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)

	proxy.WaitForStalledRequests(t, 2)
	t.Require().Equal(proofs.ZKProposalUnchallenged, game.ProposalStatus())
	proxy.Resume()
	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
}

// TestZK_HonestChallenger_InvalidChainBProposal_ChallengesProposal checks that the challenger compares
// every chain in a multi-chain super root. Existing invalid-proposal coverage corrupts the first
// (Chain A) output; this corrupts only the second (Chain B) output and expects the same challenge.
func TestZK_HonestChallenger_InvalidChainBProposal_ChallengesProposal(gt *testing.T) {
	t := devtest.ParallelT(gt)
	multiSys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestProposer(),
	)
	sys := &multiSys.SingleChainInterop
	t.Require().Negative(multiSys.L2ChainA.ChainID().Cmp(multiSys.L2ChainB.ChainID()),
		"super-root entries are sorted by chain ID, so Chain B must be the second entry")

	factory := sys.DisputeGameFactory()
	proposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	timestamp, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().Len(outputRoots, 2)
	outputRoots[1][0] ^= 0xff
	game := factory.StartZKGame(proposer,
		proofs.WithL2SequenceNumber(timestamp),
		proofs.WithSuperRootFrom(outputRoots...),
	)

	game.WaitForProposalStatus(proofs.ZKProposalChallenged)
}

// TestZK_HonestChallenger_UnsafeProposal_ChallengerWins locks in the subtle rule that a proposal for
// a timestamp the node has not yet made safe has no canonical super root (Data == nil), which the
// honest challenger treats as invalid: it challenges and, with no proof submitted, resolves
// CHALLENGER_WINS. This is a distinct path from a mismatched-root proposal at an already-safe
// timestamp, and each super-root source reaches Data == nil differently, so both are exercised.
func TestZK_HonestChallenger_UnsafeProposal_ChallengerWins(gt *testing.T) {
	gt.Run("op-node", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		honestChallengerBeatsUnsafeProposal(t, newOpNodeSystem(t))
	})
	gt.Run("supernode", func(gt *testing.T) {
		t := devtest.ParallelT(gt)
		honestChallengerBeatsUnsafeProposal(t, newSupernodeSystem(t))
	})
}

// TestZK_HonestChallenger_ChildOfInvalidParent_ChallengerWins checks resolution ordering: a child
// game referencing an invalid parent resolves CHALLENGER_WINS by inheritance only after the honest
// challenger has resolved the parent CHALLENGER_WINS. Resolution ordering is source-agnostic, so it
// runs against the op-node source only.
func TestZK_HonestChallenger_ChildOfInvalidParent_ChallengerWins(gt *testing.T) {
	t := devtest.ParallelT(gt)
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
	advanceL1To(sys, parent.ClaimData().Deadline+1)
	parent.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
	child.WaitForGameStatus(gameTypes.GameStatusChallengerWon)
}
