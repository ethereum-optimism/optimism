package silhouette

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop/raftwallogdb"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// frozenInner is P's chain container in a SEQUENCER supernode, as it really behaves: it fronts P's
// real execution client, which is producing blocks normally, but there is no public derivation
// behind it — no batcher, no DA, nothing that moves a public local-safe label forward. So every
// label surface says "not found", forever.
//
// It embeds the interface rather than implementing all of it, so any method these tests do not
// expect to be called panics instead of quietly returning a zero value.
type frozenInner struct {
	cc.InteropChain
	id     eth.ChainID
	rollup *rollup.Config
	// optimisticCalls counts delegation, so the verifier-posture test can prove it delegated.
	optimisticCalls int
}

func (f *frozenInner) ID() eth.ChainID   { return f.id }
func (f *frozenInner) BlockTime() uint64 { return f.rollup.BlockTime }

func (f *frozenInner) TimestampToBlockNumber(_ context.Context, ts uint64) (uint64, error) {
	return f.rollup.TargetBlockNumber(ts)
}

// BlockNumberToTimestamp is the inverse, from the SAME rollup config. Both halves are real arithmetic
// rather than stubs because the container's anti-fabrication check compares one against the other: a
// stub would make that check pass by construction and test nothing.
func (f *frozenInner) BlockNumberToTimestamp(_ context.Context, num uint64) (uint64, error) {
	if num < f.rollup.Genesis.L2.Number {
		return 0, fmt.Errorf("block %d is below genesis %d", num, f.rollup.Genesis.L2.Number)
	}
	return f.rollup.TimestampForBlock(num), nil
}

func (f *frozenInner) OptimisticAt(_ context.Context, _ uint64) (eth.BlockID, eth.BlockID, error) {
	f.optimisticCalls++
	return eth.BlockID{}, eth.BlockID{}, ethereum.NotFound
}

func (f *frozenInner) LocalSafeBlockAtTimestamp(_ context.Context, _ uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, ethereum.NotFound
}

func (f *frozenInner) FirstSafeHeadTimestamp(_ context.Context) (uint64, error) {
	return 0, cc.ErrSafeDBNotReady
}

// sequencerEnv is a sequencer-posture assembly: a frozen real chain container, a fact store kept
// current by the proven-head tracker walking L1, and the silhouette container over both.
type sequencerEnv struct {
	*testEnv
	inner     *frozenInner
	container *Container
	tracker   *ProvenHeadTracker
	store     *raftwallogdb.DB
}

func newSequencerEnv(t *testing.T, l1Head uint64) *sequencerEnv {
	t.Helper()
	env := newTestEnv(t, l1Head)

	store, err := raftwallogdb.Open(t.TempDir(), eth.ChainIDFromUInt64(424247))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, env.src.AttachLogSink(NewLogSink(testlog.Logger(t, log.LevelInfo), store)))

	inner := &frozenInner{id: eth.ChainIDFromUInt64(424247), rollup: env.rollup}
	return &sequencerEnv{
		testEnv:   env,
		inner:     inner,
		container: NewContainer(testlog.Logger(t, log.LevelInfo), inner, env.facts, LabelsFromProvenHead),
		tracker:   NewProvenHeadTracker(testlog.Logger(t, log.LevelInfo), env.src, env.l1, l1GenesisNum, 0),
		store:     store,
	}
}

// catchUp walks the tracker to the head of the fake L1.
func (e *sequencerEnv) catchUp(t *testing.T) {
	t.Helper()
	for i := 0; i < 512; i++ {
		advanced, err := e.tracker.Step(context.Background())
		require.NoError(t, err)
		if !advanced {
			return
		}
	}
	t.Fatal("tracker did not catch up with L1")
}

// TestSequencerLabelSourceFollowsTheProvenHead is hazard-3 symmetry, both halves.
//
// CONTROL: without the label source, P's public label surface never answers, because nothing on
// this node derives P. That is not a P problem — it is a CLUSTER problem, because the cross-safety
// round gates on every chain answering, so a frozen P freezes chain A's cross-safe too.
//
// TREATMENT: with the label source, P's local-safe follows the proven head — where P's proofs have
// actually reached — and the round can proceed.
func TestSequencerLabelSourceFollowsTheProvenHead(t *testing.T) {
	t.Parallel()
	e := newSequencerEnv(t, l1GenesisNum+40)

	firstTime := e.rollup.Genesis.L2Time + l2BlockTime
	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1, firstTime: firstTime,
		count: 4, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	provenBlock := batch.Blocks[2]

	// CONTROL: before any proof has been read, and with the inner container frozen, there is no
	// label to report. The honest answer is not-found, which the readiness check turns into a wait.
	_, _, err := e.container.OptimisticAt(context.Background(), provenBlock.Timestamp)
	require.ErrorIs(t, err, ethereum.NotFound,
		"with no proof read yet the label surface must not invent a head")
	_, err = e.container.FirstSafeHeadTimestamp(context.Background())
	require.ErrorIs(t, err, cc.ErrSafeDBNotReady)

	// The inner container is frozen in exactly the way a sequencer-side P is: it would answer
	// not-found forever, however long we waited.
	_, _, innerErr := e.inner.OptimisticAt(context.Background(), provenBlock.Timestamp)
	require.ErrorIs(t, innerErr, ethereum.NotFound,
		"the control: P's real container never advances a public label on its own")

	// TREATMENT: the tracker reads the proof batch off L1.
	e.catchUp(t)

	l2, l1, err := e.container.OptimisticAt(context.Background(), provenBlock.Timestamp)
	require.NoError(t, err, "once the proof lands, the proven head IS the public label")
	require.Equal(t, provenBlock.Number, l2.Number)
	require.Equal(t, provenBlock.Hash, l2.Hash, "the label names the block's REAL wire hash")
	require.Equal(t, spec.carrier, l1.Number,
		"the L1 inclusion is the L1 block that carried the proof")

	safe, err := e.container.LocalSafeBlockAtTimestamp(context.Background(), provenBlock.Timestamp)
	require.NoError(t, err)
	require.Equal(t, provenBlock.Hash, safe.Hash)
	require.Equal(t, provenBlock.Timestamp, safe.Time)

	first, err := e.container.FirstSafeHeadTimestamp(context.Background())
	require.NoError(t, err)
	require.Equal(t, firstTime, first, "the earliest verifiable timestamp is the oldest fact's")

	// Above the proven head the answer is still not-found rather than extrapolated arithmetic: a
	// label for a block no proof covers is exactly the fabrication the design forbids.
	beyond := batch.Blocks[len(batch.Blocks)-1].Timestamp + 10*l2BlockTime
	_, _, err = e.container.OptimisticAt(context.Background(), beyond)
	require.ErrorIs(t, err, ethereum.NotFound,
		"the label must never outrun the proofs")
}

// TestSequencerTrackerSealsExportedMessages: the sequencer posture must ingest P's messages from
// the WIRE too, not from the receipts its execution client happens to have. Both postures read the
// same wire, so both see the same chain.
func TestSequencerTrackerSealsExportedMessages(t *testing.T) {
	t.Parallel()
	e := newSequencerEnv(t, l1GenesisNum+40)

	require.Equal(t, cc.IngestionProven, cc.IngestionSourceOf(e.container),
		"a sequencer-side silhouette chain still ingests from the wire")

	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1,
		firstTime: e.rollup.Genesis.L2Time + l2BlockTime,
		count:     3, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	batch := e.buildBatch(spec)
	e.plant(batch, spec)
	e.catchUp(t)

	latest, ok := e.store.LatestSealedBlock()
	require.True(t, ok, "the tracker's acceptance path must seal into the log store")
	last := batch.Blocks[len(batch.Blocks)-1]
	require.Equal(t, last.Number, latest.Number)
	require.Equal(t, last.Hash, latest.Hash, "sealed under the real wire hash")

	// And the receipts path is refused, as it must be even here.
	_, _, err := e.container.FetchReceipts(context.Background(), eth.BlockID{Number: last.Number, Hash: last.Hash})
	require.ErrorIs(t, err, ErrNoExecutionData)
}

// TestContainerRefusesInvalidation is the E3 honesty line in code: no path may reach invalidation
// or replacement-block synthesis for a proof-carried chain. This is the backstop that turns "no
// path reaches it" from an argument into an assertion.
func TestContainerRefusesInvalidation(t *testing.T) {
	t.Parallel()
	e := newSequencerEnv(t, l1GenesisNum+20)

	rewound, err := e.container.InvalidateBlock(context.Background(), 7,
		common.HexToHash("0xabc"), 1234, eth.Bytes32{}, eth.Bytes32{}, nil)
	require.ErrorIs(t, err, ErrNoExecutionData)
	require.False(t, rewound, "nothing may be rewound on a chain whose blocks are valid by proof")
}

// TestVerifierPostureDelegatesLabels: in a verifier supernode this node really does derive P — a
// stock op-node over the injected source and the shim — so its own labels are correct and the
// container must not substitute the fact store for them. Getting this backwards would silently
// bypass the derivation the whole design exists to keep stock.
func TestVerifierPostureDelegatesLabels(t *testing.T) {
	t.Parallel()
	e := newSequencerEnv(t, l1GenesisNum+40)
	verifier := NewContainer(testlog.Logger(t, log.LevelInfo), e.inner, e.facts, LabelsFromDerivation)

	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1,
		firstTime: e.rollup.Genesis.L2Time + l2BlockTime,
		count:     3, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	batch := e.buildBatch(spec)
	e.plant(batch, spec)
	e.catchUp(t)

	before := e.inner.optimisticCalls
	_, _, err := verifier.OptimisticAt(context.Background(), batch.Blocks[1].Timestamp)
	require.ErrorIs(t, err, ethereum.NotFound, "the verifier posture reports what the DERIVATION says")
	require.Equal(t, before+1, e.inner.optimisticCalls,
		"the verifier posture must delegate to the embedded container, facts notwithstanding")

	// The same store, read through the sequencer posture, does have an answer — so the difference
	// under test is the posture and nothing else.
	seq := NewContainer(testlog.Logger(t, log.LevelInfo), e.inner, e.facts, LabelsFromProvenHead)
	_, _, err = seq.OptimisticAt(context.Background(), batch.Blocks[1].Timestamp)
	require.NoError(t, err)
}

// TestForcedBlockInheritsNearestProvenCarrier: a forced block has no carrier, because nothing
// proved it. Refusing to label it would reintroduce the freeze that forced blocks exist to
// prevent, so it inherits the nearest proven ancestor's carrier — a deliberate understatement,
// since this value is consumed as a lower bound.
func TestForcedBlockInheritsNearestProvenCarrier(t *testing.T) {
	t.Parallel()
	e := newSequencerEnv(t, l1GenesisNum+40)

	spec := batchSpec{
		prevRoot: e.cfg.Anchor.OutputRoot, firstBlock: 1,
		firstTime: e.rollup.Genesis.L2Time + l2BlockTime,
		count:     2, l1Head: l1GenesisNum + 5, carrier: l1GenesisNum + 10,
	}
	batch := e.buildBatch(spec)
	e.plant(batch, spec)
	e.catchUp(t)

	provenHead, ok := e.facts.Head()
	require.True(t, ok)
	require.False(t, provenHead.Forced)
	carrier, ok := e.facts.CarrierOf(provenHead.Number)
	require.True(t, ok)

	// Append a forced block by hand, exactly as the convention would produce one on a stall.
	forced := Fact{
		Number:                   provenHead.Number + 1,
		Timestamp:                provenHead.Timestamp + l2BlockTime,
		Hash:                     common.HexToHash("0xf0rced"),
		StateRoot:                provenHead.StateRoot,
		MessagePasserStorageRoot: provenHead.MessagePasserStorageRoot,
		L1Origin:                 provenHead.L1Origin,
		SeqNumber:                provenHead.SeqNumber + 1,
		Forced:                   true,
	}
	e.facts.Record(forced)

	_, ok = e.facts.CarrierOf(forced.Number)
	require.False(t, ok, "a forced block has no carrier of its own: nothing proved it")

	l2, l1, err := e.container.OptimisticAt(context.Background(), forced.Timestamp)
	require.NoError(t, err, "a forced block must still carry a label, or the stall freezes the cluster")
	require.Equal(t, forced.Hash, l2.Hash)
	require.Equal(t, carrier, l1,
		"it inherits the nearest proven ancestor's L1 inclusion — an understatement, which is the safe direction")
}
