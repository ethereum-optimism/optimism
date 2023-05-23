package main

import (
	"context"
	"os"

	"github.com/c2h5oh/datasize"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ledgerwatch/erigon-lib/common/datadir"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/boba-chain-ops/crossdomain"
	"github.com/ledgerwatch/erigon/boba-chain-ops/genesis"
	"github.com/ledgerwatch/erigon/node"
	"github.com/ledgerwatch/erigon/node/nodecfg"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "boba-rollover",
		Usage: "Write allocation data from the legacy data to a json file to erigon ignoring the migration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db-path",
				Usage:    "Path to database",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "alloc-path",
				Usage:    "Path to the alloc file from the legacy data",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "genesis-config-path",
				Usage:    "Path to the genesis config file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "ovm-addresses",
				Usage:    "Path to ovm-addresses.json",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "ovm-allowances",
				Usage:    "Path to ovm-allowances.json",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "witness-file",
				Usage:    "Path to witness file",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Dry run the upgrade by not committing the database",
			},
			&cli.BoolFlag{
				Name:  "no-check",
				Usage: "Do not perform sanity checks. This should only be used for testing",
			},
			&cli.StringFlag{
				Name:  "db-size-limit",
				Usage: "Maximum size of the mdbx database.",
				Value: (8 * datasize.TB).String(),
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level",
				Value: "info",
			},
		},
		Action: func(ctx *cli.Context) error {
			logLevel, err := log.LvlFromString(ctx.String("log-level"))
			if err != nil {
				logLevel = log.LvlInfo
				if ctx.String("log-level") != "" {
					log.Warn("invalid server.log_level set: " + ctx.String("log-level"))
				}
			}
			log.Root().SetHandler(
				log.LvlFilterHandler(
					logLevel,
					log.StreamHandler(os.Stdout, log.TerminalFormat(isatty.IsTerminal(os.Stdout.Fd()))),
				),
			)

			ovmAddresses, err := crossdomain.NewAddresses(ctx.String("ovm-addresses"))
			if err != nil {
				return err
			}
			ovmAllowances, err := crossdomain.NewAllowances(ctx.String("ovm-allowances"))
			if err != nil {
				return err
			}
			evmMessages, evmAddresses, err := crossdomain.ReadWitnessData(ctx.String("witness-file"))
			if err != nil {
				return err
			}

			log.Info(
				"Loaded witness data",
				"ovmAddresses", len(ovmAddresses),
				"evmAddresses", len(evmAddresses),
				"ovmAllowances", len(ovmAllowances),
				"evmMessages", len(evmMessages),
			)

			migrationData := crossdomain.MigrationData{
				OvmAddresses:  ovmAddresses,
				EvmAddresses:  evmAddresses,
				OvmAllowances: ovmAllowances,
				OvmMessages:   []*crossdomain.SentMessage{},
				EvmMessages:   evmMessages,
			}

			genesisAlloc, err := genesis.NewAlloc(ctx.String("alloc-path"))
			if err != nil {
				return err
			}
			genesisBlock, err := genesis.NewGenesis(ctx.String("genesis-config-path"))
			if err != nil {
				return err
			}
			genesisBlock.Alloc = *genesisAlloc

			dbPath := ctx.String("db-path")
			mdbxDBSize := ctx.String("db-size-limit")

			// Open and initialise both full and light databases
			nodeConfig := nodecfg.DefaultConfig
			if err := nodeConfig.MdbxDBSizeLimit.UnmarshalText([]byte(mdbxDBSize)); err != nil {
				log.Error("failed to parse mdbx db size limit", "err", err)
				return err
			}
			szLimit := nodeConfig.MdbxDBSizeLimit.Bytes()
			if szLimit%256 != 0 || szLimit < 256 {
				log.Error("mdbx db size limit must be a multiple of 256 bytes and at least 256 bytes", "limit", szLimit)
				return err
			}
			nodeConfig.Dirs = datadir.New(dbPath)

			stack, err := node.New(&nodeConfig)
			defer stack.Close()

			chaindb, err := node.OpenDatabase(stack.Config(), kv.ChainDB)
			if err != nil {
				log.Error("failed to open chaindb", "err", err)
				return err
			}

			dryRun := ctx.Bool("dry-run")
			noCheck := ctx.Bool("no-check")

			if err := genesis.RolloverDB(chaindb, genesisBlock, &migrationData, !dryRun, noCheck); err != nil {
				if err.Error() == "genesis block already exists" {
					log.Info("genesis block already exists")
				} else {
					log.Error("failed to write genesis", "err", err)
					return err
				}
			}

			// close the database handle
			chaindb.Close()

			// post check
			postChaindb, err := node.OpenDatabase(stack.Config(), kv.ChainDB)
			if err != nil {
				log.Error("failed to open post chaindb", "err", err)
				return err
			}
			defer postChaindb.Close()

			tx, err := postChaindb.BeginRw(context.Background())
			if err != nil {
				log.Error("failed to begin write genesis block", "err", err)
				return err
			}
			defer tx.Rollback()

			if err := genesis.PostCheckLegacyETH(tx, genesisBlock, migrationData); err != nil {
				log.Error("failed to post check legacy eth", "err", err)
				return err
			}

			log.Info("Legacy ETH migration complete")

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error exits", "err", err)
	}
}
