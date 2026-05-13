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
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/raft"
	wal "github.com/hashicorp/raft-wal"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
	execMsg *types.ExecutingMessage
}

type blockRecord struct {
	Hash         common.Hash
	ParentHash   common.Hash
	Timestamp    uint64
	LogCount     uint32
	ExecMsgCount uint32
}

func (r *blockRecord) encodeInto(buf []byte) {
	copy(buf[0:32], r.Hash[:])
	copy(buf[32:64], r.ParentHash[:])
	binary.BigEndian.PutUint64(buf[64:72], r.Timestamp)
	binary.BigEndian.PutUint32(buf[72:76], r.LogCount)
	binary.BigEndian.PutUint32(buf[76:80], r.ExecMsgCount)
}

func decodeBlockRecord(buf []byte) (blockRecord, error) {
	if len(buf) < blockRecordSize {
		return blockRecord{}, fmt.Errorf("blockRecord: short buffer %d", len(buf))
	}
	var r blockRecord
	copy(r.Hash[:], buf[0:32])
	copy(r.ParentHash[:], buf[32:64])
	r.Timestamp = binary.BigEndian.Uint64(buf[64:72])
	r.LogCount = binary.BigEndian.Uint32(buf[72:76])
	r.ExecMsgCount = binary.BigEndian.Uint32(buf[76:80])
	return r, nil
}

// encodeExecMsgInto writes an 88-byte execMsg record (with embedded logIdx) to buf.
func encodeExecMsgInto(buf []byte, logIdx uint32, em *types.ExecutingMessage) {
	binary.BigEndian.PutUint32(buf[0:4], logIdx)
	chainBytes := em.ChainID.Bytes32()
	copy(buf[4:36], chainBytes[:])
	binary.BigEndian.PutUint64(buf[36:44], em.BlockNum)
	binary.BigEndian.PutUint32(buf[44:48], em.LogIdx)
	binary.BigEndian.PutUint64(buf[48:56], em.Timestamp)
	copy(buf[56:88], em.Checksum[:])
}

// decodeExecMsg reads an 88-byte execMsg record and returns (logIdx, msg).
func decodeExecMsg(buf []byte) (uint32, *types.ExecutingMessage) {
	logIdx := binary.BigEndian.Uint32(buf[0:4])
	var chainBytes [32]byte
	copy(chainBytes[:], buf[4:36])
	em := &types.ExecutingMessage{
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

func errSealBlock(err error) error {
	return fmt.Errorf("failed to seal block: %w", err)
}

func errApplyLog(err error) error {
	return fmt.Errorf("failed to apply log: %w", err)
}

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

func (d *DB) FirstSealedBlock() (types.BlockSeal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return types.BlockSeal{}, types.ErrFuture
	}
	rec, err := d.readBlockAt(indexFor(d.firstBlock))
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
	}
	return types.BlockSeal{Hash: rec.Hash, Number: d.firstBlock, Timestamp: rec.Timestamp}, nil
}

func (d *DB) FindSealedBlock(number uint64) (types.BlockSeal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return types.BlockSeal{}, fmt.Errorf("block %d is not known yet: %w", number, types.ErrFuture)
	}
	if number > d.latest.Number {
		return types.BlockSeal{}, fmt.Errorf("block %d is not known yet: %w", number, types.ErrFuture)
	}
	if number < d.firstBlock {
		return types.BlockSeal{}, fmt.Errorf("failed to find sealed block %d: %w", number, types.ErrSkipped)
	}
	rec, err := d.readBlockAt(indexFor(number))
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
	}
	return types.BlockSeal{Hash: rec.Hash, Number: number, Timestamp: rec.Timestamp}, nil
}

func (d *DB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*types.ExecutingMessage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.hasBlocks {
		return eth.BlockRef{}, 0, nil, types.ErrFuture
	}
	if blockNum == 0 && d.firstBlock != 0 {
		rec, err := d.readBlockAt(indexFor(d.firstBlock))
		if err != nil {
			return eth.BlockRef{}, 0, nil, fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
		}
		first := types.BlockSeal{Hash: rec.Hash, Number: d.firstBlock, Timestamp: rec.Timestamp}
		return eth.BlockRef{}, 0, nil, fmt.Errorf("looked for block 0 but got %s: %w", first, types.ErrSkipped)
	}
	if blockNum > d.latest.Number {
		return eth.BlockRef{}, 0, nil, types.ErrFuture
	}
	if blockNum < d.firstBlock {
		return eth.BlockRef{}, 0, nil, types.ErrSkipped
	}
	if blockNum == d.firstBlock && d.firstBlock != 0 {
		return eth.BlockRef{}, 0, nil, fmt.Errorf("cannot open first non-zero block %d without parent block: %w", blockNum, types.ErrSkipped)
	}
	var log raft.Log
	if err := d.w.GetLog(indexFor(blockNum), &log); err != nil {
		return eth.BlockRef{}, 0, nil, fmt.Errorf("%w: GetLog(%d): %w", types.ErrDataCorruption, blockNum, err)
	}
	rec, err := decodeBlockRecord(log.Data)
	if err != nil {
		return eth.BlockRef{}, 0, nil, fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
	}
	ref := eth.BlockRef{
		Hash:       rec.Hash,
		Number:     blockNum,
		ParentHash: rec.ParentHash,
		Time:       rec.Timestamp,
	}
	execMsgs := make(map[uint32]*types.ExecutingMessage, rec.ExecMsgCount)
	execStart := hashesOffset + int(rec.LogCount)*logHashSize
	if len(log.Data) < execStart+int(rec.ExecMsgCount)*execMsgRecordSize {
		return eth.BlockRef{}, 0, nil, fmt.Errorf("%w: entry truncated", types.ErrDataCorruption)
	}
	for i := uint32(0); i < rec.ExecMsgCount; i++ {
		off := execStart + int(i)*execMsgRecordSize
		idx, em := decodeExecMsg(log.Data[off : off+execMsgRecordSize])
		execMsgs[idx] = em
	}
	return ref, rec.LogCount, execMsgs, nil
}

func (d *DB) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if query.BlockNum == 0 {
		return types.BlockSeal{}, types.ErrConflict
	}
	if !d.hasBlocks {
		return types.BlockSeal{}, types.ErrFuture
	}
	if query.BlockNum > d.latest.Number {
		if d.latestTS > query.Timestamp {
			return types.BlockSeal{}, types.ErrConflict
		}
		return types.BlockSeal{}, types.ErrFuture
	}
	if query.BlockNum < d.firstBlock {
		return types.BlockSeal{}, types.ErrSkipped
	}
	if query.BlockNum == d.firstBlock && d.firstBlock != 0 {
		return types.BlockSeal{}, fmt.Errorf("cannot search first non-zero block %d without parent block: %w", query.BlockNum, types.ErrSkipped)
	}

	var log raft.Log
	if err := d.w.GetLog(indexFor(query.BlockNum), &log); err != nil {
		return types.BlockSeal{}, fmt.Errorf("%w: GetLog(%d): %w", types.ErrDataCorruption, query.BlockNum, err)
	}
	rec, err := decodeBlockRecord(log.Data)
	if err != nil {
		return types.BlockSeal{}, fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
	}
	if query.LogIdx >= rec.LogCount {
		return types.BlockSeal{}, types.ErrConflict
	}
	if rec.Timestamp != query.Timestamp {
		return types.BlockSeal{}, fmt.Errorf("timestamp mismatch: expected %d, got %d %w", query.Timestamp, rec.Timestamp, types.ErrConflict)
	}

	// Direct O(1) lookup into the log-hash array.
	hashOff := hashesOffset + int(query.LogIdx)*logHashSize
	if hashOff+logHashSize > len(log.Data) {
		return types.BlockSeal{}, fmt.Errorf("%w: entry truncated", types.ErrDataCorruption)
	}
	var logHash common.Hash
	copy(logHash[:], log.Data[hashOff:hashOff+logHashSize])
	expectedChecksum := types.ChecksumArgs{
		BlockNumber: query.BlockNum,
		LogIndex:    query.LogIdx,
		Timestamp:   rec.Timestamp,
		ChainID:     d.chainID,
		LogHash:     logHash,
	}.Checksum()
	if expectedChecksum != query.Checksum {
		return types.BlockSeal{}, fmt.Errorf("payload hash mismatch: expected %s, got %s %w", query.Checksum, expectedChecksum, types.ErrConflict)
	}
	return types.BlockSeal{Hash: rec.Hash, Number: query.BlockNum, Timestamp: rec.Timestamp}, nil
}

func (d *DB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if parentBlock == (eth.BlockID{}) {
		return errApplyLog(fmt.Errorf("genesis does not have logs: %w", types.ErrOutOfOrder))
	}
	if !d.hasBlocks {
		if parentBlock.Hash != (common.Hash{}) {
			return errApplyLog(fmt.Errorf("%w: log builds on top of block %s, but have block %s", types.ErrOutOfOrder, parentBlock, common.Hash{}))
		}
		return errApplyLog(fmt.Errorf("%w: log builds on top of block %d, but have block %d", types.ErrOutOfOrder, parentBlock.Number, uint64(0)))
	}
	if d.hasBlocks {
		if parentBlock != d.latest {
			if parentBlock.Hash != d.latest.Hash {
				return errApplyLog(fmt.Errorf("%w: log builds on top of block %s, but have block %s", types.ErrOutOfOrder, parentBlock, d.latest.Hash))
			}
			return errApplyLog(fmt.Errorf("%w: log builds on top of block %d, but have block %d", types.ErrOutOfOrder, parentBlock.Number, d.latest.Number))
		}
	}
	if d.hasPending {
		if parentBlock != d.pendingParent {
			if parentBlock.Hash != d.pendingParent.Hash {
				return errApplyLog(fmt.Errorf("%w: log builds on top of block %s, but have block %s", types.ErrOutOfOrder, parentBlock, d.pendingParent.Hash))
			}
			return errApplyLog(fmt.Errorf("%w: log builds on top of block %d, but have block %d", types.ErrOutOfOrder, parentBlock.Number, d.pendingParent.Number))
		}
		if logIdx != uint32(len(d.pendingLogs)) {
			return errApplyLog(fmt.Errorf("%w: expected event index %d, cannot append %d", types.ErrOutOfOrder, len(d.pendingLogs), logIdx))
		}
	} else {
		if logIdx != 0 {
			return errApplyLog(fmt.Errorf("%w: expected event index 0, cannot append %d", types.ErrOutOfOrder, logIdx))
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
			return errSealBlock(fmt.Errorf("%w: cannot apply block %d on top of %d", types.ErrConflict, block.Number, d.latest.Number))
		}
		if parentHash != d.latest.Hash {
			return errSealBlock(fmt.Errorf("%w: cannot apply block %s (parent %s) on top of %s", types.ErrConflict, block, parentHash, d.latest.Hash))
		}
		if timestamp < d.latestTS {
			return errSealBlock(fmt.Errorf("%w: block timestamp %d must be equal or larger than current timestamp %d", types.ErrConflict, timestamp, d.latestTS))
		}
	}
	if d.hasPending {
		expectedParent := eth.BlockID{Hash: parentHash, Number: block.Number - 1}
		if d.pendingParent != expectedParent {
			return errSealBlock(fmt.Errorf("%w: cannot apply block %s (parent %s) on top of %s", types.ErrConflict, block, parentHash, d.pendingParent.Hash))
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

	if !d.hasBlocks {
		return types.ErrFuture
	}
	if newHead.Number < d.firstBlock {
		return d.clearLocked()
	}
	if newHead.Number > d.latest.Number {
		return fmt.Errorf("%w: cannot rewind to %s, latest is %s", types.ErrFuture, newHead, d.latest)
	}

	rec, err := d.readBlockAt(indexFor(newHead.Number))
	if err != nil {
		return fmt.Errorf("%w: %w", types.ErrDataCorruption, err)
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
		if err := d.w.DeleteRange(indexFor(d.firstBlock), indexFor(d.latest.Number)); err != nil && !errors.Is(err, raft.ErrLogNotFound) {
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
