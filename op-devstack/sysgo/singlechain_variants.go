package sysgo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	opconductor "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	synctesterconfig "github.com/ethereum-optimism/optimism/op-sync-tester/config"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum/go-ethereum/log"
)

func NewSingleChainMultiNodeRuntime(t devtest.T, withP2P bool) *SingleChainRuntime {
	return NewSingleChainMultiNodeRuntimeWithConfig(t, withP2P, PresetConfig{})
}

func NewSingleChainMultiNodeRuntimeWithConfig(t devtest.T, withP2P bool, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalRuntimeWithConfig(t, cfg)
	nodeB := addSingleChainOpNode(t, runtime, "b", false, "", cfg.GlobalL2CLOptions...)
	if withP2P {
		connectSingleChainNodes(t, runtime.L2EL, runtime.L2CL, nodeB)
	}
	runtime.P2PEnabled = withP2P
	return runtime
}

func NewSingleChainTwoVerifiersRuntime(t devtest.T) *SingleChainRuntime {
	return NewSingleChainTwoVerifiersRuntimeWithConfig(t, PresetConfig{})
}

func NewSingleChainTwoVerifiersRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewSingleChainMultiNodeRuntimeWithConfig(t, true, cfg)
	nodeB := runtime.Nodes["b"]
	t.Require().NotNil(nodeB, "missing single-chain node b")
	nodeC := addSingleChainOpNode(t, runtime, "c", false, nodeB.CL.UserRPC(), cfg.GlobalL2CLOptions...)

	connectSingleChainNodes(t, runtime.L2EL, runtime.L2CL, nodeC)
	connectSingleChainNodes(t, nodeB.EL, nodeB.CL, nodeC)

	// Follow legacy behavior: test-sequencer is wired against node "b".
	replaceSingleChainTestSequencer(t, runtime, "dev", nodeB)
	return runtime
}

func NewSimpleWithSyncTesterRuntime(t devtest.T) *SingleChainRuntime {
	return NewSimpleWithSyncTesterRuntimeWithConfig(t, PresetConfig{})
}

func NewSimpleWithSyncTesterRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalRuntimeWithConfig(t, cfg)
	syncTester := startSyncTesterService(t, map[eth.ChainID]string{
		runtime.L2Network.ChainID(): runtime.L2EL.UserRPC(),
	})
	syncTesterELCfg := DefaultSyncTesterELConfig()
	if len(cfg.GlobalSyncTesterELOptions) > 0 {
		syncTesterELTarget := NewComponentTarget("sync-tester-el", runtime.L2Network.ChainID())
		for _, opt := range cfg.GlobalSyncTesterELOptions {
			if opt == nil {
				continue
			}
			opt.Apply(t, syncTesterELTarget, syncTesterELCfg)
		}
	}
	syncTesterEL := startSyncTesterELNode(
		t,
		runtime.L2EL.JWTPath(),
		syncTester,
		NewComponentTarget("sync-tester-el", runtime.L2Network.ChainID()),
		syncTesterELCfg,
	)
	jwtSecret := readJWTSecretFromPath(t, runtime.L2EL.JWTPath())
	l2CL2 := startL2CLNode(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, syncTesterEL, jwtSecret, l2CLNodeStartConfig{
		Key:           "verifier",
		IsSequencer:   false,
		NoDiscovery:   true,
		EnableReqResp: true,
		UseReqResp:    true,
		L2CLOptions:   cfg.GlobalL2CLOptions,
	})
	node := newSingleChainNodeRuntime("verifier", false, syncTesterEL, l2CL2)
	runtime.Nodes[node.Name] = node
	connectSingleChainCLPeer(t, runtime.L2CL, node.CL)
	runtime.SyncTester = &SyncTesterRuntime{
		Service: syncTester,
		Node:    node,
	}
	return runtime
}

func NewMinimalWithConductorsRuntime(t devtest.T) *SingleChainRuntime {
	return NewMinimalWithConductorsRuntimeWithConfig(t, PresetConfig{})
}

func NewMinimalWithConductorsRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	// Conductor tests only exercise sequencing leadership. They do not need a
	// challenger, and rust e2e jobs do not build cannon artifacts.
	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
	nodeB := addSingleChainOpNode(t, runtime, "b", true, "", cfg.GlobalL2CLOptions...)
	nodeC := addSingleChainOpNode(t, runtime, "c", true, "", cfg.GlobalL2CLOptions...)

	conductorA := startConductorNode(t, "sequencer", runtime.L2Network, runtime.L2CL.(*OpNode), runtime.L2EL, true, false)
	conductorB := startConductorNode(t, "b", runtime.L2Network, nodeB.CL.(*OpNode), nodeB.EL, false, true)
	conductorC := startConductorNode(t, "c", runtime.L2Network, nodeC.CL.(*OpNode), nodeC.EL, false, true)
	startConductorCluster(t, conductorA, []*Conductor{conductorB, conductorC})

	runtime.Conductors = map[string]*Conductor{
		"sequencer": conductorA,
		"b":         conductorB,
		"c":         conductorC,
	}
	return runtime
}

func connectSingleChainNodes(t devtest.T, sourceEL L2ELNode, sourceCL L2CLNode, target *SingleChainNodeRuntime) {
	connectL2ELPeers(t, t.Logger(), sourceEL.UserRPC(), target.EL.UserRPC(), false)
	connectSingleChainCLPeer(t, sourceCL, target.CL)
}

func connectSingleChainCLPeer(t devtest.T, sourceCL, targetCL L2CLNode) {
	connectL2CLPeers(t, t.Logger(), sourceCL, targetCL)
}

func replaceSingleChainTestSequencer(t devtest.T, runtime *SingleChainRuntime, name string, node *SingleChainNodeRuntime) {
	l2EL, ok := node.EL.(*OpGeth)
	t.Require().True(ok, "single-chain test sequencer requires an op-geth EL node")
	l2CL, ok := node.CL.(*OpNode)
	t.Require().True(ok, "single-chain test sequencer requires an op-node CL node")
	testSequencer := startTestSequencer(
		t,
		runtime.Keys,
		runtime.L2EL.JWTPath(),
		readJWTSecretFromPath(t, runtime.L2EL.JWTPath()),
		runtime.L1Network,
		runtime.L1EL,
		runtime.L1CL,
		l2EL,
		l2CL,
	)
	runtime.TestSequencer = newTestSequencerRuntime(testSequencer, name)
}

func addSingleChainOpNode(
	t devtest.T,
	runtime *SingleChainRuntime,
	name string,
	isSequencer bool,
	followSource string,
	l2Opts ...L2CLOption,
) *SingleChainNodeRuntime {
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	identity := NewELNodeIdentity(0)
	l2EL := startL2ELNode(t, runtime.L2Network, jwtPath, jwtSecret, name, identity)
	l2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            name,
		IsSequencer:    isSequencer,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		L2FollowSource: followSource,
		L2CLOptions:    l2Opts,
	})
	node := newSingleChainNodeRuntime(name, isSequencer, l2EL, l2CL)
	runtime.Nodes[name] = node
	return node
}

func startSyncTesterService(t devtest.T, chainRPCs map[eth.ChainID]string) *SyncTesterService {
	require := t.Require()
	syncTesters := make(map[sttypes.SyncTesterID]*stconf.SyncTesterEntry)
	for chainID, elRPC := range chainRPCs {
		id := sttypes.SyncTesterID(fmt.Sprintf("dev-sync-tester-%s", chainID))
		syncTesters[id] = &stconf.SyncTesterEntry{
			ELRPC:   endpoint.MustRPC{Value: endpoint.URL(elRPC)},
			ChainID: chainID,
		}
	}
	cfg := &synctesterconfig.Config{
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		SyncTesters: &stconf.Config{
			SyncTesters: syncTesters,
		},
	}
	logger := t.Logger().New("component", "sync-tester")
	srv, err := synctester.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "must setup sync tester service")
	require.NoError(srv.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		logger.Info("Closing sync tester")
		_ = srv.Stop(ctx)
		logger.Info("Closed sync tester")
	})
	return &SyncTesterService{
		service: srv,
	}
}

func startSyncTesterELNode(
	t devtest.T,
	jwtPath string,
	syncTester *SyncTesterService,
	target ComponentTarget,
	cfg *SyncTesterELConfig,
) *SyncTesterEL {
	node := &SyncTesterEL{
		target:     target,
		jwtPath:    jwtPath,
		config:     cfg,
		p:          t,
		syncTester: syncTester,
	}
	node.Start()
	t.Cleanup(node.Stop)
	return node
}

func startConductorNode(
	t devtest.T,
	conductorName string,
	l2Net *L2Network,
	opNode *OpNode,
	l2EL L2ELNode,
	bootstrap bool,
	paused bool,
) *Conductor {
	require := t.Require()
	serverID := conductorName
	require.NotEmpty(serverID, "conductor ID key cannot be empty")

	var conductorRPCEndpoint atomic.Value
	conductorRPCEndpoint.Store("")
	opNode.cfg.ConductorEnabled = true
	opNode.cfg.ConductorRpcTimeout = 5 * time.Second
	opNode.cfg.ConductorRpc = func(ctx context.Context) (string, error) {
		for {
			if endpoint, _ := conductorRPCEndpoint.Load().(string); endpoint != "" {
				return endpoint, nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	opNode.cfg.Driver.SequencerStopped = true
	opNode.Stop()
	opNode.Start()

	cfg := opconductor.Config{
		ConsensusAddr:           "127.0.0.1",
		ConsensusPort:           0,
		ConsensusAdvertisedAddr: "",
		RaftServerID:            serverID,
		RaftStorageDir:          filepath.Join(t.TempDir(), "raft"),
		RaftBootstrap:           bootstrap,
		RaftSnapshotInterval:    120 * time.Second,
		RaftSnapshotThreshold:   8192,
		RaftTrailingLogs:        10240,
		RaftHeartbeatTimeout:    1000 * time.Millisecond,
		RaftLeaderLeaseTimeout:  500 * time.Millisecond,
		NodeRPC:                 opNode.UserRPC(),
		ExecutionRPC:            l2EL.UserRPC(),
		Paused:                  paused,
		HealthCheck: opconductor.HealthCheckConfig{
			Interval:       3600,
			UnsafeInterval: 3600,
			SafeInterval:   3600,
			MinPeerCount:   1,
		},
		RollupCfg:      *l2Net.rollupCfg,
		RPCEnableProxy: false,
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelInfo,
			Format: oplog.FormatText,
			Color:  false,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
	}

	logger := t.Logger().New("component", "conductor", "name", conductorName, "chain", l2Net.ChainID())
	svc, err := opconductor.New(t.Ctx(), &cfg, logger, "0.0.1")
	require.NoError(err)
	require.NoError(svc.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logger.Info("Closing conductor")
		if err := svc.Stop(ctx); err != nil {
			logger.Warn("Failed to close conductor cleanly", "err", err)
		}
	})

	out := &Conductor{
		name:              conductorName,
		chainID:           l2Net.ChainID(),
		serverID:          serverID,
		consensusEndpoint: svc.ConsensusEndpoint(),
		rpcEndpoint:       svc.HTTPEndpoint(),
		service:           svc,
	}
	conductorRPCEndpoint.Store(svc.HTTPEndpoint())
	return out
}

func startConductorCluster(t devtest.T, bootstrap *Conductor, members []*Conductor) {
	require := t.Require()
	ctx, cancel := context.WithTimeout(t.Ctx(), 90*time.Second)
	defer cancel()

	err := retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		if !bootstrap.service.Leader(ctx) {
			return errors.New("bootstrap conductor is not leader yet")
		}
		return nil
	})
	require.NoError(err, "bootstrap conductor never became leader")

	for _, member := range members {
		err := retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return bootstrap.service.AddServerAsNonvoter(ctx, member.ServerID(), member.ConsensusEndpoint(), 0)
		})
		require.NoErrorf(err, "failed to add conductor %s as non-voter", member.ServerID())

		err = retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return bootstrap.service.AddServerAsVoter(ctx, member.ServerID(), member.ConsensusEndpoint(), 0)
		})
		require.NoErrorf(err, "failed to add conductor %s as voter", member.ServerID())
	}

	expectedServers := 1 + len(members)
	err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		membership, err := bootstrap.service.ClusterMembership(ctx)
		if err != nil {
			return err
		}
		if len(membership.Servers) != expectedServers {
			return fmt.Errorf("expected %d conductors in cluster membership, got %d", expectedServers, len(membership.Servers))
		}
		return nil
	})
	require.NoError(err, "conductor cluster did not converge to expected membership")

	cluster := append([]*Conductor{bootstrap}, members...)
	for _, conductor := range cluster {
		err := retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return conductor.service.Resume(ctx)
		})
		require.NoErrorf(err, "failed to resume conductor %s", conductor.ServerID())
	}

	for _, conductor := range cluster {
		err := retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
			if !conductor.service.SequencerHealthy(ctx) {
				return fmt.Errorf("conductor %s sequencer is not healthy yet", conductor.ServerID())
			}
			return nil
		})
		require.NoErrorf(err, "conductor %s never became healthy", conductor.ServerID())
	}
}
