package main

import (
	"os"
	"strings"

	ops "github.com/ethereum-optimism/optimism/op-chain-ops"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "surgery",
		Usage: "migrates data from v0 to Bedrock",
		Commands: []*cli.Command{
			{
				Name:  "dump-addresses",
				Usage: "dumps addresses from OVM ETH",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "out-file",
						Aliases:  []string{"o"},
						Usage:    "file to write addresses to",
						Required: true,
					},
				},
				Action: dumpAddressesAction,
			},
			{
				Name:  "migrate",
				Usage: "migrates state in OVM ETH",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "genesis-file",
						Aliases:  []string{"g"},
						Usage:    "path to a genesis file",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "out-dir",
						Aliases:  []string{"o"},
						Usage:    "path to output directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "address-lists",
						Aliases:  []string{"a"},
						Usage:    "comma-separated list of address files to read",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "allowance-lists",
						Aliases:  []string{"l"},
						Usage:    "comma-separated list of allowance lists to read",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "chain-id",
						Usage:    "chain ID",
						Value:    1,
						Required: false,
					},
					&cli.IntFlag{
						Name:     "leveldb-cache-size-mb",
						Usage:    "leveldb cache size in MB",
						Value:    16,
						Required: false,
					},
					&cli.IntFlag{
						Name:     "leveldb-file-handles",
						Usage:    "leveldb file handles",
						Value:    16,
						Required: false,
					},
				},
				Action: migrateAction,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "data-dir",
				Aliases:  []string{"d"},
				Usage:    "data directory to read",
				Required: true,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

func dumpAddressesAction(cliCtx *cli.Context) error {
	dataDir := cliCtx.String("data-dir")
	outFile := cliCtx.String("out-file")
	return ops.DumpAddresses(dataDir, outFile)
}

func migrateAction(cliCtx *cli.Context) error {
	dataDir := cliCtx.String("data-dir")
	outDir := cliCtx.String("out-dir")
	genesisPath := cliCtx.String("genesis-file")
	addressLists := strings.Split(cliCtx.String("address-lists"), ",")
	allowanceLists := strings.Split(cliCtx.String("allowance-lists"), ",")
	chainID := cliCtx.Int("chain-id")
	levelDBCacheSize := cliCtx.Int("leveldb-cache-size-mb")
	levelDBHandles := cliCtx.Int("leveldb-file-handles")

	genesis, err := ops.ReadGenesisFromFile(genesisPath)
	if err != nil {
		return err
	}

	return ops.Migrate(dataDir, outDir, genesis, addressLists, allowanceLists, chainID, levelDBCacheSize, levelDBHandles)
}
