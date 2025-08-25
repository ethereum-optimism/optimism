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
		WithL2ChallengerPostDeploy(orch, challengerID, l1ELID, l1CLID, supervisorID, clusterID, l2CLID, l2ELIDs)
	})
}

func WithSuperL2Challenger(challengerID stack.L2ChallengerID, l1ELID stack.L1ELNodeID, l1CLID stack.L1CLNodeID,
	supervisorID *stack.SupervisorID, clusterID *stack.ClusterID, l2ELIDs []stack.L2ELNodeID,
) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithL2ChallengerPostDeploy(orch, challengerID, l1ELID, l1CLID, supervisorID, clusterID, nil, l2ELIDs)
	})
}

func WithL2ChallengerPostDeploy(orch *Orchestrator, challengerID stack.L2ChallengerID, l1ELID stack.L1ELNodeID, l1CLID stack.L1CLNodeID,
	supervisorID *stack.SupervisorID, clusterID *stack.ClusterID, l2CLID *stack.L2CLNodeID, l2ELIDs []stack.L2ELNodeID,
) {
	ctx := orch.P().Ctx()
	ctx = stack.ContextWithID(ctx, challengerID)
	p := orch.P().WithCtx(ctx)

	require := p.Require()
	require.False(orch.challengers.Has(challengerID), "challenger must not already exist")

	challengerSecret, err := orch.keys.Secret(devkeys.ChallengerRole.Key(challengerID.ChainID().ToBig()))
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

	useSuperRoots := false
	if orch.wb.outInteropMigration != nil {
		disputeGameFactoryAddr = orch.wb.outInteropMigration.DisputeGameFactory
		require.NotEmpty(disputeGameFactoryAddr, "dispute game factory address is empty")
		useSuperRoots = true
	}
	for _, l2ELID := range l2ELIDs {
		chainID := l2ELID.ChainID()
		l2Net, ok := orch.l2Nets.Get(chainID)
		require.Truef(ok, "l2Net %s not found", chainID)
		factory := l2Net.deployment.DisputeGameFactoryProxyAddr()
		if disputeGameFactoryAddr == (common.Address{}) {
			disputeGameFactoryAddr = factory
			interopScheduled = l2Net.genesis.Config.InteropTime != nil
		} else if !useSuperRoots {
			require.Equal(l2Net.genesis.Config.InteropTime != nil, interopScheduled, "Cluster not consistently using interop")
		}

		l2Geneses = append(l2Geneses, l2Net.genesis)
		rollupCfgs = append(rollupCfgs, l2Net.rollupCfg)
		l2NetIDs = append(l2NetIDs, l2Net.id)
	}

	dir := p.TempDir()
	var cfg *config.Config
	// If interop is scheduled, or if we cannot do the pre-interop connection, then set up with supervisor
	if interopScheduled || l2CLID == nil || useSuperRoots {
		require.NotNil(supervisorID, "need supervisor to connect to in interop")
		require.NotNil(clusterID, "need cluster in interop")
		supervisorNode, ok := orch.supervisors.Get(*supervisorID)
		require.True(ok)
		l2ELRPCs := make([]string, len(l2ELIDs))
		for i, l2ELID := range l2ELIDs {
			l2EL, ok := orch.l2ELs.Get(l2ELID)
			require.True(ok)
			l2ELRPCs[i] = l2EL.UserRPC()
		}
		cluster, ok := orch.clusters.Get(*clusterID)
		require.True(ok)
		prestateVariant := shared.InteropVariant
		cfg, err = shared.NewInteropChallengerConfig(dir, l1EL.userRPC, l1CL.beaconHTTPAddr, supervisorNode.userRPC, l2ELRPCs,
			shared.WithFactoryAddress(disputeGameFactoryAddr),
			shared.WithPrivKey(challengerSecret),
			shared.WithDepset(cluster.DepSet()),
			shared.WithCannonConfig(rollupCfgs, l2Geneses, prestateVariant),
			shared.WithSuperCannonTraceType(),
			shared.WithSuperPermissionedTraceType(),
		)
		require.NoError(err, "Failed to create interop challenger config")
	} else {
		require.NotNil(l2CLID, "need L2 CL to connect to pre-interop")
		// In a post-interop infra setup, with unscheduled interop, we may see multiple EL nodes.
		var l2ELID stack.L2ELNodeID
		for _, id := range l2ELIDs {
			if id.ChainID() == l2CLID.ChainID() {
				l2ELID = id
				break
			}
		}
		require.NotZero(l2ELID, "need single L2 EL to connect to pre-interop")
		l2CL, ok := orch.l2CLs.Get(*l2CLID)
		require.True(ok)
		l2EL, ok := orch.l2ELs.Get(l2ELID)
		require.True(ok)
		prestateVariant := shared.MTCannonVariant
		cfg, err = shared.NewPreInteropChallengerConfig(dir, l1EL.userRPC, l1CL.beaconHTTPAddr, l2CL.UserRPC(), l2EL.UserRPC(),
			shared.WithFactoryAddress(disputeGameFactoryAddr),
			shared.WithPrivKey(challengerSecret),
			shared.WithCannonConfig(rollupCfgs, l2Geneses, prestateVariant),
			shared.WithCannonTraceType(),
			shared.WithPermissionedTraceType(),
			shared.WithFastGames(),
		)
		require.NoError(err, "Failed to create pre-interop challenger config")
	}

	svc, err := opchallenger.Main(ctx, logger, cfg, metrics.NoopMetrics)
	require.NoError(err)

	require.NoError(svc.Start(ctx))
	p.Cleanup(func() {
		ctx, cancel := context.WithCancel(ctx)
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
}
