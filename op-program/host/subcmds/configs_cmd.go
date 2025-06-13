package subcmds

import (
	"errors"
	"fmt"
	"slices"

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
	Subcommands: []*cli.Command{
		{
			Name:   "check-custom-chains",
			Usage:  "Check that all embedded custom chain configs are valid",
			Action: CheckCustomChains,
		},
	},
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
	if cfg.InteropTime != nil {
		// If interop is scheduled, check the dependency set is available
		_, err = chainconfig.DependencySetByChainID(chainID)
		if err != nil {
			return err
		}
	}
	description := cfg.Description(chaincfg.L2ChainIDToNetworkDisplayName)
	fmt.Println(description)
	return nil
}

func CheckCustomChains(ctx *cli.Context) error {
	customChainIDs, err := chainconfig.CustomChainIDs()
	if err != nil {
		return err
	}

	var errs []error
	interopChains := make(map[eth.ChainID]bool)

	for _, chainID := range customChainIDs {
		cfg, err := chainconfig.RollupConfigByChainID(chainID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		_, err = chainconfig.ChainConfigByChainID(chainID)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if cfg.InteropTime != nil {
			depset, err := chainconfig.DependencySetByChainID(chainID)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, dep := range depset.Chains() {
				interopChains[dep] = true
			}
		}
		if err := listChain(chainID); err != nil {
			return err
		}
	}

	for chainID := range interopChains {
		if !slices.Contains(customChainIDs, chainID) {
			err := listChain(chainID)
			if err != nil {
				errs = append(errs, fmt.Errorf("chain in depset not found in superchain-registry: %w", err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors checking custom chains: %w", errors.Join(errs...))
	}
	return nil
}
