// Package raftwallogdb implements LogsDB on top of hashicorp/raft-wal.
//
// Each sealed block (block-record + all of its logs) is a single raft-wal
// entry. Entry index = block.Number + 1 (offset by 1 because raft-wal reserves
// index 0). StoreLog fsyncs the entry to disk before returning, so SealBlock
// is durable on return. AddLog buffers in memory and the entry is built on
// SealBlock — atomicity is therefore guaranteed by the single StoreLog call.
package raftwallogdb

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/raft"
	wal "github.com/hashicorp/raft-wal"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

// Entry layout for a sealed block (raft-wal stores all of this in a single Log entry):
//
//	[ 0..80) blockRecord:
//	    [ 0..32) hash
//	    [32..64) parentHash
//	    [64..72) timestamp     (uint64 BE)
//	    [72..76) logCount      (uint32 BE)
//	    [76..80) execMsgCount  (uint32 BE)
//	[80                 ..  80 + 32*N    ) logHashes[N]
//	[80 + 32*N          ..  80 + 32*N + 88*M) execMsgs[M], each 88 bytes:
//	    [ 0.. 4) localLogIdx   (uint32 BE)  // which slot in *this* block carries the executing message
//	    [ 4..36) chainID       (32-byte big-endian)
//	    [36..44) blockNum      (uint64 BE)  // initiating message's block on source chain
//	    [44..48) initLogIdx    (uint32 BE)  // initiating message's log index on source chain
//	    [48..56) timestamp     (uint64 BE)
//	    [56..88) checksum
//
// N = logCount, M = execMsgCount. Hashes are at a known fixed offset, so
// Contains is an O(1) memcpy + checksum. OpenBlock walks both arrays once.
const (
	blockRecordSize   = 80
	logHashSize       = 32
	execMsgRecordSize = 88
	hashesOffset      = blockRecordSize
)

// walStore is the subset of *wal.WAL the DB uses. Tests substitute a gated
// implementation to pin read/write concurrency.
type walStore interface {
	FirstIndex() (uint64, error)
	LastIndex() (uint64, error)
	GetLog(index uint64, log *raft.Log) error
	StoreLog(log *raft.Log) error
	DeleteRange(min, max uint64) error
	Close() error
}

// DB is safe for concurrent use. Writers are serialized internally; reads are
// answered from the last published seal and raft-wal's lock-free read path, so
// they never wait on an in-flight SealBlock (see
// ethereum-optimism/optimism#21943). Only Rewind, Clear, and Close exclude
// readers.
type DB struct {
	// writeMu serializes all mutators: AddLog, SealBlock, Rewind, Clear, Close.
	writeMu sync.Mutex
	// truncMu excludes readers during Rewind, Clear, and Close only.
	// Lock order: writeMu before truncMu.
	truncMu sync.RWMutex

	w       walStore
	chainID eth.ChainID

	// Pending (unsealed) block state; guarded by writeMu.
	pendingParent eth.BlockID
	pendingLogs   []pendingLog
	hasPending    bool

	// sealed is the atomically-published sealed-chain state. Writers replace it
	// under writeMu after the WAL write commits.
	sealed atomic.Pointer[sealedState]
}

type sealedState struct {
	latest     eth.BlockID
	latestTS   uint64
	hasBlocks  bool
	firstBlock uint64
}

type pendingLog struct {
	hash    common.Hash
	logIdx  uint32
	execMsg *messages.ExecutingMessage
}

type blockRecord struct {
	Hash         common.Hash
	ParentHash   common.Hash
	Timestamp    uint64
	LogCount     uint32
	ExecMsgCount uint32

	// hashes and execMsgs are sub-slices of the decoded entry buffer.
	// Nil on records built for encoding.
	hashes   []byte
	execMsgs []byte
}

func (r *blockRecord) encodeInto(buf []byte) {
	copy(buf[0:32], r.Hash[:])
	copy(buf[32:64], r.ParentHash[:])
	binary.BigEndian.PutUint64(buf[64:72], r.Timestamp)
	binary.BigEndian.PutUint32(buf[72:76], r.LogCount)
	binary.BigEndian.PutUint32(buf[76:80], r.ExecMsgCount)
}

// LogHash returns the log hash at slot i. Caller must ensure i < r.LogCount.
func (r *blockRecord) LogHash(i uint32) common.Hash {
	var h common.Hash
	off := int(i) * logHashSize
	copy(h[:], r.hashes[off:off+logHashSize])
	return h
}

// ExecMsg returns the i-th executing-message record as (localLogIdx, msg).
// Caller must ensure i < r.ExecMsgCount.
func (r *blockRecord) ExecMsg(i uint32) (uint32, *messages.ExecutingMessage) {
	off := int(i) * execMsgRecordSize
	return decodeExecMsg(r.execMsgs[off : off+execMsgRecordSize])
}

// decodeBlockRecord parses an entry and verifies its full length matches the
// header's declared counts.
func decodeBlockRecord(buf []byte) (blockRecord, error) {
	if len(buf) < blockRecordSize {
		return blockRecord{}, fmt.Errorf("%w: blockRecord: short buffer %d", interop.ErrDataCorruption, len(buf))
	}
	var r blockRecord
	copy(r.Hash[:], buf[0:32])
	copy(r.ParentHash[:], buf[32:64])
	r.Timestamp = binary.BigEndian.Uint64(buf[64:72])
	r.LogCount = binary.BigEndian.Uint32(buf[72:76])
	r.ExecMsgCount = binary.BigEndian.Uint32(buf[76:80])

	expected := blockRecordSize + int(r.LogCount)*logHashSize + int(r.ExecMsgCount)*execMsgRecordSize
	if len(buf) != expected {
		return blockRecord{}, fmt.Errorf("%w: entry length %d, expected %d", interop.ErrDataCorruption, len(buf), expected)
	}
	hashesEnd := hashesOffset + int(r.LogCount)*logHashSize
	r.hashes = buf[hashesOffset:hashesEnd]
	r.execMsgs = buf[hashesEnd:]
	return r, nil
}

// encodeExecMsgInto writes an 88-byte execMsg record (with embedded logIdx) to buf.
func encodeExecMsgInto(buf []byte, logIdx uint32, em *messages.ExecutingMessage) {
	binary.BigEndian.PutUint32(buf[0:4], logIdx)
	chainBytes := em.ChainID.Bytes32()
	copy(buf[4:36], chainBytes[:])
	binary.BigEndian.PutUint64(buf[36:44], em.BlockNum)
	binary.BigEndian.PutUint32(buf[44:48], em.LogIdx)
	binary.BigEndian.PutUint64(buf[48:56], em.Timestamp)
	copy(buf[56:88], em.Checksum[:])
}

// decodeExecMsg reads an 88-byte execMsg record and returns (logIdx, msg).
func decodeExecMsg(buf []byte) (uint32, *messages.ExecutingMessage) {
	logIdx := binary.BigEndian.Uint32(buf[0:4])
	var chainBytes [32]byte
	copy(chainBytes[:], buf[4:36])
	em := &messages.ExecutingMessage{
		ChainID:   eth.ChainIDFromBytes32(chainBytes),
		BlockNum:  binary.BigEndian.Uint64(buf[36:44]),
		LogIdx:    binary.BigEndian.Uint32(buf[44:48]),
		Timestamp: binary.BigEndian.Uint64(buf[48:56]),
	}
	copy(em.Checksum[:], buf[56:88])
	return logIdx, em
}

func indexFor(blockNum uint64) uint64 { return blockNum + 1 }
func blockNumFor(idx uint64) uint64   { return idx - 1 }

// Open opens or creates a raft-wal-backed LogsDB at dir.
func Open(dir string, chainID eth.ChainID) (*DB, error) {
	w, err := wal.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to open raft-wal at %s: %w", dir, err)
	}
	d := &DB{w: w, chainID: chainID}
	if err := d.refreshCache(); err != nil {
		_ = w.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) refreshCache() error {
	first, err := d.w.FirstIndex()
	if err != nil {
		return fmt.Errorf("FirstIndex: %w", err)
	}
	last, err := d.w.LastIndex()
	if err != nil {
		return fmt.Errorf("LastIndex: %w", err)
	}
	if first == 0 || last == 0 {
		d.sealed.Store(&sealedState{})
		return nil
	}
	rec, err := d.readBlockAt(last)
	if err != nil {
		return fmt.Errorf("read latest: %w", err)
	}
	s := &sealedState{
		latest:     eth.BlockID{Hash: rec.Hash, Number: blockNumFor(last)},
		latestTS:   rec.Timestamp,
		hasBlocks:  true,
		firstBlock: blockNumFor(first),
	}
	// COMPATIBILITY: hide a pre-#20726 virtual-parent head entry, if any.
	// Safe to delete with compat_virtual_parent.go once no such databases
	// remain in operation.
	if err := d.hidePreVirtualParentLayout(s); err != nil {
		return err
	}
	d.sealed.Store(s)
	return nil
}

// readBlockAt fetches the block record at the given raft-wal index.
func (d *DB) readBlockAt(idx uint64) (blockRecord, error) {
	var log raft.Log
	if err := d.w.GetLog(idx, &log); err != nil {
		return blockRecord{}, fmt.Errorf("GetLog(%d): %w", idx, err)
	}
	return decodeBlockRecord(log.Data)
}

func (d *DB) LatestSealedBlock() (eth.BlockID, bool) {
	s := d.sealed.Load()
	if !s.hasBlocks {
		return eth.BlockID{}, false
	}
	return s.latest, true
}

func (d *DB) FirstSealedBlock() (messages.BlockSeal, error) {
	d.truncMu.RLock()
	defer d.truncMu.RUnlock()
	s := d.sealed.Load()
	if !s.hasBlocks {
		return messages.BlockSeal{}, interop.ErrFuture
	}
	rec, err := d.readBlockAt(indexFor(s.firstBlock))
	if err != nil {
		return messages.BlockSeal{}, err
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: s.firstBlock, Timestamp: rec.Timestamp}, nil
}

func (d *DB) FindSealedBlock(number uint64) (messages.BlockSeal, error) {
	d.truncMu.RLock()
	defer d.truncMu.RUnlock()
	s := d.sealed.Load()
	if !s.hasBlocks {
		return messages.BlockSeal{}, interop.ErrFuture
	}
	if number > s.latest.Number {
		return messages.BlockSeal{}, interop.ErrFuture
	}
	if number < s.firstBlock {
		return messages.BlockSeal{}, interop.ErrSkipped
	}
	rec, err := d.readBlockAt(indexFor(number))
	if err != nil {
		return messages.BlockSeal{}, err
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: number, Timestamp: rec.Timestamp}, nil
}

func (d *DB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*messages.ExecutingMessage, error) {
	d.truncMu.RLock()
	defer d.truncMu.RUnlock()
	s := d.sealed.Load()
	if !s.hasBlocks {
		return eth.BlockRef{}, 0, nil, interop.ErrFuture
	}
	if blockNum > s.latest.Number {
		return eth.BlockRef{}, 0, nil, interop.ErrFuture
	}
	if blockNum < s.firstBlock {
		return eth.BlockRef{}, 0, nil, interop.ErrSkipped
	}
	var log raft.Log
	if err := d.w.GetLog(indexFor(blockNum), &log); err != nil {
		return eth.BlockRef{}, 0, nil, fmt.Errorf("GetLog(%d): %w", blockNum, err)
	}
	rec, err := decodeBlockRecord(log.Data)
	if err != nil {
		return eth.BlockRef{}, 0, nil, err
	}
	ref := eth.BlockRef{
		Hash:       rec.Hash,
		Number:     blockNum,
		ParentHash: rec.ParentHash,
		Time:       rec.Timestamp,
	}
	execMsgs := make(map[uint32]*messages.ExecutingMessage, rec.ExecMsgCount)
	for i := uint32(0); i < rec.ExecMsgCount; i++ {
		idx, em := rec.ExecMsg(i)
		execMsgs[idx] = em
	}
	return ref, rec.LogCount, execMsgs, nil
}

func (d *DB) Contains(query messages.ContainsQuery) (messages.BlockSeal, error) {
	d.truncMu.RLock()
	defer d.truncMu.RUnlock()
	s := d.sealed.Load()
	if !s.hasBlocks {
		return messages.BlockSeal{}, interop.ErrFuture
	}
	if query.BlockNum > s.latest.Number {
		if s.latestTS > query.Timestamp {
			return messages.BlockSeal{}, interop.ErrConflict
		}
		return messages.BlockSeal{}, interop.ErrFuture
	}
	if query.BlockNum < s.firstBlock {
		return messages.BlockSeal{}, interop.ErrSkipped
	}

	var log raft.Log
	if err := d.w.GetLog(indexFor(query.BlockNum), &log); err != nil {
		return messages.BlockSeal{}, fmt.Errorf("GetLog(%d): %w", query.BlockNum, err)
	}
	rec, err := decodeBlockRecord(log.Data)
	if err != nil {
		return messages.BlockSeal{}, err
	}
	if query.LogIdx >= rec.LogCount {
		return messages.BlockSeal{}, interop.ErrConflict
	}
	if rec.Timestamp != query.Timestamp {
		return messages.BlockSeal{}, fmt.Errorf("timestamp mismatch: expected %d, got %d: %w", query.Timestamp, rec.Timestamp, interop.ErrConflict)
	}

	logHash := rec.LogHash(query.LogIdx)
	expectedChecksum := messages.ChecksumArgs{
		BlockNumber: query.BlockNum,
		LogIndex:    query.LogIdx,
		Timestamp:   rec.Timestamp,
		ChainID:     d.chainID,
		LogHash:     logHash,
	}.Checksum()
	if expectedChecksum != query.Checksum {
		return messages.BlockSeal{}, fmt.Errorf("checksum mismatch: %w", interop.ErrConflict)
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: query.BlockNum, Timestamp: rec.Timestamp}, nil
}

func (d *DB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *messages.ExecutingMessage) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	// Genesis cannot carry logs: the EVM never executes the genesis block, so a
	// log against the zero BlockID is invalid from any legitimate writer.
	if parentBlock == (eth.BlockID{}) {
		return fmt.Errorf("%w: genesis does not have logs", interop.ErrOutOfOrder)
	}

	if s := d.sealed.Load(); s.hasBlocks {
		if parentBlock != s.latest {
			return fmt.Errorf("%w: AddLog parent %s does not match latest sealed %s", interop.ErrOutOfOrder, parentBlock, s.latest)
		}
	}
	if d.hasPending {
		if parentBlock != d.pendingParent {
			return fmt.Errorf("%w: AddLog parent %s does not match pending parent %s", interop.ErrOutOfOrder, parentBlock, d.pendingParent)
		}
		if logIdx != uint32(len(d.pendingLogs)) {
			return fmt.Errorf("%w: AddLog index %d does not match expected %d", interop.ErrOutOfOrder, logIdx, len(d.pendingLogs))
		}
	} else {
		if logIdx != 0 {
			return fmt.Errorf("%w: first AddLog of a block must have index 0, got %d", interop.ErrOutOfOrder, logIdx)
		}
		d.pendingParent = parentBlock
		d.hasPending = true
	}
	d.pendingLogs = append(d.pendingLogs, pendingLog{hash: logHash, logIdx: logIdx, execMsg: execMsg})
	return nil
}

func (d *DB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	s := d.sealed.Load()
	if s.hasBlocks {
		if block.Number != s.latest.Number+1 {
			return fmt.Errorf("%w: SealBlock expected number %d, got %d", interop.ErrConflict, s.latest.Number+1, block.Number)
		}
		if parentHash != s.latest.Hash {
			return fmt.Errorf("%w: SealBlock parent %s does not match latest %s", interop.ErrConflict, parentHash, s.latest.Hash)
		}
		if timestamp < s.latestTS {
			return fmt.Errorf("%w: SealBlock timestamp %d before latest %d", interop.ErrConflict, timestamp, s.latestTS)
		}
	}
	if d.hasPending {
		expectedParent := eth.BlockID{Hash: parentHash, Number: block.Number - 1}
		if d.pendingParent != expectedParent {
			return fmt.Errorf("%w: SealBlock parent %s does not match pending logs' parent %s", interop.ErrConflict, expectedParent, d.pendingParent)
		}
	}

	// Build the entry: blockRecord || logHashes || execMsgs.
	logCount := uint32(len(d.pendingLogs))
	execCount := uint32(0)
	for _, p := range d.pendingLogs {
		if p.execMsg != nil {
			execCount++
		}
	}
	dataLen := blockRecordSize + int(logCount)*logHashSize + int(execCount)*execMsgRecordSize
	data := make([]byte, dataLen)

	rec := blockRecord{
		Hash:         block.Hash,
		ParentHash:   parentHash,
		Timestamp:    timestamp,
		LogCount:     logCount,
		ExecMsgCount: execCount,
	}
	rec.encodeInto(data[:blockRecordSize])

	hashesEnd := hashesOffset + int(logCount)*logHashSize
	execOff := hashesEnd
	for i, p := range d.pendingLogs {
		copy(data[hashesOffset+i*logHashSize:hashesOffset+(i+1)*logHashSize], p.hash[:])
		if p.execMsg != nil {
			encodeExecMsgInto(data[execOff:execOff+execMsgRecordSize], p.logIdx, p.execMsg)
			execOff += execMsgRecordSize
		}
	}

	entry := &raft.Log{
		Index: indexFor(block.Number),
		Data:  data,
	}
	if err := d.w.StoreLog(entry); err != nil {
		return fmt.Errorf("failed to commit block seal: %w", err)
	}

	next := &sealedState{
		latest:     block,
		latestTS:   timestamp,
		hasBlocks:  true,
		firstBlock: s.firstBlock,
	}
	if !s.hasBlocks {
		next.firstBlock = block.Number
	}
	d.sealed.Store(next)
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Rewind(newHead eth.BlockID) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	s := d.sealed.Load()
	if !s.hasBlocks || newHead.Number < s.firstBlock {
		d.truncMu.Lock()
		defer d.truncMu.Unlock()
		return d.clearLocked(s)
	}
	if newHead.Number > s.latest.Number {
		return fmt.Errorf("%w: cannot rewind to %s, latest is %s", interop.ErrFuture, newHead, s.latest)
	}

	rec, err := d.readBlockAt(indexFor(newHead.Number))
	if err != nil {
		return err
	}
	if rec.Hash != newHead.Hash {
		return fmt.Errorf("%w: rewind target %s does not match stored hash %s", interop.ErrConflict, newHead.Hash, rec.Hash)
	}

	d.truncMu.Lock()
	defer d.truncMu.Unlock()
	if newHead.Number < s.latest.Number {
		if err := d.w.DeleteRange(indexFor(newHead.Number+1), indexFor(s.latest.Number)); err != nil {
			return fmt.Errorf("failed to truncate raft-wal: %w", err)
		}
	}

	d.sealed.Store(&sealedState{
		latest:     newHead,
		latestTS:   rec.Timestamp,
		hasBlocks:  true,
		firstBlock: s.firstBlock,
	})
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Clear() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	d.truncMu.Lock()
	defer d.truncMu.Unlock()
	return d.clearLocked(d.sealed.Load())
}

// clearLocked requires writeMu and truncMu to be held.
func (d *DB) clearLocked(s *sealedState) error {
	if s.hasBlocks {
		if err := d.w.DeleteRange(indexFor(s.firstBlock), indexFor(s.latest.Number)); err != nil {
			return fmt.Errorf("failed to clear raft-wal: %w", err)
		}
	}
	d.sealed.Store(&sealedState{})
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Close() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	d.truncMu.Lock()
	defer d.truncMu.Unlock()
	return d.w.Close()
}
