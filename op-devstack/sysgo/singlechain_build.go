package sysgo

import (
	"encoding/hex"
	"flag"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/config"
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
)

type testSequencer struct {
	name       string
	adminRPC   string
	jwtSecret  [32]byte
	controlRPC map[eth.ChainID]string
	service    *sequencer.Service
}

func buildSingleChainWorld(t devtest.T, keys devkeys.Keys, localContractArtifactsPath string, deployerOpts ...DeployerOption) (*L1Network, *L2Network) {
	wb := &worldBuilder{
		p:       t,
		logger:  t.Logger(),
		require: t.Require(),
		keys:    keys,
		builder: intentbuilder.New(),
	}

	applyConfigLocalContractSources(t, keys, wb.builder, localContractArtifactsPath)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 1, "expected exactly one L2 chain in single-chain world")
	l2ID := wb.l2Chains[0]
	l1ID := eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID)

	l1Net := &L1Network{
		name:      "l1",
		chainID:   l1ID,
		genesis:   wb.outL1Genesis,
		blockTime: 6,
	}
	l2Net := &L2Network{
		name:       "l2a",
		chainID:    l2ID,
		l1ChainID:  l1ID,
		genesis:    wb.outL2Genesis[l2ID],
		rollupCfg:  wb.outL2RollupCfg[l2ID],
		deployment: wb.outL2Deployment[l2ID],
		opcmImpl:   wb.output.ImplementationsDeployment.OpcmV2Impl,
		mipsImpl:   wb.output.ImplementationsDeployment.MipsImpl,
		keys:       keys,
	}
	return l1Net, l2Net
}

func applyConfigLocalContractSources(t devtest.T, _ devkeys.Keys, builder intentbuilder.Builder, artifactsPath string) {
	contractArtifacts, err := localContractSourcesLocator(artifactsPath)
	t.Require().NoError(err)
	builder.WithL1ContractsLocator(contractArtifacts)
	builder.WithL2ContractsLocator(contractArtifacts)
}

func applyConfigCommons(t devtest.T, keys devkeys.Keys, l1ChainID eth.ChainID, builder intentbuilder.Builder) {
	WithCommons(l1ChainID)(t, keys, builder)
}

func applyConfigPrefundedL2(t devtest.T, keys devkeys.Keys, l1ChainID, l2ChainID eth.ChainID, builder intentbuilder.Builder) {
	_, l2Config := builder.WithL2(l2ChainID)
	intentbuilder.WithDevkeyVaults(t, keys, l2Config)
	intentbuilder.WithDevkeyL2Roles(t, keys, l2Config)
	intentbuilder.WithDevkeyL1Roles(t, keys, l2Config, l1ChainID)

	funderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
	t.Require().NoError(err, "need funder addr")
	l2Config.WithPrefundedAccount(funderAddr, *eth.BillionEther.ToU256())

	addrFor := intentbuilder.RoleToAddrProvider(t, keys, l2ChainID)
	l1Config := l2Config.L1Config()
	l1Config.WithPrefundedAccount(addrFor(devkeys.BatcherRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.ProposerRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.ChallengerRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.SystemConfigOwner), *millionEth)
}

// startL2ELForKey starts an L2 EL node for the given key, respecting DEVSTACK_L2EL_KIND.
// This is the single env-aware dispatch point for L2 EL selection.
func startL2ELForKey(t devtest.T, l2Net *L2Network, jwtPath string, jwtSecret [32]byte, key string, identity *ELNodeIdentity, opts ...OpRethOption) L2ELNode {
	switch k := devstackL2ELKind(); k {
	case MixedL2ELOpGeth:
		// op-reth options have no analogue on these kinds. Dropping them silently would make a
		// binary override look like it took effect on a node that never saw it.
		t.Require().Empty(opts, "op-reth options cannot be applied to EL kind %q", k)
		return startL2ELNode(t, l2Net, jwtPath, jwtSecret, key, identity)
	case MixedL2ELOpRethV2:
		return startMixedOpRethNode(t, l2Net, key, jwtPath, jwtSecret, nil, "v2", opts...)
	case "", MixedL2ELOpReth: // unset (default) or explicit op-reth v1
		return startMixedOpRethNode(t, l2Net, key, jwtPath, jwtSecret, nil, "v1", opts...)
	default:
		t.Require().FailNow("unsupported L2 EL kind", "unknown DEVSTACK_L2EL_KIND %q", k)
		return nil // unreachable
	}
}

type l2CLStartResult struct {
	Node           L2CLNode
	FactoryHandled bool
}

// startL2CLForKeyWithKind is the explicit-fallback form used by mixed-client
// runtimes. FactoryHandled lets callers avoid assuming that an external CL
// implements op-node's devp2p admin RPCs while preserving ordinary peering
// when the factory declines a slot. A non-nil factory receives fully resolved
// L2CLOptions before it selects or declines the slot; only a nil-factory Kona
// fallback retains Kona's historical behavior of ignoring those options.
func startL2CLForKeyWithKind(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	jwtSecret [32]byte,
	clKey, elKey string,
	isSequencer bool,
	followSource string,
	depSet depset.DependencySet,
	l2CLOpts []L2CLOption,
	factory L2CLFactory,
	unsafeSourceEL L2ELNode,
	fallbackSourceFactoryHandled bool,
	fallbackKind MixedL2CLKind,
) l2CLStartResult {
	startCfg := l2CLNodeStartConfig{
		Key:            clKey,
		IsSequencer:    isSequencer,
		NoDiscovery:    true,
		EnableReqResp:  true,
		L2FollowSource: followSource,
		DependencySet:  depSet,
		L2CLOptions:    l2CLOpts,
	}
	target := NewComponentTarget(clKey, l2Net.ChainID())
	var resolvedCfg *L2CLConfig
	if factory != nil {
		resolvedCfg = resolveL2CLNodeConfig(t, target, startCfg)
		role := L2CLRoleVerifier
		if resolvedCfg.IsSequencer {
			role = L2CLRoleSequencer
		}
		launchCtx := L2CLLaunchContext{
			Target:        target,
			Role:          role,
			L1UserRPC:     l1EL.UserRPC(),
			L1BeaconRPC:   l1CL.BeaconHTTPAddr(),
			L2UserRPC:     l2EL.UserRPC(),
			L2EngineRPC:   l2EL.EngineRPC(),
			L2JWTPath:     l2EL.JWTPath(),
			L1Genesis:     l1Net.Genesis(),
			L2Genesis:     l2Net.genesis,
			RollupConfig:  l2Net.RollupConfig(),
			FollowSource:  resolvedCfg.FollowSource,
			Config:        *resolvedCfg,
			DependencySet: depSet,
		}
		if unsafeSourceEL != nil {
			launchCtx.UnsafeSourceUserRPC = unsafeSourceEL.UserRPC()
			launchCtx.UnsafeSourceEngineRPC = unsafeSourceEL.EngineRPC()
			launchCtx.UnsafeSourceJWTPath = unsafeSourceEL.JWTPath()
		}
		node, handled := factory.CreateL2CL(t, launchCtx)
		if handled {
			t.Require().NotNil(node, "L2 CL factory handled %s but returned a nil node", target)
			return l2CLStartResult{Node: node, FactoryHandled: true}
		}
		t.Require().Nil(node, "L2 CL factory declined %s but returned a node", target)
		t.Require().NoError(validateDeclinedL2CLFallback(fallbackKind, fallbackSourceFactoryHandled, resolvedCfg.FollowSource),
			"unsupported partial external L2 CL topology for %s", target)
	}
	switch fallbackKind {
	case MixedL2CLKona:
		return l2CLStartResult{Node: startMixedKonaNode(
			t, keys, l1Net, l2Net, l1EL, l1CL, l2EL, clKey, elKey, isSequencer, depSet,
		)}
	default: // op-node
		if resolvedCfg == nil {
			resolvedCfg = resolveL2CLNodeConfig(t, target, startCfg)
		}
		startCfg.ResolvedConfig = resolvedCfg
		return l2CLStartResult{Node: startL2CLNode(
			t, keys, l1Net, l2Net, l1EL, l1CL, l2EL, jwtSecret, startCfg,
		)}
	}
}

func validateDeclinedL2CLFallback(kind MixedL2CLKind, sourceFactoryHandled bool, followSource string) error {
	if kind == MixedL2CLKona && sourceFactoryHandled {
		return fmt.Errorf("Kona fallback cannot follow external source %q; handle the slot or select op-node", followSource)
	}
	return nil
}

func startSequencerEL(t devtest.T, l2Net *L2Network, jwtPath string, jwtSecret [32]byte, identity *ELNodeIdentity, opts ...OpRethOption) L2ELNode {
	return startL2ELForKey(t, l2Net, jwtPath, jwtSecret, "sequencer", identity, opts...)
}

func startL2ELNode(
	t devtest.T,
	l2Net *L2Network,
	jwtPath string,
	jwtSecret [32]byte,
	key string,
	identity *ELNodeIdentity,
) *OpGeth {
	cfg := DefaultL2ELConfig()
	cfg.P2PAddr = "127.0.0.1"
	cfg.P2PPort = identity.Port
	cfg.P2PNodeKeyHex = identity.KeyHex()

	l2EL := &OpGeth{
		name:      key,
		p:         t,
		logger:    t.Logger().New("component", "l2el-"+key),
		l2Net:     l2Net,
		jwtPath:   jwtPath,
		jwtSecret: jwtSecret,
		cfg:       cfg,
	}
	l2EL.Start()
	t.Cleanup(l2EL.Stop)
	return l2EL
}

func connectL2ELPeers(t devtest.T, logger log.Logger, initiatorRPC, acceptorRPC string) {
	require := t.Require()
	rpc1, err := dial.DialRPCClientWithTimeout(t.Ctx(), logger, initiatorRPC)
	require.NoError(err, "failed to connect initiator EL RPC")
	defer rpc1.Close()
	rpc2, err := dial.DialRPCClientWithTimeout(t.Ctx(), logger, acceptorRPC)
	require.NoError(err, "failed to connect acceptor EL RPC")
	defer rpc2.Close()
	ConnectP2P(t.Ctx(), require, rpc1, rpc2)
}

func connectL2CLPeers(t devtest.T, logger log.Logger, l2CL1, l2CL2 L2CLNode) {
	require := t.Require()
	ctx := t.Ctx()

	p := getP2PClientsAndPeers(ctx, logger, require, l2CL1, l2CL2)

	connectPeer := func(p2pClient *sources.P2PClient, multiAddress string) {
		err := retry.Do0(ctx, 6, retry.Exponential(), func() error {
			return p2pClient.ConnectPeer(ctx, multiAddress)
		})
		require.NoError(err, "failed to connect L2CL peer")
	}

	connectPeer(p.client1, p.peerInfo2.Addresses[0])
	connectPeer(p.client2, p.peerInfo1.Addresses[0])

	peerDump1, err := GetPeers(ctx, p.client1)
	require.NoError(err)
	peerDump2, err := GetPeers(ctx, p.client2)
	require.NoError(err)

	_, ok1 := peerDump1.Peers[p.peerInfo2.PeerID.String()]
	require.True(ok1, "peer register invalid (cl1 missing cl2)")
	_, ok2 := peerDump2.Peers[p.peerInfo1.PeerID.String()]
	require.True(ok2, "peer register invalid (cl2 missing cl1)")
}

type l2CLNodeStartConfig struct {
	Key            string
	IsSequencer    bool
	NoDiscovery    bool
	EnableReqResp  bool
	L2FollowSource string
	DependencySet  depset.DependencySet
	L2CLOptions    []L2CLOption
	// ResolvedConfig, when set, is used instead of resolving defaults and
	// options again. This keeps side-effecting options at one application per
	// slot across factory selection and stock fallback.
	ResolvedConfig *L2CLConfig
	// SyncMode overrides the sequencer and verifier sync modes; defaults to CLSync if unset.
	SyncMode nodeSync.Mode
	// SequencerStopped starts the sequencer in the stopped state (it must be
	// activated later via the StartSequencer RPC). Only meaningful when IsSequencer.
	SequencerStopped bool
}

func resolveL2CLNodeConfig(t devtest.T, target ComponentTarget, startCfg l2CLNodeStartConfig) *L2CLConfig {
	cfg := DefaultL2CLConfig()
	cfg.IsSequencer = startCfg.IsSequencer
	cfg.NoDiscovery = startCfg.NoDiscovery
	cfg.EnableReqRespSync = startCfg.EnableReqResp
	cfg.FollowSource = startCfg.L2FollowSource
	for _, opt := range startCfg.L2CLOptions {
		if opt != nil {
			opt.Apply(t, target, cfg)
		}
	}
	if startCfg.SyncMode != 0 {
		cfg.SequencerSyncMode = startCfg.SyncMode
		cfg.VerifierSyncMode = startCfg.SyncMode
	}
	cfg.SequencerStopped = startCfg.SequencerStopped || cfg.SequencerStopped
	return cfg
}

func startL2CLNode(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	jwtSecret [32]byte,
	startCfg l2CLNodeStartConfig,
) *OpNode {
	require := t.Require()
	cfg := startCfg.ResolvedConfig
	if cfg == nil {
		cfg = resolveL2CLNodeConfig(t, NewComponentTarget(startCfg.Key, l2Net.ChainID()), startCfg)
	}

	syncMode := cfg.VerifierSyncMode
	if cfg.IsSequencer {
		syncMode = cfg.SequencerSyncMode
	}

	logger := t.Logger().New("component", "l2cl-"+startCfg.Key)

	// Build P2P config through the same path as sysgo op-node setup.
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range opNodeFlags.P2PFlags(opNodeFlags.EnvVarPrefix) {
		require.NoError(f.Apply(fs))
	}
	require.NoError(fs.Set(opNodeFlags.AdvertiseIPName, "127.0.0.1"))
	require.NoError(fs.Set(opNodeFlags.AdvertiseTCPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.AdvertiseUDPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.ListenIPName, "127.0.0.1"))
	require.NoError(fs.Set(opNodeFlags.ListenTCPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.ListenUDPPortName, "0"))
	require.NoError(fs.Set(opNodeFlags.DiscoveryPathName, "memory"))
	require.NoError(fs.Set(opNodeFlags.PeerstorePathName, "memory"))
	require.NoError(fs.Set(opNodeFlags.BootnodesName, ""))

	networkPrivKey, err := crypto.GenerateKey()
	require.NoError(err)
	networkPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(networkPrivKey))
	require.NoError(fs.Set(opNodeFlags.P2PPrivRawName, networkPrivKeyHex))

	cliCtx := cli.NewContext(&cli.App{}, fs, nil)
	var p2pSignerSetup p2p.SignerSetup
	if cfg.IsSequencer {
		p2pKey, err := keys.Secret(devkeys.SequencerP2PRole.Key(l2Net.ChainID().ToBig()))
		require.NoError(err, "need p2p key for sequencer")
		p2pKeyHex := hex.EncodeToString(crypto.FromECDSA(p2pKey))
		require.NoError(fs.Set(opNodeFlags.SequencerP2PKeyName, p2pKeyHex))
		p2pSignerSetup, err = p2pcli.LoadSignerSetup(cliCtx, logger)
		require.NoError(err, "failed to load p2p signer")
	}
	p2pConfig, err := p2pcli.NewConfig(cliCtx, l2Net.rollupCfg.BlockTime)
	require.NoError(err, "failed to load p2p config")
	p2pConfig.NoDiscovery = cfg.NoDiscovery
	p2pConfig.EnableReqRespSync = cfg.EnableReqRespSync
	// Devstack chain timestamps are synthetic: genesis is set in the past and the
	// chain may lag many seconds behind wallclock during startup (or many minutes
	// during long tests like dispute games). The production-default 60s gossip
	// "too old" check then rejects otherwise-valid TestSequencer-produced blocks
	// — surfacing as "validation failed" out of ts.Next at startup. Match the
	// multichain devstack (see newDevstackP2PConfig) by loosening to 1 hour.
	p2pConfig.GossipTimestampThreshold = time.Hour

	nodeCfg := &config.Config{
		L1: &config.L1EndpointConfig{
			L1NodeAddr:       l1EL.UserRPC(),
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindDebugGeth,
			RateLimit:        0,
			BatchSize:        20,
			HttpPollInterval: 100 * time.Millisecond,
			MaxConcurrency:   10,
			CacheSize:        0,
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
			SequencerEnabled:    cfg.IsSequencer,
			SequencerStopped:    cfg.SequencerStopped,
			SequencerConfDepth:  2,
			SequencerMaxSafeLag: cfg.SequencerMaxSafeLag,
		},
		Rollup:        *l2Net.rollupCfg,
		DependencySet: startCfg.DependencySet,
		P2PSigner:     p2pSignerSetup,
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P:                         p2pConfig,
		L1EpochPollInterval:         time.Second * 2,
		RuntimeConfigReloadInterval: 0,
		Tracer:                      nil,
		Sync: nodeSync.Config{
			SyncMode:                       syncMode,
			SkipSyncStartCheck:             false,
			SupportsPostFinalizationELSync: false,
			L2FollowSourceEndpoint:         cfg.FollowSource,
			// Mirror op-node/service.go: a follow-mode sequencer needs a single
			// initial engine reset to trigger block building (TryInitialResetEngineForSequencer).
			NeedInitialResetEngine: cfg.IsSequencer && cfg.FollowSource != "",
			OffsetELSafe:           cfg.OffsetELSafe,
		},
		ConfigPersistence:               config.DisabledConfigPersistence{},
		Metrics:                         opmetrics.CLIConfig{},
		Pprof:                           oppprof.CLIConfig{},
		SafeDBPath:                      cfg.SafeDBPath,
		Cancel:                          nil,
		ConductorEnabled:                false,
		ConductorRpc:                    nil,
		ConductorRpcTimeout:             0,
		AltDA:                           altda.CLIConfig{},
		IgnoreMissingPectraBlobSchedule: false,
		ExperimentalOPStackAPI:          true,
	}
	l2CL := &OpNode{
		name:     startCfg.Key,
		opNode:   nil,
		cfg:      nodeCfg,
		syncMode: syncMode,
		p:        t,
		logger:   logger,
		clock:    clock.SystemClock,
	}
	l2CL.Start()
	t.Cleanup(l2CL.Stop)
	return l2CL
}

func copyControlRPCMap(in map[eth.ChainID]string) map[eth.ChainID]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[eth.ChainID]string, len(in))
	for chainID, endpoint := range in {
		out[chainID] = endpoint
	}
	return out
}
