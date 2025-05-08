package sysgo

import (
	"os"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type ContractPaths struct {
	// must be absolute paths, without file:// prefix
	FoundryArtifacts string
	SourceMap        string
}

type L2Deployment struct {
	systemConfigProxyAddr   common.Address
	disputeGameFactoryProxy common.Address
}

var _ stack.L2Deployment = &L2Deployment{}

func (d *L2Deployment) SystemConfigProxyAddr() common.Address {
	return d.systemConfigProxyAddr
}

func (d *L2Deployment) DisputeGameFactoryProxyAddr() common.Address {
	return d.disputeGameFactoryProxy
}

type SuperchainDeployment struct {
	protocolVersionsAddr common.Address
	superchainConfigAddr common.Address
}

var _ stack.SuperchainDeployment = &SuperchainDeployment{}

func (d *SuperchainDeployment) SuperchainConfigAddr() common.Address {
	return d.superchainConfigAddr
}

func (d *SuperchainDeployment) ProtocolVersionsAddr() common.Address {
	return d.protocolVersionsAddr
}

type Superchain struct {
	id         stack.SuperchainID
	deployment *SuperchainDeployment
}

func (s *Superchain) hydrate(system stack.ExtensibleSystem) {
	sysSuperchain := shim.NewSuperchain(shim.SuperchainConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           s.id,
		Deployment:   s.deployment,
	})
	system.AddSuperchain(sysSuperchain)
}

type Cluster struct {
	id     stack.ClusterID
	depset *depset.StaticConfigDependencySet
}

func (c *Cluster) hydrate(system stack.ExtensibleSystem) {
	sysCluster := shim.NewCluster(shim.ClusterConfig{
		CommonConfig:  shim.NewCommonConfig(system.T()),
		ID:            c.id,
		DependencySet: c.depset,
	})
	system.AddCluster(sysCluster)
}

// WithInteropGen is a system option that will create a L1 chain, superchain, cluster and L2 chains.
func WithInteropGen(l1ID stack.L1NetworkID, superchainID stack.SuperchainID,
	clusterID stack.ClusterID, l2IDs []stack.L2NetworkID, res ContractPaths) stack.Option[*Orchestrator] {

	return stack.Deploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		require.True(l1ID.ChainID().ToBig().IsInt64(), "interop gen uses small chain IDs")
		genesisTime := uint64(time.Now().Add(time.Second * 2).Unix())
		recipe := &interopgen.InteropDevRecipe{
			L1ChainID:        l1ID.ChainID().ToBig().Uint64(),
			L2s:              []interopgen.InteropDevL2Recipe{},
			GenesisTimestamp: genesisTime,
		}
		var ids []eth.ChainID
		for _, l2 := range l2IDs {
			require.True(l2.ChainID().ToBig().IsInt64(), "interop gen uses small chain IDs")
			recipe.L2s = append(recipe.L2s, interopgen.InteropDevL2Recipe{
				ChainID:   l2.ChainID().ToBig().Uint64(),
				BlockTime: 2,
			})
			ids = append(ids, l2.ChainID())
		}
		eth.SortChainID(ids)

		worldCfg, err := recipe.Build(orch.keys)
		require.NoError(err)

		// create a logger for the world configuration
		logger := orch.P().Logger().New("role", "world")
		require.NoError(worldCfg.Check(logger))

		// create the foundry artifacts and source map
		foundryArtifacts := foundry.OpenArtifactsDir(res.FoundryArtifacts)
		sourceMap := foundry.NewSourceMapFS(os.DirFS(res.SourceMap))

		for addr := range worldCfg.L1.Prefund {
			logger.Info("Configuring pre-funded L1 account", "addr", addr)
		}

		// deploy the world, using the logger, foundry artifacts, source map, and world configuration
		worldDeployment, worldOutput, err := interopgen.Deploy(logger, foundryArtifacts, sourceMap, worldCfg)
		require.NoError(err)

		orch.l1Nets.Set(l1ID.ChainID(), &L1Network{
			id:        l1ID,
			genesis:   worldOutput.L1.Genesis,
			blockTime: 6,
		})

		orch.superchains.Set(superchainID, &Superchain{
			id: superchainID,
			deployment: &SuperchainDeployment{
				protocolVersionsAddr: worldDeployment.Superchain.ProtocolVersions,
				superchainConfigAddr: worldDeployment.Superchain.SuperchainConfig,
			},
		})

		depSetContents := make(map[eth.ChainID]*depset.StaticConfigDependency)
		for _, l2Out := range worldOutput.L2s {
			chainID := eth.ChainIDFromBig(l2Out.Genesis.Config.ChainID)
			chainIndex := supervisortypes.ChainIndex(100 + slices.Index(ids, chainID))
			depSetContents[chainID] = &depset.StaticConfigDependency{
				ChainIndex:     chainIndex,
				ActivationTime: 0,
				HistoryMinTime: 0,
			}
		}
		staticDepSet, err := depset.NewStaticConfigDependencySet(depSetContents)
		require.NoError(err)
		orch.clusters.Set(clusterID, &Cluster{
			id:     clusterID,
			depset: staticDepSet,
		})

		for _, l2ID := range l2IDs {
			l2Out, ok := worldOutput.L2s[l2ID.ChainID().String()]
			require.True(ok, "L2 output must exist")
			l2Dep, ok := worldDeployment.L2s[l2ID.ChainID().String()]
			require.True(ok, "L2 deployment must exist")

			l2Net := &L2Network{
				id:        l2ID,
				l1ChainID: l1ID.ChainID(),
				genesis:   l2Out.Genesis,
				rollupCfg: l2Out.RollupCfg,
				deployment: &L2Deployment{
					systemConfigProxyAddr:   l2Dep.SystemConfigProxy,
					disputeGameFactoryProxy: l2Dep.DisputeGameFactoryProxy,
				},
				keys: orch.keys,
			}
			orch.l2Nets.Set(l2ID.ChainID(), l2Net)
		}
	})
}
