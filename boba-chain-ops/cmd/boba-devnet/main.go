package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/bobanetwork/boba/boba-bindings/hardhat"
	"github.com/bobanetwork/boba/boba-chain-ops/genesis"
	"github.com/ledgerwatch/erigon-lib/kv/memdb"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/rpc"

	"github.com/ledgerwatch/log/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "boba-devnet",
		Usage: "Build genesis.json for Boba devnet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "l1-rpc",
				Usage:    "L1 RPC URL",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "deploy-config",
				Usage:    "Path to hardhat deploy config file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "hardhat-deployments",
				Usage:    "Comma separated list of hardhat deployment directories",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "network",
				Usage:    "Name of hardhat deploy network",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "outfile-l2",
				Usage: "Path to output file for L2 genesis.json",
				Value: "genesis-l2.json",
			},
			&cli.StringFlag{
				Name:  "outfile-rollup",
				Usage: "Path to output file for rollup node",
				Value: "rollup",
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

			network := ctx.String("network")
			deployments := strings.Split(ctx.String("hardhat-deployments"), ",")
			hh, err := hardhat.New(network, []string{}, deployments)
			if err != nil {
				return err
			}

			if err := config.GetDeployedAddresses(hh); err != nil {
				return err
			}

			client, err := rpc.Dial(ctx.String("l1-rpc"), logger)
			if err != nil {
				return fmt.Errorf("cannot dial %s: %w", ctx.String("l1-rpc"), err)
			}
			var l1StartHeader *types.Header
			c, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if config.L1StartingBlockTag.BlockHash != nil {
				err = client.CallContext(c, &l1StartHeader, "eth_getBlockByHash", *config.L1StartingBlockTag.BlockHash, false)
			} else if config.L1StartingBlockTag.BlockNumber != nil {
				err = client.CallContext(c, &l1StartHeader, "eth_getBlockByNumber", big.NewInt(config.L1StartingBlockTag.BlockNumber.Int64()), false)
			}
			if err != nil {
				return fmt.Errorf("error getting l1 start block: %w", err)
			}

			l2Genesis, err := genesis.BuildL2DeveloperGenesis(config, l1StartHeader)
			if err != nil {
				return err
			}

			db := memdb.New("")
			defer db.Close()
			_, block, err := core.CommitGenesisBlock(db, l2Genesis, "", logger)
			if err != nil {
				return err
			}

			rollupConfig, err := config.RollupConfig(l1StartHeader, block.Hash(), block.Number().Uint64())
			if err != nil {
				return err
			}

			l2GenesisOutput := (genesis.GenesisOutput{}).PerformOutput(l2Genesis)
			if err := writeGenesisFile(ctx.String("outfile-l2"), l2GenesisOutput); err != nil {
				return err
			}
			if err := writeGenesisFile(ctx.String("outfile-rollup"), rollupConfig); err != nil {
				return err
			}

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
