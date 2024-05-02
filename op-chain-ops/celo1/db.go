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
		data, _ = db.Get(HeaderKey(number, hash))
		return nil
	})
	return data
}

var (
	// Data item prefixes (use single byte to avoid mixing data types, avoid `i`, used for indexes).
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerTDSuffix     = []byte("t") // headerPrefix + num (uint64 big endian) + hash + headerTDSuffix -> td
	headerHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	headerNumberPrefix = []byte("H") // headerNumberPrefix + hash -> num (uint64 big endian)

	blockBodyPrefix     = []byte("b") // blockBodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts

	txLookupPrefix = []byte("l") // txLookupPrefix + hash -> transaction/receipt lookup metadata
)

// encodeBlockNumber encodes a block number as big endian uint64
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

// headerKeyPrefix = headerPrefix + num (uint64 big endian)
func headerKeyPrefix(number uint64) []byte {
	return append(headerPrefix, encodeBlockNumber(number)...)
}

// headerKey = headerPrefix + num (uint64 big endian) + hash
func HeaderKey(number uint64, hash common.Hash) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// headerTDKey = headerPrefix + num (uint64 big endian) + hash + headerTDSuffix
func HeaderTDKey(number uint64, hash common.Hash) []byte {
	return append(HeaderKey(number, hash), headerTDSuffix...)
}

// headerHashKey = headerPrefix + num (uint64 big endian) + headerHashSuffix
func HeaderHashKey(number uint64) []byte {
	return append(append(headerPrefix, encodeBlockNumber(number)...), headerHashSuffix...)
}

// headerNumberKey = headerNumberPrefix + hash
func HeaderNumberKey(hash common.Hash) []byte {
	return append(headerNumberPrefix, hash.Bytes()...)
}

// blockBodyKey = blockBodyPrefix + num (uint64 big endian) + hash
func BlockBodyKey(number uint64, hash common.Hash) []byte {
	return append(append(blockBodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// blockReceiptsKey = blockReceiptsPrefix + num (uint64 big endian) + hash
func BlockReceiptsKey(number uint64, hash common.Hash) []byte {
	return append(append(blockReceiptsPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// txLookupKey = txLookupPrefix + hash
func txLookupKey(hash common.Hash) []byte {
	return append(txLookupPrefix, hash.Bytes()...)
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
