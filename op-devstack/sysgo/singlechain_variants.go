package sysgo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"

	opconductor "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
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
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
	nodeB := addSingleChainOpNode(t, runtime, "b", true, "", cfg.GlobalL2CLOptions...)
	nodeC := addSingleChainOpNode(t, runtime, "c", true, "", cfg.GlobalL2CLOptions...)

	conductorCfg := conductorConfigFromPreset(cfg)
	conductorA := startConductorNode(t, "sequencer", runtime.L2Network, runtime.L2CL.(*OpNode), runtime.L2EL, true, false, conductorCfg)
	conductorB := startConductorNode(t, "b", runtime.L2Network, nodeB.CL.(*OpNode), nodeB.EL, false, true, conductorCfg)
	conductorC := startConductorNode(t, "c", runtime.L2Network, nodeC.CL.(*OpNode), nodeC.EL, false, true, conductorCfg)
	connectSingleChainCLPeer(t, runtime.L2CL, nodeB.CL)
	connectSingleChainCLPeer(t, runtime.L2CL, nodeC.CL)
	startConductorCluster(t, conductorA, []*Conductor{conductorB, conductorC})
	waitForRollupSequencerActive(t, runtime.L2CL.UserRPC())

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
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELForKey(t, runtime.L2Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0))
	l2CL := startL2CLForKey(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, name, name, isSequencer, followSource, l2Opts)
	node := newSingleChainNodeRuntime(name, isSequencer, l2EL, l2CL)
	runtime.Nodes[name] = node
	return node
}

type conductorNodeConfig struct {
	HealthCheck          opconductor.HealthCheckConfig
	ReorgRecoveryEnabled bool
}

func conductorConfigFromPreset(cfg PresetConfig) conductorNodeConfig {
	healthCfg := opconductor.HealthCheckConfig{
		Interval:       3600,
		UnsafeInterval: 3600,
		SafeInterval:   3600,
		MinPeerCount:   1,
	}
	if cfg.ConductorHealthCheck != nil {
		healthCfg = *cfg.ConductorHealthCheck
	}
	return conductorNodeConfig{
		HealthCheck:          healthCfg,
		ReorgRecoveryEnabled: cfg.ConductorReorgRecoveryEnabled,
	}
}

// reorgRecoveryWSURL returns the op-reth WebSocket URL the conductor should
// subscribe to for reorg notifications, or "" when reorg recovery is disabled.
// In devstack the op-reth EL exposes its user RPC over WebSocket (ws://) with the
// `reth` namespace enabled, so the execution RPC endpoint doubles as the WS URL.
func reorgRecoveryWSURL(conductorCfg conductorNodeConfig, executionRPC string) string {
	if !conductorCfg.ReorgRecoveryEnabled {
		return ""
	}
	return executionRPC
}

func newConductorRPCEndpoint() *atomic.Value {
	var conductorRPCEndpoint atomic.Value
	conductorRPCEndpoint.Store("")
	return &conductorRPCEndpoint
}

func configureOpNodeConfigForConductor(cfg *opnodeconfig.Config, conductorRPCEndpoint *atomic.Value) {
	cfg.ConductorEnabled = true
	cfg.ConductorRpcTimeout = 5 * time.Second
	cfg.ConductorRpc = conductorRPCFromEndpoint(conductorRPCEndpoint)
	cfg.Driver.SequencerStopped = true
}

func configureOpNodeForConductor(opNode *OpNode, conductorRPCEndpoint *atomic.Value) {
	configureOpNodeConfigForConductor(opNode.cfg, conductorRPCEndpoint)
	if p2pCfg, ok := opNode.cfg.P2P.(*p2p.Config); ok {
		p2pCfg.Store = dssync.MutexWrap(ds.NewMapDatastore())
	}
}

func conductorRPCFromEndpoint(conductorRPCEndpoint *atomic.Value) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
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
	conductorCfg conductorNodeConfig,
) *Conductor {
	conductorRPCEndpoint := newConductorRPCEndpoint()
	configureOpNodeForConductor(opNode, conductorRPCEndpoint)
	opNode.Stop()
	opNode.Start()

	return startConductorForRPC(t, conductorName, l2Net, opNode.UserRPC(), l2EL.UserRPC(), bootstrap, paused, conductorRPCEndpoint, conductorCfg)
}

func startConductorForRPC(
	t devtest.T,
	conductorName string,
	l2Net *L2Network,
	nodeRPC string,
	executionRPC string,
	bootstrap bool,
	paused bool,
	conductorRPCEndpoint *atomic.Value,
	conductorCfg conductorNodeConfig,
) *Conductor {
	require := t.Require()
	serverID := conductorName
	require.NotEmpty(serverID, "conductor ID key cannot be empty")

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
		NodeRPC:                 nodeRPC,
		ExecutionRPC:            executionRPC,
		Paused:                  paused,
		HealthCheck:             conductorCfg.HealthCheck,
		RollupCfg:               *l2Net.rollupCfg,
		RPCEnableProxy:          false,
		ReorgRecoveryEnabled:    conductorCfg.ReorgRecoveryEnabled,
		ExecutionWSURL:          reorgRecoveryWSURL(conductorCfg, executionRPC),
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelInfo,
			Format: oplog.FormatText,
			Color:  false,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
		MetricsConfig: opmetrics.CLIConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
	}

	logger := t.Logger().New("component", "conductor", "name", conductorName, "chain", l2Net.ChainID())
	initialUnsafePayload := fetchInitialUnsafePayload(t, nodeRPC, executionRPC)
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
		name:                 conductorName,
		chainID:              l2Net.ChainID(),
		serverID:             serverID,
		consensusEndpoint:    svc.ConsensusEndpoint(),
		rpcEndpoint:          svc.HTTPEndpoint(),
		metricsEndpoint:      svc.MetricsEndpoint(),
		initialUnsafePayload: initialUnsafePayload,
		service:              svc,
	}
	if conductorRPCEndpoint != nil {
		conductorRPCEndpoint.Store(svc.HTTPEndpoint())
	}
	return out
}

func fetchInitialUnsafePayload(t devtest.T, nodeRPC string, executionRPC string) *eth.ExecutionPayloadEnvelope {
	nodeRPCClient, err := opclient.NewRPC(t.Ctx(), t.Logger(), nodeRPC)
	t.Require().NoError(err, "dial node RPC for conductor unsafe head")
	defer nodeRPCClient.Close()
	rollupClient := sources.NewRollupClient(nodeRPCClient)

	executionRPCClient, err := opclient.NewRPC(t.Ctx(), t.Logger(), executionRPC)
	t.Require().NoError(err, "dial execution RPC for conductor unsafe head")
	defer executionRPCClient.Close()
	ethClient, err := sources.NewEthClient(executionRPCClient, t.Logger(), nil, sources.DefaultEthClientConfig(10))
	t.Require().NoError(err, "create execution client for conductor unsafe head")

	var payload *eth.ExecutionPayloadEnvelope
	err = retry.Do0(t.Ctx(), 120, retry.Fixed(250*time.Millisecond), func() error {
		status, err := rollupClient.SyncStatus(t.Ctx())
		if err != nil {
			return fmt.Errorf("fetch node sync status: %w", err)
		}
		if status == nil {
			return errors.New("node sync status is nil")
		}
		currentPayload, err := ethClient.PayloadByHash(t.Ctx(), status.UnsafeL2.Hash)
		if err != nil {
			return fmt.Errorf("fetch unsafe payload %s: %w", status.UnsafeL2, err)
		}
		if currentPayload.ExecutionPayload.BlockHash != status.UnsafeL2.Hash ||
			uint64(currentPayload.ExecutionPayload.BlockNumber) != status.UnsafeL2.Number {
			return fmt.Errorf("unsafe payload %s does not match sync status %s", currentPayload.ID(), status.UnsafeL2)
		}
		payload = currentPayload
		return nil
	})
	t.Require().NoError(err, "fetch conductor initial unsafe payload")
	return payload
}

func waitForRollupSequencerActive(t devtest.T, nodeRPC string) {
	require := t.Require()
	rpcClient, err := opclient.NewRPC(t.Ctx(), t.Logger(), nodeRPC)
	require.NoError(err, "dial op-node RPC for sequencer active check")
	defer rpcClient.Close()
	rollupClient := sources.NewRollupClient(rpcClient)

	ctx, cancel := context.WithTimeout(t.Ctx(), 30*time.Second)
	defer cancel()
	err = retry.Do0(ctx, 60, retry.Fixed(500*time.Millisecond), func() error {
		active, err := rollupClient.SequencerActive(ctx)
		if err != nil {
			return err
		}
		if !active {
			return errors.New("sequencer is not active yet")
		}
		return nil
	})
	require.NoError(err, "sequencer never became active")
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

	if bootstrap.initialUnsafePayload != nil {
		err = retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return bootstrap.service.CommitUnsafePayload(ctx, bootstrap.initialUnsafePayload)
		})
		require.NoError(err, "failed to seed conductor unsafe head")
	}

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
