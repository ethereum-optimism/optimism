package silhouette

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop/raftwallogdb"
)

// THE POSTURE BRANCH.
//
// Assemble does two different things depending on `labels`, and these tests are the boundary between
// them. The organising claim is asymmetric on purpose: the VERIFIER posture is what every public node
// runs and must be untouched by the existence of the other one, while the SEQUENCER posture is
// defined by the seams it declines to replace.

// assembled is one Assemble call and everything a test needs to interrogate it.
type assembled struct {
	*testEnv
	a         *Assembly
	vncfg     *opnodecfg.Config
	overrides *rollupNode.InitializationOverrides
}

// assemble runs Assemble against a virtual-node config that has every pin's OPPOSITE value set, so
// that "the posture left this alone" and "the posture happens to agree with the default" cannot be
// confused. A sequencer-enabled, P2P-enabled, lag-limited config is what a real cluster hands in.
func assemble(t *testing.T, l1Head uint64, labels LabelSource) *assembled {
	t.Helper()
	env := newTestEnv(t, l1Head)
	vncfg := &opnodecfg.Config{Rollup: *silhouetteRollupConfig()}
	vncfg.Driver.SequencerEnabled = true
	vncfg.Driver.SequencerStopped = false
	vncfg.Driver.SequencerMaxSafeLag = 100
	vncfg.P2P = &p2p.Config{}
	vncfg.P2PSigner = &p2p.PreparedSigner{}
	overrides := &rollupNode.InitializationOverrides{}

	a, err := Assemble(testlog.Logger(t, log.LevelError), env.cfg, AssemblyConfig{
		Rollup: vncfg, L1Chain: sepoliaChainConfig(), SysCfg: vncfg.Rollup.Genesis.SystemConfig,
		L1: env.l1, Blobs: env.blobs, L1Headers: env.l1, Labels: labels,
		TrackerInterval: time.Millisecond,
	}, vncfg, overrides)
	require.NoError(t, err)
	t.Cleanup(a.Close)
	return &assembled{testEnv: env, a: a, vncfg: vncfg, overrides: overrides}
}

// TestVerifierPostureIsUntouchedByTheSequencerPosture is the regression fence around the branch.
//
// Every field the sequencer posture writes is asserted here to be UNWRITTEN, and the reason for
// asserting the absence of a change rather than trusting the branch is that all three of those
// fields are ones a verifier would tolerate quietly. An inflated window on a verifier does not
// error — it silently disables the forced-extension liveness backstop that DR-2's whole finite-window
// argument rests on, and the symptom would be a chain that stopped advancing during a prover outage
// instead of being carried by forced blocks. That is a failure nobody would trace back to here.
func TestVerifierPostureIsUntouchedByTheSequencerPosture(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+10, LabelsFromDerivation)

	require.Equal(t, LabelsFromDerivation, e.a.Labels)
	require.Equal(t, seqWindow, e.vncfg.Rollup.SeqWindowSize,
		"the verifier posture must keep the COMMITTED window: it is what makes the forced extension fire")
	require.Equal(t, uint64(100), e.vncfg.Driver.SequencerMaxSafeLag,
		"a verifier's safe head really does advance, so its lag limit is the operator's business")
	require.False(t, e.vncfg.Sync.SkipSyncStartCheck,
		"a verifier's sync-start check is bounded by its own window and must keep running")

	// And the verifier posture still is what it was: shim, override, no tracker.
	require.NotNil(t, e.a.Shim)
	require.Nil(t, e.a.Tracker, "a node that derives P needs nothing to drive the source for it")
	require.Same(t, e.a.Source, e.overrides.DataSourceOverride)
	require.False(t, e.vncfg.Driver.SequencerEnabled)
}

// TestSequencerPostureKeepsTheRealNode is the sequencer posture stated as the seams it does NOT
// replace, which is the whole of its design.
func TestSequencerPostureKeepsTheRealNode(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+10, LabelsFromProvenHead)

	require.Equal(t, LabelsFromProvenHead, e.a.Labels)

	// The EXECUTION CLIENT is P's real one. Nothing in the config was pointed at a shim, and there is
	// no shim: this node's answers about P's blocks come from an execution client that executed them.
	require.Nil(t, e.a.Shim, "the sequencer posture has no shim to replace an execution client it keeps")
	require.Nil(t, e.vncfg.L2, "the sequencer posture must leave the real engine endpoint alone")

	// The DERIVATION INPUT is untouched. Overriding it would point the pipeline at proof batches whose
	// rendered block hashes are deliberately not keccak(RLP(header)) (F-P1), so every consolidation
	// against the real execution client's real hashes would mismatch and reorg the real chain out.
	require.Nil(t, e.overrides.DataSourceOverride,
		"proof-derived attributes consolidated against real blocks reorg the real chain")

	// SEQUENCING SURVIVES. This node is P's block producer; the verifier posture's pin is the opposite
	// statement about a different node.
	require.True(t, e.vncfg.Driver.SequencerEnabled)
	require.False(t, e.vncfg.Driver.SequencerStopped)

	// And P2P survives, because the hazard the verifier posture closes is a gossiped payload halting
	// the SHIM (G3 D4) and there is no shim here. A private sequencer's gossip is its own deployment
	// question, not one this branch gets to decide.
	require.NotNil(t, e.vncfg.P2P)
	require.NotNil(t, e.vncfg.P2PSigner)

	// The tracker exists, and it is the production caller this type never had.
	require.NotNil(t, e.a.Tracker)
	require.Equal(t, l1GenesisNum, e.a.Tracker.Cursor())
	require.Same(t, e.a.Source, e.a.Tracker.src, "the tracker must drive the assembly's own source")
	require.Same(t, e.a.Facts, e.a.Source.facts)

	// One namespace, and NOT `eth`: the real execution client answers that, and a second answer at the
	// same route would be a node contradicting itself about its own chain.
	ns := map[string]bool{}
	for _, api := range e.overrides.ExtraAPIs {
		ns[api.Namespace] = true
	}
	require.True(t, ns["silhouette"], "the proven head must be readable somewhere")
	require.False(t, ns["eth"], "P's real execution client answers eth_*; the posture must not shadow it")
	require.Len(t, ns, 1)
}

// TestSequencerPostureDisarmsTheFrozenSafeHeadHazards asserts the three pins together, because they
// are one bug: on this node P's public safe head never moves, and three stock mechanisms read a
// stalled safe head as a fault and act on it.
func TestSequencerPostureDisarmsTheFrozenSafeHeadHazards(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+10, LabelsFromProvenHead)

	// (1) the empty-batch bomb: the pipeline must never conclude that an epoch it has no data for is
	// an empty epoch, because the real chain's blocks at those heights are not empty.
	require.Equal(t, SequencerPostureSeqWindow, e.vncfg.Rollup.SeqWindowSize)
	require.Greater(t, e.vncfg.Rollup.SeqWindowSize, seqWindow)

	// (2) the safe-lag stall: a limit that can never be satisfied parks the block producer.
	require.Zero(t, e.vncfg.Driver.SequencerMaxSafeLag,
		"a lag limit against a frozen safe head stops P's block production permanently")

	// (3) the sync-start walk-back: pin (1) removes the early stop, so the walk must be short-circuited
	// or every restart costs one L2 lookup per block of the whole chain.
	require.True(t, e.vncfg.Sync.SkipSyncStartCheck)
}

// TestSequencerPostureKeepsTheCommittedForcedExtension is the aliasing trap, and it is the sharpest
// test in this file.
//
// The disarm inflates the VIRTUAL NODE's sequencing window. The forced-extension convention is
// computed from a sequencing window too — `epoch.Number + SeqWindowSize` against the pipeline origin
// (forced.go) — and it is a PUBLIC rule: the guest, every verifier and this node must all compute the
// same forced blocks or they are describing different chains. If the proof-facing side read the
// virtual node's config instead of a copy, the disarm would silently make the sequencer the only
// node in the cluster that never computes a forced block, and the disagreement would surface as its
// proven head diverging from every verifier's.
//
// So: same L1, same anchor, both postures, and the forced extension each one computes must be
// identical. The control is that the extension is non-empty — otherwise the assertion would pass for
// two nodes that both computed nothing.
func TestSequencerPostureKeepsTheCommittedForcedExtension(t *testing.T) {
	t.Parallel()
	// Far enough above the anchor's origin that the COMMITTED window has expired several times over.
	const l1Head = l1GenesisNum + 200

	seq := assemble(t, l1Head, LabelsFromProvenHead)
	ver := assemble(t, l1Head, LabelsFromDerivation)

	require.Equal(t, seqWindow, seq.a.Source.rollup.SeqWindowSize,
		"the proof-facing side must hold the COMMITTED window, not the pipeline's")
	require.Equal(t, SequencerPostureSeqWindow, seq.vncfg.Rollup.SeqWindowSize,
		"...while the pipeline holds the inflated one, which is the point of the copy")

	anchor := Fact{
		Number:    seq.cfg.Anchor.BlockNumber,
		Hash:      seq.cfg.Anchor.BlockHash,
		Timestamp: seq.cfg.Anchor.Timestamp,
		L1Origin:  seq.cfg.Anchor.L1Origin,
	}
	origin := l1Head

	seqForced, err := ForcedExtension(context.Background(), seq.a.Source.forcedParams(), seq.l1, anchor, origin)
	require.NoError(t, err)
	verForced, err := ForcedExtension(context.Background(), ver.a.Source.forcedParams(), ver.l1, anchor, origin)
	require.NoError(t, err)

	require.NotEmpty(t, verForced,
		"the control failed: no forced extension is due, so equal results prove nothing")
	require.Equal(t, verForced, seqForced,
		"the sequencer computes a different public rendering of P than its verifiers do")
}

// TestAssembledTrackerAcceptsProofBatches is the gate the ProvenHeadTracker never had: its
// PRODUCTION caller, exercised.
//
// Everything here goes through Assemble rather than through a hand-built tracker. That is the whole
// point — the tracker was correct and unreachable, and a test that constructs one itself re-proves
// the part that was never broken.
func TestAssembledTrackerAcceptsProofBatches(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+40, LabelsFromProvenHead)

	store, err := raftwallogdb.Open(t.TempDir(), eth.ChainIDFromUInt64(424250))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, e.a.AttachLogStore(testlog.Logger(t, log.LevelError), store))

	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1,
		firstTime: e.rollup.Genesis.L2Time + l2BlockTime,
		count:     4, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	_, hadHead := e.a.Facts.Head()
	require.False(t, hadHead, "nothing is proven before the walk runs")

	// Run() is what the supernode calls. Let it walk to the head of the fake L1 and stop.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); e.a.Run(ctx) }()

	require.Eventually(t, func() bool {
		head, ok := e.a.Facts.Head()
		return ok && head.Number == 4
	}, 20*time.Second, 10*time.Millisecond, "the assembled tracker did not accept the planted batch")
	cancel()
	<-done

	head, _ := e.a.Facts.Head()
	last := batch.Blocks[len(batch.Blocks)-1]
	require.Equal(t, last.Hash, head.Hash, "the proven head is not the hash the batch committed to")
	require.False(t, head.Forced, "a proven block must not be reported as forced")

	// And the label surface the cluster consumes now answers, which is the whole reason the walk
	// exists: without it this node freezes chain A's cross-safe frontier from a healthy P.
	container := NewContainer(testlog.Logger(t, log.LevelError),
		&frozenInner{id: eth.ChainIDFromUInt64(424250), rollup: e.rollup}, e.a.Facts, LabelsFromProvenHead)
	ref, err := container.LocalSafeBlockAtTimestamp(context.Background(), head.Timestamp)
	require.NoError(t, err)
	require.Equal(t, head.Hash, ref.Hash)

	// The public surface reports it, including the two windows — the disarm made visible at runtime
	// rather than left as a constant in a Go file.
	api := &ProvenHeadAPI{facts: e.a.Facts, tracker: e.a.Tracker,
		committedSeqWindow: e.a.Source.rollup.SeqWindowSize, pipelineSeqWindow: e.vncfg.Rollup.SeqWindowSize}
	status := api.ProvenHead()
	require.NotNil(t, status.Head)
	require.Equal(t, head.Hash, status.Head.Hash)
	require.NotNil(t, status.Head.Carrier, "a proven block names the L1 block that carried its proof")
	require.EqualValues(t, seqWindow, status.CommittedSeqWindowSize)
	require.EqualValues(t, SequencerPostureSeqWindow, status.PipelineSeqWindowSize)
	require.Greater(t, uint64(status.TrackerCursor), l1GenesisNum, "the walk must have moved")
}

// TestSequencerLabelsAnswerBetweenBlockTimestamps is the regression fence around the bug that the
// end-to-end run found and no unit test could have.
//
// The cross-safety round asks about every timestamp, ONE SECOND AT A TIME, and it cannot skip one. A
// chain with a two-second block time therefore gets asked about timestamps no block lands on, for half
// of them. `TimestampToBlockNumber` rounds DOWN, so the honest answer is the block at or before — and
// the container used to compare the resolved fact's timestamp against the timestamp that was ASKED
// about, which rejected every one of those. The first between-blocks timestamp the round met froze the
// whole dependency set's frontier, permanently: the exact failure this posture exists to prevent,
// arriving from inside the fix for it.
//
// A unit test naturally asks about a block's own timestamp, which is why this went unseen. So this one
// asks about the odd second in between, on purpose, and asserts the answer is the block BEFORE it.
func TestSequencerLabelsAnswerBetweenBlockTimestamps(t *testing.T) {
	t.Parallel()
	e := assemble(t, l1GenesisNum+40, LabelsFromProvenHead)

	store, err := raftwallogdb.Open(t.TempDir(), eth.ChainIDFromUInt64(424250))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, e.a.AttachLogStore(testlog.Logger(t, log.LevelError), store))

	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1,
		firstTime: e.rollup.Genesis.L2Time + l2BlockTime,
		count:     4, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	e.plant(e.buildBatch(spec), spec)
	ctx := context.Background()
	for i := 0; i < 64; i++ {
		advanced, err := e.a.Tracker.Step(ctx)
		require.NoError(t, err)
		if !advanced {
			break
		}
	}
	container := NewContainer(testlog.Logger(t, log.LevelError),
		&frozenInner{id: eth.ChainIDFromUInt64(424250), rollup: e.rollup}, e.a.Facts, LabelsFromProvenHead)

	block2 := e.rollup.Genesis.L2Time + 2*l2BlockTime // a real block's own timestamp
	onBlock, err := container.LocalSafeBlockAtTimestamp(ctx, block2)
	require.NoError(t, err, "the container cannot answer for a timestamp a block lands on")
	require.EqualValues(t, 2, onBlock.Number)

	// The odd second in between. l2BlockTime is 2, so this lands on no block at all.
	between := block2 + 1
	betweenRef, err := container.LocalSafeBlockAtTimestamp(ctx, between)
	require.NoError(t, err,
		"the container refused a timestamp between two blocks; the round cannot skip one, so this "+
			"freezes the whole dependency set's frontier")
	require.Equal(t, onBlock.Hash, betweenRef.Hash,
		"a timestamp between blocks must resolve to the block BEFORE it, not to a later one")

	// And the same through the atomic surface the readiness check actually calls.
	l2, l1, err := container.OptimisticAt(ctx, between)
	require.NoError(t, err)
	require.EqualValues(t, 2, l2.Number)
	require.NotEqual(t, eth.BlockID{}, l1, "a proven block must name the L1 block that carried it")

	// The anti-fabrication check is still aimed at something: a fact whose timestamp disagrees with the
	// rollup config's own arithmetic is refused rather than served.
	// Block 5 is the next block above the planted batch, so recording it keeps the fact table
	// contiguous (which FactStore.Record requires) while giving the check something to catch: its
	// timestamp is deliberately one second off what the config's arithmetic says block 5's is.
	const badNum = 5
	badTS := e.rollup.Genesis.L2Time + badNum*l2BlockTime
	e.a.Facts.Record(Fact{Number: badNum, Hash: common.HexToHash("0xbad"), Timestamp: badTS + 1})
	_, err = container.LocalSafeBlockAtTimestamp(ctx, badTS)
	require.Error(t, err, "a fact whose timestamp contradicts the config's arithmetic must be refused")
}

// TestTrackerStartBlockIsBoundedByTheAnchor covers the field that was dead until the tracker had a
// caller. The refusal is the interesting case: a start block above the anchor's origin skips the L1
// blocks that can carry the batch extending the anchor, and the symptom is not "catch-up is faster",
// it is "every batch is rejected for not chaining" — a total proof outage from a number that looked
// like a tuning knob.
func TestTrackerStartBlockIsBoundedByTheAnchor(t *testing.T) {
	t.Parallel()
	committed := silhouetteRollupConfig()

	t.Run("zero means the anchor's origin", func(t *testing.T) {
		cfg := &Config{Anchor: Anchor{L1Origin: eth.BlockID{Number: l1GenesisNum + 50}}}
		got, err := trackerStartBlock(cfg, committed)
		require.NoError(t, err)
		require.Equal(t, l1GenesisNum, got, "the rollup genesis is lower, so it is the floor")
	})
	t.Run("below the floor is honoured", func(t *testing.T) {
		cfg := &Config{L1StartBlock: l1GenesisNum - 10,
			Anchor: Anchor{L1Origin: eth.BlockID{Number: l1GenesisNum + 50}}}
		got, err := trackerStartBlock(cfg, committed)
		require.NoError(t, err)
		require.Equal(t, l1GenesisNum-10, got)
	})
	t.Run("above the floor is refused, not clamped", func(t *testing.T) {
		cfg := &Config{L1StartBlock: l1GenesisNum + 1,
			Anchor: Anchor{L1Origin: eth.BlockID{Number: l1GenesisNum + 50}}}
		_, err := trackerStartBlock(cfg, committed)
		require.ErrorContains(t, err, "is above the anchor's L1 origin")
		require.ErrorContains(t, err, "reject every later batch for not chaining")
	})
}
