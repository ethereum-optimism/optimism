package sysgo

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opchallenger "github.com/ethereum-optimism/optimism/op-challenger"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	shared "github.com/ethereum-optimism/optimism/op-devstack/shared/challenger"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

type L2Challenger struct {
	id       stack.L2ChallengerID
	service  cliapp.Lifecycle
	l2NetIDs []stack.L2NetworkID
}

func (p *L2Challenger) hydrate(system stack.ExtensibleSystem) {
	bFrontend := shim.NewL2Challenger(shim.L2ChallengerConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           p.id,
	})

	for _, netID := range p.l2NetIDs {
		l2Net := system.L2Network(netID)
		l2Net.(stack.ExtensibleL2Network).AddL2Challenger(bFrontend)
	}
}

func WithL2Challenger(challengerID stack.L2ChallengerID, l1ELID stack.L1ELNodeID, l1CLID stack.L1CLNodeID,
	supervisorID *stack.SupervisorID, clusterID *stack.ClusterID, l2CLID *stack.L2CLNodeID, l2ELIDs []stack.L2ELNodeID,
) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), challengerID))

		require := p.Require()
		require.False(orch.challengers.Has(challengerID), "challenger must not already exist")

		challengerSecret, err := orch.keys.Secret(devkeys.ChallengerRole.Key(l1ELID.ChainID().ToBig()))
		require.NoError(err)

		logger := p.Logger()
		logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok)
		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok)

		l2Geneses := make([]*core.Genesis, 0, len(l2ELIDs))
		rollupCfgs := make([]*rollup.Config, 0, len(l2ELIDs))
		l2NetIDs := make([]stack.L2NetworkID, 0, len(l2ELIDs))
		var disputeGameFactoryAddr common.Address
		var interopScheduled bool

		for _, l2ELID := range l2ELIDs {
			chainID := l2ELID.ChainID()
			l2Net, ok := orch.l2Nets.Get(chainID)
			require.Truef(ok, "l2Net %s not found", chainID)
			factory := l2Net.deployment.DisputeGameFactoryProxyAddr()
			if disputeGameFactoryAddr == (common.Address{}) {
				disputeGameFactoryAddr = factory
				interopScheduled = l2Net.genesis.Config.InteropTime != nil
			} else {
				require.Equal(l2Net.genesis.Config.InteropTime != nil, interopScheduled, "Cluster not consistently using interop")
			}

			l2Geneses = append(l2Geneses, l2Net.genesis)
			rollupCfgs = append(rollupCfgs, l2Net.rollupCfg)
			l2NetIDs = append(l2NetIDs, l2Net.id)
		}

		dir := p.TempDir()
		var cfg *config.Config
		if interopScheduled {
			require.NotNil(supervisorID, "need supervisor to connect to in interop")
			require.NotNil(clusterID, "need cluster in interop")
			supervisorNode, ok := orch.supervisors.Get(*supervisorID)
			require.True(ok)
			l2ELRPCs := make([]string, len(l2ELIDs))
			for i, l2ELID := range l2ELIDs {
				l2EL, ok := orch.l2ELs.Get(l2ELID)
				require.True(ok)
				l2ELRPCs[i] = l2EL.userRPC
			}
			cluster, ok := orch.clusters.Get(*clusterID)
			require.True(ok)
			prestateVariant := shared.InteropVariant
			cfg, err = shared.NewInteropChallengerConfig(dir, l1EL.userRPC, l1CL.beaconHTTPAddr, supervisorNode.userRPC, l2ELRPCs,
				shared.WithFactoryAddress(disputeGameFactoryAddr),
				shared.WithPrivKey(challengerSecret),
				shared.WithDepset(cluster.DepSet()),
				shared.WithSuperCannon(rollupCfgs, l2Geneses, prestateVariant),
				shared.WithSuperPermissioned(rollupCfgs, l2Geneses, prestateVariant),
			)
			require.NoError(err, "Failed to create interop challenger config")
		} else {
			require.NotNil(l2CLID, "need L2 CL to connect to pre-interop")
			require.Len(l2ELIDs, 1, "need single L2 EL to connect to pre-interop")
			l2CL, ok := orch.l2CLs.Get(*l2CLID)
			require.True(ok)
			l2EL, ok := orch.l2ELs.Get(l2ELIDs[0])
			require.True(ok)
			prestateVariant := shared.MTCannonVariant
			cfg, err = shared.NewPreInteropChallengerConfig(dir, l1EL.userRPC, l1CL.beaconHTTPAddr, l2CL.userRPC, l2EL.userRPC,
				shared.WithFactoryAddress(disputeGameFactoryAddr),
				shared.WithPrivKey(challengerSecret),
				shared.WithCannon(rollupCfgs, l2Geneses, prestateVariant),
				shared.WithPermissioned(rollupCfgs, l2Geneses, prestateVariant),
				shared.WithFastGames(),
			)
			require.NoError(err, "Failed to create pre-interop challenger config")
		}

		svc, err := opchallenger.Main(p.Ctx(), logger, cfg, metrics.NoopMetrics)
		require.NoError(err)

		require.NoError(svc.Start(p.Ctx()))
		p.Cleanup(func() {
			ctx, cancel := context.WithCancel(p.Ctx())
			cancel() // force-quit
			logger.Info("Closing challenger")
			_ = svc.Stop(ctx)
			logger.Info("Closed challenger")
		})

		c := &L2Challenger{
			id:       challengerID,
			service:  svc,
			l2NetIDs: l2NetIDs,
		}
		orch.challengers.Set(challengerID, c)
	})
}
