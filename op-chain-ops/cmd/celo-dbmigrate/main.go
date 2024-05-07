package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/celo1"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mattn/go-isatty"

	"golang.org/x/sync/errgroup"
)

// How to run:
// 		go run main.go -oldDB /path/to/oldDB -newDB /path/to/newDB [-resetDB] [-batchSize 1000] [-verbosity 3] [-clear-all] [-clear-nonAncients]
//
// This script will migrate block data from the old database to the new database
// The new database will be reset if the -resetDB flag is provided
// You can set the log level using the -verbosity flag
// The number of ancient records to migrate in one batch can be set using the -batchSize flag
// The default batch size is 1000
// Use -clear-all to start with a fresh new database
// Use -clear-nonAncients to keep migrated ancients, but not non-ancients

const (
	DB_CACHE                = 1024 // size of the cache in MB
	DB_HANDLES              = 60   // number of handles
	LAST_MIGRATED_BLOCK_KEY = "celoLastMigratedBlock"
)

func main() {
	oldDBPath := flag.String("oldDB", "", "Path to the old database chaindata directory (read-only)")
	newDBPath := flag.String("newDB", "", "Path to the new database")
	batchSize := flag.Uint64("batchSize", 10000, "Number of records to migrate in one batch")
	verbosity := flag.Uint64("verbosity", 3, "Log level (0:crit, 1:err, 2:warn, 3:info, 4:debug, 5:trace)")

	clearAll := flag.Bool("clear-all", false, "Use this to start with a fresh new database")
	clearNonAncients := flag.Bool("clear-nonAncients", false, "Use to keep migrated ancients, but not non-ancients")
	// flag.Usage = usage
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

	var numAncientsNonAncients uint64
	if numAncientsNonAncients, err = migrateNonAncientsDb(*oldDBPath, *newDBPath, numAncientsNew-1, *batchSize); err != nil {
		log.Crit("Failed to migrate non-ancients database", "err", err)
	}

	log.Info("Migration Completed", "migratedAncients", numAncientsNew, "migratedNonAncients", numAncientsNonAncients)
}

func migrateAncientsDb(oldDBPath, newDBPath string, batchSize uint64) (uint64, error) {
	oldFreezer, err := rawdb.NewChainFreezer(filepath.Join(oldDBPath, "ancient"), "", true)
	if err != nil {
		return 0, fmt.Errorf("failed to open old freezer: %v", err)
	}
	defer oldFreezer.Close()

	newFreezer, err := rawdb.NewChainFreezer(filepath.Join(newDBPath, "ancient"), "", false)
	if err != nil {
		return 0, fmt.Errorf("failed to open new freezer: %v", err)
	}
	defer newFreezer.Close()

	numAncientsOld, err := oldFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in old freezer: %v", err)
	}

	numAncientsNew, err := newFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in new freezer: %v", err)
	}

	log.Info("Ancient Migration Initial Status", "migrated", numAncientsNew, "total", numAncientsOld)

	if err = parMigrateAncientRange(oldFreezer, newFreezer, numAncientsNew, numAncientsOld, batchSize); err != nil {
		return 0, fmt.Errorf("failed to migrate ancient range: %v", err)
	}

	numAncientsNew, err = newFreezer.Ancients()
	if err != nil {
		return 0, fmt.Errorf("failed to get number of ancients in new freezer: %v", err)
	}

	log.Info("Ancient Migration End Status", "migrated", numAncientsNew, "total", numAncientsOld)
	return numAncientsNew, nil
}

func migrateNonAncientsDb(oldDbPath, newDbPath string, fromBlock, batchSize uint64) (uint64, error) {
	// First copy files from old database to new database
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
	lastBlock := GetLastBlockNumber(newDB)
	lastMigratedBlock := readLastMigratedBlock(newDB)

	// if migration was interrupted, start from the last migrated block
	fromBlock = max(fromBlock, lastMigratedBlock+1)

	log.Info("Non Ancient Migration", "from", fromBlock, "to", lastBlock, "count", lastBlock-fromBlock)

	for i := fromBlock; i <= lastBlock; i += batchSize {
		numbersHash := rawdb.ReadAllHashesInRange(newDB, i, i+batchSize-1)

		log.Info("Processing Range", "from", i, "to(inclusve)", i+batchSize-1, "count", len(numbersHash))
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

			// write header and body
			rawdb.WriteBodyRLP(newDB, numberHash.Hash, numberHash.Number, newBody)

			if err := newDB.Put(celo1.HeaderKey(numberHash.Number, numberHash.Hash), newHeader); err != nil {
				return 0, fmt.Errorf("can't write header to new database: %v", err)
			}

			if err = writeLastMigratedBlock(newDB, numberHash.Number); err != nil {
				return 0, fmt.Errorf("failed to write last migration number: %v", err)
			}

		}
	}

	// if migration finished, remove the last migration number
	if err := deleteLastMigratedBlock(newDB); err != nil {
		return 0, fmt.Errorf("failed to delete last migration number: %v", err)
	}

	return lastBlock - fromBlock + 1, nil
}

func createEmptyNewDb(newDBPath string) error {
	if err := os.MkdirAll(newDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create new database directory: %v", err)
	}
	return nil
}

// RLPBlockRange is a range of blocks in RLP format
type RLPBlockRange struct {
	start    uint64
	hashes   [][]byte
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	tds      [][]byte
}

// parMigrateRange migrates ancient data from the old database to the new database in parallel
func parMigrateAncientRange(oldFreezer, newFreezer *rawdb.Freezer, start, end, step uint64) error {
	g, ctx := errgroup.WithContext(context.Background())
	readChan := make(chan RLPBlockRange, 10)
	transformChan := make(chan RLPBlockRange, 10)

	log.Info("Migrating data", "start", start, "end", end, "step", step)

	g.Go(func() error { return readAncientBlocks(ctx, oldFreezer, start, end, step, readChan) })
	g.Go(func() error { return transformBlocks(ctx, readChan, transformChan) })
	g.Go(func() error { return writeAncientBlocks(ctx, newFreezer, transformChan) })

	return g.Wait()
}

func readAncientBlocks(ctx context.Context, freezer *rawdb.Freezer, startBlock, endBlock, batchSize uint64, out chan<- RLPBlockRange) error {
	defer close(out)

	for i := startBlock; i < endBlock; i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			count := min(batchSize, endBlock-i+1)
			start := i

			blockRange := RLPBlockRange{
				start:    start,
				hashes:   make([][]byte, count),
				headers:  make([][]byte, count),
				bodies:   make([][]byte, count),
				receipts: make([][]byte, count),
				tds:      make([][]byte, count),
			}
			var err error

			blockRange.hashes, err = freezer.AncientRange(rawdb.ChainFreezerHashTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read hashes from old freezer: %v", err)
			}
			blockRange.headers, err = freezer.AncientRange(rawdb.ChainFreezerHeaderTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read headers from old freezer: %v", err)
			}
			blockRange.bodies, err = freezer.AncientRange(rawdb.ChainFreezerBodiesTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read bodies from old freezer: %v", err)
			}
			blockRange.receipts, err = freezer.AncientRange(rawdb.ChainFreezerReceiptTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read receipts from old freezer: %v", err)
			}
			blockRange.tds, err = freezer.AncientRange(rawdb.ChainFreezerDifficultyTable, start, count, 0)
			if err != nil {
				return fmt.Errorf("failed to read tds from old freezer: %v", err)
			}

			out <- blockRange
		}
	}
	return nil
}

func transformBlocks(ctx context.Context, in <-chan RLPBlockRange, out chan<- RLPBlockRange) error {
	// Transform blocks from the in channel and send them to the out channel
	defer close(out)
	for blockRange := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			for i := range blockRange.hashes {
				blockNumber := blockRange.start + uint64(i)

				newHeader, err := transformHeader(blockRange.headers[i])
				if err != nil {
					return fmt.Errorf("can't transform header: %v", err)
				}
				newBody, err := transformBlockBody(blockRange.bodies[i])
				if err != nil {
					return fmt.Errorf("can't transform body: %v", err)
				}

				// Check that hashing the new header gives the same hash as the saved hash
				newHash := crypto.Keccak256Hash(newHeader)
				if !bytes.Equal(blockRange.hashes[i], newHash.Bytes()) {
					log.Error("Hash mismatch", "block", blockNumber, "oldHash", common.BytesToHash(blockRange.hashes[i]), "newHash", newHash)
					return fmt.Errorf("hash mismatch at block %d", blockNumber)
				}

				blockRange.headers[i] = newHeader
				blockRange.bodies[i] = newBody
			}
			out <- blockRange
		}
	}
	return nil
}

func writeAncientBlocks(ctx context.Context, freezer *rawdb.Freezer, in <-chan RLPBlockRange) error {
	// Write blocks from the in channel to the newDb
	for blockRange := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := freezer.ModifyAncients(func(aWriter ethdb.AncientWriteOp) error {
				for i := range blockRange.hashes {
					blockNumber := blockRange.start + uint64(i)
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHashTable, blockNumber, blockRange.hashes[i]); err != nil {
						return fmt.Errorf("can't write hash to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHeaderTable, blockNumber, blockRange.headers[i]); err != nil {
						return fmt.Errorf("can't write header to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerBodiesTable, blockNumber, blockRange.bodies[i]); err != nil {
						return fmt.Errorf("can't write body to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerReceiptTable, blockNumber, blockRange.receipts[i]); err != nil {
						return fmt.Errorf("can't write receipts to Freezer: %v", err)
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerDifficultyTable, blockNumber, blockRange.tds[i]); err != nil {
						return fmt.Errorf("can't write td to Freezer: %v", err)
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to write block range: %v", err)
			}
			log.Info("Wrote ancient blocks", "start", blockRange.start, "end", blockRange.start+uint64(len(blockRange.hashes)-1), "count", len(blockRange.hashes))
		}
	}
	return nil
}

// transformHeader migrates the header from the old format to the new format (works with []byte input output)
func transformHeader(oldHeader []byte) ([]byte, error) {
	return celo1.RemoveIstanbulAggregatedSeal(oldHeader)
}

// transformBlockBody migrates the block body from the old format to the new format (works with []byte input output)
func transformBlockBody(oldBodyData []byte) ([]byte, error) {
	// decode body into celo-blockchain Body structure
	// remove epochSnarkData and randomness data
	var celoBody struct {
		Transactions   rlp.RawValue // TODO use types.Transactions to make sure all tx are deserializable
		Randomness     rlp.RawValue
		EpochSnarkData rlp.RawValue
	}
	if err := rlp.DecodeBytes(oldBodyData, &celoBody); err != nil {
		return nil, fmt.Errorf("failed to RLP decode body: %w", err)
	}

	// TODO create a types.BlockBody structure and encode it back to []byte

	// transform into op-geth types.Body structure
	// since Body is a slice of types.Transactions, we can just remove the randomness and epochSnarkData and add empty array for UnclesHashes
	newBodyData, err := rlp.EncodeToBytes([]interface{}{celoBody.Transactions, nil})
	if err != nil {
		return nil, fmt.Errorf("failed to RLP encode body: %w", err)
	}

	return newBodyData, nil
}

// GetLastBlockNumber returns the number of the last block in the database
func GetLastBlockNumber(db ethdb.Database) uint64 {
	hash := rawdb.ReadHeadHeaderHash(db)
	return *rawdb.ReadHeaderNumber(db, hash)
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
