package dsl

import (
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	foundryArtifactsDir = "../../../packages/contracts-bedrock/forge-artifacts"
	sourceMapDir        = "../../../packages/contracts-bedrock"
)

// Chain holds the most common per-chain action-test data and actors
type Chain struct {
	ChainID eth.ChainID

	RollupCfg     *rollup.Config
	L1ChainConfig *params.ChainConfig
	DependencySet depset.DependencySet
	L2Genesis     *core.Genesis
	BatcherAddr   common.Address

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

// messageExpiryTime is the time in seconds that a message will be valid for on the L2 chain.
const messageExpiryTime = 120 // 2 minutes

type setupOption func(*interopgen.InteropDevRecipe)

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

// RecipeToDepSet converts a recipe into a dependency-set.
func RecipeToDepSet(t helpers.Testing, recipe *interopgen.InteropDevRecipe) *depset.StaticConfigDependencySet {
	depSetCfg := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for _, out := range recipe.L2s {
		depSetCfg[eth.ChainIDFromUInt64(out.ChainID)] = &depset.StaticConfigDependency{}
	}
	depSet, err := depset.NewStaticConfigDependencySetWithMessageExpiryOverride(depSetCfg, recipe.ExpiryTime)
	require.NoError(t, err)
	return depSet
}

// createL2Services creates a Chain bundle, with the given configs, and attached to the given L1 miner.
func createL2Services(
	t helpers.Testing,
	logger log.Logger,
	l1Miner *helpers.L1Miner,
	keys devkeys.Keys,
	output *interopgen.L2Output,
	depSet depset.DependencySet,
	l1ChainConfig *params.ChainConfig,
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
		l1Miner.BlobStore(), altda.Disabled, seqCl, output.RollupCfg, l1ChainConfig, depSet, 0)

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
		L1ChainConfig:   l1ChainConfig,
		DependencySet:   depSet,
		L2Genesis:       output.Genesis,
		BatcherAddr:     crypto.PubkeyToAddress(batcherKey.PublicKey),
		Sequencer:       seq,
		SequencerEngine: eng,
		Batcher:         batcher,
	}
}
