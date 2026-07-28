package derivation

import (
	"math/big"
	"math/rand"
	"testing"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	upgradesHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/upgrades/helpers"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSystemConfigBatchType run each system config-related test case in singular batch mode and span batch mode.
func TestSystemConfigBatchType(t *testing.T) {
	tests := []struct {
		name string
		f    func(gt *testing.T, deltaTimeOffset *hexutil.Uint64)
	}{
		{"BatcherKeyRotation", BatcherKeyRotation},
		{"GPODefaultParams", GPODefaultParams},
		{"GasLimitChange", GasLimitChange},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SingularBatch", func(t *testing.T) {
			test.f(t, nil)
		})
	}

	deltaTimeOffset := hexutil.Uint64(0)
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SpanBatch", func(t *testing.T) {
			test.f(t, &deltaTimeOffset)
		})
	}
}

// BatcherKeyRotation tests that batcher A can operate, then be replaced with batcher B, then ignore old batcher A,
// and that the change to batcher B is reverted properly upon reorg of L1.
func BatcherKeyRotation(gt *testing.T, deltaTimeOffset *hexutil.Uint64) {
	t := actionsHelpers.NewDefaultTesting(gt)

	dp := e2eutils.MakeDeployParams(t, actionsHelpers.DefaultRollupTestParams())
	dp.DeployConfig.L2BlockTime = 2
	upgradesHelpers.ApplyDeltaTimeOffset(dp, deltaTimeOffset)
	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	miner, seqEngine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	_, verifier := actionsHelpers.SetupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{})
	rollupSeqCl := sequencer.RollupClient()

	// the default batcher
	batcherA := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, actionsHelpers.DefaultBatcherCfg(dp),
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	// a batcher with a new key
	altCfg := *actionsHelpers.DefaultBatcherCfg(dp)
	altCfg.BatcherKey = dp.Secrets.Bob
	batcherB := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, &altCfg,
		rollupSeqCl, miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)

	// build a L1 chain, and then L2 chain, for batcher A to submit
	miner.ActEmptyBlock(t)
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	batcherA.ActSubmitAll(t)

	// include the batch data on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// sync from L1
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, uint64(2), sequencer.L2Safe().L1Origin.Number, "l2 chain with new L1 origins")
	require.Equal(t, sequencer.L2Safe(), verifier.L2Safe(), "fully synced verifier")

	sysCfgContract, err := bindings.NewSystemConfig(sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.Deployer, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	owner, err := sysCfgContract.Owner(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, dp.Addresses.Deployer, owner, "system config owner mismatch")

	// Change the batch sender key to Bob!
	tx, err := sysCfgContract.SetBatcherHash(sysCfgOwner, eth.AddressAsLeftPaddedHash(dp.Addresses.Bob))
	require.NoError(t, err)
	t.Logf("batcher changes in L1 tx %s", tx.Hash())
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)

	receipt, err := miner.EthClient().TransactionReceipt(t.Ctx(), tx.Hash())
	require.NoError(t, err)

	cfgChangeL1BlockNum := bigs.Uint64Strict(miner.L1Chain().CurrentBlock().Number)
	require.Equal(t, cfgChangeL1BlockNum, bigs.Uint64Strict(receipt.BlockNumber))

	// sequence L2 blocks, and submit with new batcher
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	batcherB.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Bob)(t)
	miner.ActL1EndBlock(t)

	// check that the first L2 payload that adopted the L1 block with the batcher key change
	// indeed changed the batcher key in the system config
	engCl := seqEngine.EngineClient(t, sd.RollupCfg)
	// 12 new L2 blocks: 5 with origin before L1 block with batch, 6 with origin of L1 block
	// with batch, 1 with new origin that changed the batcher
	for i := 0; i <= 12; i++ {
		envelope, err := engCl.PayloadByNumber(t.Ctx(), sequencer.L2Safe().Number+uint64(i))
		require.NoError(t, err)
		ref, err := derive.PayloadToBlockRef(sd.RollupCfg, envelope.ExecutionPayload)
		require.NoError(t, err)
		if i < 6 {
			require.Equal(t, ref.L1Origin.Number, cfgChangeL1BlockNum-2)
			require.Equal(t, ref.SequenceNumber, uint64(i))
		} else if i < 12 {
			require.Equal(t, ref.L1Origin.Number, cfgChangeL1BlockNum-1)
			require.Equal(t, ref.SequenceNumber, uint64(i-6))
		} else {
			require.Equal(t, ref.L1Origin.Number, cfgChangeL1BlockNum)
			require.Equal(t, ref.SequenceNumber, uint64(0), "first L2 block with this origin")
			sysCfg, err := derive.PayloadToSystemConfig(sd.RollupCfg, envelope.ExecutionPayload)
			require.NoError(t, err)
			require.Equal(t, dp.Addresses.Bob, sysCfg.BatcherAddr, "bob should be batcher now")
		}
	}

	// sync from L1
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sequencer.L2Safe().L1Origin.Number, uint64(4), "safe l2 chain with two new l1 blocks")
	require.Equal(t, sequencer.L2Safe(), verifier.L2Safe(), "fully synced verifier")

	// now try to build a new L1 block, and corresponding L2 blocks, and submit with the old batcher
	before := sequencer.L2Safe()
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	batcherA.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	// check that the data submitted by the old batcher is ignored
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sequencer.L2Safe(), before, "no new safe l1 chain")
	require.Equal(t, verifier.L2Safe(), before, "verifier is ignoring old batcher too")

	// now submit with the new batcher
	batcherB.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Bob)(t)
	miner.ActL1EndBlock(t)

	// not ignored now with new batcher
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.NotEqual(t, sequencer.L2Safe(), before, "new safe l1 chain")
	require.NotEqual(t, verifier.L2Safe(), before, "verifier is not ignoring new batcher")

	// twist: reorg L1, including the batcher key change
	miner.ActL1RewindDepth(5)(t)
	for i := 0; i < 6; i++ { // build some empty blocks so the reorg is picked up
		miner.ActEmptyBlock(t)
	}
	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, uint64(2), sequencer.L2Safe().L1Origin.Number, "l2 safe is first batch submission with original batcher")
	require.Equal(t, uint64(3), sequencer.L2Unsafe().L1Origin.Number, "l2 unsafe l1 origin is the block that included the first batch")
	require.Equal(t, sequencer.L2Safe(), verifier.L2Safe(), "verifier safe head check")
	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Unsafe(), "verifier unsafe head check")

	// without building L2 chain for the new L1 blocks yet, just batch-submit the unsafe part
	batcherA.ActL2BatchBuffer(t) // forces the buffer state to handle the rewind, before we loop with ActSubmitAll
	batcherA.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, sequencer.L2Safe(), sequencer.L2Unsafe(), "all L2 blocks are safe now")
	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Unsafe(), "verifier synced")

	// and see if we can go past it, with new L2 blocks
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	batcherA.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	sequencer.ActL2PipelineFull(t)
	verifier.ActL2PipelineFull(t)
	require.Equal(t, uint64(3+6+1), verifier.L2Safe().L1Origin.Number, "sync new L1 chain, while key change is reorged out")
	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Unsafe(), "verifier synced")
}

// GPODefaultParams tests that the pre-Ecotone L1 data fee charged to an L2 transaction is
// computed from the genesis gas config: op-node derives the fee parameters, the sequencer
// applies them through the L1 attributes transaction, and they surface on the receipt.
func GPODefaultParams(gt *testing.T, deltaTimeOffset *hexutil.Uint64) {
	t := actionsHelpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, actionsHelpers.DefaultRollupTestParams())
	upgradesHelpers.ApplyDeltaTimeOffset(dp, deltaTimeOffset)

	// activating Delta only, not Ecotone and further:
	// the GPO change assertions here all apply only for the Delta transition.
	// TestGPOParamsChangeEcotone covers the Ecotone equivalent.
	dp.DeployConfig.L2GenesisEcotoneTimeOffset = nil
	dp.DeployConfig.L2GenesisFjordTimeOffset = nil
	dp.DeployConfig.L2GenesisGraniteTimeOffset = nil
	dp.DeployConfig.L2GenesisHoloceneTimeOffset = nil

	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	miner, seqEngine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)
	alice := actionsHelpers.NewBasicUser[any](log, dp.Secrets.Alice, rand.New(rand.NewSource(1234)))
	alice.SetUserEnv(&actionsHelpers.BasicUserEnv[any]{
		EthCl:  seqEngine.EthClient(),
		Signer: types.LatestSigner(sd.L2Cfg.Config),
	})

	sequencer.ActL2PipelineFull(t)

	// new L1 block, with new L2 chain
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	basefee := miner.L1Chain().CurrentBlock().BaseFee

	// alice makes a L2 tx, sequencer includes it
	alice.ActResetTxOpts(t)
	alice.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	receipt := alice.LastTxReceipt(t)
	require.Equal(t, basefee, receipt.L1GasPrice, "L1 gas price matches basefee of L1 origin")
	require.NotZero(t, receipt.L1GasUsed, "L2 tx uses L1 data")
	require.Equal(t,
		new(big.Float).Mul(
			new(big.Float).SetInt(basefee),
			new(big.Float).Mul(new(big.Float).SetInt(receipt.L1GasUsed), receipt.FeeScalar),
		),
		new(big.Float).SetInt(receipt.L1Fee), "fee field in receipt matches gas used times scalar times basefee")
	// receipt.L1GasUsed includes the overhead already, so subtract that before passing it into the L1 cost func
	l1Cost := types.L1Cost(bigs.Uint64Strict(receipt.L1GasUsed)-2100, basefee, big.NewInt(2100), big.NewInt(1000_000))
	require.Equal(t, l1Cost, receipt.L1Fee, "L1 fee is computed with standard GPO params")
	require.Equal(t, "1", receipt.FeeScalar.String(), "1000_000 divided by 6 decimals = float(1)")
}

// TestGPOParamsChangeEcotone tests that the fee scalars can be updated with setGasConfigEcotone,
// and that the L1 data fee charged to an L2 transaction is applied correctly before, during and
// after the L2 chain adopts the L1 origin carrying the update. This is the Ecotone counterpart of
// GPODefaultParams: it is not parameterized over batch type, because Ecotone implies Delta and so
// the singular-batch case is unreachable.
func TestGPOParamsChangeEcotone(gt *testing.T) {
	t := actionsHelpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, actionsHelpers.DefaultRollupTestParams())
	log := testlog.Logger(t, log.LevelDebug)

	// Ecotone is the latest active fork, so the L1 cost function under test is the Ecotone one.
	dp.DeployConfig.ActivateForkAtGenesis(forks.Ecotone)
	require.NoError(t, dp.DeployConfig.Check(log), "must have valid config")

	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	miner, seqEngine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)
	batcher := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, actionsHelpers.DefaultBatcherCfg(dp),
		sequencer.RollupClient(), miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	alice := actionsHelpers.NewBasicUser[any](log, dp.Secrets.Alice, rand.New(rand.NewSource(1234)))
	alice.SetUserEnv(&actionsHelpers.BasicUserEnv[any]{
		EthCl:  seqEngine.EthClient(),
		Signer: types.LatestSigner(sd.L2Cfg.Config),
	})

	sequencer.ActL2PipelineFull(t)

	// new L1 block, with new L2 chain
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// alice makes a L2 tx, sequencer includes it, and the genesis scalars price its L1 data
	tx := aliceTx(t, dp, alice, sequencer, seqEngine)
	genesisScalars, err := sd.RollupCfg.Genesis.SystemConfig.EcotoneScalars()
	require.NoError(t, err)
	requireEcotoneL1Fee(t, alice.LastTxReceipt(t), tx, genesisScalars)

	// confirm L2 chain on L1
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	sysCfgContract, err := bindings.NewSystemConfig(sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.Deployer, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	// basefee scalar changes from 1_000_000 (default) to 2_300_000, and the blobbasefee scalar from
	// 0 to 800_000, e.g. if the system operator starts paying for data availability with blobs
	newScalars := eth.EcotoneScalars{BaseFeeScalar: 2_300_000, BlobBaseFeeScalar: 800_000}
	_, err = sysCfgContract.SetGasConfigEcotone(sysCfgOwner, newScalars.BaseFeeScalar, newScalars.BlobBaseFeeScalar)
	require.NoError(t, err)

	// include the GPO change tx in L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)

	// build empty L2 chain, up to but excluding the L2 block with the L1 origin that processes the GPO change
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadExcl(t)

	engCl := seqEngine.EngineClient(t, sd.RollupCfg)
	envelope, err := engCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	sysCfg, err := derive.PayloadToSystemConfig(sd.RollupCfg, envelope.ExecutionPayload)
	require.NoError(t, err)
	// Compare the scalars rather than the whole config: `initialize` calls `_setGasConfigEcotone`,
	// so the derived config is already on the versioned scalar encoding with a zeroed overhead,
	// while `Genesis.SystemConfig` still carries the pre-Ecotone encoding from the deploy config.
	require.Equal(t, eth.Bytes32(eth.EncodeScalar(genesisScalars)), sysCfg.Scalar, "still have the genesis fee scalars before we adopt the L1 block with GPO change")

	// Now alice makes another transaction, which gets included in the same block that adopts the L1 origin with GPO change
	tx = aliceTx(t, dp, alice, sequencer, seqEngine)

	envelope, err = engCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	sysCfg, err = derive.PayloadToSystemConfig(sd.RollupCfg, envelope.ExecutionPayload)
	require.NoError(t, err)
	require.Equal(t, eth.Bytes32(eth.EncodeScalar(newScalars)), sysCfg.Scalar, "scalar changed")
	// setGasConfigEcotone emits a zero overhead, and op-node zeroes it after Ecotone regardless
	require.Equal(t, eth.Bytes32{}, sysCfg.Overhead, "overhead zeroed")

	requireEcotoneL1Fee(t, alice.LastTxReceipt(t), tx, newScalars)

	// build more L2 blocks, with new L1 origin
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	// and Alice makes a tx again
	tx = aliceTx(t, dp, alice, sequencer, seqEngine)

	// and verify the new GPO params are persistent, even though the L1 origin and L2 chain have progressed
	requireEcotoneL1Fee(t, alice.LastTxReceipt(t), tx, newScalars)
}

// aliceTx has alice send one L2 transaction and the sequencer include it in a fresh L2 block.
func aliceTx(
	t actionsHelpers.Testing,
	dp *e2eutils.DeployParams,
	alice *actionsHelpers.BasicUser[any],
	sequencer *actionsHelpers.L2Sequencer,
	seqEngine *actionsHelpers.L2Engine,
) *types.Transaction {
	alice.ActResetTxOpts(t)
	tx := alice.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)
	return tx
}

// requireEcotoneL1Fee checks that the L1 data fee on the receipt is priced with the given scalars,
// recomputing it from the fee parameters the receipt itself reports.
func requireEcotoneL1Fee(
	t actionsHelpers.Testing,
	receipt *types.Receipt,
	tx *types.Transaction,
	scalars eth.EcotoneScalars,
) {
	require.NotNil(t, receipt.L1BaseFeeScalar, "Ecotone receipt reports the basefee scalar")
	require.NotNil(t, receipt.L1BlobBaseFeeScalar, "Ecotone receipt reports the blobbasefee scalar")
	require.Equal(t, uint64(scalars.BaseFeeScalar), *receipt.L1BaseFeeScalar, "basefee scalar on receipt")
	require.Equal(t, uint64(scalars.BlobBaseFeeScalar), *receipt.L1BlobBaseFeeScalar, "blobbasefee scalar on receipt")

	l1Cost := opfees.L1CostEcotone(opfees.TxRollupCostData(tx), opfees.L1FeeParams{
		L1BaseFee:     receipt.L1GasPrice,
		L1BlobBaseFee: receipt.L1BlobBaseFee,
		BaseFeeScalar: new(big.Int).SetUint64(uint64(scalars.BaseFeeScalar)),
		BlobFeeScalar: new(big.Int).SetUint64(uint64(scalars.BlobBaseFeeScalar)),
	})
	require.Equal(t, l1Cost, receipt.L1Fee, "L1 fee is computed with the expected Ecotone scalars")
}

// GasLimitChange tests that the gas limit can be configured to L1,
// and that the L2 changes the gas limit instantly at the exact block that adopts the L1 origin with
// the gas limit change event. And checks if a verifier node can reproduce the same gas limit change.
func GasLimitChange(gt *testing.T, deltaTimeOffset *hexutil.Uint64) {
	t := actionsHelpers.NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, actionsHelpers.DefaultRollupTestParams())
	upgradesHelpers.ApplyDeltaTimeOffset(dp, deltaTimeOffset)
	sd := e2eutils.Setup(t, dp, actionsHelpers.DefaultAlloc)
	log := testlog.Logger(t, log.LevelDebug)
	miner, seqEngine, sequencer := actionsHelpers.SetupSequencerTest(t, sd, log)
	batcher := actionsHelpers.NewL2Batcher(log, sd.RollupCfg, actionsHelpers.DefaultBatcherCfg(dp),
		sequencer.RollupClient(), miner.EthClient(), seqEngine.EthClient(), seqEngine.EngineClient(t, sd.RollupCfg))

	sequencer.ActL2PipelineFull(t)
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	oldGasLimit := seqEngine.L2Chain().CurrentBlock().GasLimit
	require.Equal(t, oldGasLimit, uint64(dp.DeployConfig.L2GenesisBlockGasLimit))

	// change gas limit on L1 to triple what it was
	sysCfgContract, err := bindings.NewSystemConfig(sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.Deployer, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	_, err = sysCfgContract.SetGasLimit(sysCfgOwner, oldGasLimit*3)
	require.NoError(t, err)

	// include the gaslimit update on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)

	// build to latest L1, excluding the block that adopts the L1 block with the gaslimit change
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadExcl(t)

	require.Equal(t, oldGasLimit, seqEngine.L2Chain().CurrentBlock().GasLimit)
	require.Equal(t, uint64(1), sequencer.SyncStatus().UnsafeL2.L1Origin.Number)

	// now include the L1 block with the gaslimit change, and see if it changes as expected
	sequencer.ActBuildToL1Head(t)
	require.Equal(t, oldGasLimit*3, seqEngine.L2Chain().CurrentBlock().GasLimit)
	require.Equal(t, uint64(2), sequencer.SyncStatus().UnsafeL2.L1Origin.Number)

	// now submit all this to L1, and see if a verifier can sync and reproduce it
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	_, verifier := actionsHelpers.SetupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg), miner.BlobStore(), &sync.Config{})
	verifier.ActL2PipelineFull(t)

	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Safe(), "verifier stays in sync, even with gaslimit changes")
}
