package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/db"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
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
			_, err = genesis.MigrateDB(ldb, !dryRun, noCheck)
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
