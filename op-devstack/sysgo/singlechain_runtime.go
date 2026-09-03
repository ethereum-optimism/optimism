package sysgo

import (
	"context"
	"path/filepath"
	"runtime"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opchallenger "github.com/ethereum-optimism/optimism/op-challenger"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	challengermetrics "github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	sharedchallenger "github.com/ethereum-optimism/optimism/op-devstack/shared/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type singleChainRuntimeWorld struct {
	L1Network *L1Network
	L2Network *L2Network
	Interop   *SingleChainInteropSupport
}

type singleChainPrimaryRuntime struct {
	EL               L2ELNode
	CL               L2CLNode
	FactoryHandledCL bool
}

type singleChainRuntimeSpec struct {
	BuildWorld      func(t devtest.T, keys devkeys.Keys, cfg PresetConfig) singleChainRuntimeWorld
	StartPrimary    func(t devtest.T, keys devkeys.Keys, world singleChainRuntimeWorld, l1EL *L1Geth, l1CL *L1CLNode, jwtPath string, jwtSecret [32]byte, cfg PresetConfig) singleChainPrimaryRuntime
	StartBatcher    bool
	StartProposer   bool
	StartChallenger bool
	TestSequencer   string
}

// SingleChainRuntimeOptions controls which product-neutral services surround
// the primary L2 node. Additional nodes can be attached with AddSingleChainNode.
type SingleChainRuntimeOptions struct {
	StartBatcher    bool
	StartProposer   bool
	StartChallenger bool
	TestSequencer   string
}

func newSingleChainNodeRuntime(name string, isSequencer bool, el L2ELNode, cl L2CLNode) *SingleChainNodeRuntime {
	return &SingleChainNodeRuntime{
		Name:        name,
		IsSequencer: isSequencer,
		EL:          el,
		CL:          cl,
	}
}

func newDefaultSingleChainWorld(t devtest.T, keys devkeys.Keys, cfg PresetConfig) singleChainRuntimeWorld {
	if cfg.InteropAtGenesis {
		deployerOpts := append([]DeployerOption{
			WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
		}, cfg.DeployerOptions...)
		l1Net, l2Net, depSet, fullCfgSet := buildSingleChainWorldWithInterop(t, keys, true, cfg.LocalContractArtifactsPath, deployerOpts...)
		return singleChainRuntimeWorld{
			L1Network: l1Net,
			L2Network: l2Net,
			Interop: &SingleChainInteropSupport{
				DependencySet: depSet,
				FullConfigSet: fullCfgSet,
			},
		}
	}
	l1Net, l2Net := buildSingleChainWorld(t, keys, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	return singleChainRuntimeWorld{
		L1Network: l1Net,
		L2Network: l2Net,
	}
}

func startDefaultSingleChainPrimary(
	t devtest.T,
	keys devkeys.Keys,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
) singleChainPrimaryRuntime {
	safeDBPath := filepath.Join(t.TempDirWithPrefix("l2-safe-db-"+world.L2Network.ChainID().String()), "safe-head.db")
	l2CLOptions := []L2CLOption{L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.SafeDBPath = safeDBPath
	})}
	l2CLOptions = append(l2CLOptions, cfg.GlobalL2CLOptions...)
	// Env-resolved options come first so an explicit binary from the test overrides the env one.
	sequencerELOpts := append(append([]OpRethOption{}, ResolveMixedL2ELOpts(t)...), cfg.OpRethOptions...)
	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, NewELNodeIdentity(0), sequencerELOpts...)
	var depSet depset.DependencySet
	if world.Interop != nil {
		depSet = world.Interop.DependencySet
	}
	fallbackKind := singleChainPrimaryFallbackKind(world)
	result := startL2CLForKeyWithKind(
		t, keys, world.L1Network, world.L2Network, l1EL, l1CL, l2EL, jwtSecret,
		"sequencer", "sequencer", true, "", depSet, l2CLOptions, cfg.L2CLFactory,
		nil, false, fallbackKind,
	)
	return singleChainPrimaryRuntime{
		EL:               l2EL,
		CL:               result.Node,
		FactoryHandledCL: result.FactoryHandled,
	}
}

func singleChainPrimaryFallbackKind(world singleChainRuntimeWorld) MixedL2CLKind {
	if world.Interop != nil {
		// Interop primaries were explicitly op-node before the external factory
		// seam and must not become env-selected Kona nodes when the factory is
		// absent or declines.
		return MixedL2CLOpNode
	}
	return devstackL2CLKind()
}

func newSingleChainRuntimeWithConfig(t devtest.T, cfg PresetConfig, spec singleChainRuntimeSpec) *SingleChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	world := spec.BuildWorld(t, keys, cfg)
	jwtPath, jwtSecret := writeJWTSecret(t)

	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock()
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClockConfig(t, world.L1Network, jwtPath, l1Clock, cfg)

	primary := spec.StartPrimary(t, keys, world, l1EL, l1CL, jwtPath, jwtSecret, cfg)
	primaryNode := newSingleChainNodeRuntime("sequencer", true, primary.EL, primary.CL)
	primaryNode.FactoryHandledCL = primary.FactoryHandledCL

	var l2Batcher *L2Batcher
	if spec.StartBatcher {
		l2Batcher = startMinimalBatcher(t, keys, world.L2Network, l1EL, primary.CL, primary.EL, cfg.BatcherOptions...)
	}

	applyMinimalGameTypeOptions(t, keys, world.L1Network, world.L2Network, l1EL, primary.CL, cfg.AddedGameTypes, cfg.RespectedGameTypes)

	var l2Proposer *L2Proposer
	if spec.StartProposer && !cfg.SkipHonestProposer {
		l2Proposer = startMinimalProposer(t, keys, world.L2Network, l1EL, primary.CL, cfg.ProposerOptions...)
	}

	var l2Challenger *L2Challenger
	var zkChallengerSuperRootRPCProxy *StallableProxy
	if spec.StartChallenger {
		l2Challenger, zkChallengerSuperRootRPCProxy = startMinimalChallenger(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, primary.EL, primary.CL, cfg.AddedGameTypes)
	}

	testSequencer := startTestSequencerForRPCs(t, keys, "test-sequencer", jwtPath, jwtSecret, world.L1Network, l1EL, l1CL, world.L2Network.ChainID(), primary.EL.UserRPC(), primary.CL.UserRPC())
	testSequencerRuntime := newTestSequencerRuntime(testSequencer, spec.TestSequencer)

	return &SingleChainRuntime{
		Keys:                          keys,
		L2CLFactory:                   cfg.L2CLFactory,
		L1Network:                     world.L1Network,
		L2Network:                     world.L2Network,
		L1EL:                          l1EL,
		L1CL:                          l1CL,
		L2EL:                          primary.EL,
		L2CL:                          primary.CL,
		L2Batcher:                     l2Batcher,
		L2Proposer:                    l2Proposer,
		L2Challenger:                  l2Challenger,
		ZKChallengerSuperRootRPCProxy: zkChallengerSuperRootRPCProxy,
		TimeTravel:                    timeTravelClock,
		TestSequencer:                 testSequencerRuntime,
		Nodes: map[string]*SingleChainNodeRuntime{
			primaryNode.Name: primaryNode,
		},
		Interop: world.Interop,
	}
}

// SingleChainRuntime is the shared DAG runtime for single-chain preset topologies.
// It is the root for minimal, follower-node, sync-tester, conductor, and no-supernode interop
// variants.
func NewMinimalRuntime(t devtest.T) *SingleChainRuntime {
	return NewMinimalRuntimeWithConfig(t, PresetConfig{})
}

func NewMinimalRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: true,
	})
}

// NewSingleChainRuntime constructs a composable single-chain runtime using
// the default world and primary-node builders.
func NewSingleChainRuntime(t devtest.T, cfg PresetConfig, opts SingleChainRuntimeOptions) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    opts.StartBatcher,
		StartProposer:   opts.StartProposer,
		StartChallenger: opts.StartChallenger,
		TestSequencer:   opts.TestSequencer,
	})
}

// NewMinimalNoFaultProofsRuntimeWithConfig returns a minimal single-chain
// runtime without the proposer or challenger. It is intended for tests that
// only exercise the sequencer + batcher + derivation loop and do not need
// fault proofs. Skipping the challenger also avoids requiring cannon prestate
// artifacts, which are expensive to build locally.
func NewMinimalNoFaultProofsRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   false,
		StartChallenger: false,
	})
}

func startMinimalBatcher(
	t devtest.T,
	keys devkeys.Keys,
	l2Net *L2Network,
	l1EL L1ELNode,
	l2CL L2CLNode,
	l2EL L2ELNode,
	batcherOpts ...BatcherOption,
) *L2Batcher {
	require := t.Require()
	batcherSecret, err := keys.Secret(devkeys.BatcherRole.Key(l2Net.ChainID().ToBig()))
	require.NoError(err)
	batcherTarget := NewComponentTarget("main", l2Net.ChainID())

	logger := t.Logger().New("component", "l2-batcher")
	logger.SetContext(t.Ctx())
	logger.Info("Batcher key acquired", "addr", crypto.PubkeyToAddress(batcherSecret.PublicKey))

	compressionAlgo := derive.Zlib
	if l2Net.rollupCfg.IsFjord(l2Net.rollupCfg.Genesis.L2Time) {
		compressionAlgo = derive.Brotli
	}

	batcherCLIConfig := &bss.CLIConfig{
		L1EthRpc:                 l1EL.UserRPC(),
		L2EthRpc:                 []string{l2EL.UserRPC()},
		RollupRpc:                []string{l2CL.UserRPC()},
		MaxPendingTransactions:   7,
		MaxChannelDuration:       1,
		MaxL1TxSize:              120_000,
		TestUseMaxTxSizeForBlobs: false,
		TargetNumFrames:          1,
		ApproxComprRatio:         0.4,
		SubSafetyMargin:          4,
		PollInterval:             500 * time.Millisecond,
		TxMgrConfig:              setuputils.NewTxMgrConfig(endpoint.URL(l1EL.UserRPC()), batcherSecret),
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelInfo,
			Format: oplog.FormatText,
		},
		Stopped:               false,
		BatchType:             derive.SpanBatchType,
		MaxBlocksPerSpanBatch: 10,
		DataAvailabilityType:  batcherFlags.CalldataType,
		CompressionAlgo:       compressionAlgo,
		RPC: oprpc.CLIConfig{
			EnableAdmin: true,
		},
	}
	for _, opt := range batcherOpts {
		if opt == nil {
			continue
		}
		opt(batcherTarget, batcherCLIConfig)
	}

	batcherCtx, cancelBatcherCtx := context.WithCancel(t.Ctx())
	closeAppFn := func(cause error) {
		t.Errorf("closeAppFn called, batcher hit a critical error: %v", cause)
		cancelBatcherCtx()
	}
	batcher, err := bss.BatcherServiceFromCLIConfig(
		batcherCtx,
		closeAppFn,
		"0.0.1",
		batcherCLIConfig,
		logger,
	)
	require.NoError(err)
	require.NoError(batcher.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing batcher")
		_ = batcher.Stop(ctx)
		logger.Info("Closed batcher")
	})

	return &L2Batcher{
		name:    batcherTarget.Name,
		chainID: batcherTarget.ChainID,
		service: batcher,
		rpc:     batcher.HTTPEndpoint(),
		l1RPC:   l1EL.UserRPC(),
		l2CLRPC: l2CL.UserRPC(),
		l2ELRPC: l2EL.UserRPC(),
	}
}

func startMinimalProposer(
	t devtest.T,
	keys devkeys.Keys,
	l2Net *L2Network,
	l1EL L1ELNode,
	l2CL L2CLNode,
	proposerOpts ...ProposerOption,
) *L2Proposer {
	require := t.Require()
	proposerSecret, err := keys.Secret(devkeys.ProposerRole.Key(l2Net.ChainID().ToBig()))
	require.NoError(err)

	logger := t.Logger().New("component", "l2-proposer")
	logger.Info("Proposer key acquired", "addr", crypto.PubkeyToAddress(proposerSecret.PublicKey))

	proposerCLIConfig := &ps.CLIConfig{
		L1EthRpc:          l1EL.UserRPC(),
		PollInterval:      500 * time.Millisecond,
		AllowNonFinalized: true,
		TxMgrConfig:       setuputils.NewTxMgrConfig(endpoint.URL(l1EL.UserRPC()), proposerSecret),
		RPCConfig: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		LogConfig: oplog.CLIConfig{
			Level:  log.LvlInfo,
			Format: oplog.FormatText,
		},
		MetricsConfig:                opmetrics.CLIConfig{},
		PprofConfig:                  oppprof.CLIConfig{},
		DGFAddress:                   l2Net.deployment.DisputeGameFactoryProxyAddr().Hex(),
		ProposalInterval:             6 * time.Second,
		DisputeGameType:              superPermissionedGameType,
		ActiveSequencerCheckDuration: 5 * time.Second,
		WaitNodeSync:                 false,
	}
	for _, opt := range proposerOpts {
		if opt == nil {
			continue
		}
		opt(NewComponentTarget("main", l2Net.ChainID()), proposerCLIConfig)
	}
	switch proposerCLIConfig.DisputeGameType {
	case superPermissionedGameType, superCannonKonaGameType:
		proposerCLIConfig.RollupRpc = ""
		proposerCLIConfig.SuperRootRpcs = []string{l2CL.UserRPC()}
	default:
		proposerCLIConfig.SuperRootRpcs = nil
		proposerCLIConfig.RollupRpc = l2CL.UserRPC()
	}

	proposer, err := ps.ProposerServiceFromCLIConfig(t.Ctx(), "0.0.1", proposerCLIConfig, logger)
	require.NoError(err)
	require.NoError(proposer.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing proposer")
		_ = proposer.Stop(ctx)
		logger.Info("Closed proposer")
	})

	return &L2Proposer{
		name:    "main",
		chainID: l2Net.ChainID(),
		service: proposer,
		userRPC: proposer.HTTPEndpoint(),
	}
}

func startMinimalChallenger(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	l2EL L2ELNode,
	l2CL L2CLNode,
	addedGameTypes []gameTypes.GameType,
) (*L2Challenger, *StallableProxy) {
	require := t.Require()
	challengerSecret, err := keys.Secret(devkeys.ChallengerRole.Key(l2Net.ChainID().ToBig()))
	require.NoError(err)

	logger := t.Logger().New("component", "l2-challenger")
	logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

	rollupCfgs := []*rollup.Config{l2Net.rollupCfg}
	l2Geneses := []*core.Genesis{l2Net.genesis}
	dependencySet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		l2Net.ChainID(): {},
	})
	require.NoError(err)

	options := []sharedchallenger.Option{
		sharedchallenger.WithFactoryAddress(l2Net.deployment.DisputeGameFactoryProxyAddr()),
		sharedchallenger.WithPrivKey(challengerSecret),
		sharedchallenger.WithPermissionedCannonConfig(rollupCfgs, l1Net.genesis, l2Geneses),
		sharedchallenger.WithPermissionedGameType(),
		sharedchallenger.WithFastGames(),
	}
	var cannonKonaEnabled, superCannonKonaEnabled, zkEnabled bool
	for _, gameType := range addedGameTypes {
		cannonKonaEnabled = cannonKonaEnabled || gameType == gameTypes.CannonKonaGameType
		superCannonKonaEnabled = superCannonKonaEnabled || gameType == gameTypes.SuperCannonKonaGameType
		zkEnabled = zkEnabled || gameType == gameTypes.ZKDisputeGameType
	}
	require.False(cannonKonaEnabled && superCannonKonaEnabled, "minimal challenger cannot use legacy and interop Cannon Kona prestates simultaneously")
	require.False(zkEnabled && (cannonKonaEnabled || superCannonKonaEnabled), "minimal challenger cannot use the ZK game alongside cannon-kona game types")
	var zkChallengerSuperRootRPCProxy *StallableProxy
	switch {
	case zkEnabled:
		// The ZK game validates super roots from the op-node's superroot_atTimestamp endpoint;
		// it needs no VM config or dependency set.
		zkChallengerSuperRootRPCProxy = StartStallableProxy(t, "zk-challenger-super-root", l2CL.UserRPC())
		options = append(options,
			sharedchallenger.WithZKDisputeGameType(),
			sharedchallenger.WithSuperRootRPC(zkChallengerSuperRootRPCProxy.URL()),
		)
	case superCannonKonaEnabled:
		options = append(options,
			sharedchallenger.WithDepset(dependencySet),
			sharedchallenger.WithCannonKonaInteropConfig(rollupCfgs, l1Net.genesis, l2Geneses),
			sharedchallenger.WithSuperCannonKonaGameType(),
			sharedchallenger.WithSuperRootRPC(l2CL.UserRPC()),
		)
	case cannonKonaEnabled:
		options = append(options,
			sharedchallenger.WithCannonKonaConfig(rollupCfgs, l1Net.genesis, l2Geneses),
			sharedchallenger.WithCannonKonaGameType(),
		)
	}
	cfg, err := sharedchallenger.NewPreInteropChallengerConfig(
		t.Ctx(),
		t.TempDirWithPrefix("l2-challenger-"+l2Net.ChainID().String()),
		l1EL.UserRPC(),
		l1CL.beaconHTTPAddr,
		l2CL.UserRPC(),
		l2EL.UserRPC(),
		options...,
	)
	require.NoError(err, "failed to create pre-interop challenger config")

	svc, err := opchallenger.Main(t.Ctx(), logger, cfg, challengermetrics.NoopMetrics)
	require.NoError(err)
	require.NoError(svc.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing challenger")
		timer := time.AfterFunc(1*time.Minute, func() {
			if svc.Stopped() {
				return
			}
			buf := make([]byte, 1<<20)
			stackLen := runtime.Stack(buf, true)
			logger.Error("Challenger failed to stop; printing all goroutine stacks:\n%v", string(buf[:stackLen]))
		})
		_ = svc.Stop(ctx)
		timer.Stop()
		logger.Info("Closed challenger")
	})

	return &L2Challenger{
		name:     "main",
		chainIDs: []eth.ChainID{l2Net.ChainID()},
		service:  svc,
		config:   cfg,
	}, zkChallengerSuperRootRPCProxy
}

func applyMinimalGameTypeOptions(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	l2CL L2CLNode,
	addedGameTypes []gameTypes.GameType,
	respectedGameTypes []gameTypes.GameType,
) {
	if len(addedGameTypes) == 0 && len(respectedGameTypes) == 0 {
		return
	}
	l1ChainID := l1Net.ChainID()

	if len(addedGameTypes) > 0 {
		addGameTypesForRuntime(t, keys, addedGameTypes, l1ChainID, l1EL.UserRPC(), l2Net, l2CL)
	}
	for _, gameType := range respectedGameTypes {
		setRespectedGameTypeForRuntime(t, keys, gameType, l1ChainID, l1EL.UserRPC(), l2Net)
	}
}
