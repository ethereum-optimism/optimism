package sysgo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	opconductor "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
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
	return addSingleChainVerifier(t, NewMinimalRuntimeWithConfig(t, cfg), withP2P, cfg)
}

func NewSingleChainWithIsolatedVerifierRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	return addSingleChainVerifier(t, NewMinimalNoFaultProofsRuntimeWithConfig(t, cfg), false, cfg)
}

func addSingleChainVerifier(t devtest.T, runtime *SingleChainRuntime, withP2P bool, cfg PresetConfig) *SingleChainRuntime {
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
		L2CLOptions:   cfg.GlobalL2CLOptions,
	})
	node := newSingleChainNodeRuntime("verifier", false, syncTesterEL, l2CL2)
	runtime.Nodes[node.Name] = node
	connectSingleChainCLPeer(t, runtime.L2CL, node.CL)
	runtime.SyncTester = &SyncTesterRuntime{
		Service: syncTester,
		Node:    node,
		EL:      node.EL,
		CL:      node.CL,
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
		StartPrimary:    startStoppedSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
	// Every conductor member sequences once leadership reaches it, so a binary override has to
	// apply to all of them or it would silently stop taking effect on transfer. The primary already
	// received these options from startStoppedSingleChainPrimary.
	candidateELOpts := append(append([]OpRethOption{}, ResolveMixedL2ELOpts(t)...), cfg.OpRethOptions...)
	nodeB := addStoppedSingleChainSequencerNode(t, runtime, "b", candidateELOpts, cfg.GlobalL2CLOptions...)
	nodeC := addStoppedSingleChainSequencerNode(t, runtime, "c", candidateELOpts, cfg.GlobalL2CLOptions...)

	// A non-candidate node, always stock: it never sequences, so it independently validates every
	// block the candidates build. Without it an overridden binary on all three members could accept
	// its own invalid blocks and the run would still pass.
	nodeVerifier := addSingleChainOpNode(t, runtime, "verifier", false, "", cfg.GlobalL2CLOptions...)

	healthCheck := conductorHealthCheckConfig(cfg.ConductorFastHealthChecks)
	primaryCL := runtime.L2CL.(*OpNode)
	conductorA := startConductorNode(t, "sequencer", runtime.L2Network, primaryCL, runtime.L2EL, true, false, healthCheck)
	conductorB := startConductorNode(t, "b", runtime.L2Network, nodeB.CL.(*OpNode), nodeB.EL, false, true, healthCheck)
	conductorC := startConductorNode(t, "c", runtime.L2Network, nodeC.CL.(*OpNode), nodeC.EL, false, true, healthCheck)

	// Mesh the sequencer CL nodes over p2p, as in a production HA deployment:
	// the active sequencer gossips unsafe blocks to the followers, which keeps
	// them close enough to the raft-committed head for the conductor to start
	// them on leadership changes (op-conductor only backfills a single missing
	// block). EL p2p is not needed — unsafe blocks reach the follower ELs via
	// CL gossip and the engine API. Peering must happen after
	// startConductorNode, which restarts each op-node and would drop earlier
	// connections.
	connectSingleChainCLPeer(t, runtime.L2CL, nodeB.CL)
	connectSingleChainCLPeer(t, runtime.L2CL, nodeC.CL)
	connectSingleChainCLPeer(t, nodeB.CL, nodeC.CL)
	connectSingleChainCLPeer(t, runtime.L2CL, nodeVerifier.CL)
	runtime.P2PEnabled = true

	startConductorCluster(
		t,
		conductorA,
		primaryCL,
		[]*Conductor{conductorB, conductorC},
		[]*OpNode{nodeB.CL.(*OpNode), nodeC.CL.(*OpNode)},
	)

	// Match production HA deployments: downstream services use every
	// conductor endpoint and let the proxy's leader gate select the active
	// sequencer. The batcher was started before the conductors existed, so
	// restart it with the conductor endpoints now that the cluster is ready.
	if runtime.L2Batcher != nil {
		t.Require().NoError(runtime.L2Batcher.service.Stop(t.Ctx()), "failed to stop direct-endpoint batcher")
	}
	conductorEndpoints := []string{conductorA.HTTPEndpoint(), conductorB.HTTPEndpoint(), conductorC.HTTPEndpoint()}
	runtime.L2Batcher = startMinimalBatcher(t, runtime.Keys, runtime.L2Network, runtime.L1EL, primaryCL, runtime.L2EL, append(cfg.BatcherOptions,
		func(_ ComponentTarget, batcherCfg *bss.CLIConfig) {
			batcherCfg.L2EthRpc = conductorEndpoints
			batcherCfg.RollupRpc = conductorEndpoints
			batcherCfg.ActiveSequencerCheckDuration = 0
		},
	)...)

	runtime.Conductors = map[string]*Conductor{
		"sequencer": conductorA,
		"b":         conductorB,
		"c":         conductorC,
	}
	return runtime
}

func connectSingleChainNodes(t devtest.T, sourceEL L2ELNode, sourceCL L2CLNode, target *SingleChainNodeRuntime) {
	connectL2ELPeers(t, t.Logger(), sourceEL.UserRPC(), target.EL.UserRPC())
	connectSingleChainCLPeer(t, sourceCL, target.CL)
}

func connectSingleChainCLPeer(t devtest.T, sourceCL, targetCL L2CLNode) {
	connectL2CLPeers(t, t.Logger(), sourceCL, targetCL)
}

func replaceSingleChainTestSequencer(t devtest.T, runtime *SingleChainRuntime, name string, node *SingleChainNodeRuntime) {
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
		node.EL,
		runtime.L2Network,
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
	return addSingleChainOpNodeWithELOpts(t, runtime, name, isSequencer, followSource, nil, l2Opts...)
}

// addSingleChainOpNodeWithELOpts adds a node whose EL takes op-reth options. Passing none yields a
// stock op-reth, which is what lets a preset keep an independent check on blocks built by an
// overridden binary.
func addSingleChainOpNodeWithELOpts(
	t devtest.T,
	runtime *SingleChainRuntime,
	name string,
	isSequencer bool,
	followSource string,
	elOpts []OpRethOption,
	l2Opts ...L2CLOption,
) *SingleChainNodeRuntime {
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELForKey(t, runtime.L2Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0), elOpts...)
	l2CL := startL2CLForKey(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, name, name, isSequencer, followSource, l2Opts)
	node := newSingleChainNodeRuntime(name, isSequencer, l2EL, l2CL)
	runtime.Nodes[name] = node
	return node
}

// addStoppedSingleChainSequencerNode builds an op-node sequencer that cannot
// create an unsafe fork before its conductor cluster is ready.
func addStoppedSingleChainSequencerNode(
	t devtest.T,
	runtime *SingleChainRuntime,
	name string,
	elOpts []OpRethOption,
	l2Opts ...L2CLOption,
) *SingleChainNodeRuntime {
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELForKey(t, runtime.L2Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0), elOpts...)
	l2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:              name,
		IsSequencer:      true,
		NoDiscovery:      true,
		EnableReqResp:    true,
		L2CLOptions:      l2Opts,
		SequencerStopped: true,
	})
	node := newSingleChainNodeRuntime(name, true, l2EL, l2CL)
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

// conductorHealthCheckConfig returns the conductor health-monitor settings.
// By default checks effectively never run (hourly), keeping manual leadership
// operations deterministic. With fast=true, checks run every second so that a
// dead sequencer node flips its conductor to unhealthy and triggers failover;
// the staleness windows stay at an hour so health remains purely
// liveness-based, and MinPeerCount is 1 (op-conductor rejects 0) — each node
// keeps at least one peer of the three-node mesh even after one node dies.
func conductorHealthCheckConfig(fast bool) opconductor.HealthCheckConfig {
	if fast {
		return opconductor.HealthCheckConfig{
			Interval:       1,
			UnsafeInterval: 3600,
			SafeInterval:   3600,
			MinPeerCount:   1,
		}
	}
	return opconductor.HealthCheckConfig{
		Interval:       3600,
		UnsafeInterval: 3600,
		SafeInterval:   3600,
		MinPeerCount:   1,
	}
}

func startConductorNode(
	t devtest.T,
	conductorName string,
	l2Net *L2Network,
	opNode *OpNode,
	l2EL L2ELNode,
	bootstrap bool,
	paused bool,
	healthCheck opconductor.HealthCheckConfig,
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
		RaftStorageDir:          filepath.Join(t.TempDirWithPrefix("op-conductor-"+NewComponentTarget(conductorName, l2Net.ChainID()).String()), "raft"),
		RaftBootstrap:           bootstrap,
		RaftSnapshotInterval:    120 * time.Second,
		RaftSnapshotThreshold:   8192,
		RaftTrailingLogs:        10240,
		RaftHeartbeatTimeout:    1000 * time.Millisecond,
		RaftLeaderLeaseTimeout:  500 * time.Millisecond,
		NodeRPC:                 opNode.UserRPC(),
		ExecutionRPC:            l2EL.UserRPC(),
		Paused:                  paused,
		HealthCheck:             healthCheck,
		RollupCfg:               *l2Net.rollupCfg,
		RPCEnableProxy:          true,
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

func startConductorCluster(
	t devtest.T,
	bootstrap *Conductor,
	bootstrapNode *OpNode,
	members []*Conductor,
	memberNodes []*OpNode,
) {
	require := t.Require()
	require.Len(memberNodes, len(members), "each conductor member must have a sequencer node")
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

	// A fresh Raft FSM holds no unsafe payload, so a conductor refuses to
	// start a sequencer on its own (ErrNoUnsafeHead). Seed sequencing manually
	// on the bootstrap node — exactly how an operator bootstraps a production
	// HA cluster. All sequencers started stopped, so they share the same initial
	// head and cannot create competing forks before this point.
	rollupClient, err := dial.DialRollupClientWithTimeout(ctx, t.Logger(), bootstrapNode.UserRPC())
	require.NoError(err, "failed to dial bootstrap sequencer node")
	initialStatus, err := rollupClient.SyncStatus(ctx)
	require.NoError(err, "failed to fetch bootstrap sequencer sync status")
	require.NoError(rollupClient.StartSequencer(ctx, initialStatus.UnsafeL2.Hash), "failed to start sequencing on bootstrap node")
	err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		status, err := rollupClient.SyncStatus(ctx)
		if err != nil {
			return err
		}
		if status.UnsafeL2.Number <= initialStatus.UnsafeL2.Number {
			return fmt.Errorf("bootstrap sequencer has not sealed a block yet, unsafe head at %d", status.UnsafeL2.Number)
		}
		return nil
	})
	require.NoError(err, "bootstrap sequencer never sealed its first block")

	// Freeze the seed head and require every follower to have processed that
	// exact canonical block before conductor control loops can start. Without
	// this barrier, a failover can elect a follower that is unable to start on
	// the Raft-committed head.
	//
	// This is a work-around for the following issue to prevent flakes:
	// https://github.com/ethereum-optimism/core-team/issues/2938.
	seedHash, err := rollupClient.StopSequencer(ctx)
	require.NoError(err, "failed to stop bootstrap sequencer after seeding")
	var seedStatus *eth.SyncStatus
	err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		status, err := rollupClient.SyncStatus(ctx)
		if err != nil {
			return err
		}
		if status.UnsafeL2.Hash != seedHash {
			return fmt.Errorf("bootstrap sync status unsafe head is %s, want stopped seed head %s", status.UnsafeL2.Hash, seedHash)
		}
		seedStatus = status
		return nil
	})
	require.NoError(err, "bootstrap sync status never reflected the stopped seed head")
	for i, node := range memberNodes {
		memberClient, err := dial.DialRollupClientWithTimeout(ctx, t.Logger(), node.UserRPC())
		require.NoErrorf(err, "failed to dial sequencer node %s", members[i].ServerID())
		err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
			status, err := memberClient.SyncStatus(ctx)
			if err != nil {
				return err
			}
			if status.UnsafeL2.ID() != seedStatus.UnsafeL2.ID() {
				return fmt.Errorf("sequencer %s unsafe head is %s, want seeded head %s", members[i].ServerID(), status.UnsafeL2, seedStatus.UnsafeL2)
			}
			return nil
		})
		require.NoErrorf(err, "sequencer %s never reached the seeded unsafe head", members[i].ServerID())
	}

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

	clusterNodes := append([]*OpNode{bootstrapNode}, memberNodes...)
	clusterClients := make([]*sources.RollupClient, 0, len(clusterNodes))
	for _, node := range clusterNodes {
		client, err := dial.DialRollupClientWithTimeout(ctx, t.Logger(), node.UserRPC())
		require.NoError(err, "failed to dial conductor-managed sequencer")
		defer client.Close()
		clusterClients = append(clusterClients, client)
	}
	err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		activeLeader := -1
		for i, conductor := range cluster {
			active, err := clusterClients[i].SequencerActive(ctx)
			if err != nil {
				return err
			}
			if conductor.service.Leader(ctx) && active {
				if activeLeader >= 0 {
					return fmt.Errorf("multiple active leader sequencers: %s and %s", cluster[activeLeader].ServerID(), conductor.ServerID())
				}
				activeLeader = i
			} else if active {
				return fmt.Errorf("non-leader conductor %s has an active sequencer", conductor.ServerID())
			}
		}
		if activeLeader < 0 {
			return errors.New("no active leader sequencer")
		}
		return nil
	})
	require.NoError(err, "conductor cluster never reached one active leader sequencer")
}
