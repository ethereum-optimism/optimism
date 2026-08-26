package silhouette

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ForcedBlockAt computes the ONE forced block the convention puts directly on top of `parent`, given
// the L1 origin and timestamp the stock generator chose for it, and refuses anything that does not
// look like a window-expired forced block.
//
// Why this exists alongside ForcedExtension. ForcedExtension answers "what will the stock pipeline
// force above this head, given how far L1 has advanced" — the question a verifier asks when a
// resuming batch claims to extend a forced head. The shim EL asks a different question, from the
// other side of the same convention: the stock pipeline has ALREADY generated a forced block and is
// asking the engine to build it, and the engine must decide whether that is a block the convention
// sanctions before it will describe one. The pipeline origin, which is the input to the window
// predicate, is not visible at the engine seam — the attributes carry the block's own epoch, not the
// pipeline's position — so the predicate cannot simply be re-evaluated here.
//
// What is checked instead is every locally-checkable consequence of "this is a forced block":
//
//  1. the timestamp is the parent's plus one block time — a forced block extends by exactly one;
//  2. the epoch is canonical on this node's L1 at its own height;
//  3. the epoch advances by 0 or +1 over the parent's, with the sequence number that implies —
//     stock epoch monotonicity, and the L1-info transaction's own sequence-number rule;
//  4. **L1 has advanced at least a full sequencing window past the epoch.** This is a NECESSARY
//     condition for the stock generator to have fired (`deriveNextEmptyBatch` requires
//     `epoch.number + seq_window_size <= pipelineOrigin`, and the pipeline origin is a block this
//     node has), and it is not a sufficient one, because the pipeline's own origin is not readable
//     here. It is the check that matters in practice: at a normal height the epoch is recent, so
//     `epoch + window` sits far above this node's L1 head and a block with no fact fails stop
//     instead of being rendered as forced.
//
// The honest scope of that asymmetry is recorded as G3 D5. The stock pipeline downstream is the
// authority on which blocks the chain gets — whatever it generates is what the chain gets — so the
// engine's job is not to second-guess it but to refuse to invent a block the convention does not
// define, and to compute the one it does define byte-identically to the guest and the superroot
// program.
func ForcedBlockAt(
	ctx context.Context,
	p ForcedParams,
	l1 L1Headers,
	parent Fact,
	origin eth.BlockID,
	seqNumber uint64,
	timestamp uint64,
) (Fact, error) {
	if p.Rollup.BlockTime == 0 {
		return Fact{}, fmt.Errorf("rollup config has a zero block time")
	}
	if want := parent.Timestamp + p.Rollup.BlockTime; timestamp != want {
		return Fact{}, fmt.Errorf("block %d at timestamp %d does not extend block %d (timestamp %d) by "+
			"one block time: a forced block's timestamp is exactly parent + block_time",
			parent.Number+1, timestamp, parent.Number, want)
	}

	// The epoch advances by 0 or +1, and the sequence number follows from which it was. Getting this
	// wrong changes the L1-info transaction, which changes the transactions root, which changes the
	// forced block's hash — so it is checked rather than taken from the attributes.
	switch origin.Number {
	case parent.L1Origin.Number:
		if seqNumber != parent.SeqNumber+1 {
			return Fact{}, fmt.Errorf("block %d holds L1 origin %d but claims sequence number %d; "+
				"holding the epoch means parent sequence number + 1 = %d",
				parent.Number+1, origin.Number, seqNumber, parent.SeqNumber+1)
		}
	case parent.L1Origin.Number + 1:
		if seqNumber != 0 {
			return Fact{}, fmt.Errorf("block %d advances to L1 origin %d but claims sequence number %d; "+
				"the first block of an epoch has sequence number 0", parent.Number+1, origin.Number, seqNumber)
		}
	default:
		return Fact{}, fmt.Errorf("block %d claims L1 origin %d, which is neither the parent's origin %d "+
			"nor the next one: the epoch advances by exactly 0 or 1 per block",
			parent.Number+1, origin.Number, parent.L1Origin.Number)
	}

	epoch, err := l1.L1BlockRefByNumber(ctx, origin.Number)
	if err != nil {
		return Fact{}, fmt.Errorf("fetch claimed L1 origin %d of block %d: %w", origin.Number, parent.Number+1, err)
	}
	if epoch.Hash != origin.Hash {
		return Fact{}, fmt.Errorf("block %d claims L1 origin %s, but this node's canonical L1 block %d is %s",
			parent.Number+1, origin, origin.Number, epoch.Hash)
	}

	// The window-expiry necessary condition. seq_window_size L1 blocks must have happened since the
	// epoch, or the stock generator could not have decided the window had expired for it.
	expiry := origin.Number + p.Rollup.SeqWindowSize
	if _, err := l1.L1BlockRefByNumber(ctx, expiry); err != nil {
		return Fact{}, fmt.Errorf("refusing to render block %d as a forced block: no fact exists for it "+
			"and its L1 origin %d has not yet been passed by a full sequencing window (L1 block %d is not "+
			"known to this node), so the forced-extension convention does not define it: %w",
			parent.Number+1, origin.Number, expiry, err)
	}

	return forcedBlock(ctx, p, l1, parent, epoch, seqNumber, timestamp)
}
