package main

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "migrate",
		Usage: "Migrate a legacy database",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "l1-rpc-url",
				Value: "http://127.0.0.1:8545",
				Usage: "RPC URL for an L1 Node",
			},
			&cli.Uint64Flag{
				Name:  "starting-l1-block-number",
				Usage: "L1 block number to build the L2 genesis from",
			},
			&cli.StringFlag{
				Name:  "ovm-addresses",
				Usage: "Path to ovm-addresses.json",
			},
			&cli.StringFlag{
				Name:  "evm-addresses",
				Usage: "Path to evm-addresses.json",
			},
			&cli.StringFlag{
				Name:  "ovm-allowances",
				Usage: "Path to ovm-allowances.json",
			},
			&cli.StringFlag{
				Name:  "ovm-messages",
				Usage: "Path to ovm-messages.json",
			},
			&cli.StringFlag{
				Name:  "evm-messages",
				Usage: "Path to evm-messages.json",
			},
			&cli.StringFlag{
				Name:  "db-path",
				Usage: "Path to database",
			},
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config file",
			},
			cli.StringFlag{
				Name:  "network",
				Usage: "Name of hardhat deploy network",
			},
			cli.StringFlag{
				Name:  "hardhat-deployments",
				Usage: "Comma separated list of hardhat deployment directories",
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Dry run the upgrade by not committing the database",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			ovmAddresses, err := migration.NewAddresses(ctx.String("ovm-addresses"))
			if err != nil {
				return err
			}
			evmAddresess, err := migration.NewAddresses(ctx.String("evm-addresses"))
			if err != nil {
				return err
			}
			ovmAllowances, err := migration.NewAllowances(ctx.String("ovm-allowances"))
			if err != nil {
				return err
			}
			ovmMessages, err := migration.NewSentMessage(ctx.String("ovm-messages"))
			if err != nil {
				return err
			}
			evmMessages, err := migration.NewSentMessage(ctx.String("evm-messages"))
			if err != nil {
				return err
			}

			migrationData := migration.MigrationData{
				OvmAddresses:  ovmAddresses,
				EvmAddresses:  evmAddresess,
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
			var blockNumber *big.Int
			bnum := ctx.Uint64("starting-l1-block-number")
			if bnum != 0 {
				blockNumber = new(big.Int).SetUint64(bnum)
			}

			block, err := l1Client.BlockByNumber(context.Background(), blockNumber)
			if err != nil {
				return err
			}

			chaindataPath := filepath.Join(ctx.String("db-path"), "geth", "chaindata")
			ancientPath := filepath.Join(ctx.String("db-path"), "ancient")
			ldb, err := rawdb.NewLevelDBDatabaseWithFreezer(chaindataPath, int(1024), int(60), ancientPath, "", true)
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
			if _, err := genesis.MigrateDB(ldb, config, block, &migrationData, !dryRun); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}
