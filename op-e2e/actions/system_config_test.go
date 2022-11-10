package actions

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

// TestBatcherKeyRotation tests that batcher A can operate, then be replaced with batcher B, then ignore old batcher A,
// and that the change to batcher B is reverted properly upon reorg of L1.
func TestBatcherKeyRotation(gt *testing.T) {
	t := NewDefaultTesting(gt)

	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)
	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))
	rollupSeqCl := sequencer.RollupClient()

	// the default batcher
	batcherA := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient())

	// a batcher with a new key
	batcherB := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Bob,
	}, rollupSeqCl, miner.EthClient(), seqEngine.EthClient())

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

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.SysCfgOwner, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	// Change the batch sender key to Bob!
	tx, err := sysCfgContract.SetBatcherHash(sysCfgOwner, dp.Addresses.Bob.Hash())
	require.NoError(t, err)
	t.Logf("batcher changes in L1 tx %s", tx.Hash())
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.SysCfgOwner)(t)
	miner.ActL1EndBlock(t)
	cfgChangeL1BlockNum := miner.l1Chain.CurrentBlock().NumberU64()

	// sequence L2 blocks, and submit with new batcher
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	batcherB.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Bob)(t)
	miner.ActL1EndBlock(t)

	// check that the first L2 payload that adopted the L1 block with the batcher key change indeed changed the batcher key in the system config
	engCl := seqEngine.EngineClient(t, sd.RollupCfg)
	payload, err := engCl.PayloadByNumber(t.Ctx(), sequencer.L2Safe().Number+12) // 12 new L2 blocks: 5 with origin before L1 block with batch, 6 with origin of L1 block with batch, 1 with new origin that changed the batcher
	require.NoError(t, err)
	ref, err := derive.PayloadToBlockRef(payload, &sd.RollupCfg.Genesis)
	require.NoError(t, err)
	require.Equal(t, ref.L1Origin.Number, cfgChangeL1BlockNum, "L2 block with L1 origin that included config change")
	require.Equal(t, ref.SequenceNumber, uint64(0), "first L2 block with this origin")
	sysCfg, err := derive.PayloadToSystemConfig(payload, sd.RollupCfg)
	require.NoError(t, err)
	require.Equal(t, dp.Addresses.Bob, sysCfg.BatcherAddr, "bob should be batcher now")

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

// TestGPOParamsChange tests that the GPO params can be updated to adjust fees of L2 transactions,
// and that the L1 data fees to the L2 transaction are applied correctly before, during and after the GPO update in L2.
func TestGPOParamsChange(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), seqEngine.EthClient())

	alice := NewBasicUser[any](log, dp.Secrets.Alice, rand.New(rand.NewSource(1234)))
	alice.SetUserEnv(&BasicUserEnv[any]{
		EthCl:  seqEngine.EthClient(),
		Signer: types.LatestSigner(sd.L2Cfg.Config),
	})

	sequencer.ActL2PipelineFull(t)

	// new L1 block, with new L2 chain
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	basefee := miner.l1Chain.CurrentBlock().BaseFee()

	// alice makes a L2 tx, sequencer includes it
	alice.ActResetTxOpts(t)
	alice.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	receipt := alice.LastTxReceipt(t)
	require.Equal(t, basefee, receipt.L1GasPrice, "L1 gas price matches basefee of L1 origin")
	require.NotZero(t, receipt.L1GasUsed, "L2 tx uses L1 data")
	l1Cost := types.L1Cost(receipt.L1GasUsed.Uint64(), basefee, big.NewInt(2100), big.NewInt(1000_000))
	require.Equal(t, l1Cost, receipt.L1Fee, "L1 fee is computed with standard GPO params")
	require.Equal(t, "1", receipt.FeeScalar.String(), "1000_000 divided by 6 decimals = float(1)")

	// confirm L2 chain on L1
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	sysCfgContract, err := bindings.NewSystemConfig(sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.SysCfgOwner, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	// overhead changes from 2100 (default) to 1000
	// scalar changes from 1_000_000 (default) to 2_300_000
	// e.g. if system operator determines that l2 txs need to be more expensive, but small ones less
	_, err = sysCfgContract.SetGasConfig(sysCfgOwner, big.NewInt(1000), big.NewInt(2_300_000))
	require.NoError(t, err)

	// include the GPO change tx in L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.SysCfgOwner)(t)
	miner.ActL1EndBlock(t)
	basefeeGPOUpdate := miner.l1Chain.CurrentBlock().BaseFee()

	// build empty L2 chain, up to but excluding the L2 block with the L1 origin that processes the GPO change
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadExcl(t)

	engCl := seqEngine.EngineClient(t, sd.RollupCfg)
	payload, err := engCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	sysCfg, err := derive.PayloadToSystemConfig(payload, sd.RollupCfg)
	require.NoError(t, err)
	require.Equal(t, sd.RollupCfg.Genesis.SystemConfig, sysCfg, "still have genesis system config before we adopt the L1 block with GPO change")

	// Now alice makes another transaction, which gets included in the same block that adopts the L1 origin with GPO change
	alice.ActResetTxOpts(t)
	alice.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	payload, err = engCl.PayloadByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(t, err)
	sysCfg, err = derive.PayloadToSystemConfig(payload, sd.RollupCfg)
	require.NoError(t, err)
	require.Equal(t, eth.Bytes32(common.BigToHash(big.NewInt(1000))), sysCfg.Overhead, "overhead changed")
	require.Equal(t, eth.Bytes32(common.BigToHash(big.NewInt(2_300_000))), sysCfg.Scalar, "scalar changed")

	receipt = alice.LastTxReceipt(t)
	require.Equal(t, basefeeGPOUpdate, receipt.L1GasPrice, "L1 gas price matches basefee of L1 origin")
	require.NotZero(t, receipt.L1GasUsed, "L2 tx uses L1 data")
	l1Cost = types.L1Cost(receipt.L1GasUsed.Uint64(), basefeeGPOUpdate, big.NewInt(1000), big.NewInt(2_300_000))
	require.Equal(t, l1Cost, receipt.L1Fee, "L1 fee is computed with updated GPO params")
	require.Equal(t, "2.3", receipt.FeeScalar.String(), "2_300_000 divided by 6 decimals = float(2.3)")

	// build more L2 blocks, with new L1 origin
	miner.ActEmptyBlock(t)
	basefee = miner.l1Chain.CurrentBlock().BaseFee()
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	// and Alice makes a tx again
	alice.ActResetTxOpts(t)
	alice.ActMakeTx(t)
	sequencer.ActL2StartBlock(t)
	seqEngine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	// and verify the new GPO params are persistent, even though the L1 origin and L2 chain have progressed
	receipt = alice.LastTxReceipt(t)
	require.Equal(t, basefee, receipt.L1GasPrice, "L1 gas price matches basefee of L1 origin")
	require.NotZero(t, receipt.L1GasUsed, "L2 tx uses L1 data")
	l1Cost = types.L1Cost(receipt.L1GasUsed.Uint64(), basefee, big.NewInt(1000), big.NewInt(2_300_000))
	require.Equal(t, l1Cost, receipt.L1Fee, "L1 fee is computed with updated GPO params")
	require.Equal(t, "2.3", receipt.FeeScalar.String(), "2_300_000 divided by 6 decimals = float(2.3)")
}

// TestGasLimitChange tests that the gas limit can be configured to L1,
// and that the L2 changes the gas limit instantly at the exact block that adopts the L1 origin with
// the gas limit change event. And checks if a verifier node can reproduce the same gas limit change.
func TestGasLimitChange(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	miner, seqEngine, sequencer := setupSequencerTest(t, sd, log)
	batcher := NewL2Batcher(log, sd.RollupCfg, &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}, sequencer.RollupClient(), miner.EthClient(), seqEngine.EthClient())

	sequencer.ActL2PipelineFull(t)
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	oldGasLimit := seqEngine.l2Chain.CurrentBlock().GasLimit()
	require.Equal(t, oldGasLimit, uint64(dp.DeployConfig.L2GenesisBlockGasLimit))

	// change gas limit on L1 to triple what it was
	sysCfgContract, err := bindings.NewSystemConfig(sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)

	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(dp.Secrets.SysCfgOwner, sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	_, err = sysCfgContract.SetGasLimit(sysCfgOwner, oldGasLimit*3)
	require.NoError(t, err)

	// include the gaslimit update on L1
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.SysCfgOwner)(t)
	miner.ActL1EndBlock(t)

	// build to latest L1, excluding the block that adopts the L1 block with the gaslimit change
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1HeadExcl(t)

	require.Equal(t, oldGasLimit, seqEngine.l2Chain.CurrentBlock().GasLimit())
	require.Equal(t, uint64(1), sequencer.SyncStatus().UnsafeL2.L1Origin.Number)

	// now include the L1 block with the gaslimit change, and see if it changes as expected
	sequencer.ActBuildToL1Head(t)
	require.Equal(t, oldGasLimit*3, seqEngine.l2Chain.CurrentBlock().GasLimit())
	require.Equal(t, uint64(2), sequencer.SyncStatus().UnsafeL2.L1Origin.Number)

	// now submit all this to L1, and see if a verifier can sync and reproduce it
	batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(dp.Addresses.Batcher)(t)
	miner.ActL1EndBlock(t)

	_, verifier := setupVerifier(t, sd, log, miner.L1Client(t, sd.RollupCfg))
	verifier.ActL2PipelineFull(t)

	require.Equal(t, sequencer.L2Unsafe(), verifier.L2Safe(), "verifier stays in sync, even with gaslimit changes")
}
