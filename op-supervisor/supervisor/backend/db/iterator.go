package db

import (
	"errors"
	"fmt"
	"io"
)

type iterator struct {
	db           *DB
	nextEntryIdx int64

	current    logContext
	hasExecMsg bool

	entriesRead int64
}

func (i *iterator) NextLog() (blockNum uint64, logIdx uint32, evtHash TruncatedHash, outErr error) {
	for i.nextEntryIdx <= i.db.lastEntryIdx() {
		entryIdx := i.nextEntryIdx
		entry, err := i.db.store.Read(entryIdx)
		if err != nil {
			outErr = fmt.Errorf("failed to read entry %v: %w", i, err)
			return
		}
		i.nextEntryIdx++
		i.entriesRead++
		i.hasExecMsg = false
		switch entry[0] {
		case typeSearchCheckpoint:
			current, err := newSearchCheckpointFromEntry(entry)
			if err != nil {
				outErr = fmt.Errorf("failed to parse search checkpoint at idx %v: %w", entryIdx, err)
				return
			}
			i.current.blockNum = current.blockNum
			i.current.logIdx = current.logIdx
		case typeInitiatingEvent:
			evt, err := newInitiatingEventFromEntry(entry)
			if err != nil {
				outErr = fmt.Errorf("failed to parse initiating event at idx %v: %w", entryIdx, err)
				return
			}
			i.current = evt.postContext(i.current)
			blockNum = i.current.blockNum
			logIdx = i.current.logIdx
			evtHash = evt.logHash
			i.hasExecMsg = evt.hasExecMsg
			return
		case typeCanonicalHash: // Skip
		case typeExecutingCheck: // Skip
		case typeExecutingLink: // Skip
		default:
			outErr = fmt.Errorf("unknown entry type at idx %v %v", entryIdx, entry[0])
			return
		}
	}
	outErr = io.EOF
	return
}

func (i *iterator) ExecMessage() (ExecutingMessage, error) {
	if !i.hasExecMsg {
		return ExecutingMessage{}, nil
	}
	// Look ahead to find the exec message info
	logEntryIdx := i.nextEntryIdx - 1
	execMsg, err := i.readExecMessage(logEntryIdx)
	if err != nil {
		return ExecutingMessage{}, fmt.Errorf("failed to read exec message for initiating event at %v: %w", logEntryIdx, err)
	}
	return execMsg, nil
}

func (i *iterator) readExecMessage(initEntryIdx int64) (ExecutingMessage, error) {
	linkIdx := initEntryIdx + 1
	if linkIdx%searchCheckpointFrequency == 0 {
		linkIdx += 2 // skip the search checkpoint and canonical hash entries
	}
	linkEntry, err := i.db.store.Read(linkIdx)
	if errors.Is(err, io.EOF) {
		return ExecutingMessage{}, fmt.Errorf("%w: missing expected executing link event at idx %v", ErrDataCorruption, linkIdx)
	} else if err != nil {
		return ExecutingMessage{}, fmt.Errorf("failed to read executing link event at idx %v: %w", linkIdx, err)
	}

	checkIdx := linkIdx + 1
	if checkIdx%searchCheckpointFrequency == 0 {
		checkIdx += 2 // skip the search checkpoint and canonical hash entries
	}
	checkEntry, err := i.db.store.Read(checkIdx)
	if errors.Is(err, io.EOF) {
		return ExecutingMessage{}, fmt.Errorf("%w: missing expected executing check event at idx %v", ErrDataCorruption, checkIdx)
	} else if err != nil {
		return ExecutingMessage{}, fmt.Errorf("failed to read executing check event at idx %v: %w", checkIdx, err)
	}
	return newExecutingMessageFromEntries(linkEntry, checkEntry)
}
