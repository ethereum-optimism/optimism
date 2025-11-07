package sysgo

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/superutil"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

// WithL2NetworkFromSuperchainRegistry creates an L2 network using the rollup config from the superchain registry
func WithL2NetworkFromSuperchainRegistry(l2NetworkID stack.L2NetworkID, networkName string) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2NetworkID))
		require := p.Require()

		// Load the rollup config from the superchain registry
		rollupCfg, err := chaincfg.GetRollupConfig(networkName)
		require.NoError(err, "failed to load rollup config for network %s", networkName)

		// Get the chain config from the superchain registry
		chainCfg := chaincfg.ChainByName(networkName)
		require.NotNil(chainCfg, "chain config not found for network %s", networkName)

		// Load the chain config using superutil
		paramsChainConfig, err := superutil.LoadOPStackChainConfigFromChainID(chainCfg.ChainID)
		require.NoError(err, "failed to load chain config for network %s", networkName)

		// Create a genesis config from the chain config
		genesis := &core.Genesis{
			Config: paramsChainConfig,
		}

		// Create the L2 network
		l2Net := &L2Network{
			id:        l2NetworkID,
			l1ChainID: eth.ChainIDFromBig(rollupCfg.L1ChainID),
			genesis:   genesis,
			rollupCfg: rollupCfg,
			keys:      orch.keys,
		}

		require.True(orch.l2Nets.SetIfMissing(l2NetworkID.ChainID(), l2Net),
			fmt.Sprintf("must not already exist: %s", l2NetworkID))
	})
}

// WithEmptyDepSet creates an L2 network using the rollup config from the superchain registry
func WithEmptyDepSet(l2NetworkID stack.L2NetworkID, networkName string) stack.Option[*Orchestrator] {
	return stack.Combine(
		WithL2NetworkFromSuperchainRegistry(l2NetworkID, networkName),
		stack.BeforeDeploy(func(orch *Orchestrator) {
			p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2NetworkID))
			require := p.Require()

			// Check that chain config is available in registry
			chainCfg := chaincfg.ChainByName(networkName)
			require.NotNil(chainCfg, "chain config not found for network %s", networkName)

			// Create a minimal cluster with empty dependency set
			clusterID := stack.ClusterID(networkName)
			cluster := &Cluster{
				id:     clusterID,
				cfgset: depset.FullConfigSetMerged{},
			}

			orch.clusters.Set(clusterID, cluster)
		}),
	)
}
