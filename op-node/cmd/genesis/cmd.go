package genesis

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

			l1Genesis, err := genesis.BuildL1DeveloperGenesis(config)
			if err != nil {
				return err
			}

			l1StartBlock := l1Genesis.ToBlock()
			l2Addrs := &genesis.L2Addresses{
				ProxyAdmin:                  predeploys.DevProxyAdminAddr,
				L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
				L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
				L1ERC721BridgeProxy:         predeploys.DevL1ERC721BridgeAddr,
			}
			l2Genesis, err := genesis.BuildL2DeveloperGenesis(config, l1StartBlock, l2Addrs)
			if err != nil {
				return err
			}

			rollupConfig := makeRollupConfig(config, l1StartBlock, l2Genesis, predeploys.DevOptimismPortalAddr)

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
				return err
			}

			depPath, network := filepath.Split(ctx.String("deployment-dir"))
			hh, err := hardhat.New(network, nil, []string{depPath})
			if err != nil {
				return err
			}

			proxyAdmin, err := hh.GetDeployment("ProxyAdmin")
			if err != nil {
				return err
			}
			l1SBP, err := hh.GetDeployment("L1StandardBridgeProxy")
			if err != nil {
				return err
			}
			l1XDMP, err := hh.GetDeployment("L1CrossDomainMessengerProxy")
			if err != nil {
				return err
			}
			portalProxy, err := hh.GetDeployment("OptimismPortalProxy")
			if err != nil {
				return err
			}
			l1ERC721BP, err := hh.GetDeployment("L1ERC721BridgeProxy")
			if err != nil {
				return err
			}

			l2Addrs := &genesis.L2Addresses{
				ProxyAdmin:                  proxyAdmin.Address,
				L1StandardBridgeProxy:       l1SBP.Address,
				L1CrossDomainMessengerProxy: l1XDMP.Address,
				L1ERC721BridgeProxy:         l1ERC721BP.Address,
			}
			l2Genesis, err := genesis.BuildL2DeveloperGenesis(config, l1StartBlock, l2Addrs)
			if err != nil {
				return err
			}

			rollupConfig := makeRollupConfig(config, l1StartBlock, l2Genesis, portalProxy.Address)

			if err := writeGenesisFile(ctx.String("outfile.l2"), l2Genesis); err != nil {
				return err
			}
			return writeGenesisFile(ctx.String("outfile.rollup"), rollupConfig)
		},
	},
}

func makeRollupConfig(
	config *genesis.DeployConfig,
	l1StartBlock *types.Block,
	l2Genesis *core.Genesis,
	portalAddr common.Address,
) *rollup.Config {
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
		},
		BlockTime:              config.L2BlockTime,
		MaxSequencerDrift:      config.MaxSequencerDrift,
		SeqWindowSize:          config.SequencerWindowSize,
		ChannelTimeout:         config.ChannelTimeout,
		L1ChainID:              new(big.Int).SetUint64(config.L1ChainID),
		L2ChainID:              new(big.Int).SetUint64(config.L2ChainID),
		P2PSequencerAddress:    config.P2PSequencerAddress,
		FeeRecipientAddress:    config.OptimismL2FeeRecipient,
		BatchInboxAddress:      config.BatchInboxAddress,
		BatchSenderAddress:     config.BatchSenderAddress,
		DepositContractAddress: portalAddr,
	}
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
