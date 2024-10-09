package fromda

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
)

// searchCheckpoint is both a checkpoint for searching, as well as a checkpoint for sealing blocks.
type searchCheckpoint struct {
	// number of the L1 block that we derived from
	blockNum uint64
	// timestamp of the L1 block that was derived from
	timestamp uint64
	// number of L2 blocks that were derived after this checkpoint
	derivedSince uint32
	// L2 block that we last derived until starting deriving from this L1 block
	derivedUntil uint64
}

func newSearchCheckpoint(blockNum uint64, timestamp uint64, blocksSince uint32, derivedUntil uint64) searchCheckpoint {
	return searchCheckpoint{
		blockNum:     blockNum,
		timestamp:    timestamp,
		derivedSince: blocksSince,
		derivedUntil: derivedUntil,
	}
}

func newSearchCheckpointFromEntry(data Entry) (searchCheckpoint, error) {
	if data.Type() != TypeSearchCheckpoint {
		return searchCheckpoint{}, fmt.Errorf("%w: attempting to decode search checkpoint but was type %s", entrydb.ErrDataCorruption, data.Type())
	}
	return searchCheckpoint{
		blockNum:     binary.LittleEndian.Uint64(data[1:9]),
		timestamp:    binary.LittleEndian.Uint64(data[9:17]),
		derivedSince: binary.LittleEndian.Uint32(data[17:21]),
		derivedUntil: binary.LittleEndian.Uint64(data[21:29]),
	}, nil
}

// encode creates a checkpoint entry
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint64 timestamp: 8 bytes><uint32 derivedSince count: 4 bytes><uint64 derivedUntil: 8 bytes> = 29 bytes
func (s searchCheckpoint) encode() Entry {
	var data Entry
	data[0] = uint8(TypeSearchCheckpoint)
	binary.LittleEndian.PutUint64(data[1:9], s.blockNum)
	binary.LittleEndian.PutUint64(data[9:17], s.timestamp)
	binary.LittleEndian.PutUint32(data[17:21], s.derivedSince)
	binary.LittleEndian.PutUint64(data[21:29], s.derivedUntil)
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
		return canonicalHash{}, fmt.Errorf("%w: attempting to decode canonical hash but was type %s", entrydb.ErrDataCorruption, data.Type())
	}
	return newCanonicalHash(common.Hash(data[1:33])), nil
}

func (c canonicalHash) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeCanonicalHash)
	copy(entry[1:33], c.hash[:])
	return entry
}

type derivedLink struct {
	number    uint64
	timestamp uint64
	// May contain additional flag value in the future
}

func newDerivedLink(num uint64, timestamp uint64) derivedLink {
	return derivedLink{number: num, timestamp: timestamp}
}

func newDerivedLinkFromEntry(data Entry) (derivedLink, error) {
	if data.Type() != TypeDerivedLink {
		return derivedLink{}, fmt.Errorf("%w: attempting to decode derived link but was type %s", entrydb.ErrDataCorruption, data.Type())
	}
	return newDerivedLink(binary.LittleEndian.Uint64(data[1:9]), binary.LittleEndian.Uint64(data[9:17])), nil
}

func (d derivedLink) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeDerivedLink)
	binary.LittleEndian.PutUint64(entry[1:9], d.number)
	binary.LittleEndian.PutUint64(entry[9:17], d.timestamp)
	return entry
}

type derivedCheck struct {
	hash common.Hash
}

func newDerivedCheck(hash common.Hash) derivedCheck {
	return derivedCheck{hash: hash}
}

func newDerivedCheckFromEntry(data Entry) (derivedCheck, error) {
	if data.Type() != TypeDerivedCheck {
		return derivedCheck{}, fmt.Errorf("%w: attempting to decode derived check but was type %s", entrydb.ErrDataCorruption, data.Type())
	}
	return newDerivedCheck(common.Hash(data[1:33])), nil
}

func (d derivedCheck) encode() Entry {
	var entry Entry
	entry[0] = uint8(TypeDerivedCheck)
	copy(entry[1:33], d.hash[:])
	return entry
}
