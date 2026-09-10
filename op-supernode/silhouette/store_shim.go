package silhouette

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The shim EL's half of the fact store: the public rendering of each block, and the label cursors.
//
// This lives with the fact store rather than inside the shim because the store is this node's whole
// answer to "what do you know about chain P", and after G3 it knows two more things: the exact bytes
// it served for a block, and where the safety labels stand. G4's sequencer-side label source reads
// the cursors; the superroot composition reads the renderings.
//
// Both tables are windows over the frontier, like the fact table itself, and are persisted by a
// standalone EL so restart preserves the exact public chain view and safety labels.

// Rendering is the public rendering of one L2 block: the header this node serves for it and the
// transaction bytes the CL supplied when the block was built.
//
// Hash is kept alongside the header because for a PROVEN block the two disagree by design: Hash is
// the hash the proof committed to, and the header is a rendering whose interior nobody executed. The
// disagreement is DR-1's declared fabrication, and nothing on the verifier's stock path re-hashes a
// header (trustCache, engine_client.go:21-26). For a FORCED block they agree, because the convention
// defines the block.
type Rendering struct {
	Header *types.Header
	Txs    [][]byte
	Hash   common.Hash
}

// Cursors are the three engine labels, as the shim's forkchoiceUpdated moves them.
//
// This IS the safety ladder: the stock Finalizer and the cross-safety judge drive safe and finalized
// down through ordinary forkchoice calls, so the shim never computes a label — it records where the
// stock machinery put one.
type Cursors struct {
	Unsafe    eth.L2BlockRef
	Safe      eth.L2BlockRef
	Finalized eth.L2BlockRef
}

// RecordRendering stores the header and body served for a block.
func (f *FactStore) RecordRendering(r Rendering) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.renderings == nil {
		f.renderings = make(map[common.Hash]Rendering)
	}
	f.renderings[r.Hash] = r
}

// Rendering returns the stored rendering of a block, if this process built it.
//
// A miss is ordinary, not an error: the shim can re-derive a proven block's body from configuration
// (RenderedBody), and a restarted verifier has facts long before it has re-rendered anything.
func (f *FactStore) Rendering(hash common.Hash) (Rendering, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	r, ok := f.renderings[hash]
	return r, ok
}

// DeleteRendering removes a temporary rendering, such as the stock rewind sentinel.
func (f *FactStore) DeleteRendering(hash common.Hash) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.renderings, hash)
}

// SetCursors records where forkchoice put the three labels.
func (f *FactStore) SetCursors(c Cursors) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cursors = c
}

// Cursors reports the three labels.
func (f *FactStore) Cursors() Cursors {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cursors
}

// ClampCursorsTo moves labels that named discarded history back to the surviving proven head.
//
// The stock CL normally performs this move with forkchoiceUpdated after its reset walk. There is a
// small but important ordering constraint here, though: that walk first asks the EL for its current
// safe/finalized labels. An L1-driven fact rewind has already made an orphaned label unservable by
// then, so leaving it in the cursor table traps the CL in a reset loop before it can send the
// correcting forkchoice update. Clamping only labels whose hash is no longer canonical gives that
// stock walk a valid starting point; labels that still name surviving history are left untouched.
func (f *FactStore) ClampCursorsTo(head Fact) {
	f.mu.Lock()
	defer f.mu.Unlock()

	headRef := eth.L2BlockRef{
		Hash:           head.Hash,
		Number:         head.Number,
		ParentHash:     head.ParentHash,
		Time:           head.Timestamp,
		L1Origin:       head.L1Origin,
		SequenceNumber: head.SeqNumber,
	}
	canonical := func(ref eth.L2BlockRef) bool {
		if ref == (eth.L2BlockRef{}) {
			return true
		}
		if ref.Hash == head.Hash && ref.Number == head.Number {
			return true // the configured anchor is intentionally not in byHash
		}
		number, ok := f.byHash[ref.Hash]
		return ok && number == ref.Number && number <= head.Number
	}
	if !canonical(f.cursors.Unsafe) {
		f.cursors.Unsafe = headRef
	}
	if !canonical(f.cursors.Safe) {
		f.cursors.Safe = headRef
	}
	if !canonical(f.cursors.Finalized) {
		f.cursors.Finalized = headRef
	}
}

// dropRenderingsAbove forgets renderings of blocks above `number`, and is called from the two places
// history can retreat: an L1-driven rewind of the facts, and a forkchoice update to an older head.
//
// Forgetting matters for re-derivation identity rather than for memory: a rendering held over a
// reorg would let a stale body answer a query about a height whose facts have changed, and the whole
// point of the store is that every answer is traceable to a proof or to the convention.
func (f *FactStore) dropRenderingsAbove(number uint64) {
	for hash, r := range f.renderings {
		if bigs.Uint64Strict(r.Header.Number) > number {
			delete(f.renderings, hash)
		}
	}
}

// DropRenderingsAbove is dropRenderingsAbove for callers outside the store's own rewind path — the
// shim's guarded forkchoice rewind.
func (f *FactStore) DropRenderingsAbove(number uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dropRenderingsAbove(number)
}

// RecordReplacement retains a real deposits-only block until the corrected proof makes it an
// ordinary canonical fact. It is engine-visible state and is therefore part of the durable store.
func (f *FactStore) RecordReplacement(fact Fact) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.replacementsByHash == nil {
		f.replacementsByHash = make(map[common.Hash]Fact)
	}
	f.replacementsByHash[fact.Hash] = fact
	if f.replacementsByNum == nil {
		f.replacementsByNum = make(map[uint64]Fact)
	}
	f.replacementsByNum[fact.Number] = fact
}

func (f *FactStore) ReplacementByNumber(number uint64) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	fact, ok := f.replacementsByNum[number]
	return fact, ok
}

func (f *FactStore) ReplacementByHash(hash common.Hash) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	replacement, ok := f.replacementsByHash[hash]
	if !ok {
		return Fact{}, false
	}
	// A corrected proof may have occupied this height while a caller retained the temporary hash.
	// Once that happens only the canonical proof fact is visible.
	if canonical, exists := f.byNumberLocked(replacement.Number); exists && canonical.Hash != hash {
		return Fact{}, false
	}
	return replacement, true
}

func (f *FactStore) RecordRewindFact(fact Fact) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rewindFacts == nil {
		f.rewindFacts = make(map[common.Hash]Fact)
	}
	f.rewindFacts[fact.Hash] = fact
}

func (f *FactStore) RewindFact(hash common.Hash) (Fact, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	fact, ok := f.rewindFacts[hash]
	return fact, ok
}

func (f *FactStore) IsRewindFact(hash common.Hash) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.rewindFacts[hash]
	return ok
}

func (f *FactStore) ClearRewindFacts() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for hash := range f.rewindFacts {
		delete(f.renderings, hash)
	}
	f.rewindFacts = nil
}
