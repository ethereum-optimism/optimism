package proofs

import (
	"context"
	"math/rand"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/host/kvstore"
	"github.com/ethereum-optimism/optimism/op-program/host/prefetcher"
	hostTypes "github.com/ethereum-optimism/optimism/op-program/host/types"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// L2FaultProofEnv is a test harness for a fault provable L2 chain.
type L2FaultProofEnv struct {
	log       log.Logger
	batcher   *actions.L2Batcher
	sequencer *actions.L2Sequencer
	engine    *actions.L2Engine
	engCl     *sources.EngineClient
	sd        *e2eutils.SetupData
	dp        *e2eutils.DeployParams
	miner     *actions.L1Miner
	alice     *actions.CrossLayerUser
}

func NewL2FaultProofEnv[c any](t actions.Testing, testCfg *TestCfg[c], tp *e2eutils.TestParams, batcherCfg *actions.BatcherCfg) *L2FaultProofEnv {
	log := testlog.Logger(t, log.LvlDebug)
	dp := NewDeployParams(t, func(dp *e2eutils.DeployParams) {
		genesisBlock := hexutil.Uint64(0)

		// Enable cancun always
		dp.DeployConfig.L1CancunTimeOffset = &genesisBlock

		// Enable L2 feature.
		switch testCfg.Hardfork {
		case Regolith:
			dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
		case Canyon:
			dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
		case Delta:
			dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
		case Ecotone:
			dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
		case Fjord:
			dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisBlock
		case Granite:
			dp.DeployConfig.L2GenesisGraniteTimeOffset = &genesisBlock
		}
	})
	sd := e2eutils.Setup(t, dp, actions.DefaultAlloc)

	jwtPath := e2eutils.WriteDefaultJWT(t)
	cfg := &actions.SequencerCfg{VerifierCfg: *actions.DefaultVerifierCfg()}

	miner := actions.NewL1Miner(t, log.New("role", "l1-miner"), sd.L1Cfg)

	l1Cl, err := sources.NewL1Client(miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindStandard))
	require.NoError(t, err)
	engine := actions.NewL2Engine(t, log.New("role", "sequencer-engine"), sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath, actions.EngineWithP2P())
	l2EngineCl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	sequencer := actions.NewL2Sequencer(t, log.New("role", "sequencer"), l1Cl, miner.BlobStore(), altda.Disabled, l2EngineCl, sd.RollupCfg, 0, cfg.InteropBackend)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	sequencer.ActL2PipelineFull(t)
	engCl := engine.EngineClient(t, sd.RollupCfg)

	// Set the batcher key to the secret key of the batcher
	batcherCfg.BatcherKey = dp.Secrets.Batcher
	batcher := actions.NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engCl)

	addresses := e2eutils.CollectAddresses(sd, dp)
	l1EthCl := miner.EthClient()
	l2EthCl := engine.EthClient()
	l1UserEnv := &actions.BasicUserEnv[*actions.L1Bindings]{
		EthCl:          l1EthCl,
		Signer:         types.LatestSigner(sd.L1Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       actions.NewL1Bindings(t, l1EthCl),
	}
	l2UserEnv := &actions.BasicUserEnv[*actions.L2Bindings]{
		EthCl:          l2EthCl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       actions.NewL2Bindings(t, l2EthCl, engine.GethClient()),
	}
	alice := actions.NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(0xa57b)))
	alice.L1.SetUserEnv(l1UserEnv)
	alice.L2.SetUserEnv(l2UserEnv)

	return &L2FaultProofEnv{
		log:       log,
		batcher:   batcher,
		sequencer: sequencer,
		engine:    engine,
		engCl:     engCl,
		sd:        sd,
		dp:        dp,
		miner:     miner,
		alice:     alice,
	}
}

type FixtureInputParam func(f *FixtureInputs)

type CheckResult func(actions.Testing, error)

func ExpectNoError() CheckResult {
	return func(t actions.Testing, err error) {
		require.NoError(t, err, "fault proof program should have succeeded")
	}
}

func ExpectError(expectedErr error) CheckResult {
	return func(t actions.Testing, err error) {
		require.ErrorIs(t, err, expectedErr, "fault proof program should have failed with expected error")
	}
}

func WithL2Claim(claim common.Hash) FixtureInputParam {
	return func(f *FixtureInputs) {
		f.L2Claim = claim
	}
}

func (env *L2FaultProofEnv) RunFaultProofProgram(t actions.Testing, l2ClaimBlockNum uint64, checkResult CheckResult, fixtureInputParams ...FixtureInputParam) {
	// Fetch the pre and post output roots for the fault proof.
	preRoot, err := env.sequencer.RollupClient().OutputAtBlock(t.Ctx(), l2ClaimBlockNum-1)
	require.NoError(t, err)
	claimRoot, err := env.sequencer.RollupClient().OutputAtBlock(t.Ctx(), l2ClaimBlockNum)
	require.NoError(t, err)
	l1Head := env.miner.L1Chain().CurrentBlock()

	fixtureInputs := &FixtureInputs{
		L2BlockNumber: l2ClaimBlockNum,
		L2Claim:       common.Hash(claimRoot.OutputRoot),
		L2Head:        preRoot.BlockRef.Hash,
		L2OutputRoot:  common.Hash(preRoot.OutputRoot),
		L2ChainID:     env.sd.RollupCfg.L2ChainID.Uint64(),
		L1Head:        l1Head.Hash(),
	}
	for _, apply := range fixtureInputParams {
		apply(fixtureInputs)
	}

	// Run the fault proof program from the state transition from L2 block 0 -> 1.
	programCfg := NewOpProgramCfg(
		t,
		env,
		fixtureInputs,
	)
	withInProcessPrefetcher := host.WithPrefetcher(func(ctx context.Context, logger log.Logger, kv kvstore.KV, cfg *config.Config) (host.Prefetcher, error) {
		// Set up in-process L1 sources
		l1Cl := env.miner.L1Client(t, env.sd.RollupCfg)
		l1BlobFetcher := env.miner.BlobStore()

		// Set up in-process L2 source
		l2ClCfg := sources.L2ClientDefaultConfig(env.sd.RollupCfg, true)
		l2RPC := env.engine.RPCClient()
		l2Client, err := host.NewL2Client(l2RPC, env.log, nil, &host.L2ClientConfig{L2ClientConfig: l2ClCfg, L2Head: cfg.L2Head})
		require.NoError(t, err, "failed to create L2 client")
		l2DebugCl := &host.L2Source{L2Client: l2Client, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}

		return prefetcher.NewPrefetcher(logger, l1Cl, l1BlobFetcher, l2DebugCl, kv), nil
	})
	err = host.FaultProofProgram(t.Ctx(), env.log, programCfg, withInProcessPrefetcher)
	tryDumpTestFixture(t, err, t.Name(), env, programCfg)
}

type TestParam func(p *e2eutils.TestParams)

func NewTestParams(params ...TestParam) *e2eutils.TestParams {
	dfault := actions.DefaultRollupTestParams
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

type DeployParam func(p *e2eutils.DeployParams)

func NewDeployParams(t actions.Testing, params ...DeployParam) *e2eutils.DeployParams {
	dfault := e2eutils.MakeDeployParams(t, NewTestParams())
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

type BatcherCfgParam func(c *actions.BatcherCfg)

func NewBatcherCfg(params ...BatcherCfgParam) *actions.BatcherCfg {
	dfault := &actions.BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		DataAvailabilityType: batcherFlags.BlobsType,
	}
	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

type OpProgramCfgParam func(p *config.Config)

func NewOpProgramCfg(
	t actions.Testing,
	env *L2FaultProofEnv,
	fi *FixtureInputs,
	params ...OpProgramCfgParam,
) *config.Config {
	dfault := config.NewConfig(env.sd.RollupCfg, env.sd.L2Cfg.Config, fi.L1Head, fi.L2Head, fi.L2OutputRoot, fi.L2Claim, fi.L2BlockNumber)

	if dumpFixtures {
		dfault.DataDir = t.TempDir()
		dfault.DataFormat = hostTypes.DataFormatPebble
	}

	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}
