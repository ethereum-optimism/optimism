package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/urfave/cli/v2"
)

func dbOpen(path string, cache int, handles int) (ethdb.Database, error) {
	chaindataPath := filepath.Join(path, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         chaindataPath,
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

func main() {
	app := &cli.App{
		Name:  "boba-migrate",
		Usage: "Write allocation data from the legacy data to a json file for erigon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "db-path",
				Usage: "Path to database",
			},
			&cli.IntFlag{
				Name:  "db-cache",
				Usage: "LevelDB cache size in mb",
				Value: 1024,
			},
			&cli.IntFlag{
				Name:  "db-handles",
				Usage: "LevelDB number of handles",
				Value: 60,
			},
			&cli.StringFlag{
				Name:  "output-path",
				Usage: "Path to output file",
			},
			&cli.Int64Flag{
				Name:  "hardfork-block",
				Usage: "Block number of the hardfork",
				Value: 0,
			},
		},
		Action: func(ctx *cli.Context) error {
			dbPath := ctx.String("db-path")
			dbCache := ctx.Int("db-cache")
			dbHandles := ctx.Int("db-handles")
			outputPath := ctx.String("output-path")
			hardforkBlock := ctx.Int64("hardfork-block")

			db, err := dbOpen(dbPath, dbCache, dbHandles)
			if err != nil {
				return fmt.Errorf("error opening db: %w", err)
			}
			defer db.Close()

			hash := rawdb.ReadHeadHeaderHash(db)
			if hardforkBlock != 0 {
				hash = rawdb.ReadCanonicalHash(db, uint64(hardforkBlock))
			}

			num := rawdb.ReadHeaderNumber(db, hash)
			log.Info("Dumping genesis state", "hash", hash, "number", num)

			header := rawdb.ReadHeader(db, hash, *num)
			config := &state.DumpConfig{
				SkipCode:          false,
				SkipStorage:       false,
				OnlyWithAddresses: false,
				Start:             common.Hash{}.Bytes(),
				Max:               uint64(0),
			}

			statedb, err := state.New(header.Root, state.NewDatabaseWithConfig(db, &triedb.Config{Preimages: true}), nil)
			if err != nil {
				return err
			}

			state := statedb.RawDump(config)
			alloc, err := json.Marshal(state.Accounts)
			if err != nil {
				return err
			}

			f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				log.Warn("Failed to open genesis alloc file", "err", err)
				return err
			}
			defer f.Close()
			_, err = f.Write(alloc)
			if err != nil {
				log.Warn("Failed to write genesis alloc", "err", err)
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error exits", "err", err)
	}
}
