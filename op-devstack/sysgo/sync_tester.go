package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-sync-tester/config"

	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"

	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type SyncTester struct {
	id      stack.SyncTesterID
	service *synctester.Service
}

func WithSyncTester(l2ELs []stack.L2ELNodeID, fcus sttypes.FCUState) stack.Option[*Orchestrator] {
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
				Cfg: stconf.EntryCfg{
					ChainID: elID.ChainID(),
					Target:  fcus,
				},
			}
		}

		cfg := &config.Config{
			RPC: oprpc.CLIConfig{
				ListenAddr: "127.0.0.1",
			},
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
		orch.syncTester = &SyncTester{id: syncTesterID, service: srv}
	})
}
