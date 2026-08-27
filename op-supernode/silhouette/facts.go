package silhouette

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

const (
	// maxIndexedBlocks bounds the fact table. Proven history is unbounded, so the durable table is a
	// window over the recent past rather than an archive:
	// enough for the cross-safety judge, superroot composition and the shim's getPayload, all of
	// which look at the frontier.
	maxIndexedBlocks = 1 << 16

	// maxCarriers bounds the batch-provenance table. It is what the reset walk searches, so it must
	// cover more L1 history than any reorg this node could survive: at one batch per ten minutes it
	// is over a month.
	maxCarriers = 1 << 12
)

// Fact is everything this node knows about one L2 block of a chain it never executes: a table of
// facts standing in for state.
//
// It covers the two durable kinds of block, plus a short-lived replacement handoff:
//
//   - a PROVEN block's roots and hash came off the wire, inside an accepted proof batch;
//
//   - a FORCED block's came from the forced-extension convention (G2 D2/D7), computed identically
//     by this node, the producer and the superroot program.
//
//   - a REPLACEMENT block was executed by P's real EL from stock deposits-only attributes and is
//     retained by the Silhouette EL only until the proof submitter publishes that same block.
type Fact struct {
	Number    uint64
	Timestamp uint64
	// ParentHash is retained explicitly so a remote interop consumer can reconstruct the chain
	// without sharing this process's fact table. It is public block identity, not private body data.
	ParentHash               common.Hash
	Hash                     common.Hash
	StateRoot                common.Hash
	MessagePasserStorageRoot common.Hash
	// OutputRoot is derived rather than carried: it is a function of the other three roots, and
	// deriving it here is what proves a batch's own newOutputRoot is consistent with its last block.
	OutputRoot common.Hash
	// L1Origin is the block's L1 origin. Wire v4 carries it explicitly; legacy v2/v3 batches use the
	// greedy rendered-origin convention.
	L1Origin eth.BlockID
	// SeqNumber is the block's position within its epoch, which its L1-info transaction commits to.
	SeqNumber uint64
	// Forced marks a block the forced-extension convention produced rather than a proof.
	Forced bool
	// Replacement marks the temporary handoff of a stock Holocene replacement payload. Replacement
	// facts live in the shim, not the durable FactStore, and become ordinary proven facts when the
	// proof submitter publishes the same block.
	Replacement bool
	// ExecMsgs are the executing messages this block consumed, in the wire's canonical order, as
	// the cross-safety judge consumes them. This is G7's import list, and it is what makes P's
	// dependencies checkable by the STOCK machinery instead of trusted.
	//
	// It lives here rather than in the LogsDB on purpose. The LogsDB addresses a block's executing
	// messages by the POSITION of the CrossL2Inbox log that carried them, and wire v3 deliberately
	// carries no position (proofbatch.ExecMsg) — so there is no honest index to file them under.
	// The fact table already is "a table of facts standing in for state", and an import list is
	// exactly such a fact.
	//
	// A FORCED block's list is empty, and that is a fact rather than an absence: a forced block has
	// exactly one transaction, the L1-info transaction, so it consumes nothing.
	ExecMsgs []messages.ExecutingMessage
	// ExecMsgsKnown says whether ExecMsgs is a CLAIM or merely UNKNOWN.
	//
	// Empty-and-known means "this block consumed nothing", which is checkable. Empty-and-unknown
	// means the wire version that carried this block said nothing about imports at all (v2), and
	// reading that as "consumed nothing" would be the whole G7 flip failing open — a verifier
	// validating no dependencies while reporting that it validates them. Every consumer must branch
	// on this flag rather than on len(ExecMsgs).
	ExecMsgsKnown bool
	// LogExports are the proof-declared initiating-message hashes and their real block indices.
	// Keeping them beside the imports makes the standalone EL the sole owner of proof-derived
	// interop data; a supernode consumes this over RPC instead of running a second proof observer.
	LogExports []proofbatch.LogExport
	// Header is the full forced header, retained for forced blocks only: a hash disagreement between
	// implementations is diagnosed by diffing headers, not by staring at two hashes. Nil for proven
	// blocks, whose headers are private by construction — we hold their hash, never their bytes.
	Header *types.Header
}

// carrier is one accepted batch's L1 provenance: the L1 block that carried it, the L2 range it
// proved, and the head it moved the chain from and to.
//
// This is the record that turns L1 position into L2 safety, and it is what the rewind searches: an
// accepted batch whose carrier is no longer canonical was never posted, so everything it proved has
// to go.
type carrier struct {
	L1     eth.BlockID
	L1Time uint64
	// FirstBlock..LastBlock is the L2 range the batch proved, contiguous by construction. Forced
	// blocks below FirstBlock that this batch resumed over are NOT part of the range: they were
	// derived, not proved.
	FirstBlock uint64
	LastBlock  uint64
	LastHash   common.Hash
	// ParentHash / PrevOutputRoot describe the head this batch extended, which is the rewind target
	// when it is re-derived. On a resume that head is a FORCED block, not a proven one.
	ParentHash     common.Hash
	PrevOutputRoot common.Hash
	NewOutputRoot  common.Hash
}

// FactStore holds per-block facts, per-batch L1 provenance, and the engine's canonical public
// view. OpenFactStore makes the complete view durable; a zero-value FactStore remains useful for
// focused tests and ephemeral embeddings.
//
// It is written by the derivation loop and read by whoever asks the node a question, so it carries
// its own lock: a table of facts standing in for state is exactly the thing other goroutines want
// to read while derivation is running. G3's shim is one of those readers.
type FactStore struct {
	mu sync.RWMutex
	// blocks is ascending and contiguous, so a block is found by arithmetic rather than by a map.
	// Proven and forced rows interleave freely; contiguity is what matters.
	blocks []Fact
	// byHash indexes the same window by block hash, because superroot composition and the shim ask
	// for facts by HASH and a linear scan of a 65 536-entry window is fine until it is not.
	byHash   map[common.Hash]uint64
	carriers []carrier
	// denied records proof-committed blocks that the cross-safety judge invalidated. A later valid
	// proof may supersede the suffix containing one of these blocks, but the original proof may
	// never become canonical again when the L1 source is replayed.
	denied map[common.Hash]uint64
	// deniedFacts retains the full fact at each denied height after the canonical suffix is
	// truncated. The Silhouette EL needs it exactly once more: to recognize the stock deposits-only build
	// job for that height and delegate it to the private execution engine.
	deniedFacts map[uint64]Fact
	// deniedChecker reads the chain container's durable denylist as the authoritative decision source.
	deniedChecker func(uint64, common.Hash) (bool, error)
	// renderings and cursors are the shim EL's half of the store — see store_shim.go. They live
	// under the same lock because a rewind has to move the facts and forget the renderings above
	// them together, or a query lands between the two and gets an answer from a chain that no
	// longer exists.
	renderings map[common.Hash]Rendering
	cursors    Cursors
	// replacements and rewindFacts are engine-visible canonicality handoffs. Keeping them in the
	// store, rather than in Shim, makes an EL restart indistinguishable from a process pause.
	replacementsByHash map[common.Hash]Fact
	replacementsByNum  map[uint64]Fact
	rewindFacts        map[common.Hash]Fact
	tracker            trackerState
	persist            *persistentStore
}

// SetDeniedChecker attaches the durable denial lookup after the chain container is constructed.
func (f *FactStore) SetDeniedChecker(checker func(uint64, common.Hash) (bool, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deniedChecker = checker
}

// MarkDenied records the cross-safety verdict for a proof-committed block. It is idempotent so a
// verifier may encounter the same invalid timestamp again while waiting for the producer's
// replacement proof.
func (f *FactStore) MarkDenied(number uint64, hash common.Hash) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if deniedAt, ok := f.denied[hash]; ok {
		if deniedAt != number {
			return fmt.Errorf("denied block %s was recorded at height %d, not %d", hash, deniedAt, number)
		}
		return nil
	}
	fact, ok := f.byNumberLocked(number)
	if !ok {
		return fmt.Errorf("cannot deny block %d (%s): no fact at that height", number, hash)
	}
	if fact.Hash != hash {
		return fmt.Errorf("cannot deny block %d (%s): canonical proof fact is %s", number, hash, fact.Hash)
	}
	if f.denied == nil {
		f.denied = make(map[common.Hash]uint64)
	}
	if f.deniedFacts == nil {
		f.deniedFacts = make(map[uint64]Fact)
	}
	f.denied[hash] = number
	f.deniedFacts[number] = fact
	return nil
}

// DeniedFact returns the proof-committed fact retained for replacement construction.
func (f *FactStore) DeniedFact(number uint64) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	fact, ok := f.deniedFacts[number]
	return fact, ok
}

// IsDenied reports whether a block hash has received an invalid cross-safety verdict.
func (f *FactStore) IsDenied(hash common.Hash) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.denied[hash]
	return ok
}

// Denied reports whether the exact block was denied, consulting the chain container's durable
// denylist when the local mirror has not yet observed the verdict.
func (f *FactStore) Denied(number uint64, hash common.Hash) (bool, error) {
	f.mu.RLock()
	deniedAt, denied := f.denied[hash]
	checker := f.deniedChecker
	f.mu.RUnlock()
	if denied {
		if deniedAt != number {
			return false, fmt.Errorf("denied block %s was recorded at height %d, not %d", hash, deniedAt, number)
		}
		return true, nil
	}
	if checker == nil {
		return false, nil
	}
	return checker(number, hash)
}

// SupersessionBase resolves an output root to an ancestor and requires its immediate child to be
// denied. This authorizes the exact suffix the stock replacement path rewrote, and no earlier one.
func (f *FactStore) SupersessionBase(outputRoot common.Hash) (Fact, bool, error) {
	f.mu.RLock()
	var base Fact
	found := false
	for _, fact := range f.blocks {
		if fact.OutputRoot == outputRoot {
			base, found = fact, true
			break
		}
	}
	if !found {
		f.mu.RUnlock()
		return Fact{}, false, nil
	}
	replacement, ok := f.byNumberLocked(base.Number + 1)
	if !ok {
		f.mu.RUnlock()
		return Fact{}, false, nil
	}
	_, denied := f.denied[replacement.Hash]
	checker := f.deniedChecker
	f.mu.RUnlock()
	if denied {
		return base, true, nil
	}
	if checker == nil {
		return Fact{}, false, nil
	}
	denied, err := checker(replacement.Number, replacement.Hash)
	if err != nil || !denied {
		return Fact{}, false, err
	}
	if err := f.MarkDenied(replacement.Number, replacement.Hash); err != nil {
		return Fact{}, false, err
	}
	return base, true, nil
}

// AnchorSupersession reports whether the first fact above the configured anchor was durably denied.
func (f *FactStore) AnchorSupersession(anchor Fact) (bool, error) {
	f.mu.RLock()
	replacement, ok := f.byNumberLocked(anchor.Number + 1)
	if !ok {
		f.mu.RUnlock()
		return false, nil
	}
	_, denied := f.denied[replacement.Hash]
	checker := f.deniedChecker
	f.mu.RUnlock()
	if denied {
		return true, nil
	}
	if checker == nil {
		return false, nil
	}
	denied, err := checker(replacement.Number, replacement.Hash)
	if err != nil || !denied {
		return false, err
	}
	if err := f.MarkDenied(replacement.Number, replacement.Hash); err != nil {
		return false, err
	}
	return true, nil
}

// PruneDenied mirrors durable denylist pruning after the L1 decision basis reorgs away.
func (f *FactStore) PruneDenied(removed map[uint64][]common.Hash) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for number, hashes := range removed {
		for _, hash := range hashes {
			if f.denied[hash] == number {
				delete(f.denied, hash)
			}
		}
	}
}

// ReplaceSuffix drops proof facts and carrier ranges above base. The replacement proof is recorded
// immediately afterwards by the caller.
func (f *FactStore) ReplaceSuffix(base Fact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	got, ok := f.byNumberLocked(base.Number)
	if !ok || got.Hash != base.Hash || got.OutputRoot != base.OutputRoot {
		return fmt.Errorf("supersession base block %d (%s) is no longer canonical", base.Number, base.Hash)
	}

	keptCarriers := 0
	for keptCarriers < len(f.carriers) && f.carriers[keptCarriers].LastBlock <= base.Number {
		keptCarriers++
	}
	if keptCarriers < len(f.carriers) && f.carriers[keptCarriers].FirstBlock <= base.Number {
		c := f.carriers[keptCarriers]
		c.LastBlock = base.Number
		c.LastHash = base.Hash
		c.NewOutputRoot = base.OutputRoot
		f.carriers[keptCarriers] = c
		keptCarriers++
	}
	f.carriers = f.carriers[:keptCarriers]
	f.truncateBlocksLocked(base.Number, true)
	return nil
}

func (f *FactStore) replaceAllForSupersession() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.carriers = nil
	f.truncateBlocksLocked(0, false)
}

// Head is the highest block the table holds — the tip of proven-or-forced history.
func (f *FactStore) Head() (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.blocks) == 0 {
		return Fact{}, false
	}
	return f.blocks[len(f.blocks)-1], true
}

// ByNumber returns a block's facts, if it is still inside the window.
func (f *FactStore) ByNumber(number uint64) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.byNumberLocked(number)
}

func (f *FactStore) byNumberLocked(number uint64) (Fact, bool) {
	if len(f.blocks) == 0 || number < f.blocks[0].Number || number > f.blocks[len(f.blocks)-1].Number {
		return Fact{}, false
	}
	return f.blocks[number-f.blocks[0].Number], true
}

// ByHash returns a block's facts by its real L2 hash.
func (f *FactStore) ByHash(hash common.Hash) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	number, ok := f.byHash[hash]
	if !ok {
		return Fact{}, false
	}
	return f.byNumberLocked(number)
}

// Oldest is the lowest block still inside the window. Below it this node's answer is "not here any
// more", which is a different statement from "not proven" and must never be conflated with it — the
// shim's guard has to distinguish them to avoid failing stop on a merely-trimmed block.
func (f *FactStore) Oldest() (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.blocks) == 0 {
		return Fact{}, false
	}
	return f.blocks[0], true
}

// Record appends one block's facts. Blocks arrive contiguous and in order — the batch's own
// structure check and the head check together guarantee it for proven blocks, and the convention
// generates forced ones in sequence — which is what lets the table be a slice addressed by
// arithmetic.
func (f *FactStore) Record(fact Fact) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byHash == nil {
		f.byHash = make(map[common.Hash]uint64)
	}
	f.blocks = append(f.blocks, fact)
	f.byHash[fact.Hash] = fact.Number
	if excess := len(f.blocks) - maxIndexedBlocks; excess > 0 {
		for _, dropped := range f.blocks[:excess] {
			delete(f.byHash, dropped.Hash)
		}
		f.blocks = append([]Fact(nil), f.blocks[excess:]...)
	}
}

// RecordProven appends a proven block's facts, with the rendered origin the transcoder assigned it
// and the import list the wire declared for it.
//
// wireVersion is passed rather than inferred, because whether an empty import list is a claim is a
// property of the VERSION, not of the list (see Fact.ExecMsgsKnown).
func (f *FactStore) RecordProven(blk proofbatch.BlockExport, parentHash common.Hash, origin eth.BlockID, seqNumber uint64, wireVersion uint8) {
	// Not `wireVersion >= proofbatch.Version`: whether a version carries an import list is a fixed
	// fact about version 3, not about whatever this codec currently encodes (see proofbatch.VersionV3).
	known := proofbatch.VersionHasExecMsgs(wireVersion)
	var execMsgs []messages.ExecutingMessage
	if known {
		// Materialised at acceptance rather than derived on every read: the checksum is a keccak per
		// message, and the judge asks the same question once per round per timestamp.
		execMsgs = make([]messages.ExecutingMessage, len(blk.ExecMsgs))
		for i := range blk.ExecMsgs {
			execMsgs[i] = *blk.ExecMsgs[i].Executing()
		}
	}
	f.Record(Fact{
		Number:                   blk.Number,
		Timestamp:                blk.Timestamp,
		ParentHash:               parentHash,
		Hash:                     blk.Hash,
		ExecMsgs:                 execMsgs,
		ExecMsgsKnown:            known,
		StateRoot:                blk.StateRoot,
		MessagePasserStorageRoot: blk.MessagePasserStorageRoot,
		OutputRoot:               blk.OutputRoot(),
		L1Origin:                 origin,
		SeqNumber:                seqNumber,
		LogExports:               cloneLogExports(blk.Logs),
	})
}

func cloneLogExports(in []proofbatch.LogExport) []proofbatch.LogExport {
	out := make([]proofbatch.LogExport, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Preimage = append([]byte(nil), in[i].Preimage...)
	}
	return out
}

func (f *FactStore) recordCarrier(c carrier) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.carriers = append(f.carriers, c)
	if excess := len(f.carriers) - maxCarriers; excess > 0 {
		f.carriers = append([]carrier(nil), f.carriers[excess:]...)
	}
}

// carrierSnapshot is the batch-provenance table as a walker sees it.
func (f *FactStore) carrierSnapshot() []carrier {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]carrier(nil), f.carriers...)
}

// CarrierOf returns the L1 block that carried the proof batch which proved a given L2 block.
//
// This is the answer to "at which L1 block did this L2 block become safe" for a chain nobody here
// derives. A forced block has no carrier — nothing proved it — and reports false, which is the
// honest answer and the one the safety ladder needs.
func (f *FactStore) CarrierOf(number uint64) (eth.BlockID, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	// Carriers are ascending and their ranges are disjoint, so a binary search on LastBlock lands on
	// the only batch that could contain the block.
	lo, hi := 0, len(f.carriers)
	for lo < hi {
		mid := (lo + hi) / 2
		if f.carriers[mid].LastBlock < number {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == len(f.carriers) || f.carriers[lo].FirstBlock > number {
		return eth.BlockID{}, false
	}
	return f.carriers[lo].L1, true
}

// rewindToL1Below drops everything derived from L1 block `l1Number` and above, and reports the head
// that survives.
//
// This is the whole of the chaining-state ↔ stock-reset mapping (G2 D5). `OpenData(ref)` means
// "about to read L1 block ref", so the state that must be true before reading it is exactly "every
// L1 block below ref.Number has been processed". Forward progress makes this a no-op; a stock
// pipeline reset makes it the correct rewind. There is no separate reset protocol and no reset hook
// on the data-source interface, which is why acceptance stays a pure function of L1 — the same
// property docs/SPEC-WIRE-V4.md already demands when it measures l1Head depth against the
// carrying block rather than the live head.
//
// Forced blocks are dropped with the batch they sat on top of: they are a function of the proven
// head and the L1 chain, so re-deriving that batch re-derives them.
func (f *FactStore) rewindToL1Below(l1Number uint64) (dropped int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	keptCarriers := 0
	for keptCarriers < len(f.carriers) && f.carriers[keptCarriers].L1.Number < l1Number {
		keptCarriers++
	}
	dropped = len(f.carriers) - keptCarriers
	if dropped == 0 {
		return 0
	}
	// Everything above the surviving carrier's range goes, forced blocks included.
	var keepThrough uint64
	if keptCarriers > 0 {
		keepThrough = f.carriers[keptCarriers-1].LastBlock
	}
	f.carriers = f.carriers[:keptCarriers]
	f.truncateBlocksLocked(keepThrough, keptCarriers > 0)
	return dropped
}

// truncateBlocksLocked drops every block above `through`. When there is no surviving carrier at all
// the whole table goes: proven history starts over at the configured anchor.
func (f *FactStore) truncateBlocksLocked(through uint64, anySurvivor bool) {
	kept := 0
	if anySurvivor {
		for kept < len(f.blocks) && f.blocks[kept].Number <= through {
			kept++
		}
	}
	for _, d := range f.blocks[kept:] {
		delete(f.byHash, d.Hash)
	}
	f.blocks = f.blocks[:kept]
	if anySurvivor {
		f.dropRenderingsAbove(through)
	} else {
		f.renderings = nil
	}
}

// dropOrphanedCarriers is the belt-and-braces canonicality walk, ported from Cove's E2 reset walk.
//
// The rewind above is sound on its own whenever an L1 reorg is shallower than the stock reset
// lookback, which is stock op-node's own assumption (G2 F1). This walk removes the reliance on that
// assumption for the carriers still in the window: it asks L1 about each surviving carrier
// newest-first and drops everything above the deepest one that is still canonical. It runs only when
// a rewind actually dropped something, so the hot path pays nothing.
func (f *FactStore) dropOrphanedCarriers(canonical func(c carrier) (bool, error)) (dropped int, err error) {
	snapshot := f.carrierSnapshot()
	survivor := -1
	for i := len(snapshot) - 1; i >= 0; i-- {
		ok, err := canonical(snapshot[i])
		if err != nil {
			return 0, err
		}
		if ok {
			survivor = i
			break
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	keep := survivor + 1
	if keep >= len(f.carriers) {
		return 0, nil
	}
	dropped = len(f.carriers) - keep
	var keepThrough uint64
	if keep > 0 {
		keepThrough = f.carriers[keep-1].LastBlock
	}
	f.carriers = f.carriers[:keep]
	f.truncateBlocksLocked(keepThrough, keep > 0)
	return dropped, nil
}
