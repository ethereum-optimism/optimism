package main

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	DB_CACHE                = 1024 // size of the cache in MB
	DB_HANDLES              = 60   // number of handles
	LAST_MIGRATED_BLOCK_KEY = "celoLastMigratedBlock"
)

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

// readLastMigratedBlock returns the last migration number.
func readLastMigratedBlock(db ethdb.KeyValueReader) uint64 {
	data, err := db.Get([]byte(LAST_MIGRATED_BLOCK_KEY))
	if err != nil {
		return 0
	}
	number := binary.BigEndian.Uint64(data)
	return number
}

// writeLastMigratedBlock stores the last migration number.
func writeLastMigratedBlock(db ethdb.KeyValueWriter, number uint64) error {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return db.Put([]byte(LAST_MIGRATED_BLOCK_KEY), enc)
}

// deleteLastMigratedBlock removes the last migration number.
func deleteLastMigratedBlock(db ethdb.KeyValueWriter) error {
	return db.Delete([]byte(LAST_MIGRATED_BLOCK_KEY))
}
