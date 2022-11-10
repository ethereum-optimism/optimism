package genesis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var Subcommands = cli.Commands{
	{
		Name:  "devnet",
		Usage: "Initialize new L1 and L2 genesis files and rollup config suitable for a local devnet",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config file",
			},
			cli.StringFlag{
				Name:  "outfile.l1",
				Usage: "Path to L1 genesis output file",
			},
			cli.StringFlag{
				Name:  "outfile.l2",
				Usage: "Path to L2 genesis output file",
			},
			cli.StringFlag{
				Name:  "outfile.rollup",
				Usage: "Path to rollup output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			// Add the developer L1 addresses to the config
			if err := config.InitDeveloperDeployedAddresses(); err != nil {
				return err
			}

			if err := config.Check(); err != nil {
				return err
			}

			l1Genesis, err := genesis.BuildL1DeveloperGenesis(config)
			if err != nil {
				return err
			}

			l1StartBlock := l1Genesis.ToBlock()
			l2Genesis, err := genesis.BuildL2DeveloperGenesis(config, l1StartBlock)
			if err != nil {
				return err
			}

			rollupConfig, err := makeRollupConfig(config, l1StartBlock, l2Genesis)
			if err != nil {
				return err
			}

			if err := writeGenesisFile(ctx.String("outfile.l1"), l1Genesis); err != nil {
				return err
			}
			if err := writeGenesisFile(ctx.String("outfile.l2"), l2Genesis); err != nil {
				return err
			}
			return writeGenesisFile(ctx.String("outfile.rollup"), rollupConfig)
		},
	},
	{
		Name:  "l2",
		Usage: "Generates an L2 genesis file and rollup config suitable for a deployed network",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "l1-rpc",
				Usage: "L1 RPC URL",
			},
			cli.StringFlag{
				Name:  "deploy-config",
				Usage: "Path to hardhat deploy config file",
			},
			cli.StringFlag{
				Name:  "deployment-dir",
				Usage: "Path to deployment directory",
			},
			cli.StringFlag{
				Name:  "outfile.l2",
				Usage: "Path to L2 genesis output file",
			},
			cli.StringFlag{
				Name:  "outfile.rollup",
				Usage: "Path to rollup output file",
			},
		},
		Action: func(ctx *cli.Context) error {
			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			if config.L1StartingBlockTag == nil {
				return errors.New("must specify a starting block tag in genesis")
			}
			if config.L2GenesisBlockGasLimit == 0 { // TODO: this is a hotfix, need to set default values in more clean way + sanity check the config
				config.L2GenesisBlockGasLimit = 15_000_000
			}

			client, err := ethclient.Dial(ctx.String("l1-rpc"))
			if err != nil {
				return err
			}

			var l1StartBlock *types.Block
			if config.L1StartingBlockTag.BlockHash != nil {
				l1StartBlock, err = client.BlockByHash(context.Background(), *config.L1StartingBlockTag.BlockHash)
			} else if config.L1StartingBlockTag.BlockNumber != nil {
				l1StartBlock, err = client.BlockByNumber(context.Background(), big.NewInt(config.L1StartingBlockTag.BlockNumber.Int64()))
			}
			if err != nil {
				return fmt.Errorf("error getting l1 start block: %w", err)
			}

			depPath, network := filepath.Split(ctx.String("deployment-dir"))
			hh, err := hardhat.New(network, nil, []string{depPath})
			if err != nil {
				return err
			}

			// Read the appropriate deployment addresses from disk
			if err := config.GetDeployedAddresses(hh); err != nil {
				return err
			}
			// Sanity check the config
			if err := config.Check(); err != nil {
				return err
			}
			// Build the developer L2 genesis block
			l2Genesis, err := genesis.BuildL2DeveloperGenesis(config, l1StartBlock)
			if err != nil {
				return fmt.Errorf("error creating l2 developer genesis: %w", err)
			}

			rollupConfig, err := makeRollupConfig(config, l1StartBlock, l2Genesis)
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

func makeRollupConfig(config *genesis.DeployConfig, l1StartBlock *types.Block, l2Genesis *core.Genesis) (*rollup.Config, error) {
	if config.OptimismPortalProxy == (common.Address{}) {
		return nil, errors.New("OptimismPortalProxy cannot be address(0)")
	}
	if config.SystemConfigProxy == (common.Address{}) {
		return nil, errors.New("SystemConfigProxy cannot be address(0)")
	}

	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   l1StartBlock.Hash(),
				Number: l1StartBlock.NumberU64(),
			},
			L2: eth.BlockID{
				Hash:   l2Genesis.ToBlock().Hash(),
				Number: 0,
			},
			L2Time: l1StartBlock.Time(),
			SystemConfig: eth.SystemConfig{
				BatcherAddr: config.BatchSenderAddress,
				Overhead:    eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleOverhead))),
				Scalar:      eth.Bytes32(common.BigToHash(new(big.Int).SetUint64(config.GasPriceOracleScalar))),
				GasLimit:    uint64(config.L2GenesisBlockGasLimit),
			},
		},
		BlockTime:              config.L2BlockTime,
		MaxSequencerDrift:      config.MaxSequencerDrift,
		SeqWindowSize:          config.SequencerWindowSize,
		ChannelTimeout:         config.ChannelTimeout,
		L1ChainID:              new(big.Int).SetUint64(config.L1ChainID),
		L2ChainID:              new(big.Int).SetUint64(config.L2ChainID),
		P2PSequencerAddress:    config.P2PSequencerAddress,
		BatchInboxAddress:      config.BatchInboxAddress,
		DepositContractAddress: config.OptimismPortalProxy,
		L1SystemConfigAddress:  config.SystemConfigProxy,
	}, nil
}

func writeGenesisFile(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
