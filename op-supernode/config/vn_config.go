package config

import (
	"flag"
	"fmt"
	"strings"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func VirtualNodeConfigs(ctx *cli.Context, vnFlags flags.VNFlagMap, log gethlog.Logger) (map[types.ChainID]*opnodecfg.Config, error) {
	vnCfgs := make(map[types.ChainID]*opnodecfg.Config)
	chains := ctx.Uint64Slice(flags.ChainsFlag.Name)
	// initialize flag sets for each chain, with all op-node flags
	flagSets := make(map[types.ChainID]*flag.FlagSet)
	for _, chainID := range chains {
		flagSets[types.ChainID(chainID)] = flag.NewFlagSet("", flag.ContinueOnError)
		for _, f := range opnodeflags.Flags {
			if err := f.Apply(flagSets[types.ChainID(chainID)]); err != nil {
				return nil, fmt.Errorf("failed to apply flag for chain %d: %w", chainID, err)
			}
		}
	}

	// proliferate vn.* flags to the appropriate chains
	for name, value := range vnFlags {
		for _, chainID := range chains {
			if appliesTo(name, types.ChainID(chainID)) {
				stripped := stripVNPrefix(name, types.ChainID(chainID))
				err := flagSets[types.ChainID(chainID)].Set(stripped, value)
				if err != nil {
					return nil, fmt.Errorf("failed to set flag %s for chain %d: %w", stripped, chainID, err)
				}
			}
		}
	}

	// create cli contexts for each chain with the appropriate flags
	// and create virtual node configs from them
	for chainID, fs := range flagSets {
		cliCtx := cli.NewContext(ctx.App, fs, ctx)
		vnCfg, err := opnode.NewConfig(cliCtx, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create node config for chain %d: %w", chainID, err)
		}
		vnCfgs[chainID] = vnCfg
	}

	// return the virtual node configs
	return vnCfgs, nil
}

func appliesTo(name string, chainID types.ChainID) bool {
	if strings.HasPrefix(name, "vn.all.") {
		return true
	}
	if strings.HasPrefix(name, fmt.Sprintf("vn.%d.", chainID)) {
		return true
	}
	return false
}

// stripVNPrefix removes the vn.all. or vn.<chainID>. prefix from a flag name
// Returns the remaining flag name, or empty string if it doesn't match
func stripVNPrefix(flag string, chainID types.ChainID) string {
	if strings.HasPrefix(flag, flags.VNFlagGlobalPrefix) {
		return strings.TrimPrefix(flag, flags.VNFlagGlobalPrefix)
	}
	chainPrefix := fmt.Sprintf("%s%d.", flags.VNFlagNamePrefix, chainID)
	if strings.HasPrefix(flag, chainPrefix) {
		return strings.TrimPrefix(flag, chainPrefix)
	}
	return ""
}
