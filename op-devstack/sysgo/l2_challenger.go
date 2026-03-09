package sysgo

import (
	"context"
	"runtime"
	"time"

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

type l2ChallengerOpts struct {
	useCannonKonaConfig bool
}

type L2Challenger struct {
	id       stack.ComponentID
	service  cliapp.Lifecycle
	l2NetIDs []stack.ComponentID
	config   *config.Config
}

func (p *L2Challenger) hydrate(system stack.ExtensibleSystem) {
	bFrontend := shim.NewL2Challenger(shim.L2ChallengerConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           p.id,
		Config:       p.config,
	})

	for _, netID := range p.l2NetIDs {
		l2Net := system.L2Network(stack.ByID[stack.L2Network](netID))
		l2Net.(stack.ExtensibleL2Network).AddL2Challenger(bFrontend)
	}
}

func WithL2Challenger(challengerID stack.ComponentID, l1ELID stack.ComponentID, l1CLID stack.ComponentID,
	supervisorID *stack.ComponentID, clusterID *stack.ComponentID, l2CLID *stack.ComponentID, l2ELIDs []stack.ComponentID,
) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		WithL2ChallengerPostDeploy(orch, challengerID, l1ELID, l1CLID, supervisorID, clusterID, l2CLID, l2ELIDs, nil)
	})
}

func WithSuperL2Challenger(challengerID stack.ComponentID, l1ELID stack.ComponentID, l1CLID stack.ComponentID,
	supervisorID *stack.ComponentID, clusterID *stack.ComponentID, l2ELIDs []stack.ComponentID,
) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithL2ChallengerPostDeploy(orch, challengerID, l1ELID, l1CLID, supervisorID, clusterID, nil, l2ELIDs, nil)
	})
}

func WithSupernodeL2Challenger(challengerID stack.ComponentID, l1ELID stack.ComponentID, l1CLID stack.ComponentID,
	supernodeID *stack.SupernodeID, clusterID *stack.ComponentID, l2ELIDs []stack.ComponentID,
) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithL2ChallengerPostDeploy(orch, challengerID, l1ELID, l1CLID, nil, clusterID, nil, l2ELIDs, supernodeID)
	})
}

func WithL2ChallengerPostDeploy(orch *Orchestrator, challengerID stack.ComponentID, l1ELID stack.ComponentID, l1CLID stack.ComponentID,
	supervisorID *stack.ComponentID, clusterID *stack.ComponentID, l2CLID *stack.ComponentID, l2ELIDs []stack.ComponentID,
	supernodeID *stack.SupernodeID,
) {
	ctx := orch.P().Ctx()
	ctx = stack.ContextWithID(ctx, challengerID)
	p := orch.P().WithCtx(ctx)

	require := p.Require()
	challengerCID := challengerID
	require.False(orch.registry.Has(challengerCID), "challenger must not already exist")

	challengerSecret, err := orch.keys.Secret(devkeys.ChallengerRole.Key(challengerID.ChainID().ToBig()))
	require.NoError(err)

	logger := p.Logger()
	logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

	l1EL, ok := orch.GetL1EL(l1ELID)
	require.True(ok)
	l1CL, ok := orch.GetL1CL(l1CLID)
	require.True(ok)

	l2Geneses := make([]*core.Genesis, 0, len(l2ELIDs))
	rollupCfgs := make([]*rollup.Config, 0, len(l2ELIDs))
	l2NetIDs := make([]stack.ComponentID, 0, len(l2ELIDs))
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
		l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(chainID))
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

	l1Net, ok := orch.GetL1Network(stack.NewL1NetworkID(l1ELID.ChainID()))
	if !ok {
		require.Fail("l1 network not found")
	}
	l1Genesis := l1Net.genesis

	if orch.l2ChallengerOpts.useCannonKonaConfig {
		p.Log("Enabling cannon-kona, you may need to build kona-host and prestates with: cd kona && just")
	}

	dir := p.TempDir()
	var cfg *config.Config
	// If interop is scheduled, or if we cannot do the pre-interop connection, then set up with supervisor
	if interopScheduled || l2CLID == nil || useSuperRoots {
		require.NotNil(clusterID, "need cluster in interop")
		require.False(supervisorID != nil && supernodeID != nil, "cannot set both supervisorID and supernodeID")

		superRPC := ""
		useSuperNode := false
		switch {
		case supervisorID != nil:
			supervisorNode, ok := orch.GetSupervisor(*supervisorID)
			require.True(ok)
			superRPC = supervisorNode.UserRPC()
		case supernodeID != nil:
			supernode, ok := orch.supernodes.Get(*supernodeID)
			require.True(ok)
			superRPC = supernode.UserRPC()
			useSuperNode = true
		default:
			require.FailNow("need supervisor or supernode to connect to in interop/super-roots")
		}

		l2ELRPCs := make([]string, len(l2ELIDs))
		for i, l2ELID := range l2ELIDs {
			l2EL, ok := orch.GetL2EL(l2ELID)
			require.True(ok)
			l2ELRPCs[i] = l2EL.UserRPC()
		}
		cluster, ok := orch.GetCluster(*clusterID)
		require.True(ok)
		prestateVariant := shared.InteropVariant
		options := []shared.Option{
			shared.WithFactoryAddress(disputeGameFactoryAddr),
			shared.WithPrivKey(challengerSecret),
			shared.WithDepset(cluster.DepSet()),
			shared.WithCannonConfig(rollupCfgs, l1Genesis, l2Geneses, prestateVariant),
			shared.WithSuperCannonGameType(),
			shared.WithSuperPermissionedGameType(),
		}
		if orch.l2ChallengerOpts.useCannonKonaConfig {
			options = append(options,
				shared.WithCannonKonaInteropConfig(rollupCfgs, l1Genesis, l2Geneses),
				shared.WithSuperCannonKonaGameType(),
			)
		}
		cfg, err = shared.NewInteropChallengerConfig(dir, l1EL.UserRPC(), l1CL.beaconHTTPAddr, superRPC, l2ELRPCs, options...)
		require.NoError(err, "Failed to create interop challenger config")
		cfg.UseSuperNode = useSuperNode
	} else {
		require.NotNil(l2CLID, "need L2 CL to connect to pre-interop")
		// In a post-interop infra setup, with unscheduled interop, we may see multiple EL nodes.
		var l2ELID stack.ComponentID
		for _, id := range l2ELIDs {
			if id.ChainID() == l2CLID.ChainID() {
				l2ELID = id
				break
			}
		}
		require.NotZero(l2ELID, "need single L2 EL to connect to pre-interop")
		l2CL, ok := orch.GetL2CL(*l2CLID)
		require.True(ok)
		l2EL, ok := orch.GetL2EL(l2ELID)
		require.True(ok)
		prestateVariant := shared.MTCannonVariant
		options := []shared.Option{
			shared.WithFactoryAddress(disputeGameFactoryAddr),
			shared.WithPrivKey(challengerSecret),
			shared.WithCannonConfig(rollupCfgs, l1Genesis, l2Geneses, prestateVariant),
			shared.WithCannonGameType(),
			shared.WithPermissionedGameType(),
			shared.WithFastGames(),
		}
		if orch.l2ChallengerOpts.useCannonKonaConfig {
			options = append(options,
				shared.WithCannonKonaConfig(rollupCfgs, l1Genesis, l2Geneses),
				shared.WithCannonKonaGameType(),
			)
		}
		cfg, err = shared.NewPreInteropChallengerConfig(dir, l1EL.UserRPC(), l1CL.beaconHTTPAddr, l2CL.UserRPC(), l2EL.UserRPC(), options...)
		require.NoError(err, "Failed to create pre-interop challenger config")
	}

	svc, err := opchallenger.Main(ctx, logger, cfg, metrics.NoopMetrics)
	require.NoError(err)

	require.NoError(svc.Start(ctx))
	p.Cleanup(func() {
		ctx, cancel := context.WithCancel(ctx)
		cancel() // force-quit
		logger.Info("Closing challenger")
		// Start a separate goroutine to print a stack trace if the challenger fails to stop in a timely manner.
		timer := time.AfterFunc(1*time.Minute, func() {
			if svc.Stopped() {
				return
			}
			// Print stack trace of all goroutines
			buf := make([]byte, 1<<20) // 1MB buffer
			stacklen := runtime.Stack(buf, true)
			logger.Error("Challenger failed to stop; printing all goroutine stacks:\n%v", string(buf[:stacklen]))
		})
		_ = svc.Stop(ctx)
		timer.Stop()
		logger.Info("Closed challenger")
	})

	c := &L2Challenger{
		id:       challengerID,
		service:  svc,
		l2NetIDs: l2NetIDs,
		config:   cfg,
	}
	orch.registry.Register(challengerID, c)
}
