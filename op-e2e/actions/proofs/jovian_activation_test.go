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
			// Activate Isthmus at genesis
			zero := hexutil.Uint64(0)
			dc.L2GenesisIsthmusTimeOffset = &zero
			// Then set Jovian at 10s
			ten := hexutil.Uint64(10)
			dc.L2GenesisJovianTimeOffset = &ten
		}

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), setJovianTime)

		t.Logf("L2 Genesis Time: %d, JovianTime: %d ", env.Sequencer.RollupCfg.Genesis.L2Time, *env.Sequencer.RollupCfg.JovianTime)

		// Build the L2 chain until the Jovian activation time,
		// which for the Execution Engine is an L2 block timestamp
		for env.Engine.L2Chain().CurrentBlock().Time < *env.Sequencer.RollupCfg.JovianTime {
			b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
			// Since Holocene is active at genesis, extra data is already 9 bytes
			require.Len(t, b.Extra(), 9, "extra data should be 9 bytes (Holocene active)")
			env.Sequencer.ActL2EmptyBlock(t)
			t.Log("Unsafe block with timestamp %d", b.Time)
		}
		b := env.Engine.L2Chain().GetBlockByHash(env.Sequencer.L2Unsafe().Hash)
		require.Len(t, b.Extra(), 17, "extra data should be 17 bytes after Jovian activation (adds minBaseFee)")

		env.BatchMineAndSync(t)

		l2SafeHead := env.Sequencer.L2Safe()
		t.Logf("Safe head block number: %d, timestamp: %d", l2SafeHead.Number, l2SafeHead.Time)
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
		helpers.NewForkMatrix(helpers.Isthmus),
		runJovianDerivationTest,
		helpers.ExpectNoError(),
	)
	matrix.AddTestCase(
		"JunkClaim-JovianActivation",
		nil,
		helpers.NewForkMatrix(helpers.Isthmus),
		runJovianDerivationTest,
		helpers.ExpectError(claim.ErrClaimNotValid),
		helpers.WithL2Claim(common.HexToHash("0xdeadbeef")),
	)
}
