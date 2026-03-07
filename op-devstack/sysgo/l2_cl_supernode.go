package sysgo

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type SuperNode struct {
	mu sync.Mutex

	id               stack.SupernodeID
	sn               *supernode.Supernode
	cancel           context.CancelFunc
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	p                devtest.P
	logger           log.Logger
	els              []*stack.ComponentID // Optional: nil when using SyncTester
	chains           []eth.ChainID
	l1UserRPC        string
	l1BeaconAddr     string

	// Configs stored for Start()/restart.
	snCfg  *snconfig.CLIConfig
	vnCfgs map[eth.ChainID]*config.Config
}

var _ L2CLNode = (*SuperNode)(nil)

func (n *SuperNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)
	// note that the system is also hydrated by the SuperNodeProxy.
	// It would be redundant to register nodes here as well.
	system.AddSupernode(shim.NewSuperNode(shim.SuperNodeConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           n.id,
		Client:       rpcCl,
	}))
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

	n.p.Require().NotNil(n.snCfg, "supernode CLI config required")

	ctx, cancel := context.WithCancel(n.p.Ctx())
	exitFn := func(err error) { n.p.Require().NoError(err, "supernode critical error") }
	sn, err := supernode.New(ctx, n.logger, "devstack", exitFn, n.snCfg, n.vnCfgs)
	n.p.Require().NoError(err, "supernode failed to create")
	n.sn = sn
	n.cancel = cancel

	n.p.Require().NoError(n.sn.Start(ctx))

	// Wait for the RPC addr and save userRPC/interop endpoints
	addr, err := n.sn.WaitRPCAddr(ctx)
	n.p.Require().NoError(err, "supernode failed to bind RPC address")
	base := "http://" + addr
	n.userRPC = base
	n.interopEndpoint = base
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

// PauseInteropActivity pauses the interop activity at the given timestamp.
// This function is for integration test control only.
func (n *SuperNode) PauseInteropActivity(ts uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn != nil {
		n.sn.PauseInteropActivity(ts)
	}
}

// ResumeInteropActivity clears any pause on the interop activity.
// This function is for integration test control only.
func (n *SuperNode) ResumeInteropActivity() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn != nil {
		n.sn.ResumeInteropActivity()
	}
}

// WithSupernode constructs a Supernode-based L2 CL node
func WithSupernode(supernodeID stack.SupernodeID, l2CLID stack.ComponentID, l1CLID stack.ComponentID, l1ELID stack.ComponentID, l2ELID stack.ComponentID, opts ...L2CLOption) stack.Option[*Orchestrator] {
	args := []L2CLs{{CLID: l2CLID, ELID: l2ELID}}
	return WithSharedSupernodeCLs(supernodeID, args, l1CLID, l1ELID)
}

// SuperNodeProxy is a thin wrapper that points to a shared supernode instance.
type SuperNodeProxy struct {
	id               stack.ComponentID
	p                devtest.P
	logger           log.Logger
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	el               *stack.ComponentID
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
	l2Net := system.L2Network(stack.ByID[stack.L2Network](stack.NewL2NetworkID(n.id.ChainID())))
	l2Net.(stack.ExtensibleL2Network).AddL2CLNode(sysL2CL)
	if n.el != nil {
		sysL2CL.(stack.LinkableL2CLNode).LinkEL(l2Net.L2ELNode(stack.ByID[stack.L2ELNode](*n.el)))
	}
}

func (n *SuperNodeProxy) Start()          {}
func (n *SuperNodeProxy) Stop()           {}
func (n *SuperNodeProxy) UserRPC() string { return n.userRPC }
func (n *SuperNodeProxy) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

type L2CLs struct {
	CLID stack.ComponentID
	ELID stack.ComponentID
}

// SupernodeConfig holds configuration options for the shared supernode.
type SupernodeConfig struct {
	// InteropActivationTimestamp enables the interop activity at the given timestamp.
	// Set to nil to disable interop (default). Non-nil (including 0) enables interop.
	InteropActivationTimestamp *uint64

	// UseGenesisInterop, when true, sets InteropActivationTimestamp to the genesis
	// timestamp of the first configured chain at deploy time. Takes effect inside
	// withSharedSupernodeCLsImpl after deployment, when the genesis time is known.
	UseGenesisInterop bool
}

// SupernodeOption is a functional option for configuring the supernode.
type SupernodeOption func(*SupernodeConfig)

// WithSupernodeInterop enables the interop activity with the given activation timestamp.
func WithSupernodeInterop(activationTimestamp uint64) SupernodeOption {
	return func(cfg *SupernodeConfig) {
		ts := activationTimestamp
		cfg.InteropActivationTimestamp = &ts
	}
}

// WithSupernodeInteropAtGenesis enables interop at the genesis timestamp of the first
// configured chain. The timestamp is resolved after deployment, when genesis is known.
func WithSupernodeInteropAtGenesis() SupernodeOption {
	return func(cfg *SupernodeConfig) {
		cfg.UseGenesisInterop = true
	}
}

// WithSharedSupernodeCLsInterop starts one supernode for N L2 chains with interop enabled at genesis.
// The interop activation timestamp is computed from the first chain's genesis time.
func WithSharedSupernodeCLsInterop(supernodeID stack.SupernodeID, cls []L2CLs, l1CLID stack.ComponentID, l1ELID stack.ComponentID) stack.Option[*Orchestrator] {
	return WithSharedSupernodeCLs(supernodeID, cls, l1CLID, l1ELID, WithSupernodeInteropAtGenesis())
}

// WithSharedSupernodeCLsInteropDelayed starts one supernode for N L2 chains with interop enabled
// at a specified offset from genesis. This allows testing the transition from non-interop to interop mode.
func WithSharedSupernodeCLsInteropDelayed(supernodeID stack.SupernodeID, cls []L2CLs, l1CLID stack.ComponentID, l1ELID stack.ComponentID, delaySeconds uint64) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		if len(cls) == 0 {
			orch.P().Require().Fail("no chains provided")
			return
		}
		l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(cls[0].CLID.ChainID()))
		if !ok {
			orch.P().Require().Fail("l2 network not found")
			return
		}
		genesisTime := l2Net.rollupCfg.Genesis.L2Time
		activationTime := genesisTime + delaySeconds
		orch.P().Logger().Info("enabling supernode interop with delay",
			"genesis_time", genesisTime,
			"activation_timestamp", activationTime,
			"delay_seconds", delaySeconds,
		)
		withSharedSupernodeCLsImpl(orch, supernodeID, cls, l1CLID, l1ELID, WithSupernodeInterop(activationTime))
	})
}

// WithSharedSupernodeCLs starts one supernode for N L2 chains and registers thin L2CL wrappers.
func WithSharedSupernodeCLs(supernodeID stack.SupernodeID, cls []L2CLs, l1CLID stack.ComponentID, l1ELID stack.ComponentID, opts ...SupernodeOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		withSharedSupernodeCLsImpl(orch, supernodeID, cls, l1CLID, l1ELID, opts...)
	})
}

// withSharedSupernodeCLsImpl is the implementation for starting a shared supernode.
func withSharedSupernodeCLsImpl(orch *Orchestrator, supernodeID stack.SupernodeID, cls []L2CLs, l1CLID stack.ComponentID, l1ELID stack.ComponentID, opts ...SupernodeOption) {
	p := orch.P()
	require := p.Require()

	require.Equal(stack.KindSupernode, supernodeID.Kind(), "supernode ID must be kind Supernode")
	require.Equal(stack.KindL1CLNode, l1CLID.Kind(), "l1 CL ID must be kind L1CLNode")
	require.Equal(stack.KindL1ELNode, l1ELID.Kind(), "l1 EL ID must be kind L1ELNode")
	require.Equal(l1CLID.ChainID(), l1ELID.ChainID(), "l1 CL and EL IDs must be on the same chain")
	require.NotEmpty(cls, "at least one L2 CL/EL pair is required")
	for i := range cls {
		ids := cls[i]
		require.Equalf(stack.KindL2CLNode, ids.CLID.Kind(), "cls[%d].CLID must be kind L2CLNode", i)
		require.Truef(ids.CLID.HasChainID(), "cls[%d].CLID must be chain-scoped", i)
		require.Truef(ids.ELID.HasChainID(), "cls[%d].ELID must be chain-scoped", i)
		require.Equalf(ids.CLID.ChainID(), ids.ELID.ChainID(), "cls[%d] CL and EL IDs must be on the same chain", i)
	}

	// Apply options
	snOpts := &SupernodeConfig{}
	for _, opt := range opts {
		opt(snOpts)
	}

	// Resolve UseGenesisInterop: read the activation timestamp from the first chain's genesis.
	if snOpts.UseGenesisInterop && snOpts.InteropActivationTimestamp == nil {
		p.Require().NotEmpty(cls, "no chains provided for genesis interop resolution")
		l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(cls[0].CLID.ChainID()))
		p.Require().True(ok, "l2 network not found for genesis interop resolution")
		genesisTime := l2Net.rollupCfg.Genesis.L2Time
		p.Logger().Info("enabling supernode interop at genesis", "activation_timestamp", genesisTime)
		snOpts.InteropActivationTimestamp = &genesisTime
	}

	l1EL, ok := orch.GetL1EL(l1ELID)
	require.True(ok, "l1 EL node required")
	l1CL, ok := orch.GetL1CL(l1CLID)
	require.True(ok, "l1 CL node required")

	// Get L1 network to access L1 chain config
	l1Net, ok := orch.GetL1Network(stack.NewL1NetworkID(l1ELID.ChainID()))
	require.True(ok, "l1 network required")

	_, jwtSecret := orch.writeDefaultJWT()

	logger := p.Logger()

	// Build per-chain op-node configs
	makeNodeCfg := func(l2Net *L2Network, l2ChainID eth.ChainID, l2EL L2ELNode, isSequencer bool) *config.Config {
		interopCfg := &interop.Config{}
		l2EngineAddr := l2EL.EngineRPC()
		var depSet depset.DependencySet
		if cluster, ok := orch.ClusterForL2(l2ChainID); ok {
			depSet = cluster.DepSet()
		}
		sequencerP2PKeyHex := ""
		if isSequencer {
			p2pKey, err := orch.keys.Secret(devkeys.SequencerP2PRole.Key(l2ChainID.ToBig()))
			require.NoError(err, "need p2p key for supernode virtual sequencer")
			sequencerP2PKeyHex = hex.EncodeToString(crypto.FromECDSA(p2pKey))
		}
		p2pConfig, p2pSignerSetup := newDevstackP2PConfig(
			p,
			logger.New("chain_id", l2ChainID.String(), "component", "supernode-p2p"),
			l2Net.rollupCfg.BlockTime,
			false,
			true,
			sequencerP2PKeyHex,
		)
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
			DependencySet:                   depSet,
			Beacon:                          &config.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
			Driver:                          driver.Config{SequencerEnabled: isSequencer, SequencerConfDepth: 2},
			Rollup:                          *l2Net.rollupCfg,
			P2PSigner:                       p2pSignerSetup,
			RPC:                             oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
			InteropConfig:                   interopCfg,
			P2P:                             p2pConfig,
			L1EpochPollInterval:             2 * time.Second,
			RuntimeConfigReloadInterval:     0,
			Sync:                            nodeSync.Config{SyncMode: nodeSync.CLSync, SyncModeReqResp: true},
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
	els := make([]*stack.ComponentID, 0, len(cls))
	for i := range cls {
		a := cls[i]
		l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(a.CLID.ChainID()))
		require.True(ok, "l2 network required")
		l2ELNode, ok := orch.GetL2EL(a.ELID)
		require.True(ok, "l2 EL node required")
		l2ChainID := a.CLID.ChainID()
		cfg := makeNodeCfg(l2Net, l2ChainID, l2ELNode, true)
		require.NoError(cfg.Check(), "invalid op-node config for chain %s", a.CLID.ChainID())
		id := eth.EvilChainIDToUInt64(a.CLID.ChainID())
		chainIDs = append(chainIDs, id)
		vnCfgs[eth.ChainIDFromUInt64(id)] = cfg
		els = append(els, &cls[i].ELID)
	}

	// Build supernode CLI config
	snCfg := &snconfig.CLIConfig{
		Chains:                     chainIDs,
		DataDir:                    p.TempDir(),
		L1NodeAddr:                 l1EL.UserRPC(),
		L1BeaconAddr:               l1CL.beaconHTTPAddr,
		RPCConfig:                  oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
		InteropActivationTimestamp: snOpts.InteropActivationTimestamp,
	}
	if snOpts.InteropActivationTimestamp != nil {
		logger.Info("supernode interop enabled", "activation_timestamp", *snOpts.InteropActivationTimestamp)
	}

	snode := &SuperNode{
		id:               supernodeID,
		userRPC:          "",
		interopEndpoint:  "",
		interopJwtSecret: jwtSecret,
		p:                p,
		logger:           logger,
		els:              els,
		chains:           idsFromCLs(cls),
		l1UserRPC:        l1EL.UserRPC(),
		l1BeaconAddr:     l1CL.beaconHTTPAddr,
		snCfg:            snCfg,
		vnCfgs:           vnCfgs,
	}

	// Start and register cleanup, following the same pattern as OpNode.
	snode.Start()
	p.Cleanup(snode.Stop)

	base := snode.UserRPC()

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
	for i := range cls {
		a := cls[i]
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
			el:               &cls[i].ELID,
		}
		cid := a.CLID
		require.False(orch.registry.Has(cid), fmt.Sprintf("must not already exist: %s", a.CLID))
		orch.registry.Register(cid, proxy)
	}

	orch.supernodes.Set(supernodeID, snode)
}

func idsFromCLs(cls []L2CLs) []eth.ChainID {
	out := make([]eth.ChainID, 0, len(cls))
	seen := make(map[eth.ChainID]struct{}, len(cls))
	for _, c := range cls {
		id := c.CLID.ChainID()
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return eth.EvilChainIDToUInt64(out[i]) < eth.EvilChainIDToUInt64(out[j])
	})
	return out
}
