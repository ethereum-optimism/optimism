package proofs

import (
	"math/rand"
	"testing"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/actions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	hostTypes "github.com/ethereum-optimism/optimism/op-program/host/types"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
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

func NewL2FaultProofEnv(t actions.Testing, tp *e2eutils.TestParams, dp *e2eutils.DeployParams, batcherCfg *actions.BatcherCfg) *L2FaultProofEnv {
	log := testlog.Logger(t, log.LvlDebug)
	sd := e2eutils.Setup(t, dp, actions.DefaultAlloc)

	jwtPath := e2eutils.WriteDefaultJWT(t)
	cfg := &actions.SequencerCfg{VerifierCfg: *actions.DefaultVerifierCfg()}

	miner := actions.NewL1Miner(t, log.New("role", "l1-miner"), sd.L1Cfg)

	l1F, err := sources.NewL1Client(miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindStandard))
	require.NoError(t, err)
	engine := actions.NewL2Engine(t, log.New("role", "sequencer-engine"), sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath, actions.EngineWithP2P())
	l2Cl, err := sources.NewEngineClient(engine.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	sequencer := actions.NewL2Sequencer(t, log.New("role", "sequencer"), l1F, miner.BlobStore(), altda.Disabled, l2Cl, sd.RollupCfg, 0, cfg.InteropBackend)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	sequencer.ActL2PipelineFull(t)
	engCl := engine.EngineClient(t, sd.RollupCfg)

	// Set the batcher key to the secret key of the batcher
	batcherCfg.BatcherKey = dp.Secrets.Batcher
	batcher := actions.NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engCl)

	addresses := e2eutils.CollectAddresses(sd, dp)
	cl := engine.EthClient()
	l2UserEnv := &actions.BasicUserEnv[*actions.L2Bindings]{
		EthCl:          cl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       actions.NewL2Bindings(t, cl, engine.GethClient()),
	}
	alice := actions.NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(0xa57b)))
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

func (env *L2FaultProofEnv) RunFaultProofProgram(t actions.Testing, gt *testing.T, l2ClaimBlockNum uint64, fixtureInputParams ...FixtureInputParam) error {
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
	err = host.FaultProofProgram(t.Ctx(), env.log, programCfg)
	tryDumpTestFixture(gt, err, t.Name(), env, programCfg)
	return err
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

	// Set up in-process L1 sources
	dfault.L1ProcessSource = env.miner.L1Client(t, env.sd.RollupCfg)
	dfault.L1BeaconProcessSource = env.miner.BlobStore()

	// Set up in-process L2 source
	l2ClCfg := sources.L2ClientDefaultConfig(env.sd.RollupCfg, true)
	l2RPC := env.engine.RPCClient()
	l2Client, err := host.NewL2Client(l2RPC, env.log, nil, &host.L2ClientConfig{L2ClientConfig: l2ClCfg, L2Head: fi.L2Head})
	require.NoError(t, err, "failed to create L2 client")
	l2DebugCl := &host.L2Source{L2Client: l2Client, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}
	dfault.L2ProcessSource = l2DebugCl

	if dumpFixtures {
		dfault.DataDir = t.TempDir()
		dfault.DataFormat = hostTypes.DataFormatPebble
	}

	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}
