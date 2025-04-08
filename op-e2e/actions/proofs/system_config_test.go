package proofs_test

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Test_ProgramAction_SystemConfigEarlyIsthmusUpgrade tests that setting the operator
// fee parameters pre-Isthmus is ignored and doesn't cause problems during derivation.
func Test_ProgramAction_SystemConfigEarlyIsthmusUpgrade(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Holocene),
		testSystemConfigEarlyIsthmusUpgrade,
	)
	matrix.Run(gt)
}

func testSystemConfigEarlyIsthmusUpgrade(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	const isthmusOffset = 24

	testOperatorFeeScalar := uint32(20000)
	testOperatorFeeConstant := uint64(500)

	t := actionsHelpers.NewDefaultTesting(gt)
	testSetup := func(dp *genesis.DeployConfig) {
		dp.ActivateForkAtOffset(rollup.Isthmus, isthmusOffset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)
	sequencer := env.Sequencer
	miner := env.Miner

	sysCfg, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
	require.NoError(t, err)
	opts := &bind.CallOpts{}

	sysCfgVerStr, err := sysCfg.Version(opts)
	require.NoError(t, err)
	t.Logf("SystemConfig version: %s", sysCfgVerStr)
	ver, err := semver.NewVersion(sysCfgVerStr)
	require.NoError(t, err)
	require.GreaterOrEqual(t, ver.Major(), uint64(3), "expect Isthmus contracts or later")

	opFeeScalarAndConstant := func() (uint32, uint64) {
		t.Helper()
		scalar, err := sysCfg.OperatorFeeScalar(opts)
		require.NoError(t, err)
		constant, err := sysCfg.OperatorFeeConstant(opts)
		require.NoError(t, err)
		return scalar, constant
	}

	requireL1InfoParams := func(block eth.L2BlockRef, scalar uint32, constant uint64) {
		t.Helper()
		l1infoTx := env.Engine.L2Chain().GetBlockByHash(block.Hash).Transactions()[0]
		l1info, err := derive.L1BlockInfoFromBytes(env.Sd.RollupCfg, block.Time, l1infoTx.Data())
		require.NoError(t, err)
		require.Equal(t, scalar, l1info.OperatorFeeScalar)
		require.Equal(t, constant, l1info.OperatorFeeConstant)
	}

	scalar0, constant0 := opFeeScalarAndConstant()
	require.Zero(t, scalar0)
	require.Zero(t, constant0)

	owner, err := sysCfg.Owner(opts)
	require.NoError(t, err)
	require.Equal(t, env.Dp.Addresses.Deployer, owner)
	sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
	require.NoError(t, err)

	_, err = sysCfg.SetOperatorFeeScalars(sysCfgOwner, testOperatorFeeScalar, testOperatorFeeConstant)
	require.NoError(t, err)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)
	scalar1, constant1 := opFeeScalarAndConstant()
	require.Equal(t, testOperatorFeeScalar, scalar1)
	require.Equal(t, testOperatorFeeConstant, constant1)

	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	safeL2 := env.BatchMineAndSync(t)
	require.False(t, env.Sd.RollupCfg.IsIsthmus(sequencer.SyncStatus().SafeL2.Time))

	requireL1InfoParams(safeL2, 0, 0)
	env.RunFaultProofProgram(t, safeL2.Number, testCfg.CheckResult, testCfg.InputParams...)

	sequencer.ActBuildL2ToIsthmus(t)
	sequencer.ActL2EmptyBlock(t) // one more to have Isthmus L1 info deposit
	safeL2 = env.BatchMineAndSync(t)
	require.True(t, env.Sd.RollupCfg.IsIsthmus(sequencer.SyncStatus().SafeL2.Time))

	// Assert that operator fee params are zero since they were set before Isthmus activated.
	requireL1InfoParams(safeL2, 0, 0)
	env.RunFaultProofProgram(t, safeL2.Number, testCfg.CheckResult, testCfg.InputParams...)

	// modify both to ensure we test with different parameters
	testOperatorFeeScalar *= 2
	testOperatorFeeConstant *= 2
	// Now set them again with Isthmus active and see that they are set correctly.
	_, err = sysCfg.SetOperatorFeeScalars(sysCfgOwner, testOperatorFeeScalar, testOperatorFeeConstant)
	require.NoError(t, err)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
	miner.ActL1EndBlock(t)
	sequencer.ActL1HeadSignal(t)
	sequencer.ActBuildToL1Head(t)
	safeL2 = env.BatchMineAndSync(t)

	requireL1InfoParams(safeL2, testOperatorFeeScalar, testOperatorFeeConstant)
	env.RunFaultProofProgram(t, safeL2.Number, testCfg.CheckResult, testCfg.InputParams...)
}
