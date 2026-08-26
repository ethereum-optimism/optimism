package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

var (
	testRollupConfigHash = common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	testDepSetHash       = common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
)

// fakeChain answers the handful of RPC methods the builder uses, from an in-memory chain.
type fakeChain struct {
	// head is the chain's unsafe head: what its sequencer has produced.
	head uint64
	// safeLag is how far the SAFE head sits behind the unsafe one. Zero by default so the two
	// coincide and the head-source choice is invisible to every test that is not about it; a
	// silhouette chain sets it, because there the safe head is only ever the last proven block.
	safeLag      uint64
	l1Head       uint64
	logsPer      map[uint64]int
	safeOverride uint64
	// badOutputAt makes the node publish an output root that its own roots do not derive.
	badOutputAt uint64
}

func newFakeChain() *fakeChain {
	return &fakeChain{head: 100, l1Head: 5000, logsPer: map[uint64]int{}}
}

// safeOverride, when non-zero, is reported as the safe head verbatim — the only way to describe a
// chain whose safe head is ABOVE its unsafe one, which is a state the builder must refuse.
func (f *fakeChain) safe() uint64 {
	if f.safeOverride != 0 {
		return f.safeOverride
	}
	return f.head - f.safeLag
}

func (f *fakeChain) CallContext(_ context.Context, result any, method string, args ...any) error {
	switch method {
	case "optimism_syncStatus":
		return assign(result, map[string]any{
			"unsafe_l2": map[string]any{"number": f.head},
			"safe_l2":   map[string]any{"number": f.safe()},
		})
	case "optimism_outputAtBlock":
		n := uint64(args[0].(hexutil.Uint64))
		root := outputRootOf(n)
		if f.badOutputAt != 0 && n == f.badOutputAt {
			root = crypto.Keccak256Hash([]byte("not this block's output root"))
		}
		return assign(result, map[string]any{
			"outputRoot":            root,
			"stateRoot":             stateRootOf(n),
			"withdrawalStorageRoot": messagePasserRootOf(n),
			"blockRef":              map[string]any{"hash": l2HashOf(n), "number": n, "timestamp": 1_800_000_000 + 2*n},
		})
	case "eth_blockNumber":
		*(result.(*hexutil.Uint64)) = hexutil.Uint64(f.l1Head)
		return nil
	case "eth_getBlockByNumber":
		n := uint64(args[0].(hexutil.Uint64))
		return assign(result, map[string]any{
			"timestamp": hexutil.Uint64(1_800_000_000 + 2*n),
			"hash":      l1HashOf(n),
		})
	case "eth_getBlockReceipts":
		n := uint64(args[0].(hexutil.Uint64))
		return assign(result, f.receipts(n))
	}
	return fmt.Errorf("unexpected method %s", method)
}

// receipts renders a block's logs in the JSON shape a node serves, so the builder's decoding is
// exercised rather than bypassed. Indices are block-level and continue across receipts, which is
// the property the builder must carry onto the wire.
func (f *fakeChain) receipts(block uint64) []map[string]any {
	count := f.logsPer[block]
	out := []map[string]any{}
	index := 0
	// Two receipts, so a positional reading of one receipt's logs would produce the wrong index
	// for everything in the second.
	for _, share := range []int{count / 2, count - count/2} {
		logs := []map[string]any{}
		for i := 0; i < share; i++ {
			logs = append(logs, map[string]any{
				"address":  logAddress(block, index),
				"topics":   []common.Hash{logTopic(block, index)},
				"data":     hexutil.Bytes{byte(block), byte(index)},
				"logIndex": hexutil.Uint64(index),
			})
			index++
		}
		out = append(out, map[string]any{"logs": logs})
	}
	return out
}

func logAddress(block uint64, i int) common.Address {
	return common.BytesToAddress(crypto.Keccak256([]byte(fmt.Sprintf("addr-%d-%d", block, i)))[:20])
}

func logTopic(block uint64, i int) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("topic-%d-%d", block, i)))
}

func l2HashOf(n uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("l2-block-%d", n)))
}

func stateRootOf(n uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("l2-state-%d", n)))
}

func messagePasserRootOf(n uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("l2-mp-%d", n)))
}

// outputRootOf is derived from the three roots the fake node serves, exactly as a real node's is.
// A fake that published an unrelated output root would be a node whose own answers disagree, and
// the builder is supposed to notice that (see TestBuilderRejectsInconsistentNode).
func outputRootOf(n uint64) common.Hash {
	blk := proofbatch.BlockExport{
		Hash:                     l2HashOf(n),
		StateRoot:                stateRootOf(n),
		MessagePasserStorageRoot: messagePasserRootOf(n),
	}
	return blk.OutputRoot()
}

func l1HashOf(n uint64) common.Hash {
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("l1-%d", n)))
}

// assign renders a value through JSON into the RPC result pointer, exactly as a real client does.
func assign(result any, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func testBuilderConfig(cursorPath string) builderConfig {
	return builderConfig{
		RollupConfigHash: testRollupConfigHash,
		DepSetHash:       testDepSetHash,
		MaxBlocks:        300,
		L1Lag:            8,
		CursorPath:       cursorPath,
	}
}

func newTestBuilder(t *testing.T, chain *fakeChain) *builder {
	t.Helper()
	b, err := newBuilder(testBuilderConfig(filepath.Join(t.TempDir(), "cursor.json")), chain, chain, chain)
	require.NoError(t, err)
	return b
}

// anchor runs the first cycle, which establishes the anchor a verifier is configured with rather
// than producing a batch.
func anchor(t *testing.T, b *builder, chain *fakeChain) {
	t.Helper()
	batch, err := b.next(context.Background())
	require.Nil(t, batch)
	require.ErrorContains(t, err, "anchored at L2 block")
	require.Equal(t, chain.head, b.cur.LastBlock)
	require.Equal(t, outputRootOf(chain.head), b.cur.OutputRoot)
}

func TestBuilderAnchorsThenBatches(t *testing.T) {
	chain := newFakeChain()
	chain.logsPer[101] = 2
	chain.logsPer[103] = 1
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	chain.head = 103
	batch, err := b.next(context.Background())
	require.NoError(t, err)
	require.NotNil(t, batch)

	require.Equal(t, outputRootOf(100), batch.PrevOutputRoot)
	require.Equal(t, outputRootOf(103), batch.NewOutputRoot)
	require.Equal(t, proofbatch.ExportPolicyAllHashes, batch.ExportPolicyHash)
	require.Equal(t, testRollupConfigHash, batch.RollupConfigHash)
	require.Equal(t, testDepSetHash, batch.DepSetHash)
	// l1Head is the L1 head minus the configured lag, so the batch stays clear of a reorg.
	require.Equal(t, l1HashOf(chain.l1Head-8), batch.L1Head)
	require.NoError(t, batch.CheckStructure())

	require.Len(t, batch.Blocks, 3)
	require.Equal(t, uint64(101), batch.Blocks[0].Number)
	require.Equal(t, uint64(1_800_000_000+2*101), batch.Blocks[0].Timestamp)
	// Every block carries its real identity and both roots, and the head root the batch claims is
	// the one the last block's roots derive.
	for _, blk := range batch.Blocks {
		require.Equal(t, l2HashOf(blk.Number), blk.Hash)
		require.Equal(t, stateRootOf(blk.Number), blk.StateRoot)
		require.Equal(t, messagePasserRootOf(blk.Number), blk.MessagePasserStorageRoot)
	}
	require.Equal(t, batch.Blocks[2].OutputRoot(), batch.NewOutputRoot)

	require.Len(t, batch.Blocks[0].Logs, 2)
	require.Empty(t, batch.Blocks[1].Logs)
	require.Len(t, batch.Blocks[2].Logs, 1)

	// Log indices are the node's own block-level numbering, and the hashes are the interop log
	// hashes of the logs at those indices.
	require.Equal(t, uint32(0), batch.Blocks[0].Logs[0].Index)
	require.Equal(t, uint32(1), batch.Blocks[0].Logs[1].Index)
	require.Equal(t, messages.LogToLogHash(&types.Log{
		Address: logAddress(101, 1),
		Topics:  []common.Hash{logTopic(101, 1)},
		Data:    []byte{101, 1},
	}), batch.Blocks[0].Logs[1].Hash)
	// The default policy exports hashes only.
	require.Empty(t, batch.Blocks[0].Logs[0].Preimage)
}

// TestBuilderUsesBlockLevelLogIndices: the node numbers logs across a whole block, not within a
// receipt, and that number is what an executing message references. Reading the position within a
// receipt instead would silently mis-index everything after the first transaction.
func TestBuilderUsesBlockLevelLogIndices(t *testing.T) {
	chain := newFakeChain()
	chain.logsPer[101] = 5 // split 2 + 3 across two receipts by fakeChain.receipts
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	chain.head = 101
	batch, err := b.next(context.Background())
	require.NoError(t, err)
	require.Len(t, batch.Blocks[0].Logs, 5)
	for i, l := range batch.Blocks[0].Logs {
		require.Equal(t, uint32(i), l.Index)
		require.Equal(t, messages.LogToLogHash(&types.Log{
			Address: logAddress(101, i),
			Topics:  []common.Hash{logTopic(101, i)},
			Data:    []byte{101, byte(i)},
		}), l.Hash, "log %d", i)
	}
}

// TestBuilderRejectsInconsistentNode: the derived output root is checked against the one the node
// publishes, so a batch that a v2 verifier would refuse is never spent a blob on.
func TestBuilderRejectsInconsistentNode(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	chain.badOutputAt = 102
	chain.head = 103
	_, err := b.next(context.Background())
	require.ErrorContains(t, err, "derived output root")
}

func TestBuilderChainsBatches(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	chain.head = 102
	first, err := b.next(context.Background())
	require.NoError(t, err)
	require.NoError(t, b.commit(first))

	chain.head = 105
	second, err := b.next(context.Background())
	require.NoError(t, err)

	require.Equal(t, first.NewOutputRoot, second.PrevOutputRoot)
	require.Equal(t, uint64(103), second.Blocks[0].Number)
	require.Equal(t, first.Blocks[len(first.Blocks)-1].Number+1, second.Blocks[0].Number)
}

func TestBuilderCapsBatchSize(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	b.cfg.MaxBlocks = 4
	anchor(t, b, chain)

	chain.head = 200
	batch, err := b.next(context.Background())
	require.NoError(t, err)
	require.Len(t, batch.Blocks, 4)
	require.Equal(t, uint64(104), batch.Blocks[3].Number)
	require.Equal(t, outputRootOf(104), batch.NewOutputRoot)
}

func TestBuilderWaitsForNewBlocks(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	batch, err := b.next(context.Background())
	require.NoError(t, err)
	require.Nil(t, batch)
}

func TestBuilderResumesFromCursor(t *testing.T) {
	chain := newFakeChain()
	cfg := testBuilderConfig(filepath.Join(t.TempDir(), "cursor.json"))

	b, err := newBuilder(cfg, chain, chain, chain)
	require.NoError(t, err)
	anchor(t, b, chain)
	chain.head = 102
	first, err := b.next(context.Background())
	require.NoError(t, err)
	require.NoError(t, b.commit(first))

	// A restarted submitter picks up exactly where the last landed batch ended.
	resumed, err := newBuilder(cfg, chain, chain, chain)
	require.NoError(t, err)
	require.Equal(t, uint64(102), resumed.cur.LastBlock)
	chain.head = 104
	next, err := resumed.next(context.Background())
	require.NoError(t, err)
	require.Equal(t, first.NewOutputRoot, next.PrevOutputRoot)
	require.Equal(t, uint64(103), next.Blocks[0].Number)
}

func TestBuilderRoundTripsThroughTheWire(t *testing.T) {
	chain := newFakeChain()
	chain.logsPer[101] = 3
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)
	chain.head = 102

	batch, err := b.next(context.Background())
	require.NoError(t, err)
	payload, err := proofbatch.Encode(batch, nil)
	require.NoError(t, err)
	blobs, err := proofbatch.ToBlobs(payload)
	require.NoError(t, err)
	back, err := proofbatch.FromBlobs(blobs)
	require.NoError(t, err)
	env, err := proofbatch.Decode(back)
	require.NoError(t, err)
	require.Empty(t, env.Proof)
	// Equality is asserted on the bytes: a block with no logs encodes an empty array either way,
	// so the decoded object differs from the built one only in nil-vs-empty.
	reEncoded, err := proofbatch.Encode(&env.Batch, env.Proof)
	require.NoError(t, err)
	require.Equal(t, payload, reEncoded)
	require.Equal(t, batch.NewOutputRoot, env.Batch.NewOutputRoot)
	require.Equal(t, batch.Blocks[0].Logs, env.Batch.Blocks[0].Logs)
}

func TestBuilderRequiresL1Depth(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)

	chain.l1Head = 3
	chain.head = 102
	_, err := b.next(context.Background())
	require.ErrorContains(t, err, "below the configured lag")
}

// TestHeadSourceParsing: both spellings are legal-looking, and an unknown one is refused rather
// than defaulted, because picking the wrong head is invisible until nothing batches.
func TestHeadSourceParsing(t *testing.T) {
	got, err := ParseHeadSource("unsafe")
	require.NoError(t, err)
	require.Equal(t, HeadUnsafe, got)
	require.Equal(t, "unsafe", got.String())

	got, err = ParseHeadSource("safe")
	require.NoError(t, err)
	require.Equal(t, HeadSafe, got)
	require.Equal(t, "safe", got.String())

	_, err = ParseHeadSource("proven")
	require.ErrorContains(t, err, `unknown head source "proven"`)
	require.ErrorContains(t, err, "silhouette")

	// The zero value is the silhouette default, so a config that forgets the field is correct for
	// the chain this tool exists to serve.
	require.Equal(t, HeadUnsafe, HeadSource(0))
}

// TestBuilderBatchesOnTheUnsafeHead is the silhouette spec change, stated as the deadlock it avoids.
//
// The chain here is a silhouette chain: its safe head IS the last proven block, because the only
// thing that could advance it is derivation from the proof batches this tool posts. So safeLag
// tracks the cursor exactly. On the safe head the submitter posts nothing, forever, while looking
// healthy. On the unsafe head it follows the sequencer.
func TestBuilderBatchesOnTheUnsafeHead(t *testing.T) {
	// The silhouette default: batch to the unsafe head.
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	anchor(t, b, chain)
	// Anchored at 100, and nothing has made 100 safe — that is the point.
	chain.safeLag = 20
	chain.head = 130

	batch, err := b.next(context.Background())
	require.NoError(t, err)
	require.NotNil(t, batch)
	require.Equal(t, uint64(101), batch.Blocks[0].Number)
	require.Equal(t, uint64(130), batch.Blocks[len(batch.Blocks)-1].Number)
	require.NoError(t, batch.CheckStructure())
	require.NoError(t, b.commit(batch))

	// And it chains: the next cycle picks up from where this one ended, still ignoring the safe head.
	chain.head = 145
	batch, err = b.next(context.Background())
	require.NoError(t, err)
	require.NotNil(t, batch)
	require.Equal(t, uint64(131), batch.Blocks[0].Number)
	require.Equal(t, uint64(145), batch.Blocks[len(batch.Blocks)-1].Number)
}

// TestBuilderOnSafeHeadDeadlocksOnASilhouetteChain is the negative half, and it is the reason the
// flag exists. Same chain, same cadence, HeadSafe: the safe head never leaves the cursor, so every
// cycle returns "nothing new" and the chain's public history never starts.
func TestBuilderOnSafeHeadDeadlocksOnASilhouetteChain(t *testing.T) {
	chain := newFakeChain()
	cfg := testBuilderConfig(filepath.Join(t.TempDir(), "cursor.json"))
	cfg.Head = HeadSafe
	b, err := newBuilder(cfg, chain, chain, chain)
	require.NoError(t, err)

	// Anchors on the safe head, which on this chain is block 100.
	batch, err := b.next(context.Background())
	require.Nil(t, batch)
	require.ErrorContains(t, err, "anchored at L2 block 100")

	// The sequencer produces 60 more blocks. Nothing proves them, so the safe head does not move.
	for _, h := range []uint64{130, 145, 160} {
		chain.safeLag = h - 100
		chain.head = h
		batch, err := b.next(context.Background())
		require.NoError(t, err, "the deadlock is silent: no error, just no batch")
		require.Nil(t, batch, "safe head %d is still the cursor, so nothing to batch", chain.safe())
	}
	require.Equal(t, uint64(100), b.cur.LastBlock, "the cursor never left the anchor")

	// The same chain, on the unsafe head, batches immediately — so the chain was never the problem.
	cfg.Head = HeadUnsafe
	cfg.CursorPath = filepath.Join(t.TempDir(), "cursor2.json")
	b2, err := newBuilder(cfg, chain, chain, chain)
	require.NoError(t, err)
	_, err = b2.next(context.Background())
	require.ErrorContains(t, err, "anchored at L2 block 160")
	chain.head = 200
	batch, err = b2.next(context.Background())
	require.NoError(t, err)
	require.NotNil(t, batch)
	require.Equal(t, uint64(200), batch.Blocks[len(batch.Blocks)-1].Number)
}

// TestBuilderRefusesUnsafeBehindSafe: a chain reporting an unsafe head below its safe head is not
// in a state this tool should build a batch from, so it refuses instead of underflowing.
func TestBuilderRefusesUnsafeBehindSafe(t *testing.T) {
	chain := newFakeChain()
	b := newTestBuilder(t, chain)
	chain.head = 90
	chain.safeOverride = 100
	_, err := b.next(context.Background())
	require.ErrorContains(t, err, "is below the safe head")
}
