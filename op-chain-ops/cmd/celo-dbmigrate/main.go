package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
)

// How to run:
// 		go run main.go -oldDB /path/to/oldDB -newDB /path/to/newDB [-resetDB] [-batchSize 1000]
//
// This script will migrate ancient data from the old database to the new database
// The new database will be reset if the -resetDB flag is provided
// The number of ancient records to migrate in one batch can be set using the -batchSize flag
// The default batch size is 1000

const (
	dbCache         = 1024 // size of the cache in MB
	dbHandles       = 60   // number of handles
	ancientHashSize = 38   // size of a hash entry on ancient database (empiric value)
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	oldDBPath := flag.String("oldDB", "", "Path to the old database")
	newDBPath := flag.String("newDB", "", "Path to the new database")
	resetDB := flag.Bool("resetDB", false, "Use to reset the new database before migrating data")
	batchSize := flag.Uint64("batchSize", 1000, "Number of ancient records to migrate in one batch")

	flag.Parse()

	if *oldDBPath == "" || *newDBPath == "" {
		log.Info("Please provide both oldDB and newDB flags")
		return
	}

	if *resetDB {
		if err := os.RemoveAll(*newDBPath); err != nil {
			log.Crit("Failed to remove new database", "err", err)
		}
	}

	// Open the existing database in read-only mode
	oldDB, err := openCeloDb(*oldDBPath, dbCache, dbHandles, true)
	if err != nil {
		log.Crit("Failed to open old database", "err", err)
	}

	// Create a new database
	newDB, err := openCeloDb(*newDBPath, dbCache, dbHandles, false)
	if err != nil {
		log.Crit("Failed to create new database", "err", err)
	}

	// Close the databases
	defer oldDB.Close()
	defer newDB.Close()

	toMigrateRecords := MustAncientLength(oldDB)
	migratedRecords := MustAncientLength(newDB)
	log.Info("Ancient Migration Initial Status", "migrated", migratedRecords, "total", toMigrateRecords)

	if err = freezeRange(oldDB, newDB, migratedRecords, *batchSize); err != nil {
		log.Crit("Failed to freeze range", "err", err)
	}

	log.Info("Ancient Migration End Status", "migrated", MustAncientLength(newDB), "total", toMigrateRecords)

}

// Opens a Celo database, stored in the `celo` subfolder
func openCeloDb(path string, cache int, handles int, readonly bool) (ethdb.Database, error) {
	ancientPath := filepath.Join(path, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         path,
		AncientsDirectory: ancientPath,
		Namespace:         "",
		Cache:             cache,
		Handles:           handles,
		ReadOnly:          readonly,
	})
	if err != nil {
		return nil, err
	}
	return ldb, nil
}

func freezeRange(oldDb ethdb.Database, newDb ethdb.Database, number, count uint64) error {
	_, err := newDb.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		hashes, err := oldDb.AncientRange(rawdb.ChainFreezerHashTable, number, count, 0)
		if err != nil {
			return fmt.Errorf("can't read hashes from old freezer: %v", err)
		}
		headers, err := oldDb.AncientRange(rawdb.ChainFreezerHeaderTable, number, count, 0)
		if err != nil {
			return fmt.Errorf("can't read headers from old freezer: %v", err)
		}
		bodies, err := oldDb.AncientRange(rawdb.ChainFreezerBodiesTable, number, count, 0)
		if err != nil {
			return fmt.Errorf("can't read bodies from old freezer: %v", err)
		}
		receipts, err := oldDb.AncientRange(rawdb.ChainFreezerReceiptTable, number, count, 0)
		if err != nil {
			return fmt.Errorf("can't read receipts from old freezer: %v", err)
		}
		tds, err := oldDb.AncientRange(rawdb.ChainFreezerDifficultyTable, number, count, 0)
		if err != nil {
			return fmt.Errorf("can't read tds from old freezer: %v", err)
		}

		if len(hashes) != len(headers) || len(headers) != len(bodies) || len(bodies) != len(receipts) || len(receipts) != len(tds) {
			return fmt.Errorf("inconsistent data in ancient tables")
		}

		for i := 0; i < len(hashes); i++ {
			log.Debug("Migrating ancient data", "number", number+uint64(i))

			newHeader, err := transformHeader(headers[i])
			if err != nil {
				return fmt.Errorf("can't transform header: %v", err)
			}
			newBody, err := transformBlockBody(bodies[i])
			if err != nil {
				return fmt.Errorf("can't transform body: %v", err)
			}

			if err := op.AppendRaw(rawdb.ChainFreezerHashTable, number+uint64(i), hashes[i]); err != nil {
				return fmt.Errorf("can't write hash to Freezer: %v", err)
			}
			if err := op.AppendRaw(rawdb.ChainFreezerHeaderTable, number+uint64(i), newHeader); err != nil {
				return fmt.Errorf("can't write header to Freezer: %v", err)
			}
			if err := op.AppendRaw(rawdb.ChainFreezerBodiesTable, number+uint64(i), newBody); err != nil {
				return fmt.Errorf("can't write body to Freezer: %v", err)
			}
			if err := op.AppendRaw(rawdb.ChainFreezerReceiptTable, number+uint64(i), receipts[i]); err != nil {
				return fmt.Errorf("can't write receipts to Freezer: %v", err)
			}
			if err := op.AppendRaw(rawdb.ChainFreezerDifficultyTable, number+uint64(i), tds[i]); err != nil {
				return fmt.Errorf("can't write td to Freezer: %v", err)
			}
		}
		return nil
	})

	return err
}

// transformHeader migrates the header from the old format to the new format (works with []byte input output)
func transformHeader(oldHeader []byte) ([]byte, error) {
	// TODO: implement the transformation (remove only validator bls Signature)
	return oldHeader, nil
}

// transformBlockBody migrates the block body from the old format to the new format (works with []byte input output)
func transformBlockBody(oldBody []byte) ([]byte, error) {
	// TODO: implement the transformation (remove epochSnarkData and randomness data from the body)
	return oldBody, nil
}

// MustAncientLength returns the number of items in the ancients database
func MustAncientLength(db ethdb.Database) uint64 {
	byteSize, err := db.AncientSize(rawdb.ChainFreezerHashTable)
	if err != nil {
		log.Crit("Failed to get ancient size", "error", err)
	}
	return byteSize / ancientHashSize
}

// finds number of items in the ancients database
func findAncientsSize(ldb ethdb.Database, high uint64) uint64 {
	// runs a binary search using Ancient.HasAncient to find the first hash it can't find
	low := uint64(0)
	for low < high {
		mid := (low + high) / 2
		if ok, err := ldb.HasAncient(rawdb.ChainFreezerHashTable, mid); ok && err == nil {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low

}
