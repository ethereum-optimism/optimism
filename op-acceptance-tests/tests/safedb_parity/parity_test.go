package safedb_parity

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// verifySafeHeadRecordsMatch asserts that kona-node and op-node give the same safe head for every
// recorded L1 block, walking each node's records and asking the other, and the same super-root
// answer for every L2 block both have recorded. The op-node-driven walk also fails if kona-node
// lost any record op-node still has, since op-node is never restarted in these tests.
func verifySafeHeadRecordsMatch(t devtest.T, sys *paritySystem) {
	sys.Kona.VerifySafeHeadDatabaseMatches(sys.OpNode)
	sys.OpNode.VerifySafeHeadDatabaseMatches(sys.Kona)
	verifySuperRootAnswersMatch(t, sys)
}

// verifySuperRootAnswersMatch compares the super-root answer of both nodes for every L2 block
// from the higher recorded floor to the lower local-safe head. The data section carries the L1
// block the record says made the block safe, which is the answer op-challenger and op-proposer act
// on. A node's own progress fields are not compared: the two nodes need not be at the same L1
// block, only agree about every block they have both passed.
func verifySuperRootAnswersMatch(t devtest.T, sys *paritySystem) {
	require := t.Require()
	opNodeStatus, konaStatus := sys.OpNode.SyncStatus(), sys.Kona.SyncStatus()
	to := min(opNodeStatus.LocalSafeL2.Number, konaStatus.LocalSafeL2.Number)
	// Every block up to `to` was derived from L1 at or below either node's current L1 block, so
	// once both have processed past the higher one, every record the range depends on is final.
	l1Derived := max(opNodeStatus.CurrentL1.Number, konaStatus.CurrentL1.Number)
	sys.OpNode.AwaitMinL1Processed(l1Derived)
	sys.Kona.AwaitMinL1Processed(l1Derived)

	from := max(sys.OpNode.SafeHeadDBFloor(), sys.Kona.SafeHeadDBFloor())
	require.LessOrEqual(from, to, "no L2 block is recorded by both nodes")
	t.Logger().Info("Comparing super-root answers", "fromL2Block", from, "toL2Block", to)

	opNodeRoots := dsl.NewOpNodeSuperRoots(sys.OpNode)
	konaRoots := dsl.NewOpNodeSuperRoots(sys.Kona)
	for number := from; number <= to; number++ {
		timestamp := sys.L2Network.TimestampForBlockNum(number)
		expected := opNodeRoots.SuperRootAtTimestamp(timestamp)
		actual := konaRoots.SuperRootAtTimestamp(timestamp)
		require.Equalf(expected.ChainIDs, actual.ChainIDs, "chain ids differ at L2 block %d", number)
		require.Equalf(expected.OptimisticAtTimestamp, actual.OptimisticAtTimestamp, "optimistic output differs at L2 block %d", number)
		require.Equalf(expected.Data, actual.Data, "super-root data differs at L2 block %d", number)
	}
}

// deriveTogether lets both verifiers derive a few more L2 blocks and brings kona-node level with
// op-node, so every later comparison covers blocks both have recorded.
func deriveTogether(t devtest.T, sys *paritySystem, blocks uint64) {
	dsl.CheckAll(t,
		sys.OpNode.AdvancedFn(safety.LocalSafe, blocks, 60),
		sys.Kona.AdvancedFn(safety.LocalSafe, blocks, 60))
	sys.Kona.InSync(sys.OpNode, safety.LocalSafe, 30)
}

func TestSafeHeadRecordsMatchWhileDeriving(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t)

	deriveTogether(t, sys, 5)

	verifySafeHeadRecordsMatch(t, sys)
}

func TestSafeHeadRecordsMatchAfterL1Reorg(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t)
	deriveTogether(t, sys, 2)

	// Build L1 by hand until the sequencer's safe head has an L1 origin on the hand-built chain,
	// so the reorg below unwinds batches rather than empty L1 blocks, and make sure both
	// verifiers have recorded that safe head so the reorg truncates a record on each.
	sys.L1Network.WaitForBlock()
	sys.L1CL.Stop()
	startL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	l2SafeBeforeReorg := sys.TestSequencer.BuildL1UntilSafeOriginPast(t, sys.L1Network.ChainID(), sys.SequencerEL, startL1Block.Number)
	dsl.CheckAll(t,
		sys.OpNodeEL.ReachedFn(eth.Safe, l2SafeBeforeReorg.Number, 30),
		sys.KonaEL.ReachedFn(eth.Safe, l2SafeBeforeReorg.Number, 30))

	l1BlockBeforeReorg := sys.L1EL.BlockRefByNumber(l2SafeBeforeReorg.L1Origin.Number)
	t.Logger().Info("Reorging the L1 block the safe head was derived from", "l1", l1BlockBeforeReorg, "l2Safe", l2SafeBeforeReorg)
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), l1BlockBeforeReorg.ParentHash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(l1BlockBeforeReorg, 5)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)

	// Both verifiers must reset and derive past the reorg on the new L1 chain.
	sys.OpNodeEL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 60)
	sys.KonaEL.WaitL1OriginHash(eth.Safe, l1BlockAfterReorg.ID(), 60)
	deriveTogether(t, sys, 3)

	verifySafeHeadRecordsMatch(t, sys)
}

func TestSafeHeadRecordsMatchAfterKonaCLRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t)
	deriveTogether(t, sys, 3)

	// Restart the CL only, so the EL keeps the chain and the CL resumes against existing data.
	// kona-node only leaves EL sync once a gossiped block arrives, so it must be re-peered.
	sys.Kona.ManagePeer(sys.OpNode)
	sys.Kona.Stop()
	sys.OpNode.Advanced(safety.LocalSafe, 3, 60)
	sys.Kona.Start()
	deriveTogether(t, sys, 1)

	verifySafeHeadRecordsMatch(t, sys)
}

func TestSafeHeadRecordsMatchAfterKonaCLAndELRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystem(t)
	deriveTogether(t, sys, 3)

	// Restart CL and EL together. The EL keeps its data dir, so the CL resumes from the EL's
	// heads rather than from genesis. kona-node only leaves EL sync once a gossiped block
	// arrives, so it must be re-peered.
	sys.Kona.ManagePeer(sys.OpNode)
	sys.Kona.Stop()
	sys.KonaEL.Stop()
	sys.OpNode.Advanced(safety.LocalSafe, 3, 60)
	sys.KonaEL.Start()
	sys.Kona.Start()
	sys.KonaEL.PeerWith(sys.OpNodeEL)
	deriveTogether(t, sys, 1)

	verifySafeHeadRecordsMatch(t, sys)
}

func TestDisabledSafeHeadDatabaseAnswersLikeOpNode(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newParitySystemWithoutSafeDB(t)
	require := t.Require()
	deriveTogether(t, sys, 2)
	l1Block := sys.OpNode.SyncStatus().CurrentL1.Number

	_, opNodeErr := sys.OpNode.SafeHeadAtL1Block(l1Block)
	_, konaErr := sys.Kona.SafeHeadAtL1Block(l1Block)

	require.ErrorContains(opNodeErr, safedb.ErrNotEnabled.Error(), "op-node must refuse the query without a safe head database")
	require.ErrorContains(konaErr, safedb.ErrNotEnabled.Error(), "kona-node must report a disabled database the way op-node does")
	require.NotErrorIs(konaErr, safedb.ErrNotFound, "a disabled database is not an empty one")
}
