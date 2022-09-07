package genesis

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/urfave/cli"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
)

var Subcommands = cli.Commands{
	{
		Name:  "devnet",
		Usage: "Initialize new L1 and L2 genesis files and rollup config suitable for a local devnet",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "artifacts",
				Usage: "Comma delimited list of hardhat artifact directories",
			},
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
			artifact := ctx.String("artifacts")
			artifacts := strings.Split(artifact, ",")
			hh, err := hardhat.New("", artifacts, nil)
			if err != nil {
				return err
			}

			deployConfig := ctx.String("deploy-config")
			config, err := genesis.NewDeployConfig(deployConfig)
			if err != nil {
				return err
			}

			l1Genesis, err := genesis.BuildL1DeveloperGenesis(hh, config)
			if err != nil {
				return err
			}

			l1StartBlock := l1Genesis.ToBlock()
			l2Addrs := &genesis.L2Addresses{
				ProxyAdmin:                  predeploys.DevProxyAdminAddr,
				L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
				L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
			}
			l2Genesis, err := genesis.BuildL2DeveloperGenesis(hh, config, l1StartBlock, l2Addrs)
			if err != nil {
				return err
			}

			rollupConfig := &rollup.Config{
				Genesis: rollup.Genesis{
					L1: eth.BlockID{
						Hash:   l1StartBlock.Hash(),
						Number: 0,
					},
					L2: eth.BlockID{
						Hash:   l2Genesis.ToBlock().Hash(),
						Number: 0,
					},
					L2Time: uint64(config.L1GenesisBlockTimestamp),
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
				DepositContractAddress: predeploys.DevOptimismPortalAddr,
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
