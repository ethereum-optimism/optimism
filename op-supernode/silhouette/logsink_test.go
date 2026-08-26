package silhouette

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	coreinterop "github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop/raftwallogdb"
)

// These tests drive the REAL interop log database — the same raft-wal store a driven chain's logs
// are sealed into — rather than a mock. That is the whole point: the claim under test is that a
// proof-carried chain's LogsDB is keyed exactly like a driven chain's, and a mock store would let
// that claim be true of the mock instead of of the database.

const (
	sinkChainID   = 424247
	sinkBlockTime = uint64(2)
)

// newSinkHarness opens a real LogsDB and a sink over it.
func newSinkHarness(t *testing.T) (*LogSink, *raftwallogdb.DB) {
	t.Helper()
	db, err := raftwallogdb.Open(t.TempDir(), eth.ChainIDFromUInt64(sinkChainID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	// The compile-time claim that the supernode's own store satisfies the narrow write interface,
	// with neither package importing the other.
	var _ LogStore = db
	return NewLogSink(testlog.Logger(t, log.LevelInfo), db), db
}

// exportedLog is a real log hash: keccak over an address and a payload, exactly as
// messages.LogToLogHash computes it, so nothing here is a hash-shaped placeholder.
func exportedLog(addr byte, payload string) common.Hash {
	return messages.PayloadHashToLogHash(
		crypto.Keccak256Hash([]byte(payload)),
		common.Address{addr},
	)
}

// sinkBlock builds one block export at the given number with the given (index, hash) logs.
func sinkBlock(number uint64, logs ...proofbatch.LogExport) proofbatch.BlockExport {
	return proofbatch.BlockExport{
		Number:    number,
		Timestamp: 1_000_000 + number*sinkBlockTime,
		Hash:      common.BytesToHash([]byte{byte(number >> 8), byte(number), 0xbb}),
		StateRoot: common.BytesToHash([]byte{byte(number), 0x57}),
		MessagePasserStorageRoot: common.BytesToHash(
			[]byte{byte(number), 0x11}),
		Logs: logs,
	}
}

// queryFor builds the ContainsQuery an executing message referencing (block, logIdx) would carry,
// for a caller who believes the log there hashes to logHash. This is the executing side of the
// protocol, computed the same way the CrossL2Inbox computes it.
func queryFor(blk proofbatch.BlockExport, logIdx uint32, logHash common.Hash) messages.ContainsQuery {
	return messages.ChecksumArgs{
		BlockNumber: blk.Number,
		LogIndex:    logIdx,
		Timestamp:   blk.Timestamp,
		ChainID:     eth.ChainIDFromUInt64(sinkChainID),
		LogHash:     logHash,
	}.Query()
}

// TestSinkSealsExportedLogsAtTheirWireIndices is the explicit-indices half of the design: a
// curated export policy names each surviving log's REAL block-level index, and the survivors must
// land on those indices rather than being packed down to 0,1,2…
//
// Packing them down is the failure this test exists to prevent, and it is silent: every log would
// still be present and every hash would still be right, but every executing message referencing
// this chain would point at the wrong log.
func TestSinkSealsExportedLogsAtTheirWireIndices(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	first := exportedLog(0xa1, "curated-first")
	fifth := exportedLog(0xa5, "curated-fifth")
	blk := sinkBlock(101,
		proofbatch.LogExport{Index: 1, Hash: first},
		proofbatch.LogExport{Index: 5, Hash: fifth},
	)
	parent := common.HexToHash("0xfeed")
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{blk}, parent))

	ref, logCount, execMsgs, err := db.OpenBlock(blk.Number)
	require.NoError(t, err)
	require.Equal(t, uint32(6), logCount, "six slots: indices 0..5, the highest exported index plus one")
	require.Equal(t, blk.Hash, ref.Hash, "the block is sealed under its REAL wire hash")
	require.Empty(t, execMsgs,
		"a silhouette chain contributes no executing messages: its own are proof-trusted")

	// The two exported logs resolve at their own indices.
	seal, err := db.Contains(queryFor(blk, 1, first))
	require.NoError(t, err, "the log exported at index 1 must be referenceable at index 1")
	require.Equal(t, blk.Hash, seal.Hash)
	seal, err = db.Contains(queryFor(blk, 5, fifth))
	require.NoError(t, err, "the log exported at index 5 must be referenceable at index 5")
	require.Equal(t, blk.Hash, seal.Hash)

	// And they do NOT resolve at the packed-down positions a naive implementation would use.
	_, err = db.Contains(queryFor(blk, 0, first))
	require.ErrorIs(t, err, coreinterop.ErrConflict,
		"index 0 is a gap, not the first exported log: packing would have made this succeed")
}

// TestPoisonedGapsAreUnreferenceable is the export-policy gate.
//
// A gap is a log P deliberately did NOT export. Nothing must be able to reference it. The slot is
// occupied by a domain-separated constant with no known preimage, so every query naming a real or
// attacker-chosen log hash at that position fails its checksum comparison — which is the mechanism
// that makes the export policy enforceable rather than advisory.
func TestPoisonedGapsAreUnreferenceable(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	exported := exportedLog(0xa5, "the-one-we-exported")
	blk := sinkBlock(101, proofbatch.LogExport{Index: 5, Hash: exported})
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{blk}, common.HexToHash("0xfeed")))

	// Every gap index, probed with a plausible guess at what P might have emitted there. This is
	// the attack: an insider who knows the interior of P knows the real log hash at index 3, and
	// tries to execute against it even though the export policy withheld it.
	withheld := exportedLog(0xa3, "a-log-P-chose-not-to-export")
	for _, gap := range []uint32{0, 1, 2, 3, 4} {
		_, err := db.Contains(queryFor(blk, gap, withheld))
		require.ErrorIsf(t, err, coreinterop.ErrConflict,
			"index %d is unexported and must be unreferenceable", gap)

		// The same slot is equally unreferenceable naming the log that WAS exported, so a caller
		// cannot smuggle an exported log's hash onto an unexported index either.
		_, err = db.Contains(queryFor(blk, gap, exported))
		require.ErrorIsf(t, err, coreinterop.ErrConflict,
			"the exported log's hash must not resolve at gap index %d", gap)
	}

	// Above the highest exported index there is no slot at all — a different refusal, and also a
	// refusal.
	_, err := db.Contains(queryFor(blk, 6, withheld))
	require.ErrorIs(t, err, coreinterop.ErrConflict, "index 6 was never sealed")
}

// TestPoisonValueHasNoRecoverablePreimage records the security argument for publishing the poison
// constant, as an executable statement rather than a comment.
//
// The constant is public. That is safe because referencing a slot requires a payload the
// CrossL2Inbox will hash INTO the log hash, and the poison is not the LogToLogHash of any
// (address, payload) anyone can produce. What this test can check is the shape claim underneath
// that argument: the poison is not accidentally equal to the hash of an empty or trivial log, which
// is the one way it could be reachable without breaking keccak.
func TestPoisonValueHasNoRecoverablePreimage(t *testing.T) {
	t.Parallel()
	trivial := []common.Hash{
		messages.PayloadHashToLogHash(common.Hash{}, common.Address{}),
		messages.PayloadHashToLogHash(crypto.Keccak256Hash(nil), common.Address{}),
		crypto.Keccak256Hash(nil),
		{},
	}
	for _, h := range trivial {
		require.NotEqual(t, unexportedLogHash, h,
			"the poison must not coincide with a log hash anyone can construct directly")
	}
	require.Equal(t, crypto.Keccak256Hash([]byte("silhouette:unexported-log")), unexportedLogHash,
		"the poison is a domain-separated constant; changing it changes which slots are poisoned")
}

// TestSinkSealsUnderRealWireHashes is the real-identity half: a proof-carried chain's blocks are
// sealed under the block hashes the proof committed to, so nothing downstream needs a re-anchoring
// or hash-translation concept.
func TestSinkSealsUnderRealWireHashes(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	blocks := []proofbatch.BlockExport{
		sinkBlock(101, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xb1, "one")}),
		sinkBlock(102),
		sinkBlock(103, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xb3, "three")}),
	}
	parent := common.HexToHash("0xfeed")
	require.NoError(t, sink.Accept(blocks, parent))

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, blocks[2].Hash, latest.Hash, "the head is the last block's real wire hash")
	require.Equal(t, uint64(103), latest.Number)

	for _, blk := range blocks {
		seal, err := db.FindSealedBlock(blk.Number)
		require.NoError(t, err)
		require.Equal(t, blk.Hash, seal.Hash, "block %d sealed under its wire hash", blk.Number)
		require.Equal(t, blk.Timestamp, seal.Timestamp)
	}

	// A block with no exported logs is sealed, not skipped: skipping it would break the chaining of
	// the block after it and leave a hole the judge would read as a gap.
	_, logCount, _, err := db.OpenBlock(102)
	require.NoError(t, err, "a block that exported nothing is still a sealed block")
	require.Zero(t, logCount)
}

// TestSinkReplayComparesRatherThanReseals is the restart property. Proven history is re-derived
// from L1 on every start, so the sink walks back over blocks the store already holds. Those must be
// re-CHECKED, not re-sealed and not skipped — which is also what makes accepting the same batch
// twice idempotent instead of an error.
func TestSinkReplayComparesRatherThanReseals(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	blocks := []proofbatch.BlockExport{
		sinkBlock(101, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xc1, "one")}),
		sinkBlock(102, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xc2, "two")}),
	}
	parent := common.HexToHash("0xfeed")
	require.NoError(t, sink.Accept(blocks, parent))
	before, ok := db.LatestSealedBlock()
	require.True(t, ok)

	require.NoError(t, sink.Accept(blocks, parent), "replaying proven history must be idempotent")
	after, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, before, after, "a replay must not move the head")

	// And the logs are still there exactly once, at their own indices.
	_, logCount, _, err := db.OpenBlock(102)
	require.NoError(t, err)
	require.Equal(t, uint32(1), logCount, "a replay must not duplicate a block's logs")
}

// TestSinkRefusesForeignHistory: a store built under a different anchor holds a different block at
// a height this node's proofs commit to. Writing over it would produce history that does not chain,
// so the sink refuses and says what to do about it.
func TestSinkRefusesForeignHistory(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	blk := sinkBlock(101, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xd1, "one")})
	parent := common.HexToHash("0xfeed")
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{blk}, parent))

	imposter := blk
	imposter.Hash = common.HexToHash("0xdeadbeef")
	err := sink.Accept([]proofbatch.BlockExport{imposter}, parent)
	require.ErrorIs(t, err, ErrForeignHistory)

	seal, findErr := db.FindSealedBlock(101)
	require.NoError(t, findErr)
	require.Equal(t, blk.Hash, seal.Hash, "the refusal must not have overwritten the real block")
}

// TestSinkUnwindsAPartiallyWrittenBatch is the whole-or-nothing property. A batch is one
// transaction and one proof; a failure partway through must leave the store as it was, or the
// proven head and the message database disagree about where history ends.
func TestSinkUnwindsAPartiallyWrittenBatch(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	// Establish a head so there is a "before" state that is not merely empty.
	base := sinkBlock(100, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xe0, "base")})
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{base}, common.HexToHash("0xfeed")))

	// The second block of the batch claims a log index above what this node will index, so it fails
	// AFTER the first block of the same batch has already been sealed.
	good := sinkBlock(101, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xe1, "good")})
	bad := sinkBlock(102, proofbatch.LogExport{Index: maxBlockLogIndex + 1, Hash: exportedLog(0xe2, "bad")})
	err := sink.Accept([]proofbatch.BlockExport{good, bad}, base.Hash)
	require.Error(t, err)
	require.Contains(t, err.Error(), "above the")

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, base.Hash, latest.Hash,
		"a failed batch must leave the store at the head it had before the batch")
	require.Equal(t, uint64(100), latest.Number)
}

// TestSinkRewindUnsealsOrphanedMessages: when an L1 reorg drops the batch that proved a block, the
// messages that block exported must stop being referenceable. Facts and messages move together or
// another chain can execute against an initiating message nothing on L1 proves.
func TestSinkRewindUnsealsOrphanedMessages(t *testing.T) {
	t.Parallel()
	sink, db := newSinkHarness(t)

	keep := sinkBlock(101, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xf1, "kept")})
	orphan := sinkBlock(102, proofbatch.LogExport{Index: 0, Hash: exportedLog(0xf2, "orphaned")})
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{keep, orphan}, common.HexToHash("0xfeed")))

	// Before: the orphaned block's message is referenceable.
	_, err := db.Contains(queryFor(orphan, 0, orphan.Logs[0].Hash))
	require.NoError(t, err)

	require.NoError(t, sink.Rewind(eth.BlockID{Hash: keep.Hash, Number: keep.Number}))

	// After: it is not, and the surviving block is untouched.
	_, err = db.Contains(queryFor(orphan, 0, orphan.Logs[0].Hash))
	require.Error(t, err, "a message whose proof was orphaned must stop being referenceable")
	seal, err := db.Contains(queryFor(keep, 0, keep.Logs[0].Hash))
	require.NoError(t, err, "the surviving block's message is unaffected")
	require.Equal(t, keep.Hash, seal.Hash)

	// A rewind to a target at or above the head is a no-op rather than an error: forward progress
	// calls this on every reset that dropped nothing.
	require.NoError(t, sink.Rewind(eth.BlockID{Hash: keep.Hash, Number: keep.Number}))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, keep.Hash, latest.Hash)
}

// TestNilSinkIsANoOp: deriving P's chain and making P's messages referenceable are separable, and a
// source with no sink attached must not panic on either path.
func TestNilSinkIsANoOp(t *testing.T) {
	t.Parallel()
	var sink *LogSink
	require.NoError(t, sink.Accept([]proofbatch.BlockExport{sinkBlock(101)}, common.Hash{}))
	require.NoError(t, sink.Rewind(eth.BlockID{Number: 100}))
}
