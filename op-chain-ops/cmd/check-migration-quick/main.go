package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-chain-ops/db"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "check-migration-quick",
		Usage: "Quick check for a migrated database that only checks the header magic in the extradata",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Usage:    "Path to database",
				Required: true,
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
		},
		Action: func(ctx *cli.Context) error {
			dbCache := ctx.Int("db-cache")
			dbHandles := ctx.Int("db-handles")

			ldb, err := db.Open(ctx.String("db-path"), dbCache, dbHandles)
			if err != nil {
				return err
			}

			hash := rawdb.ReadHeadHeaderHash(ldb)
			log.Info("Reading chain tip from database", "hash", hash)
			num := rawdb.ReadHeaderNumber(ldb, hash)
			if num == nil {
				return fmt.Errorf("cannot find header number for %s", hash)
			}

			header := rawdb.ReadHeader(ldb, hash, *num)
			log.Info("Read header from database", "number", *num)

			log.Info(
				"Header info",
				"parentHash", header.ParentHash.Hex(),
				"root", header.Root.Hex(),
				"number", header.Number,
				"gasLimit", header.GasLimit,
				"time", header.Time,
				"extra", hexutil.Encode(header.Extra),
			)

			if !bytes.Equal(header.Extra, genesis.BedrockTransitionBlockExtraData) {
				return fmt.Errorf("expected extra data to be %x, but got %x", genesis.BedrockTransitionBlockExtraData, header.Extra)
			}

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
