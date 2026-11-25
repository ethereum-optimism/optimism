package proofs_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
)

// TestBatcherChangeWithinChannelTimeout verifies that after a
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
func TestBatcherChangeWithinChannelTimeout(gt *testing.T) {
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
	safeAfterA := env.BatchMineAndSync(t)
	require.Greater(t, safeAfterA.Number, uint64(0), "safe head should advance after batcher A's batch")

	// Step 2: Change batcher from A to B (Bob) on L1 and replace env.Batcher.
	env.RotateBatcher(t, env.Dp.Secrets.Bob)

	// Step 3: Build L2 blocks adopting the batcher change, submit with batcher B.
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	safeAfterB := env.BatchMineAndSync(t)
	require.Greater(t, safeAfterB.Number, safeAfterA.Number, "safe head should advance after batcher B's batch")

	// Step 4: Run the fault proof program. This re-derives the chain from
	// scratch, triggering a pipeline reset. The pipeline must walk back by
	// channel_timeout and use batcher A's system config for the initial
	// derivation window. If it uses batcher B's config, batcher A's batch
	// is rejected and derivation produces a different (shorter) safe chain.
	env.RunFaultProofProgram(t, safeAfterB.Number, testCfg.CheckResult, testCfg.InputParams...)
}
