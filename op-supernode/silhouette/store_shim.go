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
// Both tables are windows over the frontier, like the fact table itself, and both are derived: a
// restart re-derives the facts from L1 and re-renders the blocks as the CL replays the build dance.

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
