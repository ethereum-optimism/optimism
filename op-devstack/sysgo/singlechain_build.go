package sysgo

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	faucetConfig "github.com/ethereum-optimism/optimism/op-faucet/config"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-node/config"
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	sequencerConfig "github.com/ethereum-optimism/optimism/op-test-sequencer/config"
	testmetrics "github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/fakepos"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/standardbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/standardcommitter"
	workconfig "github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/config"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/standardpublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/sequencers/fullseq"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/localkey"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type testSequencer struct {
	name       string
	adminRPC   string
	jwtSecret  [32]byte
	controlRPC map[eth.ChainID]string
	service    *sequencer.Service
}

func buildSingleChainWorld(t devtest.T, keys devkeys.Keys, deployerOpts ...DeployerOption) (*L1Network, *L2Network) {
	wb := &worldBuilder{
		p:       t,
		logger:  t.Logger(),
		require: t.Require(),
		keys:    keys,
		builder: intentbuilder.New(),
	}

	applyConfigLocalContractSources(t, keys, wb.builder)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 1, "expected exactly one L2 chain in flashblocks world")
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
		opcmImpl:   wb.output.ImplementationsDeployment.OpcmImpl,
		mipsImpl:   wb.output.ImplementationsDeployment.MipsImpl,
		keys:       keys,
	}
	return l1Net, l2Net
}

func applyConfigLocalContractSources(t devtest.T, _ devkeys.Keys, builder intentbuilder.Builder) {
	paths, err := contractPaths()
	t.Require().NoError(err)
	wd, err := os.Getwd()
	t.Require().NoError(err)
	artifactsPath := filepath.Join(wd, paths.FoundryArtifacts)
	t.Require().NoError(ensureDir(artifactsPath))
	contractArtifacts, err := artifacts.NewFileLocator(artifactsPath)
	t.Require().NoError(err)
	builder.WithL1ContractsLocator(contractArtifacts)
	builder.WithL2ContractsLocator(contractArtifacts)
}

func applyConfigCommons(t devtest.T, keys devkeys.Keys, l1ChainID eth.ChainID, builder intentbuilder.Builder) {
	_, l1Config := builder.WithL1(l1ChainID)

	l1StartTimestamp := uint64(time.Now().Unix()) + 1
	l1Config.WithTimestamp(l1StartTimestamp)
	l1Config.WithL1ForkAtGenesis(forks.Prague)

	faucetFunderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
	t.Require().NoError(err, "need funder addr")
	l1Config.WithPrefundedAccount(faucetFunderAddr, *eth.BillionEther.ToU256())

	addrFor := intentbuilder.RoleToAddrProvider(t, keys, l1ChainID)
	_, superCfg := builder.WithSuperchain()
	intentbuilder.WithDevkeySuperRoles(t, keys, l1ChainID, superCfg)
	l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainProxyAdminOwner), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainProtocolVersionsOwner), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.SuperchainConfigGuardianKey), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.L1ProxyAdminOwnerRole), *millionEth)
}

func applyConfigPrefundedL2(t devtest.T, keys devkeys.Keys, l1ChainID, l2ChainID eth.ChainID, builder intentbuilder.Builder) {
	_, l2Config := builder.WithL2(l2ChainID)
	intentbuilder.WithDevkeyVaults(t, keys, l2Config)
	intentbuilder.WithDevkeyL2Roles(t, keys, l2Config)
	intentbuilder.WithDevkeyL1Roles(t, keys, l2Config, l1ChainID)

	faucetFunderAddr, err := keys.Address(devkeys.UserKey(funderMnemonicIndex))
	t.Require().NoError(err, "need funder addr")
	l2Config.WithPrefundedAccount(faucetFunderAddr, *eth.BillionEther.ToU256())

	addrFor := intentbuilder.RoleToAddrProvider(t, keys, l2ChainID)
	l1Config := l2Config.L1Config()
	l1Config.WithPrefundedAccount(addrFor(devkeys.BatcherRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.ProposerRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.ChallengerRole), *millionEth)
	l1Config.WithPrefundedAccount(addrFor(devkeys.SystemConfigOwner), *millionEth)
}

func startSequencerEL(t devtest.T, l2Net *L2Network, jwtPath string, jwtSecret [32]byte, identity *ELNodeIdentity) *OpGeth {
	return startL2ELNode(t, l2Net, jwtPath, jwtSecret, "sequencer", identity)
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

func connectL2ELPeers(t devtest.T, logger log.Logger, initiatorRPC, acceptorRPC string, trusted bool) {
	require := t.Require()
	rpc1, err := dial.DialRPCClientWithTimeout(t.Ctx(), logger, initiatorRPC)
	require.NoError(err, "failed to connect initiator EL RPC")
	defer rpc1.Close()
	rpc2, err := dial.DialRPCClientWithTimeout(t.Ctx(), logger, acceptorRPC)
	require.NoError(err, "failed to connect acceptor EL RPC")
	defer rpc2.Close()
	ConnectP2P(t.Ctx(), require, rpc1, rpc2, trusted)
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

func startSequencerCL(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	jwtSecret [32]byte,
	l2CLOpts []L2CLOption,
) *OpNode {
	return startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   false,
		L2FollowSource: "",
		L2CLOptions:    l2CLOpts,
	})
}

type l2CLNodeStartConfig struct {
	Key            string
	IsSequencer    bool
	NoDiscovery    bool
	EnableReqResp  bool
	UseReqResp     bool
	IndexingMode   bool
	L2FollowSource string
	DependencySet  depset.DependencySet
	L2CLOptions    []L2CLOption
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
	cfg := DefaultL2CLConfig()
	cfg.IsSequencer = startCfg.IsSequencer
	cfg.NoDiscovery = startCfg.NoDiscovery
	cfg.EnableReqRespSync = startCfg.EnableReqResp
	cfg.UseReqRespSync = startCfg.UseReqResp
	cfg.IndexingMode = startCfg.IndexingMode
	cfg.FollowSource = startCfg.L2FollowSource
	if len(startCfg.L2CLOptions) > 0 {
		l2CLTarget := NewComponentTarget(startCfg.Key, l2Net.ChainID())
		for _, opt := range startCfg.L2CLOptions {
			if opt == nil {
				continue
			}
			opt.Apply(t, l2CLTarget, cfg)
		}
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

	interopCfg := &interop.Config{}
	if startCfg.IndexingMode {
		interopCfg = &interop.Config{
			RPCAddr:          "127.0.0.1",
			RPCPort:          0,
			RPCJwtSecretPath: l2EL.JWTPath(),
		}
	}

	nodeCfg := &config.Config{
		L1: &config.L1EndpointConfig{
			L1NodeAddr:       l1EL.UserRPC(),
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindDebugGeth,
			RateLimit:        0,
			BatchSize:        20,
			HttpPollInterval: 100,
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
			SequencerEnabled:   cfg.IsSequencer,
			SequencerConfDepth: 2,
		},
		Rollup:            *l2Net.rollupCfg,
		DependencySet:     startCfg.DependencySet,
		SupervisorEnabled: cfg.IndexingMode,
		P2PSigner:         p2pSignerSetup,
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
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
			NeedInitialResetEngine:         false,
		},
		ConfigPersistence:               config.DisabledConfigPersistence{},
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
	l2CL := &OpNode{
		name:   startCfg.Key,
		opNode: nil,
		cfg:    nodeCfg,
		p:      t,
		logger: logger,
		clock:  clock.SystemClock,
	}
	l2CL.Start()
	t.Cleanup(l2CL.Stop)
	return l2CL
}

func startTestSequencer(
	t devtest.T,
	keys devkeys.Keys,
	jwtPath string,
	jwtSecret [32]byte,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	l2EL *OpGeth,
	l2CL *OpNode,
) *testSequencer {
	require := t.Require()
	logger := t.Logger().New("component", "test-sequencer")

	l1ELClient, err := ethclient.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err, "failed to dial L1 EL RPC for test-sequencer")
	t.Cleanup(l1ELClient.Close)

	engineCl, err := dialEngine(t.Ctx(), l1EL.AuthRPC(), jwtSecret)
	require.NoError(err, "failed to dial L1 engine API for test-sequencer")
	t.Cleanup(func() {
		engineCl.inner.Close()
	})

	l1ChainID := l1Net.ChainID()
	l2ChainID := l2EL.l2Net.ChainID()

	// L1 sequencer components: fakepos builder + noop signer/committer/publisher.
	bidL1 := seqtypes.BuilderID("test-l1-builder")
	cidL1 := seqtypes.CommitterID("test-noop-committer")
	sidL1 := seqtypes.SignerID("test-noop-signer")
	pidL1 := seqtypes.PublisherID("test-noop-publisher")
	seqIDL1 := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l1ChainID))

	ensemble := &workconfig.Ensemble{
		Builders: map[seqtypes.BuilderID]*workconfig.BuilderEntry{
			bidL1: {
				L1: &fakepos.Config{
					ChainConfig:       l1Net.genesis.Config,
					EngineAPI:         engineCl,
					Backend:           l1ELClient,
					Beacon:            l1CL.beacon,
					FinalizedDistance: 20,
					SafeDistance:      10,
					BlockTime:         6,
				},
			},
		},
		Signers: map[seqtypes.SignerID]*workconfig.SignerEntry{
			sidL1: {
				Noop: &noopsigner.Config{},
			},
		},
		Committers: map[seqtypes.CommitterID]*workconfig.CommitterEntry{
			cidL1: {
				Noop: &noopcommitter.Config{},
			},
		},
		Publishers: map[seqtypes.PublisherID]*workconfig.PublisherEntry{
			pidL1: {
				Noop: &nooppublisher.Config{},
			},
		},
		Sequencers: map[seqtypes.SequencerID]*workconfig.SequencerEntry{
			seqIDL1: {
				Full: &fullseq.Config{
					ChainID:   l1ChainID,
					Builder:   bidL1,
					Signer:    sidL1,
					Committer: cidL1,
					Publisher: pidL1,
				},
			},
		},
	}

	// L2 sequencer components: standard builder/committer/publisher + local signer.
	bidL2 := seqtypes.BuilderID("test-standard-builder")
	cidL2 := seqtypes.CommitterID("test-standard-committer")
	sidL2 := seqtypes.SignerID("test-local-signer")
	pidL2 := seqtypes.PublisherID("test-standard-publisher")
	seqIDL2 := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l2ChainID))

	p2pKey, err := keys.Secret(devkeys.SequencerP2PRole.Key(l2ChainID.ToBig()))
	require.NoError(err, "need p2p key for test sequencer")
	rawKey := hexutil.Bytes(crypto.FromECDSA(p2pKey))

	ensemble.Builders[bidL2] = &workconfig.BuilderEntry{
		Standard: &standardbuilder.Config{
			L1ChainConfig: l1Net.genesis.Config,
			L1EL: endpoint.MustRPC{
				Value: endpoint.HttpURL(l1EL.UserRPC()),
			},
			L2EL: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2EL.UserRPC()),
			},
			L2CL: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2CL.UserRPC()),
			},
		},
	}
	ensemble.Signers[sidL2] = &workconfig.SignerEntry{
		LocalKey: &localkey.Config{
			RawKey:  &rawKey,
			ChainID: l2ChainID,
		},
	}
	ensemble.Committers[cidL2] = &workconfig.CommitterEntry{
		Standard: &standardcommitter.Config{
			RPC: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2CL.UserRPC()),
			},
		},
	}
	ensemble.Publishers[pidL2] = &workconfig.PublisherEntry{
		Standard: &standardpublisher.Config{
			RPC: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2CL.UserRPC()),
			},
		},
	}
	ensemble.Sequencers[seqIDL2] = &workconfig.SequencerEntry{
		Full: &fullseq.Config{
			ChainID:             l2ChainID,
			Builder:             bidL2,
			Signer:              sidL2,
			Committer:           cidL2,
			Publisher:           pidL2,
			SequencerConfDepth:  2,
			SequencerEnabled:    true,
			SequencerStopped:    false,
			SequencerMaxSafeLag: 0,
		},
	}

	sequencerIDs := map[eth.ChainID]seqtypes.SequencerID{
		l1ChainID: seqIDL1,
		l2ChainID: seqIDL2,
	}

	jobs := work.NewJobRegistry()
	startedEnsemble, err := ensemble.Start(t.Ctx(), &work.StartOpts{
		Log:     logger,
		Metrics: &testmetrics.NoopMetrics{},
		Jobs:    jobs,
	})
	require.NoError(err, "failed to start test-sequencer ensemble")

	cfg := &sequencerConfig.Config{
		MetricsConfig: opmetrics.CLIConfig{
			Enabled: false,
		},
		PprofConfig: oppprof.CLIConfig{
			ListenEnabled: false,
		},
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelDebug,
			Format: oplog.FormatText,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		Ensemble:      startedEnsemble,
		JWTSecretPath: jwtPath,
		Version:       "dev",
		MockRun:       false,
	}

	sq, err := sequencer.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "failed to initialize test-sequencer service")
	require.NoError(sq.Start(t.Ctx()), "failed to start test-sequencer service")

	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing test-sequencer service")
		closeErr := sq.Stop(ctx)
		logger.Info("Closed test-sequencer service", "err", closeErr)
	})

	adminRPC := sq.RPC()
	controlRPCs := make(map[eth.ChainID]string, len(sequencerIDs))
	for chainID, seqID := range sequencerIDs {
		controlRPCs[chainID] = adminRPC + "/sequencers/" + seqID.String()
	}

	return &testSequencer{
		name:       "test-sequencer",
		adminRPC:   adminRPC,
		jwtSecret:  jwtSecret,
		controlRPC: controlRPCs,
		service:    sq,
	}
}

func startFaucets(
	t devtest.T,
	keys devkeys.Keys,
	l1ChainID eth.ChainID,
	l2ChainID eth.ChainID,
	l1ELRPC string,
	l2ELRPC string,
) *faucet.Service {
	require := t.Require()
	logger := t.Logger().New("component", "faucet")

	funderKey, err := keys.Secret(devkeys.UserKey(funderMnemonicIndex))
	require.NoError(err, "need faucet funder key")
	funderKeyStr := hexutil.Encode(crypto.FromECDSA(funderKey))

	faucets := map[ftypes.FaucetID]*fconf.FaucetEntry{
		ftypes.FaucetID(fmt.Sprintf("dev-faucet-%s", l1ChainID)): {
			ELRPC:   endpoint.MustRPC{Value: endpoint.URL(l1ELRPC)},
			ChainID: l1ChainID,
			TxCfg: fconf.TxManagerConfig{
				PrivateKey: funderKeyStr,
			},
		},
		ftypes.FaucetID(fmt.Sprintf("dev-faucet-%s", l2ChainID)): {
			ELRPC:   endpoint.MustRPC{Value: endpoint.URL(l2ELRPC)},
			ChainID: l2ChainID,
			TxCfg: fconf.TxManagerConfig{
				PrivateKey: funderKeyStr,
			},
		},
	}

	cfg := &faucetConfig.Config{
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		Faucets: &fconf.Config{
			Faucets: faucets,
		},
	}

	srv, err := faucet.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "failed to create faucet service")
	require.NoError(srv.Start(t.Ctx()), "failed to start faucet service")

	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // force-close
		logger.Info("Closing faucet service")
		closeErr := srv.Stop(ctx)
		logger.Info("Closed faucet service", "err", closeErr)
	})

	return srv
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
