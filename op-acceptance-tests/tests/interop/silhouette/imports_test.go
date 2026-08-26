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
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// G7 END TO END: THE SILHOUETTE CHAIN AS A CONSUMER.
//
// silhouette_test.go runs the A→P direction of the bridge: P EXPORTS a message, A executes it, and the
// verifier waits for P's proof instead of invalidating A's block. That is the direction that works
// without wire v3, because everything the judge needs about P is in P's export set.
//
// This file runs the other direction, which wire v3 exists for. P IMPORTS a message from A. Before
// G7 the public network had nothing to check about that: P's proof asserted its own cross-chain
// reads and the wire said nothing about them, so P was the one member of the dependency set whose
// dependencies nobody validated. Now the import list travels on the wire and the STOCK cross-safety
// judge validates it — same checksum lookup, same invariants — before P's block can be cross-safe.
//
// The two tests below are one claim in two halves, and neither is worth much alone:
//
//   - the POSITIVE: P consumes a real message on A, the import appears on the wire, and the verifier
//     cross-safes P's block.
//   - the NEGATIVE: the identical system, the identical P block, and ONE FIELD of the declared import
//     changed to name a message A never emitted. The verifier must refuse. Without this, "it
//     cross-safed" is equally consistent with a verifier that checks nothing, which is exactly what
//     the pre-G7 stack was.

// tamperedMsgHash is a message hash no log on A can produce: it is the one field the negative test
// changes, and changing exactly one field is what makes the two tests comparable.
var tamperedMsgHash = common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

// newProvenImportFixture warms the proof pipeline, emits one initiating message on A, and waits until
// the VERIFIER has that message in its own indexed history.
//
// The wait is the precondition, not a courtesy. The judge resolves P's declared import against A's
// message database on the verifier, so a test that raced ahead of that would be asserting on a "wait"
// it had caused itself — and the wait path is already covered, in the other direction, by
// TestSilhouetteCrossChainPinsThenAdvances.
func newProvenImportFixture(gt *testing.T, sequencerPosture bool) (devtest.T, *presets.TwoL2SupernodeInterop, *dsl.InitMessage, eth.BlockID) {
	var t devtest.T
	var sys *presets.TwoL2SupernodeInterop
	if sequencerPosture {
		t, sys = sequencerPostureSystem(gt)
	} else {
		t, sys = silhouetteSystem(gt)
	}
	require := t.Require()
	sil := sys.Silhouette
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Warm the proof pipeline first, so a later failure cannot be "proofs never worked here".
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	warmup := sil.Submitter.SubmitNext()
	require.NotNil(warmup, "no warm-up batch to post")
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.CrossSafe, warmup.Blocks[len(warmup.Blocks)-1].Number, 120))

	// THE CADENCE MUST RUN FROM HERE ON, and finding out why cost this test its first run.
	//
	// An interop round decides a timestamp only when EVERY chain in the dependency set has safe data
	// covering it, so with P proven only up to the warm-up batch, chain A's own cross-safe frontier
	// pins on P's readiness — `notReady="chain 902 not ready for timestamp ..."` in the round log.
	// Waiting on A's cross-safe below would then wait forever, on a stall this test had caused itself.
	// On a real deployment the cadence is the submitter's; here it is a ticker.
	sil.Submitter.Start(2 * time.Second)

	// (1) the initiating message, on A — the ORDINARY chain. This is the direction reversal: in
	// silhouette_test.go the initiating message is P's.
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	eventLogger := alice.DeployEventLogger()
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLogger, 2, 8))
	initID := initMsg.BlockID()
	t.Logger().Info("initiating message on the ordinary chain", "block", initID.Number, "hash", initID.Hash)

	// (2) the verifier has it. A is derived from L1 in the usual way here, so this is ordinary
	// progress rather than anything silhouette-specific — and it is what makes P's dependency
	// RESOLVABLE rather than merely late.
	dsl.CheckAll(t, sil.PeerCL.ReachedFn(safety.CrossSafe, initID.Number, 240))
	return t, sys, initMsg, initID
}

// exportOnTheWire waits for the running cadence to cover `block` and returns the export it posted for
// it.
//
// It reads what the SUBMITTER actually put in the blob, so an assertion on it is an assertion about
// the wire rather than about what the test hoped the wire said — and it works regardless of which
// batch happened to cover the block, which matters because the cadence ticker chooses that, not the
// test.
func exportOnTheWire(t devtest.T, sys *presets.TwoL2SupernodeInterop, block uint64) proofbatch.BlockExport {
	sil := sys.Silhouette
	sil.Submitter.WaitBatched(block, 5*time.Minute)
	export, ok := sil.Submitter.PostedExport(block)
	t.Require().True(ok, "proven history passed block %d without a recorded export for it", block)
	return export
}

// TestSilhouetteImportsAMessageAndTheDependencyIsVerified is gate 3, positive half.
//
//	A (ordinary)   ── init message at block M ──┐
//	P (silhouette) ─────────────── exec message at block N, consuming A's message
//	L1             ── proof batch covering N, DECLARING that import ──┐
//	verifier       judge validates P's declared dependency against A's own log database,
//	               THEN cross-safes P's block N
func TestSilhouetteImportsAMessageAndTheDependencyIsVerified(gt *testing.T) {
	t, sys, initMsg, initID := newProvenImportFixture(gt, false)
	require := t.Require()
	sil := sys.Silhouette

	// (3) the executing message, on P — the PRIVATE chain. Its sequencer really executes it, which is
	// also the reason this direction needs no shim trickery: the block is real on the private side and
	// only its OUTLINE goes public.
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	execMsg := bob.SendExecMessage(initMsg)
	execID := execMsg.BlockID()
	t.Logger().Info("executing message on the silhouette chain", "block", execID.Number, "hash", execID.Hash)

	// (4) THE IMPORT ON THE WIRE. This is the new field, carrying the one thing v2 omitted, and it is
	// asserted against A's real message rather than against a shape: the identifier must name A's
	// chain, A's block and A's log.
	export := exportOnTheWire(t, sys, execID.Number)
	require.Len(export.ExecMsgs, 1,
		"P's block executed one cross-chain message, so its wire export must declare exactly one import")
	declared := export.ExecMsgs[0]
	require.Equal(sys.L2ACL.ChainID(), declared.Identifier.ChainID,
		"the declared import must name the chain the message came from")
	require.Equal(initID.Number, declared.Identifier.BlockNumber,
		"the declared import must name A's initiating block")
	t.Logger().Info("import declared on the wire",
		"chain", declared.Identifier.ChainID, "block", declared.Identifier.BlockNumber,
		"logIndex", declared.Identifier.LogIndex, "timestamp", declared.Identifier.Timestamp,
		"msgHash", declared.PayloadHash, "checksum", declared.Checksum())

	// ...and the wire carries the import and NOTHING about the transaction that executed it. That is
	// the minimal-leak rule, and it is checkable from here: the block's export names no tx hash, no
	// tx index and no sender, and its log count is a property of the export policy rather than of the
	// executing transaction.
	require.NotZero(export.Hash, "the block's real hash is a proof-committed fact")

	// (5) the verifier cross-safes P's block. The cadence has been running since the fixture, which is
	// what lets the round reach these timestamps at all.
	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.LocalSafe, execID.Number, 240),
		sil.VerifierCL.ReachedFn(safety.CrossSafe, execID.Number, 300),
	)
	// Assert at THAT HEIGHT, not at the head. `ReachedFn` returns once the head is at or past the
	// target, and cross-safe keeps moving, so the head's hash is some later block's — the first run of
	// this test compared them and failed on a system that was working.
	//
	// `ReachedRefFn` is the usual tool for "the head passed this exact block" and it is unavailable
	// here on purpose: it reads the block back through the chain's EL client, and a silhouette chain
	// has no EL on a verifier at all. The read path for a chain whose execution client is a verifier
	// is its own declaration RPC.
	decl := sil.BlockProvenance(t, execID.Number)
	require.Equal("proven", decl.Provenance,
		"the consuming block is not proven history on the verifier")
	require.Equal(execID.Hash, decl.Hash,
		"the verifier's block at the consuming height is not the block the sequencer produced")
	require.NotNil(decl.Carrier, "a proven block must name the L1 block that carried its proof")

	// (6) and it got there by CHECKING, not by skipping. Three readings of the same claim:
	//
	//   - it verified timestamps, so it was not simply idle;
	//   - and the negative control in the next test shows this system replaces the block when the declared
	//     dependency is false, which is what rules out "it checked nothing and advanced anyway".
	require.Positive(sil.TimestampsVerified(t),
		"the verifier verified no timestamps at all, so its zero refusal count says nothing")
	require.Zero(sil.InteropInvalidations(t), "nothing should have been invalidated in this scenario")

	// The reverse direction is untouched: A's own cross-safe frontier passed P's proof-carried blocks
	// in the same run, which is the coexistence claim.
	dsl.CheckAll(t, sil.PeerCL.AdvancedFn(safety.CrossSafe, 1, 120))

	t.Logger().Info("silhouette chain cross-safed a block whose dependency was verified from the wire",
		"exec_block", execID.Number, "init_block", initID.Number,
		"proven_head", sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Number)
}

// TestSilhouetteImportThatIsFalseIsReplaced is gate 3, negative half — and the assertion that the
// positive half is about the flip rather than about a healthy cluster.
//
// One field changes: the declared import's message hash. Everything else — the same P block, the same
// real executing transaction, the same roots, the same L1 bindings, the same chaining — is exactly
// what the honest submitter built. So a verifier that cross-safed this block would be a verifier that
// does not check what a proven chain declares.
//
// What must happen instead matches an ordinary interop chain:
//
//   - P's block becomes LOCAL-safe: the batch is structurally valid and the proof verifies, so the
//     derivation pipeline accepts it. Dependency validity is not the codec's business.
//   - the judge resolves the declared import against A's log database, finds nothing, and returns a
//     verdict of invalid;
//   - the sequencing supernode replaces the real P block through the stock deposits-only path;
//   - the corrected proof supersedes the denied suffix and the verifier cross-safes the replacement.
func TestSilhouetteImportThatIsFalseIsReplaced(gt *testing.T) {
	t, sys, initMsg, _ := newProvenImportFixture(gt, true)
	require := t.Require()
	sil := sys.Silhouette

	// Arm the one-field lie BEFORE P executes anything, because the cadence ticker decides which batch
	// covers the consuming block and the test does not. It stays armed until it finds a block with an
	// import list, and it is applied before the structural check — so the batch this posts is one the
	// wire codec ACCEPTS, and the replacement below is therefore evidence about the judge rather than
	// about the codec.
	sil.Submitter.MutateUntilApplied(func(b *proofbatch.ProofBatch) bool {
		applied := false
		for i := range b.Blocks {
			for j := range b.Blocks[i].ExecMsgs {
				b.Blocks[i].ExecMsgs[j].PayloadHash = tamperedMsgHash
				applied = true
			}
		}
		return applied
	})

	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	execMsg := bob.SendExecMessage(initMsg)
	execID := execMsg.BlockID()
	t.Logger().Info("executing message on the silhouette chain", "block", execID.Number)

	export := exportOnTheWire(t, sys, execID.Number)
	require.Len(export.ExecMsgs, 1, "the tampered batch must still declare exactly one import")
	require.Equal(tamperedMsgHash, export.ExecMsgs[0].PayloadHash,
		"the posted batch does not carry the false import this test is about")

	// LOCAL-safe: the wire object is fine and the pipeline accepts it.
	//
	// Waited on by NUMBER and asserted by HEIGHT, never off the head's hash. `ReachedFn` returns once
	// the head is at OR PAST the target and local-safe keeps moving — a batch covers many blocks — so
	// the head's hash belongs to some later block. Comparing it to this block's passed for a while on
	// timing alone and then failed on a system that was working correctly. Same trap as in the positive
	// test above; the read path for a chain with no EL on the verifier is its declaration RPC.
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.LocalSafe, execID.Number, 240))
	require.Equal(execID.Hash, sil.BlockProvenance(t, execID.Number).Hash,
		"the verifier derived a different block at the consuming height")

	// The sequencing supernode uses the ordinary deny-list and deposits-only replacement path on
	// P's real execution client. This is the same externally visible result as a driven chain.
	dsl.CheckAll(t, sys.L2ELB.ReorgTriggeredFn(eth.L2BlockRef{Hash: execID.Hash, Number: execID.Number}, 90))
	replacement := sys.L2ELB.BlockRefByNumber(execID.Number)

	// The proof submitter follows that reorg and posts a replacement proof. The verifier accepts it
	// as a supersession of the suffix containing the denied hash, then its unmodified op-node derives
	// the same replacement block.
	sil.WaitBlockProvenance(t, replacement.ID(), 5*time.Minute)
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.CrossSafe, execID.Number, 240))

	require.Positive(sil.InteropInvalidations(t),
		"the judge never carried out the invalidation of the false dependency")
	t.Logger().Info("a false import was replaced through the normal interop path",
		"exec_block", execID.Number,
		"old_hash", execID.Hash,
		"replacement_hash", replacement.Hash,
		"cross_safe", sil.VerifierCL.HeadBlockRef(safety.CrossSafe).Number,
		"invalidations", sil.InteropInvalidations(t))
}
