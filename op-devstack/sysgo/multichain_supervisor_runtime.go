package sysgo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	supervisorConfig "github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
)

func NewSingleChainInteropRuntime(t devtest.T) *MultiChainRuntime {
	return NewSingleChainInteropRuntimeWithConfig(t, PresetConfig{})
}

func NewSingleChainInteropRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	migration, l1Net, l2Net, depSet, fullCfgSet := buildSingleChainWorldWithInteropAndState(t, keys, true, cfg.DeployerOptions...)
	validateSimpleInteropPresetConfig(t, cfg, l2Net)

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClock(t, l1Net, jwtPath, l1Clock)
	supervisor := startSupervisor(t, "1-primary", l1EL, fullCfgSet, map[eth.ChainID]*rollup.Config{
		l2Net.ChainID(): l2Net.rollupCfg,
	})

	l2EL := startL2ELNodeWithSupervisor(t, l2Net, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0), supervisor.UserRPC())
	l2CL := startL2CLNode(t, keys, l1Net, l2Net, l1EL, l1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   true,
		DependencySet:  depSet,
		L2FollowSource: "",
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	connectManagedL2CLToSupervisor(t, supervisor, l2CL)

	l2Batcher := startMinimalBatcher(t, keys, l2Net, l1EL, l2CL, l2EL, cfg.BatcherOptions...)
	l2Proposer := startMinimalProposer(t, keys, l2Net, l1EL, l2CL, cfg.ProposerOptions...)
	applyMinimalGameTypeOptions(t, keys, l1Net, l2Net, l1EL, cfg.AddedGameTypes, cfg.RespectedGameTypes)
	testSequencer := startTestSequencer(t, keys, jwtPath, jwtSecret, l1Net, l1EL, l1CL, l2EL, l2CL)
	faucetService := startFaucets(t, keys, l1Net.ChainID(), l2Net.ChainID(), l1EL.UserRPC(), l2EL.UserRPC())

	chainA := &MultiChainNodeRuntime{
		Name:     "l2a",
		Network:  l2Net,
		EL:       l2EL,
		CL:       l2CL,
		Batcher:  l2Batcher,
		Proposer: l2Proposer,
	}

	return &MultiChainRuntime{
		Keys:              keys,
		FullConfigSet:     fullCfgSet,
		DependencySet:     depSet,
		Migration:         migration,
		L1Network:         l1Net,
		L1EL:              l1EL,
		L1CL:              l1CL,
		Chains:            map[string]*MultiChainNodeRuntime{"l2a": chainA},
		PrimarySupervisor: supervisor,
		FaucetService:     faucetService,
		TimeTravel:        timeTravelClock,
		TestSequencer:     newTestSequencerRuntime(testSequencer, "dev"),
	}
}

func NewSimpleInteropRuntime(t devtest.T) *MultiChainRuntime {
	return NewSimpleInteropRuntimeWithConfig(t, PresetConfig{})
}

func NewSimpleInteropRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	migration, l1Net, l2ANet, l2BNet, fullCfgSet := buildTwoL2WorldWithState(t, keys, true, cfg.DeployerOptions...)
	validateSimpleInteropPresetConfig(t, cfg, l2ANet, l2BNet)
	depSet := fullCfgSet.DependencySet

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClock(t, l1Net, jwtPath, l1Clock)
	supervisor := startSupervisor(t, "1-primary", l1EL, fullCfgSet, map[eth.ChainID]*rollup.Config{
		l2ANet.ChainID(): l2ANet.rollupCfg,
		l2BNet.ChainID(): l2BNet.rollupCfg,
	})

	l2AEL := startL2ELNodeWithSupervisor(t, l2ANet, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0), supervisor.UserRPC())
	l2BEL := startL2ELNodeWithSupervisor(t, l2BNet, jwtPath, jwtSecret, "sequencer", NewELNodeIdentity(0), supervisor.UserRPC())
	l2ACL := startL2CLNode(t, keys, l1Net, l2ANet, l1EL, l1CL, l2AEL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   true,
		DependencySet:  depSet,
		L2FollowSource: "",
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	l2BCL := startL2CLNode(t, keys, l1Net, l2BNet, l1EL, l1CL, l2BEL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   true,
		DependencySet:  depSet,
		L2FollowSource: "",
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	connectManagedL2CLToSupervisor(t, supervisor, l2ACL)
	connectManagedL2CLToSupervisor(t, supervisor, l2BCL)

	l2ABatcher := startMinimalBatcher(t, keys, l2ANet, l1EL, l2ACL, l2AEL, cfg.BatcherOptions...)
	l2AProposer := startMinimalProposer(t, keys, l2ANet, l1EL, l2ACL, cfg.ProposerOptions...)
	l2BBatcher := startMinimalBatcher(t, keys, l2BNet, l1EL, l2BCL, l2BEL, cfg.BatcherOptions...)
	l2BProposer := startMinimalProposer(t, keys, l2BNet, l1EL, l2BCL, cfg.ProposerOptions...)
	testSequencer := startTestSequencer(t, keys, jwtPath, jwtSecret, l1Net, l1EL, l1CL, l2AEL, l2ACL)
	faucetService := startFaucetsForRPCs(t, keys, map[eth.ChainID]string{
		l1Net.ChainID():  l1EL.UserRPC(),
		l2ANet.ChainID(): l2AEL.UserRPC(),
		l2BNet.ChainID(): l2BEL.UserRPC(),
	})

	return &MultiChainRuntime{
		Keys:          keys,
		FullConfigSet: fullCfgSet,
		DependencySet: depSet,
		Migration:     migration,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:     "l2a",
				Network:  l2ANet,
				EL:       l2AEL,
				CL:       l2ACL,
				Batcher:  l2ABatcher,
				Proposer: l2AProposer,
			},
			"l2b": {
				Name:     "l2b",
				Network:  l2BNet,
				EL:       l2BEL,
				CL:       l2BCL,
				Batcher:  l2BBatcher,
				Proposer: l2BProposer,
			},
		},
		PrimarySupervisor: supervisor,
		FaucetService:     faucetService,
		TimeTravel:        timeTravelClock,
		TestSequencer:     newTestSequencerRuntime(testSequencer, "dev"),
	}
}

func validateSimpleInteropPresetConfig(t devtest.T, cfg PresetConfig, l2Nets ...*L2Network) {
	require := t.Require()
	if cfg.MaxSequencingWindow != nil {
		for _, l2Net := range l2Nets {
			require.LessOrEqualf(
				l2Net.rollupCfg.SeqWindowSize,
				*cfg.MaxSequencingWindow,
				"sequencing window of chain %s must fit in max sequencing window size",
				l2Net.ChainID(),
			)
		}
	}
	if cfg.RequireInteropNotAtGen {
		for _, l2Net := range l2Nets {
			interopTime := l2Net.genesis.Config.InteropTime
			require.NotNilf(interopTime, "chain %s must have interop", l2Net.ChainID())
			require.NotZerof(*interopTime, "chain %s interop must not be at genesis", l2Net.ChainID())
		}
	}
}

func NewMultiSupervisorInteropRuntime(t devtest.T) *MultiChainRuntime {
	runtime := NewSimpleInteropRuntime(t)
	chainA := runtime.Chains["l2a"]
	chainB := runtime.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a interop chain")
	t.Require().NotNil(chainB, "missing l2b interop chain")

	supervisorSecondary := startSupervisor(
		t,
		"2-secondary",
		runtime.L1EL,
		runtime.FullConfigSet,
		map[eth.ChainID]*rollup.Config{
			chainA.Network.ChainID(): chainA.Network.rollupCfg,
			chainB.Network.ChainID(): chainB.Network.rollupCfg,
		},
	)

	l2A2EL := startL2ELNodeWithSupervisor(
		t,
		chainA.Network,
		chainA.EL.JWTPath(),
		readJWTSecretFromPath(t, chainA.EL.JWTPath()),
		"verifier",
		NewELNodeIdentity(0),
		supervisorSecondary.UserRPC(),
	)
	l2A2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, chainA.Network, runtime.L1EL, runtime.L1CL, l2A2EL, readJWTSecretFromPath(t, chainA.EL.JWTPath()), l2CLNodeStartConfig{
		Key:            "verifier",
		IsSequencer:    false,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   true,
		DependencySet:  runtime.DependencySet,
		L2FollowSource: "",
	})

	l2B2EL := startL2ELNodeWithSupervisor(
		t,
		chainB.Network,
		chainB.EL.JWTPath(),
		readJWTSecretFromPath(t, chainB.EL.JWTPath()),
		"verifier",
		NewELNodeIdentity(0),
		supervisorSecondary.UserRPC(),
	)
	l2B2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, chainB.Network, runtime.L1EL, runtime.L1CL, l2B2EL, readJWTSecretFromPath(t, chainB.EL.JWTPath()), l2CLNodeStartConfig{
		Key:            "verifier",
		IsSequencer:    false,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		IndexingMode:   true,
		DependencySet:  runtime.DependencySet,
		L2FollowSource: "",
	})

	connectL2CLPeers(t, t.Logger(), chainA.CL, l2A2CL)
	connectL2CLPeers(t, t.Logger(), chainB.CL, l2B2CL)

	connectManagedL2CLToSupervisor(t, supervisorSecondary, l2A2CL)
	connectManagedL2CLToSupervisor(t, supervisorSecondary, l2B2CL)

	if chainA.Followers == nil {
		chainA.Followers = make(map[string]*SingleChainNodeRuntime)
	}
	if chainB.Followers == nil {
		chainB.Followers = make(map[string]*SingleChainNodeRuntime)
	}
	chainA.Followers["verifier"] = &SingleChainNodeRuntime{Name: "verifier", EL: l2A2EL, CL: l2A2CL}
	chainB.Followers["verifier"] = &SingleChainNodeRuntime{Name: "verifier", EL: l2B2EL, CL: l2B2CL}
	runtime.SecondarySupervisor = supervisorSecondary
	return runtime
}

func startSupervisor(
	t devtest.T,
	supervisorName string,
	l1EL *L1Geth,
	fullCfgSet depset.FullConfigSetMerged,
	rollupCfgs map[eth.ChainID]*rollup.Config,
) Supervisor {
	switch os.Getenv("DEVSTACK_SUPERVISOR_KIND") {
	case "kona":
		return startKonaSupervisor(t, supervisorName, l1EL, fullCfgSet, rollupCfgs)
	default:
		return startOPSupervisor(t, supervisorName, l1EL, fullCfgSet)
	}
}

func startOPSupervisor(
	t devtest.T,
	supervisorName string,
	l1EL *L1Geth,
	fullCfgSet depset.FullConfigSetMerged,
) *OpSupervisor {
	cfg := &supervisorConfig.Config{
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
		SyncSources:           &syncnode.CLISyncNodes{},
		L1RPC:                 l1EL.UserRPC(),
		Datadir:               t.TempDir(),
		Version:               "dev",
		FullConfigSetSource:   fullCfgSet,
		MockRun:               false,
		SynchronousProcessors: false,
		DatadirSyncEndpoint:   "",
	}
	supervisorNode := &OpSupervisor{
		name:    supervisorName,
		userRPC: "",
		cfg:     cfg,
		p:       t,
		logger:  t.Logger().New("component", "supervisor"),
		service: nil,
	}
	supervisorNode.Start()
	t.Cleanup(supervisorNode.Stop)
	return supervisorNode
}

func startKonaSupervisor(
	t devtest.T,
	supervisorName string,
	l1EL *L1Geth,
	fullCfgSet depset.FullConfigSetMerged,
	rollupCfgs map[eth.ChainID]*rollup.Config,
) *KonaSupervisor {
	require := t.Require()

	cfgDir := t.TempDir()
	depSetJSON, err := json.Marshal(fullCfgSet.DependencySet)
	require.NoError(err, "failed to marshal dependency set")
	depSetCfgPath := filepath.Join(cfgDir, "depset.json")
	require.NoError(os.WriteFile(depSetCfgPath, depSetJSON, 0o644))

	rollupCfgPath := filepath.Join(cfgDir, "rollup-config-*.json")
	for chainID, cfg := range rollupCfgs {
		rollupData, err := json.Marshal(cfg)
		require.NoError(err, "failed to marshal rollup config for chain %s", chainID)
		filePath := filepath.Join(cfgDir, "rollup-config-"+chainID.String()+".json")
		require.NoError(os.WriteFile(filePath, rollupData, 0o644))
	}

	execPath, err := EnsureRustBinary(t, RustBinarySpec{
		SrcDir:  "rust/kona",
		Package: "kona-supervisor",
		Binary:  "kona-supervisor",
	})
	require.NoError(err, "prepare kona-supervisor binary")
	require.NotEmpty(execPath, "kona-supervisor binary path resolved")

	envVars := []string{
		"RPC_ADDR=127.0.0.1",
		"DATADIR=" + t.TempDir(),
		"DEPENDENCY_SET=" + depSetCfgPath,
		"ROLLUP_CONFIG_PATHS=" + rollupCfgPath,
		"L1_RPC=" + l1EL.UserRPC(),
		"RPC_ENABLE_ADMIN=true",
		"L2_CONSENSUS_NODES=",
		"L2_CONSENSUS_JWT_SECRET=",
		"KONA_LOG_LEVEL=3",
		"KONA_LOG_STDOUT_FORMAT=json",
	}

	konaSupervisor := &KonaSupervisor{
		name:     supervisorName,
		userRPC:  "",
		execPath: execPath,
		args:     []string{},
		env:      envVars,
		p:        t,
	}
	konaSupervisor.Start()
	t.Cleanup(konaSupervisor.Stop)
	return konaSupervisor
}

func connectManagedL2CLToSupervisor(t devtest.T, supervisor Supervisor, l2CL L2CLNode) {
	interopEndpoint, secret := l2CL.InteropRPC()
	supClient, err := dial.DialSupervisorClientWithTimeout(t.Ctx(), t.Logger(), supervisor.UserRPC(), client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(supClient.Close)

	err = retry.Do0(t.Ctx(), 10, retry.Exponential(), func() error {
		return supClient.AddL2RPC(t.Ctx(), interopEndpoint, secret)
	})
	t.Require().NoErrorf(err, "must connect CL node %s to supervisor %s", l2CL, supervisorIDString(supervisor))
}

func supervisorIDString(supervisor Supervisor) string {
	switch s := supervisor.(type) {
	case *OpSupervisor:
		return s.name
	case *KonaSupervisor:
		return s.name
	default:
		return "<unknown>"
	}
}

func startL2ELNodeWithSupervisor(
	t devtest.T,
	l2Net *L2Network,
	jwtPath string,
	jwtSecret [32]byte,
	key string,
	identity *ELNodeIdentity,
	supervisorRPC string,
) *OpGeth {
	cfg := DefaultL2ELConfig()
	cfg.P2PAddr = "127.0.0.1"
	cfg.P2PPort = identity.Port
	cfg.P2PNodeKeyHex = identity.KeyHex()

	l2EL := &OpGeth{
		name:          key,
		p:             t,
		logger:        t.Logger().New("component", "l2el-"+key),
		l2Net:         l2Net,
		jwtPath:       jwtPath,
		jwtSecret:     jwtSecret,
		supervisorRPC: supervisorRPC,
		cfg:           cfg,
	}
	l2EL.Start()
	t.Cleanup(l2EL.Stop)
	return l2EL
}

func readJWTSecretFromPath(t devtest.T, jwtPath string) [32]byte {
	content, err := os.ReadFile(jwtPath)
	t.Require().NoError(err, "failed to read jwt path %s", jwtPath)
	raw, err := hexutil.Decode(strings.TrimSpace(string(content)))
	t.Require().NoError(err, "failed to decode jwt secret from %s", jwtPath)
	t.Require().Len(raw, 32, "invalid jwt secret length from %s", jwtPath)
	var secret [32]byte
	copy(secret[:], raw)
	return secret
}
