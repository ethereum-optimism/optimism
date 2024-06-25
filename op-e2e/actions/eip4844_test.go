package actions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func setupEIP4844Test(t Testing, log log.Logger) (*e2eutils.SetupData, *e2eutils.DeployParams, *L1Miner, *L2Sequencer, *L2Engine, *L2Verifier, *L2Engine) {
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	genesisActivation := hexutil.Uint64(0)
	dp.DeployConfig.L1CancunTimeOffset = &genesisActivation
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &genesisActivation
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &genesisActivation
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &genesisActivation

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	verifEngine, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{})
	return sd, dp, miner, sequencer, seqEngine, verifier, verifEngine
}

func setupBatcher(t Testing, log log.Logger, sd *e2eutils.SetupData, dp *e2eutils.DeployParams, miner *L1Miner,
	sequencer *L2Sequencer, engine *L2Engine, daType batcherFlags.DataAvailabilityType,
) *L2Batcher {
	return NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		BatcherKey:           dp.Secrets.Batcher,
		DataAvailabilityType: daType,
	}, sequencer.RollupClient(), miner.EthClient(), engine.EthClient(), engine.EngineClient(t, sd.RollupCfg))
}

func TestEIP4844DataAvailability(gt *testing.T) {
	t := NewDefaultTesting(gt)

	log := testlog.Logger(t, log.LevelDebug)
	sd, dp, miner, sequencer, seqEngine, verifier, _ := setupEIP4844Test(t, log)

	batcher := setupBatcher(t, log, sd, dp, miner, sequencer, seqEngine, batcherFlags.BlobsType)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 block
	miner.ActEmptyBlock(t)
	// finalize it, so the L1 geth blob pool doesn't log errors about missing finality
	miner.ActL1SafeNext(t)
	miner.ActL1FinalizeNext(t)

	// Create L2 blocks, and reference the L1 head as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks
	batcher.ActSubmitAll(t)
	batchTx := batcher.LastSubmitted
	require.Equal(t, uint8(types.BlobTxType), batchTx.Type(), "batch tx must be blob-tx")

	// new L1 block with L2 batch
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")
}

func TestEIP4844MultiBlobs(gt *testing.T) {
	t := NewDefaultTesting(gt)

	log := testlog.Logger(t, log.LevelDebug)
	sd, dp, miner, sequencer, seqEngine, verifier, _ := setupEIP4844Test(t, log)

	batcher := setupBatcher(t, log, sd, dp, miner, sequencer, seqEngine, batcherFlags.BlobsType)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 block
	miner.ActEmptyBlock(t)
	// finalize it, so the L1 geth blob pool doesn't log errors about missing finality
	miner.ActL1SafeNext(t)
	miner.ActL1FinalizeNext(t)

	// Create L2 blocks, and reference the L1 head as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks
	batcher.ActSubmitAllMultiBlobs(t, 6)
	batchTx := batcher.LastSubmitted
	require.Equal(t, uint8(types.BlobTxType), batchTx.Type(), "batch tx must be blob-tx")
	require.Len(t, batchTx.BlobTxSidecar().Blobs, 6)

	// new L1 block with L2 batch
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")
}

func TestEIP4844DataAvailabilitySwitch(gt *testing.T) {
	t := NewDefaultTesting(gt)

	log := testlog.Logger(t, log.LevelDebug)
	sd, dp, miner, sequencer, seqEngine, verifier, _ := setupEIP4844Test(t, log)

	oldBatcher := setupBatcher(t, log, sd, dp, miner, sequencer, seqEngine, batcherFlags.CalldataType)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 block
	miner.ActEmptyBlock(t)
	// finalize it, so the L1 geth blob pool doesn't log errors about missing finality
	miner.ActL1SafeNext(t)
	miner.ActL1FinalizeNext(t)

	// Create L2 blocks, and reference the L1 head as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks, with legacy calldata DA
	oldBatcher.ActSubmitAll(t)
	batchTx := oldBatcher.LastSubmitted
	require.Equal(t, uint8(types.DynamicFeeTxType), batchTx.Type(), "batch tx must be eip1559 tx")

	// new L1 block with L2 batch
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	miner.ActL1EndBlock(t)

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")

	newBatcher := setupBatcher(t, log, sd, dp, miner, sequencer, seqEngine, batcherFlags.BlobsType)

	// build empty L1 block
	miner.ActEmptyBlock(t)

	// Create L2 blocks, and reference the L1 head as origin
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// submit all new L2 blocks, now with Blobs DA!
	newBatcher.ActSubmitAll(t)
	batchTx = newBatcher.LastSubmitted

	// new L1 block with L2 batch
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(batchTx.Hash())(t)
	miner.ActL1EndBlock(t)

	require.Equal(t, uint8(types.BlobTxType), batchTx.Type(), "batch tx must be blob-tx")

	// verifier picks up the L2 chain that was submitted
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, verifier.L2Safe(), sequencer.L2Unsafe(), "verifier syncs from sequencer via L1")
	require.NotEqual(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "sequencer has not processed L1 yet")
}
