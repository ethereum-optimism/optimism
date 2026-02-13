package depset

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/superchain"
)

type RegistryFullConfigSetSource struct {
	l1RPCURL      string
	rollupCfgs    []*rollup.Config
	dependencySet DependencySet
}

func NewRegistryFullConfigSetSource(l1RPCURL string, networks []string) (*RegistryFullConfigSetSource, error) {
	rollupCfgs := make([]*rollup.Config, 0, len(networks))
	var dependencySet DependencySet
	for _, network := range networks {
		chainID, err := superchain.ChainIDByName(network)
		if err != nil {
			return nil, err
		}
		// Use the dependency set from the first chain.
		// superchain-registry has checks to ensure consistency for all chains in the same set
		if dependencySet == nil {
			depSet, err := FromRegistry(eth.ChainIDFromUInt64(chainID))
			if err != nil {
				return nil, fmt.Errorf("failed to load dependency set for network %s: %w", network, err)
			}
			dependencySet = depSet
		}

		rollupCfg, err := rollup.LoadOPStackRollupConfig(chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to load rollup config for network %s: %w", network, err)
		}

		rollupCfgs = append(rollupCfgs, rollupCfg)
	}
	return &RegistryFullConfigSetSource{
		l1RPCURL:      l1RPCURL,
		rollupCfgs:    rollupCfgs,
		dependencySet: dependencySet,
	}, nil
}

func (s *RegistryFullConfigSetSource) LoadFullConfigSet(ctx context.Context) (FullConfigSet, error) {
	client, err := ethclient.Dial(s.l1RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	rollupConfigs := make(map[eth.ChainID]*StaticRollupConfig, len(s.rollupCfgs))
	for _, rollupCfg := range s.rollupCfgs {
		l1Genesis, err := client.HeaderByHash(ctx, rollupCfg.Genesis.L1.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 genesis header for hash %s (chainID: %s): %w", rollupCfg.Genesis.L1.Hash, rollupCfg.L2ChainID, err)
		}

		rollupConfigs[eth.ChainIDFromBig(rollupCfg.L2ChainID)] = StaticRollupConfigFromRollupConfig(rollupCfg, l1Genesis.Time)
	}
	return NewFullConfigSetMerged(StaticRollupConfigSet(rollupConfigs), s.dependencySet)
}
