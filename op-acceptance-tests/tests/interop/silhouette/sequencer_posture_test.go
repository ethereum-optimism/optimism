package silhouette

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// THE SEQUENCER POSTURE, END TO END.
//
// The tests in silhouette_test.go are about the VERIFIER: a node whose entire access to P is L1, and
// whose verdicts about P's messages are therefore statements about proofs. They deliberately leave
// the sequencer side ordinary, and TestSilhouetteCrossChainPinsThenAdvances closes with the contrast
// that makes the verifier's result sharp — the sequencer supernode, holding P's receipts in the
// execution client it drives, still cannot cross-safe A's block, because P has no batcher and P's
// public local-safe label never moves.
//
// That contrast is hazard 3, and on a real cluster it is not a curiosity: the cross-safety round gates
// on EVERY chain in the dependency set, so a P whose public label never moves freezes chain A's
// cross-safe frontier cluster-wide, permanently, from a chain that is perfectly healthy.
//
// These tests are the fix, on the same preset with one option added. The system they run on is the
// production shape:
//
//   - P is sequenced by the preset's own supernode, on P's REAL execution client, executing real
//     transactions. Nothing about that changes.
//   - that same supernode ALSO runs P as a silhouette chain in the `proven-head` posture: P's public
//     labels come from the proven head, walked out of L1 by the same data source every verifier runs.
//   - the second supernode is still there, still deriving P from proofs alone, and still the only
//     node whose verdict about P is a statement about proofs.
//
// So the claim is not "the sequencer can cross-safe A" on its own — that would be satisfiable by a
// node that trusted itself. It is that BOTH supernodes reach the same verdict about the same block by
// the same proofs, one of which has P's receipts and does not use them (G4 D1).

// sequencerPostureSystem brings up the preset with l2b proof-carried AND the sequencing supernode in
// the proven-head posture.
func sequencerPostureSystem(gt *testing.T) (devtest.T, *presets.TwoL2SupernodeInterop) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithSilhouetteChain(presets.SilhouetteChainB),
		presets.WithSilhouetteSequencerPosture())
	t.Require().NotNil(sys.Silhouette, "preset did not wire the silhouette half")
	t.Require().True(sys.Silhouette.SequencerPosture(),
		"the sequencing supernode is not in the proven-head posture, so these tests are about the wrong system")
	return t, sys
}

// TestSequencerPostureKeepsSequencingAndTakesLabelsFromProofs is the shape gate, and every assertion
// is one that would fail if the posture had taken something away from the block producer.
//
// The two halves are the point. P must still be REALLY sequencing on a REAL execution client — the
// verifier posture replaces that client with a shim that executes nothing, and applying that to the
// sequencer is exactly the failure this posture exists to avoid. And P's public label on that node
// must come from the proof stream, at proof cadence, and must be honest about the fact that the
// virtual node's own label has not moved.
func TestSequencerPostureKeepsSequencingAndTakesLabelsFromProofs(gt *testing.T) {
	t, sys := sequencerPostureSystem(gt)
	require := t.Require()
	sil := sys.Silhouette

	// (1) P is sequencing, on a real execution client, executing real transactions. A shim would
	// refuse to build anything and a deposits-only chain would carry no user transaction, so a
	// transaction landing in a block is the discriminating evidence.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	eventLogger := bob.DeployEventLogger()
	require.NotEqual(common.Address{}, eventLogger,
		"nothing was deployed on P, so its execution client is not executing")
	// A contract deployed and then USED is the strong form: SendInitMessage calls into it and the DSL
	// requires a successful receipt, so a shim — which executes nothing and would refuse to build the
	// block at all — cannot produce this.
	initMsg := bob.SendInitMessage(interop.RandomInitTrigger(rand.New(rand.NewSource(1)), eventLogger, 2, 8))
	initBlock := initMsg.BlockID()
	require.Equal(initBlock.Hash, sys.L2ELB.BlockRefByNumber(initBlock.Number).Hash,
		"P's execution client does not hold the block its own sequencer produced")

	// (2) the virtual node's OWN public label is still at genesis, and that is correct rather than a
	// symptom. There is no batcher behind it; a node reporting anything else would be inventing a
	// safety claim. The posture does not touch these labels and this asserts that it does not.
	require.Equal(uint64(0), sys.L2BSupernodeCL.HeadBlockRef(safety.LocalSafe).Number,
		"the sequencing supernode's own derivation advanced a safe label for a chain nobody batches")

	// (3) the label the CLUSTER consumes is the proven head, and before any proof lands it is honestly
	// absent rather than optimistic.
	status := sil.SequencerProvenHead(t)
	require.Nil(status.Head, "the sequencer claims proven history with no proof batch posted")
	require.Greater(uint64(status.PipelineSeqWindowSize), uint64(status.CommittedSeqWindowSize),
		"the sequencer's pipeline is running the COMMITTED window, so the empty-batch bomb is armed")

	// (4) a proof batch, and the proven head follows it — on the node that already had the blocks in
	// its own execution client and is deliberately not reading them from there.
	batch := sil.Submitter.SubmitNext()
	require.NotNil(batch, "P produced no blocks to batch")
	last := batch.Blocks[len(batch.Blocks)-1]

	var proven *eth.BlockID
	require.Eventually(func() bool {
		status = sil.SequencerProvenHead(t)
		if status.Head == nil || uint64(status.Head.Number) < last.Number {
			return false
		}
		proven = &eth.BlockID{Number: uint64(status.Head.Number), Hash: status.Head.Hash}
		return true
	}, 120*time.Second, 2*time.Second,
		"the sequencing supernode's proven head did not follow the proof batch")

	require.Equal(last.Hash, proven.Hash,
		"the sequencer's proven head is not the hash the proof batch committed to")
	require.False(status.Head.Forced, "nothing here should be a forced block")
	require.NotNil(status.Head.Carrier, "a proven block must name the L1 block that carried its proof")
	require.Greater(uint64(status.TrackerCursor), uint64(0), "the proven-head walk has not moved")

	// (5) and the walk is at proof cadence, not block cadence — which is the price G4 D1 accepts out
	// loud. The private chain is ahead of what has been proven, on the very node that produced it.
	privateHead := sys.L2BCL.HeadBlockRef(safety.LocalUnsafe).Number
	require.GreaterOrEqual(privateHead, uint64(status.Head.Number),
		"the proven head is above the private head, which is not a thing a proof could establish")

	t.Logger().Info("sequencer posture up",
		"private_head", privateHead, "proven_head", status.Head.Number,
		"tracker_l1", status.TrackerCursor,
		"committed_window", status.CommittedSeqWindowSize,
		"pipeline_window", status.PipelineSeqWindowSize)
}

// TestSequencerPostureUnfreezesTheClusterFrontier is the hazard-3 gate, and it is the one the
// rotation was blocked on.
//
// The structure mirrors TestSilhouetteCrossChainPinsThenAdvances deliberately, so that the two can be
// read side by side: same order of events, same pin, same release. What is different is the closing
// assertion. There, the sequencer supernode CANNOT cross-safe the executing message and the test says
// so. Here it must, and by the same proofs the verifier used.
//
//	P (silhouette)   ── init message at block B ──┐        (batches withheld above B-1)
//	A (ordinary)     ────────────── exec message at block N, referencing P's log
//	BOTH supernodes  A cross-safe PINS below N, zero invalidations
//	L1               ── proof batch covering B ──┐
//	BOTH supernodes  A cross-safe ADVANCES past N, still zero invalidations
func TestSequencerPostureUnfreezesTheClusterFrontier(gt *testing.T) {
	t, sys := sequencerPostureSystem(gt)
	require := t.Require()
	sil := sys.Silhouette
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther) // on A, the ordinary chain
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)   // on P, the silhouette chain

	// Prove some of P's early history first, so both nodes' proof paths are demonstrably working
	// before anything is withheld. A pin from "nothing has ever been proven" is indistinguishable from
	// a broken system.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	warmup := sil.Submitter.SubmitNext()
	require.NotNil(warmup, "no warm-up batch to post")
	warmupHead := warmup.Blocks[len(warmup.Blocks)-1].Number
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.CrossSafe, warmupHead, 120))

	// (1) the initiating message, on P — the private chain, sequenced on its real execution client.
	eventLogger := bob.DeployEventLogger()
	initMsg := bob.SendInitMessage(interop.RandomInitTrigger(rng, eventLogger, 2, 8))
	initBlock := initMsg.BlockID()
	t.Logger().Info("initiating message on the silhouette chain", "block", initBlock.Number)

	// (2) prove P up to just BELOW it, and stop.
	sil.Submitter.SubmitUpTo(initBlock.Number - 1)
	require.Equal(initBlock.Number-1, sil.Submitter.BatchedHead(),
		"the withheld batch boundary is not where the test needs it")
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.LocalSafe, initBlock.Number-1, 120))

	// (3) the executing message, on A, after the boundary is in place.
	execMsg := alice.SendExecMessage(initMsg)
	execBlock := execMsg.BlockID()
	t.Logger().Info("executing message on the ordinary chain", "block", execBlock.Number)

	invalidationsBefore := sil.InteropInvalidations(t)

	// (4a) THE PIN, on BOTH supernodes. A's block is on L1 and locally safe on each; neither may
	// cross-safe it, because the message it executes is not proven yet. Asserting it on the SEQUENCER
	// is the new part: that node has P's receipts for the initiating message sitting in the execution
	// client it drives, and it must still wait (G4 D1).
	dsl.CheckAll(t,
		sil.PeerCL.ReachedFn(safety.LocalSafe, execBlock.Number, 240),
		sys.L2ACL.ReachedFn(safety.LocalSafe, execBlock.Number, 240),
	)
	pinnedOnVerifier := sil.PeerEL.BlockRefByNumber(execBlock.Number).Hash
	pinnedOnSequencer := sys.L2ELA.BlockRefByNumber(execBlock.Number).Hash
	require.Equal(execBlock.Hash, pinnedOnVerifier)
	require.Equal(execBlock.Hash, pinnedOnSequencer,
		"the two nodes derived different blocks at the executing message's height")

	dsl.CheckAll(t,
		staysBelowFn(t, sil.PeerCL, safety.CrossSafe, execBlock.Number, 15),
		staysBelowFn(t, sys.L2ACL, safety.CrossSafe, execBlock.Number, 15),
	)
	require.Equal(invalidationsBefore, sil.InteropInvalidations(t),
		"a block was invalidated while waiting for a proof; a late message is not a bad one")

	// (5) release the proof, and keep proving: the round decides a timestamp only when EVERY chain in
	// the dependency set has safe data covering it, which on a real deployment is the proof cadence.
	sil.Submitter.SubmitUpTo(initBlock.Number)
	sil.Submitter.Start(2 * time.Second)

	// (4b) THE ADVANCE, on BOTH. Same block, by hash. On the verifier this is the phase-1 result; on
	// the sequencer it is the frozen frontier moving, which is what the whole posture is for.
	dsl.CheckAll(t,
		sil.PeerCL.ReachedRefFn(safety.CrossSafe, execBlock, 300),
		sys.L2ACL.ReachedRefFn(safety.CrossSafe, execBlock, 300),
	)

	// And the sequencer's own account of why it could: its proven head covers the initiating message.
	status := sil.SequencerProvenHead(t)
	require.NotNil(status.Head)
	require.GreaterOrEqual(uint64(status.Head.Number), initBlock.Number,
		"A's frontier passed the executing message on the sequencer without P being proven that far")

	// The structural half of "zero invalidations": an invalidation REPLACES the block at that height
	// with a deposits-only one, so the executing message's block still having its original hash on
	// BOTH nodes is the same claim, read off the chains instead of off a counter.
	require.Equal(pinnedOnVerifier, sil.PeerEL.BlockRefByNumber(execBlock.Number).Hash,
		"the verifier replaced the block at the executing message's height")
	require.Equal(pinnedOnSequencer, sys.L2ELA.BlockRefByNumber(execBlock.Number).Hash,
		"the sequencer replaced the block at the executing message's height")

	require.Equal(invalidationsBefore, sil.InteropInvalidations(t),
		"a block was invalidated on the way to cross-safing it")
	require.Positive(sil.TimestampsVerified(t),
		"no timestamps were verified at all, so the zero invalidation count says nothing")

	// P's blocks are still P's blocks. The empty-batch bomb would have rewritten the private chain
	// with deposits-only blocks by now on a long enough run; the initiating message's block still
	// being the one the sequencer produced is that failure's absence, read off the chain.
	require.Equal(initBlock.Hash, sys.L2ELB.BlockRefByNumber(initBlock.Number).Hash,
		"the initiating message's block on P was replaced; the sequencer reorged its own real chain")

	t.Logger().Info("cluster frontier advanced on both supernodes",
		"exec_block", execBlock.Number,
		"verifier_cross_safe", sil.PeerCL.HeadBlockRef(safety.CrossSafe).Number,
		"sequencer_cross_safe", sys.L2ACL.HeadBlockRef(safety.CrossSafe).Number,
		"proven_p_head", status.Head.Number,
		"invalidations", sil.InteropInvalidations(t))
}
