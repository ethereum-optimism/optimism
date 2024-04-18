package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

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
	"github.com/ledgerwatch/erigon/rpc"

	"github.com/bobanetwork/boba/boba-bindings/hardhat"
	"github.com/bobanetwork/boba/boba-chain-ops/crossdomain"
	"github.com/bobanetwork/boba/boba-chain-ops/genesis"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "boba-migrate",
		Usage: "Write allocation data from the legacy data to a json file to erigon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "l1-rpc-url",
				Value:    "http://127.0.0.1:8545",
				Usage:    "RPC URL for an L1 Node",
				Required: true,
			},
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
				Name:     "ovm-messages",
				Usage:    "Path to ovm-messages.json",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "witness-file",
				Usage:    "Path to witness file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "deploy-config",
				Usage:    "Path to contracts config file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "network",
				Usage:    "Name of hardhat deploy network",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "hardhat-deployments",
				Usage:    "Comma separated list of hardhat deployment directories",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "outfile-rollup",
				Usage:    "Path to output file for rollup node",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "outfile-genesis",
				Usage: "Path to output file for genesis",
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

			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			ovmAddresses, err := crossdomain.NewAddresses(ctx.String("ovm-addresses"))
			if err != nil {
				return err
			}
			ovmAllowances, err := crossdomain.NewAllowances(ctx.String("ovm-allowances"))
			if err != nil {
				return err
			}
			ovmMessages, err := crossdomain.NewSentMessageFromJSON(ctx.String("ovm-messages"))
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
				"ovmMessages", len(ovmMessages),
				"evmMessages", len(evmMessages),
			)

			migrationData := crossdomain.MigrationData{
				OvmAddresses:  ovmAddresses,
				EvmAddresses:  evmAddresses,
				OvmAllowances: ovmAllowances,
				OvmMessages:   ovmMessages,
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

			if genesisBlock.Difficulty == nil || genesisBlock.Difficulty.Cmp(common.Big0) == 0 {
				log.Warn("difficulty is not set in genesis config, setting to 1")
				genesisBlock.Difficulty = common.Big1
			}

			// deep copy genesis for later checking
			var transitionBlockOrigin types.Genesis
			genesisByte, err := json.Marshal(genesisBlock)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(genesisByte, &transitionBlockOrigin); err != nil {
				return err
			}

			var hh *hardhat.Hardhat
			network := ctx.String("network")
			deployments := strings.Split(ctx.String("hardhat-deployments"), ",")
			if network != "" && len(deployments) != 0 {
				hh, err = hardhat.New(network, []string{}, deployments)
				if err != nil {
					return err
				}
			}

			l1RpcURL := ctx.String("l1-rpc-url")
			l1Client, err := rpc.Dial(l1RpcURL, logger)
			if err != nil {
				return err
			}
			defer l1Client.Close()

			var header *types.Header
			tag := config.L1StartingBlockTag
			if tag == nil {
				return errors.New("l1StartingBlockTag cannot be nil")
			}
			log.Info("Using L1 Starting Block Tag", "tag", tag.String())
			if number, isNumber := tag.Number(); isNumber {
				err = l1Client.Call(&header, "eth_getBlockByNumber", big.NewInt(number.Int64()), false)
			} else if hash, isHash := tag.Hash(); isHash {
				err = l1Client.Call(&header, "eth_getBlockByHash", hash, false)
			} else {
				return fmt.Errorf("invalid l1StartingBlockTag in deploy config: %v", tag)
			}
			if err != nil {
				return err
			}
			if header.Number == nil {
				return fmt.Errorf("invalid l1StartingBlockTag in deploy config: %v", tag)
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

			stack, err := node.New(context.Background(), &nodeConfig, logger)
			if err != nil {
				log.Error("failed to create node", "err", err)
				return err
			}
			defer stack.Close()

			chaindb, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
			if err != nil {
				log.Error("failed to open chaindb", "err", err)
				return err
			}

			// Read the required deployment addresses from disk if required
			if hh != nil {
				log.Info("reading deployed addresses from hardhat")
				if err := config.GetDeployedAddresses(hh); err != nil {
					return err
				}
			} else {
				log.Info("reading deployed addresses from config")
				if err := config.InitDeveloperDeployedAddresses(); err != nil {
					return err
				}
			}

			if err := config.Check(); err != nil {
				return err
			}

			dryRun := ctx.Bool("dry-run")
			noCheck := ctx.Bool("no-check")

			block, err := genesis.MigrateDB(chaindb, genesisBlock, config, header, &migrationData, !dryRun, noCheck)
			if err != nil {
				if err.Error() != "cannot write genesis: cannot write genesis: genesis block already exists" {
					log.Error("failed to migrate db", "err", err)
					return err
				} else {
					log.Info("skipping migration, running post migration checks")
				}
			}

			// close the database handle
			chaindb.Close()

			if ctx.String("outfile-genesis") != "" {
				if err := writeGenesisFile(ctx.String("outfile-genesis"), genesisBlock); err != nil {
					return err
				}
			}

			postChaindb, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
			if err != nil {
				log.Error("failed to open post chaindb", "err", err)
				return err
			}
			defer postChaindb.Close()

			if err := genesis.PostCheckMigratedDB(
				postChaindb,
				&transitionBlockOrigin,
				migrationData,
				&config.L1CrossDomainMessengerProxy,
				&config.L1StandardBridgeProxy,
				&config.L1ERC721BridgeProxy,
				config.L1ChainID,
				config.FinalSystemOwner,
				config.ProxyAdminOwner,
				config.L2OutputOracleStartingBlockNumber,
				config.L2OutputOracleStartingTimestamp,
				&genesis.L1BlockInfo{
					Number:        header.Number.Uint64(),
					Time:          header.Time,
					BaseFee:       header.BaseFee,
					BlockHash:     header.Hash(),
					BatcherAddr:   config.BatchSenderAddress,
					L1FeeOverhead: common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleOverhead)),
					L1FeeScalar:   common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleScalar)),
				},
			); err != nil {
				return err
			}

			if block == nil {
				tx, err := postChaindb.BeginRo(context.Background())
				if err != nil {
					log.Error("failed to read DB", "err", err)
					return err
				}
				defer tx.Rollback()

				header := rawdb.ReadHeaderByNumber(tx, config.L2OutputOracleStartingBlockNumber)
				if err != nil {
					log.Error("failed to read header", "err", err)
					return err
				}
				block = rawdb.ReadBlock(tx, header.Hash(), header.Number.Uint64())
				if block == nil {
					log.Error("failed to read block", "err", err)
					return err
				}
			}

			rollupConfig, err := config.RollupConfig(header, block.Hash(), block.Number().Uint64())
			if err != nil {
				return err
			}

			if err := writeGenesisFile(ctx.String("outfile-rollup"), rollupConfig); err != nil {
				return err
			}

			log.Info("Successfully migrated database. The genesis block hash is", "hash", block.Hash().String())

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error exits", "err", err)
	}
}

func writeGenesisFile(outfile string, input any) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
