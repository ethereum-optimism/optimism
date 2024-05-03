package main

import (
	"bytes"
	"context"
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
// 		go run main.go -oldDB /path/to/oldDB -newDB /path/to/newDB [-batchSize 1000]
//
// This script will migrate block data from the old database to the new database
// The new database will be reset if the -resetDB flag is provided
// The number of ancient records to migrate in one batch can be set using the -batchSize flag
// The default batch size is 1000

const (
	dbCache         = 1024 // size of the cache in MB
	dbHandles       = 60   // number of handles
	ancientHashSize = 38   // size of a hash entry on ancient database (empiric value)
)

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd())))))

	oldDBPath := flag.String("oldDB", "", "Path to the old database")
	newDBPath := flag.String("newDB", "", "Path to the new database")
	resetDB := flag.Bool("resetDB", false, "Use to reset the new database before migrating data (recommended)")
	batchSize := flag.Uint64("batchSize", 1000, "Number of records to migrate in one batch")

	flag.Parse()

	if *oldDBPath == "" || *newDBPath == "" {
		log.Info("Please provide both oldDB and newDB flags")
		return
	}

	if *resetDB {
		// Reset the new database
		if err := os.RemoveAll(*newDBPath); err != nil {
			log.Crit("Failed to remove new database", "err", err)
		}
	}
	// if err := os.MkdirAll(filepath.Join(*newDBPath, "geth"), 0755); err != nil {
	// 	log.Crit("Failed to make new datadir", "err", err)
	// }

	// Copy the old database to the new database
	// cmd := exec.Command("cp", "-r", filepath.Join(*oldDBPath, "celo", "chaindata"), filepath.Join(*newDBPath, "geth"))
	// err := cmd.Run()
	// if err != nil {
	// 	log.Crit("Failed to copy old database to new database", "err", err)
	// }

	// Open the existing database in read-only mode
	oldDB, err := openDB(filepath.Join(*oldDBPath, "celo"), dbCache, dbHandles, true)
	if err != nil {
		log.Crit("Failed to open old database", "err", err)
	}

	// Create a new database
	newDB, err := openDB(filepath.Join(*newDBPath, "geth"), dbCache, dbHandles, false)
	if err != nil {
		log.Crit("Failed to create new database", "err", err)
	}

	// Ancients is append only, so we need to remove and recreate it to transform the data
	// newAncientPath := filepath.Join(*newDBPath, "geth", "chaindata", "ancient")
	// if err := os.RemoveAll(newAncientPath); err != nil {
	// 	log.Crit("Failed to remove copied ancient database", "err", err)
	// }

	// Close the databases
	defer oldDB.Close()
	defer newDB.Close()

	numAncients := MustAncientLength(oldDB)
	numAncientsMigrated := MustAncientLength(newDB)
	log.Info("Ancient Migration Initial Status", "migrated", numAncientsMigrated, "total", numAncients)

	if err = parMigrateRange(oldDB, newDB, numAncientsMigrated, numAncients, *batchSize, readAncientBlockRange, writeAncientBlockRange); err != nil {
		log.Crit("Failed to migrate ancient range", "err", err)
	}

	log.Info("Ancient Migration End Status", "migrated", MustAncientLength(newDB), "total", numAncients)

	// Move the ancient directory up one level, delete everything else and move it back

	// Move the ancient directory up one level
	cmd := exec.Command("rsync", "-av", filepath.Join(*newDBPath, "geth", "chaindata", "ancient"), filepath.Join(*newDBPath, "geth"))
	err = cmd.Run()
	if err != nil {
		log.Crit("Failed to move ancient directory up one level", "err", err)
	}
	// Delete everything in chaindata directory
	err = os.RemoveAll(filepath.Join(*newDBPath, "geth", "chaindata", "ancient"))
	if err != nil {
		log.Crit("Failed to clean chaindata directory", "err", err)
	}
	// Copy the old database to the new database, excluding the ancient folder
	cmd = exec.Command("rsync", "-av", filepath.Join(*oldDBPath, "celo", "chaindata"), filepath.Join(*newDBPath, "geth"), "--exclude", "ancient")
	err = cmd.Run()
	if err != nil {
		log.Crit("Failed to copy old database to new database", "err", err)
	}
	// Move the ancient directory back into the chaindata directory
	cmd = exec.Command("rsync", "-av", filepath.Join(*newDBPath, "geth", "ancient"), filepath.Join(*newDBPath, "geth", "chaindata"))
	err = cmd.Run()
	if err != nil {
		log.Crit("Failed to move ancient directory up one level", "err", err)
	}
	// Delete the extra ancient directory
	err = os.RemoveAll(filepath.Join(*newDBPath, "geth", "ancient"))
	if err != nil {
		log.Crit("Failed to remove extra ancient directory", "err", err)
	}

	numBlocks := GetLastBlockNumber(oldDB) + 1
	numBlocksMigrated := GetLastBlockNumber(newDB) + 1
	log.Info("Migration Initial Status", "migrated", numBlocksMigrated, "total", numBlocks)

	if err := parMigrateRange(oldDB, newDB, numBlocksMigrated, numBlocks, *batchSize, readBlockRange, writeBlockRange); err != nil {
		log.Crit("Failed to migrate range", "err", err)
	}

	// TODO do we still need to do this now that we've copied everything over?
	rawdb.WriteHeadHeaderHash(newDB, rawdb.ReadHeadHeaderHash(oldDB))

	log.Info("Migration End Status", "migrated", GetLastBlockNumber(newDB)+1, "total", numBlocks)

	log.Info("Migration complete")
}

// Opens a database
func openDB(path string, cache int, handles int, readonly bool) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
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

// RLPBlockRange is a range of blocks in RLP format
type RLPBlockRange struct {
	start    uint64
	hashes   [][]byte
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	tds      [][]byte
}

// seqMigrateAncientRange migrates ancient data from the old database to the new database sequentially
func seqMigrateAncientRange(oldDb ethdb.Database, newDb ethdb.Database, number, count uint64) error {
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

// seqMigrateRange migrates data from the old database to the new database sequentially
func seqMigrateRange(oldDb ethdb.Database, newDb ethdb.Database, start, count uint64) error {
	blockRange, err := readBlockRange(oldDb, start, count)
	if err != nil {
		return fmt.Errorf("can't read block range: %v", err)
	}
	err = transformBlockRange(blockRange)
	if err != nil {
		return fmt.Errorf("can't transform block range: %v", err)
	}
	err = writeBlockRange(newDb, blockRange)
	if err != nil {
		return fmt.Errorf("can't write block range: %v", err)
	}
	return nil
}

// parMigrateRange migrates ancient data from the old database to the new database in parallel
func parMigrateRange(oldDb ethdb.Database, newDb ethdb.Database, start, end, step uint64, reader func(ethdb.Database, uint64, uint64) (*RLPBlockRange, error), writer func(ethdb.Database, *RLPBlockRange) error) error {
	g, ctx := errgroup.WithContext(context.Background())
	readChan := make(chan RLPBlockRange, 10)
	transformChan := make(chan RLPBlockRange, 10)

	g.Go(func() error { return readBlocks(ctx, oldDb, start, end, step, readChan, reader) })
	g.Go(func() error { return transformBlocks(ctx, readChan, transformChan) })
	g.Go(func() error { return writeBlocks(ctx, newDb, transformChan, writer) })

	return g.Wait()
}

func readBlocks(ctx context.Context, oldDb ethdb.Database, startBlock, endBlock, batchSize uint64, out chan<- RLPBlockRange, readBlockRange func(ethdb.Database, uint64, uint64) (*RLPBlockRange, error)) error {
	defer close(out)
	// Read blocks and send them to the out channel
	// This could be reading from a database, a file, etc.
	for i := startBlock; i < endBlock; i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			count := min(batchSize, endBlock-i+1)
			blockRange, err := readBlockRange(oldDb, i, count)
			if err != nil {
				return fmt.Errorf("Failed to read block range: %v", err)
			}
			out <- *blockRange
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
			err := transformBlockRange(&blockRange)
			if err != nil {
				return fmt.Errorf("Failed to transform block range: %v", err)
			}
			out <- blockRange
		}
	}
	return nil
}

func writeBlocks(ctx context.Context, newDb ethdb.Database, in <-chan RLPBlockRange, writeBlockRange func(ethdb.Database, *RLPBlockRange) error) error {
	// Write blocks from the in channel to the newDb
	for blockRange := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := writeBlockRange(newDb, &blockRange)
			if err != nil {
				return fmt.Errorf("Failed to write block range: %v", err)
			}
		}
	}
	return nil
}

func readAncientBlockRange(db ethdb.Database, start, count uint64) (*RLPBlockRange, error) {
	blockRange := RLPBlockRange{
		start:    start,
		hashes:   make([][]byte, count),
		headers:  make([][]byte, count),
		bodies:   make([][]byte, count),
		receipts: make([][]byte, count),
		tds:      make([][]byte, count),
	}
	var err error

	log.Debug("Reading ancient blocks", "start", start, "end", start+count-1, "count", count)

	blockRange.hashes, err = db.AncientRange(rawdb.ChainFreezerHashTable, start, count, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to read hashes from old freezer: %v", err)
	}
	blockRange.headers, err = db.AncientRange(rawdb.ChainFreezerHeaderTable, start, count, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to read headers from old freezer: %v", err)
	}
	blockRange.bodies, err = db.AncientRange(rawdb.ChainFreezerBodiesTable, start, count, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to read bodies from old freezer: %v", err)
	}
	blockRange.receipts, err = db.AncientRange(rawdb.ChainFreezerReceiptTable, start, count, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to read receipts from old freezer: %v", err)
	}
	blockRange.tds, err = db.AncientRange(rawdb.ChainFreezerDifficultyTable, start, count, 0)
	if err != nil {
		return nil, fmt.Errorf("Failed to read tds from old freezer: %v", err)
	}

	return &blockRange, nil
}

func readBlockRange(db ethdb.Database, start, count uint64) (*RLPBlockRange, error) {
	blockRange := RLPBlockRange{
		start:    start,
		hashes:   make([][]byte, count),
		headers:  make([][]byte, count),
		bodies:   make([][]byte, count),
		receipts: make([][]byte, count),
		tds:      make([][]byte, count),
	}
	var err error

	log.Debug("Reading blocks", "start", start, "end", start+count-1, "count", count)

	for i := start; i < start+count; i++ {
		log.Debug("Reading old data", "number", i)

		blockRange.hashes[i-start], err = db.Get(celo1.HeaderHashKey(i))
		if err != nil {
			return nil, fmt.Errorf("Failed to load hash, number: %d, err: %v", i, err)
		}
		hash := common.BytesToHash(blockRange.hashes[i-start])
		blockRange.headers[i-start], err = db.Get(celo1.HeaderKey(i, hash))
		if err != nil {
			return nil, fmt.Errorf("Failed to load header, number: %d, err: %v", i, err)
		}
		blockRange.bodies[i-start], err = db.Get(celo1.BlockBodyKey(i, hash))
		if err != nil {
			return nil, fmt.Errorf("Failed to load body, number: %d, err: %v", i, err)
		}
		blockRange.receipts[i-start], err = db.Get(celo1.BlockReceiptsKey(i, hash))
		if err != nil {
			return nil, fmt.Errorf("Failed to load receipts, number: %d, err: %v", i, err)
		}
		blockRange.tds[i-start], err = db.Get(celo1.HeaderTDKey(i, hash))
		if err != nil {
			return nil, fmt.Errorf("Failed to load td, number: %d, err: %v", i, err)
		}
	}

	return &blockRange, nil
}

func transformBlockRange(blockRange *RLPBlockRange) error {
	for i := range blockRange.hashes {
		blockNumber := blockRange.start + uint64(i)
		log.Debug("Migrating data", "number", blockNumber)

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

	return nil
}

func writeAncientBlockRange(db ethdb.Database, blockRange *RLPBlockRange) error {
	_, err := db.ModifyAncients(func(aWriter ethdb.AncientWriteOp) error {
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

	return err
}

func writeBlockRange(db ethdb.Database, blockRange *RLPBlockRange) error {
	for i, hashBytes := range blockRange.hashes {
		hash := common.BytesToHash(hashBytes)
		blockNumber := blockRange.start + uint64(i)

		log.Debug("Writing data", "number", blockNumber)

		if err := db.Put(celo1.HeaderHashKey(blockNumber), hashBytes); err != nil {
			return fmt.Errorf("can't write hash to new database: %v", err)
		}
		if err := db.Put(celo1.HeaderKey(blockNumber, hash), blockRange.headers[i]); err != nil {
			return fmt.Errorf("can't write header to new database: %v", err)
		}
		if err := db.Put(celo1.BlockBodyKey(blockNumber, hash), blockRange.bodies[i]); err != nil {
			return fmt.Errorf("can't write body to new database: %v", err)
		}
		if err := db.Put(celo1.BlockReceiptsKey(blockNumber, hash), blockRange.receipts[i]); err != nil {
			return fmt.Errorf("can't write receipts to new database: %v", err)
		}
		if err := db.Put(celo1.HeaderTDKey(blockNumber, hash), blockRange.tds[i]); err != nil {
			return fmt.Errorf("can't write td to new database: %v", err)
		}
		// TODO there are other misc things like this that need to be written
		rawdb.WriteHeaderNumber(db, hash, blockNumber)
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
	celoBody := new(CeloBody)
	if err := rlp.DecodeBytes(oldBodyData, celoBody); err != nil {
		return nil, fmt.Errorf("Failed to RLP decode body: %v", err)
	}

	// TODO(Alec) is this doing everything its supposed to

	// Alternatively, decode into op-geth types.Body structure
	// body := new(types.Body)
	// newBodyData, err := rlp.EncodeToBytes(body)

	// transform into op-geth types.Body structure
	// since Body is a slice of types.Transactions, we can just remove the randomness and epochSnarkData and add empty array for UnclesHashes
	newBodyData, err := rlp.EncodeToBytes([]interface{}{celoBody.Transactions, nil})
	if err != nil {
		log.Crit("Failed to RLP encode body", "err", err)
	}
	// encode the new structure into []byte
	return newBodyData, nil
}

// CeloBody is the body of a celo block
type CeloBody struct {
	Transactions   rlp.RawValue
	Randomness     rlp.RawValue
	EpochSnarkData rlp.RawValue
}

// MustAncientLength returns the number of items in the ancients database
func MustAncientLength(db ethdb.Database) uint64 {
	byteSize, err := db.AncientSize(rawdb.ChainFreezerHashTable)
	if err != nil {
		log.Crit("Failed to get ancient size", "error", err)
	}
	return byteSize / ancientHashSize
}

// GetLastBlockNumber returns the number of the last block in the database
func GetLastBlockNumber(db ethdb.Database) uint64 {
	hash := rawdb.ReadHeadHeaderHash(db)
	if hash == (common.Hash{}) {
		return max(MustAncientLength(db)-1, 0)
	}
	return *rawdb.ReadHeaderNumber(db, hash)
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
