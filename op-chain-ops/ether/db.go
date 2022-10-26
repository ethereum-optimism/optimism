package ether

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// MustOpenDB opens a Geth database, or panics. Note that
// the database must be opened with a freezer in order to
// properly read historical data.
func MustOpenDB(dataDir string) ethdb.Database {
	return MustOpenDBWithCacheOpts(dataDir, 0, 0)
}

// MustOpenDBWithCacheOpts opens a Geth database or panics. Allows
// the caller to pass in LevelDB cache parameters.
func MustOpenDBWithCacheOpts(dataDir string, cacheSize, handles int) ethdb.Database {
	dir := filepath.Join(dataDir, "geth", "chaindata")
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(
		dir,
		cacheSize,
		handles,
		filepath.Join(dir, "ancient"),
		"",
		true,
	)
	if err != nil {
		log.Crit("error opening raw DB", "err", err)
	}
	return db
}
