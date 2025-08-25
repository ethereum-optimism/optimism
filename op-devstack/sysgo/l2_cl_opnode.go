package sysgo

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/urfave/cli/v2"

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
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

type OpNode struct {
	mu sync.Mutex

	id               stack.L2CLNodeID
	opNode           *opnode.Opnode
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	cfg              *config.Config
	p                devtest.P
	logger           log.Logger
	el               *stack.L2ELNodeID // Optional: nil when using SyncTester
	userProxy        *tcpproxy.Proxy
	interopProxy     *tcpproxy.Proxy
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
		InteropEndpoint:  n.interopEndpoint,
		InteropJwtSecret: n.interopJwtSecret,
	})
	sysL2CL.SetLabel(match.LabelVendor, string(match.OpNode))
	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(n.el))
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
	opNode, err := opnode.NewOpnode(n.logger, n.cfg, func(err error) {
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

func WithOpNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELOrSyncTester interface{}, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2CLID))

		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID())
		require.True(ok, "l2 network required")

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")

		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		// Handle either L2EL node or SyncTester
		var l2EL L2ELNode
		var l2ELID stack.L2ELNodeID
		var syncTesterID *stack.SyncTesterID
		var depSet depset.DependencySet
		var useSyncTester bool
		var fcus sttypes.FCUState

		switch v := l2ELOrSyncTester.(type) {
		case stack.L2ELNodeID:
			l2ELID = v
			l2EL, ok = orch.l2ELs.Get(v)
			require.True(ok, "l2 EL node required")
			if cluster, ok := orch.ClusterForL2(v.ChainID()); ok {
				depSet = cluster.DepSet()
			}
		case nil:
			syncTester := orch.syncTester
			syncTesterID = &syncTester.id

			useSyncTester = true

			for _, st := range orch.syncTester.service.SyncTesters() {
				if st.ChainID == syncTesterID.ChainID() {
					fcus = st.Target
					break
				}
			}
			require.NotNil(fcus, "target blocks not found for sync tester")

			// When using a SyncTester, we don't need an L2EL node
			// The SyncTester will provide the L2 endpoint
			if cluster, ok := orch.ClusterForL2(syncTesterID.ChainID()); ok {
				depSet = cluster.DepSet()
			}
		default:
			require.Fail("l2ELOrSyncTester must be either stack.L2ELNodeID or nil")
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

		var p2pSignerSetup p2p.SignerSetup
		var p2pConfig *p2p.Config
		// code block for P2P setup
		{
			// make a dummy flagset since p2p config initialization helpers only input cli context
			fs := flag.NewFlagSet("", flag.ContinueOnError)
			// use default flags
			for _, f := range opNodeFlags.P2PFlags(opNodeFlags.EnvVarPrefix) {
				require.NoError(f.Apply(fs))
			}
			// mandatory P2P flags
			require.NoError(fs.Set(opNodeFlags.AdvertiseIPName, "127.0.0.1"))
			require.NoError(fs.Set(opNodeFlags.AdvertiseTCPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.AdvertiseUDPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.ListenIPName, "127.0.0.1"))
			require.NoError(fs.Set(opNodeFlags.ListenTCPPortName, "0"))
			require.NoError(fs.Set(opNodeFlags.ListenUDPPortName, "0"))
			// avoid resource unavailable error by using memorydb
			require.NoError(fs.Set(opNodeFlags.DiscoveryPathName, "memory"))
			require.NoError(fs.Set(opNodeFlags.PeerstorePathName, "memory"))
			// For peer ID
			networkPrivKey, err := crypto.GenerateKey()
			require.NoError(err)
			networkPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(networkPrivKey))
			require.NoError(fs.Set(opNodeFlags.P2PPrivRawName, networkPrivKeyHex))
			// Explicitly set to empty; do not default to resolving DNS of external bootnodes
			require.NoError(fs.Set(opNodeFlags.BootnodesName, ""))

			cliCtx := cli.NewContext(&cli.App{}, fs, nil)
			if cfg.IsSequencer {
				p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2CLID.ChainID().ToBig()))
				require.NoError(err, "need p2p key for sequencer")
				p2pKeyHex := hex.EncodeToString(crypto.FromECDSA(p2pKey))
				require.NoError(fs.Set(opNodeFlags.SequencerP2PKeyName, p2pKeyHex))
				p2pSignerSetup, err = p2pcli.LoadSignerSetup(cliCtx, logger)
				require.NoError(err, "failed to load p2p signer")
				logger.Info("Sequencer key acquired")
			}
			p2pConfig, err = p2pcli.NewConfig(cliCtx, l2Net.rollupCfg.BlockTime)
			require.NoError(err, "failed to load p2p config")
		}

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

		// Determine the L2 endpoint based on whether we're using a SyncTester or L2EL
		var l2EngineAddr string
		if useSyncTester {
			require.NotNil(orch.syncTester, "sync tester service required when using SyncTester")
			l2EngineAddr = orch.syncTester.service.NewEndpoint(syncTesterID.ChainID())
			l2EngineAddr += fmt.Sprintf("?latest=%d&safe=%d&finalized=%d", fcus.Latest, fcus.Safe, fcus.Finalized)
		} else {
			l2EngineAddr = l2EL.EngineRPC()
		}

		nodeCfg := &config.Config{
			L1: &config.L1EndpointConfig{
				L1NodeAddr:       l1EL.userRPC,
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: time.Millisecond * 100,
				MaxConcurrency:   10,
				CacheSize:        0, // auto-adjust to sequence window
			},
			L2: &config.L2EndpointConfig{
				L2EngineAddr:      l2EngineAddr,
				L2EngineJWTSecret: jwtSecret,
			},
			Beacon: &config.L1BeaconEndpointConfig{
				BeaconAddr: l1CL.beacon.BeaconAddr(),
			},
			Driver: driver.Config{
				SequencerEnabled:   cfg.IsSequencer,
				SequencerConfDepth: 2,
			},
			Rollup:        *l2Net.rollupCfg,
			DependencySet: depSet,
			P2PSigner:     p2pSignerSetup, // nil when not sequencer
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
				SkipSyncStartCheck:             false,
				SupportsPostFinalizationELSync: false,
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

		// Set the EL field only if we have an L2EL node
		if !useSyncTester {
			l2CLNode.el = &l2ELID
		}
		require.True(orch.l2CLs.SetIfMissing(l2CLID, l2CLNode), fmt.Sprintf("must not already exist: %s", l2CLID))
		l2CLNode.Start()
		p.Cleanup(l2CLNode.Stop)
	})
}
