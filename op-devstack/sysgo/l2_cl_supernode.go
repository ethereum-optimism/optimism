package sysgo

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	sncfg "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
	sntypes "github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
)

// SuperNode implements a devstack L2CL node backed by op-supernode, configured for a single chain.
type SuperNode struct {
	id     stack.L2CLNodeID
	p      devtest.P
	logger log.Logger
	el     *stack.L2ELNodeID

	// Managed by aggregator
	svc    *supernode.Supernode
	cancel context.CancelFunc

	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32

	// prepared configs
	snCfg  *sncfg.CLIConfig
	vnCfgs map[sntypes.ChainID]*opnodecfg.Config
}

var _ L2CLNode = (*SuperNode)(nil)

// aggregator maintains a single shared supernode process across multiple chains within a test process.
var aggregator = &snAggregator{}

type snAggregator struct {
	mu     sync.Mutex
	port   int
	svc    *supernode.Supernode
	snCfg  *sncfg.CLIConfig
	vnCfgs map[sntypes.ChainID]*opnodecfg.Config
	nodes  []*SuperNode
	logger log.Logger
}

func (a *snAggregator) addChain(n *SuperNode, p devtest.P, baseCfg *sncfg.CLIConfig, chainID sntypes.ChainID, nodeCfg *opnodecfg.Config, logger log.Logger) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logger = logger
	if a.vnCfgs == nil {
		a.vnCfgs = make(map[sntypes.ChainID]*opnodecfg.Config)
	}
	a.vnCfgs[chainID] = nodeCfg
	a.nodes = append(a.nodes, n)
	if a.snCfg == nil {
		cfg := *baseCfg
		cfg.Chains = nil
		// Reserve a shared port up-front so dependents can form URLs
		if a.port == 0 {
			a.port = findFreePort(p)
		}
		cfg.RPCConfig.ListenAddr = "127.0.0.1"
		cfg.RPCConfig.ListenPort = a.port
		cfg.RPCConfig.EnableAdmin = true
		a.snCfg = &cfg
	}
	// Provide a stable userRPC immediately so dependents (e.g. batcher) can dial
	n.userRPC = fmt.Sprintf("http://127.0.0.1:%d/%d", a.port, uint64(chainID))

	// If we have multiple chains configured and the service is not yet started,
	// start the supernode now so downstream services (batcher/proposer) can dial it.
	if a.svc == nil && len(a.vnCfgs) >= 2 {
		chains := make([]uint64, 0, len(a.vnCfgs))
		for cid := range a.vnCfgs {
			chains = append(chains, uint64(cid))
		}
		a.snCfg.Chains = chains
		ctx, cancel := context.WithCancel(p.Ctx())
		stopWithCause := func(err error) { cancel() }
		sn, err := supernode.New(ctx, a.logger, "dev", stopWithCause, a.snCfg, a.vnCfgs)
		p.Require().NoError(err, "failed to create supernode")
		a.svc = sn
		go func() { _ = sn.Start(ctx) }()

		// Wait for each configured chain RPC to expose optimism_rollupConfig
		for cid := range a.vnCfgs {
			url := fmt.Sprintf("http://127.0.0.1:%d/%d", a.port, uint64(cid))
			waitRollupReady(p, logger, url)
		}
	}
}

func (a *snAggregator) start(p devtest.P) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.svc != nil {
		return
	}
	if a.port == 0 {
		a.port = findFreePort(p)
	}
	chains := make([]uint64, 0, len(a.vnCfgs))
	for cid := range a.vnCfgs {
		chains = append(chains, uint64(cid))
	}
	a.snCfg.Chains = chains

	ctx, cancel := context.WithCancel(p.Ctx())
	stopWithCause := func(err error) { cancel() }
	sn, err := supernode.New(ctx, a.logger, "dev", stopWithCause, a.snCfg, a.vnCfgs)
	p.Require().NoError(err, "failed to create supernode")
	a.svc = sn
	go func() { _ = sn.Start(ctx) }()

	for _, n := range a.nodes {
		chainIDU64, _ := n.id.ChainID().Uint64()
		n.userRPC = fmt.Sprintf("http://127.0.0.1:%d/%d", a.port, chainIDU64)
		waitRollupReady(p, a.logger, n.userRPC)
	}
}

func (a *snAggregator) stop(p devtest.P) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.svc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = a.svc.Stop(ctx)
	a.svc = nil
}

// waitRollupReady polls the rollup RPC until optimism_rollupConfig is served
func waitRollupReady(p devtest.P, logger log.Logger, url string) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(p.Ctx(), 500*time.Millisecond)
		rpc, err := client.NewRPC(ctx, logger, url)
		if err == nil {
			var out any
			callErr := rpc.CallContext(ctx, &out, "optimism_rollupConfig")
			rpc.Close()
			if callErr == nil {
				cancel()
				return
			}
		}
		cancel()
		if time.Now().After(deadline) {
			// Don't fail here; downstream will retry too. Just log for visibility.
			logger.Warn("rollup RPC not ready before timeout", "url", url)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

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
	sysL2CL.SetLabel(match.LabelVendor, "supernode")
	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(n.el))
	}
}

func (n *SuperNode) UserRPC() string { return n.userRPC }

func (n *SuperNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

func (n *SuperNode) Start() {}

func (n *SuperNode) Stop() {}

// WithSuperNode adds a supernode-backed L2CL to the orchestrator for a single chain.
func WithSuperNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.Combine[*Orchestrator](
		stack.AfterDeploy(func(orch *Orchestrator) {
			p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), l2CLID))
			require := p.Require()

			l2Net, ok := orch.l2Nets.Get(l2CLID.ChainID())
			require.True(ok, "l2 network required")

			l1EL, ok := orch.l1ELs.Get(l1ELID)
			require.True(ok, "l1 EL node required")

			l1CL, ok := orch.l1CLs.Get(l1CLID)
			require.True(ok, "l1 CL node required")

			l2EL, ok := orch.l2ELs.Get(l2ELID)
			require.True(ok, "l2 EL node required")

			cfg := DefaultL2CLConfig()
			orch.l2CLOptions.Apply(p, l2CLID, cfg)
			L2CLOptionBundle(opts).Apply(p, l2CLID, cfg)

			logger := p.Logger()
			// Build supernode CLI config (single chain)
			// Convert devstack eth.ChainID to uint64 via big-int conversion
			chainIDU64, _ := l2CLID.ChainID().Uint64()
			chainID := sntypes.ChainID(chainIDU64)
			snCfg := &sncfg.CLIConfig{
				Sample:        "devstack",
				Chains:        []uint64{uint64(chainID)},
				DataDir:       p.TempDir(),
				L1NodeAddr:    l1EL.UserRPC(),
				L1BeaconAddr:  l1CL.beaconHTTPAddr,
				RPCConfig:     oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
				MetricsConfig: opmetrics.CLIConfig{},
				PprofConfig:   oppprof.CLIConfig{ListenEnabled: false},
			}

			// Build virtual node config for this chain (op-node config)
			_, jwtSecret := orch.writeDefaultJWT()
			interopCfg := &interop.Config{}
			nodeCfg := &opnodecfg.Config{
				L1: &opnodecfg.L1EndpointConfig{
					L1NodeAddr:       l1EL.UserRPC(),
					L1TrustRPC:       false,
					L1RPCKind:        sources.RPCKindDebugGeth,
					RateLimit:        0,
					BatchSize:        20,
					HttpPollInterval: 100 * time.Millisecond,
					MaxConcurrency:   10,
					CacheSize:        0,
				},
				L2: &opnodecfg.L2EndpointConfig{
					L2EngineAddr:      l2EL.EngineRPC(),
					L2EngineJWTSecret: jwtSecret,
				},
				Beacon:                      &opnodecfg.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
				Driver:                      driver.Config{SequencerEnabled: cfg.IsSequencer, SequencerConfDepth: 2},
				Rollup:                      *l2Net.rollupCfg,
				DependencySet:               nil,
				P2PSigner:                   nil,
				RPC:                         oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
				InteropConfig:               interopCfg,
				P2P:                         nil,
				L1EpochPollInterval:         2 * time.Second,
				RuntimeConfigReloadInterval: 0,
				Tracer:                      nil,
				Sync: nodeSync.Config{SyncMode: func() nodeSync.Mode {
					if cfg.IsSequencer {
						return cfg.SequencerSyncMode
					}
					return cfg.VerifierSyncMode
				}()},
				ConfigPersistence:               opnodecfg.DisabledConfigPersistence{},
				Metrics:                         opmetrics.CLIConfig{},
				Pprof:                           oppprof.CLIConfig{},
				SafeDBPath:                      cfg.SafeDBPath,
				RollupHalt:                      "",
				Cancel:                          nil,
				ConductorEnabled:                false,
				ConductorRpc:                    nil,
				ConductorRpcTimeout:             0,
				AltDA:                           altda.CLIConfig{},
				IgnoreMissingPectraBlobSchedule: false,
				ExperimentalOPStackAPI:          true,
			}
			_ = nodeCfg // ensure used below

			n := &SuperNode{
				id:     l2CLID,
				p:      p,
				logger: logger,
				el:     &l2ELID,
				snCfg:  snCfg,
				vnCfgs: map[sntypes.ChainID]*opnodecfg.Config{chainID: nodeCfg},
			}
			require.True(orch.l2CLs.SetIfMissing(l2CLID, n), fmt.Sprintf("must not already exist: %s", l2CLID))
			aggregator.addChain(n, p, snCfg, chainID, nodeCfg, logger)
		}),
		stack.Finally(func(orch *Orchestrator) {
			aggregator.start(orch.P())
			orch.P().Cleanup(func() { aggregator.stop(orch.P()) })
		}),
	)
}

// findFreePort returns an available localhost TCP port.
func findFreePort(p devtest.P) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	p.Require().NoError(err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// no extra helpers
