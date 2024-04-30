package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
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

	if err = parMigrateRange(oldDB, newDB, migratedRecords, toMigrateRecords, *batchSize); err != nil {
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

// seqMigrateRange migrates ancient data from the old database to the new database sequentially
func seqMigrateRange(oldDb ethdb.Database, newDb ethdb.Database, number, count uint64) error {
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

type RLPBlockRange struct {
	start    uint64
	hashes   [][]byte
	headers  [][]byte
	bodies   [][]byte
	receipts [][]byte
	tds      [][]byte
}

// parMigrateRange migrates ancient data from the old database to the new database in parallel
func parMigrateRange(oldDb ethdb.Database, newDb ethdb.Database, start, end, step uint64) error {
	g, ctx := errgroup.WithContext(context.Background())
	readChan := make(chan RLPBlockRange, 10)
	transformChan := make(chan RLPBlockRange, 10)

	g.Go(func() error { return readBlocks(ctx, oldDb, start, end, step, readChan) })
	g.Go(func() error { return transformBlocks(ctx, readChan, transformChan) })
	g.Go(func() error { return writeAncients(ctx, newDb, transformChan) })

	return g.Wait()
}

func readBlocks(ctx context.Context, oldDb ethdb.Database, startBlock, endBlock, batchSize uint64, out chan<- RLPBlockRange) error {
	defer close(out)
	// Read blocks and send them to the out channel
	// This could be reading from a database, a file, etc.
	for i := startBlock; i < endBlock; i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var block RLPBlockRange
			var err error

			count := min(batchSize, endBlock-i+1)
			log.Info("Reading blocks", "start", i, "end", i+count-1, "count", count)
			block.start = i
			block.hashes, err = oldDb.AncientRange(rawdb.ChainFreezerHashTable, i, count, 0)
			if err != nil {
				log.Error("can't read hashes from old freezer", "err", err)
				return err
			}

			block.headers, err = oldDb.AncientRange(rawdb.ChainFreezerHeaderTable, i, count, 0)
			if err != nil {
				log.Error("can't read headers from old freezer", "err", err)
				return err
			}

			block.bodies, err = oldDb.AncientRange(rawdb.ChainFreezerBodiesTable, i, count, 0)
			if err != nil {
				log.Error("can't read bodies from old freezer", "err", err)
				return err
			}

			block.receipts, err = oldDb.AncientRange(rawdb.ChainFreezerReceiptTable, i, count, 0)
			if err != nil {
				log.Error("can't read receipts from old freezer", "err", err)
				return err
			}

			block.tds, err = oldDb.AncientRange(rawdb.ChainFreezerDifficultyTable, i, count, 0)
			if err != nil {
				log.Error("can't read tds from old freezer", "err", err)
				return err
			}

			out <- block
		}
	}
	return nil
}

func transformBlocks(ctx context.Context, in <-chan RLPBlockRange, out chan<- RLPBlockRange) error {
	// Transform blocks from the in channel and send them to the out channel
	defer close(out)
	for block := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Transform headers
			for i, header := range block.headers {
				newHeader, err := transformHeader(header)
				if err != nil {
					log.Error("Failed to transform header", "err", err)
					return err
				}

				// Check that hashing the new header gives the same hash as the saved hash
				newHash := crypto.Keccak256Hash(newHeader)
				if !bytes.Equal(block.hashes[i], newHash.Bytes()) {
					log.Error("Hash mismatch", "block", block.start+uint64(i), "oldHash", common.BytesToHash(block.hashes[i]), "newHash", newHash)
					return fmt.Errorf("hash mismatch at block %d", block.start+uint64(i))
				}
				block.headers[i] = newHeader
			}
			// Transform bodies
			for i, body := range block.bodies {
				newBody, err := transformBlockBody(body)
				if err != nil {
					log.Error("Failed to transform body", "err", err)
					return err
				}
				block.bodies[i] = newBody
			}
			out <- block
		}
	}
	return nil
}

func writeAncients(ctx context.Context, newDb ethdb.Database, in <-chan RLPBlockRange) error {
	// Write blocks from the in channel to the newDb
	for block := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := newDb.ModifyAncients(func(aWriter ethdb.AncientWriteOp) error {
				for i := range block.hashes {
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHashTable, block.start+uint64(i), block.hashes[i]); err != nil {
						log.Error("can't write hash to Freezer", "err", err)
						return err
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerHeaderTable, block.start+uint64(i), block.headers[i]); err != nil {
						log.Error("can't write header to Freezer", "err", err)
						return err
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerBodiesTable, block.start+uint64(i), block.bodies[i]); err != nil {
						log.Error("can't write body to Freezer", "err", err)
						return err
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerReceiptTable, block.start+uint64(i), block.receipts[i]); err != nil {
						log.Error("can't write receipts to Freezer", "err", err)
						return err
					}
					if err := aWriter.AppendRaw(rawdb.ChainFreezerDifficultyTable, block.start+uint64(i), block.tds[i]); err != nil {
						log.Error("can't write td to Freezer", "err", err)
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Error("Failed to write ancients", "err", err)
				return err
			}
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
	celoBody := new(CeloBody)
	if err := rlp.DecodeBytes(oldBodyData, celoBody); err != nil {
		log.Error("Invalid block body RLP", "err", err)
		return nil, err
	}

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
