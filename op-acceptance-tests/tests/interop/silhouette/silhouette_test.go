package silhouette

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// A silhouette chain, end to end, on a local multi-process devstack.
//
// The system these tests run on is the two-L2 LightCL-sequencer interop preset with one chain —
// l2b, called P below — turned into a silhouette chain. That means:
//
//   - P keeps its own sequencer and execution client. That is the PRIVATE side, and it is unchanged:
//     P produces real blocks with real transactions in them.
//   - P uses the ordinary op-batcher, initially stopped for deterministic tests. Its terminal
//     encoder posts transaction-stripped proof batches to a dedicated inbox.
//   - A SECOND supernode is started. It holds no execution client for P at all; its entire access to
//     P is L1. It also runs the other chain (l2a, called A below) as an ordinary chain derived from
//     L1 in the usual way.
//
// Every assertion that matters here is made on that second supernode, and the reason is worth being
// blunt about: P's LightCL and execution client are the private producer. The verifier supernode
// only sees P through proof batches and the magic EL, and P's LightCL follows that verifier's safety
// route for ordinary interop invalidation decisions.

// silhouetteSystem brings up the preset with l2b proof-carried.
func silhouetteSystem(gt *testing.T) (devtest.T, *presets.TwoL2SupernodeInterop) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0, presets.WithSilhouetteChain(presets.SilhouetteChainB))
	t.Require().NotNil(sys.Silhouette, "preset did not wire the silhouette half")
	t.Require().Equal(presets.SilhouetteChainB, sys.Silhouette.ChainKey)
	return t, sys
}

// TestSilhouetteVerifierStarts is gate A: the verifier supernode comes up with P as a silhouette
// virtual node, and the shape of the system is the one the thesis needs.
//
// Everything asserted here is a precondition for the later gates rather than a restatement of the
// setup:
//
//  1. The verifier's route for P answers. That means the whole silhouette assembly was built — the
//     manifest loaded, the L1 chain config resolved from the FILE (the only way a devstack L1 can be
//     named at all; see TestSilhouetteRefusesWrongL1ChainConfig), the shim started and mounted, the
//     proof-batch source injected.
//  2. P's private side runs. Everything below is only interesting because there IS history to prove.
//  3. P's safe head on its own SEQUENCER is still genesis. No ordinary channel batch has been
//     published; the proof-encoding batcher is initially stopped and targets a separate inbox.
//  4. None of that private history is proven history on the verifier — its proven head is the anchor.
//  5. Chain A, on the same verifier, derives from L1 normally. Without this, "P does not advance"
//     would be indistinguishable from a verifier that is not working at all.
//  6. P declares itself at its own route (G2 D8), which is also the only honest way to read a chain
//     that has no execution client.
func TestSilhouetteVerifierStarts(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette

	// (1) the silhouette route is live and is P's.
	status := sil.VerifierCL.SyncStatus()
	require.NotNil(status, "verifier has no sync status for the silhouette chain")

	// (2) the private side runs. Everything below is only interesting because there IS history to
	// prove and nothing has proved it.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)

	// (3) no ordinary channel publication: blocks produced, public safe head unmoved, on the chain's
	// OWN sequencer.
	require.Equal(uint64(0), sys.L2BSupernodeCL.HeadBlockRef(safety.LocalSafe).Number,
		"the silhouette chain's safe head advanced on its own sequencer, so something IS batching it")

	// (4) and none of that private history is proven history. A verifier that reported any head above
	// the anchor with no proof batch posted would be reading the chain from somewhere it should not.
	anchor := sil.Runtime.Config.Anchor
	require.Equal(uint64(0), anchor.BlockNumber, "this preset anchors the silhouette chain at its own genesis")
	provenHead := sil.VerifierCL.HeadBlockRef(safety.LocalSafe)
	require.Equal(anchor.BlockNumber, provenHead.Number,
		"the verifier claims proven history above the anchor with no proof batch posted")
	require.Equal(anchor.BlockHash, provenHead.Hash, "the verifier's proven head is not the anchor block")

	// (5) the same verifier derives the ordinary chain from L1 in the usual way.
	dsl.CheckAll(t, sil.PeerCL.AdvancedFn(safety.LocalSafe, 2, 90))

	// (6) the chain declares itself at its own route (G2 D8). Without this the chain would derive
	// perfectly and be unaskable, which is a service nothing can use — and the declaration is also
	// the only honest read path, since the chain has no execution client to ask.
	decl := sil.BlockProvenance(t, anchor.BlockNumber)
	require.Equal("genesis", decl.Provenance, "the anchor block should declare itself as genesis")
	require.Equal(anchor.BlockHash, decl.Hash, "the chain declares a different genesis than the anchor names")
	require.False(decl.RootsKnown, "genesis roots are configuration, not something this node knows")

	t.Logger().Info("silhouette verifier up",
		"silhouette_chain", sil.ChainID, "anchor", anchor.BlockNumber,
		"anchor_output_root", anchor.OutputRoot, "manifest", sil.Runtime.ManifestPath,
		"peer_local_safe", sil.PeerCL.HeadBlockRef(safety.LocalSafe).Number)
}

// TestSilhouetteRefusesWrongL1ChainConfig is the other half of gate A: the G5 D3 refusal.
//
// silhouette.L1ChainConfig knows mainnet, sepolia, holesky and hoodi, and refuses everything else
// unless the operator names a file. The value it reads decides whether an L1 block's excess blob gas
// is priced under Cancun or Prague, for the L1-info transaction's blob-base-fee field — a
// consensus-relevant number — so a guess is worse than a refusal. The file's chain ID must therefore
// match the chain the verifier is configured to settle on, and this test breaks exactly that and
// nothing else.
//
// The control matters as much as the assertion: the same probe against the LIVE manifest must
// succeed, or "it refused" would only be evidence that the probe refuses everything.
func TestSilhouetteRefusesWrongL1ChainConfig(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette

	require.NoError(sil.Runtime.TryVerifierWithManifest(sil.Runtime.ManifestPath),
		"a verifier must be constructible from the manifest it is already running")

	wrong := sil.Runtime.ManifestWithWrongL1ChainID(t)
	err := sil.Runtime.TryVerifierWithManifest(wrong)
	require.Error(err, "a verifier accepted an L1 chain config for a different chain")
	require.Contains(err.Error(), "but this chain settles on",
		"the refusal is not the L1-chain-config mismatch: %v", err)
	t.Logger().Info("verifier refused a mismatched L1 chain config", "err", err)
}

// TestSilhouetteProofBatchAdvancesVerifier is gate B: a proof batch, posted to L1 in a blob, moves
// P's proven head on a node that has no other access to P.
//
// Two batches rather than one, on purpose. A single batch proves that the anchor can be extended; the
// second proves that a batch can extend the FIRST — which is the chaining rule (acceptance rule 3),
// the rule that makes proven history a chain rather than a set of claims, and the rule a submitter
// bug would break silently on batch two while looking perfect on batch one.
func TestSilhouetteProofBatchAdvancesVerifier(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette

	// Let the private chain produce some history worth proving.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)

	first := sil.Batcher.SubmitNext()
	require.NotNil(first, "the silhouette chain produced no blocks to batch")
	firstLast := first.Blocks[len(first.Blocks)-1]
	t.Logger().Info("posted first proof batch",
		"blocks", len(first.Blocks), "first", first.Blocks[0].Number, "last", firstLast.Number)
	require.Equal(uint64(1), first.Blocks[0].Number, "the first batch must start one block above the anchor")

	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.LocalSafe, firstLast.Number, 120),
		sil.VerifierCL.ReachedFn(safety.CrossSafe, firstLast.Number, 120),
		sys.L2BCL.ReachedFn(safety.LocalSafe, firstLast.Number, 120),
	)
	// The head the verifier derived is the head the wire committed to — proven history is the
	// sequencer's real history, not a re-execution of it.
	provenHead := sil.VerifierCL.HeadBlockRef(safety.LocalSafe)
	require.Equal(firstLast.Hash, provenHead.Hash,
		"the verifier's proven head hash is not the hash the proof batch committed to")

	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	second := sil.Batcher.SubmitNext()
	require.NotNil(second, "no second batch to post")
	require.Equal(firstLast.Number+1, second.Blocks[0].Number, "the second batch does not chain onto the first")
	require.Equal(first.NewOutputRoot, second.PrevOutputRoot, "the second batch does not extend the first's output root")
	secondLast := second.Blocks[len(second.Blocks)-1]

	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.LocalSafe, secondLast.Number, 120))
	require.Equal(secondLast.Hash, sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Hash,
		"the verifier's proven head is not the second batch's last block")

	// No invalidations anywhere in this: deriving a chain from proofs is not supposed to disagree
	// with anything.
	require.Zero(sil.InteropInvalidations(t), "the verifier invalidated a block while consuming proofs")

	t.Logger().Info("proof-carried head advanced",
		"proven_head", secondLast.Number, "proven_hash", secondLast.Hash,
		"private_head", sys.L2BCL.HeadBlockRef(safety.LocalUnsafe).Number)
}

// TestSilhouetteCrossChainPinsThenAdvances is the phase-1 gate.
//
// The claim: a node that does not derive P can cross-safe another chain's message that references P,
// and it will WAIT for P's proof rather than invalidate the referencing block. Absence of data is not
// evidence of a conflict.
//
// The order is the whole test. The executing message lands on A before the proof batch covering P's
// initiating message does, which is what makes the pin an assertion instead of a hope: if the batch
// went first, "cross-safe advanced" would prove nothing about waiting.
//
//	P (silhouette)   ── init message at block B ──┐        (batches withheld above B-1)
//	A (ordinary)     ────────────── exec message at block N, referencing P's log
//	verifier         A cross-safe PINS below N, zero invalidations
//	L1               ── proof batch covering B ──┐
//	verifier         A cross-safe ADVANCES past N, still zero invalidations
func TestSilhouetteCrossChainPinsThenAdvances(gt *testing.T) {
	t, sys := silhouetteSystem(gt)
	require := t.Require()
	sil := sys.Silhouette
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther) // on A, the ordinary chain
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)   // on P, the silhouette chain

	// Prove some of P's early history first, so the verifier's proof pipeline is demonstrably
	// working before anything is withheld from it. A pin that started from "nothing has ever been
	// proven" would be indistinguishable from a broken verifier.
	sys.L2BCL.Advanced(safety.LocalUnsafe, 3, 60)
	warmup := sil.Batcher.SubmitNext()
	require.NotNil(warmup, "no warm-up batch to post")
	warmupHead := warmup.Blocks[len(warmup.Blocks)-1].Number
	dsl.CheckAll(t,
		sil.VerifierCL.ReachedFn(safety.CrossSafe, warmupHead, 120),
		sys.L2BCL.ReachedFn(safety.LocalSafe, warmupHead, 120),
	)

	// (1) the initiating message, on P — the private chain.
	eventLogger := bob.DeployEventLogger()
	initMsg := bob.SendInitMessage(interop.RandomInitTrigger(rng, eventLogger, 2, 8))
	initBlock := initMsg.BlockID()
	t.Logger().Info("initiating message on the silhouette chain", "block", initBlock.Number, "hash", initBlock.Hash)

	// (2) prove P up to just BELOW the initiating message's block, and stop. From here the verifier
	// holds no proof of P's log, which is the live-edge state a verifier is in between batches.
	// The ordinary op-batcher remains stopped while this block is created, so the verifier cannot
	// have received a proof covering it yet.
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.LocalSafe, initBlock.Number-1, 120))
	require.Less(sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Number, initBlock.Number,
		"the verifier already has proof of the initiating message's block")

	// (3) the executing message, on A — after the withheld boundary is in place.
	execMsg := alice.SendExecMessage(initMsg)
	execBlock := execMsg.BlockID()
	t.Logger().Info("executing message on the ordinary chain", "block", execBlock.Number, "hash", execBlock.Hash)

	invalidationsBefore := sil.InteropInvalidations(t)

	// (4a) THE PIN. A's block is on L1 and locally safe on the verifier, but it cannot be cross-safe:
	// the message it executes is not proven yet. It must WAIT, not invalidate.
	dsl.CheckAll(t,
		sil.PeerCL.ReachedFn(safety.LocalSafe, execBlock.Number, 240),
	)
	pinnedHash := sil.PeerEL.BlockRefByNumber(execBlock.Number).Hash
	require.Equal(execBlock.Hash, pinnedHash,
		"the verifier derived a different block at the executing message's height")

	dsl.CheckAll(t, staysBelowFn(t, sil.PeerCL, safety.CrossSafe, execBlock.Number, 15))
	require.Equal(invalidationsBefore, sil.InteropInvalidations(t),
		"the verifier invalidated a block while waiting for a proof; a late message is not a bad one")

	pinned := sil.PeerCL.HeadBlockRef(safety.CrossSafe)
	t.Logger().Info("cross-safe pinned below the executing message",
		"cross_safe", pinned.Number, "exec_block", execBlock.Number,
		"local_safe", sil.PeerCL.HeadBlockRef(safety.LocalSafe).Number,
		"proven_p_head", sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Number)

	// (5) release the proof. P is proven past the initiating message, and past the executing
	// message's timestamp, which is what an interop round needs from every chain in the set.
	sil.Batcher.Start(2 * time.Second)
	sil.Batcher.WaitBatched(initBlock, 5*time.Minute)
	released := sil.Batcher.BatchedHead()
	dsl.CheckAll(t, sil.VerifierCL.ReachedFn(safety.CrossSafe, initBlock.Number, 120))
	t.Logger().Info("proof released", "proven_p_head", released, "init_block", initBlock.Number)

	// P must keep being proven for the round to reach the executing message's timestamp: the
	// interop round decides a timestamp only when EVERY chain in the dependency set has safe data
	// covering it. On a real deployment that is the proof cadence; here it is the ticker.

	// (4b) THE ADVANCE. Same block, by hash, now cross-safe on the ordinary chain — and the
	// silhouette chain declares the initiating message's block PROVEN, at the hash the sequencer
	// really produced.
	dsl.CheckAll(t,
		sil.PeerCL.ReachedRefFn(safety.CrossSafe, execBlock, 300),
		sil.VerifierCL.ReachedFn(safety.CrossSafe, initBlock.Number, 300),
	)
	initDecl := sil.BlockProvenance(t, initBlock.Number)
	require.Equal("proven", initDecl.Provenance,
		"the initiating message's block is not proven on the verifier")
	require.Equal(initBlock.Hash, initDecl.Hash,
		"the verifier's proven block at the initiating message's height is a different block")
	require.NotNil(initDecl.Carrier, "a proven block must name the L1 block that carried its proof")

	// The structural half of "zero invalidations": an invalidation REPLACES the block at that height
	// with a deposits-only one, so the executing message's block still having the hash it had before
	// cross-safe passed it is the same claim, read off the chain instead of off a counter.
	// (ReachedRefFn above already refuses a hash change at that height; this states it.)
	require.Equal(pinnedHash, sil.PeerEL.BlockRefByNumber(execBlock.Number).Hash,
		"the block at the executing message's height was replaced on the way to cross-safe")

	require.Equal(invalidationsBefore, sil.InteropInvalidations(t),
		"the verifier invalidated a block on the way to cross-safing it")
	// ...and that zero is a real zero rather than a quiet verifier: the same scrape of the same
	// registry shows it has been verifying timestamps.
	require.Positive(sil.TimestampsVerified(t),
		"the verifier verified no timestamps at all, so its zero invalidation count says nothing")

	// And the contrast that makes the claim sharp. The SEQUENCER supernode has P's receipts sitting
	// in the execution client it drives, and it still cannot cross-safe A's block: it runs P as an
	// ordinary chain, P has no ordinary channel batches, so P's public local-safe label never moves and the round's
	// readiness check gates on every chain in the set. Proofs are what unblocked the verifier, and
	// only the verifier has them. (In a real deployment the sequencer side runs the "proven-head"
	// posture for exactly this reason; phase 1 leaves it ordinary so the difference is visible.)
	seqCrossSafe := sys.L2ACL.HeadBlockRef(safety.CrossSafe)
	require.Less(seqCrossSafe.Number, execBlock.Number,
		"the sequencer supernode cross-safed the executing message without any proof; "+
			"the gate above then proves nothing about proofs")

	t.Logger().Info("cross-safe advanced past the executing message",
		"cross_safe", sil.PeerCL.HeadBlockRef(safety.CrossSafe).Number,
		"exec_block", execBlock.Number,
		"proven_p_head", sil.VerifierCL.HeadBlockRef(safety.LocalSafe).Number,
		"invalidations", sil.InteropInvalidations(t))
}

// staysBelowFn asserts that a head stays STRICTLY BELOW target for the whole window.
//
// NotAdvancedFn is the wrong tool for a pin like this: the head is expected to keep moving, just not
// past a particular block. Freezing entirely would be a different and worse symptom, and a check that
// demanded it would pass for a verifier that had simply stopped.
func staysBelowFn(t devtest.T, cl *dsl.L2CLNode, lvl safety.Level, target uint64, attempts int) dsl.CheckFunc {
	return func() error {
		logger := t.Logger().With("chain", cl.ChainID(), "label", lvl, "must_stay_below", target)
		logger.Info("Expecting head to pin below target")
		for i := range attempts {
			head := cl.HeadBlockRef(lvl)
			if head.Number >= target {
				return fmt.Errorf("%s head reached %d, which is not below %d: the pin did not hold",
					lvl, head.Number, target)
			}
			logger.Info("Head pinned", "attempt", i+1, "current", head.Number)
			if err := clock.SystemClock.SleepCtx(t.Ctx(), 2*time.Second); err != nil { // nosemgrep: flake-sleep-in-test -- asserting absence of progress past a bound; no chain event to wait on
				return err
			}
		}
		logger.Info("Head stayed below target for the whole window")
		return nil
	}
}
