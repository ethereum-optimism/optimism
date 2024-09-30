package fromda

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum/go-ethereum/common"
)

// searchCheckpoint is both a checkpoint for searching, as well as a checkpoint for sealing blocks.
type searchCheckpoint struct {
	blockNum uint64
	// number of L2 blocks that were derived after this checkpoint
	blocksSince uint32
	lastDerived uint64
	timestamp   uint64
}

func newSearchCheckpoint(blockNum uint64, blocksSince uint32, timestamp uint64) searchCheckpoint {
	return searchCheckpoint{
		blockNum:    blockNum,
		blocksSince: blocksSince,
		timestamp:   timestamp,
	}
}

func newSearchCheckpointFromEntry(data Entry) (searchCheckpoint, error) {
	if data.Type() != TypeSearchCheckpoint {
		return searchCheckpoint{}, fmt.Errorf("%w: attempting to decode search checkpoint but was type %s", entrydb.ErrDataCorruption, data.Type())
	}
	return searchCheckpoint{
		blockNum:    binary.LittleEndian.Uint64(data[1:9]),
		blocksSince: binary.LittleEndian.Uint32(data[9:13]),
		timestamp:   binary.LittleEndian.Uint64(data[13:21]),
	}, nil
}

// encode creates a checkpoint entry
// type 0: "search checkpoint" <type><uint64 block number: 8 bytes><uint32 blocksSince count: 4 bytes><uint64 timestamp: 8 bytes> = 21 bytes
func (s searchCheckpoint) encode() Entry {
	var data Entry
	data[0] = uint8(TypeSearchCheckpoint)
	binary.LittleEndian.PutUint64(data[1:9], s.blockNum)
	binary.LittleEndian.PutUint32(data[9:13], s.blocksSince)
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
