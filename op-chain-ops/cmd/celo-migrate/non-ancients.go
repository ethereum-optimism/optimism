package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
)

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
	newDB, err := rawdb.NewLevelDBDatabase(newDbPath, DBCache, DBHandles, "", false)
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
