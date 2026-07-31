package zk

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// The live proposer keeps creating games while these tests run (including
// during time travel), so assertions only reference the specific games
// returned by WaitForZKGameAtIndex and never assume a total game count.

// TestProposerChainsSecondZKGameOnFirst covers the create path end to end: the
// proposer's first game is a well-formed root game (max-uint32 parent sentinel,
// non-zero sequence number) and its second game chains on the first.
// Broader root-game lifecycle behavior (challenge, prove, anchor) is covered by
// TestChallengedValidProposalAnchors.
func TestProposerChainsSecondZKGameOnFirst(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithZK())
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	t.Require().Equal(uint32(math.MaxUint32), game0.ParentIndex(),
		"first proposer game must be a root game using the max-uint32 parent sentinel")
	// TODO(#22086): strengthen back to an anchor comparison by recording the
	// anchor before the proposer starts, once the start-proposer-mid-test
	// hook from #22105 is available (adopt whichever PR lands first).
	// Not compared against the live anchor: with the always-on proposer the
	// anchor may already have advanced past game0 by the time this test
	// runs (the proposer-creates-beyond-the-anchor rule is unit-tested on
	// the Rust side in proposal_timing).
	t.Require().NotZero(game0.L2SequenceNumber(),
		"root game must carry a super-root timestamp")

	game1 := factory.WaitForZKGameAtIndex(1)
	t.Require().Equal(uint32(0), game1.ParentIndex(),
		"second proposer game must chain on the first game at factory index 0")
	t.Require().Greater(game1.L2SequenceNumber(), game0.L2SequenceNumber(),
		"second game must propose a sequence number beyond its parent")
}

func TestProposerBuildsOnValidGameFromAnotherProposer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
	)
	factory := sys.DisputeGameFactory()

	foreignProposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	foreignGame := factory.StartZKGame(foreignProposer)
	t.Require().Equal(uint32(0), foreignGame.FactoryIndex(),
		"foreign game must be the first game observed by the honest proposer")

	sys.StartZKProposer()

	honestChild := factory.WaitForZKGameAtIndex(int64(foreignGame.FactoryIndex() + 1))
	t.Require().Equal(foreignGame.FactoryIndex(), honestChild.ParentIndex(),
		"honest proposer must adopt a valid foreign game as its canonical parent")
	t.Require().Greater(honestChild.L2SequenceNumber(), foreignGame.L2SequenceNumber(),
		"honest proposer child must advance beyond the foreign parent")
}

func TestProposerRejectsValidChildOfInvalidGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
	)
	factory := sys.DisputeGameFactory()

	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	foreignProposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	invalidSequence, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	invalidGame := factory.StartZKGame(
		foreignProposer,
		proofs.WithL2SequenceNumber(invalidSequence),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	t.Require().Equal(uint32(0), invalidGame.FactoryIndex())

	orphanSequence, _ := factory.WaitForSafeSuperRootAfter(invalidGame.L2SequenceNumber())
	orphan := factory.StartZKGame(
		foreignProposer,
		proofs.WithZKParent(invalidGame.FactoryIndex()),
		proofs.WithL2SequenceNumber(orphanSequence),
	)
	t.Require().Equal(uint32(1), orphan.FactoryIndex())

	sys.StartZKProposer()

	honestChild := factory.WaitForZKGameAtIndex(int64(orphan.FactoryIndex() + 1))
	t.Require().Equal(uint32(math.MaxUint32), honestChild.ParentIndex(),
		"honest proposer must not extend a valid claim whose parent was rejected")
}

func TestProposerDoesNotBuildOnGameCreatedAheadOfSuperRootRPC(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
	)
	factory := sys.DisputeGameFactory()

	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()
	_, outputRoots := factory.WaitForSafeSuperRootAfter(anchorSequence)

	supernode := sys.Supernode()
	futureSequence := supernode.EnsureInteropPaused(sys.L2CLA, sys.L2CLB, 10)
	t.Cleanup(supernode.ResumeInterop)
	t.Require().Nil(sys.SuperRoots.SuperRootAtTimestamp(futureSequence).Data,
		"super-root RPC must be behind the future game's timestamp")
	// Keep the game invalid once its canonical root becomes available.
	outputRoots[0][0] ^= 0xff
	futureGame := factory.StartZKGame(
		sys.FunderL1.NewFundedEOA(eth.OneEther),
		proofs.WithL2SequenceNumber(futureSequence),
		proofs.WithFutureProposal(),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	t.Require().Equal(uint32(0), futureGame.FactoryIndex(),
		"future game must be the first game observed by the honest proposer")

	sys.StartZKProposer()
	honestRoot := factory.WaitForZKGameAtIndex(int64(futureGame.FactoryIndex() + 1))
	t.Require().Equal(uint32(math.MaxUint32), honestRoot.ParentIndex(),
		"honest proposer must not extend a game whose super root is unavailable")

	supernode.ResumeInterop()
	sys.SuperRoots.AwaitValidatedTimestamp(futureSequence)

	honestChild := factory.WaitForZKGameAtIndex(int64(honestRoot.FactoryIndex() + 1))
	t.Require().Equal(honestRoot.FactoryIndex(), honestChild.ParentIndex(),
		"honest proposer must reject the mismatched future game after the super-root RPC catches up")
}

func TestProposerResolvesOwnUnchallengedGame(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// The challenger resolves all games, not just those it challenges; disable
	// it so this test proves the proposer alone drives resolution.
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
	)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	advanceL1To(&sys.SingleChainInterop, game0.ClaimData().Deadline+1)

	// The proposer's resolution task must resolve its own unchallenged game;
	// the test never calls resolve itself.
	game0.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
}

func TestProposerClaimsBondAfterResolution(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// The challenger can resolve games and claim credit on the proposer's
	// behalf; disable it so this test proves the proposer alone resolves,
	// unlocks, and claims.
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
	)
	factory := sys.DisputeGameFactory()
	proposerAddr := zkProposerAddress(t, sys)
	weth := factory.DelayedWETH(factory.ZKGameImpl().Args.Weth)

	game0 := factory.WaitForZKGameAtIndex(0)
	advanceL1To(&sys.SingleChainInterop, game0.ClaimData().Deadline+1)
	game0.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	advanceL1To(&sys.SingleChainInterop, game0.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)

	// Phase 1 (unlock): the proposer's claim task closes the game and unlocks
	// its bond credit into DelayedWETH.
	var withdrawal proofs.ZKWithdrawal
	t.Require().Eventuallyf(func() bool {
		withdrawal = weth.Withdrawal(game0.Address, proposerAddr)
		return withdrawal.Amount.Sign() > 0
	}, 2*time.Minute, time.Second, "proposer did not unlock its bond credit")

	// Phase 2 (payout): only possible once the WETH withdrawal delay has
	// elapsed in L1 time. The payout transfer itself is enforced by
	// DelayedWETH.withdraw, so "withdrawal fully drained and credit zeroed"
	// is the deterministic claim-completion observable; a raw balance-growth
	// check would race the live proposer bonding new games in this window.
	advanceL1To(&sys.SingleChainInterop, withdrawal.MaturesAt(weth.Delay()))

	t.Require().Eventuallyf(func() bool {
		return weth.Withdrawal(game0.Address, proposerAddr).Amount.Sign() == 0 &&
			game0.Credit(proposerAddr).IsZero()
	}, 2*time.Minute, time.Second, "proposer did not claim its bond after the withdrawal delay")

	t.Require().True(game0.Credit(proposerAddr).IsZero(), "claimed game must hold no credit for the proposer")
	t.Require().Zero(weth.Withdrawal(game0.Address, proposerAddr).Amount.Sign(), "claimed game must hold no pending withdrawal")
}

// TestProposerFastFinalityProvesAtCreation covers fast finality mode end to
// end: the proposer proves its own game while it is still unchallenged, and
// the game resolves DefenderWins BEFORE the challenge window elapses. Nothing
// ever challenges, and L1 never time-travels past the deadline before
// resolution (every other test advances L1 past the deadline first).
func TestProposerFastFinalityProvesAtCreation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// The challenger resolves all games, not just those it challenges; disable
	// it so this test proves the proposer alone proves and resolves.
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithZKProposerOption(sysgo.WithZKFastFinality()),
	)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	// The challenge deadline; nothing challenges in this test, so it is
	// never rewritten.
	deadline := game0.ClaimData().Deadline

	// Fast finality proves the game with NO challenge.
	game0.WaitForProposalStatus(proofs.ZKProposalUnchallengedAndValidProofProvided)
	t.Require().Equal(zkProposerAddress(t, sys), game0.ClaimData().Prover,
		"the proposer itself must be the prover")

	// The proven game resolves without waiting out the challenge window:
	// deliberately no advanceL1To before this wait.
	game0.WaitForGameStatus(gameTypes.GameStatusDefenderWon)
	t.Require().Less(game0.ResolvedAt(), deadline,
		"fast finality must resolve before the challenge window elapses")

	// The early-resolved game feeds the anchor like any other win.
	advanceL1To(&sys.SingleChainInterop, game0.ResolvedAt()+uint64(presets.DefaultZKFinalityDelay/time.Second)+1)
	sys.AnchorStateRegistry(sys.L2ChainA).WaitForAnchorRootAtLeast(game0)
}

// zkProposerAddress derives the address the sysgo ZK proposer signs with: the
// ProposerRole dev key for the proof chain (chain A).
func zkProposerAddress(t devtest.T, sys *presets.SimpleInterop) common.Address {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "derive dev keys")
	addr, err := keys.Address(devkeys.ProposerRole.Key(sys.L2ChainA.ChainID().ToBig()))
	t.Require().NoError(err, "derive proposer address")
	return addr
}
