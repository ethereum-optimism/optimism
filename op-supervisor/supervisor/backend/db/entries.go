package db

import (
	"encoding/binary"
	"fmt"
	"math"
)

// createSearchCheckpoint creates a search checkpoint entry
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 event index offset: 4 bytes><uint64 timestamp: 8 bytes> = 20 bytes
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func createSearchCheckpoint(blockNum uint64, logIdx uint32, timestamp uint64) [entrySize]byte {
	var entry [entrySize]byte
	entry[0] = typeSearchCheckpoint
	binary.LittleEndian.PutUint64(entry[1:9], blockNum)
	binary.LittleEndian.PutUint32(entry[9:13], logIdx)
	binary.LittleEndian.PutUint64(entry[13:21], timestamp)
	return entry
}

func parseSearchCheckpoint(entry [entrySize]byte) (checkpointData, error) {
	if entry[0] != typeSearchCheckpoint {
		return checkpointData{}, fmt.Errorf("%w: attempting to decode search checkpoint but was type %v", ErrDataCorruption, entry[0])
	}
	return checkpointData{
		blockNum:  binary.LittleEndian.Uint64(entry[1:9]),
		logIdx:    binary.LittleEndian.Uint32(entry[9:13]),
		timestamp: binary.LittleEndian.Uint64(entry[13:21]),
	}, nil
}

// createCanonicalHash creates a canonical hash entry
// type 1: "canonical hash" <type><parent blockhash truncated: 20 bytes> = 21 bytes
func createCanonicalHash(hash TruncatedHash) [entrySize]byte {
	var entry [entrySize]byte
	entry[0] = typeCanonicalHash
	copy(entry[1:21], hash[:])
	return entry
}

func parseCanonicalHash(entry [entrySize]byte) (TruncatedHash, error) {
	if entry[0] != typeCanonicalHash {
		return TruncatedHash{}, fmt.Errorf("%w: attempting to decode canonical hash but was type %v", ErrDataCorruption, entry[0])
	}
	var truncated TruncatedHash
	copy(truncated[:], entry[1:21])
	return truncated, nil
}

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

func parseInitiatingEvent(pre logContext, entry [entrySize]byte) (logContext, TruncatedHash, error) {
	if entry[0] != typeInitiatingEvent {
		return logContext{}, TruncatedHash{}, fmt.Errorf("%w: attempting to decode initiating event but was type %v", ErrDataCorruption, entry[0])
	}
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
	logCtx := logContext{
		blockNum: blockNum,
		logIdx:   logIdx,
	}
	return logCtx, eventHash, nil
}
