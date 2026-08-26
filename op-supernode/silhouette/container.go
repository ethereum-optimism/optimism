package silhouette

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// ErrNoExecutionData is returned by the surfaces that would require re-executing a silhouette
// chain. A proof commits to the message hashes a block exported and to nothing else about its
// transactions.
var ErrNoExecutionData = errors.New("proof-carried chain has no execution data")

// LabelSource says where a silhouette chain's public safety labels come from, which is the ONE
// thing that differs between a supernode that verifies P and a supernode that sequences it.
type LabelSource uint8

const (
	// LabelsFromDerivation is the verifier posture: this node derives P itself, through a stock
	// op-node reading the injected proof-batch source and talking to the shim. The embedded
	// container's own labels are therefore correct, and the fact store is only consulted for
	// provenance.
	LabelsFromDerivation LabelSource = iota
	// LabelsFromProvenHead is the sequencer posture: this node fronts P's REAL execution client,
	// which has no public derivation behind it at all. See Container.OptimisticAt.
	LabelsFromProvenHead
)

// Container is a silhouette chain's presence in a supernode's chains map.
//
// It wraps a stock chain container rather than replacing one, because almost everything a silhouette
// chain needs is what every chain needs: a virtual node, an engine controller, a deny list, an RPC
// handler. What it changes is exactly three things, and each is a design decision rather than an
// adaptation:
//
//  1. INGESTION. Its initiating messages come from the wire, not from receipts (the capability
//     seam). This holds in BOTH postures — see the note on the sequencer side below, where it is
//     least obvious and most important.
//  2. REFUSALS. FetchReceipts and InvalidateBlock refuse, which is where the honesty line lives in
//     code (DR-1, E3).
//  3. LABELS, in the sequencer posture only.
type Container struct {
	cc.InteropChain
	log    log.Logger
	facts  *FactStore
	labels LabelSource
	// onInvalidationRefused fires when the judge reaches a verdict of invalid for this chain and is
	// refused. It is the alerting hook, deliberately shaped like the shim's OnHalt (G4 D7): the
	// condition is operationally identical — this node cannot advance the chain and only an
	// off-node action can fix it — so it must be able to page rather than only appear in a log.
	onInvalidationRefused func(height uint64, hash common.Hash)
}

var (
	_ cc.InteropChain         = (*Container)(nil)
	_ cc.MessageIngestion     = (*Container)(nil)
	_ cc.ProvenMessageImports = (*Container)(nil)
)

// NewContainer wraps a chain container as a silhouette chain.
func NewContainer(logger log.Logger, inner cc.InteropChain, facts *FactStore, labels LabelSource) *Container {
	return &Container{InteropChain: inner, log: logger, facts: facts, labels: labels}
}

// OnInvalidationRefused installs the alerting hook. Idempotent callers replace the hook; there is
// deliberately no chaining, because two owners of one alert is worse than one.
func (c *Container) OnInvalidationRefused(fn func(height uint64, hash common.Hash)) {
	c.onInvalidationRefused = fn
}

// IngestionSource declares that this chain's initiating messages are sealed from validity proofs.
//
// It is IngestionProven in BOTH postures, and the sequencer side is the one worth stating out loud:
// that node HAS P's receipts, sitting in the execution client it fronts. It must not use them. The
// public network's view of P is the set of messages P chose to EXPORT, and a sequencer supernode is
// a member of that public network. A node that ingested from receipts would hold a strictly larger
// message set than every other verifier, and would accept executing messages referencing logs P
// never exported — a private-side node unilaterally widening the export policy. Both postures read
// the same wire, so both see the same chain.
func (c *Container) IngestionSource() cc.IngestionSource { return cc.IngestionProven }

// FetchReceipts refuses.
//
// Every interop path that would reach this is gated on IngestionSource, so it should never be
// called — but it refuses rather than returning an empty receipt list, because an empty list is a
// CLAIM ("this block emitted no logs") that this node cannot make. The refusal is what turns "no
// path reaches a proven chain's receipts" from an argument into an assertion.
func (c *Container) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, optypes.Receipts, error) {
	return nil, nil, fmt.Errorf("chain %s: FetchReceipts for block %s: %w", c.ID(), blockID, ErrNoExecutionData)
}

// ProvenExecMsgs reports the executing messages the wire declared for one of this chain's blocks —
// the G7 import list, read out of the fact store and handed to the STOCK cross-safety judge.
//
// THE KEYS ARE SET ORDINALS, NOT LOG POSITIONS. The map shape is the judge's, whose keys are log
// indices for a driven chain; wire v3 carries no position for an executing message, so what goes in
// is 0..n-1 over the wire's canonically ordered set. Nothing on the verification path may read them
// as positions, and one thing would if it could — the same-timestamp cycle graph orders executing
// messages by log index. It never sees these, because the codec REFUSES a batch whose import is
// stamped at its own block's timestamp (proofbatch.ErrSameTimestampImport), so a proven chain
// contributes no same-timestamp executing messages by construction. That refusal and this ordinal
// key are one decision, not two (G7G D1/D2).
//
// The three returns are the capability's tri-state, and the error case is the one that earns its
// keep: the fact table is a WINDOW, so a judge lagging far behind the proven head asks about a block
// whose facts have been pruned. Answering "no imports" there would be a verifier that validates
// nothing while reporting that it validates dependencies — fail-open, silently, exactly at the
// moment it is furthest behind. So absence is an error, and the round retries or stalls loudly.
func (c *Container) ProvenExecMsgs(blockNum uint64) (map[uint32]*messages.ExecutingMessage, bool, error) {
	fact, ok := c.facts.ByNumber(blockNum)
	if !ok {
		oldest, haveWindow := c.facts.Oldest()
		if haveWindow && blockNum < oldest.Number {
			return nil, false, fmt.Errorf("chain %s: block %d is below the fact window (oldest %d), so this "+
				"node can no longer say what it imported: %w", c.ID(), blockNum, oldest.Number, ErrNoExecutionData)
		}
		return nil, false, fmt.Errorf("chain %s: no facts for block %d, so this node cannot say what it "+
			"imported: %w", c.ID(), blockNum, ErrNoExecutionData)
	}
	if !fact.ExecMsgsKnown {
		// A pre-v3 wire said nothing about imports. Reported as "not on the wire" rather than as an
		// error, because it is a correct description of a correctly configured older chain — and the
		// caller is required to log the weaker posture rather than assume it.
		return nil, false, nil
	}
	msgs := make(map[uint32]*messages.ExecutingMessage, len(fact.ExecMsgs))
	for i := range fact.ExecMsgs {
		msg := fact.ExecMsgs[i]
		msgs[uint32(i)] = &msg //nolint:gosec // ordinal over a wire set bounded by a block's gas limit
	}
	return msgs, true, nil
}

// InvalidateBlock refuses, and under G7 that refusal changed meaning without changing behaviour.
//
// BEFORE (ruling-7 posture, retired): no path could reach here at all. P's own executing messages
// were proof-trusted, so the message check never marked it invalid; its block hashes came from the
// same proofs the frontier was built from, so the hash-mismatch check never fired; and it
// contributed no same-timestamp executing messages, so it could not participate in a cycle. The
// refusal was a backstop behind an argument.
//
// NOW: the first of those three is GONE ON PURPOSE. P has real dependencies on the wire and the stock
// judge validates them exactly like a driven chain's — checksum, hazards, expiry, activation — so a
// verdict of "invalid" is reachable for P for the first time. What replaces the retired clause is a
// sharper statement of what was always the load-bearing half:
//
//	A PROVEN CHAIN IS NEVER REPLACED, ONLY STOPPED.
//
// Invalidating a block means rewinding the chain onto a SYNTHESISED replacement block — a
// deposits-only block the verifier builds itself. A verifier may not do that to a proven chain,
// whatever verdict it reached, because that chain's canonical history is the proof stream on L1 and
// the verifier is not its author. Rewriting it locally would make this node's P a different chain
// from every other member's P, which is the one failure the whole design exists to prevent.
//
// So the invalidation path is now REACHABLE and TERMINAL: it refuses here, loudly, and the round
// cannot advance past the offending timestamp. The depset's frontier pins — which is the honest
// consequence of DR-RESUME's ruling that there is no supersession mechanism: a P block whose
// cross-chain assumption is false can only be fixed by the prover re-proving from the last valid
// point, and no local action is a substitute for that. The interim posture (P's sequencer consumes
// only already-cross-safe messages) is what keeps this practically unreachable. Recorded as G7G D3.
//
// The shim's halt (G3 D4) is the other end of the same guarantee, and it is unchanged: nothing may
// reach the replacement-block synthesiser for P.
func (c *Container) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32, parentPayload *eth.ExecutionPayloadEnvelope) (bool, error) {
	c.log.Error("PROVEN CHAIN INVALIDATION REFUSED — a proven chain is never replaced, only stopped. "+
		"The cross-safe frontier will pin at this block until its prover re-proves from the last valid "+
		"point (there is no supersession mechanism, by ruling). Investigate the dependency this block "+
		"declared on the wire.",
		"chain", c.ID(), "height", height, "hash", payloadHash, "decisionTimestamp", decisionTimestamp)
	if c.onInvalidationRefused != nil {
		c.onInvalidationRefused(height, payloadHash)
	}
	return false, fmt.Errorf("chain %s: refusing to invalidate proven block %d (%s): a proven chain is "+
		"never replaced, only stopped: %w", c.ID(), height, payloadHash, ErrNoExecutionData)
}

// OptimisticAt is the chain's answer to "which of your blocks is at this timestamp, and which L1
// block made it safe" — the single atomic question the cross-safety round's readiness check asks of
// every chain.
//
// In the VERIFIER posture the embedded container answers it, because this node really does derive
// P and really does have a SafeDB for it.
//
// In the SEQUENCER posture it must be answered from the proven head, and this is hazard-3
// symmetry — the reason this method exists at all. A sequencer supernode fronts P's real execution
// client, which is producing blocks normally, but there is no public derivation behind it: no
// batcher, no DA, nothing that would move a public local-safe label forward. Left to the stock
// path, P's local-safe would sit still forever. And because the readiness check gates the round on
// EVERY chain, a P that never advances freezes the cross-safe frontier for the whole dependency
// set — chain A included, cluster-wide, from a chain that is perfectly healthy. The proven head is
// the label that is actually true of P publicly: it is where P's proofs have reached.
func (c *Container) OptimisticAt(ctx context.Context, ts uint64) (l2, l1 eth.BlockID, err error) {
	if c.labels == LabelsFromDerivation {
		return c.InteropChain.OptimisticAt(ctx, ts)
	}
	fact, ok := c.factAtTimestamp(ctx, ts)
	if !ok {
		// Not proven yet. NotFound rather than an error: the readiness check converts exactly this
		// into "wait", which is the correct response to a proof that has not landed.
		return eth.BlockID{}, eth.BlockID{}, ethereum.NotFound
	}
	inclusion, ok := c.l1InclusionOf(fact)
	if !ok {
		return eth.BlockID{}, eth.BlockID{}, ethereum.NotFound
	}
	return eth.BlockID{Number: fact.Number, Hash: fact.Hash}, inclusion, nil
}

// LocalSafeBlockAtTimestamp is OptimisticAt's L2 half, for the callers that want only that.
func (c *Container) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	if c.labels == LabelsFromDerivation {
		return c.InteropChain.LocalSafeBlockAtTimestamp(ctx, ts)
	}
	fact, ok := c.factAtTimestamp(ctx, ts)
	if !ok {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return eth.L2BlockRef{
		Hash:           fact.Hash,
		Number:         fact.Number,
		Time:           fact.Timestamp,
		L1Origin:       fact.L1Origin,
		SequenceNumber: fact.SeqNumber,
	}, nil
}

// FirstSafeHeadTimestamp is the earliest timestamp this node can verify for the chain. In the
// sequencer posture that is the oldest fact still in the window, not a SafeDB entry — there is no
// SafeDB behind a chain nobody here derives.
func (c *Container) FirstSafeHeadTimestamp(ctx context.Context) (uint64, error) {
	if c.labels == LabelsFromDerivation {
		return c.InteropChain.FirstSafeHeadTimestamp(ctx)
	}
	oldest, ok := c.facts.Oldest()
	if !ok {
		// No proofs read yet. This is the transient cold-start condition the caller backs off on,
		// which is exactly what it means here too.
		return 0, cc.ErrSafeDBNotReady
	}
	return oldest.Timestamp, nil
}

// factAtTimestamp resolves a timestamp to the proven-or-forced block AT OR BEFORE it.
//
// The mapping is rollup-config arithmetic, not a search: P's block time is fixed and its genesis is
// known, so the block number at a timestamp is a division. The fact table then says whether this
// node has that block, and the timestamp is re-checked against the fact rather than assumed — an
// arithmetic answer that disagreed with the proof would be a fabrication of exactly the kind the
// fabrication classes forbid.
//
// AT OR BEFORE, and the difference is not cosmetic. `TimestampToBlockNumber` rounds DOWN (op-node's
// TargetBlockNumber: "we should not request blocks into the future"), and the cross-safety round asks
// about every timestamp, one second at a time — not only the ones a block lands on. On a chain with a
// two-second block time that is half of them. So the re-check must compare the fact against what the
// ARITHMETIC says that block's timestamp is, not against the timestamp that was asked about;
// comparing against the query rejected every between-blocks timestamp, and because the round cannot
// skip a timestamp, the first one it met froze the whole dependency set's frontier permanently.
//
// That is the same failure this posture exists to prevent, arriving from inside the fix for it, and
// it was invisible in unit tests because a test naturally asks about a block's own timestamp. Found
// by running the posture end to end (`TestSequencerPostureUnfreezesTheClusterFrontier`).
//
// The anti-fabrication check is not weakened by the change, it is aimed properly: both the block
// number and its expected timestamp now come from the same rollup config, so a fact whose timestamp
// disagrees with the config's own arithmetic is still refused.
func (c *Container) factAtTimestamp(ctx context.Context, ts uint64) (Fact, bool) {
	num, err := c.InteropChain.TimestampToBlockNumber(ctx, ts)
	if err != nil {
		return Fact{}, false
	}
	want, err := c.InteropChain.BlockNumberToTimestamp(ctx, num)
	if err != nil {
		return Fact{}, false
	}
	fact, ok := c.facts.ByNumber(num)
	if !ok || fact.Timestamp != want {
		return Fact{}, false
	}
	return fact, true
}

// l1InclusionOf answers "at which L1 block did this L2 block become safe" for a silhouette chain.
//
// For a PROVEN block that is the block that carried its proof batch — a real fact, recorded at
// acceptance.
//
// A FORCED block has no carrier: nothing proved it, and reporting one would be an invention (G3
// D8 makes "forced, no carrier" a distinct state precisely so a consumer cannot mistake it for an
// L1 inclusion at block zero). It is still a real block of the public chain, though, and refusing
// to label it would reintroduce the freeze that forced blocks exist to prevent. So it inherits the
// carrier of the nearest proven ancestor.
//
// That is a deliberate UNDERSTATEMENT and the direction matters. This value is consumed as a lower
// bound — the round takes the max across chains and checks L1 canonicality against it — so
// reporting an L1 block that is too old costs finality latency, while reporting one that is too new
// would claim safety L1 has not yet conferred. Understating is the safe error; overstating is not.
func (c *Container) l1InclusionOf(fact Fact) (eth.BlockID, bool) {
	for num := fact.Number; ; num-- {
		if carrier, ok := c.facts.CarrierOf(num); ok {
			return carrier, true
		}
		f, ok := c.facts.ByNumber(num)
		if !ok || num == 0 {
			// Walked off the bottom of the window without finding a proven ancestor. Honest answer:
			// this node cannot say when the block became safe.
			return eth.BlockID{}, false
		}
		if !f.Forced {
			// A proven block with no carrier means the fact table and the carrier table disagree,
			// which is a bug rather than a state. Say so instead of walking past it.
			c.log.Error("proven block has no carrier; refusing to guess its L1 inclusion",
				"chain", c.ID(), "block", num)
			return eth.BlockID{}, false
		}
	}
}
