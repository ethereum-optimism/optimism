package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
)

// How to run:
// 		go run ./op-chain-ops/cmd/celo-dbmigrate -oldDB /path/to/oldDB -newDB /path/to/newDB [-batchSize 1000] [-verbosity 3] [-clear-all] [-clear-nonAncients]
//
// This script will migrate block data from the old database to the new database
// You can set the log level using the -verbosity flag
// The number of ancient records to migrate in one batch can be set using the -batchSize flag
// The default batch size is 1000
// Use -clear-all to start with a fresh new database
// Use -clear-nonAncients to keep migrated ancients, but not non-ancients

func main() {
	oldDBPath := flag.String("oldDB", "", "Path to the old database chaindata directory (read-only)")
	newDBPath := flag.String("newDB", "", "Path to the new database")
	batchSize := flag.Uint64("batchSize", 10000, "Number of records to migrate in one batch")
	verbosity := flag.Uint64("verbosity", 3, "Log level (0:crit, 1:err, 2:warn, 3:info, 4:debug, 5:trace)")

	clearAll := flag.Bool("clear-all", false, "Use this to start with a fresh new database")
	clearNonAncients := flag.Bool("clear-nonAncients", false, "Use to keep migrated ancients, but not non-ancients")
	flag.Parse()

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*verbosity), log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd())))))

	var err error

	// check that `rsync` command is available
	if _, err := exec.LookPath("rsync"); err != nil {
		log.Info("Please install `rsync` to use this script")
		return
	}

	if *oldDBPath == "" || *newDBPath == "" {
		log.Info("Please provide both oldDB and newDB flags")
		flag.Usage()
		return
	}

	if *clearAll {
		if err := os.RemoveAll(*newDBPath); err != nil {
			log.Crit("Failed to remove new database", "err", err)
		}
	}
	if *clearNonAncients {
		if err := cleanupNonAncientDb(*newDBPath); err != nil {
			log.Crit("Failed to cleanup non-ancient database", "err", err)
		}
	}

	if err := createEmptyNewDb(*newDBPath); err != nil {
		log.Crit("Failed to create new database", "err", err)
	}

	var numAncientsNew uint64
	if numAncientsNew, err = migrateAncientsDb(*oldDBPath, *newDBPath, *batchSize); err != nil {
		log.Crit("Failed to migrate ancients database", "err", err)
	}

	var numNonAncients uint64
	if numNonAncients, err = migrateNonAncientsDb(*oldDBPath, *newDBPath, numAncientsNew-1, *batchSize); err != nil {
		log.Crit("Failed to migrate non-ancients database", "err", err)
	}

	log.Info("Migration Completed", "migratedAncients", numAncientsNew, "migratedNonAncients", numNonAncients)
}

func migrateNonAncientsDb(oldDbPath, newDbPath string, lastAncientBlock, batchSize uint64) (uint64, error) {
	// First copy files from old database to new database
	log.Info("Copy files from old database", "process", "db migration")
	cmd := exec.Command("rsync", "-v", "-a", "--exclude=ancient", oldDbPath+"/", newDbPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to copy old database to new database: %v", err)
	}

	// Open the new database without access to AncientsDb
	newDB, err := rawdb.NewLevelDBDatabase(newDbPath, DB_CACHE, DB_HANDLES, "", false)
	if err != nil {
		return 0, fmt.Errorf("failed to open new database: %v", err)
	}
	defer newDB.Close()

	// get the last block number
	hash := rawdb.ReadHeadHeaderHash(newDB)
	lastBlock := *rawdb.ReadHeaderNumber(newDB, hash)
	lastMigratedBlock := readLastMigratedBlock(newDB)

	// if migration was interrupted, start from the last migrated block
	fromBlock := max(lastAncientBlock, lastMigratedBlock) + 1

	log.Info("Migration started", "process", "db migration", "startBlock", fromBlock, "endBlock", lastBlock, "count", lastBlock-fromBlock)

	for i := fromBlock; i <= lastBlock; i += batchSize {
		numbersHash := rawdb.ReadAllHashesInRange(newDB, i, i+batchSize-1)

		log.Info("Processing Range", "process", "db migration", "from", i, "to(inclusve)", i+batchSize-1, "count", len(numbersHash))
		for _, numberHash := range numbersHash {
			// read header and body
			header := rawdb.ReadHeaderRLP(newDB, numberHash.Hash, numberHash.Number)
			body := rawdb.ReadBodyRLP(newDB, numberHash.Hash, numberHash.Number)

			// transform header and body
			newHeader, err := transformHeader(header)
			if err != nil {
				return 0, fmt.Errorf("failed to transform header: block %d - %x: %w", numberHash.Number, numberHash.Hash, err)
			}
			newBody, err := transformBlockBody(body)
			if err != nil {
				return 0, fmt.Errorf("failed to transform body: block %d - %x: %w", numberHash.Number, numberHash.Hash, err)
			}

			if yes, newHash := hasSameHash(newHeader, numberHash.Hash[:]); !yes {
				log.Error("Hash mismatch", "block", numberHash.Number, "oldHash", numberHash.Hash, "newHash", newHash)
				return 0, fmt.Errorf("hash mismatch at block %d - %x", numberHash.Number, numberHash.Hash)
			}

			// write header and body
			batch := newDB.NewBatch()
			rawdb.WriteBodyRLP(batch, numberHash.Hash, numberHash.Number, newBody)
			_ = batch.Put(headerKey(numberHash.Number, numberHash.Hash), newHeader)
			_ = writeLastMigratedBlock(batch, numberHash.Number)
			if err := batch.Write(); err != nil {
				return 0, fmt.Errorf("failed to write header and body: block %d - %x: %w", numberHash.Number, numberHash.Hash, err)
			}
		}
	}

	toBeRemoved := rawdb.ReadAllHashesInRange(newDB, 1, lastAncientBlock)
	log.Info("Removing frozen blocks", "process", "db migration", "count", len(toBeRemoved))
	batch := newDB.NewBatch()
	for _, numberHash := range toBeRemoved {
		rawdb.DeleteBlockWithoutNumber(batch, numberHash.Hash, numberHash.Number)
		rawdb.DeleteCanonicalHash(batch, numberHash.Number)
	}
	if err := batch.Write(); err != nil {
		return 0, fmt.Errorf("failed to delete frozen blocks: %w", err)
	}

	// if migration finished, remove the last migration number
	if err := deleteLastMigratedBlock(newDB); err != nil {
		return 0, fmt.Errorf("failed to delete last migration number: %v", err)
	}
	log.Info("Migration ended", "process", "db migration", "migratedBlocks", lastBlock-fromBlock+1, "removedBlocks", len(toBeRemoved))

	return lastBlock - fromBlock + 1, nil
}

func createEmptyNewDb(newDBPath string) error {
	if err := os.MkdirAll(newDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create new database directory: %v", err)
	}
	return nil
}

func cleanupNonAncientDb(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
	for _, file := range files {
		if file.Name() != "ancient" {
			err := os.RemoveAll(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to remove file: %v", err)
			}
		}
	}
	return nil
}
