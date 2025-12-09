package logsdb

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	interoptypes "github.com/ethereum-optimism/optimism/op-core/interop/types"
	"github.com/ethereum-optimism/optimism/op-core/persistence/dberrors"
	"github.com/ethereum-optimism/optimism/op-core/persistence/entrydb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	errNotReadyForCanonicalHash = errors.New("not ready for canonical hash entry, already sealed the last block")
	errLogBeforeLastLogComplete = errors.New("cannot process log before last log completes")
	errUnexpectedExecChainID    = errors.New("unexpected execChainID")
	errUnexpectedExecPosition   = errors.New("unexpected execPosition")
	errUnexpectedExecChecksum   = errors.New("unexpected execChecksum")
	errNeedAppliedExecChainID   = errors.New("need execChainID to be applied")
	errNeedAppliedExecPosition  = errors.New("need execPosition to be applied")
	errUnknownEntryType         = errors.New("unknown entry type")
	errIncompleteBlock          = errors.New("incomplete block")
)

// logContext is a buffer on top of the DB
type logContext struct {
	nextEntryIndex entrydb.EntryIdx
	blockHash      common.Hash
	blockNum       uint64
	timestamp      uint64
	logsSince      uint32
	logHash        common.Hash
	execMsg        *interoptypes.ExecutingMessage
	need           EntryTypeFlag
	out            []Entry
}

func (l *logContext) NextIndex() entrydb.EntryIdx {
	return l.nextEntryIndex
}

func (l *logContext) SealedBlock() (hash common.Hash, num uint64, ok bool) {
	if !l.hasCompleteBlock() {
		return common.Hash{}, 0, false
	}
	return l.blockHash, l.blockNum, true
}

func (l *logContext) SealedTimestamp() (timestamp uint64, ok bool) {
	if !l.hasCompleteBlock() {
		return 0, false
	}
	return l.timestamp, true
}

func (l *logContext) hasCompleteBlock() bool {
	return !l.need.Any(FlagCanonicalHash)
}

func (l *logContext) hasIncompleteLog() bool {
	return l.need.Any(FlagInitiatingEvent | FlagExecChainID | FlagExecPosition | FlagExecChecksum)
}

func (l *logContext) hasReadableLog() bool {
	return l.logsSince > 0 && !l.hasIncompleteLog()
}

func (l *logContext) InitMessage() (hash common.Hash, logIndex uint32, ok bool) {
	if !l.hasReadableLog() {
		return common.Hash{}, 0, false
	}
	return l.logHash, l.logsSince - 1, true
}

func (l *logContext) ExecMessage() *interoptypes.ExecutingMessage {
	if l.hasCompleteBlock() && l.hasReadableLog() && l.execMsg != nil {
		return l.execMsg
	}
	return nil
}

func (l *logContext) ApplyEntry(entry Entry) error {
	err := l.processEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to process type %s entry at idx %d (%x): %w", entry.Type().String(), l.nextEntryIndex, entry[:], err)
	}
	return nil
}

func (l *logContext) processEntry(entry Entry) error {
	if len(l.out) != 0 {
		panic("can only apply without appending if the state is still empty")
	}
	switch entry.Type() {
	case TypeSearchCheckpoint:
		current, err := newSearchCheckpointFromEntry(entry)
		if err != nil {
			return err
		}
		l.blockNum = current.blockNum
		l.blockHash = common.Hash{}
		l.logsSince = current.logsSince
		l.timestamp = current.timestamp
		l.need.Add(FlagCanonicalHash)
		if l.logsSince == 0 {
			l.logHash = common.Hash{}
			l.execMsg = nil
		}
	case TypeCanonicalHash:
		if !l.need.Any(FlagCanonicalHash) {
			return errNotReadyForCanonicalHash
		}
		canonHash, err := newCanonicalHashFromEntry(entry)
		if err != nil {
			return err
		}
		l.blockHash = canonHash.hash
		l.need.Remove(FlagCanonicalHash)
	case TypeInitiatingEvent:
		if !l.hasCompleteBlock() {
			return fmt.Errorf("%w: cannot append log before last known block is sealed", errIncompleteBlock)
		}
		if l.hasIncompleteLog() {
			return errLogBeforeLastLogComplete
		}
		evt, err := newInitiatingEventFromEntry(entry)
		if err != nil {
			return err
		}
		l.execMsg = nil
		l.logHash = evt.logHash
		if evt.hasExecMsg {
			l.need.Add(FlagExecChainID)
		} else {
			l.logsSince += 1
		}
		l.need.Remove(FlagInitiatingEvent)
	case TypeExecChainID:
		if !l.need.Any(FlagExecChainID) {
			return errUnexpectedExecChainID
		}
		idEntry, err := newExecChainIDFromEntry(entry)
		if err != nil {
			return err
		}
		l.execMsg = &interoptypes.ExecutingMessage{
			ChainID:   idEntry.chainID,
			BlockNum:  0,
			LogIdx:    0,
			Timestamp: 0,
			Checksum:  interoptypes.MessageChecksum{},
		}
		l.need.Remove(FlagExecChainID)
		l.need.Add(FlagExecPosition)
	case TypeExecPosition:
		if l.need.Any(FlagExecChainID) {
			return errNeedAppliedExecChainID
		}
		if !l.need.Any(FlagExecPosition) {
			return errUnexpectedExecPosition
		}
		posEntry, err := newExecPositionFromEntry(entry)
		if err != nil {
			return err
		}
		l.execMsg.BlockNum = posEntry.blockNum
		l.execMsg.LogIdx = posEntry.logIdx
		l.execMsg.Timestamp = posEntry.timestamp
		l.need.Remove(FlagExecPosition)
		l.need.Add(FlagExecChecksum)
	case TypeExecChecksum:
		if l.need.Any(FlagExecPosition) {
			return errNeedAppliedExecPosition
		}
		if !l.need.Any(FlagExecChecksum) {
			return errUnexpectedExecChecksum
		}
		checkEntry, err := newExecChecksumFromEntry(entry)
		if err != nil {
			return err
		}
		l.execMsg.Checksum = checkEntry.checksum
		l.need.Remove(FlagExecChecksum)
		l.logsSince += 1
	case TypePadding:
		if l.need.Any(FlagPadding) {
			l.need.Remove(FlagPadding)
		} else if l.need.Any(FlagPadding2) {
			l.need.Remove(FlagPadding2)
		} else {
			l.need.Remove(FlagPadding3)
		}
	default:
		return fmt.Errorf("%w: %s", errUnknownEntryType, entry.Type())
	}
	l.nextEntryIndex += 1
	return nil
}

func (l *logContext) appendEntry(obj EntryObj) {
	entry := obj.encode()
	l.out = append(l.out, entry)
	l.nextEntryIndex += 1
}

func (l *logContext) infer() error {
	if l.nextEntryIndex%searchCheckpointFrequency == 0 {
		l.need.Add(FlagSearchCheckpoint)
	}
	if l.need.Any(FlagSearchCheckpoint) {
		l.appendEntry(newSearchCheckpoint(l.blockNum, l.logsSince, l.timestamp))
		l.need.Add(FlagCanonicalHash)
		l.need.Remove(FlagSearchCheckpoint)
		return nil
	}
	if l.need.Any(FlagCanonicalHash) {
		l.appendEntry(newCanonicalHash(l.blockHash))
		l.need.Remove(FlagCanonicalHash)
		return nil
	}
	if l.need.Any(FlagPadding) {
		l.appendEntry(paddingEntry{})
		l.need.Remove(FlagPadding)
		return nil
	}
	if l.need.Any(FlagPadding2) {
		l.appendEntry(paddingEntry{})
		l.need.Remove(FlagPadding2)
		return nil
	}
	if l.need.Any(FlagPadding3) {
		l.appendEntry(paddingEntry{})
		l.need.Remove(FlagPadding3)
		return nil
	}
	if l.need.Any(FlagInitiatingEvent) {
		if l.execMsg != nil {
			switch l.nextEntryIndex % searchCheckpointFrequency {
			case searchCheckpointFrequency - 1:
				l.need.Add(FlagPadding)
				return nil
			case searchCheckpointFrequency - 2:
				l.need.Add(FlagPadding | FlagPadding2)
				return nil
			case searchCheckpointFrequency - 3:
				l.need.Add(FlagPadding | FlagPadding2 | FlagPadding3)
				return nil
			}
		}
		evt := newInitiatingEvent(l.logHash, l.execMsg != nil)
		l.appendEntry(evt)
		l.need.Remove(FlagInitiatingEvent)
		if l.execMsg == nil {
			l.logsSince += 1
		}
		return nil
	}
	if l.need.Any(FlagExecChainID) {
		chainIDEntry, err := newExecChainID(*l.execMsg)
		if err != nil {
			return fmt.Errorf("failed to create execChainID: %w", err)
		}
		l.appendEntry(chainIDEntry)
		l.need.Remove(FlagExecChainID)
		l.need.Add(FlagExecPosition)
		return nil
	}
	if l.need.Any(FlagExecPosition) {
		posEntry, err := newExecPosition(*l.execMsg)
		if err != nil {
			return fmt.Errorf("failed to create execPosition: %w", err)
		}
		l.appendEntry(posEntry)
		l.need.Remove(FlagExecPosition)
		l.need.Add(FlagExecChecksum)
		return nil
	}
	if l.need.Any(FlagExecChecksum) {
		l.appendEntry(newExecChecksum(l.execMsg.Checksum))
		l.need.Remove(FlagExecChecksum)
		l.logsSince += 1
		return nil
	}
	return io.EOF
}

func (l *logContext) inferFull() error {
	for i := 0; i < 20; i++ {
		err := l.infer()
		if err == nil {
			continue
		}
		if err == io.EOF {
			return nil
		} else {
			return err
		}
	}
	panic("hit sanity limit")
}

func (l *logContext) forceBlock(upd eth.BlockID, timestamp uint64) error {
	// Only valid on empty state
	l.blockHash = upd.Hash
	l.blockNum = upd.Number
	l.timestamp = timestamp
	l.logsSince = 0
	l.execMsg = nil
	l.logHash = common.Hash{}
	l.need = 0
	l.out = nil
	return l.inferFull()
}

func (l *logContext) SealBlock(parent common.Hash, upd eth.BlockID, timestamp uint64) error {
	if l.nextEntryIndex != 0 {
		if err := l.inferFull(); err != nil {
			return err
		}
		if l.blockHash != parent {
			return fmt.Errorf("%w: cannot apply block %s (parent %s) on top of %s", dberrors.ErrConflict, upd, parent, l.blockHash)
		}
		if l.blockHash != (common.Hash{}) && l.blockNum+1 != upd.Number {
			return fmt.Errorf("%w: cannot apply block %d on top of %d", dberrors.ErrConflict, upd.Number, l.blockNum)
		}
		if l.timestamp > timestamp {
			return fmt.Errorf("%w: block timestamp %d must be equal or larger than current timestamp %d", dberrors.ErrConflict, timestamp, l.timestamp)
		}
	}
	l.blockHash = upd.Hash
	l.blockNum = upd.Number
	l.timestamp = timestamp
	l.logsSince = 0
	l.execMsg = nil
	l.logHash = common.Hash{}
	l.need.Add(FlagSearchCheckpoint)
	return l.inferFull()
}

func (l *logContext) ApplyLog(parentBlock eth.BlockID, logIdx uint32, logHash common.Hash, execMsg *interoptypes.ExecutingMessage) error {
	if parentBlock == (eth.BlockID{}) {
		return fmt.Errorf("genesis does not have logs: %w", dberrors.ErrOutOfOrder)
	}
	if err := l.inferFull(); err != nil {
		return err
	}
	if !l.hasCompleteBlock() {
		if l.blockNum == 0 {
			return fmt.Errorf("%w: should not have logs in block 0", dberrors.ErrOutOfOrder)
		} else {
			return fmt.Errorf("%w: cannot append log before last known block is sealed", errIncompleteBlock)
		}
	}
	if l.blockHash != parentBlock.Hash {
		return fmt.Errorf("%w: log builds on top of block %s, but have block %s", dberrors.ErrOutOfOrder, parentBlock, l.blockHash)
	}
	if l.blockNum != parentBlock.Number {
		return fmt.Errorf("%w: log builds on top of block %d, but have block %d", dberrors.ErrOutOfOrder, parentBlock.Number, l.blockNum)
	}
	if logIdx != l.logsSince {
		return fmt.Errorf("%w: expected event index %d, cannot append %d", dberrors.ErrOutOfOrder, l.logsSince, logIdx)
	}
	l.logHash = logHash
	l.execMsg = execMsg
	l.need.Add(FlagInitiatingEvent)
	if execMsg != nil {
		l.need.Add(FlagExecChainID)
	}
	return l.inferFull()
}
