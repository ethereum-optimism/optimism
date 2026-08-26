package silhouette

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// THE FORCED EXTENSION, THROUGH THE ENGINE (G3 D5).
//
// PLAN.md DR-2 makes the forced extension DESIGNED liveness: when P's L1 origin advances a full
// sequencing window past the last proven block's origin, stock derivation force-generates empty
// blocks so a dead prover can never stall the dependency set's cross-safe frontier. The generator
// sits UPSTREAM of anything a verifier controls — it is the stock batch stage, not our source — so
// the engine does not get to hold it back; it only gets to predict it.
//
// That puts the fail-stop and the liveness guarantee in direct tension at exactly one seam. An engine
// that served only pre-recorded forced facts would stall the chain's public rendering precisely when
// a dead prover is supposed to cost nothing. An engine that described whatever the pipeline asked for
// would have no fail-stop at all. The resolution is that the shim computes the forced block through
// the SHARED convention (the same code the guest is matched against) and refuses anything that does
// not look like a window-expired forced block.
//
// These gates exercise both halves against the real stock generator.

// TestStockNodeForcesBlocksThroughTheShim: the sequencing window expires, the stock pipeline
// force-generates, and the shim renders the forced blocks from the convention.
func TestStockNodeForcesBlocksThroughTheShim(t *testing.T) {
	// L1 far enough past the batch's epoch for the window to have expired.
	e := newTestEnv(t, l1GenesisNum+seqWindow+8).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	v := se.newVerifier(t)
	v.runToQuiescence(t, 800)

	last := batch.Blocks[len(batch.Blocks)-1]
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.Greater(t, head.Number, last.Number,
		"the window has expired, so stock derivation must have forced blocks above the proven head")
	require.True(t, head.Forced, "the head above a proven range is a forced block")

	// The forced facts the ENGINE computed must be exactly the ones the convention predicts. This is
	// the cross-check that matters: ForcedExtension is what a resuming batch is chained against (G2's
	// resumeHead) and what the guest and the superroot program compute, so the engine agreeing with it
	// is what keeps three implementations on one chain.
	provenHead, ok := e.facts.ByNumber(last.Number)
	require.True(t, ok)
	predicted, err := ForcedExtension(context.Background(),
		ForcedParams{Rollup: e.rollup, L1Chain: sepoliaChainConfig(), SysCfg: e.sysCfg},
		e.l1, provenHead, e.l1.head)
	require.NoError(t, err)
	require.NotEmpty(t, predicted)

	for _, want := range predicted {
		got, ok := e.facts.ByNumber(want.Number)
		if !ok {
			break // the pipeline may not have reached the whole predicted extension yet
		}
		require.True(t, got.Forced, "block %d must be marked forced", want.Number)
		require.Equal(t, want.Hash, got.Hash,
			"forced block %d: the engine's hash must be the convention's hash", want.Number)
		require.Equal(t, want.OutputRoot, got.OutputRoot, "forced block %d output root", want.Number)
		require.Equal(t, want.L1Origin, got.L1Origin, "forced block %d rendered origin", want.Number)
		require.Equal(t, want.SeqNumber, got.SeqNumber, "forced block %d sequence number", want.Number)
	}

	// Identity-STF: state does not move through a forced extension, so every settlement value above
	// the proven head carries the PROVEN roots and only the block hash progresses. A shim that had
	// invented a state root here would be inventing a settlement claim.
	ctx := context.Background()
	out, err := se.eng.OutputV0AtBlockNumber(ctx, head.Number)
	require.NoError(t, err)
	require.Equal(t, eth.Bytes32(last.StateRoot), out.StateRoot,
		"a forced block carries the last PROVEN state root: nothing executed")
	require.Equal(t, eth.Bytes32(last.MessagePasserStorageRoot), out.MessagePasserStorageRoot)
	require.Equal(t, head.Hash, out.BlockHash)
	require.Equal(t, head.OutputRoot, common.Hash(eth.OutputRoot(out)))

	// The forced block declares itself as forced and, honestly, as having no proof carrier.
	var decl BlockDeclaration
	require.NoError(t, se.rpc.CallContext(ctx, &decl, "silhouette_blockProvenance",
		rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(head.Number))))
	require.Equal(t, "forced", decl.Provenance)
	require.Nil(t, decl.Carrier, "nothing proved a forced block, and saying so is the honest answer")

	_, halted := se.shim.Halted()
	require.False(t, halted, "designed liveness is not a fabrication")
}

// TestShimRefusesAForcedBlockWhoseWindowHasNotExpired is the other half of D5: the window-expiry
// check is what stops the on-demand forced path from becoming a hole in the fail-stop.
//
// A transcoder bug that produced an empty batch at an ordinary height would arrive at the engine
// looking exactly like a forced block — one L1-info transaction, no facts — and the only thing that
// tells the two apart locally is that a forced block's epoch has been passed by a full sequencing
// window. At an ordinary height the epoch is recent, so epoch+window sits above this node's L1 head
// and the block is refused.
func TestShimRefusesAForcedBlockWhoseWindowHasNotExpired(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4).withRealFeeScalars(t)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	se := e.newShim(t)
	built := se.deriveAndBuild(t, 3)
	require.Len(t, built, 3)
	head := built[len(built)-1]
	ctx := context.Background()

	// Perfectly well-formed attributes for the next block, on a recent epoch.
	attrs := se.attributesFor(t, head, head.L1Origin.Number)
	res, err := se.eng.ForkchoiceUpdate(ctx, &eth.ForkchoiceState{
		HeadBlockHash: head.Hash, SafeBlockHash: head.Hash, FinalizedBlockHash: head.Hash}, attrs)
	require.NoError(t, err)
	require.NotNil(t, res.PayloadID)

	_, err = se.eng.GetPayload(ctx, eth.PayloadInfo{ID: *res.PayloadID, Timestamp: uint64(attrs.Timestamp)})
	require.Error(t, err)
	require.ErrorContains(t, err, "has not yet been passed by a full sequencing window",
		"a block with no fact whose epoch is recent is not a forced block, and must not be rendered as one")

	_, ok := e.facts.ByNumber(head.Number + 1)
	require.False(t, ok, "a refused block must leave no fact behind")
	_, halted := se.shim.Halted()
	require.False(t, halted)
}
