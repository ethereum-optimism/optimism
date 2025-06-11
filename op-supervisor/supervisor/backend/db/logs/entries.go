package logs

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// searchCheckpoint is both a checkpoint for searching, as well as a checkpoint for sealing blocks.
type searchCheckpoint struct {
	blockNum uint64
	// seen logs *after* the seal of the mentioned block, i.e. not part of this block, but building on top of it.
	// There is at least one checkpoint per L2 block with logsSince == 0, i.e. the exact block boundary.
	logsSince uint32
	timestamp uint64
}

func newSearchCheckpoint(blockNum uint64, logsSince uint32, timestamp uint64) searchCheckpoint {
	return searchCheckpoint{
		blockNum:  blockNum,
		logsSince: logsSince,
		timestamp: timestamp,
	}
}

func newSearchCheckpointFromEntry(data Entry) (searchCheckpoint, error) {
	if data.Type() != TypeSearchCheckpoint {
		return searchCheckpoint{}, fmt.Errorf("%w: attempting to decode search checkpoint but was type %s", types.ErrDataCorruption, data.Type())
	}
	return searchCheckpoint{
		blockNum:  binary.LittleEndian.Uint64(data[1:9]),
		logsSince: binary.LittleEndian.Uint32(data[9:13]),
		timestamp: binary.LittleEndian.Uint64(data[13:21]),
	}, nil
}

// encode creates a checkpoint entry
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 logsSince count: 4 bytes><uint64 timestamp: 8 bytes> = 21 bytes
func (s searchCheckpoint) encode() Entry {
	var data Entry
	data[0] = uint8(TypeSearchCheckpoint)
	binary.LittleEndian.PutUint64(data[1:9], s.blockNum)
	binary.LittleEndian.PutUint32(data[9:13], s.logsSince)
	binary.LittleEndian.PutUint64(data[13:21], s.timestamp)
	return data
}

type canonicalHash struct {
	hash common.Hash
}

func newCanonicalHash(hash common.Hash) canonicalHash {
	return canonicalHash{hash: hash}
}

func newCanonicalHashFromEntry(data Entry) (canonicalHash, error) {
	if data.Type() != TypeCanonicalHash {
		return canonicalHash{}, fmt.Errorf("%w: attempting to decode canonical hash but was type %s", types.ErrDataCorruption, data.Type())
	}
	return newCanonicalHash(common.Hash(data[1:33])), nil
}

func (c canonicalHash) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeCanonicalHash)
	copy(entry[1:33], c.hash[:])
	return entry
}

type initiatingEvent struct {
	hasExecMsg bool
	logHash    common.Hash
}

func newInitiatingEventFromEntry(data Entry) (initiatingEvent, error) {
	if data.Type() != TypeInitiatingEvent {
		return initiatingEvent{}, fmt.Errorf("%w: attempting to decode initiating event but was type %s", types.ErrDataCorruption, data.Type())
	}
	flags := data[1]
	return initiatingEvent{
		hasExecMsg: flags&eventFlagHasExecutingMessage != 0,
		logHash:    common.Hash(data[2:34]),
	}, nil
}

func newInitiatingEvent(logHash common.Hash, hasExecMsg bool) initiatingEvent {
	return initiatingEvent{
		hasExecMsg: hasExecMsg,
		logHash:    logHash,
	}
}

// encode creates an initiating event entry
// type 2: "initiating event" <type><flags><event-hash: 20 bytes> = 22 bytes
func (i initiatingEvent) encode() Entry {
	var data Entry
	data[0] = uint8(TypeInitiatingEvent)
	flags := byte(0)
	if i.hasExecMsg {
		flags = flags | eventFlagHasExecutingMessage
	}
	data[1] = flags
	copy(data[2:34], i.logHash[:])
	return data
}

type execChainID struct {
	chainID eth.ChainID
}

func newExecChainID(msg types.ExecutingMessage) (execChainID, error) {
	return execChainID{
		chainID: msg.ChainID,
	}, nil
}

func newExecChainIDFromEntry(data Entry) (execChainID, error) {
	if data.Type() != TypeExecChainID {
		return execChainID{}, fmt.Errorf("%w: attempting to decode execChainID but was type %s", types.ErrDataCorruption, data.Type())
	}
	return execChainID{
		chainID: eth.ChainIDFromBytes32([32]byte(data[1:33])),
	}, nil
}

// encode creates an execChainID entry.
// type 3: "execChainID" <type><chainID: 32 bytes> = 33 bytes
func (e execChainID) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeExecChainID)
	id := e.chainID.Bytes32()
	copy(entry[1:33], id[:])
	return entry
}

type execPosition struct {
	blockNum  uint64
	logIdx    uint32
	timestamp uint64
}

func newExecPosition(msg types.ExecutingMessage) (execPosition, error) {
	return execPosition{
		blockNum:  msg.BlockNum,
		logIdx:    msg.LogIdx,
		timestamp: msg.Timestamp,
	}, nil
}

func newExecPositionFromEntry(data Entry) (execPosition, error) {
	if data.Type() != TypeExecPosition {
		return execPosition{}, fmt.Errorf("%w: attempting to decode execPosition but was type %s", types.ErrDataCorruption, data.Type())
	}
	return execPosition{
		blockNum:  binary.LittleEndian.Uint64(data[1:9]),
		logIdx:    binary.LittleEndian.Uint32(data[9:13]),
		timestamp: binary.LittleEndian.Uint64(data[13:21]),
	}, nil
}

// encode creates an execPosition entry.
// type 4: "execPosition" <type><blocknum: 8 bytes><event index: 4 bytes><uint64 timestamp: 8 bytes> = 21 bytes
func (e execPosition) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeExecPosition)
	binary.LittleEndian.PutUint64(entry[1:9], e.blockNum)
	binary.LittleEndian.PutUint32(entry[9:13], e.logIdx)
	binary.LittleEndian.PutUint64(entry[13:21], e.timestamp)
	return entry
}

type execChecksum struct {
	checksum types.MessageChecksum
}

func newExecChecksum(checksum types.MessageChecksum) execChecksum {
	return execChecksum{checksum: checksum}
}

func newExecChecksumFromEntry(data Entry) (execChecksum, error) {
	if data.Type() != TypeExecChecksum {
		return execChecksum{}, fmt.Errorf("%w: attempting to decode execChecksum but was type %s", types.ErrDataCorruption, data.Type())
	}
	return newExecChecksum(types.MessageChecksum(data[1:33])), nil
}

// encode creates an executing check entry
// type 5: "execChecksum" <type><event-hash: 32 bytes> = 33 bytes
func (e execChecksum) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeExecChecksum)
	copy(entry[1:33], e.checksum[:])
	return entry
}

type paddingEntry struct{}

// encoding of the padding entry
// type 6: "padding" <type><padding: 33 bytes> = 34 bytes
func (e paddingEntry) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypePadding)
	return entry
}
