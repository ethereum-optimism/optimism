package sysgo

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supervisorConfig "github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
)

type Supervisor struct {
	id      stack.SupervisorID
	userRPC string
}

func (s *Supervisor) hydrate(sys stack.ExtensibleSystem) {
	tlog := sys.Logger().New("id", s.id)
	supClient, err := client.NewRPC(sys.T().Ctx(), tlog, s.userRPC, client.WithLazyDial())
	sys.T().Require().NoError(err)
	sys.T().Cleanup(supClient.Close)

	sys.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           s.id,
		Client:       supClient,
	}))
}

func WithSupervisor(supervisorID stack.SupervisorID, clusterID stack.ClusterID, l1ELID stack.L1ELNodeID) stack.Option {
	return func(o stack.Orchestrator) {
		orch := o.(*Orchestrator)
		require := orch.P().Require()

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "need L1 EL node to connect supervisor to")

		cluster, ok := orch.clusters.Get(clusterID)
		require.True(ok, "need cluster to determine dependency set")

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
			Datadir:               orch.p.TempDir(),
			Version:               "dev",
			DependencySetSource:   cluster.depset,
			MockRun:               false,
			SynchronousProcessors: false,
			DatadirSyncEndpoint:   "",
		}

		plog := orch.P().Logger().New("id", supervisorID)

		super, err := supervisor.SupervisorFromConfig(context.Background(), cfg, plog)
		require.NoError(err)

		err = super.Start(context.Background())
		require.NoError(err)

		orch.p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			plog.Info("Closing supervisor")
			closeErr := super.Stop(ctx)
			plog.Info("Closed supervisor", "err", closeErr)
		})

		supervisorNode := &Supervisor{
			id:      supervisorID,
			userRPC: super.RPC(),
		}
		orch.supervisors.Set(supervisorID, supervisorNode)
	}
}

func WithManagedBySupervisor(l2CLID stack.L2CLNodeID, supervisorID stack.SupervisorID) stack.Option {
	return func(o stack.Orchestrator) {
		orch := o.(*Orchestrator)
		require := orch.P().Require()

		l2CL, ok := orch.l2CLs.Get(l2CLID)
		require.True(ok, "looking for L2 CL node to connect to supervisor")
		interopEndpoint, secret := l2CL.opNode.InteropRPC()

		s, ok := orch.supervisors.Get(supervisorID)
		require.True(ok, "looking for supervisor")

		ctx := o.P().Ctx()
		rpcClient, err := client.NewRPC(ctx, o.P().Logger(), s.userRPC, client.WithLazyDial())
		o.P().Require().NoError(err)
		supClient := sources.NewSupervisorClient(rpcClient)

		err = retry.Do0(ctx, 10, retry.Exponential(), func() error {
			return supClient.AddL2RPC(ctx, interopEndpoint, secret)
		})
		require.NoError(err, "must connect CL node %s to supervisor %s", l2CLID, supervisorID)
	}
}
