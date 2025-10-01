package sysgo

import (
	"context"
	"fmt"
	"net"
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

func (n *SuperNode) Start() {
	if n.svc != nil {
		n.logger.Warn("supernode already started")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	// Ensure we have a listening port
	port := findFreePort(n.p)
	if n.snCfg != nil {
		n.snCfg.RPCConfig.ListenPort = port
	}

	// supernode.New expects a context.CancelCauseFunc; wrap cancel
	stopWithCause := func(cause error) { cancel() }
	sn, err := supernode.New(ctx, n.logger, "dev", stopWithCause, n.snCfg, n.vnCfgs)
	n.p.Require().NoError(err, "failed to create supernode")
	n.svc = sn

	go func() {
		_ = sn.Start(ctx)
	}()

	// Construct per-chain user RPC (proxy server multiplexes by path)
	var onlyChain sntypes.ChainID
	for cid := range n.vnCfgs {
		onlyChain = cid
		break
	}
	n.userRPC = fmt.Sprintf("http://127.0.0.1:%d/%d", port, uint64(onlyChain))
}

func (n *SuperNode) Stop() {
	if n.svc == nil {
		n.logger.Warn("supernode already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	_ = n.svc.Stop(ctx)
	if n.cancel != nil {
		n.cancel()
	}
	n.svc = nil
}

// WithSuperNode adds a supernode-backed L2CL to the orchestrator for a single chain.
func WithSuperNode(l2CLID stack.L2CLNodeID, l1CLID stack.L1CLNodeID, l1ELID stack.L1ELNodeID, l2ELID stack.L2ELNodeID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
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
		n.Start()
		p.Cleanup(n.Stop)
	})
}

// findFreePort returns an available localhost TCP port.
func findFreePort(p devtest.P) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	p.Require().NoError(err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// no extra helpers
