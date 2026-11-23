package proofs_test

import (
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// TestL1AmsterdamForkAfterGenesis proves an L2 chain whose L1 activates Amsterdam: blob batches on
// both sides of the activation, and a deposit on an Amsterdam L1 block.
func TestL1AmsterdamForkAfterGenesis(gt *testing.T) {
	runL1AmsterdamTest := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(),
			helpers.NewBatcherCfg(
				func(c *actionsHelpers.BatcherCfg) {
					c.DataAvailabilityType = batcherFlags.BlobsType
				},
			),
			func(dc *genesis.DeployConfig) {
				dc.L1AmsterdamTimeOffset = ptr(hexutil.Uint64(36))
			},
		)
		l1Cfg := env.Sd.L1Cfg.Config

		// A batch with a pre-Amsterdam L1 origin, included before Amsterdam.
		env.Miner.ActEmptyBlock(t)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActBuildToL1Head(t)
		env.BatchMineAndSync(t)
		l1Head := env.Miner.L1Chain().CurrentHeader()
		require.False(t, l1Cfg.IsAmsterdam(l1Head.Number, l1Head.Time), "the first batch lands before Amsterdam")

		// A deposit on the Amsterdam activation block, and a batch with an Amsterdam L1 origin.
		env.Alice.L1.ActResetTxOpts(t)
		env.Alice.L2.ActResetTxOpts(t)
		env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
		env.Alice.ActDeposit(t)
		env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
		env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
		env.Miner.ActL1EndBlock(t)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActBuildToL1Head(t)
		env.Alice.ActCheckDepositStatus(true, true)(t)
		l2Safe := env.BatchMineAndSync(t)
		l1Origin := env.Miner.L1Chain().GetHeaderByHash(l2Safe.L1Origin.Hash)
		require.True(t, l1Cfg.IsAmsterdam(l1Origin.Number, l1Origin.Time), "the safe head has an Amsterdam L1 origin")

		env.RunFaultProofProgramFromGenesis(t, l2Safe.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.LatestFork), runL1AmsterdamTest)
	matrix.Run(gt)
}
