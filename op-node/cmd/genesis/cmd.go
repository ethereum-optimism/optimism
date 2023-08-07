package genesis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

var Subcommands = cli.Commands{
	{
		Name:  "l1",
		Usage: "Generates a L1 genesis state file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "deploy-config",
				Usage:    "Path to hardhat deploy config file",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "l1-allocs",
				Usage: "Path to L1 genesis state dump",
			},
			&cli.StringFlag{
				Name:  "l1-deployments",
				Usage: "Path to L1 deployments file",
			},
			&cli.StringFlag{
				Name:  "outfile.l1",
				Usage: "Path to L1 genesis output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			var deployments *genesis.L1Deployments
			if l1Deployments := ctx.String("l1-deployments"); l1Deployments != "" {
				deployments, err = genesis.NewL1Deployments(l1Deployments)
				if err != nil {
					return err
				}
			}

			if deployments != nil {
				config.SetDeployments(deployments)
			}

			// Do the check after setting the deployments
			if err := config.Check(); err != nil {
				return fmt.Errorf("deploy config at %s invalid: %w", deployConfig, err)
			}

			var dump *state.Dump
			if l1Allocs := ctx.String("l1-allocs"); l1Allocs != "" {
				dump, err = genesis.NewStateDump(l1Allocs)
				if err != nil {
					return err
				}
			}

			l1Genesis, err := genesis.BuildL1DeveloperGenesis(config, dump, deployments, true)
			if err != nil {
				return err
			}

			return writeGenesisFile(ctx.String("outfile.l1"), l1Genesis)
		},
	},
	{
		Name:  "l2",
		Usage: "Generates an L2 genesis file and rollup config suitable for a deployed network",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "l1-rpc",
				Usage: "L1 RPC URL",
			},
			&cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to deploy config file",
			},
			&cli.StringFlag{
				Name:  "deployment-dir",
				Usage: "Path to network deployment directory",
			},
			&cli.StringFlag{
				Name:  "outfile.l2",
				Usage: "Path to L2 genesis output file",
			},
			&cli.StringFlag{
				Name:  "outfile.rollup",
				Usage: "Path to rollup output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			log.Info("Deploy config", "path", deployConfig)
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			deployDir := ctx.String("deployment-dir")
			if deployDir == "" {
				return errors.New("Must specify --deployment-dir")
			}

			log.Info("Deployment directory", "path", deployDir)
			depPath, network := filepath.Split(deployDir)
			hh, err := hardhat.New(network, nil, []string{depPath})
			if err != nil {
				return err
			}

			// Read the appropriate deployment addresses from disk
			if err := config.GetDeployedAddresses(hh); err != nil {
				return err
			}

			client, err := ethclient.Dial(ctx.String("l1-rpc"))
			if err != nil {
				return fmt.Errorf("cannot dial %s: %w", ctx.String("l1-rpc"), err)
			}

			var l1StartBlock *types.Block
			if config.L1StartingBlockTag == nil {
				l1StartBlock, err = client.BlockByNumber(context.Background(), nil)
				tag := rpc.BlockNumberOrHashWithHash(l1StartBlock.Hash(), true)
				config.L1StartingBlockTag = (*genesis.MarshalableRPCBlockNumberOrHash)(&tag)
			} else if config.L1StartingBlockTag.BlockHash != nil {
				l1StartBlock, err = client.BlockByHash(context.Background(), *config.L1StartingBlockTag.BlockHash)
			} else if config.L1StartingBlockTag.BlockNumber != nil {
				l1StartBlock, err = client.BlockByNumber(context.Background(), big.NewInt(config.L1StartingBlockTag.BlockNumber.Int64()))
			}
			if err != nil {
				return fmt.Errorf("error getting l1 start block: %w", err)
			}

			// Sanity check the config. Do this after filling in the L1StartingBlockTag
			// if it is not defined.
			if err := config.Check(); err != nil {
				return err
			}

			log.Info("Using L1 Start Block", "number", l1StartBlock.Number(), "hash", l1StartBlock.Hash().Hex())

			// Build the L2 genesis block
			l2Genesis, err := genesis.BuildL2Genesis(config, l1StartBlock)
			if err != nil {
				return fmt.Errorf("error creating l2 genesis: %w", err)
			}

			l2GenesisBlock := l2Genesis.ToBlock()
			rollupConfig, err := config.RollupConfig(l1StartBlock, l2GenesisBlock.Hash(), l2GenesisBlock.Number().Uint64())
			if err != nil {
				return err
			}
			if err := rollupConfig.Check(); err != nil {
				return fmt.Errorf("generated rollup config does not pass validation: %w", err)
			}

			if err := writeGenesisFile(ctx.String("outfile.l2"), l2Genesis); err != nil {
				return err
			}
			return writeGenesisFile(ctx.String("outfile.rollup"), rollupConfig)
		},
	},
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
