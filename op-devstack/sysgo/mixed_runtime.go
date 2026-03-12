package sysgo

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
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

type MixedL2ELKind string

const (
	MixedL2ELOpGeth MixedL2ELKind = "op-geth"
	MixedL2ELOpReth MixedL2ELKind = "op-reth"
)

type MixedL2CLKind string

const (
	MixedL2CLOpNode MixedL2CLKind = "op-node"
	MixedL2CLKona   MixedL2CLKind = "kona-node"
)

type MixedSingleChainNodeSpec struct {
	ELKey          string
	CLKey          string
	ELKind         MixedL2ELKind
	ELProofHistory bool
	CLKind         MixedL2CLKind
	IsSequencer    bool
}

type MixedSingleChainPresetConfig struct {
	NodeSpecs         []MixedSingleChainNodeSpec
	WithTestSequencer bool
	TestSequencerName string
	DeployerOptions   []DeployerOption
}

type mixedSingleChainNode struct {
	spec MixedSingleChainNodeSpec
	el   L2ELNode
	cl   L2CLNode
}

type MixedSingleChainRuntime struct {
	L1Network     *L1Network
	L1EL          *L1Geth
	L1CL          *L1CLNode
	L2Network     *L2Network
	Nodes         []MixedSingleChainNodeRefs
	L2Batcher     *L2Batcher
	FaucetService *faucet.Service
	TestSequencer *TestSequencerRuntime
}

type MixedSingleChainNodeRefs struct {
	Spec MixedSingleChainNodeSpec
	EL   L2ELNode
	CL   L2CLNode
}

type mixedNoopMetricsRegistrar struct{}

func (mixedNoopMetricsRegistrar) RegisterL2MetricsTargets(_ string, _ ...PrometheusMetricsTarget) {
}

func NewMixedSingleChainRuntime(t devtest.T, cfg MixedSingleChainPresetConfig) *MixedSingleChainRuntime {
	require := t.Require()
	require.NotEmpty(cfg.NodeSpecs, "mixed runtime requires at least one L2 node spec")

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	l1Net, l2Net := buildSingleChainWorld(t, keys, cfg.DeployerOptions...)
	jwtPath, jwtSecret := writeJWTSecret(t)
	l1EL, l1CL := startInProcessL1(t, l1Net, jwtPath)

	metricsRegistrar := mixedNoopMetricsRegistrar{}

	nodes := make([]mixedSingleChainNode, 0, len(cfg.NodeSpecs))
	for _, spec := range cfg.NodeSpecs {
		identity := NewELNodeIdentity(0)

		var el L2ELNode
		switch spec.ELKind {
		case MixedL2ELOpGeth:
			el = startL2ELNode(t, l2Net, jwtPath, jwtSecret, spec.ELKey, identity)
		case MixedL2ELOpReth:
			el = startMixedOpRethNode(t, l2Net, spec.ELKey, jwtPath, jwtSecret, spec.ELProofHistory, metricsRegistrar)
		default:
			require.FailNowf("unsupported EL kind", "unsupported mixed EL kind %q", spec.ELKind)
		}

		var cl L2CLNode
		switch spec.CLKind {
		case MixedL2CLOpNode:
			cl = startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, el, jwtSecret, l2CLNodeStartConfig{
				Key:           spec.CLKey,
				IsSequencer:   spec.IsSequencer,
				NoDiscovery:   true,
				EnableReqResp: true,
				UseReqResp:    true,
			})
		case MixedL2CLKona:
			cl = startMixedKonaNode(
				t,
				keys,
				l1Net,
				l2Net,
				l1EL,
				l1CL,
				el,
				spec.CLKey,
				spec.ELKey,
				spec.IsSequencer,
				metricsRegistrar,
			)
		default:
			require.FailNowf("unsupported CL kind", "unsupported mixed CL kind %q", spec.CLKind)
		}

		nodes = append(nodes, mixedSingleChainNode{
			spec: spec,
			el:   el,
			cl:   cl,
		})
	}

	for i := range nodes {
		for j := 0; j < i; j++ {
			connectL2CLPeers(t, t.Logger(), nodes[i].cl, nodes[j].cl)
			connectL2ELPeers(t, t.Logger(), nodes[i].el.UserRPC(), nodes[j].el.UserRPC(), false)
		}
	}

	var sequencerNode *mixedSingleChainNode
	for i := range nodes {
		if nodes[i].spec.IsSequencer {
			sequencerNode = &nodes[i]
			break
		}
	}
	require.NotNil(sequencerNode, "mixed runtime requires at least one sequencer node")

	l2Batcher := startMinimalBatcher(t, keys, l2Net, l1EL, sequencerNode.cl, sequencerNode.el)
	faucetService := startFaucets(t, keys, l1Net.ChainID(), l2Net.ChainID(), l1EL.UserRPC(), sequencerNode.el.UserRPC())

	var testSequencer *testSequencer
	if cfg.WithTestSequencer {
		testSequencerName := cfg.TestSequencerName
		if testSequencerName == "" {
			testSequencerName = "test-sequencer"
		}
		testSequencer = startTestSequencerForRPCs(
			t,
			keys,
			testSequencerName,
			jwtPath,
			jwtSecret,
			l1Net,
			l1EL,
			l1CL,
			l2Net.ChainID(),
			sequencerNode.el.UserRPC(),
			sequencerNode.cl.UserRPC(),
		)
	}

	return &MixedSingleChainRuntime{
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		L2Network:     l2Net,
		Nodes:         mixedNodeRefs(nodes),
		L2Batcher:     l2Batcher,
		FaucetService: faucetService,
		TestSequencer: newTestSequencerRuntime(testSequencer, cfg.TestSequencerName),
	}
}

func mixedNodeRefs(nodes []mixedSingleChainNode) []MixedSingleChainNodeRefs {
	out := make([]MixedSingleChainNodeRefs, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, MixedSingleChainNodeRefs{
			Spec: node.spec,
			EL:   node.el,
			CL:   node.cl,
		})
	}
	return out
}

func startMixedOpRethNode(
	t devtest.T,
	l2Net *L2Network,
	key string,
	jwtPath string,
	jwtSecret [32]byte,
	proofHistory bool,
	metricsRegistrar L2MetricsRegistrar,
) *OpReth {
	tempDir := t.TempDir()

	data, err := json.Marshal(l2Net.genesis)
	t.Require().NoError(err, "must json-encode genesis")
	chainConfigPath := filepath.Join(tempDir, "genesis.json")
	t.Require().NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write genesis file")

	dataDirPath := filepath.Join(tempDir, "data")
	t.Require().NoError(os.MkdirAll(dataDirPath, 0o755), "must create datadir")

	logDirPath := filepath.Join(tempDir, "logs")
	t.Require().NoError(os.MkdirAll(logDirPath, 0o755), "must create logs dir")

	tempP2PPath := filepath.Join(tempDir, "p2pkey.txt")

	execPath := os.Getenv("OP_RETH_EXEC_PATH")
	t.Require().NotEmpty(execPath, "OP_RETH_EXEC_PATH environment variable must be set")
	_, err = os.Stat(execPath)
	t.Require().NotErrorIs(err, os.ErrNotExist, "executable must exist")

	args := []string{
		"node",
		"--addr=127.0.0.1",
		"--authrpc.addr=127.0.0.1",
		"--authrpc.jwtsecret=" + jwtPath,
		"--authrpc.port=0",
		"--builder.deadline=2",
		"--builder.interval=100ms",
		"--chain=" + chainConfigPath,
		"--color=never",
		"--datadir=" + dataDirPath,
		"--disable-discovery",
		"--http",
		"--http.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
		"--http.addr=127.0.0.1",
		"--http.port=0",
		"--ipcdisable",
		"--log.file.directory=" + logDirPath,
		"--log.stdout.format=json",
		"--nat=none",
		"--p2p-secret-key=" + tempP2PPath,
		"--port=0",
		"--rpc.eth-proof-window=30",
		"--txpool.minimum-priority-fee=1",
		"--txpool.nolocals",
		"--with-unused-ports",
		"--ws",
		"--ws.api=admin,debug,eth,net,trace,txpool,web3,rpc,reth,miner",
		"--ws.addr=127.0.0.1",
		"--ws.port=0",
		"-vvvv",
	}

	if areMetricsEnabled() {
		args = append(args, "--metrics=127.0.0.1:0")
	}

	initArgs := []string{
		"init",
		"--datadir=" + dataDirPath,
		"--chain=" + chainConfigPath,
	}
	err = exec.Command(execPath, initArgs...).Run()
	t.Require().NoError(err, "must init op-reth node")

	if proofHistory {
		proofHistoryDir := filepath.Join(tempDir, "proof-history")

		initProofsArgs := []string{
			"proofs",
			"init",
			"--datadir=" + dataDirPath,
			"--chain=" + chainConfigPath,
			"--proofs-history.storage-path=" + proofHistoryDir,
		}
		err = exec.Command(execPath, initProofsArgs...).Run()
		t.Require().NoError(err, "must init op-reth proof history")

		args = append(
			args,
			"--proofs-history",
			"--proofs-history.window=200",
			"--proofs-history.prune-interval=1m",
			"--proofs-history.storage-path="+proofHistoryDir,
		)
	}

	l2EL := &OpReth{
		name:               key,
		chainID:            l2Net.ChainID(),
		jwtPath:            jwtPath,
		jwtSecret:          jwtSecret,
		authRPC:            "",
		userRPC:            "",
		execPath:           execPath,
		args:               args,
		env:                []string{},
		p:                  t,
		l2MetricsRegistrar: metricsRegistrar,
	}

	t.Logger().Info("Starting op-reth", "name", key, "chain", l2Net.ChainID())
	l2EL.Start()
	t.Cleanup(l2EL.Stop)
	t.Logger().Info("op-reth is ready", "name", key, "chain", l2Net.ChainID(), "userRPC", l2EL.userRPC, "authRPC", l2EL.authRPC)
	return l2EL
}

func startMixedKonaNode(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	clKey string,
	elKey string,
	isSequencer bool,
	metricsRegistrar L2MetricsRegistrar,
) *KonaNode {
	tempKonaDir := t.TempDir()

	tempP2PPath := filepath.Join(tempKonaDir, "p2pkey.txt")

	tempRollupCfgPath := filepath.Join(tempKonaDir, "rollup.json")
	rollupCfgData, err := json.Marshal(l2Net.rollupCfg)
	t.Require().NoError(err, "must write rollup config")
	t.Require().NoError(os.WriteFile(tempRollupCfgPath, rollupCfgData, 0o644))

	tempL1CfgPath := filepath.Join(tempKonaDir, "l1-chain-config.json")
	l1CfgData, err := json.Marshal(l1Net.genesis.Config)
	t.Require().NoError(err, "must write l1 chain config")
	t.Require().NoError(os.WriteFile(tempL1CfgPath, l1CfgData, 0o644))

	envVars := []string{
		"KONA_NODE_L1_ETH_RPC=" + l1EL.UserRPC(),
		"KONA_NODE_L1_BEACON=" + l1CL.beaconHTTPAddr,
		"KONA_NODE_L2_ENGINE_RPC=" + strings.ReplaceAll(l2EL.EngineRPC(), "ws://", "http://"),
		"KONA_NODE_L2_ENGINE_AUTH=" + l2EL.JWTPath(),
		"KONA_NODE_ROLLUP_CONFIG=" + tempRollupCfgPath,
		"KONA_NODE_L1_CHAIN_CONFIG=" + tempL1CfgPath,
		"KONA_NODE_P2P_PRIV_PATH=" + tempP2PPath,
		propagateEnvVarOrDefault("KONA_NODE_P2P_NO_DISCOVERY", "true"),
		propagateEnvVarOrDefault("KONA_NODE_RPC_ADDR", "127.0.0.1"),
		propagateEnvVarOrDefault("KONA_NODE_RPC_PORT", "0"),
		propagateEnvVarOrDefault("KONA_NODE_RPC_WS_ENABLED", "true"),
		propagateEnvVarOrDefault("KONA_METRICS_ADDR", ""),
		propagateEnvVarOrDefault("KONA_LOG_LEVEL", "3"),
		propagateEnvVarOrDefault("KONA_LOG_STDOUT_FORMAT", "json"),
		propagateEnvVarOrDefault("KONA_NODE_P2P_LISTEN_IP", "127.0.0.1"),
		propagateEnvVarOrDefault("KONA_NODE_P2P_LISTEN_TCP_PORT", "0"),
		propagateEnvVarOrDefault("KONA_NODE_P2P_LISTEN_UDP_PORT", "0"),
	}

	if areMetricsEnabled() {
		metricsPort, err := getAvailableLocalPort()
		t.Require().NoError(err, "startMixedKonaNode: getting metrics port")
		envVars = append(envVars, propagateEnvVarOrDefault("KONA_METRICS_PORT", metricsPort))
		envVars = append(envVars, "KONA_METRICS_ENABLED=true")
	}

	if isSequencer {
		p2pKey, err := keys.Secret(devkeys.SequencerP2PRole.Key(l2Net.ChainID().ToBig()))
		t.Require().NoError(err, "need p2p key for sequencer")
		p2pKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSA(p2pKey))
		tempSeqKeyPath := filepath.Join(tempKonaDir, "p2p-sequencer.txt")
		t.Require().NoError(os.WriteFile(tempSeqKeyPath, []byte(p2pKeyHex), 0o644))
		envVars = append(envVars,
			"KONA_NODE_P2P_SEQUENCER_KEY_PATH="+tempSeqKeyPath,
			"KONA_NODE_SEQUENCER_L1_CONFS=2",
			"KONA_NODE_MODE=Sequencer",
		)
	} else {
		envVars = append(envVars, "KONA_NODE_MODE=Validator")
	}

	execPath, err := EnsureRustBinary(t, RustBinarySpec{
		SrcDir:  "rust/kona",
		Package: "kona-node",
		Binary:  "kona-node",
	})
	t.Require().NoError(err, "prepare kona-node binary")
	t.Require().NotEmpty(execPath, "kona-node binary path resolved")

	k := &KonaNode{
		name:               clKey,
		chainID:            l2Net.ChainID(),
		userRPC:            "",
		interopEndpoint:    "",
		interopJwtSecret:   eth.Bytes32{},
		execPath:           execPath,
		args:               []string{"node"},
		env:                envVars,
		p:                  t,
		l2MetricsRegistrar: metricsRegistrar,
	}
	t.Logger().Info("Starting kona-node", "name", clKey, "chain", l2Net.ChainID(), "el", elKey)
	k.Start()
	t.Cleanup(k.Stop)
	t.Logger().Info("Kona-node is up", "name", clKey, "chain", l2Net.ChainID(), "rpc", k.UserRPC())
	return k
}

func startTestSequencerForRPCs(
	t devtest.T,
	keys devkeys.Keys,
	testSequencerName string,
	jwtPath string,
	jwtSecret [32]byte,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	l2ChainID eth.ChainID,
	l2ELRPC string,
	l2CLRPC string,
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

	bidL1 := seqtypes.BuilderID("test-l1-builder")
	cidL1 := seqtypes.CommitterID("test-noop-committer")
	sidL1 := seqtypes.SignerID("test-noop-signer")
	pidL1 := seqtypes.PublisherID("test-noop-publisher")
	seqIDL1 := seqtypes.SequencerID("test-seq-" + l1ChainID.String())

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
			sidL1: {Noop: &noopsigner.Config{}},
		},
		Committers: map[seqtypes.CommitterID]*workconfig.CommitterEntry{
			cidL1: {Noop: &noopcommitter.Config{}},
		},
		Publishers: map[seqtypes.PublisherID]*workconfig.PublisherEntry{
			pidL1: {Noop: &nooppublisher.Config{}},
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

	bidL2 := seqtypes.BuilderID("test-standard-builder")
	cidL2 := seqtypes.CommitterID("test-standard-committer")
	sidL2 := seqtypes.SignerID("test-local-signer")
	pidL2 := seqtypes.PublisherID("test-standard-publisher")
	seqIDL2 := seqtypes.SequencerID("test-seq-" + l2ChainID.String())

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
				Value: endpoint.HttpURL(l2ELRPC),
			},
			L2CL: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2CLRPC),
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
				Value: endpoint.HttpURL(l2CLRPC),
			},
		},
	}
	ensemble.Publishers[pidL2] = &workconfig.PublisherEntry{
		Standard: &standardpublisher.Config{
			RPC: endpoint.MustRPC{
				Value: endpoint.HttpURL(l2CLRPC),
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
	controlRPCs := map[eth.ChainID]string{
		l1ChainID: adminRPC + "/sequencers/" + seqIDL1.String(),
		l2ChainID: adminRPC + "/sequencers/" + seqIDL2.String(),
	}

	return &testSequencer{
		name:       testSequencerName,
		adminRPC:   adminRPC,
		jwtSecret:  jwtSecret,
		controlRPC: controlRPCs,
		service:    sq,
	}
}
