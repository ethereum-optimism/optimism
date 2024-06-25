package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Constants for the database
const (
	DBCache                        = 1024 // size of the cache in MB
	DBHandles                      = 60   // number of handles
	LastMigratedNonAncientBlockKey = "celoLastMigratedNonAncientBlock"
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

// readLastMigratedNonAncientBlock returns the last migration number. If it doesn't exist, it returns 0.
func readLastMigratedNonAncientBlock(db ethdb.KeyValueReader) uint64 {
	data, err := db.Get([]byte(LastMigratedNonAncientBlockKey))
	if err != nil {
		return 0
	}
	number := binary.BigEndian.Uint64(data)
	return number
}

// writeLastMigratedNonAncientBlock stores the last migration number.
func writeLastMigratedNonAncientBlock(db ethdb.KeyValueWriter, number uint64) error {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return db.Put([]byte(LastMigratedNonAncientBlockKey), enc)
}

// deleteLastMigratedNonAncientBlock removes the last migration number.
func deleteLastMigratedNonAncientBlock(db ethdb.KeyValueWriter) error {
	return db.Delete([]byte(LastMigratedNonAncientBlockKey))
}

// openDB opens the chaindata database at the given path. Note this path is below the datadir
func openDB(chaindataPath string) (ethdb.Database, error) {
	if _, err := os.Stat(chaindataPath); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
		AncientsDirectory: filepath.Join(chaindataPath, "ancient"),
		Namespace:         "",
		Cache:             DBCache,
		Handles:           DBHandles,
		ReadOnly:          false,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}

func createNewDbIfNotExists(newDBPath string) error {
	if err := os.MkdirAll(newDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create new database directory: %w", err)
	}
	return nil
}

func cleanupNonAncientDb(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	for _, file := range files {
		if file.Name() != "ancient" {
			err := os.RemoveAll(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to remove file: %w", err)
			}
		}
	}
	return nil
}
