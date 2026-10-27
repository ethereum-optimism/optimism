package zk

import (
	"math"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/zkproposer"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
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
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestProposer(),
	)
	factory := sys.DisputeGameFactory()
	_, anchorSequence := sys.AnchorStateRegistry(sys.L2ChainA).AnchorRoot()

	sys.StartZKProposer()

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
	futureSequence := supernode.EnsureInteropPaused(10, sys.L2CLA, sys.L2CLB)
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

// TestProposerProvesBelowHeadMessageOnTablePath proves a game whose span holds
// an executing message on chain B that references a block of chain A below A's
// block at that timestamp, on the proposer's default table path.
func TestProposerProvesBelowHeadMessageOnTablePath(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
		presets.WithZKProposerOption(sysgo.WithZKFastFinality()),
		presets.WithZKProposerOption(sysgo.WithZKMetrics()),
		// WithZK pins Jovian at genesis; executing messages need Lagoon.
		presets.WithDeployerOptions(sysgo.WithInteropAtGenesis()),
	)
	rng := rand.New(rand.NewSource(1234))

	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)
	initMsg := aliceA.SendRandomInitMessage(rng, aliceA.DeployEventLogger(), 2, 10)
	initBlock := bigs.Uint64Strict(initMsg.BlockNumber())
	// The margin keeps the initiating block below A's block at B's exec timestamp if B lags.
	sys.L2ELA.WaitForBlockNumber(initBlock + 3)
	execMsg := aliceB.SendExecMessage(initMsg)
	execTimestamp := sys.L2ChainB.TimestampForBlockNum(bigs.Uint64Strict(execMsg.BlockNumber()))
	headA, err := sys.L2ChainA.Escape().RollupConfig().TargetBlockNumber(execTimestamp)
	t.Require().NoError(err, "chain A block at the executing timestamp")
	t.Require().Less(initBlock, headA, "the initiating block must be below chain A's block at the executing timestamp")
	sys.SuperRoots.AwaitValidatedTimestamp(execTimestamp)

	proposer := sys.StartZKProposer()
	factory := sys.DisputeGameFactory()
	game := factory.WaitForZKGameAtIndex(0)
	// Proposer games chain in factory order, so the first one at or past execTimestamp contains it.
	for game.L2SequenceNumber() < execTimestamp {
		game = factory.WaitForZKGameAtIndex(int64(game.FactoryIndex() + 1))
	}

	game.WaitForProposalStatus(proofs.ZKProposalUnchallengedAndValidProofProvided)
	// One message references one block, so exactly one batch: more means a batch was
	// submitted again or the block was split across batches.
	proposer.VerifyState(
		zkproposer.LogBatchesSubmitted(1),
		zkproposer.ProvingFailures(0),
	)
}

// newReplacedBlockSystem starts an interop system with the ZK proposer, not yet
// running, whose L2 ELs refuse every request for a block hidden through the
// returned HiddenBlocks.
func newReplacedBlockSystem(t devtest.T, opts ...presets.Option) (*presets.SimpleInterop, *sysgo.HiddenBlocks) {
	hidden := sysgo.NewHiddenBlocks()
	sys := presets.NewSimpleInterop(t, append([]presets.Option{
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
		presets.WithZKProposerOption(sysgo.WithZKFastFinality()),
		presets.WithZKProposerOption(sysgo.WithZKMetrics()),
		presets.WithZKProposerOption(hidden.ZKProposerOption()),
		// WithZK pins Jovian at genesis; executing messages need Lagoon.
		presets.WithDeployerOptions(sysgo.WithInteropAtGenesis()),
	}, opts...)...)
	return sys, hidden
}

// sendDeniedMessage sends an invalid executing message on chain B, hides its
// block from the proposer's ELs, waits for the supernode to replace the block,
// and returns the block's timestamp. The message references a block of chain A
// below A's block at that timestamp.
func sendDeniedMessage(t devtest.T, sys *presets.SimpleInterop, hidden *sysgo.HiddenBlocks) uint64 {
	rng := rand.New(rand.NewSource(1234))
	aliceA := sys.FunderA.NewFundedEOA(eth.OneEther)
	aliceB := aliceA.AsEL(sys.L2ELB)
	sys.FunderB.Fund(aliceB, eth.OneEther)
	initMsg := aliceA.SendRandomInitMessage(rng, aliceA.DeployEventLogger(), 2, 10)
	initBlock := bigs.Uint64Strict(initMsg.BlockNumber())
	// Send once B is at A's next block's timestamp, so the executing message lands
	// after it, where A's block is above the initiating one whatever the block times.
	sys.L2ELB.WaitForTime(sys.L2ChainA.TimestampForBlockNum(initBlock + 1))
	execMsg := aliceB.SendInvalidExecMessage(initMsg)
	hidden.Hide(execMsg.BlockHash())
	execBlock := bigs.Uint64Strict(execMsg.BlockNumber())
	execTimestamp := sys.L2ChainB.TimestampForBlockNum(execBlock)
	headA, err := sys.L2ChainA.Escape().RollupConfig().TargetBlockNumber(execTimestamp)
	t.Require().NoError(err, "chain A block at the executing timestamp")
	t.Require().Less(initBlock, headA, "the initiating block must be below chain A's block at the executing timestamp")

	sys.SuperRoots.AwaitValidatedTimestamp(execTimestamp)
	// The EL exposes the replacement through an async forkchoice update, so wait on
	// it too before reading the block's contents.
	dsl.CheckAll(t,
		sys.L2CLB.ReachedFn(safety.CrossSafe, execBlock, 30),
		sys.L2ELB.ReachedFn(eth.Safe, execBlock, 30),
	)
	sys.L2ELB.AssertExecMessageNotInBlock(execMsg)
	atExec := sys.SuperRoots.SuperRootAtTimestamp(execTimestamp)
	optimistic := atExec.OptimisticAtTimestamp[sys.L2ChainB.ChainID()]
	t.Require().NotNil(optimistic.Output, "the optimistic block of chain B must have an output")
	t.Require().Equal(execMsg.BlockHash(), optimistic.Output.BlockHash,
		"the supernode must report the denied block as optimistic at its timestamp")
	t.Require().NotNil(atExec.Data, "the supernode must have validated the executing timestamp")
	superV1, ok := atExec.Data.Super.(*eth.SuperV1)
	t.Require().True(ok, "the super root at the executing timestamp must be a SuperV1")
	chainB := slices.IndexFunc(superV1.Chains, func(c eth.ChainIDAndOutput) bool {
		return c.ChainID.Cmp(sys.L2ChainB.ChainID()) == 0
	})
	t.Require().NotEqual(-1, chainB, "the super root must commit an output for chain B")
	t.Require().NotEqual(optimistic.OutputRoot, superV1.Chains[chainB].Output,
		"the super root must commit the replacement's output, not the denied block's")
	return execTimestamp
}

// TestProposerProvesGameSpanningReplacedBlock proves a game whose span holds a
// block of chain B that the supernode replaced for an invalid executing message.
// The supernode keeps reporting the denied block as optimistic at its timestamp,
// and the proposer must read it through kona-host, since the L2 node may have
// pruned it: the proposer's ELs refuse every request for the denied block. The
// message references a block of chain A below A's block at that timestamp, so
// the scan must see the denied block's messages to batch it.
func TestProposerProvesGameSpanningReplacedBlock(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys, hidden := newReplacedBlockSystem(t)
	execTimestamp := sendDeniedMessage(t, sys, hidden)

	proposer := sys.StartZKProposer()
	factory := sys.DisputeGameFactory()
	game := factory.WaitForZKGameAtIndex(0)
	// Proposer games chain in factory order, so the first one at or past execTimestamp contains it.
	for game.L2SequenceNumber() < execTimestamp {
		game = factory.WaitForZKGameAtIndex(int64(game.FactoryIndex() + 1))
	}
	game.WaitForProposalStatus(proofs.ZKProposalUnchallengedAndValidProofProvided)

	// One batch means the scan read the denied block's message to chain A's block;
	// no failure means no attempt stopped on the denied block or a missed reference.
	proposer.VerifyState(
		zkproposer.LogBatchesSubmitted(1),
		zkproposer.ProvingFailures(0),
	)
	t.Require().Positive(hidden.ForwardedRequests(), "the proposer must read the L2 ELs through the hiding proxies")
	t.Require().Zero(hidden.RefusedRequests(), "the proposer must not ask the L2 ELs for the denied block")
}

// TestProposerProvesGameAgreeingOnReplacedBlock proves a game whose parent claims
// the timestamp of a block of chain B that the supernode replaced for an invalid
// executing message. The supernode reports the denied block as optimistic at that
// timestamp, so the game's first chunk must read its agreed output, the
// replacement's, through kona-host. A foreign game at the denied timestamp makes
// it the agreed timestamp of the proposer's first game, whatever its span.
func TestProposerProvesGameAgreeingOnReplacedBlock(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys, hidden := newReplacedBlockSystem(t)
	execTimestamp := sendDeniedMessage(t, sys, hidden)
	proposer, game := startProposerOnForeignGame(t, sys, execTimestamp)
	requireProvenReadingReplacedAgreedOutput(t, proposer, game, hidden)
}

// TestProposerProvesChunkAgreeingOnReplacedBlockInside proves a game whose single
// chunk agrees on the timestamp of a block of chain B that the supernode
// replaced, after the chunk's first timestamp. Chain B makes a block every
// second, so the next timestamp's optimistic block, whose output the chunk
// claims, is not the replacement: the chunk must read the replacement's output
// through kona-host at the denied timestamp. A foreign game one timestamp before
// the denied block makes the denied timestamp the second agreed one of the
// proposer's first game, which one chunk proves whole.
func TestProposerProvesChunkAgreeingOnReplacedBlockInside(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys, hidden := newReplacedBlockSystem(t,
		presets.WithL2BlockTimes(map[eth.ChainID]uint64{sysgo.DefaultL2AID: 2, sysgo.DefaultL2BID: 1}),
		presets.WithZKProposerOption(sysgo.WithZKRangeSplitCount(1)),
	)
	execTimestamp := sendDeniedMessage(t, sys, hidden)
	proposer, game := startProposerOnForeignGame(t, sys, execTimestamp-1)
	t.Require().Greater(game.L2SequenceNumber(), execTimestamp,
		"the game must claim past the denied timestamp, so its chunk agrees on it")
	requireProvenReadingReplacedAgreedOutput(t, proposer, game, hidden)
}

// startProposerOnForeignGame starts a foreign game that claims parentTimestamp,
// then the proposer, and returns the proposer and its game on the foreign one.
func startProposerOnForeignGame(t devtest.T, sys *presets.SimpleInterop, parentTimestamp uint64) (*zkproposer.ZKProposer, *proofs.ZKGame) {
	factory := sys.DisputeGameFactory()
	foreign := factory.StartZKGame(sys.FunderL1.NewFundedEOA(eth.OneEther), proofs.WithL2SequenceNumber(parentTimestamp))
	proposer := sys.StartZKProposer()
	game := factory.WaitForZKGameAtIndex(int64(foreign.FactoryIndex() + 1))
	t.Require().Equal(foreign.FactoryIndex(), game.ParentIndex(), "the proposer must build on the foreign game")
	return proposer, game
}

// requireProvenReadingReplacedAgreedOutput waits for game to be proven and
// requires that the proposer never asked the L2 ELs for the denied block. The
// game's chunk agrees on the denied timestamp, whose super root commits the
// replacement's output in place of the optimistic one (see sendDeniedMessage),
// so the game proves only if the proposer read that output through kona-host.
func requireProvenReadingReplacedAgreedOutput(t devtest.T, proposer *zkproposer.ZKProposer, game *proofs.ZKGame, hidden *sysgo.HiddenBlocks) {
	game.WaitForProposalStatus(proofs.ZKProposalUnchallengedAndValidProofProvided)
	proposer.VerifyState(zkproposer.ProvingFailures(0))
	t.Require().Positive(hidden.ForwardedRequests(), "the proposer must read the L2 ELs through the hiding proxies")
	t.Require().Zero(hidden.RefusedRequests(), "the proposer must not ask the L2 ELs for the denied block")
}

// TestProposerBisectsSpanOverBudget proves a game whose chunks are each at least
// two timestamps long under a gas limit with no budget, so every chunk is over
// its estimate and is bisected into single timestamps, which are requested
// anyway. The game proves only if the halves' synthesized inputs, witnesses,
// and outputs chain back into the aggregation in order.
func TestProposerBisectsSpanOverBudget(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t,
		presets.WithZK(),
		presets.WithoutHonestChallenger(),
		presets.WithoutHonestProposer(),
		presets.WithZKProposerOption(sysgo.WithZKFastFinality()),
		presets.WithZKProposerOption(sysgo.WithZKMetrics()),
		// Keeps each of the default 16 chunks at least two timestamps long.
		presets.WithZKProposerOption(sysgo.WithZKProposalInterval(32*time.Second)),
		presets.WithZKProposerOption(sysgo.WithZKRangeGasLimit(1)),
	)
	proposer := sys.StartZKProposer()
	sys.DisputeGameFactory().WaitForZKGameAtIndex(0).
		WaitForProposalStatus(proofs.ZKProposalUnchallengedAndValidProofProvided)
	// Without a bisection the game would still prove if the estimate check stopped
	// gating splittable spans, so assert that one happened.
	proposer.VerifyState(
		zkproposer.SpanBisectionsAtLeast(1),
		zkproposer.ProvingFailures(0),
	)
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
