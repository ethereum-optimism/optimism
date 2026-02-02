package sysgo

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
)

type SuperNode struct {
	mu sync.Mutex

	id               stack.L2CLNodeID
	sn               *supernode.Supernode
	cancel           context.CancelFunc
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	p                devtest.P
	logger           log.Logger
	el               *stack.L2ELNodeID // Optional: nil when using SyncTester
	l1UserRPC        string
	l1BeaconAddr     string
}

var _ L2CLNode = (*SuperNode)(nil)

func (n *SuperNode) hydrate(system stack.ExtensibleSystem) {
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
	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(n.el))
	}
}

func (n *SuperNode) UserRPC() string {
	return n.userRPC
}

func (n *SuperNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

func (n *SuperNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn != nil {
		n.logger.Warn("Supernode already started")
		return
	}

	// Build CLI config for supernode (single-chain)
	cfg := &snconfig.CLIConfig{
		Chains:       []uint64{eth.EvilChainIDToUInt64(n.id.ChainID())},
		DataDir:      n.p.TempDir(),
		L1NodeAddr:   n.l1UserRPC,
		L1BeaconAddr: n.l1BeaconAddr,
		RPCConfig: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		// Other configs (Log/Metrics/Pprof) left default
	}

	// Construct VN config map
	vnCfgs := map[eth.ChainID]*config.Config{}

	// Create Supernode instance
	ctx, cancel := context.WithCancel(n.p.Ctx())
	sn, err := supernode.New(ctx, n.logger, "devstack", func(err error) { n.p.Require().NoError(err, "supernode critical error") }, cfg, vnCfgs)
	n.p.Require().NoError(err, "supernode failed to create")
	n.sn = sn
	n.cancel = cancel

	// Start Supernode in background
	go func() {
		_ = n.sn.Start(ctx)
	}()

	// Wait for the RPC addr and save userRPC/interop endpoints
	if addr, err := n.sn.WaitRPCAddr(ctx); err == nil {
		base := "http://" + addr
		// single-chain instance routes at root
		n.userRPC = base
		n.interopEndpoint = base
	} else {
		n.p.Require().NoError(err, "supernode failed to bind RPC address")
	}

}

func (n *SuperNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		n.logger.Warn("Supernode already stopped")
		return
	}
	if n.cancel != nil {
		n.cancel()
	}
	// Attempt graceful stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = n.sn.Stop(stopCtx)
	n.sn = nil
}

// WithSuperNode constructs a Supernode-based L2 CL node
func WithSuperNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	args := []L2CLs{{CLID: l2CLID, ELID: l2ELID}}
	return WithSharedSupernodeCLs(args, l1CLID, l1ELID)
}

// SuperNodeProxy is a thin wrapper that points to a shared supernode instance.
type SuperNodeProxy struct {
	id               stack.L2CLNodeID
	p                devtest.P
	logger           log.Logger
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	el               *stack.L2ELNodeID
}

var _ L2CLNode = (*SuperNodeProxy)(nil)

func (n *SuperNodeProxy) hydrate(system stack.ExtensibleSystem) {
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
	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(n.el))
	}
}

func (n *SuperNodeProxy) Start()          {}
func (n *SuperNodeProxy) Stop()           {}
func (n *SuperNodeProxy) UserRPC() string { return n.userRPC }
func (n *SuperNodeProxy) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

type L2CLs struct {
	CLID stack.L2CLNodeID
	ELID stack.L2ELNodeID
}

// WithSharedSupernodeCLs starts one supernode for N L2 chains and registers thin L2CL wrappers.
func WithSharedSupernodeCLs(cls []L2CLs, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P()
		require := p.Require()

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		require.True(ok, "l1 EL node required")
		l1CL, ok := orch.l1CLs.Get(l1CLID)
		require.True(ok, "l1 CL node required")

		// Get L1 network to access L1 chain config
		l1Net, ok := orch.l1Nets.Get(l1ELID.ChainID())
		require.True(ok, "l1 network required")

		_, jwtSecret := orch.writeDefaultJWT()

		logger := p.Logger()

		// Build per-chain op-node configs
		makeNodeCfg := func(l2Net *L2Network, l2EL L2ELNode, isSequencer bool) *config.Config {
			interopCfg := &interop.Config{}
			l2EngineAddr := l2EL.EngineRPC()
			return &config.Config{
				L1: &config.L1EndpointConfig{
					L1NodeAddr:       l1EL.UserRPC(),
					L1TrustRPC:       false,
					L1RPCKind:        sources.RPCKindDebugGeth,
					RateLimit:        0,
					BatchSize:        20,
					HttpPollInterval: time.Millisecond * 100,
					MaxConcurrency:   10,
					CacheSize:        0,
				},
				L1ChainConfig: l1Net.genesis.Config,
				L2: &config.L2EndpointConfig{
					L2EngineAddr:      l2EngineAddr,
					L2EngineJWTSecret: jwtSecret,
				},
				Beacon:                          &config.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
				Driver:                          driver.Config{SequencerEnabled: isSequencer, SequencerConfDepth: 2},
				Rollup:                          *l2Net.rollupCfg,
				RPC:                             oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
				InteropConfig:                   interopCfg,
				P2P:                             nil,
				L1EpochPollInterval:             2 * time.Second,
				RuntimeConfigReloadInterval:     0,
				Sync:                            nodeSync.Config{SyncMode: nodeSync.CLSync},
				ConfigPersistence:               config.DisabledConfigPersistence{},
				Metrics:                         opmetrics.CLIConfig{},
				Pprof:                           oppprof.CLIConfig{},
				AltDA:                           altda.CLIConfig{},
				IgnoreMissingPectraBlobSchedule: false,
				ExperimentalOPStackAPI:          true,
			}
		}

		// Gather VN configs and chain IDs
		vnCfgs := make(map[eth.ChainID]*config.Config)
		chainIDs := make([]uint64, 0, len(cls))
		for _, a := range cls {
			l2Net, ok := orch.l2Nets.Get(a.CLID.ChainID())
			require.True(ok, "l2 network required")
			l2ELNode, ok := orch.l2ELs.Get(a.ELID)
			require.True(ok, "l2 EL node required")
			cfg := makeNodeCfg(l2Net, l2ELNode, true)
			id := eth.EvilChainIDToUInt64(a.CLID.ChainID())
			chainIDs = append(chainIDs, id)
			vnCfgs[eth.ChainIDFromUInt64(id)] = cfg
		}

		// Start shared supernode with all chains
		snCfg := &snconfig.CLIConfig{
			Chains:       chainIDs,
			DataDir:      p.TempDir(),
			L1NodeAddr:   l1EL.UserRPC(),
			L1BeaconAddr: l1CL.beaconHTTPAddr,
			RPCConfig:    oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
		}
		ctx, cancel := context.WithCancel(p.Ctx())
		exitFn := func(err error) { p.Require().NoError(err, "supernode critical error") }
		sn, err := supernode.New(ctx, logger, "devstack", exitFn, snCfg, vnCfgs)
		require.NoError(err)
		go func() { _ = sn.Start(ctx) }()
		// Resolve bound address
		addr, err := sn.WaitRPCAddr(ctx)
		require.NoError(err, "failed waiting for supernode RPC addr")
		base := "http://" + addr
		p.Cleanup(func() {
			stopCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
			_ = sn.Stop(stopCtx)
			c()
			cancel()
		})
		// Wait for per-chain RPC routes to serve optimism_rollupConfig and register proxies
		waitReady := func(u string) {
			deadline := time.Now().Add(15 * time.Second)
			for {
				if time.Now().After(deadline) {
					require.FailNow(fmt.Sprintf("timed out waiting for RPC to be ready at %s", u))
				}
				rpcCl, err := client.NewRPC(p.Ctx(), logger, u, client.WithLazyDial())
				if err == nil {
					var v any
					if callErr := rpcCl.CallContext(p.Ctx(), &v, "optimism_rollupConfig"); callErr == nil {
						rpcCl.Close()
						break
					}
					rpcCl.Close()
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
		for _, a := range cls {
			// Multi-chain router exposes per-chain namespace paths
			rpc := base + "/" + strconv.FormatUint(eth.EvilChainIDToUInt64(a.CLID.ChainID()), 10)
			waitReady(rpc)
			proxy := &SuperNodeProxy{
				id:               a.CLID,
				p:                p,
				logger:           logger,
				userRPC:          rpc,
				interopEndpoint:  rpc,
				interopJwtSecret: jwtSecret,
				el:               &a.ELID,
			}
			require.True(orch.l2CLs.SetIfMissing(a.CLID, proxy), fmt.Sprintf("must not already exist: %s", a.CLID))
		}
	})
}
