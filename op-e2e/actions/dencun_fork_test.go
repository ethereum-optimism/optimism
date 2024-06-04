package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestDencunL1ForkAfterGenesis(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(24)
	dp.DeployConfig.L1CancunTimeOffset = &offset
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	l1Head := miner.l1Chain.CurrentBlock()
	require.False(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun not active yet")
	require.Nil(t, l1Head.ExcessBlobGas, "Cancun blob gas not in header")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks, crossing the fork boundary
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t) // Cancun activates here
	miner.ActEmptyBlock(t)
	// verify Cancun is active
	l1Head = miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// build L2 chain up to and including L2 blocks referencing Cancun L1 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)
	// verify verifier accepted Cancun L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes Cancun headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}

func TestDencunL1ForkAtGenesis(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(0)
	dp.DeployConfig.L1CancunTimeOffset = &offset
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	_, _, miner, sequencer, _, verifier, _, batcher := setupReorgTestActors(t, dp, sd, log)

	l1Head := miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active at genesis")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build empty L1 blocks
	miner.ActL1SetFeeRecipient(common.Address{'A', 0})
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)

	// verify Cancun is still active
	l1Head = miner.l1Chain.CurrentBlock()
	require.True(t, sd.L1Cfg.Config.IsCancun(l1Head.Number, l1Head.Time), "Cancun active")
	require.NotNil(t, l1Head.ExcessBlobGas, "Cancun blob gas in header")

	// build L2 chain
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	miner.ActL1StartBlock(12)(t)
	batcher.ActSubmitAll(t)
	miner.ActL1IncludeTx(batcher.batcherAddr)(t)
	miner.ActL1EndBlock(t)

	// sync verifier
	verifier.ActL1HeadSignal(t)
	verifier.ActL2PipelineFull(t)

	// verify verifier accepted Cancun L1 inputs
	require.Equal(t, l1Head.Hash(), verifier.SyncStatus().SafeL2.L1Origin.Hash, "verifier synced L1 chain that includes Cancun headers")
	require.Equal(t, sequencer.SyncStatus().UnsafeL2, verifier.SyncStatus().UnsafeL2, "verifier and sequencer agree")
}

func verifyPreEcotoneBlock(gt *testing.T, header *types.Header) {
	require.Nil(gt, header.ParentBeaconRoot)
	require.Nil(gt, header.ExcessBlobGas)
	require.Nil(gt, header.BlobGasUsed)
}

func verifyEcotoneBlock(gt *testing.T, header *types.Header) {
	require.NotNil(gt, header.ParentBeaconRoot)
	require.NotNil(gt, header.ExcessBlobGas)
	require.Equal(gt, *header.ExcessBlobGas, uint64(0))
	require.NotNil(gt, header.BlobGasUsed)
	require.Equal(gt, *header.BlobGasUsed, uint64(0))
}

func TestDencunL2ForkAfterGenesis(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)

	cancunOffset := hexutil.Uint64(0)
	dp.DeployConfig.L1CancunTimeOffset = &cancunOffset
	// This test wil fork on the second block
	offset := hexutil.Uint64(dp.DeployConfig.L2BlockTime * 2)
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	_, _, _, sequencer, engine, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Genesis block is pre-ecotone
	verifyPreEcotoneBlock(gt, engine.l2Chain.CurrentBlock())

	// Block before fork block
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)
	verifyPreEcotoneBlock(gt, engine.l2Chain.CurrentBlock())

	// Fork block is ecotone
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)
	verifyEcotoneBlock(gt, engine.l2Chain.CurrentBlock())

	// Blocks post fork have Ecotone properties
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)
	verifyEcotoneBlock(gt, engine.l2Chain.CurrentBlock())
}

func TestDencunL2ForkAtGenesis(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &offset
	dp.DeployConfig.L1CancunTimeOffset = &offset
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	_, _, _, sequencer, engine, verifier, _, _ := setupReorgTestActors(t, dp, sd, log)

	// start op-nodes
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// Genesis block has ecotone properties
	verifyEcotoneBlock(gt, engine.l2Chain.CurrentBlock())

	// Blocks post fork have Ecotone properties
	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)
	verifyEcotoneBlock(gt, engine.l2Chain.CurrentBlock())
}

func aliceSimpleBlobTx(t Testing, dp *e2eutils.DeployParams) *types.Transaction {
	txData := transactions.CreateEmptyBlobTx(true, dp.DeployConfig.L2ChainID)
	// Manual signer creation, so we can sign a blob tx on the chain,
	// even though we have disabled cancun signer support in Ecotone.
	signer := types.NewCancunSigner(txData.ChainID.ToBig())
	tx, err := types.SignNewTx(dp.Secrets.Alice, signer, txData)
	require.NoError(t, err, "must sign tx")
	return tx
}

func newEngine(t Testing, sd *e2eutils.SetupData, log log.Logger) *L2Engine {
	jwtPath := e2eutils.WriteDefaultJWT(t)
	return NewL2Engine(t, log, sd.L2Cfg, sd.RollupCfg.Genesis.L1, jwtPath)
}

// TestDencunBlobTxRPC tries to send a Blob tx to the L2 engine via RPC, it should not be accepted.
func TestDencunBlobTxRPC(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &offset
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	engine := newEngine(t, sd, log)
	cl := engine.EthClient()
	tx := aliceSimpleBlobTx(t, dp)
	err := cl.SendTransaction(context.Background(), tx)
	require.ErrorContains(t, err, "transaction type not supported")
}

// TestDencunBlobTxInTxPool tries to insert a blob tx directly into the tx pool, it should not be accepted.
func TestDencunBlobTxInTxPool(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &offset
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	engine := newEngine(t, sd, log)
	tx := aliceSimpleBlobTx(t, dp)
	errs := engine.eth.TxPool().Add([]*types.Transaction{tx}, true, true)
	require.ErrorContains(t, errs[0], "transaction type not supported")
}

// TestDencunBlobTxInclusion tries to send a Blob tx to the L2 engine, it should not be accepted.
func TestDencunBlobTxInclusion(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	offset := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisRegolithTimeOffset = &offset
	dp.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	dp.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)

	_, engine, sequencer := setupSequencerTest(t, sd, log)
	sequencer.ActL2PipelineFull(t)

	tx := aliceSimpleBlobTx(t, dp)

	sequencer.ActL2StartBlock(t)
	err := engine.engineApi.IncludeTx(tx, dp.Addresses.Alice)
	require.ErrorContains(t, err, "invalid L2 block (tx 1): failed to apply transaction to L2 block (tx 1): transaction type not supported")
}
