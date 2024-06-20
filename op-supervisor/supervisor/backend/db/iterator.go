package db

import (
	"fmt"
	"io"
)

type iterator struct {
	db           *DB
	nextEntryIdx int64
	blockNum     uint64
	logIdx       uint32

	entriesRead int64
}

func (i *iterator) NextLog() (blockNum uint64, logIdx uint32, evtHash TruncatedHash, outErr error) {
	for i.nextEntryIdx <= i.db.lastEntryIdx {
		entry, err := i.db.readEntry(i.nextEntryIdx)
		if err != nil {
			outErr = fmt.Errorf("failed to read entry %v: %w", i, err)
			return
		}
		i.nextEntryIdx++
		i.entriesRead++
		switch entry[0] {
		case typeSearchCheckpoint:
			current := i.db.parseSearchCheckpoint(entry)
			i.blockNum = current.blockNum
			i.logIdx = current.logIdx
		case typeCanonicalHash:
			// Skip
		case typeInitiatingEvent:
			blockNum, logIdx, evtHash = i.db.parseInitiatingEvent(i.blockNum, i.logIdx, entry)
			i.blockNum = blockNum
			i.logIdx = logIdx
			return
		case typeExecutingCheck:
		// TODO: Handle this properly
		case typeExecutingLink:
		// TODO: Handle this properly
		default:
			outErr = fmt.Errorf("unknown entry type %v", entry[0])
			return
		}
	}
	outErr = io.EOF
	return
}
