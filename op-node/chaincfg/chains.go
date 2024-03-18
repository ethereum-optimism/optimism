package chaincfg

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/superchain-registry/superchain"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var Mainnet, Goerli, Sepolia *rollup.Config

func init() {
	mustCfg := func(name string) *rollup.Config {
		cfg, err := GetRollupConfig(name)
		if err != nil {
			panic(fmt.Errorf("failed to load rollup config %q: %w", name, err))
		}
		return cfg
	}
	Mainnet = mustCfg("op-mainnet")
	Goerli = mustCfg("op-goerli")
	Sepolia = mustCfg("op-sepolia")
}

var L2ChainIDToNetworkDisplayName = func() map[string]string {
	out := make(map[string]string)
	for _, netCfg := range superchain.OPChains {
		out[fmt.Sprintf("%d", netCfg.ChainID)] = netCfg.Name
	}
	return out
}()

// AvailableNetworks returns the selection of network configurations that is available by default.
func AvailableNetworks() []string {
	var networks []string
	for _, cfg := range superchain.OPChains {
		networks = append(networks, cfg.Chain+"-"+cfg.Superchain)
	}
	return networks
}

func handleLegacyName(name string) string {
	switch name {
	case "goerli":
		return "op-goerli"
	case "mainnet":
		return "op-mainnet"
	case "sepolia":
		return "op-sepolia"
	default:
		return name
	}
}

// ChainByName returns a chain, from known available configurations, by name.
// ChainByName returns nil when the chain name is unknown.
func ChainByName(name string) *superchain.ChainConfig {
	// Handle legacy name aliases
	name = handleLegacyName(name)
	for _, chainCfg := range superchain.OPChains {
		if strings.EqualFold(chainCfg.Chain+"-"+chainCfg.Superchain, name) {
			return chainCfg
		}
	}
	return nil
}

func GetRollupConfig(name string) (*rollup.Config, error) {
	chainCfg := ChainByName(name)
	if chainCfg == nil {
		return nil, fmt.Errorf("invalid network: %q", name)
	}
	rollupCfg, err := rollup.LoadOPStackRollupConfig(chainCfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to load rollup config: %w", err)
	}
	return rollupCfg, nil
}
