package logsdb

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	interoptypes "github.com/ethereum-optimism/optimism/op-core/interop/types"
	"github.com/ethereum-optimism/optimism/op-core/persistence/dberrors"
	"github.com/ethereum-optimism/optimism/op-core/persistence/entrydb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	searchCheckpointFrequency    = 256
	eventFlagHasExecutingMessage = byte(1)
)

var (
	errIteratorStoppedButNoSealedBlock = errors.New("iterator stopped but no sealed block found")
	errUnexpectedLogSkip               = errors.New("unexpected log-skip")
)

type Metrics interface {
	RecordDBEntryCount(kind string, count int64)
	RecordDBSearchEntriesRead(count int64)
}

// DB implements an append only database for log data and cross-chain dependencies.
type DB struct {
	log    log.Logger
	m      Metrics
	store  entrydb.EntryStore[EntryType, Entry]
	rwLock sync.RWMutex

	chainID eth.ChainID

	lastEntryContext logContext
}

func NewFromFile(logger log.Logger, m Metrics, chainID eth.ChainID, path string, trimToLastSealed bool) (*DB, error) {
	store, err := entrydb.NewEntryDB[EntryType, Entry, EntryBinary](logger, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	return NewFromEntryStore(logger, m, chainID, store, trimToLastSealed)
}

func NewFromEntryStore(logger log.Logger, m Metrics, chainID eth.ChainID, store entrydb.EntryStore[EntryType, Entry], trimToLastSealed bool) (*DB, error) {
	db := &DB{
		log:     logger,
		m:       m,
		store:   store,
		chainID: chainID,
	}
	if err := db.init(trimToLastSealed); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	return db, nil
}

func (db *DB) lastEntryIdx() entrydb.EntryIdx {
	return db.store.LastEntryIdx()
}

func (db *DB) init(trimToLastSealed bool) error {
	defer db.updateEntryCountMetric()
	if trimToLastSealed {
		if err := db.trimToLastSealed(); err != nil {
			return fmt.Errorf("failed to trim invalid trailing entries: %w", err)
		}
	}
	if db.lastEntryIdx() < 0 {
		db.lastEntryContext = logContext{
			nextEntryIndex: 0,
			blockHash:      common.Hash{},
			blockNum:       0,
			timestamp:      0,
			logsSince:      0,
			logHash:        common.Hash{},
			execMsg:        nil,
			out:            nil,
		}
		return nil
	}
	lastCheckpoint := (db.lastEntryIdx() / searchCheckpointFrequency) * searchCheckpointFrequency
	i := db.newIterator(lastCheckpoint)
	i.current.need.Add(FlagCanonicalHash)
	if err := i.End(); err != nil {
		return fmt.Errorf("failed to init from remaining trailing data: %w", err)
	}
	db.lastEntryContext = i.current
	return nil
}

func (db *DB) trimToLastSealed() error {
	i := db.lastEntryIdx()
	for ; i >= 0; i-- {
		entry, err := db.store.Read(i)
		if err != nil {
			return fmt.Errorf("failed to read %v to check for trailing entries: %w", i, err)
		}
		if entry.Type() == TypeCanonicalHash {
			break
		}
	}
	if i < db.lastEntryIdx() {
		db.log.Warn("Truncating unexpected trailing entries", "prev", db.lastEntryIdx(), "new", i)
		return db.store.Truncate(i)
	}
	return nil
}

func (db *DB) updateEntryCountMetric() {
	db.m.RecordDBEntryCount("log", db.store.Size())
}

func (db *DB) IsEmpty() bool {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	return db.lastEntryContext.nextEntryIndex == 0
}

func (db *DB) IteratorStartingAt(sealedNum uint64, logsSince uint32) (Iterator, error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	return db.newIteratorAt(sealedNum, logsSince)
}

func (db *DB) FindSealedBlock(number uint64) (seal interoptypes.BlockSeal, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	iter, err := db.newIteratorAt(number, 0)
	if errors.Is(err, dberrors.ErrFuture) {
		return interoptypes.BlockSeal{}, fmt.Errorf("block %d is not known yet: %w", number, dberrors.ErrFuture)
	} else if err != nil {
		return interoptypes.BlockSeal{}, fmt.Errorf("failed to find sealed block %d: %w", number, err)
	}
	h, n, ok := iter.SealedBlock()
	if !ok {
		panic("expected block")
	}
	if n != number {
		panic(fmt.Sprintf("found block seal %s %d does not match expected block number %d", h, n, number))
	}
	timestamp, ok := iter.SealedTimestamp()
	if !ok {
		panic("expected timestamp")
	}
	return interoptypes.BlockSeal{
		Hash:      h,
		Number:    n,
		Timestamp: timestamp,
	}, nil
}

func (db *DB) FirstSealedBlock() (seal interoptypes.BlockSeal, err error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	iter := db.newIterator(0)
	if err := iter.NextBlock(); err != nil {
		return interoptypes.BlockSeal{}, err
	}
	h, n, _ := iter.SealedBlock()
	t, _ := iter.SealedTimestamp()
	return interoptypes.BlockSeal{
		Hash:      h,
		Number:    n,
		Timestamp: t,
	}, nil
}

// OpenBlock returns the Executing Messages for the block at the given number.
func (db *DB) OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*interoptypes.ExecutingMessage, retErr error) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()

	if blockNum == 0 {
		seal, err := db.FirstSealedBlock()
		if err != nil {
			retErr = err
			return
		}
		if seal.Number != 0 {
			return eth.BlockRef{}, 0, nil, fmt.Errorf("looked for block 0 but got %s: %w", seal, dberrors.ErrSkipped)
		}
		ref = eth.BlockRef{
			Hash:       seal.Hash,
			Number:     seal.Number,
			ParentHash: common.Hash{},
			Time:       seal.Timestamp,
		}
		logCount = 0
		execMsgs = nil
		return
	}

	blockIter, err := db.newIteratorAt(blockNum-1, 0)
	if err != nil {
		retErr = err
		return
	}
	parentHash, _, ok := blockIter.SealedBlock()
	if ok {
		ref.ParentHash = parentHash
	}
	logCount = 0
	execMsgs = make(map[uint32]*interoptypes.ExecutingMessage, 0)
	retErr = blockIter.TraverseConditional(func(state IteratorState) error {
		_, logIndex, ok := state.InitMessage()
		if ok {
			logCount = logIndex + 1
		}
		if m := state.ExecMessage(); m != nil {
			execMsgs[logIndex] = m
		}
		h, n, ok := state.SealedBlock()
		if !ok {
			return nil
		}
		if n == blockNum {
			ref.Number = n
			ref.Hash = h
			ref.Time, _ = state.SealedTimestamp()
			return dberrors.ErrStop
		}
		if n > blockNum {
			return fmt.Errorf("expected to run into block %d, but did not find it, found %d: %w", blockNum, n, dberrors.ErrDataCorruption)
		}
		return nil
	})
	if errors.Is(retErr, dberrors.ErrStop) {
		retErr = nil
	}
	return
}

func (db *DB) LatestSealedBlock() (id eth.BlockID, ok bool) {
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	if db.lastEntryContext.nextEntryIndex == 0 {
		return eth.BlockID{}, false
	}
	if !db.lastEntryContext.hasCompleteBlock() {
		db.log.Debug("New block is already in progress", "num", db.lastEntryContext.blockNum)
	}
	return eth.BlockID{
		Hash:   db.lastEntryContext.blockHash,
		Number: db.lastEntryContext.blockNum,
	}, true
}

func (db *DB) Contains(query interoptypes.ContainsQuery) (interoptypes.BlockSeal, error) {
	blockNum, logIdx, timestamp := query.BlockNum, query.LogIdx, query.Timestamp
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()
	db.log.Trace("Checking for log", "blockNum", blockNum, "logIdx", logIdx)

	if db.lastEntryContext.hasCompleteBlock() && db.lastEntryContext.blockNum < blockNum {
		if db.lastEntryContext.timestamp > timestamp {
			return interoptypes.BlockSeal{}, dberrors.ErrConflict
		}
		return interoptypes.BlockSeal{}, dberrors.ErrFuture
	}

	entryLogHash, iter, err := db.findLogInfo(blockNum, logIdx)
	if err != nil {
		if errors.Is(err, dberrors.ErrFuture) && db.lastEntryContext.hasCompleteBlock() {
			return interoptypes.BlockSeal{}, dberrors.ErrConflict
		}
		return interoptypes.BlockSeal{}, err
	}
	db.log.Trace("Found initiatingEvent", "blockNum", blockNum, "logIdx", logIdx, "hash", entryLogHash)
	err = iter.TraverseConditional(func(state IteratorState) error {
		_, n, ok := state.SealedBlock()
		if !ok {
			return nil
		}
		if n == blockNum {
			return dberrors.ErrStop
		}
		if n > blockNum {
			return dberrors.ErrDataCorruption
		}
		return nil
	})
	if err == nil {
		panic("expected iterator to stop with error")
	}
	if errors.Is(err, dberrors.ErrStop) {
		h, n, ok := iter.SealedBlock()
		if !ok {
			return interoptypes.BlockSeal{}, errIteratorStoppedButNoSealedBlock
		}
		t, _ := iter.SealedTimestamp()
		if t != timestamp {
			return interoptypes.BlockSeal{}, fmt.Errorf("timestamp mismatch: expected %d, got %d %w", timestamp, t, dberrors.ErrConflict)
		}
		entryChecksum := interoptypes.ChecksumArgs{
			BlockNumber: n,
			LogIndex:    logIdx,
			Timestamp:   t,
			ChainID:     db.chainID,
			LogHash:     entryLogHash,
		}.Checksum()
		if entryChecksum != query.Checksum {
			return interoptypes.BlockSeal{}, fmt.Errorf("payload hash mismatch: expected %s, got %s %w", query.Checksum, entryChecksum, dberrors.ErrConflict)
		}
		return interoptypes.BlockSeal{
			Hash:      h,
			Number:    n,
			Timestamp: t,
		}, nil
	}
	return interoptypes.BlockSeal{}, err
}

func (db *DB) findLogInfo(blockNum uint64, logIdx uint32) (common.Hash, Iterator, error) {
	if blockNum == 0 {
		return common.Hash{}, nil, dberrors.ErrConflict
	}
	iter, err := db.newIteratorAt(blockNum-1, logIdx)
	if errors.Is(err, dberrors.ErrFuture) {
		db.log.Trace("Could not find log yet", "blockNum", blockNum, "logIdx", logIdx)
		return common.Hash{}, nil, err
	} else if err != nil {
		db.log.Error("Failed searching for log", "blockNum", blockNum, "logIdx", logIdx)
		return common.Hash{}, nil, err
	}
	if err := iter.NextInitMsg(); err != nil {
		return common.Hash{}, nil, fmt.Errorf("failed to read initiating message %d, on top of block %d: %w", logIdx, blockNum, err)
	}
	if _, x, ok := iter.SealedBlock(); !ok {
		panic("expected block")
	} else if x < blockNum-1 {
		panic(fmt.Sprintf("bug in newIteratorAt, expected to have found parent block %d but got %d", blockNum-1, x))
	} else if x > blockNum-1 {
		return common.Hash{}, nil, fmt.Errorf("log does not exist, found next block already: %w", dberrors.ErrConflict)
	}
	logHash, x, ok := iter.InitMessage()
	if !ok {
		panic("expected init message")
	} else if x != logIdx {
		panic(fmt.Sprintf("bug in newIteratorAt, expected to have found log %d but got %d", logIdx, x))
	}
	return logHash, iter, nil
}

func (db *DB) newIteratorAt(blockNum uint64, logIndex uint32) (*iterator, error) {
	searchCheckpointIndex, err := db.searchCheckpoint(blockNum, logIndex)
	if errors.Is(err, io.EOF) {
		return nil, dberrors.ErrFuture
	} else if err != nil {
		return nil, err
	}
	iter := db.newIterator(searchCheckpointIndex)
	iter.current.need.Add(FlagCanonicalHash)
	defer func() {
		db.m.RecordDBSearchEntriesRead(iter.entriesRead)
	}()
	for {
		if _, n, ok := iter.SealedBlock(); ok && n == blockNum {
			break
		}
		if err := iter.NextBlock(); errors.Is(err, dberrors.ErrFuture) {
			db.log.Trace("ran out of data, could not find block", "nextIndex", iter.NextIndex(), "target", blockNum)
			return nil, dberrors.ErrFuture
		} else if err != nil {
			db.log.Error("failed to read next block", "nextIndex", iter.NextIndex(), "target", blockNum, "err", err)
			return nil, err
		}
		h, num, ok := iter.SealedBlock()
		if !ok {
			panic("expected sealed block")
		}
		db.log.Trace("found sealed block", "num", num, "hash", h)
		if num < blockNum {
			continue
		}
		if num != blockNum {
			return nil, fmt.Errorf("looking for %d, but already at %d: %w", blockNum, num, dberrors.ErrConflict)
		}
		break
	}
	for iter.current.logsSince < logIndex {
		if err := iter.NextInitMsg(); err == io.EOF {
			return nil, dberrors.ErrFuture
		} else if err != nil {
			return nil, err
		}
		_, num, ok := iter.SealedBlock()
		if !ok {
			panic("expected sealed block")
		}
		if num > blockNum {
			return nil, dberrors.ErrConflict
		}
		_, idx, ok := iter.InitMessage()
		if !ok {
			panic("expected initializing message")
		}
		if idx+1 < logIndex {
			continue
		}
		if idx+1 == logIndex {
			break
		}
		return nil, fmt.Errorf("%w: at block %d log %d", errUnexpectedLogSkip, blockNum, idx)
	}
	return iter, nil
}

func (db *DB) newIterator(index entrydb.EntryIdx) *iterator {
	return &iterator{
		db: db,
		current: logContext{
			nextEntryIndex: index,
		},
	}
}

func (db *DB) searchCheckpoint(sealedBlockNum uint64, logsSince uint32) (entrydb.EntryIdx, error) {
	if db.lastEntryContext.nextEntryIndex == 0 {
		return 0, dberrors.ErrFuture
	}
	n := (db.lastEntryIdx() / searchCheckpointFrequency) + 1
	i, j := entrydb.EntryIdx(0), n
	for i+1 < j {
		h := entrydb.EntryIdx((uint64(i) + uint64(j)) >> 1)
		checkpoint, err := db.readSearchCheckpoint(h * searchCheckpointFrequency)
		if err != nil {
			return 0, fmt.Errorf("failed to read entry %v: %w", h, err)
		}
		if checkpoint.blockNum < sealedBlockNum ||
			(checkpoint.blockNum == sealedBlockNum && checkpoint.logsSince < logsSince) {
			i = h
		} else {
			j = h
		}
	}
	if i+1 != j {
		panic("expected to have 1 checkpoint left")
	}
	result := i * searchCheckpointFrequency
	checkpoint, err := db.readSearchCheckpoint(result)
	if err != nil {
		return 0, fmt.Errorf("failed to read final search checkpoint result: %w", err)
	}
	if checkpoint.blockNum > sealedBlockNum ||
		(checkpoint.blockNum == sealedBlockNum && checkpoint.logsSince > logsSince) {
		return 0, fmt.Errorf("missing data, earliest search checkpoint is %d with %d logs, cannot find something before or at %d with %d logs: %w",
			checkpoint.blockNum, checkpoint.logsSince, sealedBlockNum, logsSince, dberrors.ErrSkipped)
	}
	return result, nil
}

func (db *DB) debugTip() {
	for x := 0; x < 10; x++ {
		index := db.lastEntryIdx() - entrydb.EntryIdx(x)
		if index < 0 {
			continue
		}
		e, err := db.store.Read(index)
		if err == nil {
			db.log.Debug("tip", "index", index, "type", e.Type())
		}
	}
}

func (db *DB) flush() error {
	for i, e := range db.lastEntryContext.out {
		db.log.Trace("appending entry", "type", e.Type(), "entry", hexutil.Bytes(e[:]),
			"next", int(db.lastEntryContext.nextEntryIndex)-len(db.lastEntryContext.out)+i)
	}
	if err := db.store.Append(db.lastEntryContext.out...); err != nil {
		return fmt.Errorf("failed to append entries: %w", err)
	}
	db.lastEntryContext.out = db.lastEntryContext.out[:0]
	db.updateEntryCountMetric()
	return nil
}

func (db *DB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.lastEntryContext.SealBlock(parentHash, block, timestamp); err != nil {
		return fmt.Errorf("failed to seal block: %w", err)
	}
	db.log.Trace("Sealed block", "parent", parentHash, "block", block, "timestamp", timestamp)
	return db.flush()
}

func (db *DB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *interoptypes.ExecutingMessage) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()

	if err := db.lastEntryContext.ApplyLog(parentBlock, logIdx, logHash, execMsg); err != nil {
		return fmt.Errorf("failed to apply log: %w", err)
	}
	db.log.Trace("Applied log", "parentBlock", parentBlock, "logIndex", logIdx, "logHash", logHash, "executing", execMsg != nil)
	return db.flush()
}

// Clear clears the DB such that there is no data left.
// An invalidator is required as argument, to force users to invalidate any current open reads.
func (db *DB) Clear(inv Invalidator) error {
	release, invalidateErr := inv.TryInvalidate(InvalidationRules{
		DerivedInvalidation{Timestamp: 0},
	})
	if invalidateErr != nil {
		return invalidateErr
	}
	defer release()
	defer db.updateEntryCountMetric()
	if truncateErr := db.store.Truncate(-1); truncateErr != nil {
		return fmt.Errorf("failed to empty DB: %w", truncateErr)
	}
	db.lastEntryContext = logContext{}
	return nil
}

func (db *DB) Rewind(inv Invalidator, newHead eth.BlockID) error {
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	defer db.updateEntryCountMetric()
	iter, err := db.newIteratorAt(newHead.Number, 0)
	if err != nil {
		if errors.Is(err, dberrors.ErrPreviousToFirst) || errors.Is(err, dberrors.ErrSkipped) {
			if err := db.Clear(inv); err != nil {
				return fmt.Errorf("failed to clear logs DB, upon rewinding to log block %s before first block: %w", newHead, err)
			}
			return nil
		}
		return err
	}
	if hash, num, ok := iter.SealedBlock(); !ok {
		return fmt.Errorf("expected sealed block for rewind reference-point: %w", dberrors.ErrDataCorruption)
	} else if hash != newHead.Hash {
		return fmt.Errorf("cannot rewind to %s, have %s: %w", newHead, eth.BlockID{Hash: hash, Number: num}, dberrors.ErrConflict)
	}
	t, ok := iter.SealedTimestamp()
	if !ok {
		panic("expected timestamp in block seal")
	}
	release, err := inv.TryInvalidate(DerivedInvalidation{Timestamp: t})
	if err != nil {
		return err
	}
	defer release()
	if err := db.store.Truncate(iter.NextIndex() - 1); err != nil {
		return fmt.Errorf("failed to truncate to block %s: %w", newHead, err)
	}
	if err := db.init(true); err != nil {
		return fmt.Errorf("failed to find new last entry context: %w", err)
	}
	return nil
}

func (db *DB) readSearchCheckpoint(entryIdx entrydb.EntryIdx) (searchCheckpoint, error) {
	data, err := db.store.Read(entryIdx)
	if err != nil {
		return searchCheckpoint{}, fmt.Errorf("failed to read entry %v: %w", entryIdx, err)
	}
	return newSearchCheckpointFromEntry(data)
}

func (db *DB) Close() error {
	return db.store.Close()
}
