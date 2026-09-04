package silhouette

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
)

// V1: THE ATTESTED SYSTEM, END TO END, AND WHAT IT DOES NOT GIVE.
//
// Every test in this package already ran under v1's proving system — the preset configures P with
// `proofType: attested` and its submitter posts an empty proof slot. That was previously called "mock
// mode", and calling it that hid the interesting fact: this IS the system. The proof of the chain is
// the operator's attestation, and the attestation is the L1 batch transaction's own signature, bound
// by acceptance rule 1's tx.from == submitterEOA.
//
// This file makes that explicit rather than incidental:
//
//   - the label is asserted, so nobody has to infer the trust model from a submitter's behaviour;
//   - the proof slot rule is exercised through the whole multi-process stack;
//   - and the trust model's COST is written down as a passing test, which is the part that matters.
//     TestAttestedFabricatedExportIsAccepted asserts that a fabricated export IS accepted. It is not
//     a bug and it is not a gap in the implementation: it is what attestation means, and a system
//     whose weakness lives only in a document is a system whose weakness will be forgotten.
//
// See op-supernode/silhouette/docs/TRUST-MODEL.md.

// TestAttestedIsTheConfiguredProvingSystem is the label, stated once, so that every other assertion in
// this package is known to be about the attested system.
//
// It reads the verifier's OWN configuration rather than inferring the mode from an empty proof slot: a
// submitter posting no proof is consistent both with attested mode and with a proof-verifying verifier
// about to reject everything, and those two are the same picture until a batch is accepted.
func TestAttestedIsTheConfiguredProvingSystem(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	cfg := sys.Silhouette.Runtime.Config

	require.Equal(silhouette.ProofTypeAttested, cfg.ProofType,
		"this suite claims to exercise the attested system; the verifier says otherwise")
	require.True(cfg.Attested())

	// The other half of v1's posture, and the reason it is asserted beside the proving system: wire v3
	// and the judge flip stay ON. Cross-chain import consistency is genuinely checked even though state
	// validity is not, so these two facts are only meaningful together.
	require.True(cfg.DependenciesVerified(),
		"v1 keeps the G7 judge flip: a lying attester's claimed imports are still checked against real A")

	t.Logger().Info("v1 posture", "proofType", cfg.ProofType,
		"wireVersion", cfg.EffectiveWireVersion(), "dependenciesVerified", cfg.DependenciesVerified(),
		"attester", cfg.Submitter, "inbox", cfg.Inbox)
}

// TestAttestedRefusesABatchCarryingProofBytes runs the proof-slot rule through the whole stack: a real
// submitter, a real blob transaction on a real L1, a real derivation pipeline.
//
// The rule reads oddly until you name what it protects. An attested verifier cannot check a proof, so
// proof bytes reaching one are bytes nobody will ever look at. Accepting the batch anyway would leave
// the system in its worst reachable state: something proof-shaped on the wire, an operator who
// believes it was checked, and no error anywhere to disagree. So the batch is refused and the proven
// head does not move.
//
// The warm-up is the control. It proves this exact pipeline, submitter and verifier accept an honest
// batch minutes earlier, so the refusal below is about the four bytes and nothing else.
func TestAttestedRefusesABatchCarryingProofBytes(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette
	require.Equal(silhouette.ProofTypeAttested, sil.Runtime.Config.ProofType)

	// (1) the control: an honest attested batch is accepted, and cross-safed.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	warmup := sil.Batcher.SubmitNext()
	require.NotNil(warmup, "no warm-up batch to post")
	warmupHead := warmup.Blocks[len(warmup.Blocks)-1].Number
	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.CrossSafe, warmupHead, 120),
		sys.L2BCL.ReachedFn(safety.LocalSafe, warmupHead, 120),
	)

	// (2) the same submitter, the same chain, the next range of blocks — with bytes in the proof slot.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	sil.Batcher.ProofBytesOnNext([]byte{0xde, 0xad, 0xbe, 0xef})
	refused := sil.Batcher.SubmitNext()
	require.NotNil(refused, "no batch to post")
	require.Greater(refused.Blocks[len(refused.Blocks)-1].Number, warmupHead,
		"the proof-carrying batch must extend beyond blocks the verifier has already accepted")
	t.Logger().Info("posted a batch carrying proof bytes to an attested verifier",
		"first", refused.Blocks[0].Number, "last", refused.Blocks[len(refused.Blocks)-1].Number)

	// (3) it is refused: the proven head stays where the honest batch left it. Reject-and-log, so the
	// verifier keeps running — a node that halted here would be a node an adversary can stop by
	// posting garbage to a public inbox.
	dsl.CheckAll(t, staysBelowFn(t, sil.VerifierCL, safety.LocalSafe, warmupHead+1, 15))
	require.Zero(sil.InteropInvalidations(t),
		"a refused batch must not invalidate anything; it was never accepted in the first place")

	t.Logger().Info("a batch carrying proof bytes was refused and the proven head held",
		"proven_head", sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Number,
		"refused_from", refused.Blocks[0].Number)
}

// TestAttestedFabricatedExportIsAccepted IS THE TRUST MODEL, AS A PASSING TEST.
//
// P's operator attests to a batch declaring an exported log that P never emitted — no such log exists
// on any chain, and its hash is invented. The verifier ACCEPTS it, seals it into its interop log
// database at a real block and a real index, and cross-safes the block that carries it. From that
// moment the fabricated message is referenceable: it sits in the same database, under the same key,
// as every genuine message a peer resolves an executing message against.
//
// THIS IS NOT A BUG. It is what "the proof of the chain is the operator's attestation" means, stated
// where it cannot be overlooked. A verifier of an attested chain has no execution client for it, no
// state, and no way to know which logs P really emitted — its whole knowledge of P is a signed blob.
// So an operator who lies is believed, and the only thing standing behind P's exports is that P's
// operator signed for them with a key the whole dependency set knows.
//
// What is NOT weakened, and is worth reading beside this: the fabricated log is a fabricated EXPORT.
// P's IMPORTS are still checked — see TestSilhouetteImportThatIsFalseIsReplaced, where one false
// declared import causes the consuming block to be replaced. An attester can invent what its own
// chain said; it cannot invent what someone else's chain said.
//
// Under a proving system that actually checks P's state this test would fail at step (3), and that is
// the upgrade path: same wire, same proof slot, one config value. The test is therefore also a marker
// — if it ever starts failing, someone has changed the trust model, and that is a thing to know
// deliberately.
func TestAttestedFabricatedExportIsAccepted(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	require.Equal(silhouette.ProofTypeAttested, sil.Runtime.Config.ProofType)

	// Warm up, so a later acceptance cannot be "this verifier accepts anything from the start".
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	warmup := sil.Batcher.SubmitNext()
	require.NotNil(warmup, "no warm-up batch to post")
	warmupHead := warmup.Blocks[len(warmup.Blocks)-1].Number
	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.CrossSafe, warmupHead, 120),
		sys.L2BCL.ReachedFn(safety.LocalSafe, warmupHead, 120),
	)
	sil.Batcher.Start(2 * time.Second)

	// (1) a real message on P, so the batch has a block with a real export to append the lie to. The
	// fabrication then differs from an honest batch in exactly one entry.
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	eventLogger := bob.DeployEventLogger()
	initMsg := bob.SendInitMessage(interop.RandomInitTrigger(rng, eventLogger, 2, 8))
	initID := initMsg.BlockID()

	// (2) THE LIE. One extra exported log, at the next index in the same block, with a hash no log on
	// any chain produces. Nothing else about the batch changes: real block hash, real roots, real
	// chaining, real L1 bindings, a real L1 transaction signed by the real submitter key.
	//
	// It is armed until applied because the cadence ticker decides which batch covers this block.
	var fabricated fabricatedLog
	sil.Batcher.MutateUntilApplied(func(b *proofbatch.ProofBatch) bool {
		for i := range b.Blocks {
			if b.Blocks[i].Number != initID.Number || len(b.Blocks[i].Logs) == 0 {
				continue
			}
			last := b.Blocks[i].Logs[len(b.Blocks[i].Logs)-1]
			fabricated = fabricatedLog{
				block: b.Blocks[i].Number,
				index: last.Index + 1,
				hash:  fabricatedLogHash,
			}
			b.Blocks[i].Logs = append(b.Blocks[i].Logs, proofbatch.LogExport{
				Index: fabricated.index,
				Hash:  fabricated.hash,
			})
			return true
		}
		return false
	})

	// (3) THE ACCEPTANCE. This is the assertion the trust model lives in.
	export := exportOnTheWire(t, sys, initID)
	require.NotZero(fabricated.block, "the fabrication was never applied, so this test asserts nothing")
	require.Equal(initID.Number, fabricated.block)

	var found bool
	for _, l := range export.Logs {
		if l.Index == fabricated.index && l.Hash == fabricated.hash {
			found = true
		}
	}
	require.True(found, "the fabricated log is not on the wire, so nothing below is about it")
	t.Logger().Info("fabricated export posted to L1 under the operator's signature",
		"block", fabricated.block, "logIndex", fabricated.index, "logHash", fabricated.hash,
		"attester", sil.Runtime.Config.Submitter)

	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.LocalSafe, fabricated.block, 240),
		// ...and CROSS-safe, which is the state that makes it referenceable by the rest of the
		// dependency set rather than merely present on this node.
		sil.VerifierCL.ReachedFn(safety.CrossSafe, fabricated.block, 300),
	)

	// Read it off the chain, not off a head: the block is proven history at the hash P's sequencer
	// really produced, and it carries a log P never emitted.
	decl := sil.BlockProvenance(t, fabricated.block)
	require.Equal("proven", decl.Provenance,
		"a fabricated export must be ACCEPTED under attestation — if this fails, the trust model changed")
	require.Equal(initID.Hash, decl.Hash)
	require.NotNil(decl.Carrier, "the accepted batch must name the L1 block that carried it")

	// Nothing was invalidated and nothing was refused. The verifier did not detect the lie, because
	// there is nothing here for it to detect the lie WITH.
	require.Zero(sil.InteropInvalidations(t),
		"the fabrication must be accepted outright, not accepted-then-repaired")
	require.Positive(sil.TimestampsVerified(t),
		"the verifier verified no timestamps, so this acceptance says nothing about a working verifier")

	t.Logger().Info("V1 TRUST MODEL: a fabricated export was accepted and cross-safed",
		"block", fabricated.block, "logIndex", fabricated.index,
		"cross_safe", sil.VerifierCL.HeadBlockRef(safety.CrossSafe).Number,
		"note", "this is what attestation buys and what it does not; proving P's state makes it impossible")
}

// fabricatedLogHash is a log hash no log on any chain can produce: it is the invented export.
var fabricatedLogHash = common.HexToHash("0xfabf1cafabf1cafabf1cafabf1cafabf1cafabf1cafabf1cafabf1cafabf1ca0")

type fabricatedLog struct {
	block uint64
	index uint32
	hash  common.Hash
}
