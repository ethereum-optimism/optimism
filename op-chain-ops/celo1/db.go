package celo1

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// ReadHeaderRLP retrieves a block header in its raw RLP database encoding. (copied from rawdb)
func ReadHeaderRLP(db ethdb.Reader, hash common.Hash, number uint64) rlp.RawValue {
	var data []byte
	db.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		// First try to look up the data in ancient database. Extra hash
		// comparison is necessary since ancient database only maintains
		// the canonical data.
		data, _ = reader.Ancient(rawdb.ChainFreezerHeaderTable, number)
		if len(data) != 0 {
			return nil
		}
		// If not, try reading from leveldb
		data, _ = db.Get(headerKey(number, hash))
		return nil
	})
	return data
}

var (
	headerPrefix = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKey = headerPrefix + num (uint64 big endian) + hash
func headerKey(number uint64, hash common.Hash) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

func ReadCeloHeader(db ethdb.Reader, hash common.Hash, number uint64) (*Header, error) {
	data := ReadHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil, errors.New("header not found")
	}

	header := new(Header)
	if err := rlp.DecodeBytes(data, header); err != nil {
		return nil, err
	}
	return header, nil
}

// ReadHeader retrieves the block header corresponding to the hash. (copied from rawdb)
func ReadHeader(db ethdb.Reader, hash common.Hash, number uint64) (*types.Header, error) {
	data := ReadHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil, errors.New("header not found")
	}
	header := new(types.Header)
	if err := rlp.DecodeBytes(data, header); err != nil {
		return nil, err
	}
	return header, nil
}

// ReadCanonicalHeader retrieves the cannoical block header at the given number.
func ReadCanonicalHeader(db ethdb.Reader, number uint64) (*types.Header, error) {
	hash := rawdb.ReadCanonicalHash(db, number)
	return ReadHeader(db, hash, number)
}

// ReadCanonicalHeader retrieves the cannoical block header at the given number.
func ReadCeloCanonicalHeader(db ethdb.Reader, number uint64) (*Header, error) {
	hash := rawdb.ReadCanonicalHash(db, number)
	return ReadCeloHeader(db, hash, number)
}
