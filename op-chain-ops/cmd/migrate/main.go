package main

import (
	"context"
	"errors"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/l2geth/core/rawdb"
	"github.com/ethereum-optimism/optimism/l2geth/core/state"
	"github.com/ethereum-optimism/optimism/l2geth/log"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"

	op_state "github.com/ethereum-optimism/optimism/op-chain-ops/state"
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
				Name:  "l2-addresses",
				Usage: "Path to l2-addresses.json",
			},
			&cli.StringFlag{
				Name:  "db-path",
				Usage: "Path to database",
			},
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config file",
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

			ovmAddresses, err := genesis.NewAddresses(ctx.String("ovm-addresses"))
			if err != nil {
				return err
			}
			evmAddresess, err := genesis.NewAddresses(ctx.String("evm-addresses"))
			if err != nil {
				return err
			}
			ovmAllowances, err := genesis.NewAllowances(ctx.String("ovm-allowances"))
			if err != nil {
				return err
			}
			ovmMessages, err := genesis.NewSentMessage(ctx.String("ovm-messages"))
			if err != nil {
				return err
			}
			evmMessages, err := genesis.NewSentMessage(ctx.String("evm-messages"))
			if err != nil {
				return err
			}

			migrationData := genesis.MigrationData{
				OvmAddresses:  ovmAddresses,
				EvmAddresses:  evmAddresess,
				OvmAllowances: ovmAllowances,
				OvmMessages:   ovmMessages,
				EvmMessages:   evmMessages,
			}

			l2Addrs, err := genesis.NewL2Addresses(ctx.String("l2-addresses"))
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
			ldb, err := rawdb.NewLevelDBDatabase(chaindataPath, 1024, 64, "")
			if err != nil {
				return err
			}

			hash := rawdb.ReadHeadHeaderHash(ldb)
			if err != nil {
				return err
			}
			num := rawdb.ReadHeaderNumber(ldb, hash)
			header := rawdb.ReadHeader(ldb, hash, *num)

			sdb, err := state.New(header.Root, state.NewDatabase(ldb))
			if err != nil {
				return err
			}
			wrappedDB, err := op_state.NewWrappedStateDB(nil, sdb)
			if err != nil {
				return err
			}

			// TODO: think about optimal config, there are a lot of deps
			// regarding changing this
			if config.ProxyAdminOwner != l2Addrs.ProxyAdminOwner {
				return errors.New("mismatched ProxyAdminOwner config")

			}

			if err := genesis.MigrateDB(wrappedDB, config, block, l2Addrs, &migrationData); err != nil {
				return err
			}

			if ctx.Bool("dry-run") {
				log.Info("Dry run complete")
				return nil
			}

			root, err := sdb.Commit(true)
			if err != nil {
				return err
			}
			log.Info("Migration complete", "root", root)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}
