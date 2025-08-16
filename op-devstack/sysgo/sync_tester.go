package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-sync-tester/config"

	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"

	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type SyncTesterService struct {
	service *synctester.Service
}

func (n *SyncTesterService) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()

	for syncTesterID, chainID := range n.service.SyncTesters() {
		syncTesterRPC := n.service.SyncTesterEndpoint(chainID)
		rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), syncTesterRPC, client.WithLazyDial())
		require.NoError(err)
		system.T().Cleanup(rpcCl.Close)
		id := stack.NewSyncTesterID(syncTesterID.String(), chainID)
		front := shim.NewSyncTester(shim.SyncTesterConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			ID:           id,
			Client:       rpcCl,
		})
		net := system.Network(chainID).(stack.ExtensibleNetwork)
		net.AddSyncTester(front)
	}
}

func WithSyncTesters(l2ELs []stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		syncTesterID := stack.NewSyncTesterID("dev-sync-tester", l2ELs[0].ChainID())
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), syncTesterID))

		require := p.Require()

		require.Nil(orch.syncTester, "can only support a single sync-tester-service in sysgo")

		syncTesters := make(map[sttypes.SyncTesterID]*stconf.SyncTesterEntry)

		for _, elID := range l2ELs {
			id := sttypes.SyncTesterID(fmt.Sprintf("dev-sync-tester-%s", elID.ChainID()))
			require.NotContains(syncTesters, id, "one sync tester per chain only")

			el, ok := orch.l2ELs.Get(elID)
			require.True(ok, "need L2 EL for sync tester", elID)

			syncTesters[id] = &stconf.SyncTesterEntry{
				ELRPC: endpoint.MustRPC{Value: endpoint.URL(el.UserRPC())},
				// EngineRPC: endpoint.MustRPC{Value: endpoint.URL(el.authRPC)},
				// JwtPath:   el.jwtPath,
				ChainID: elID.ChainID(),
			}
		}

		cfg := &config.Config{
			RPC: oprpc.CLIConfig{},
			SyncTesters: &stconf.Config{
				SyncTesters: syncTesters,
			},
		}
		logger := p.Logger()
		srv, err := synctester.FromConfig(p.Ctx(), cfg, logger)
		require.NoError(err, "must setup sync tester service")
		require.NoError(srv.Start(p.Ctx()))
		p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			logger.Info("Closing sync tester")
			_ = srv.Stop(ctx)
			logger.Info("Closed sync tester")
		})
		orch.syncTester = &SyncTesterService{service: srv}
	})
}
