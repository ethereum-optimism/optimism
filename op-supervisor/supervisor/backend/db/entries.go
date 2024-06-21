package db

import (
	"fmt"
	"math"
)

// createInitiatingEvent creates an initiating event
// type 2: "initiating event" <type><blocknum diff: 1 byte><event flags: 1 byte><event-hash: 20 bytes> = 23 bytes
func createInitiatingEvent(pre logContext, post logContext, logHash TruncatedHash) ([entrySize]byte, error) {
	var entry [entrySize]byte
	entry[0] = typeInitiatingEvent
	blockDiff := post.blockNum - pre.blockNum
	if blockDiff > math.MaxUint8 {
		// TODO(optimism#10857): Need to find a way to support this.
		return entry, fmt.Errorf("too many block skipped between %v and %v", pre.blockNum, post.blockNum)
	}
	entry[1] = byte(blockDiff)
	currLogIdx := pre.logIdx
	if blockDiff > 0 {
		currLogIdx = 0
	}
	flags := byte(0)
	logDiff := post.logIdx - currLogIdx
	if logDiff > 1 {
		return entry, fmt.Errorf("skipped logs between %v and %v", currLogIdx, post.logIdx)
	}
	if logDiff > 0 {
		// Set flag to indicate log idx needs to be incremented (ie we're not directly after a checkpoint)
		flags = flags | eventFlagIncrementLogIdx
	}
	entry[2] = flags
	copy(entry[3:23], logHash[:])
	return entry, nil
}

func parseInitiatingEvent(pre logContext, entry [entrySize]byte) (logContext, TruncatedHash) {
	blockNumDiff := entry[1]
	flags := entry[2]
	blockNum := pre.blockNum + uint64(blockNumDiff)
	logIdx := pre.logIdx
	if blockNumDiff > 0 {
		logIdx = 0
	}
	if flags&0x01 != 0 {
		logIdx++
	}
	eventHash := TruncatedHash(entry[3:23])
	return logContext{
		blockNum: blockNum,
		logIdx:   logIdx,
	}, eventHash
}
