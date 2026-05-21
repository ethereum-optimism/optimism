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

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/raft"
	wal "github.com/hashicorp/raft-wal"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

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

type DB struct {
	mu      sync.RWMutex
	w       *wal.WAL
	chainID eth.ChainID

	pendingParent eth.BlockID
	pendingLogs   []pendingLog
	hasPending    bool

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
		return blockRecord{}, fmt.Errorf("%w: blockRecord: short buffer %d", types.ErrDataCorruption, len(buf))
	}
	var r blockRecord
	copy(r.Hash[:], buf[0:32])
	copy(r.ParentHash[:], buf[32:64])
	r.Timestamp = binary.BigEndian.Uint64(buf[64:72])
	r.LogCount = binary.BigEndian.Uint32(buf[72:76])
	r.ExecMsgCount = binary.BigEndian.Uint32(buf[76:80])

	expected := blockRecordSize + int(r.LogCount)*logHashSize + int(r.ExecMsgCount)*execMsgRecordSize
	if len(buf) != expected {
		return blockRecord{}, fmt.Errorf("%w: entry length %d, expected %d", types.ErrDataCorruption, len(buf), expected)
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
		d.hasBlocks = false
		return nil
	}
	rec, err := d.readBlockAt(last)
	if err != nil {
		return fmt.Errorf("read latest: %w", err)
	}
	d.firstBlock = blockNumFor(first)
	d.latest = eth.BlockID{Hash: rec.Hash, Number: blockNumFor(last)}
	d.latestTS = rec.Timestamp
	d.hasBlocks = true
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return eth.BlockID{}, false
	}
	return d.latest, true
}

func (d *DB) FirstSealedBlock() (messages.BlockSeal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return messages.BlockSeal{}, types.ErrFuture
	}
	rec, err := d.readBlockAt(indexFor(d.firstBlock))
	if err != nil {
		return messages.BlockSeal{}, err
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: d.firstBlock, Timestamp: rec.Timestamp}, nil
}

func (d *DB) FindSealedBlock(number uint64) (messages.BlockSeal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return messages.BlockSeal{}, types.ErrFuture
	}
	if number > d.latest.Number {
		return messages.BlockSeal{}, types.ErrFuture
	}
	if number < d.firstBlock {
		return messages.BlockSeal{}, types.ErrSkipped
	}
	rec, err := d.readBlockAt(indexFor(number))
	if err != nil {
		return messages.BlockSeal{}, err
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: number, Timestamp: rec.Timestamp}, nil
}

func (d *DB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*messages.ExecutingMessage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return eth.BlockRef{}, 0, nil, types.ErrFuture
	}
	if blockNum > d.latest.Number {
		return eth.BlockRef{}, 0, nil, types.ErrFuture
	}
	// Matches the old op-supervisor logs DB: the first sealed block is an
	// anchor with no resolvable parent state, so OpenBlock on it returns
	// ErrSkipped. Genesis (firstBlock == 0) is exempt since it is its own
	// anchor. Callers retrieve the anchor via FirstSealedBlock.
	if blockNum < d.firstBlock || (blockNum == d.firstBlock && d.firstBlock > 0) {
		return eth.BlockRef{}, 0, nil, types.ErrSkipped
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return messages.BlockSeal{}, types.ErrFuture
	}
	if query.BlockNum > d.latest.Number {
		if d.latestTS > query.Timestamp {
			return messages.BlockSeal{}, types.ErrConflict
		}
		return messages.BlockSeal{}, types.ErrFuture
	}
	if query.BlockNum < d.firstBlock {
		return messages.BlockSeal{}, types.ErrSkipped
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
		return messages.BlockSeal{}, types.ErrConflict
	}
	if rec.Timestamp != query.Timestamp {
		return messages.BlockSeal{}, fmt.Errorf("timestamp mismatch: expected %d, got %d: %w", query.Timestamp, rec.Timestamp, types.ErrConflict)
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
		return messages.BlockSeal{}, fmt.Errorf("checksum mismatch: %w", types.ErrConflict)
	}
	return messages.BlockSeal{Hash: rec.Hash, Number: query.BlockNum, Timestamp: rec.Timestamp}, nil
}

func (d *DB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *messages.ExecutingMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Genesis cannot carry logs: the EVM never executes the genesis block, so a
	// log against the zero BlockID is invalid from any legitimate writer.
	if parentBlock == (eth.BlockID{}) {
		return fmt.Errorf("%w: genesis does not have logs", types.ErrOutOfOrder)
	}

	if d.hasBlocks {
		if parentBlock != d.latest {
			return fmt.Errorf("%w: AddLog parent %s does not match latest sealed %s", types.ErrOutOfOrder, parentBlock, d.latest)
		}
	}
	if d.hasPending {
		if parentBlock != d.pendingParent {
			return fmt.Errorf("%w: AddLog parent %s does not match pending parent %s", types.ErrOutOfOrder, parentBlock, d.pendingParent)
		}
		if logIdx != uint32(len(d.pendingLogs)) {
			return fmt.Errorf("%w: AddLog index %d does not match expected %d", types.ErrOutOfOrder, logIdx, len(d.pendingLogs))
		}
	} else {
		if logIdx != 0 {
			return fmt.Errorf("%w: first AddLog of a block must have index 0, got %d", types.ErrOutOfOrder, logIdx)
		}
		d.pendingParent = parentBlock
		d.hasPending = true
	}
	d.pendingLogs = append(d.pendingLogs, pendingLog{hash: logHash, logIdx: logIdx, execMsg: execMsg})
	return nil
}

func (d *DB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.hasBlocks {
		if block.Number != d.latest.Number+1 {
			return fmt.Errorf("%w: SealBlock expected number %d, got %d", types.ErrConflict, d.latest.Number+1, block.Number)
		}
		if parentHash != d.latest.Hash {
			return fmt.Errorf("%w: SealBlock parent %s does not match latest %s", types.ErrConflict, parentHash, d.latest.Hash)
		}
		if timestamp < d.latestTS {
			return fmt.Errorf("%w: SealBlock timestamp %d before latest %d", types.ErrConflict, timestamp, d.latestTS)
		}
	}
	if d.hasPending {
		expectedParent := eth.BlockID{Hash: parentHash, Number: block.Number - 1}
		if d.pendingParent != expectedParent {
			return fmt.Errorf("%w: SealBlock parent %s does not match pending logs' parent %s", types.ErrConflict, expectedParent, d.pendingParent)
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

	if !d.hasBlocks {
		d.firstBlock = block.Number
	}
	d.latest = block
	d.latestTS = timestamp
	d.hasBlocks = true
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Rewind(newHead eth.BlockID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.hasBlocks || newHead.Number < d.firstBlock {
		return d.clearLocked()
	}
	if newHead.Number > d.latest.Number {
		return fmt.Errorf("%w: cannot rewind to %s, latest is %s", types.ErrFuture, newHead, d.latest)
	}

	rec, err := d.readBlockAt(indexFor(newHead.Number))
	if err != nil {
		return err
	}
	if rec.Hash != newHead.Hash {
		return fmt.Errorf("%w: rewind target %s does not match stored hash %s", types.ErrConflict, newHead.Hash, rec.Hash)
	}

	if newHead.Number < d.latest.Number {
		if err := d.w.DeleteRange(indexFor(newHead.Number+1), indexFor(d.latest.Number)); err != nil {
			return fmt.Errorf("failed to truncate raft-wal: %w", err)
		}
	}

	d.latest = newHead
	d.latestTS = rec.Timestamp
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Clear() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.clearLocked()
}

func (d *DB) clearLocked() error {
	if d.hasBlocks {
		if err := d.w.DeleteRange(indexFor(d.firstBlock), indexFor(d.latest.Number)); err != nil {
			return fmt.Errorf("failed to clear raft-wal: %w", err)
		}
	}
	d.hasBlocks = false
	d.latest = eth.BlockID{}
	d.latestTS = 0
	d.firstBlock = 0
	d.pendingLogs = d.pendingLogs[:0]
	d.hasPending = false
	d.pendingParent = eth.BlockID{}
	return nil
}

func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.w.Close()
}
