package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/c2h5oh/datasize"
	"github.com/urfave/cli/v2"

	"github.com/ledgerwatch/log/v3"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/datadir"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/node"
	"github.com/ledgerwatch/erigon/node/nodecfg"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/chain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/crossdomain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/genesis"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

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
			logger := log.New()
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
					log.StreamHandler(os.Stdout, log.TerminalFormat()),
				),
			)

			ovmAddresses, err := crossdomain.NewAddresses(ctx.String("ovm-addresses"))
			if err != nil {
				return err
			}

			log.Info(
				"Loaded data",
				"ovmAddresses", len(ovmAddresses),
			)

			migrationData := crossdomain.MigrationData{
				OvmAddresses:  ovmAddresses,
				EvmAddresses:  crossdomain.OVMETHAddresses{},
				OvmAllowances: []*crossdomain.Allowance{},
				OvmMessages:   []*crossdomain.SentMessage{},
				EvmMessages:   []*crossdomain.SentMessage{},
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

			if genesisBlock.Difficulty == nil || genesisBlock.Difficulty.Cmp(common.Big0) == 0 {
				log.Warn("difficulty is not set in genesis config, setting to 1")
				genesisBlock.Difficulty = common.Big1
			}

			// deep copy genesis for later checking
			var genesisBlockOrigin types.Genesis
			genesisByte, err := json.Marshal(genesisBlock)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(genesisByte, &genesisBlockOrigin); err != nil {
				return err
			}

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

			stack, err := node.New(&nodeConfig, logger)
			defer stack.Close()

			chaindb, err := node.OpenDatabase(stack.Config(), kv.ChainDB, logger)
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
			postChaindb, err := node.OpenDatabase(stack.Config(), kv.ChainDB, logger)
			if err != nil {
				log.Error("failed to open post chaindb", "err", err)
				return err
			}
			defer postChaindb.Close()

			tx, err := postChaindb.BeginRo(context.Background())
			if err != nil {
				log.Error("failed to begin write genesis block", "err", err)
				return err
			}
			defer tx.Rollback()

			hash := rawdb.ReadHeadHeaderHash(tx)
			log.Info("Reading chain tip from database", "hash", hash)
			num := rawdb.ReadHeaderNumber(tx, hash)
			if num == nil {
				log.Error("failed to read chain tip from database", "hash", hash)
				return fmt.Errorf("cannot find header number for %s", hash)
			}
			if *num != 0 {
				log.Error("chain tip is not genesis block", "hash", hash)
				return fmt.Errorf("expected chain tip to be block 0, but got %d", *num)
			}

			header := rawdb.ReadHeader(tx, hash, *num)
			log.Info("Read header from database", "number", *num)

			bobaGenesisHash := common.HexToHash(chain.GetBobaGenesisHash(genesisBlockOrigin.Config.ChainID))
			if header.Hash() != bobaGenesisHash {
				log.Error("genesis block hash mismatch", "expected", bobaGenesisHash, "got", header.Hash())
				return fmt.Errorf("genesis block hash mismatch, expected %s, got %s", bobaGenesisHash, header.Hash())
			}

			if err := genesis.PostCheckLegacyETH(tx, &genesisBlockOrigin, migrationData); err != nil {
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
