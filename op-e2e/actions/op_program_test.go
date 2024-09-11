package actions

import (
	"context"
	"math/rand"
	"os"
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-program/host"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var dumpFixtures = false

func init() {
	if os.Getenv("OP_E2E_DUMP_FIXTURES") == "1" {
		dumpFixtures = true
	}
}

// L2FaultProofEnv is a test harness for a fault provable L2 chain.
type L2FaultProofEnv struct {
	log       log.Logger
	batcher   *L2Batcher
	sequencer *L2Sequencer
	engine    *L2Engine
	engCl     *sources.EngineClient
	sd        *e2eutils.SetupData
	dp        *e2eutils.DeployParams
	miner     *L1Miner
	alice     *CrossLayerUser
}

type TestParam func(p *e2eutils.TestParams)

func NewL2FaultProofEnv(t Testing, batcherCfg *BatcherCfg, params ...TestParam) *L2FaultProofEnv {
	p := defaultRollupTestParams
	for _, apply := range params {
		apply(p)
	}
	log := testlog.Logger(t, log.LvlDebug)
	dp := e2eutils.MakeDeployParams(t, p)

	// Enable Cancun on L1 & Granite on L2 at genesis
	// TODO: Hardfork configurability.
	genesisBlock := hexutil.Uint64(0)
	dp.DeployConfig.L1CancunTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisFjordTimeOffset = &genesisBlock
	dp.DeployConfig.L2GenesisGraniteTimeOffset = &genesisBlock
	sd := e2eutils.Setup(t, dp, defaultAlloc)

	miner, engine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	sequencer.ActL2PipelineFull(t)
	engCl := engine.EngineClient(t, sd.RollupCfg)

	// Set the batcher key to the secret key of the batcher
	batcherCfg.BatcherKey = dp.Secrets.Batcher
	batcher := NewL2Batcher(log, sd.RollupCfg, batcherCfg, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engCl)

	addresses := e2eutils.CollectAddresses(sd, dp)
	cl := engine.EthClient()
	l2UserEnv := &BasicUserEnv[*L2Bindings]{
		EthCl:          cl,
		Signer:         types.LatestSigner(sd.L2Cfg.Config),
		AddressCorpora: addresses,
		Bindings:       NewL2Bindings(t, cl, engine.GethClient()),
	}
	alice := NewCrossLayerUser(log, dp.Secrets.Alice, rand.New(rand.NewSource(0xa57b)))
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

type BatcherCfgParam func(c *BatcherCfg)

func NewBatcherCfg(params ...BatcherCfgParam) *BatcherCfg {
	dfault := &BatcherCfg{
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
	t Testing,
	env *L2FaultProofEnv,
	l1Head common.Hash,
	l2Head common.Hash,
	l2OutputRoot common.Hash,
	l2Claim common.Hash,
	l2ClaimBlockNum uint64,
	params ...OpProgramCfgParam,
) *config.Config {
	dfault := config.NewConfig(env.sd.RollupCfg, env.sd.L2Cfg.Config, l1Head, l2Head, l2OutputRoot, l2Claim, l2ClaimBlockNum)

	// Set up in-process L1 sources
	dfault.L1ProcessSource = env.miner.L1Client(t, env.sd.RollupCfg)
	dfault.L1BeaconProcessSource = env.miner.blobStore

	// Set up in-process L2 source
	l2ClCfg := sources.L2ClientDefaultConfig(env.sd.RollupCfg, true)
	l2RPC := env.engine.RPCClient()
	l2Client, err := host.NewL2Client(l2RPC, env.log, nil, &host.L2ClientConfig{L2ClientConfig: l2ClCfg, L2Head: l2Head})
	require.NoError(t, err, "failed to create L2 client")
	l2DebugCl := &host.L2Source{L2Client: l2Client, DebugClient: sources.NewDebugClient(l2RPC.CallContext)}
	dfault.L2ProcessSource = l2DebugCl

	for _, apply := range params {
		apply(dfault)
	}
	return dfault
}

func Test_ProgramAction_SimpleEmptyChain_HonestClaim_Granite(gt *testing.T) {
	t := NewDefaultTesting(gt)
	env := NewL2FaultProofEnv(t, NewBatcherCfg())

	// Build an empty block on L2
	env.sequencer.ActL2StartBlock(t)
	env.sequencer.ActL2EndBlock(t)

	// Instruct the batcher to submit the block to L1, and include the transaction.
	env.batcher.ActSubmitAll(t)
	env.miner.ActL1StartBlock(12)(t)
	env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
	env.miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.miner.ActL1SafeNext(t)
	env.miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.sequencer.ActL1HeadSignal(t)
	env.sequencer.ActL2PipelineFull(t)

	l1Head := env.miner.l1Chain.CurrentBlock()
	l2SafeHead := env.engine.l2Chain.CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), l1Head.Number.Uint64())
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), l2SafeHead.Number.Uint64())

	// Fetch the pre and post output roots for the fault proof.
	preRoot, err := env.sequencer.RollupClient().OutputAtBlock(context.Background(), l2SafeHead.Number.Uint64() - 1)
	require.NoError(t, err)
	claimRoot, err := env.sequencer.RollupClient().OutputAtBlock(context.Background(), l2SafeHead.Number.Uint64())
	require.NoError(t, err)

	// Run the fault proof program from the state transition from L2 block 0 -> 1.
	programCfg := NewOpProgramCfg(
		t,
		env,
		l1Head.Hash(),
		preRoot.BlockRef.Hash,
		common.Hash(preRoot.OutputRoot),
		common.Hash(claimRoot.OutputRoot),
		l2SafeHead.Number.Uint64(),
	)
	err = host.FaultProofProgram(context.Background(), env.log, programCfg)
	require.NoError(t, err)
}

func Test_ProgramAction_SimpleEmptyChain_JunkClaim_Granite(gt *testing.T) {
	t := NewDefaultTesting(gt)
	env := NewL2FaultProofEnv(t, NewBatcherCfg())

	// Build an empty block on L2
	env.sequencer.ActL2StartBlock(t)
	env.sequencer.ActL2EndBlock(t)

	// Instruct the batcher to submit the block to L1, and include the transaction.
	env.batcher.ActSubmitAll(t)
	env.miner.ActL1StartBlock(12)(t)
	env.miner.ActL1IncludeTxByHash(env.batcher.LastSubmitted.Hash())(t)
	env.miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.miner.ActL1SafeNext(t)
	env.miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.sequencer.ActL1HeadSignal(t)
	env.sequencer.ActL2PipelineFull(t)

	l1Head := env.miner.l1Chain.CurrentBlock()
	l2SafeHead := env.engine.l2Chain.CurrentSafeBlock()

	// Ensure there is only 1 block on L1.
	require.Equal(t, uint64(1), l1Head.Number.Uint64())
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(1), l2SafeHead.Number.Uint64())

	// Fetch the pre and post output roots for the fault proof.
	preRoot, err := env.sequencer.RollupClient().OutputAtBlock(context.Background(), l2SafeHead.Number.Uint64() - 1)
	require.NoError(t, err)

	// Run the fault proof program from the state transition from L2 block 0 -> 1, with a junk claim.
	programCfg := NewOpProgramCfg(
		t,
		env,
		l1Head.Hash(),
		preRoot.BlockRef.Hash,
		common.Hash(preRoot.OutputRoot),
		common.HexToHash("0xdeadbeef"),
		l2SafeHead.Number.Uint64(),
	)
	err = host.FaultProofProgram(context.Background(), env.log, programCfg)
	require.Error(t, err)
}
