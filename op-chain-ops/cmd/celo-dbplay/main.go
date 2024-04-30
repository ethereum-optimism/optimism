package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/celo1"
	"github.com/mattn/go-isatty"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	dbPathFlag = &cli.StringFlag{
		Name:     "db-path",
		Usage:    "Path to database",
		Required: true,
	}
	dbCacheFlag = &cli.IntFlag{
		Name:  "db-cache",
		Usage: "LevelDB cache size in mb",
		Value: 1024,
	}
	dbHandlesFlag = &cli.IntFlag{
		Name:  "db-handles",
		Usage: "LevelDB number of handles",
		Value: 60,
	}
	dryRunFlag = &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Dry run the upgrade by not committing the database",
	}

	flags = []cli.Flag{
		dbPathFlag,
		dbCacheFlag,
		dbHandlesFlag,
		dryRunFlag,
	}

	// from `packages/contracts-bedrock/deploy-config/internal-devnet.json`
	EIP1559Denominator = uint64(50) // TODO(pl): select values
	EIP1559Elasticity  = uint64(10)
)

var app = &cli.App{
	Name:   "migrate",
	Usage:  "Migrate Celo state to a CeL2 DB",
	Flags:  flags,
	Action: appMain,
}

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))
	if err := app.Run(os.Args); err != nil {
		log.Crit("error", "err", err)
	}
}

func appMain(ctx *cli.Context) error {
	// Write changes to state to actual state database
	dbPath := ctx.String("db-path")
	if dbPath == "" {
		return fmt.Errorf("must specify --db-path")
	}
	dbCache := ctx.Int("db-cache")
	dbHandles := ctx.Int("db-handles")
	// dryRun := ctx.Bool("dry-run")

	log.Info("Opening database", "dbCache", dbCache, "dbHandles", dbHandles, "dbPath", dbPath)
	ldb, err := openCeloDb(dbPath, dbCache, dbHandles)
	if err != nil {
		return fmt.Errorf("cannot open DB: %w", err)
	}
	log.Info("Loaded Celo L1 DB", "db", ldb)

	printStats(ldb)

	findFirstCorruptedHeader(ldb)

	// tryHeader(ldb, 19814000-1) //just before gingerbread activation (alfajores)
	// tryHeader(ldb, 19814000)   // just after gingerbread activation (alfajores)
	// tryHeader(ldb, 4960000-1)  // before churrito block
	// tryHeader(ldb, 4960000)    // churrito block
	// tryHeader(ldb, 4960000)    // donut block
	// tryHeader(ldb, 9472000)    // espresso block
	// tryHeader(ldb, 1)          // espresso block

	return nil
}

func tryHeader(ldb ethdb.Database, number uint64) {
	header, err := celo1.ReadCeloCanonicalHeader(ldb, number)
	if err != nil {
		log.Error("failed to load header", "number", number, "error", err)
	} else {
		log.Info("loaded header", "number", number, "hash", header.Hash())
		log.Info("Read header", "header", header)
	}
}

// Opens a Celo database, stored in the `celo` subfolder
func openCeloDb(path string, cache int, handles int) (ethdb.Database, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	ancientPath := filepath.Join(path, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         path,
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

// print stats about the database
func printStats(ldb ethdb.Database) {
	// Print some stats about the database
	chainMetaData := rawdb.ReadChainMetadata(ldb)
	for _, v := range chainMetaData {
		if len(v) == 2 {
			log.Info("Database Metadata", v[0], v[1])
		} else {
			log.Info("Database Metadata", v[0], v[1:])
		}
	}
}

func canLoadHeader(ldb ethdb.Database, number uint64) bool {
	// log.Trace("Checking if header can be loaded", "number", number)
	_, err := celo1.ReadCeloCanonicalHeader(ldb, number)
	if err != nil {
		log.Debug("failed to load header", "number", number, "error", err)
	}
	return err == nil
}

// does a binary search to find the first header that fails to load
func findFirstCorruptedHeader(ldb ethdb.Database) {
	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	lastBlockNumber := *rawdb.ReadHeaderNumber(ldb, hash)

	log.Info("Starting from HEAD of then chain", "number", lastBlockNumber)

	if !canLoadHeader(ldb, lastBlockNumber) {
		log.Error("Can't fetch the last block header, something is wrong")
		return
	}

	// Binary search from 1 to LastBlockNumber
	low := uint64(1)
	high := lastBlockNumber

	for low <= high {
		mid := (low + high) / 2

		// Call the test condition function to check if the header can be loaded
		if !canLoadHeader(ldb, mid) {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	log.Info("Search Finished", "lastBlockThatLoads", high+1, "firstBlockThatFails", high)
}

// prints the hash of the last x blocks
func printLastBlocks(ldb ethdb.Database, x uint64) {

	hash := rawdb.ReadHeadHeaderHash(ldb)
	lastBlockNumber := *rawdb.ReadHeaderNumber(ldb, hash)
	toBlockNumber := lastBlockNumber - x
	log.Debug("Iterating over blocks", "from", lastBlockNumber, "to", toBlockNumber)
	log.Info("Block", "number", lastBlockNumber, "hash", hash)

	for i := lastBlockNumber; i > toBlockNumber; i-- {
		header := rawdb.ReadHeader(ldb, hash, i)
		log.Info("Block", "number", header.Number, "hash", header.Hash())
		hash = header.ParentHash
	}
}
