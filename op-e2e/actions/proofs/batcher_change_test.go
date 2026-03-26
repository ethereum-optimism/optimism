package proofs_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// Test_ProgramAction_BatcherChangeWithinChannelTimeout verifies that after a
// pipeline reset, the system config is loaded from the walked-back L2 block
// (channel_timeout L1 blocks behind the safe head's L1 origin), not from the
// safe head itself.
//
// Scenario:
//  1. Batcher A submits a batch.
//  2. The batcher address is changed from A to B on L1.
//  3. Batcher B submits a batch.
//  4. The fault proof program re-derives the chain from scratch. During reset,
//     the pipeline walks back by channel_timeout and should find the old system
//     config (batcher A). If it incorrectly uses the safe head's config
//     (batcher B), it rejects batcher A's batch and derivation diverges.
func Test_ProgramAction_BatcherChangeWithinChannelTimeout(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Granite, helpers.Jovian),
		testBatcherChangeWithinChannelTimeout,
	)
	matrix.Run(gt)
}

func testBatcherChangeWithinChannelTimeout(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	miner := env.Miner
	sequencer := env.Sequencer

	// Step 1: Batcher A (default) submits a batch.
	miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	env.Batcher.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	miner.ActL1EndBlock(t)

	// Derive batcher A's batch so the safe head advances.
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	safeAfterA := sequencer.SyncStatus().SafeL2
	require.Greater(t, safeAfterA.Number, uint64(0), "safe head should advance after batcher A's batch")

	// Step 2: Change batcher from A (dp.Addresses.Batcher) to B (dp.Addresses.Bob).
	sysCfgContract, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, miner.EthClient())
	require.NoError(t, err)
	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
	require.NoError(t, err)
	tx, err := sysCfgContract.SetBatcherHash(sysCfgOwner, eth.AddressAsLeftPaddedHash(env.Dp.Addresses.Bob))
	require.NoError(t, err)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)
	t.Logf("batcher changed from %s to %s in L1 tx %s", env.Dp.Addresses.Batcher, env.Dp.Addresses.Bob, tx.Hash())

	// Step 3: Build L2 blocks adopting the batcher change, submit with batcher B.
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)

	// Create a second batcher with Bob's key.
	batcherBCfg := *helpers.NewBatcherCfg()
	batcherBCfg.BatcherKey = env.Dp.Secrets.Bob
	batcherB := actionsHelpers.NewL2Batcher(
		testlog.Logger(t, log.LevelDebug),
		env.Sd.RollupCfg,
		&batcherBCfg,
		sequencer.RollupClient(),
		miner.EthClient(),
		env.Engine.EthClient(),
		env.Engine.EngineClient(t, env.Sd.RollupCfg),
	)
	batcherB.ActSubmitAll(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(batcherB.LastSubmitted.Hash())(t)
	miner.ActL1EndBlock(t)

	// Derive batcher B's batch.
	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)
	safeAfterB := sequencer.SyncStatus().SafeL2
	require.Greater(t, safeAfterB.Number, safeAfterA.Number, "safe head should advance after batcher B's batch")

	// Step 4: Run the fault proof program. This re-derives the chain from
	// scratch, triggering a pipeline reset. The pipeline must walk back by
	// channel_timeout and use batcher A's system config for the initial
	// derivation window. If it uses batcher B's config, batcher A's batch
	// is rejected and derivation produces a different (shorter) safe chain.
	env.RunFaultProofProgram(t, safeAfterB.Number, testCfg.CheckResult, testCfg.InputParams...)
}
