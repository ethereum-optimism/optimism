package dsl

import (
	"context"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	foundryArtifactsDir = "../../../packages/contracts-bedrock/forge-artifacts"
	sourceMapDir        = "../../../packages/contracts-bedrock"
)

// Chain holds the most common per-chain action-test data and actors
type Chain struct {
	ChainID eth.ChainID

	RollupCfg   *rollup.Config
	L2Genesis   *core.Genesis
	BatcherAddr common.Address

	Sequencer       *helpers.L2Sequencer
	SequencerEngine *helpers.L2Engine
	Batcher         *helpers.L2Batcher
}

// InteropSetup holds the chain deployment and config contents, before instantiating any services.
type InteropSetup struct {
	Log        log.Logger
	Deployment *interopgen.WorldDeployment
	Out        *interopgen.WorldOutput
	CfgSet     depset.FullConfigSetMerged
	Keys       devkeys.Keys
	T          helpers.Testing
}

// InteropActors holds a bundle of global actors and actors of 2 chains.
type InteropActors struct {
	L1Miner    *helpers.L1Miner
	Supervisor *SupervisorActor
	ChainA     *Chain
	ChainB     *Chain
}

func (actors *InteropActors) PrepareChainState(t helpers.Testing) {
	// Initialize both chain states
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	t.Log("Sequencers should initialize, and produce initial reset requests")

	// Process the anchor point
	actors.Supervisor.ProcessFull(t)
	t.Log("Supervisor should have anchor points now")

	// Sync supervisors, i.e. the reset request makes it to the supervisor now
	actors.ChainA.Sequencer.SyncSupervisor(t)
	actors.ChainB.Sequencer.SyncSupervisor(t)
	t.Log("Supervisor has events now")

	// Pick up the reset request
	actors.Supervisor.ProcessFull(t)
	t.Log("Supervisor processed initial resets")

	// Process reset work
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	t.Log("Processed!")
}

func (actors *InteropActors) VerifyGenesisState(t helpers.Testing) {
	// Verify initial state
	statusA := actors.ChainA.Sequencer.SyncStatus()
	statusB := actors.ChainB.Sequencer.SyncStatus()
	require.Equal(t, uint64(0), statusA.UnsafeL2.Number)
	require.Equal(t, uint64(0), statusB.UnsafeL2.Number)
}

func (actors *InteropActors) PrepareAndVerifyInitialState(t helpers.Testing) {
	actors.PrepareChainState(t)
	actors.VerifyGenesisState(t)
}

// messageExpiryTime is the time in seconds that a message will be valid for on the L2 chain.
// At a 2 second block time, this should be small enough to cover all events buffered in the supervisor event queue.
const messageExpiryTime = 120 // 2 minutes

type setupOption func(*interopgen.InteropDevRecipe)

func SetBlockTimeForChainA(blockTime uint64) setupOption {
	return func(recipe *interopgen.InteropDevRecipe) {
		recipe.L2s[0].BlockTime = blockTime
	}
}

func SetBlockTimeForChainB(blockTime uint64) setupOption {
	return func(recipe *interopgen.InteropDevRecipe) {
		recipe.L2s[1].BlockTime = blockTime
	}
}

func SetMessageExpiryTime(expiryTime uint64) setupOption {
	return func(recipe *interopgen.InteropDevRecipe) {
		recipe.ExpiryTime = expiryTime
	}
}

func SetInteropOffsetForAllL2s(offset uint64) setupOption {
	return func(recipe *interopgen.InteropDevRecipe) {
		for i, l2 := range recipe.L2s {
			l2.InteropOffset = offset
			recipe.L2s[i] = l2
		}
	}
}

func SetInteropForkScheduledButInactive() setupOption {
	return func(recipe *interopgen.InteropDevRecipe) {
		// Update in place to avoid making a copy and losing the change.
		// Set to a year in the future. Far enough tests won't hit it
		// but not so far it will overflow when added to current time.
		val := uint64(365 * 24 * 60 * 60)
		for key := range recipe.L2s {
			recipe.L2s[key].InteropOffset = val
		}
	}
}

// SetupInterop creates an InteropSetup to instantiate actors on, with 2 L2 chains.
func SetupInterop(t helpers.Testing, opts ...setupOption) *InteropSetup {
	recipe := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2s:              []interopgen.InteropDevL2Recipe{{ChainID: 900200}, {ChainID: 900201}},
		GenesisTimestamp: uint64(time.Now().Unix() + 3),
		ExpiryTime:       messageExpiryTime,
	}
	for _, opt := range opts {
		opt(&recipe)
	}

	logger := testlog.Logger(t, log.LevelDebug)
	hdWallet, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	worldCfg, err := recipe.Build(hdWallet)
	require.NoError(t, err)

	for _, l2Cfg := range worldCfg.L2s {
		require.NotNil(t, l2Cfg.L2GenesisIsthmusTimeOffset, "expecting isthmus fork to be enabled for interop deployments")
	}

	// create the foundry artifacts and source map
	foundryArtifacts := foundry.OpenArtifactsDir(foundryArtifactsDir)
	sourceMap := foundry.NewSourceMapFS(os.DirFS(sourceMapDir))

	// deploy the world, using the logger, foundry artifacts, source map, and world configuration
	worldDeployment, worldOutput, err := interopgen.Deploy(logger, foundryArtifacts, sourceMap, worldCfg)
	require.NoError(t, err)
	depSet := RecipeToDepSet(t, &recipe)
	rollupConfigSet := worldOutput.RollupConfigSet()
	cfgSet, err := depset.NewFullConfigSetMerged(rollupConfigSet, depSet)
	require.NoError(t, err)

	return &InteropSetup{
		Log:        logger,
		Deployment: worldDeployment,
		Out:        worldOutput,
		CfgSet:     cfgSet,
		Keys:       hdWallet,
		T:          t,
	}
}

func (is *InteropSetup) CreateActors() *InteropActors {
	l1Miner := helpers.NewL1Miner(is.T, is.Log.New("role", "l1Miner"), is.Out.L1.Genesis)
	supervisorAPI := NewSupervisor(is.T, is.Log, is.CfgSet)
	supervisorAPI.backend.AttachL1Source(l1Miner.L1ClientSimple(is.T))
	require.NoError(is.T, supervisorAPI.backend.Start(is.T.Ctx()))
	is.T.Cleanup(func() {
		require.NoError(is.T, supervisorAPI.backend.Stop(context.Background()))
	})
	chainA := createL2Services(is.T, is.Log, l1Miner, is.Keys, is.Out.L2s["900200"], is.CfgSet)
	chainB := createL2Services(is.T, is.Log, l1Miner, is.Keys, is.Out.L2s["900201"], is.CfgSet)
	// Hook up L2 RPCs to supervisor, to fetch event data from
	srcA := chainA.Sequencer.InteropSyncNode(is.T)
	srcB := chainB.Sequencer.InteropSyncNode(is.T)
	nodeA, err := supervisorAPI.backend.AttachSyncNode(is.T.Ctx(), srcA, true)
	require.NoError(is.T, err)
	nodeB, err := supervisorAPI.backend.AttachSyncNode(is.T.Ctx(), srcB, true)
	require.NoError(is.T, err)
	chainA.Sequencer.InteropControl = nodeA
	chainB.Sequencer.InteropControl = nodeB
	return &InteropActors{
		L1Miner:    l1Miner,
		Supervisor: supervisorAPI,
		ChainA:     chainA,
		ChainB:     chainB,
	}
}

// SupervisorActor represents a supervisor, instrumented to run synchronously for action-test purposes.
type SupervisorActor struct {
	exec    *event.GlobalSyncExec
	backend *backend.SupervisorBackend
	sources.SupervisorClient
}

func (sa *SupervisorActor) ProcessFull(t helpers.Testing) {
	require.NoError(t, sa.exec.Drain(), "process all supervisor events")
}

func (sa *SupervisorActor) SignalLatestL1(t helpers.Testing) {
	require.NoError(t, sa.backend.PullLatestL1())
}

func (sa *SupervisorActor) SignalFinalizedL1(t helpers.Testing) {
	require.NoError(t, sa.backend.PullFinalizedL1())
}

func (sa *SupervisorActor) Rewind(chain eth.ChainID, block eth.BlockID) error {
	return sa.backend.Rewind(context.Background(), chain, block)
}

// RecipeToDepSet converts a recipe into a dependency-set for the supervisor.
func RecipeToDepSet(t helpers.Testing, recipe *interopgen.InteropDevRecipe) *depset.StaticConfigDependencySet {
	depSetCfg := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for _, out := range recipe.L2s {
		depSetCfg[eth.ChainIDFromUInt64(out.ChainID)] = &depset.StaticConfigDependency{}
	}
	depSet, err := depset.NewStaticConfigDependencySetWithMessageExpiryOverride(depSetCfg, recipe.ExpiryTime)
	require.NoError(t, err)
	return depSet
}

// NewSupervisor creates a new SupervisorActor, to action-test the supervisor with.
func NewSupervisor(t helpers.Testing, logger log.Logger, fullCfgSet depset.FullConfigSetSource) *SupervisorActor {
	logger = logger.New("role", "supervisor")
	supervisorDataDir := t.TempDir()
	logger.Info("supervisor data dir", "dir", supervisorDataDir)
	svCfg := &config.Config{
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		Datadir:               supervisorDataDir,
		SyncSources:           &syncnode.CLISyncNodes{}, // sources are added dynamically afterwards
	}
	evExec := event.NewGlobalSynchronous(t.Ctx())
	b, err := backend.NewSupervisorBackend(t.Ctx(), logger, metrics.NoopMetrics, svCfg, evExec)
	require.NoError(t, err)
	b.SetConfDepthL1(0)

	rpcServer := helpers.NewSimpleRPCServer()
	supervisor.RegisterRPCs(logger, svCfg, rpcServer, b, metrics.NoopMetrics)
	rpcServer.Start(t)
	supervisorClient := sources.NewSupervisorClient(rpcServer.Connect(t))
	return &SupervisorActor{
		exec:             evExec,
		backend:          b,
		SupervisorClient: *supervisorClient,
	}
}

// createL2Services creates a Chain bundle, with the given configs, and attached to the given L1 miner.
func createL2Services(
	t helpers.Testing,
	logger log.Logger,
	l1Miner *helpers.L1Miner,
	keys devkeys.Keys,
	output *interopgen.L2Output,
	depSet depset.DependencySet,
) *Chain {
	logger = logger.New("chain", output.Genesis.Config.ChainID)

	jwtPath := e2eutils.WriteDefaultJWT(t)

	eng := helpers.NewL2Engine(t, logger.New("role", "engine"), output.Genesis, jwtPath)

	seqCl, err := sources.NewEngineClient(eng.RPCClient(), logger, nil, sources.EngineClientDefaultConfig(output.RollupCfg))
	require.NoError(t, err)

	l1F, err := sources.NewL1Client(l1Miner.RPCClient(), logger, nil,
		sources.L1ClientDefaultConfig(output.RollupCfg, false, sources.RPCKindStandard))
	require.NoError(t, err)

	seq := helpers.NewL2Sequencer(t, logger.New("role", "sequencer"), l1F,
		l1Miner.BlobStore(), altda.Disabled, seqCl, output.RollupCfg, depSet,
		0)

	batcherKey, err := keys.Secret(devkeys.ChainOperatorKey{
		ChainID: output.Genesis.Config.ChainID,
		Role:    devkeys.BatcherRole,
	})
	require.NoError(t, err)

	batcherCfg := &helpers.BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		BatcherKey:           batcherKey,
		DataAvailabilityType: batcherFlags.CalldataType,
	}

	batcher := helpers.NewL2Batcher(logger.New("role", "batcher"), output.RollupCfg, batcherCfg,
		seq.RollupClient(), l1Miner.EthClient(),
		eng.EthClient(), eng.EngineClient(t, output.RollupCfg))

	return &Chain{
		ChainID:         eth.ChainIDFromBig(output.Genesis.Config.ChainID),
		RollupCfg:       output.RollupCfg,
		L2Genesis:       output.Genesis,
		BatcherAddr:     crypto.PubkeyToAddress(batcherKey.PublicKey),
		Sequencer:       seq,
		SequencerEngine: eng,
		Batcher:         batcher,
	}
}

type batchAndMineOption func(*batchAndMineConfig)

type batchAndMineConfig struct {
	shouldMarkSafe  bool
	shouldMarkFinal bool
}

// WithMarkFinal marks the L1 block with L2 batches as safe and finalized.
// Necessary for doing this is creating a second L1 block so that the final head can be be promoted.
func WithMarkFinal() batchAndMineOption {
	return func(cfg *batchAndMineConfig) {
		cfg.shouldMarkFinal = true
	}
}

func WithMarkSafe() batchAndMineOption {
	return func(cfg *batchAndMineConfig) {
		cfg.shouldMarkSafe = true
	}
}

// Creates a new L2 block, submits it to L1, and mines the L1 block.
func (actors *InteropActors) ActBatchAndMine(t helpers.Testing, opts ...batchAndMineOption) {
	cfg := &batchAndMineConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	var batches []*gethTypes.Transaction
	for _, c := range []*Chain{actors.ChainA, actors.ChainB} {
		c.Batcher.ActSubmitAll(t)
		batches = append(batches, c.Batcher.LastSubmitted)
	}
	actors.L1Miner.ActL1StartBlock(12)(t)
	for _, b := range batches {
		actors.L1Miner.ActL1IncludeTxByHash(b.Hash())(t)
	}
	actors.L1Miner.ActL1EndBlock(t)

	if cfg.shouldMarkSafe || cfg.shouldMarkFinal {
		actors.L1Miner.ActL1SafeNext(t)
	}
	if cfg.shouldMarkFinal {
		actors.L1Miner.ActEmptyBlock(t)
		actors.L1Miner.ActL1FinalizeNext(t)
	}
}

type actSyncSupernodeOption func(*actSyncSupernodeConfig)

type actSyncSupernodeConfig struct {
	ChainOpts
	shouldSendL1FinalizedSignal bool
	shouldSendL1LatestSignal    bool
}

func WithChains(chains ...*Chain) actSyncSupernodeOption {
	return func(cfg *actSyncSupernodeConfig) {
		cfg.Chains = chains
	}
}

func WithFinalizedSignal() actSyncSupernodeOption {
	return func(cfg *actSyncSupernodeConfig) {
		cfg.shouldSendL1FinalizedSignal = true
	}
}

func WithLatestSignal() actSyncSupernodeOption {
	return func(cfg *actSyncSupernodeConfig) {
		cfg.shouldSendL1LatestSignal = true
	}
}

func (actors *InteropActors) SyncStatuses(t helpers.Testing, chain *Chain) (*eth.SyncStatus, *eth.SupervisorChainSyncStatus) {
	seqSyncStatus := chain.Sequencer.SyncStatus()
	supSyncStatus, err := actors.Supervisor.SyncStatus(t.Ctx())
	require.NoError(t, err)
	supChainStatus, ok := supSyncStatus.Chains[chain.ChainID]
	require.True(t, ok, "supervisor should have chain status for chain id: %s", chain.ChainID)
	return seqSyncStatus, supChainStatus
}
