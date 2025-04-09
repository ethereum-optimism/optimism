package subcmds

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/urfave/cli/v2"
)

var (
	ConfigsChainIDFlag = &cli.StringFlag{
		Name:  "chain-id",
		Usage: "Chain ID to report chain configuration for",
	}
	ConfigsNetworkFlag = &cli.StringFlag{
		Name:  "network",
		Usage: "Network to report chain configuration for",
	}
)

var ConfigsCommand = &cli.Command{
	Name:        "configs",
	Usage:       "List the supported chain configurations",
	Description: "List the supported chain configurations.",
	Action:      ListConfigs,
	Flags: []cli.Flag{
		ConfigsChainIDFlag,
		ConfigsNetworkFlag,
	},
}

func ListConfigs(ctx *cli.Context) error {
	if ctx.IsSet(ConfigsChainIDFlag.Name) {
		chainID, err := eth.ParseDecimalChainID(ctx.String(ConfigsChainIDFlag.Name))
		if err != nil {
			return fmt.Errorf("invalid chain ID: %w", err)
		}
		if err := listChain(chainID); err != nil {
			return err
		}
	}
	if ctx.IsSet(ConfigsNetworkFlag.Name) {
		if err := listNamedChain(ctx.String(ConfigsNetworkFlag.Name)); err != nil {
			return err
		}
	}
	if !ctx.IsSet(ConfigsChainIDFlag.Name) && !ctx.IsSet(ConfigsNetworkFlag.Name) {
		return listAllChains()
	}
	return nil
}

func listAllChains() error {
	chainNames := superchain.ChainNames()
	for _, name := range chainNames {
		if err := listNamedChain(name); err != nil {
			return err
		}
	}
	customChainIDs, err := chainconfig.CustomChainIDs()
	if err != nil {
		return err
	}
	for _, chainID := range customChainIDs {
		if err := listChain(chainID); err != nil {
			return err
		}
	}
	return nil
}

func listNamedChain(name string) error {
	ch := chaincfg.ChainByName(name)
	chainID := eth.ChainIDFromUInt64(ch.ChainID)
	err := listChain(chainID)
	if err != nil {
		return err
	}
	return nil
}

func listChain(chainID eth.ChainID) error {
	cfg, err := chainconfig.RollupConfigByChainID(chainID)
	if err != nil {
		return err
	}
	// Double check the L2 genesis is really available
	_, err = chainconfig.ChainConfigByChainID(chainID)
	if err != nil {
		return err
	}
	description := cfg.Description(chaincfg.L2ChainIDToNetworkDisplayName)
	fmt.Println(description)
	return nil
}
