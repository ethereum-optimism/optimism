package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/db"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "migrate",
		Usage: "Migrate Celo state to a CeL2 genesis DB",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Usage:    "Path to database",
				Required: true,
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Dry run the upgrade by not committing the database",
			},
			cli.BoolFlag{
				Name:  "no-check",
				Usage: "Do not perform sanity checks. This should only be used for testing",
			},
			cli.IntFlag{
				Name:  "db-cache",
				Usage: "LevelDB cache size in mb",
				Value: 1024,
			},
			cli.IntFlag{
				Name:  "db-handles",
				Usage: "LevelDB number of handles",
				Value: 60,
			},
		},
		Action: func(ctx *cli.Context) error {
			dbCache := ctx.Int("db-cache")
			dbHandles := ctx.Int("db-handles")
			dbPath := ctx.String("db-path")
			log.Info("Opening database", "dbCache", dbCache, "dbHandles", dbHandles, "dbPath", dbPath)
			ldb, err := db.Open(dbPath, dbCache, dbHandles)
			if err != nil {
				return fmt.Errorf("cannot open DB: %w", err)
			}

			dryRun := ctx.Bool("dry-run")
			noCheck := ctx.Bool("no-check")
			if noCheck {
				panic("must run with check on")
			}

			// Perform the migration
			_, err = MigrateDB(ldb, !dryRun, noCheck)
			if err != nil {
				return err
			}

			// Close the database handle
			if err := ldb.Close(); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

type MigrationResult struct {
	TransitionHeight    uint64
	TransitionTimestamp uint64
	TransitionBlockHash common.Hash
}

// MigrateDB will migrate an l2geth legacy Optimism database to a Bedrock database.
func MigrateDB(ldb ethdb.Database, commit, noCheck bool) (*MigrationResult, error) {
	log.Info("Migrating DB")
	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)

	// Grab the header number.
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}
	log.Info("Reading chain tip num from database", "number", num)

	// Grab the full header.
	header := rawdb.ReadHeader(ldb, hash, *num)
	trieRoot := header.Root

	log.Info("Read header from database", "number", header)

	// dbFactory := func() (*state.StateDB, error) {
	// 	// Set up the backing store.
	// 	underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
	// 		Preimages: true,
	// 		Cache:     1024,
	// 	})

	// 	// Open up the state database.
	// 	db, err := state.New(header.Root, underlyingDB, nil)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("cannot open StateDB: %w", err)
	// 	}

	// 	return db, nil
	// }

	// db, err := dbFactory()
	// if err != nil {
	// 	return nil, fmt.Errorf("cannot create StateDB: %w", err)
	// }

	// Remove old blocks, so that we start with a fresh genesis block
	currentHash := header.ParentHash
	for {
		// There are no uncles in Celo
		num = rawdb.ReadHeaderNumber(ldb, currentHash)
		hash = rawdb.ReadCanonicalHash(ldb, *num)
		// if header == nil {
		// 	return nil, fmt.Errorf("couldn't find header")
		// }

		log.Info("Deleting block", "hash", currentHash, "c", hash, "number", *num)
		// rawdb.DeleteBlock(ldb, currentHash, *num)
		if *num == 0 {
			break
		}

		header = rawdb.ReadHeader(ldb, currentHash, *num)
		currentHash = header.ParentHash
	}

	log.Info("Successfully cleaned old blocks")

	// We're done messing around with the database, so we can now commit the changes to the DB.
	// Note that this doesn't actually write the changes to disk.
	// log.Info("Committing state DB")
	// newRoot, err := db.Commit(true)
	// if err != nil {
	// 	return nil, err
	// }

	log.Info("Creating new Genesis block")
	// Create the header for the Bedrock transition block.
	cel2Header := &types.Header{
		ParentHash:  common.Hash{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    predeploys.SequencerFeeVaultAddr, // TODO
		Root:        trieRoot,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      common.Big0,
		GasLimit:    (uint64)(20_000_000),
		GasUsed:     0,
		Time:        uint64(12345),
		Extra:       []byte("CeL2"),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Create the Bedrock transition block from the header. Note that there are no transactions,
	// uncle blocks, or receipts in the Bedrock transition block.
	cel2Block := types.NewBlock(cel2Header, nil, nil, nil, trie.NewStackTrie(nil))

	// We did it!
	log.Info(
		"Built Bedrock transition",
		"hash", cel2Block.Hash(),
		"root", cel2Block.Root(),
		"number", cel2Block.NumberU64(),
		"gas-used", cel2Block.GasUsed(),
		"gas-limit", cel2Block.GasLimit(),
	)

	log.Info("Header", "header", cel2Header)
	log.Info("Body", "Body", cel2Block)

	// Create the result of the migration.
	res := &MigrationResult{
		TransitionHeight:    cel2Block.NumberU64(),
		TransitionTimestamp: cel2Block.Time(),
		TransitionBlockHash: cel2Block.Hash(),
	}

	// If we're not actually writing this to disk, then we're done.
	if !commit {
		log.Info("Dry run complete")
		return res, nil
	}

	// Otherwise we need to write the changes to disk. First we commit the state changes.
	// log.Info("Committing trie DB")
	// if err := db.Database().TrieDB().Commit(newRoot, true); err != nil {
	// 	return nil, err
	// }

	// Next we write the Cel2 genesis block to the database.
	rawdb.WriteTd(ldb, cel2Block.Hash(), cel2Block.NumberU64(), cel2Block.Difficulty())
	rawdb.WriteBlock(ldb, cel2Block)
	rawdb.WriteReceipts(ldb, cel2Block.Hash(), cel2Block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, cel2Block.Hash(), cel2Block.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadHeaderHash(ldb, cel2Block.Hash())

	// TODO
	// Make the first Bedrock block a finalized block.
	rawdb.WriteFinalizedBlockHash(ldb, cel2Block.Hash())

	// TODO: need to update chainconfig

	// We're done!
	log.Info(
		"Wrote CeL2 transition block",
		"height", cel2Header.Number,
		"root", cel2Header.Root.String(),
		"hash", cel2Header.Hash().String(),
		"timestamp", cel2Header.Time,
	)

	// Return the result and have a nice day.
	return res, nil
}
