package silhouette

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// The acceptance rules, the reject-and-log discipline, the transcode, and the reset rewind.
//
// The transcode assertions deliberately read the emitted payload back with STOCK derivation code —
// ParseFrames, Channel, BatchReader — rather than with a local decoder. The whole architectural claim
// is that this source's output is ordinary channel frames; a bespoke decoder in the test would be
// able to agree with a source that produced something only it understood.

// goodBatch is a batch that should be accepted: extends the anchor, three blocks, sane L1 head, and
// a carrier comfortably above the blocks' own timestamps.
func (e *testEnv) goodBatch() batchSpec {
	return batchSpec{
		prevRoot:   e.cfg.Anchor.OutputRoot,
		firstBlock: 1,
		firstTime:  l1GenesisT + l2BlockTime,
		count:      3,
		l1Head:     l1GenesisNum + 2,
		carrier:    l1GenesisNum + 4,
	}
}

func (e *testEnv) plantSpec(s batchSpec) {
	e.plant(e.buildBatch(s), s)
}

// decodeBatches reads an emitted payload back through stock derivation: frame parsing, channel
// assembly and batch decoding, exactly as FrameQueue -> ChannelMux -> ChannelInReader would.
func decodeBatches(t *testing.T, cfg *rollup.Config, payload []byte, l1 eth.L1BlockRef) []*derive.SingularBatch {
	t.Helper()
	frames, err := derive.ParseFrames(payload)
	require.NoError(t, err, "the emitted payload must be stock channel frames")
	require.NotEmpty(t, frames)

	ch := derive.NewChannel(frames[0].ID, l1, false)
	for _, f := range frames {
		require.NoError(t, ch.AddFrame(f, l1))
	}
	require.True(t, ch.IsReady(), "the emitted frames must form one complete channel")

	next, err := derive.BatchReader(ch.Reader(), rollup.NewChainSpec(cfg).MaxRLPBytesPerChannel(l1.Time), true)
	require.NoError(t, err)
	var out []*derive.SingularBatch
	for {
		data, err := next()
		if err != nil {
			break
		}
		require.Equal(t, derive.SingularBatchType, int(data.GetBatchType()))
		sb, err := derive.GetSingularBatch(data)
		require.NoError(t, err)
		out = append(out, sb)
	}
	return out
}

// TestAcceptAndTranscode is the core happy path: an accepted batch becomes stock channel frames that
// decode back into one empty singular batch per proven block, with the REAL wire hashes as parent
// hashes.
func TestAcceptAndTranscode(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	payloads := e.derive(spec.carrier)
	require.Len(t, payloads, 1, "one accepted batch must emit exactly one payload")

	batches := decodeBatches(t, e.rollup, payloads[0], e.l1.ref(spec.carrier))
	require.Len(t, batches, 3, "one singular batch per proven block")

	// The load-bearing assertion: each batch's parent hash is the previous PROVEN block's real hash
	// off the wire, chaining from the anchor. This is what binds the rendered chain to the
	// proof-committed hashes through the stock batch queue's own parent check.
	wantParent := e.cfg.Anchor.BlockHash
	for i, b := range batches {
		require.Equal(t, wantParent, b.ParentHash, "batch %d parent hash", i)
		require.Equal(t, batch.Blocks[i].Timestamp, b.Timestamp, "batch %d timestamp", i)
		require.Empty(t, b.Transactions,
			"a rendered block's batch carries no transactions: the interior is private and the "+
				"stock attributes builder supplies the L1-info deposit")
		wantParent = batch.Blocks[i].Hash
	}

	// The facts are recorded for exactly the proven blocks, with real roots.
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.Equal(t, uint64(3), head.Number)
	require.Equal(t, batch.Blocks[2].Hash, head.Hash)
	require.Equal(t, batch.NewOutputRoot, head.OutputRoot)
	require.False(t, head.Forced)
	carrier, ok := e.facts.CarrierOf(2)
	require.True(t, ok, "a proven block must know which L1 block made it safe")
	require.Equal(t, spec.carrier, carrier.Number)
}

// TestRenderedOriginIsTheStockEpoch is the coordinator's round-trip rule: the epoch this source
// renders must be the epoch the stock CL reads back out of the L1-info transaction bytes. The batch's
// EpochNum/EpochHash come from PayloadToSingularBatch parsing those bytes, so asserting them against
// the recorded facts closes the loop through the stock parser rather than around it.
func TestRenderedOriginIsTheStockEpoch(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+4)
	spec := e.goodBatch()
	batch := e.buildBatch(spec)
	e.plant(batch, spec)

	payloads := e.derive(spec.carrier)
	require.Len(t, payloads, 1)
	batches := decodeBatches(t, e.rollup, payloads[0], e.l1.ref(spec.carrier))
	require.Len(t, batches, 3)

	for i, b := range batches {
		fact, ok := e.facts.ByNumber(batch.Blocks[i].Number)
		require.True(t, ok)
		require.Equal(t, fact.L1Origin.Number, uint64(b.EpochNum),
			"block %d: the stock parser must read back the origin this source rendered", i)
		require.Equal(t, fact.L1Origin.Hash, b.EpochHash, "block %d epoch hash", i)
		// The greedy rule never points an L2 block at a future L1 block.
		require.LessOrEqual(t, l1Time(uint64(b.EpochNum)), b.Timestamp, "block %d origin is in the future", i)
	}
}

// TestRejectAndLog is gate 3: every way a batch can be bad ends in "skipped, logged, keep deriving",
// never in a halt and never in a partial advance of the chaining head. A verifier that a malformed
// submitter transaction can stop is a verifier an adversary can stop.
func TestRejectAndLog(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(e *testEnv, s *batchSpec)
	}{
		{
			name: "proof bytes in attested mode",
			// Attested mode is "no proof", never "any proof": a batch that arrives carrying proof bytes
			// is REFUSED rather than waved through, so nobody can smuggle an unverifiable batch past a
			// node by dressing it up as one that was verified. See AttestedVerifier.
			prepare: func(e *testEnv, s *batchSpec) { s.proof = []byte{0xde, 0xad, 0xbe, 0xef} },
		},
		{
			name: "wrong rollup config hash",
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) { b.RollupConfigHash = common.HexToHash("0xbad") }
			},
		},
		{
			name: "wrong dependency set hash",
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) { b.DepSetHash = common.HexToHash("0xbad") }
			},
		},
		{
			name: "wrong export policy",
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) { b.ExportPolicyHash = common.HexToHash("0xbad") }
			},
		},
		{
			name:    "does not extend the chaining head",
			prepare: func(e *testEnv, s *batchSpec) { s.prevRoot = common.HexToHash("0xnotthehead") },
		},
		{
			name: "claims an output root its own blocks do not derive",
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) { b.NewOutputRoot = common.HexToHash("0xbad") }
			},
		},
		{
			name: "l1Head is not canonical",
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) { b.L1Head = common.HexToHash("0xdeadbeef") }
			},
		},
		{
			name: "l1Head is above the carrying block",
			// Acceptance must be replayable: an l1Head the carrier could not have known about would
			// make the verdict depend on when the batch was read.
			prepare: func(e *testEnv, s *batchSpec) { s.l1Head = s.carrier + 1 },
		},
		{
			name: "timestamps are not one block time apart",
			// G2 D6: the whole batch stands or falls together. CheckStructure only requires strict
			// increase, and letting a mistimed batch through would advance the chaining head past
			// blocks the stock batch queue silently drops downstream.
			prepare: func(e *testEnv, s *batchSpec) {
				s.mutate = func(b *proofbatch.ProofBatch) {
					b.Blocks[1].Timestamp += 1
					b.Blocks[2].Timestamp += 1
					b.NewOutputRoot = b.Blocks[2].OutputRoot()
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t, l1GenesisNum+4)
			spec := e.goodBatch()
			tc.prepare(e, &spec)
			if spec.l1Head > e.l1.head {
				e.l1.head = spec.l1Head
			}
			e.plantSpec(spec)

			payloads := e.derive(spec.carrier)
			require.Empty(t, payloads, "a rejected batch must emit nothing")
			_, ok := e.facts.Head()
			require.False(t, ok, "a rejected batch must not advance the chaining head")
		})
	}
}

// TestRejectedBatchDoesNotBlockTheNextOne is the other half of reject-and-log: derivation keeps
// going. A bad batch in one L1 block must not poison the good batch in the next.
func TestRejectedBatchDoesNotBlockTheNextOne(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)

	bad := e.goodBatch()
	bad.mutate = func(b *proofbatch.ProofBatch) { b.RollupConfigHash = common.HexToHash("0xbad") }
	e.plantSpec(bad)
	require.Empty(t, e.derive(bad.carrier))

	good := e.goodBatch()
	good.carrier = l1GenesisNum + 6
	e.plantSpec(good)
	require.Len(t, e.derive(good.carrier), 1, "derivation must continue past a rejected batch")
	head, ok := e.facts.Head()
	require.True(t, ok)
	require.Equal(t, uint64(3), head.Number)
}

// TestNonSubmitterAndNonInboxAreIgnored is acceptance rule 1. The from/to pair is the whole
// authenticity rule for the envelope stream, so anyone can put a well-formed envelope on L1 and it
// must be invisible.
func TestNonSubmitterAndNonInboxAreIgnored(t *testing.T) {
	t.Run("wrong sender", func(t *testing.T) {
		e := newTestEnv(t, l1GenesisNum+4)
		spec := e.goodBatch()
		batch := e.buildBatch(spec)
		raw, err := proofbatch.Encode(batch, nil)
		require.NoError(t, err)
		blobs, err := proofbatch.ToBlobs(raw)
		require.NoError(t, err)
		var hashes []common.Hash
		for _, blob := range blobs {
			commit, err := blob.ComputeKZGCommitment()
			require.NoError(t, err)
			h := eth.KZGToVersionedHash(commit)
			e.blobs.byHash[h] = blob
			hashes = append(hashes, h)
		}
		e.l1.txsByBlock[spec.carrier] = append(e.l1.txsByBlock[spec.carrier], otherTx(t, e.cfg.Inbox, hashes))
		require.Empty(t, e.derive(spec.carrier))
	})

	t.Run("wrong inbox", func(t *testing.T) {
		e := newTestEnv(t, l1GenesisNum+4)
		spec := e.goodBatch()
		batch := e.buildBatch(spec)
		raw, err := proofbatch.Encode(batch, nil)
		require.NoError(t, err)
		blobs, err := proofbatch.ToBlobs(raw)
		require.NoError(t, err)
		var hashes []common.Hash
		for _, blob := range blobs {
			commit, err := blob.ComputeKZGCommitment()
			require.NoError(t, err)
			h := eth.KZGToVersionedHash(commit)
			e.blobs.byHash[h] = blob
			hashes = append(hashes, h)
		}
		e.l1.txsByBlock[spec.carrier] = append(e.l1.txsByBlock[spec.carrier],
			e.key.blobTx(t, common.HexToAddress("0xc0ffee"), hashes))
		require.Empty(t, e.derive(spec.carrier))
	})
}

// TestDuplicateBatchIsANoOp: a replayed envelope fails the chaining rule, which is the same rule that
// rejects a forgery. Nothing special is needed to make replay safe.
func TestDuplicateBatchIsANoOp(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)
	spec := e.goodBatch()
	e.plantSpec(spec)
	require.Len(t, e.derive(spec.carrier), 1)
	head, _ := e.facts.Head()

	dup := spec
	dup.carrier = l1GenesisNum + 6
	e.plantSpec(dup)
	require.Empty(t, e.derive(dup.carrier), "a replayed batch must be rejected")
	again, _ := e.facts.Head()
	require.Equal(t, head.Number, again.Number, "the head must not move on a replay")
}

// TestResetRewindsToTheL1Cursor is gate 4's first half (G2 D5): re-opening an L1 block at or below a
// carrier rewinds the chaining state so that block is re-derived, and re-deriving it reproduces
// exactly the same facts.
func TestResetRewindsToTheL1Cursor(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)
	spec := e.goodBatch()
	e.plantSpec(spec)

	first := e.derive(spec.carrier)
	require.Len(t, first, 1)
	head, _ := e.facts.Head()
	require.Equal(t, uint64(3), head.Number)

	// A stock reset points the pipeline back at an earlier L1 block. Opening it must take the batch
	// back, because the pipeline is about to read it again.
	_, err := e.src.OpenData(context.Background(), e.l1.ref(l1GenesisNum+1), common.Address{})
	require.NoError(t, err)
	_, ok := e.facts.Head()
	require.False(t, ok, "the reset must have taken the batch back")

	// Re-deriving the same L1 block must produce byte-identical output: acceptance is a pure function
	// of L1, so a replay cannot reach a different verdict.
	second := e.derive(spec.carrier)
	require.Len(t, second, 1)
	require.Equal(t, first[0], second[0], "re-derivation must be byte-identical")
	again, _ := e.facts.Head()
	require.Equal(t, head, again, "re-derivation must reproduce the same facts")
}

// TestL1ReorgDropsOrphanedCarriers is gate 4's second half: a reorg at the carrier drops the batch it
// carried, and the blocks it proved stop being referenceable.
func TestL1ReorgDropsOrphanedCarriers(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)
	spec := e.goodBatch()
	e.plantSpec(spec)
	require.Len(t, e.derive(spec.carrier), 1)
	require.Equal(t, 1, len(e.facts.carrierSnapshot()))

	// L1 reorgs below the carrier: the block that carried the batch is no longer canonical, so the
	// batch was never posted and everything it proved has to go.
	e.l1.reorgAbove = l1GenesisNum + 3
	_, err := e.src.OpenData(context.Background(), e.l1.ref(l1GenesisNum+5), common.Address{})
	require.NoError(t, err)

	require.Empty(t, e.facts.carrierSnapshot(), "an orphaned carrier must be dropped")
	_, ok := e.facts.Head()
	require.False(t, ok, "blocks proved by a dropped batch must stop being referenceable")
	_, ok = e.facts.ByNumber(2)
	require.False(t, ok)
}

// TestForwardProgressDoesNotRewind: the rewind is a no-op on the hot path. If ordinary traversal
// dropped carriers, every L1 block would re-derive the whole window.
func TestForwardProgressDoesNotRewind(t *testing.T) {
	e := newTestEnv(t, l1GenesisNum+8)
	spec := e.goodBatch()
	e.plantSpec(spec)
	require.Len(t, e.derive(spec.carrier), 1)
	head, _ := e.facts.Head()

	for n := spec.carrier + 1; n <= l1GenesisNum+8; n++ {
		require.Empty(t, e.derive(n))
	}
	again, ok := e.facts.Head()
	require.True(t, ok, "forward progress must not drop facts")
	require.Equal(t, head, again)
	require.Len(t, e.facts.carrierSnapshot(), 1)
}
