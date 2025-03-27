package systemgo

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	supervisorConfig "github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
)

type Supervisor struct {
	userRPC string
}

func WithSupervisor(supervisorID system2.SupervisorID, clusterID system2.ClusterID, l1ELID system2.L1ELNodeID) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		setup.Require.True(ok, "need L1 EL node to connect supervisor to")

		cluster := setup.System.Cluster(clusterID)

		cfg := &supervisorConfig.Config{
			MetricsConfig: metrics.CLIConfig{
				Enabled: false,
			},
			PprofConfig: oppprof.CLIConfig{
				ListenEnabled: false,
			},
			LogConfig: oplog.CLIConfig{ // ignored, logger overrides this
				Level:  log.LevelDebug,
				Format: oplog.FormatText,
			},
			RPC: oprpc.CLIConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0,
				EnableAdmin: true,
			},
			SyncSources:           &syncnode.CLISyncNodes{}, // no sync-sources
			L1RPC:                 l1EL.userRPC,
			Datadir:               orch.t.TempDir(),
			Version:               "dev",
			DependencySetSource:   cluster.DependencySet().(*depset.StaticConfigDependencySet),
			MockRun:               false,
			SynchronousProcessors: false,
			DatadirSyncEndpoint:   "",
		}

		logger := setup.Log.New("id", supervisorID)

		super, err := supervisor.SupervisorFromConfig(context.Background(), cfg, logger)
		setup.Require.NoError(err)

		err = super.Start(context.Background())
		setup.Require.NoError(err)

		orch.t.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			logger.Info("Closing supervisor")
			closeErr := super.Stop(ctx)
			logger.Info("Closed supervisor", "err", closeErr)
		})

		supervisorNode := &Supervisor{
			userRPC: super.RPC(),
		}
		orch.supervisors.Set(supervisorID, supervisorNode)

		supClient, err := client.NewRPC(setup.Ctx, logger, supervisorNode.userRPC, client.WithLazyDial())
		setup.Require.NoError(err)

		setup.System.AddSupervisor(system2.NewSupervisor(system2.SupervisorConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           supervisorID,
			Client:       supClient,
		}))
	}
}

func WithManagedBySupervisor(l2CLID system2.L2CLNodeID, supervisorID system2.SupervisorID) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		setup.Require.True(ok, "looking for L2 CL node to connect to supervisor")
		interopEndpoint, secret := l2CL.opNode.InteropRPC()

		super := setup.System.Supervisor(supervisorID)
		err := retry.Do0(setup.Ctx, 10, retry.Exponential(), func() error {
			return super.AdminAPI().AddL2RPC(setup.Ctx, interopEndpoint, secret)
		})
		setup.Require.NoError(err, "must connect CL node %s to supervisor %s", l2CLID, supervisorID)
	}
}
