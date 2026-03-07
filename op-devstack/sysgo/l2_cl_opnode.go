package sysgo

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type OpNode struct {
	mu sync.Mutex

	id               stack.ComponentID
	opNode           *opnode.Opnode
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	cfg              *config.Config
	p                devtest.P
	logger           log.Logger
	el               *stack.ComponentID // Optional: nil when using SyncTester
	userProxy        *tcpproxy.Proxy
	interopProxy     *tcpproxy.Proxy
	clock            clock.Clock
}

var _ L2CLNode = (*OpNode)(nil)

func (n *OpNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	sysL2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		CommonConfig:     shim.NewCommonConfig(system.T()),
		ID:               n.id,
		Client:           rpcCl,
		UserRPC:          n.userRPC,
		InteropEndpoint:  n.interopEndpoint,
		InteropJwtSecret: n.interopJwtSecret,
	})
	sysL2CL.SetLabel(match.LabelVendor, string(match.OpNode))
	l2Net := system.L2Network(stack.ByID[stack.L2Network](stack.NewL2NetworkID(n.id.ChainID())))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		for _, el := range l2Net.L2ELNodes() {
			if el.ID() == *n.el {
				sysL2CL.(stack.LinkableL2CLNode).LinkEL(el)
				return
			}
		}
		rbID := stack.NewRollupBoostNodeID(n.el.Key(), n.el.ChainID())
		for _, rb := range l2Net.RollupBoostNodes() {
			if rb.ID() == rbID {
				sysL2CL.(stack.LinkableL2CLNode).LinkRollupBoostNode(rb)
				return
			}
		}
		oprbID := stack.NewOPRBuilderNodeID(n.el.Key(), n.el.ChainID())
		for _, oprb := range l2Net.OPRBuilderNodes() {
			if oprb.ID() == oprbID {
				sysL2CL.(stack.LinkableL2CLNode).LinkOPRBuilderNode(oprb)
				return
			}
		}
	}
}

func (n *OpNode) UserRPC() string {
	return n.userRPC
}

func (n *OpNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	// Make sure to use the proxied interop endpoint
	return n.interopEndpoint, n.interopJwtSecret
}

func (n *OpNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode != nil {
		n.logger.Warn("Op-node already started")
		return
	}

	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.logger.New("proxy", "l2cl-user"))
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "http://" + n.userProxy.Addr()
	}
	if n.interopProxy == nil {
		n.interopProxy = tcpproxy.New(n.logger.New("proxy", "l2cl-interop"))
		n.p.Require().NoError(n.interopProxy.Start())
		n.p.Cleanup(func() {
			n.interopProxy.Close()
		})
		n.interopEndpoint = "ws://" + n.interopProxy.Addr()
	}
	n.logger.Info("Starting op-node")
	opNode, err := opnode.NewOpnode(n.logger, n.cfg, n.clock, func(err error) {
		n.p.Require().NoError(err, "op-node critical error")
	})
	n.p.Require().NoError(err, "op-node failed to start")
	n.logger.Info("Started op-node")
	n.opNode = opNode

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), opNode.UserRPC().RPC()))

	interopEndpoint, interopJwtSecret := opNode.InteropRPC()
	n.interopProxy.SetUpstream(ProxyAddr(n.p.Require(), interopEndpoint))
	n.interopJwtSecret = interopJwtSecret
}

func (n *OpNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode == nil {
		n.logger.Warn("Op-node already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	n.logger.Info("Closing op-node")
	closeErr := n.opNode.Stop(ctx)
	n.logger.Info("Closed op-node", "err", closeErr)

	n.opNode = nil
}

func WithOpNodeFollowL2(l2CLID stack.ComponentID, l1CLID stack.ComponentID, l1ELID stack.ComponentID, l2ELID stack.ComponentID, l2FollowSourceID stack.ComponentID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		followSource := func(orch *Orchestrator) string {
			p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2CLID))
			l2CLFollowSource, ok := orch.GetL2CL(l2FollowSourceID)
			p.Require().True(ok, "l2 CL Follow Source required")
			return l2CLFollowSource.UserRPC()
		}(orch)
		opts = append(opts, L2CLFollowSource(followSource))
		withOpNode(l2CLID, l1CLID, l1ELID, l2ELID, opts...)(orch)
	})
}

func WithOpNode(l2CLID stack.ComponentID, l1CLID stack.ComponentID, l1ELID stack.ComponentID, l2ELID stack.ComponentID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(withOpNode(l2CLID, l1CLID, l1ELID, l2ELID, opts...))
}

func withOpNode(l2CLID stack.ComponentID, l1CLID stack.ComponentID, l1ELID stack.ComponentID, l2ELID stack.ComponentID, opts ...L2CLOption) func(orch *Orchestrator) {
	return func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2CLID))

		require := p.Require()

		l1Net, ok := orch.GetL1Network(stack.NewL1NetworkID(l1CLID.ChainID()))
		require.True(ok, "l1 network required")

		l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(l2CLID.ChainID()))
		require.True(ok, "l2 network required")

		l1EL, ok := orch.GetL1EL(l1ELID)
		require.True(ok, "l1 EL node required")

		l1CL, ok := orch.GetL1CL(l1CLID)
		require.True(ok, "l1 CL node required")

		// Get the L2EL node (which can be a regular EL node or a SyncTesterEL)
		l2EL, ok := orch.GetL2EL(l2ELID)
		require.True(ok, "l2 EL node required")

		// Get dependency set from cluster if available
		var depSet depset.DependencySet
		if cluster, ok := orch.ClusterForL2(l2ELID.ChainID()); ok {
			depSet = cluster.DepSet()
		}

		cfg := DefaultL2CLConfig()
		orch.l2CLOptions.Apply(p, l2CLID, cfg)       // apply global options
		L2CLOptionBundle(opts).Apply(p, l2CLID, cfg) // apply specific options

		syncMode := cfg.VerifierSyncMode
		if cfg.IsSequencer {
			syncMode = cfg.SequencerSyncMode
			// Sanity check, to navigate legacy sync-mode test assumptions.
			// Can't enable ELSync on the sequencer or it will never start sequencing because
			// ELSync needs to receive gossip from the sequencer to drive the sync
			p.Require().NotEqual(nodeSync.ELSync, syncMode, "sequencer cannot use EL sync")
		}

		jwtPath, jwtSecret := orch.writeDefaultJWT()

		logger := p.Logger()

		sequencerP2PKeyHex := ""
		if cfg.IsSequencer {
			p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID().ToBig()))
			require.NoError(err, "need p2p key for sequencer")
			sequencerP2PKeyHex = hex.EncodeToString(crypto.FromECDSA(p2pKey))
		}
		p2pConfig, p2pSignerSetup := newDevstackP2PConfig(
			p,
			logger,
			l2Net.rollupCfg.BlockTime,
			cfg.NoDiscovery,
			cfg.EnableReqRespSync,
			sequencerP2PKeyHex,
		)

		// specify interop config, but do not configure anything, to disable indexing mode
		interopCfg := &interop.Config{}

		if cfg.IndexingMode {
			interopCfg = &interop.Config{
				RPCAddr: "127.0.0.1",
				// When L2CL starts, store its RPC port here
				// given by the os, to reclaim when restart.
				RPCPort:          0,
				RPCJwtSecretPath: jwtPath,
			}
		}

		nodeCfg := &config.Config{
			L1: &config.L1EndpointConfig{
				L1NodeAddr:       l1EL.UserRPC(),
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: time.Millisecond * 100,
				MaxConcurrency:   10,
				CacheSize:        0, // auto-adjust to sequence window
			},
			L1ChainConfig: l1Net.genesis.Config,
			L2: &config.L2EndpointConfig{
				L2EngineAddr:      l2EL.EngineRPC(),
				L2EngineJWTSecret: jwtSecret,
			},
			L2FollowSource: &config.L2FollowSourceConfig{
				L2RPCAddr: cfg.FollowSource,
			},
			Beacon: &config.L1BeaconEndpointConfig{
				BeaconAddr: l1CL.beaconHTTPAddr,
			},
			Driver: driver.Config{
				SequencerEnabled:   cfg.IsSequencer,
				SequencerConfDepth: 2,
			},
			Rollup:            *l2Net.rollupCfg,
			DependencySet:     depSet,
			SupervisorEnabled: cfg.IndexingMode,
			P2PSigner:         p2pSignerSetup, // nil when not sequencer
			RPC: oprpc.CLIConfig{
				ListenAddr: "127.0.0.1",
				// When L2CL starts, store its RPC port here
				// given by the os, to reclaim when restart.
				ListenPort:  0,
				EnableAdmin: true,
			},
			InteropConfig:               interopCfg,
			P2P:                         p2pConfig,
			L1EpochPollInterval:         time.Second * 2,
			RuntimeConfigReloadInterval: 0,
			Tracer:                      nil,
			Sync: nodeSync.Config{
				SyncMode:                       syncMode,
				SyncModeReqResp:                cfg.UseReqRespSync,
				SkipSyncStartCheck:             false,
				SupportsPostFinalizationELSync: false,
				L2FollowSourceEndpoint:         cfg.FollowSource,
				NeedInitialResetEngine:         cfg.IsSequencer && cfg.FollowSource != "",
			},
			ConfigPersistence:               config.DisabledConfigPersistence{},
			Metrics:                         opmetrics.CLIConfig{},
			Pprof:                           oppprof.CLIConfig{},
			SafeDBPath:                      "",
			RollupHalt:                      "",
			Cancel:                          nil,
			ConductorEnabled:                false,
			ConductorRpc:                    nil,
			ConductorRpcTimeout:             0,
			AltDA:                           altda.CLIConfig{},
			IgnoreMissingPectraBlobSchedule: false,
			ExperimentalOPStackAPI:          true,
		}
		if cfg.SafeDBPath != "" {
			nodeCfg.SafeDBPath = cfg.SafeDBPath
		}

		l2CLNode := &OpNode{
			id:     l2CLID,
			cfg:    nodeCfg,
			logger: logger,
			p:      p,
		}

		if orch.timeTravelClock != nil {
			l2CLNode.clock = orch.timeTravelClock
		}

		// Set the EL field to link to the L2EL node
		l2CLNode.el = &l2ELID
		cid := l2CLID
		require.False(orch.registry.Has(cid), fmt.Sprintf("must not already exist: %s", l2CLID))
		orch.registry.Register(cid, l2CLNode)
		l2CLNode.Start()
		p.Cleanup(l2CLNode.Stop)
	}
}
