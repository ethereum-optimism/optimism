package sysgo

import (
	"context"
	"runtime"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opchallenger "github.com/ethereum-optimism/optimism/op-challenger"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	challengermetrics "github.com/ethereum-optimism/optimism/op-challenger/metrics"
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
	EL          L2ELNode
	CL          L2CLNode
	Flashblocks *FlashblocksRuntimeSupport
}

type singleChainRuntimeSpec struct {
	BuildWorld      func(t devtest.T, keys devkeys.Keys, cfg PresetConfig) singleChainRuntimeWorld
	StartPrimary    func(t devtest.T, keys devkeys.Keys, world singleChainRuntimeWorld, l1EL *L1Geth, l1CL *L1CLNode, jwtPath string, jwtSecret [32]byte, cfg PresetConfig) singleChainPrimaryRuntime
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
	sequencerIdentity := NewELNodeIdentity(0)
	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, sequencerIdentity)
	l2CL := startSequencerCL(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, l2EL, jwtSecret, cfg.GlobalL2CLOptions)
	return singleChainPrimaryRuntime{
		EL: l2EL,
		CL: l2CL,
	}
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
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClock(t, world.L1Network, jwtPath, l1Clock)

	primary := spec.StartPrimary(t, keys, world, l1EL, l1CL, jwtPath, jwtSecret, cfg)
	primaryNode := newSingleChainNodeRuntime("sequencer", true, primary.EL, primary.CL)

	var l2Batcher *L2Batcher
	if spec.StartBatcher {
		l2Batcher = startMinimalBatcher(t, keys, world.L2Network, l1EL, primary.CL, primary.EL, cfg.BatcherOptions...)
	}

	var l2Proposer *L2Proposer
	if spec.StartProposer {
		l2Proposer = startMinimalProposer(t, keys, world.L2Network, l1EL, primary.CL, cfg.ProposerOptions...)
	}

	var l2Challenger *L2Challenger
	if spec.StartChallenger {
		l2Challenger = startMinimalChallenger(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, primary.EL, primary.CL, cfg.EnableCannonKonaForChall)
	}

	applyMinimalGameTypeOptions(t, keys, world.L1Network, world.L2Network, l1EL, cfg.AddedGameTypes, cfg.RespectedGameTypes)

	sequencerEL, ok := primary.EL.(*OpGeth)
	require.True(ok, "single-chain runtime primary EL must be op-geth for test sequencer")
	sequencerCL, ok := primary.CL.(*OpNode)
	require.True(ok, "single-chain runtime primary CL must be op-node for test sequencer")
	testSequencer := startTestSequencer(t, keys, jwtPath, jwtSecret, world.L1Network, l1EL, l1CL, sequencerEL, sequencerCL)
	testSequencerRuntime := newTestSequencerRuntime(testSequencer, spec.TestSequencer)
	faucetService := startFaucets(t, keys, world.L1Network.ChainID(), world.L2Network.ChainID(), l1EL.UserRPC(), primary.EL.UserRPC())

	return &SingleChainRuntime{
		Keys:          keys,
		L1Network:     world.L1Network,
		L2Network:     world.L2Network,
		L1EL:          l1EL,
		L1CL:          l1CL,
		L2EL:          primary.EL,
		L2CL:          primary.CL,
		L2Batcher:     l2Batcher,
		L2Proposer:    l2Proposer,
		L2Challenger:  l2Challenger,
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
		TestSequencer: testSequencerRuntime,
		Nodes: map[string]*SingleChainNodeRuntime{
			primaryNode.Name: primaryNode,
		},
		Flashblocks: primary.Flashblocks,
		Interop:     world.Interop,
	}
}

// SingleChainRuntime is the shared DAG runtime for single-chain preset topologies.
// It is the root for minimal, flashblocks, follower-node, sync-tester, conductor,
// and no-supervisor interop variants.
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
		CompressionAlgo:       derive.Brotli,
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
		DisputeGameType:              1,
		ActiveSequencerCheckDuration: 5 * time.Second,
		WaitNodeSync:                 false,
		RollupRpc:                    l2CL.UserRPC(),
	}
	for _, opt := range proposerOpts {
		if opt == nil {
			continue
		}
		opt(NewComponentTarget("main", l2Net.ChainID()), proposerCLIConfig)
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
	enableCannonKona bool,
) *L2Challenger {
	require := t.Require()
	challengerSecret, err := keys.Secret(devkeys.ChallengerRole.Key(l2Net.ChainID().ToBig()))
	require.NoError(err)

	logger := t.Logger().New("component", "l2-challenger")
	logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

	rollupCfgs := []*rollup.Config{l2Net.rollupCfg}
	l2Geneses := []*core.Genesis{l2Net.genesis}
	options := []sharedchallenger.Option{
		sharedchallenger.WithFactoryAddress(l2Net.deployment.DisputeGameFactoryProxyAddr()),
		sharedchallenger.WithPrivKey(challengerSecret),
		sharedchallenger.WithCannonConfig(rollupCfgs, l1Net.genesis, l2Geneses, sharedchallenger.MTCannonVariant),
		sharedchallenger.WithCannonGameType(),
		sharedchallenger.WithPermissionedGameType(),
		sharedchallenger.WithFastGames(),
	}
	if enableCannonKona {
		t.Log("Enabling cannon-kona for challenger")
		options = append(options,
			sharedchallenger.WithCannonKonaConfig(rollupCfgs, l1Net.genesis, l2Geneses),
			sharedchallenger.WithCannonKonaGameType(),
		)
	}
	cfg, err := sharedchallenger.NewPreInteropChallengerConfig(
		t.TempDir(),
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
	}
}

func applyMinimalGameTypeOptions(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l2Net *L2Network,
	l1EL L1ELNode,
	addedGameTypes []gameTypes.GameType,
	respectedGameTypes []gameTypes.GameType,
) {
	if len(addedGameTypes) == 0 && len(respectedGameTypes) == 0 {
		return
	}
	l1ChainID := l1Net.ChainID()

	for _, gameType := range addedGameTypes {
		if gameType == gameTypes.PermissionedGameType {
			continue
		}
		addGameTypeForRuntime(t, keys, PrestateForGameType(t, gameType), gameType, l1ChainID, l1EL.UserRPC(), l2Net)
	}
	for _, gameType := range respectedGameTypes {
		setRespectedGameTypeForRuntime(t, keys, gameType, l1ChainID, l1EL.UserRPC(), l2Net)
	}
}
