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

const (
	zkSafetyTestProposalInterval  = time.Minute
	zkSafetyTestChallengeDuration = 5 * time.Minute
)

// The live proposer keeps creating games while these tests run (including
// during time travel), so assertions only reference the specific games
// returned by WaitForZKGameAtIndex and never assume a total game count.

// TestProposerChainsSecondZKGameOnFirst covers the create path end to end: the
// proposer's first game is a well-formed root game (max-uint32 parent sentinel,
// sequence number beyond the anchor) and its second game chains on the first.
// Broader root-game lifecycle behavior (challenge, prove, anchor) is covered by
// TestChallengedValidProposalAnchors.
func TestProposerChainsSecondZKGameOnFirst(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t, presets.WithZK())
	factory := sys.DisputeGameFactory()
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	game0 := factory.WaitForZKGameAtIndex(0)
	t.Require().Equal(uint32(math.MaxUint32), game0.ParentIndex(),
		"first proposer game must be a root game using the max-uint32 parent sentinel")
	t.Require().Greater(game0.L2SequenceNumber(), anchorSequence,
		"root game must propose a sequence number beyond the anchor")

	game1 := factory.WaitForZKGameAtIndex(1)
	t.Require().Equal(uint32(0), game1.ParentIndex(),
		"second proposer game must chain on the first game at factory index 0")
	t.Require().Greater(game1.L2SequenceNumber(), game0.L2SequenceNumber(),
		"second game must propose a sequence number beyond its parent")
}

func TestProposerBuildsOnValidGameFromAnotherProposer(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithZKChallengeDuration(zkSafetyTestChallengeDuration),
		presets.WithoutHonestChallenger(),
		presets.WithZKProposerOption(sysgo.WithZKProposalInterval(zkSafetyTestProposalInterval)),
	)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	foreignProposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	foreignSequence, _ := factory.WaitForSafeSuperRootAfter(game0.L2SequenceNumber())
	foreignGame := factory.StartZKGame(
		foreignProposer,
		proofs.WithZKParent(game0.FactoryIndex()),
		proofs.WithL2SequenceNumber(foreignSequence),
	)
	t.Require().Equal(uint32(1), foreignGame.FactoryIndex(),
		"foreign game must be created before the honest proposer reaches its next interval")

	sys.AdvanceTime(zkSafetyTestProposalInterval)
	honestChild := factory.WaitForZKGameAtIndex(int64(foreignGame.FactoryIndex() + 1))
	t.Require().Equal(foreignGame.FactoryIndex(), honestChild.ParentIndex(),
		"honest proposer must adopt a valid foreign game as its canonical parent")
	t.Require().Greater(honestChild.L2SequenceNumber(), foreignGame.L2SequenceNumber(),
		"honest proposer child must advance beyond the foreign parent")
}

func TestProposerRejectsValidChildOfInvalidGame(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithZKChallengeDuration(zkSafetyTestChallengeDuration),
		presets.WithoutHonestChallenger(),
		presets.WithZKProposerOption(sysgo.WithZKProposalInterval(zkSafetyTestProposalInterval)),
	)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	foreignProposer := sys.FunderL1.NewFundedEOA(eth.OneEther)
	invalidSequence, outputRoots := factory.WaitForSafeSuperRootAfter(game0.L2SequenceNumber())
	t.Require().NotEmpty(outputRoots)
	outputRoots[0][0] ^= 0xff
	invalidGame := factory.StartZKGame(
		foreignProposer,
		proofs.WithZKParent(game0.FactoryIndex()),
		proofs.WithL2SequenceNumber(invalidSequence),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	t.Require().Equal(uint32(1), invalidGame.FactoryIndex(),
		"invalid game must be created before the honest proposer reaches its next interval")

	// Force a proposal after the invalid game is visible. Building on game0
	// proves the proposer classified the higher-sequence foreign game as
	// invalid before its child is introduced.
	sys.AdvanceTime(zkSafetyTestProposalInterval)
	validSibling := factory.WaitForZKGameAtIndex(int64(invalidGame.FactoryIndex() + 1))
	t.Require().Equal(game0.FactoryIndex(), validSibling.ParentIndex(),
		"honest proposer must reject the invalid foreign game")

	orphanSequence, _ := factory.WaitForSafeSuperRootAfter(validSibling.L2SequenceNumber())
	orphan := factory.StartZKGame(
		foreignProposer,
		proofs.WithZKParent(invalidGame.FactoryIndex()),
		proofs.WithL2SequenceNumber(orphanSequence),
	)
	t.Require().Equal(uint32(3), orphan.FactoryIndex(),
		"orphan must be created before the honest proposer reaches its next interval")

	sys.AdvanceTime(zkSafetyTestProposalInterval)
	honestChild := factory.WaitForZKGameAtIndex(int64(orphan.FactoryIndex() + 1))
	t.Require().Equal(validSibling.FactoryIndex(), honestChild.ParentIndex(),
		"honest proposer must not extend a valid claim whose parent was rejected")
	t.Require().Greater(honestChild.L2SequenceNumber(), validSibling.L2SequenceNumber(),
		"honest proposer must continue advancing the connected valid branch")
}

func TestProposerDefersGameUntilSuperRootRPCCatchesUp(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithZKChallengeDuration(zkSafetyTestChallengeDuration),
		presets.WithoutHonestChallenger(),
		presets.WithZKProposerOption(sysgo.WithZKProposalInterval(zkSafetyTestProposalInterval)),
	)
	factory := sys.DisputeGameFactory()

	game0 := factory.WaitForZKGameAtIndex(0)
	safeSequence, outputRoots := factory.WaitForSafeSuperRootAfter(game0.L2SequenceNumber())
	sys.L2BatcherA.Stop()
	sys.L2BatcherB.Stop()
	batchersStopped := true
	t.Cleanup(func() {
		if batchersStopped {
			sys.L2BatcherA.Start()
			sys.L2BatcherB.Start()
		}
	})

	pendingSequence := safeSequence + uint64(zkSafetyTestProposalInterval/time.Second)
	t.Require().Nil(sys.SuperRoots.SuperRootAtTimestamp(pendingSequence).Data,
		"super-root RPC must be behind the future game's timestamp")
	// Keep the game invalid after the RPC catches up, so the proposer must
	// discard it instead of adopting it after the pending period.
	outputRoots[0][0] ^= 0xff
	pendingGame := factory.StartZKGame(
		sys.FunderL1.NewFundedEOA(eth.OneEther),
		proofs.WithZKParent(game0.FactoryIndex()),
		proofs.WithL2SequenceNumber(pendingSequence),
		proofs.WithFutureProposal(),
		proofs.WithSuperRootFrom(outputRoots...),
	)
	t.Require().Equal(uint32(1), pendingGame.FactoryIndex(),
		"future game must be created before another honest proposal")

	// Let the proposer observe the game while the RPC still reports no
	// canonical super root at its timestamp.
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	sys.L1EL.WaitForBlockNumber(l1Head.Number + 5)

	sys.L2BatcherA.Start()
	sys.L2BatcherB.Start()
	batchersStopped = false
	sys.AdvanceTime(zkSafetyTestProposalInterval)
	factory.WaitForSafeSuperRootAfter(pendingSequence - 1)

	honestChild := factory.WaitForZKGameAtIndex(int64(pendingGame.FactoryIndex() + 1))
	t.Require().Equal(game0.FactoryIndex(), honestChild.ParentIndex(),
		"game unavailable from the super-root RPC must never become a proposal parent")
	t.Require().Greater(honestChild.L2SequenceNumber(), game0.L2SequenceNumber(),
		"honest proposer must resume creating games after the super-root RPC catches up")
}

func TestProposerResolvesOwnUnchallengedGame(gt *testing.T) {
	t := devtest.SerialT(gt)
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
	t := devtest.SerialT(gt)
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
	advanceL1To(&sys.SingleChainInterop, game0.ResolvedAt()+uint64(zkFinalityDelay/time.Second)+1)

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

// zkProposerAddress derives the address the sysgo ZK proposer signs with: the
// ProposerRole dev key for the proof chain (chain A).
func zkProposerAddress(t devtest.T, sys *presets.SimpleInterop) common.Address {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err, "derive dev keys")
	addr, err := keys.Address(devkeys.ProposerRole.Key(sys.L2ChainA.ChainID().ToBig()))
	t.Require().NoError(err, "derive proposer address")
	return addr
}
