package db

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

func Open(path string, cache int, handles int) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.NewLevelDBDatabaseWithFreezer(chaindataPath, cache, handles, ancientPath, "", false)
	if err != nil {
		return nil, err
	}
	return ldb, nil
}
