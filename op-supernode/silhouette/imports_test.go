package silhouette

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// THE IMPORT LIST, from the wire to the judge's door.
//
// The other half of G7 lives in op-supernode/supernode/activity/interop, where the judge consumes
// this. What is under test here is the carriage: an accepted batch's import list has to arrive in the
// fact store, decoded and in order, and — the part that is easy to get subtly wrong — the DIFFERENCE
// between "this block imported nothing" and "this wire version does not say" has to survive the trip.

// anImport is one plausible executing message: a message on some peer chain, stamped below the
// consuming block.
func anImport(chain uint64, blockNum uint64, logIdx uint32, ts uint64) proofbatch.ExecMsg {
	return proofbatch.ExecMsg{Message: messages.Message{
		Identifier: messages.Identifier{
			Origin:      common.HexToAddress("0x00000000000000000000000000000000000a11ce"),
			BlockNumber: blockNum,
			LogIndex:    logIdx,
			Timestamp:   ts,
			ChainID:     eth.ChainIDFromUInt64(chain),
		},
		PayloadHash: common.HexToHash("0xc0ffee"),
	}}
}

// TestAcceptedBatchCarriesItsImportList is the carriage gate: what the wire declared is what the
// container hands the judge, decoded into the judge's own type.
func TestAcceptedBatchCarriesItsImportList(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+10)
	spec := e.goodBatch()
	// Two imports, in canonical order, on the batch's first block. The consuming block's timestamp is
	// l1GenesisT + l2BlockTime, so both are stamped below it.
	first := anImport(424246, 5, 0, l1GenesisT)
	second := anImport(424246, 5, 3, l1GenesisT)
	spec.imports = []proofbatch.ExecMsg{first, second}
	e.plantSpec(spec)
	require.NotEmpty(t, e.derive(spec.carrier), "the batch must be accepted")

	fact, ok := e.facts.ByNumber(1)
	require.True(t, ok)
	require.True(t, fact.ExecMsgsKnown, "a v3 batch's import list is a claim, not an unknown")
	require.Len(t, fact.ExecMsgs, 2)
	require.Equal(t, eth.ChainIDFromUInt64(424246), fact.ExecMsgs[0].ChainID)
	require.Equal(t, uint64(5), fact.ExecMsgs[0].BlockNum)
	require.Equal(t, uint32(0), fact.ExecMsgs[0].LogIdx)
	require.Equal(t, uint32(3), fact.ExecMsgs[1].LogIdx)
	// The checksum is DERIVED at acceptance, not carried: same value the frontier view would compute
	// from the real log.
	require.Equal(t, first.Checksum(), fact.ExecMsgs[0].Checksum)
	require.Equal(t, second.Checksum(), fact.ExecMsgs[1].Checksum)

	// And a block of the same batch that imported nothing says so, rather than saying nothing.
	quiet, ok := e.facts.ByNumber(2)
	require.True(t, ok)
	require.True(t, quiet.ExecMsgsKnown)
	require.Empty(t, quiet.ExecMsgs)

	// Through the capability, which is the surface the judge actually uses.
	container := NewContainer(testlog.Logger(t, log.LevelError),
		&frozenInner{id: eth.ChainIDFromUInt64(424247), rollup: e.rollup}, e.facts, LabelsFromDerivation)
	msgs, onWire, err := cc.ProvenExecMsgsOf(container, 1)
	require.NoError(t, err)
	require.True(t, onWire)
	require.Len(t, msgs, 2)
	require.Equal(t, first.Checksum(), msgs[0].Checksum, "ordinal 0 is the first of the canonical set")
	require.Equal(t, second.Checksum(), msgs[1].Checksum)

	msgs, onWire, err = cc.ProvenExecMsgsOf(container, 2)
	require.NoError(t, err)
	require.True(t, onWire, "an empty import list is still a claim")
	require.Empty(t, msgs)
}

// TestSameTimestampImportIsRejectedAtAcceptance is where G7G D2 becomes an operational fact rather
// than a codec rule: a batch declaring a same-timestamp import is REFUSED, so the proven head does
// not move and there is nothing for the judge to be unable to order.
//
// That failure mode is the one this design wants. The alternative — accepting it and letting the
// judge find it invalid — would put proven history into the state that has no local remedy (G7G D3).
// Refusing the batch instead means the chain simply does not advance, which is the same shape as any
// other bad batch and is recoverable by the prover posting a correct one.
func TestSameTimestampImportIsRejectedAtAcceptance(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+10)
	spec := e.goodBatch()
	consumingTime := spec.firstTime
	spec.imports = []proofbatch.ExecMsg{anImport(424246, 5, 0, consumingTime)}
	e.plantSpec(spec)

	require.Empty(t, e.derive(spec.carrier), "a batch importing a same-timestamp message must be refused")
	_, ok := e.facts.ByNumber(1)
	require.False(t, ok, "the proven head must not move on a refused batch")

	// The control: one second earlier and the same batch is accepted, so the refusal is the rule and
	// not the fixture.
	e2 := newTestEnv(t, l1GenesisNum+10)
	ok2 := e2.goodBatch()
	ok2.imports = []proofbatch.ExecMsg{anImport(424246, 5, 0, consumingTime-1)}
	e2.plantSpec(ok2)
	require.NotEmpty(t, e2.derive(ok2.carrier))
	fact, found := e2.facts.ByNumber(1)
	require.True(t, found)
	require.Len(t, fact.ExecMsgs, 1)
}

// TestPreV3WireLeavesDependenciesUnknown is the two-version story at the fact layer, and it is the
// fail-open guard.
//
// A v2 batch says nothing about imports. The temptation is to record that as an empty list, because
// the field is simply not there — and that is precisely the bug: an empty list is the CLAIM "this
// block consumed nothing", which the judge would happily validate, producing a verifier that reports
// verified dependencies and verifies none.
func TestPreV3WireLeavesDependenciesUnknown(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+10)
	e.cfg.WireVersion = proofbatch.VersionV2
	require.NoError(t, e.cfg.Check())
	require.False(t, e.cfg.DependenciesVerified(),
		"a v2-configured verifier must not claim it checks dependencies")

	spec := e.goodBatch()
	spec.wireVersion = proofbatch.VersionV2
	e.plantSpec(spec)
	require.NotEmpty(t, e.derive(spec.carrier), "a v2 verifier must accept a v2 batch")

	fact, ok := e.facts.ByNumber(1)
	require.True(t, ok)
	require.False(t, fact.ExecMsgsKnown, "a v2 batch's imports are UNKNOWN, not empty")
	require.Empty(t, fact.ExecMsgs)

	container := NewContainer(testlog.Logger(t, log.LevelError),
		&frozenInner{id: eth.ChainIDFromUInt64(424247), rollup: e.rollup}, e.facts, LabelsFromDerivation)
	msgs, onWire, err := cc.ProvenExecMsgsOf(container, 1)
	require.NoError(t, err)
	require.False(t, onWire, "the capability must report that this wire carries no import list")
	require.Nil(t, msgs)
}

// TestVerifierAcceptsExactlyItsConfiguredWireVersion is the rotation's config gate, in both
// directions. A verifier's vkey pins its guest and its guest pins the wire version, so a node that
// read the other version would be applying a dependency posture its operator did not choose.
func TestVerifierAcceptsExactlyItsConfiguredWireVersion(t *testing.T) {
	t.Run("a v3 node refuses a v2 batch", func(t *testing.T) {
		e := newTestEnv(t, l1GenesisNum+10)
		require.True(t, e.cfg.DependenciesVerified())
		spec := e.goodBatch()
		spec.wireVersion = proofbatch.VersionV2
		e.plantSpec(spec)
		require.Empty(t, e.derive(spec.carrier))
		_, ok := e.facts.ByNumber(1)
		require.False(t, ok)
	})

	t.Run("a v2 node refuses a v3 batch", func(t *testing.T) {
		e := newTestEnv(t, l1GenesisNum+10)
		e.cfg.WireVersion = proofbatch.VersionV2
		spec := e.goodBatch()
		spec.wireVersion = proofbatch.Version
		e.plantSpec(spec)
		require.Empty(t, e.derive(spec.carrier))
		_, ok := e.facts.ByNumber(1)
		require.False(t, ok)
	})

	// A version the codec cannot read is refused at CONFIG LOAD, not at the first blob: the
	// difference between a node that will not start and a node that starts, looks healthy and
	// rejects everything forever.
	t.Run("an unimplemented version is a config error", func(t *testing.T) {
		e := newTestEnv(t, l1GenesisNum+10)
		e.cfg.WireVersion = proofbatch.VersionV1
		require.ErrorContains(t, e.cfg.Check(), "wireVersion")
		e.cfg.WireVersion = 42
		require.ErrorContains(t, e.cfg.Check(), "wireVersion")
	})
}

// TestForcedBlockImportsNothingAndSaysSo is G7G D4.
//
// A forced block is not a wire object — both sides COMPUTE it from the convention — and it carries
// exactly one transaction, the L1-info transaction. So "it imported nothing" is knowledge, not
// absence, and it must be known at every wire version. Marking it unknown would make the forced
// extension, which exists so a dead prover cannot stall the dependency set, the one thing that
// stalls it: the judge would refuse to describe the block and the round would never decide.
func TestForcedBlockImportsNothingAndSaysSo(t *testing.T) {
	// The L1 head has to sit a full sequencing window above the block's origin, or the convention does
	// not define a forced block there at all and ForcedBlockAt correctly refuses (G3 D5).
	e := newTestEnv(t, l1GenesisNum+seqWindow+10)
	spec := e.goodBatch()
	e.plantSpec(spec)
	require.NotEmpty(t, e.derive(spec.carrier))
	head, ok := e.facts.Head()
	require.True(t, ok)

	forced, err := ForcedBlockAt(context.Background(), e.src.forcedParams(), e.l1, head,
		head.L1Origin, head.SeqNumber+1, head.Timestamp+l2BlockTime)
	require.NoError(t, err)
	require.True(t, forced.Forced)
	require.True(t, forced.ExecMsgsKnown,
		"a forced block's empty import list is computed, not read; an unknown one would stall the round")
	require.Empty(t, forced.ExecMsgs)
}

// TestBothPosturesCarryTheImportListIdentically is gate 4.
//
// The claim the sequencer posture rests on is that it replaces NO seam of the verifier posture: it
// adds a tracker that drives the SAME DataSource, so acceptance — and therefore the import list — is
// one implementation, not two. Under G7 that claim gains a new field to be wrong about, and the cost
// of being wrong is high in a specific way: the sequencer's own supernode is a member of the public
// network, so a sequencer that recorded a different import list than every verifier would be a node
// disagreeing with its own verifiers about what its chain consumed.
//
// So this asserts equality rather than presence, and it does it by running the two postures over the
// same planted L1.
func TestBothPosturesCarryTheImportListIdentically(t *testing.T) {
	imports := []proofbatch.ExecMsg{
		anImport(424246, 5, 0, l1GenesisT),
		anImport(424248, 9, 2, l1GenesisT),
	}

	// The verifier posture: the derivation pipeline pulls the source.
	verifier := newTestEnv(t, l1GenesisNum+10)
	vspec := verifier.goodBatch()
	vspec.imports = imports
	verifier.plantSpec(vspec)
	require.NotEmpty(t, verifier.derive(vspec.carrier))

	// The sequencer posture: the proven-head tracker drains the same source over the same L1.
	seq := newTestEnv(t, l1GenesisNum+10)
	sspec := seq.goodBatch()
	sspec.imports = imports
	seq.plantSpec(sspec)
	tracker := NewProvenHeadTracker(testlog.Logger(t, log.LevelError), seq.src, seq.l1, l1GenesisNum, time.Millisecond)
	for range 20 {
		if _, err := tracker.Step(context.Background()); err != nil {
			require.NoError(t, err)
		}
		if _, ok := seq.facts.ByNumber(1); ok {
			break
		}
	}

	vFact, ok := verifier.facts.ByNumber(1)
	require.True(t, ok, "the verifier posture did not accept the batch")
	sFact, ok := seq.facts.ByNumber(1)
	require.True(t, ok, "the sequencer posture did not accept the batch")

	require.Equal(t, vFact.ExecMsgsKnown, sFact.ExecMsgsKnown)
	require.Equal(t, vFact.ExecMsgs, sFact.ExecMsgs,
		"the two postures must record the same import list: they run one acceptance path, and a "+
			"sequencer that recorded a different one would disagree with its own verifiers about what "+
			"its chain consumed")
	// Non-empty control: an equality assertion over two empty lists proves nothing.
	require.Len(t, vFact.ExecMsgs, 2)
	// ...and the rest of the fact agrees too, which is the broader claim the postures rest on.
	require.Equal(t, vFact.Hash, sFact.Hash)
	require.Equal(t, vFact.OutputRoot, sFact.OutputRoot)
}
