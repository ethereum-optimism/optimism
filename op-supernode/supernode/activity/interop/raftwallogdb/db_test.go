package raftwallogdb

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

func hash(b byte) common.Hash {
	var h common.Hash
	for i := range h {
		h[i] = b
	}
	return h
}

func blockID(num uint64, b byte) eth.BlockID {
	return eth.BlockID{Hash: hash(b), Number: num}
}

func tempDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(t.TempDir(), eth.ChainIDFromUInt64(10))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// sealRange seals blocks [start..end] (inclusive) on top of parent, each with
// timestamp = number * 100. Returns the last sealed block ID.
func sealRange(t *testing.T, db *DB, parent eth.BlockID, start, end uint64) eth.BlockID {
	t.Helper()
	prev := parent
	for n := start; n <= end; n++ {
		blk := blockID(n, byte(n))
		require.NoError(t, db.SealBlock(prev.Hash, blk, n*100))
		prev = blk
	}
	return prev
}

func TestEmpty(t *testing.T) {
	db := tempDB(t)
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
	_, err := db.FirstSealedBlock()
	require.ErrorIs(t, err, interop.ErrFuture)
	_, err = db.FindSealedBlock(0)
	require.ErrorIs(t, err, interop.ErrFuture)
	_, _, _, err = db.OpenBlock(1)
	require.ErrorIs(t, err, interop.ErrFuture)
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 1, Timestamp: 1})
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestSealAndOpenBlock(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 100))

	em := &messages.ExecutingMessage{
		ChainID:   eth.ChainIDFromUInt64(20),
		BlockNum:  1,
		LogIdx:    1,
		Timestamp: 200,
		Checksum:  messages.MessageChecksum(hash(0xEE)),
	}
	require.NoError(t, db.AddLog(hash(0x01), parent, 0, nil))
	require.NoError(t, db.AddLog(hash(0x02), parent, 1, em))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 200))

	ref, count, msgs, err := db.OpenBlock(1)
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, ref.Hash)
	require.Equal(t, parent.Hash, ref.ParentHash)
	require.Equal(t, uint64(200), ref.Time)
	require.Equal(t, uint32(2), count)
	require.Len(t, msgs, 1)
	require.Equal(t, em, msgs[1])
}

func TestSealBlock_FirstBlockAcceptsAnyNumber(t *testing.T) {
	// Empty DB accepts any first block, including a non-zero number.
	db := tempDB(t)
	first := blockID(100, 0xAA)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, first, latest)
}

func TestSealBlock_Validation(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 100))

	// Wrong parent hash.
	err := db.SealBlock(hash(0xFF), blockID(1, 0x11), 200)
	require.ErrorIs(t, err, interop.ErrConflict)

	// Wrong block number. Matches old logs DB: state desync on SealBlock is
	// ErrConflict so op-interop-filter's failsafe (logsdb_chain_ingester.go:793)
	// trips rather than silently retrying.
	err = db.SealBlock(parent.Hash, blockID(2, 0x22), 200)
	require.ErrorIs(t, err, interop.ErrConflict)

	// Timestamp regression. Same rationale: regressing timestamps indicate
	// upstream desync, not a transient out-of-order condition.
	err = db.SealBlock(parent.Hash, blockID(1, 0x11), 50)
	require.ErrorIs(t, err, interop.ErrConflict)
}

func TestAddLog_Validation(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))

	// First log must have index 0.
	err := db.AddLog(hash(0x01), parent, 1, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)

	// Wrong parent for the first AddLog establishes pendingParent;
	// subsequent calls must match. Matches old logs DB: parent identity
	// mismatch in AddLog is ErrOutOfOrder, not ErrConflict.
	require.NoError(t, db.AddLog(hash(0x01), parent, 0, nil))
	err = db.AddLog(hash(0x02), blockID(99, 0xFF), 1, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)

	// Duplicate index.
	err = db.AddLog(hash(0x02), parent, 0, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)

	// Skipping an index.
	err = db.AddLog(hash(0x02), parent, 2, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)
}

// TestAddLog_RejectsGenesisParent ensures AddLog refuses logs whose parent is
// the zero BlockID. Genesis blocks cannot carry receipts (the EVM does not
// execute genesis), so logs against block 0 are invalid input from any
// legitimate writer. Rejecting at the write boundary matches the old logs DB
// (state.go:431) and keeps the structural invariant out of the read path.
func TestAddLog_RejectsGenesisParent(t *testing.T) {
	db := tempDB(t)
	err := db.AddLog(hash(0xAA), eth.BlockID{}, 0, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)
}

func TestAddLog_WrongParentAgainstLatest(t *testing.T) {
	// When the DB already has a sealed block, AddLog's parent must match it.
	// Matches old logs DB: AddLog parent mismatch is ErrOutOfOrder.
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	err := db.AddLog(hash(0xAA), blockID(99, 0xFF), 0, nil)
	require.ErrorIs(t, err, interop.ErrOutOfOrder)
}

func TestContains(t *testing.T) {
	db := tempDB(t)
	chain := eth.ChainIDFromUInt64(10)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 100))
	logHash := hash(0xAB)
	require.NoError(t, db.AddLog(logHash, parent, 0, nil))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 200))

	good := messages.ChecksumArgs{
		BlockNumber: 1, LogIndex: 0, Timestamp: 200, ChainID: chain, LogHash: logHash,
	}.Checksum()

	// Happy path.
	seal, err := db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: 0, Timestamp: 200, Checksum: good})
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, seal.Hash)

	// Wrong checksum.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: 0, Timestamp: 200, Checksum: messages.MessageChecksum(hash(0xDE))})
	require.ErrorIs(t, err, interop.ErrConflict)

	// Wrong timestamp for the block.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: 0, Timestamp: 999, Checksum: good})
	require.ErrorIs(t, err, interop.ErrConflict)

	// LogIdx out of range for an existing block.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: 5, Timestamp: 200, Checksum: good})
	require.ErrorIs(t, err, interop.ErrConflict)

	// Future block (timestamp consistent with growth): ErrFuture.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 10, LogIdx: 0, Timestamp: 999, Checksum: good})
	require.ErrorIs(t, err, interop.ErrFuture)

	// Future block but past timestamp: ErrConflict (cannot be in the future).
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 10, LogIdx: 0, Timestamp: 50, Checksum: good})
	require.ErrorIs(t, err, interop.ErrConflict)
}

func TestContains_Block0(t *testing.T) {
	// Block 0 must flow through the same validation paths as any other block —
	// no special-case rejection. Matches op-supervisor's logs.DB. The supernode
	// only seals block 0 as a logless genesis (see processBlockLogs in
	// op-supernode/.../logdb.go), so we test that shape here.
	db := tempDB(t)

	// Empty DB → ErrFuture, regardless of block number.
	_, err := db.Contains(messages.ContainsQuery{BlockNum: 0, LogIdx: 0, Timestamp: 0})
	require.ErrorIs(t, err, interop.ErrFuture)

	// Seal block 0 as a logless genesis.
	genesis := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, genesis, 0))

	// Sealed genesis is discoverable by FindSealedBlock / OpenBlock.
	seal, err := db.FindSealedBlock(0)
	require.NoError(t, err)
	require.Equal(t, genesis.Hash, seal.Hash)
	ref, count, msgs, err := db.OpenBlock(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), ref.Number)
	require.Equal(t, uint32(0), count)
	require.Empty(t, msgs)

	// Contains on block 0 with logIdx 0 hits the "logIdx >= logCount" check (not
	// a hard-coded block-0 reject) and returns ErrConflict.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 0, LogIdx: 0, Timestamp: 0})
	require.ErrorIs(t, err, interop.ErrConflict)
}

func TestFindSealedBlock(t *testing.T) {
	db := tempDB(t)
	// Anchor first block at 100 to exercise the "before first" path.
	first := blockID(100, 0x64)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))
	sealRange(t, db, first, 101, 105)

	seal, err := db.FindSealedBlock(100)
	require.NoError(t, err)
	require.Equal(t, first.Hash, seal.Hash)
	require.Equal(t, uint64(1000), seal.Timestamp)

	seal, err = db.FindSealedBlock(103)
	require.NoError(t, err)
	require.Equal(t, hash(103), seal.Hash)

	// Below first block.
	_, err = db.FindSealedBlock(50)
	require.True(t, errors.Is(err, interop.ErrSkipped))

	// Past the latest block.
	_, err = db.FindSealedBlock(999)
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestOpenBlock_Boundaries(t *testing.T) {
	db := tempDB(t)
	require.NoError(t, db.SealBlock(common.Hash{}, blockID(100, 0x64), 1000))
	sealRange(t, db, blockID(100, 0x64), 101, 103)

	// Below first block.
	_, _, _, err := db.OpenBlock(50)
	require.ErrorIs(t, err, interop.ErrSkipped)

	// Above latest block.
	_, _, _, err = db.OpenBlock(999)
	require.ErrorIs(t, err, interop.ErrFuture)

	// Block 0 only valid if DB anchors at 0.
	db2 := tempDB(t)
	require.NoError(t, db2.SealBlock(common.Hash{}, blockID(0, 0xA0), 0))
	ref, count, msgs, err := db2.OpenBlock(0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), ref.Number)
	require.Equal(t, uint32(0), count)
	require.Empty(t, msgs)
}

// TestOpenBlock_FirstBlockReturnsSkipped asserts that OpenBlock at the first
// sealed block number returns ErrSkipped when the first block is not genesis,
// matching the old op-supervisor logs DB contract. Callers (op-supernode's
// algo.go fallback) rely on this to identify the anchor and use
// FirstSealedBlock instead. FirstSealedBlock must still return the block.
func TestOpenBlock_FirstBlockReturnsSkipped(t *testing.T) {
	db := tempDB(t)
	first := blockID(100, 0x64)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))
	sealRange(t, db, first, 101, 103)

	_, _, _, err := db.OpenBlock(100)
	require.ErrorIs(t, err, interop.ErrSkipped, "OpenBlock(firstBlock) must return ErrSkipped when firstBlock > 0")

	seal, err := db.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(100), seal.Number)
	require.Equal(t, first.Hash, seal.Hash)
	require.Equal(t, uint64(1000), seal.Timestamp)

	// OpenBlock(firstBlock + 1) still succeeds.
	ref, _, _, err := db.OpenBlock(101)
	require.NoError(t, err)
	require.Equal(t, uint64(101), ref.Number)
}

func TestMultiBlockRoundtrip(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))

	prev := parent
	for n := uint64(1); n <= 10; n++ {
		// 3 logs per block, one carries an execMsg.
		for idx := uint32(0); idx < 3; idx++ {
			var em *messages.ExecutingMessage
			if idx == 1 {
				em = &messages.ExecutingMessage{
					ChainID:   eth.ChainIDFromUInt64(42),
					BlockNum:  n,
					LogIdx:    idx,
					Timestamp: n * 10,
					Checksum:  messages.MessageChecksum(hash(byte(n*10 + uint64(idx)))),
				}
			}
			require.NoError(t, db.AddLog(hash(byte(n*10+uint64(idx))), prev, idx, em))
		}
		blk := blockID(n, byte(n))
		require.NoError(t, db.SealBlock(prev.Hash, blk, n*10))
		prev = blk
	}

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, uint64(10), latest.Number)

	for n := uint64(1); n <= 10; n++ {
		ref, count, msgs, err := db.OpenBlock(n)
		require.NoError(t, err)
		require.Equal(t, n, ref.Number)
		require.Equal(t, uint32(3), count)
		require.Len(t, msgs, 1)
		require.Contains(t, msgs, uint32(1))
		require.Equal(t, uint64(n*10), msgs[1].Timestamp)
	}
}

func TestRewind(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	last := sealRange(t, db, parent, 1, 5)
	require.Equal(t, uint64(5), last.Number)

	target := blockID(3, 0x03)
	require.NoError(t, db.Rewind(target))

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, target, latest)

	// Read-after-rewind: deleted blocks are gone, target survives.
	_, err := db.FindSealedBlock(5)
	require.ErrorIs(t, err, interop.ErrFuture)
	_, err = db.FindSealedBlock(4)
	require.ErrorIs(t, err, interop.ErrFuture)
	_, err = db.FindSealedBlock(3)
	require.NoError(t, err)
	_, _, _, err = db.OpenBlock(5)
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestRewind_HashMismatch(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	sealRange(t, db, parent, 1, 3)

	bogus := eth.BlockID{Hash: hash(0xFF), Number: 2}
	err := db.Rewind(bogus)
	require.ErrorIs(t, err, interop.ErrConflict)
}

func TestRewind_Empty(t *testing.T) {
	// Rewind on an empty DB is a no-op.
	db := tempDB(t)
	require.NoError(t, db.Rewind(blockID(5, 0x05)))
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
}

func TestRewind_AtLatest(t *testing.T) {
	// Rewind to the existing latest block is a no-op.
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	last := sealRange(t, db, parent, 1, 3)

	require.NoError(t, db.Rewind(last))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, last, latest)
}

func TestRewind_AboveLatest(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))

	_, err := db.FindSealedBlock(0)
	require.NoError(t, err)
	err = db.Rewind(blockID(99, 0x99))
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestRewind_BeforeFirstClears(t *testing.T) {
	db := tempDB(t)
	first := blockID(100, 0x64)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))

	require.NoError(t, db.Rewind(blockID(50, 0x32)))
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
}

func TestRewind_AtFirstKeepsIt(t *testing.T) {
	db := tempDB(t)
	first := blockID(100, 0x64)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))
	sealRange(t, db, first, 101, 105)

	require.NoError(t, db.Rewind(first))
	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, first, latest)
	_, err := db.FindSealedBlock(101)
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestRewind_DropsPendingLogs(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	require.NoError(t, db.AddLog(hash(0xAA), parent, 0, nil))

	require.NoError(t, db.Rewind(parent))
	// After rewind, pending buffer must be clear — first AddLog accepts index 0 again.
	require.NoError(t, db.AddLog(hash(0xBB), parent, 0, nil))
}

func TestClear_Populated(t *testing.T) {
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	sealRange(t, db, parent, 1, 3)

	require.NoError(t, db.Clear())
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
	_, err := db.FirstSealedBlock()
	require.ErrorIs(t, err, interop.ErrFuture)
	_, err = db.FindSealedBlock(2)
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestClear_Empty(t *testing.T) {
	db := tempDB(t)
	require.NoError(t, db.Clear())
	_, ok := db.LatestSealedBlock()
	require.False(t, ok)
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	require.NoError(t, db.AddLog(hash(0xAA), parent, 0, nil))
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 100))
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, blk1, latest)

	_, count, _, err := db2.OpenBlock(1)
	require.NoError(t, err)
	require.Equal(t, uint32(1), count)
}

func TestPreSealCrashLosesPending(t *testing.T) {
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	parent := blockID(5, 0x05)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 500))
	require.NoError(t, db.AddLog(hash(0xAA), parent, 0, nil))
	require.NoError(t, db.AddLog(hash(0xBB), parent, 1, nil))
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, parent, latest)

	// Block 6 must not exist.
	_, err = db2.FindSealedBlock(6)
	require.ErrorIs(t, err, interop.ErrFuture)
}

func TestRecordEncoding_Roundtrip(t *testing.T) {
	logHashes := []common.Hash{hash(0x11), hash(0x22), hash(0x33)}
	execMsgs := []struct {
		localLogIdx uint32
		msg         messages.ExecutingMessage
	}{
		{
			localLogIdx: 0,
			msg: messages.ExecutingMessage{
				ChainID:   eth.ChainIDFromUInt64(8453),
				BlockNum:  99,
				LogIdx:    7,
				Timestamp: 4242,
				Checksum:  messages.MessageChecksum(hash(0x77)),
			},
		},
		{
			localLogIdx: 2,
			msg: messages.ExecutingMessage{
				ChainID:   eth.ChainIDFromUInt64(10),
				BlockNum:  100,
				LogIdx:    1,
				Timestamp: 4243,
				Checksum:  messages.MessageChecksum(hash(0x88)),
			},
		},
	}

	br := blockRecord{
		Hash:         hash(0xAB),
		ParentHash:   hash(0xCD),
		Timestamp:    1234567890,
		LogCount:     uint32(len(logHashes)),
		ExecMsgCount: uint32(len(execMsgs)),
	}
	dataLen := blockRecordSize + len(logHashes)*logHashSize + len(execMsgs)*execMsgRecordSize
	encoded := make([]byte, dataLen)
	br.encodeInto(encoded[:blockRecordSize])
	for i, h := range logHashes {
		copy(encoded[hashesOffset+i*logHashSize:hashesOffset+(i+1)*logHashSize], h[:])
	}
	execOff := hashesOffset + len(logHashes)*logHashSize
	for i, em := range execMsgs {
		encodeExecMsgInto(encoded[execOff+i*execMsgRecordSize:execOff+(i+1)*execMsgRecordSize], em.localLogIdx, &em.msg)
	}

	got, err := decodeBlockRecord(encoded)
	require.NoError(t, err)
	require.Equal(t, br.Hash, got.Hash)
	require.Equal(t, br.ParentHash, got.ParentHash)
	require.Equal(t, br.Timestamp, got.Timestamp)
	require.Equal(t, br.LogCount, got.LogCount)
	require.Equal(t, br.ExecMsgCount, got.ExecMsgCount)

	for i, want := range logHashes {
		require.Equal(t, want, got.LogHash(uint32(i)))
	}
	for i, want := range execMsgs {
		gotIdx, gotEm := got.ExecMsg(uint32(i))
		require.Equal(t, want.localLogIdx, gotIdx)
		require.Equal(t, &want.msg, gotEm)
	}
}

func TestMultipleExecMsgsInBlock(t *testing.T) {
	// Exercise the SealBlock loop with M > 1: several logs carry executing
	// messages, interleaved with logs that don't. OpenBlock must return all of
	// them keyed by their local log index.
	db := tempDB(t)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))

	want := map[uint32]*messages.ExecutingMessage{}
	const logCount = 6
	for idx := uint32(0); idx < logCount; idx++ {
		var em *messages.ExecutingMessage
		// Logs 0, 2, 5 carry exec messages — non-contiguous slots.
		if idx == 0 || idx == 2 || idx == 5 {
			em = &messages.ExecutingMessage{
				ChainID:   eth.ChainIDFromUInt64(uint64(100 + idx)),
				BlockNum:  uint64(idx) + 1,
				LogIdx:    idx + 10,
				Timestamp: uint64(idx) * 7,
				Checksum:  messages.MessageChecksum(hash(byte(0xA0 + idx))),
			}
			want[idx] = em
		}
		require.NoError(t, db.AddLog(hash(byte(idx)), parent, idx, em))
	}
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 100))

	ref, count, msgs, err := db.OpenBlock(1)
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, ref.Hash)
	require.Equal(t, uint32(logCount), count)
	require.Equal(t, want, msgs)
}

func TestContains_LastIndexBoundary(t *testing.T) {
	// A Contains lookup at the highest valid logIdx hits the final 32 bytes of
	// the log-hash array exactly — exercises the off-by-one at the end of the
	// entry buffer.
	db := tempDB(t)
	chain := eth.ChainIDFromUInt64(10)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))

	const n = 4
	hashes := make([]common.Hash, n)
	for i := uint32(0); i < n; i++ {
		hashes[i] = hash(byte(0x10 + i))
		require.NoError(t, db.AddLog(hashes[i], parent, i, nil))
	}
	blk1 := blockID(1, 0x11)
	require.NoError(t, db.SealBlock(parent.Hash, blk1, 200))

	// Last valid logIdx is n-1.
	lastIdx := uint32(n - 1)
	good := messages.ChecksumArgs{
		BlockNumber: 1, LogIndex: lastIdx, Timestamp: 200, ChainID: chain, LogHash: hashes[lastIdx],
	}.Checksum()
	seal, err := db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: lastIdx, Timestamp: 200, Checksum: good})
	require.NoError(t, err)
	require.Equal(t, blk1.Hash, seal.Hash)

	// logIdx == n is one past the end and must be ErrConflict.
	_, err = db.Contains(messages.ContainsQuery{BlockNum: 1, LogIdx: n, Timestamp: 200, Checksum: good})
	require.ErrorIs(t, err, interop.ErrConflict)
}

func TestPersistence_AfterRewind(t *testing.T) {
	// Rewinding a non-zero-anchored DB and reopening must report the correct
	// firstBlock and latest from the trimmed WAL.
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	first := blockID(100, 0x64)
	require.NoError(t, db.SealBlock(common.Hash{}, first, 1000))
	last := sealRange(t, db, first, 101, 105)
	require.Equal(t, uint64(105), last.Number)

	target := blockID(102, 102)
	require.NoError(t, db.Rewind(target))
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()

	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, target, latest)

	firstSeal, err := db2.FirstSealedBlock()
	require.NoError(t, err)
	require.Equal(t, uint64(100), firstSeal.Number)
	require.Equal(t, first.Hash, firstSeal.Hash)

	// Trimmed blocks are gone.
	_, err = db2.FindSealedBlock(103)
	require.ErrorIs(t, err, interop.ErrFuture)

	// Surviving blocks are intact.
	for n := uint64(100); n <= 102; n++ {
		_, err := db2.FindSealedBlock(n)
		require.NoErrorf(t, err, "block %d should survive rewind", n)
	}
}

func TestPersistence_AfterClear(t *testing.T) {
	// Clearing a populated DB and reopening must come up empty, with no
	// orphaned segments from the prior incarnation.
	dir := t.TempDir()
	chain := eth.ChainIDFromUInt64(10)
	db, err := Open(dir, chain)
	require.NoError(t, err)
	parent := blockID(0, 0xA0)
	require.NoError(t, db.SealBlock(common.Hash{}, parent, 0))
	sealRange(t, db, parent, 1, 5)
	require.NoError(t, db.Clear())
	require.NoError(t, db.Close())

	db2, err := Open(dir, chain)
	require.NoError(t, err)
	defer db2.Close()
	_, ok := db2.LatestSealedBlock()
	require.False(t, ok)
	_, err = db2.FirstSealedBlock()
	require.ErrorIs(t, err, interop.ErrFuture)

	// And we can seal fresh blocks after reopen.
	fresh := blockID(42, 0x2A)
	require.NoError(t, db2.SealBlock(common.Hash{}, fresh, 4200))
	latest, ok := db2.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, fresh, latest)
}
