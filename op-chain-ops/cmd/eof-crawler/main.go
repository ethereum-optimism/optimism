package main

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/eof"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "eof-crawler",
		Usage: "Scan a Geth database for EOF-prefixed contracts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "db-path",
				Usage: "Path to the geth LevelDB",
			},
			&cli.StringFlag{
				Name:  "out",
				Value: "eof-contracts.json",
				Usage: "Path to the output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			dbPath := ctx.String("db-path")
			if len(dbPath) == 0 {
				return errors.New("Must specify a db-path")
			}
			out := ctx.String("out")

			return eof.IndexEOFContracts(dbPath, out)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error indexing state", "err", err)
	}
}
