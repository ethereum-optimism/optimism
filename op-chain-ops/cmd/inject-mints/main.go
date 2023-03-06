package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/ethereum-optimism/optimism/op-chain-ops/db"
	"github.com/ethereum-optimism/optimism/op-chain-ops/ether"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli"
)

func main() {
	lvlHdlr := log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd())))
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, lvlHdlr))

	app := &cli.App{
		Name:  "inject-mints",
		Usage: "Injects mints into l2geth witness data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Usage:    "Path to database",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "witness-file-out",
				Usage:    "Path to the witness file",
				Required: true,
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
			ldb, err := db.Open(ctx.String("db-path"), ctx.Int("db-cache"), ctx.Int("db-handles"))
			if err != nil {
				return fmt.Errorf("error opening db: %w", err)
			}
			defer ldb.Close()

			f, err := os.OpenFile(ctx.String("witness-file-out"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("error opening witness file: %w", err)
			}

			log.Info("Reading mint events from DB")
			headBlock := rawdb.ReadHeadBlock(ldb)
			headNum := headBlock.NumberU64()
			seenAddrs := make(map[common.Address]bool)
			bar := progressbar.Default(int64(headNum))
			var count uint64
			progressCb := func(headNum uint64) {
				_ = bar.Add(1)
			}
			err = ether.IterateMintEvents(ldb, headNum, func(address common.Address, headNum uint64) error {
				if seenAddrs[address] {
					return nil
				}
				count++
				seenAddrs[address] = true
				_, err := fmt.Fprintf(f, "ETH|%s\n", address.Hex())
				return err
			}, progressCb)
			if err != nil {
				return fmt.Errorf("error iterating mint events: %w", err)
			}
			log.Info("Done")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in inject-mints", "err", err)
	}
}
