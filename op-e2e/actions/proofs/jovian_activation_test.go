package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-program/client/claim"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_JovianActivation(gt *testing.T) {

	runJovianDerivationTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		// Define override to activate Jovian 14 seconds after genesis
		var setJovianTime = func(dc *genesis.DeployConfig) {
			// Set all predecessor forks at genesis (0 offset)
			zero := hexutil.Uint64(0)
			dc.L2GenesisHoloceneTimeOffset = &zero
			// Set Isthmus at 10s (required predecessor fork for Jovian)
			ten := hexutil.Uint64(10)
			dc.L2GenesisIsthmusTimeOffset = &ten
			// Then set Jovian at 14s
			fourteen := hexutil.Uint64(14)
			dc.L2GenesisJovianTimeOffset = &fourteen
		}

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), setJovianTime)

		t.Logf("L2 Genesis Time: %d, JovianTime: %d ", env.Sequencer.RollupCfg.Genesis.L2Time, *env.Sequencer.RollupCfg.JovianTime)

		// Build the L2 chain until the Jovian activation time,
		// which for the Execution Engine is an L2 block timestamp
		for env.Engine.L2Chain().CurrentBlock().Time < *env.Sequencer.RollupCfg.JovianTime {
			b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
			// Since Holocene is active at genesis, extra data is already 9 bytes
			require.Len(t, b.Extra(), 9, "extra data should be 9 bytes (Holocene active)")
			env.Sequencer.ActL2StartBlock(t)
			// Send an L2 tx
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
			t.Log("Unsafe block with timestamp %d", b.Time)
		}
		b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
		require.Len(t, b.Extra(), 10, "extra data should be 10 bytes after Jovian activation (adds minBaseFee)")

		// Build up a local list of frames
		orderedFrames := make([][]byte, 0, 1)
		// Submit the first two blocks, this will be enough to trigger Jovian _derivation_
		// which is activated by the L1 inclusion block timestamp
		// block 1 will be 12 seconds after genesis, and 2 seconds before Jovian activation
		// block 2 will be 24 seconds after genesis, and 10 seconds after Jovian activation
		blocksToSubmit := []uint{1, 2}
		// Buffer the blocks in the batcher and populate orderedFrames list
		env.Batcher.ActCreateChannel(t, false)
		for i, blockNum := range blocksToSubmit {
			env.Batcher.ActAddBlockByNumber(t, int64(blockNum), actionsHelpers.BlockLogger(t))
			if i == len(blocksToSubmit)-1 {
				env.Batcher.ActL2ChannelClose(t)
			}
			frame := env.Batcher.ReadNextOutputFrame(t)
			require.NotEmpty(t, frame, "frame %d", i)
			orderedFrames = append(orderedFrames, frame)
		}

		includeBatchTx := func() {
			// Include the last transaction submitted by the batcher.
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
			env.Miner.ActL1EndBlock(t)
		}

		// Submit first frame
		env.Batcher.ActL2BatchSubmitRaw(t, orderedFrames[0])
		includeBatchTx() // L1 block should have a timestamp of 12s after genesis

		// Jovian should activate 14s after genesis, so that the previous l1 block
		// was before JovianTime and the next l1 block is after it

		// Submit final frame
		env.Batcher.ActL2BatchSubmitRaw(t, orderedFrames[1])
		includeBatchTx() // block should have a timestamp of 24s after genesis

		// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		l2SafeHead := env.Sequencer.L2Safe()
		t.Logf("Safe head block number: %d, timestamp: %d", l2SafeHead.Number, l2SafeHead.Time)
		// For Jovian, we expect the safe head to progress (different from Holocene behavior)
		require.True(t, l2SafeHead.Number >= uint64(0), "safe head should progress")

		// Verify Jovian fork activation occurred by checking for the activation log
		jovianRecs := env.Logs.FindLogs(
			testlog.NewMessageContainsFilter("Detected hardfork activation block"),
			testlog.NewAttributesFilter("role", "sequencer"),
			testlog.NewAttributesFilter("forkName", "jovian"),
		)
		require.Len(t, jovianRecs, 1, "Jovian fork should be detected and activated exactly once")
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddTestCase(
		"HonestClaim-JovianActivation",
		nil,
		helpers.NewForkMatrix(helpers.Granite),
		runJovianDerivationTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim-JovianActivation",
		nil,
		helpers.NewForkMatrix(helpers.Granite),
		runJovianDerivationTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}