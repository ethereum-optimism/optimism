package db

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func Open(path string, cache int, handles int) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
		AncientsDirectory: ancientPath,
		Namespace:         "",
		Cache:             cache,
		Handles:           handles,
		ReadOnly:          false,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}
