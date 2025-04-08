package sysgo

import (
	"os"
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

// WithInteropGen is a system option that will create a L1 chain, superchain, cluster and L2 chains.
func WithInteropGen(l1ID stack.L1NetworkID, superchainID stack.SuperchainID,
	clusterID stack.ClusterID, l2IDs []stack.L2NetworkID, res ContractPaths) stack.Option {

	return func(setup *stack.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)

		setup.Require.True(l1ID.ChainID.ToBig().IsInt64(), "interop gen uses small chain IDs")
		genesisTime := uint64(time.Now().Add(time.Second * 2).Unix())
		recipe := &interopgen.InteropDevRecipe{
			L1ChainID:        l1ID.ChainID.ToBig().Uint64(),
			L2s:              []interopgen.InteropDevL2Recipe{},
			GenesisTimestamp: genesisTime,
		}
		for _, l2 := range l2IDs {
			setup.Require.True(l2.ChainID.ToBig().IsInt64(), "interop gen uses small chain IDs")
			recipe.L2s = append(recipe.L2s, interopgen.InteropDevL2Recipe{
				ChainID:   l2.ChainID.ToBig().Uint64(),
				BlockTime: 2,
			})
		}

		worldCfg, err := recipe.Build(orch.keys)
		setup.Require.NoError(err)

		// create a logger for the world configuration
		logger := setup.Log.New("role", "world")
		setup.Require.NoError(worldCfg.Check(logger))

		// create the foundry artifacts and source map
		foundryArtifacts := foundry.OpenArtifactsDir(res.FoundryArtifacts)
		sourceMap := foundry.NewSourceMapFS(os.DirFS(res.SourceMap))

		for addr := range worldCfg.L1.Prefund {
			logger.Info("Configuring pre-funded L1 account", "addr", addr)
		}

		// deploy the world, using the logger, foundry artifacts, source map, and world configuration
		worldDeployment, worldOutput, err := interopgen.Deploy(logger, foundryArtifacts, sourceMap, worldCfg)
		setup.Require.NoError(err)

		l1Net := &L1Network{
			genesis:   worldOutput.L1.Genesis,
			blockTime: 6,
		}
		orch.l1Nets.Set(l1ID, l1Net)

		sysL1Net := shim.NewL1Network(shim.L1NetworkConfig{
			NetworkConfig: shim.NetworkConfig{
				CommonConfig: shim.CommonConfigFromSetup(setup),
				ChainConfig:  worldOutput.L1.Genesis.Config,
			},
			ID: l1ID,
		})
		setup.System.AddL1Network(sysL1Net)

		sysSuperchain := shim.NewSuperchain(shim.SuperchainConfig{
			CommonConfig: shim.CommonConfigFromSetup(setup),
			ID:           superchainID,
			Deployment: &SuperchainDeployment{
				protocolVersionsAddr: worldDeployment.Superchain.ProtocolVersions,
				superchainConfigAddr: worldDeployment.Superchain.SuperchainConfig,
			},
		})
		setup.System.AddSuperchain(sysSuperchain)

		depSetContents := make(map[eth.ChainID]*depset.StaticConfigDependency)
		for _, l2Out := range worldOutput.L2s {
			chainID := eth.ChainIDFromBig(l2Out.Genesis.Config.ChainID)
			index, err := chainID.ToUInt32()
			setup.Require.NoError(err)
			depSetContents[chainID] = &depset.StaticConfigDependency{
				ChainIndex:     supervisortypes.ChainIndex(index),
				ActivationTime: 0,
				HistoryMinTime: 0,
			}
		}
		staticDepSet, err := depset.NewStaticConfigDependencySet(depSetContents)
		setup.Require.NoError(err)

		sysCluster := shim.NewCluster(shim.ClusterConfig{
			CommonConfig:  shim.CommonConfigFromSetup(setup),
			ID:            clusterID,
			DependencySet: staticDepSet,
		})
		setup.System.AddCluster(sysCluster)

		for _, l2ID := range l2IDs {
			l2Out, ok := worldOutput.L2s[l2ID.ChainID.String()]
			setup.Require.True(ok, "L2 output must exist")
			l2Dep, ok := worldDeployment.L2s[l2ID.ChainID.String()]
			setup.Require.True(ok, "L2 deployment must exist")

			l2Net := &L2Network{
				genesis:   l2Out.Genesis,
				rollupCfg: l2Out.RollupCfg,
			}
			orch.l2Nets.Set(l2ID, l2Net)

			dep := &L2Deployment{
				systemConfigProxyAddr:   l2Dep.SystemConfigProxy,
				disputeGameFactoryProxy: l2Dep.DisputeGameFactoryProxy,
			}
			sysL2Net := shim.NewL2Network(shim.L2NetworkConfig{
				NetworkConfig: shim.NetworkConfig{
					CommonConfig: shim.CommonConfigFromSetup(setup),
					ChainConfig:  l2Out.Genesis.Config,
				},
				ID:           l2ID,
				RollupConfig: l2Out.RollupCfg,
				Deployment:   dep,
				Keys:         &keyring{keys: orch.keys, require: setup.Require},
				Superchain:   nil,
				L1:           sysL1Net,
				Cluster:      nil,
			})
			setup.System.AddL2Network(sysL2Net)
		}
	}
}
