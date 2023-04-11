package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"

	"github.com/ethereum-optimism/optimism/op-chain-ops/db"
	"github.com/mattn/go-isatty"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/urfave/cli"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "migrate",
		Usage: "Migrate a legacy database",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "l1-rpc-url",
				Value:    "http://127.0.0.1:8545",
				Usage:    "RPC URL for an L1 Node",
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
				Name:     "db-path",
				Usage:    "Path to database",
				Required: true,
			},
			cli.StringFlag{
				Name:     "deploy-config",
				Usage:    "Path to hardhat deploy config file",
				Required: true,
			},
			cli.StringFlag{
				Name:     "network",
				Usage:    "Name of hardhat deploy network",
				Required: true,
			},
			cli.StringFlag{
				Name:     "hardhat-deployments",
				Usage:    "Comma separated list of hardhat deployment directories",
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
			cli.StringFlag{
				Name:     "rollup-config-out",
				Usage:    "Path that op-node config will be written to disk",
				Value:    "rollup.json",
				Required: true,
			},
			cli.BoolFlag{
				Name:     "post-check-only",
				Usage:    "Only perform sanity checks",
				Required: false,
			},
		},
		Action: func(ctx *cli.Context) error {
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

			network := ctx.String("network")
			deployments := strings.Split(ctx.String("hardhat-deployments"), ",")
			hh, err := hardhat.New(network, []string{}, deployments)
			if err != nil {
				return err
			}

			l1RpcURL := ctx.String("l1-rpc-url")
			l1Client, err := ethclient.Dial(l1RpcURL)
			if err != nil {
				return err
			}

			var block *types.Block
			tag := config.L1StartingBlockTag
			if tag == nil {
				return errors.New("l1StartingBlockTag cannot be nil")
			}
			log.Info("Using L1 Starting Block Tag", "tag", tag.String())
			if number, isNumber := tag.Number(); isNumber {
				block, err = l1Client.BlockByNumber(context.Background(), big.NewInt(number.Int64()))
			} else if hash, isHash := tag.Hash(); isHash {
				block, err = l1Client.BlockByHash(context.Background(), hash)
			} else {
				return fmt.Errorf("invalid l1StartingBlockTag in deploy config: %v", tag)
			}
			if err != nil {
				return err
			}

			dbCache := ctx.Int("db-cache")
			dbHandles := ctx.Int("db-handles")
			ldb, err := db.Open(ctx.String("db-path"), dbCache, dbHandles)
			if err != nil {
				return err
			}

			// Read the required deployment addresses from disk if required
			if err := config.GetDeployedAddresses(hh); err != nil {
				return err
			}

			if err := config.Check(); err != nil {
				return err
			}

			dryRun := ctx.Bool("dry-run")
			noCheck := ctx.Bool("no-check")
			// Perform the migration
			res, err := genesis.MigrateDB(ldb, config, block, &migrationData, !dryRun, noCheck)
			if err != nil {
				return err
			}

			// Close the database handle
			if err := ldb.Close(); err != nil {
				return err
			}

			postLDB, err := db.Open(ctx.String("db-path"), dbCache, dbHandles)
			if err != nil {
				return err
			}

			if err := genesis.PostCheckMigratedDB(
				postLDB,
				migrationData,
				&config.L1CrossDomainMessengerProxy,
				config.L1ChainID,
				config.FinalSystemOwner,
				config.ProxyAdminOwner,
				&derive.L1BlockInfo{
					Number:        block.NumberU64(),
					Time:          block.Time(),
					BaseFee:       block.BaseFee(),
					BlockHash:     block.Hash(),
					BatcherAddr:   config.BatchSenderAddress,
					L1FeeOverhead: eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleOverhead))),
					L1FeeScalar:   eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleScalar))),
				},
			); err != nil {
				return err
			}

			if err := postLDB.Close(); err != nil {
				return err
			}

			opNodeConfig, err := config.RollupConfig(block, res.TransitionBlockHash, res.TransitionHeight)
			if err != nil {
				return err
			}

			if err := writeJSON(ctx.String("rollup-config-out"), opNodeConfig); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

func writeJSON(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
