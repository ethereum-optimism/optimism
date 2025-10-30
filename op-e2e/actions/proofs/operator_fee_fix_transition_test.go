package proofs

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// This test runs through the migration from Isthmus to Jovian
// for a chain already running nonzero operator fee scalars.
// It establishes that no special logic is in place to automatically reset
// the scalars, and fees are therefore expected to vastly increase
// under the new Jovian formula.
func Test_ProgramAction_OperatorFeeFixTransition(gt *testing.T) {

	run := func(gt *testing.T, testCfg *helpers.TestCfg[any]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		deployConfigOverrides := func(dp *genesis.DeployConfig) {
			dp.ActivateForkAtOffset(rollup.Jovian, 15)
		}

		testOperatorFeeScalar := uint32(345e6)
		testOperatorFeeConstant := uint64(123e4)
		expectedOperatorFeeIsthmus := uint64(8_475_000)          // (21000 * 345e6 / 1e6) + 123e4
		expectedOperatorFeeJovian := uint64(724_500_001_230_000) // (21000 * 345e6 * 100) + 123e4

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), deployConfigOverrides)

		balanceAt := func(a common.Address, blockNumber *big.Int) *big.Int {
			t.Helper()
			bal, err := env.Engine.EthClient().BalanceAt(t.Ctx(), a, blockNumber)
			require.NoError(t, err)
			return bal
		}

		getCurrentBalances := func(blockNumberU64 uint64) (alice *big.Int, l1FeeVault *big.Int, baseFeeVault *big.Int, sequencerFeeVault *big.Int, operatorFeeVault *big.Int) {
			blockNumber := new(big.Int).SetUint64(blockNumberU64)
			alice = balanceAt(env.Alice.Address(), blockNumber)
			l1FeeVault = balanceAt(predeploys.L1FeeVaultAddr, blockNumber)
			baseFeeVault = balanceAt(predeploys.BaseFeeVaultAddr, blockNumber)
			sequencerFeeVault = balanceAt(predeploys.SequencerFeeVaultAddr, blockNumber)
			operatorFeeVault = balanceAt(predeploys.OperatorFeeVaultAddr, blockNumber)

			return alice, l1FeeVault, baseFeeVault, sequencerFeeVault, operatorFeeVault
		}

		// Bind to the GasPriceOracle L2 contract and the L1 SystemConfig contract
		gpo, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, env.Engine.EthClient())
		require.NoError(t, err)
		sysCfgContract, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
		require.NoError(t, err)
		sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
		require.NoError(t, err)

		checkGPOStatusAndCall := func(isJovian bool, gasUsed uint64) {
			isIsthmus, err := gpo.IsIsthmus(nil)
			require.NoError(t, err)
			require.True(t, isIsthmus, isIsthmus)
			isJovianResult, err := gpo.IsJovian(nil)
			require.NoError(t, err)
			if isJovian {
				require.True(t, isJovianResult, "GPO should report that Jovian is active")
			} else {
				// On a live chain, this would actually revert
				require.False(t, isJovianResult, "GPO should report that Jovian is not active")
			}
			gpoOperatorFee, err := gpo.GetOperatorFee(nil, new(big.Int).SetUint64(gasUsed))
			require.NoError(t, err)
			if isJovian {
				require.Equal(t, expectedOperatorFeeJovian, gpoOperatorFee.Uint64())
			} else {
				require.Equal(t, expectedOperatorFeeIsthmus, gpoOperatorFee.Uint64())
			}
		}

		buildL2BlockWithSingleTx := func() (unsafeL2Head eth.L2BlockRef, receipt *types.Receipt) {
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			unsafeL2Head = env.Sequencer.ActL2EndBlock(t)
			receipt = env.Alice.L2.LastTxReceipt(t)
			return unsafeL2Head, receipt
		}

		updateOperatorFeeScalars := func() {
			_, err = sysCfgContract.SetOperatorFeeScalars(sysCfgOwner, testOperatorFeeScalar, testOperatorFeeConstant)
			require.NoError(t, err)
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
			env.Miner.ActL1EndBlock(t)
		}

		advanceL1Origin := func() eth.L2BlockRef {
			// sequence L2 blocks to adopt an l1 origin with the updated operator fee parameters
			env.Sequencer.ActL1HeadSignal(t)
			return env.Sequencer.ActBuildToL1Head(t)
		}

		checkFeeVaultChanges := func(initialBalance *big.Int, finalBalance *big.Int, expectedOperatorFee uint64) {
			delta := new(big.Int).Sub(finalBalance, initialBalance).Uint64()
			require.Equal(t, expectedOperatorFee, delta)
		}

		// Update params on L1 and progress L2 chain
		// to adopt the new operator fee parameters
		updateOperatorFeeScalars()
		unsafeL2Head := advanceL1Origin()

		// Cache balances
		aliceInitialBalance, _, _, _, operatorFeeVaultInitialBalance := getCurrentBalances(unsafeL2Head.Number)
		require.Zero(t, operatorFeeVaultInitialBalance.Sign())

		// Build an L2 block with a single L2 tx consuming precisely 21,000 gas
		unsafeL2Head, receipt := buildL2BlockWithSingleTx()

		// Check that the scalars are in the receipt
		require.Equal(t, testOperatorFeeScalar, uint32(*receipt.OperatorFeeScalar))
		require.Equal(t, testOperatorFeeConstant, *receipt.OperatorFeeConstant)

		// Check GPO status
		checkGPOStatusAndCall(false, receipt.GasUsed)

		// Isthmus formula: (gasUsed * operatorFeeScalar / 1e6) + operatorFeeConstant
		aliceFinalBalance, _, _, _, operatorFeeVaultFinalBalance := getCurrentBalances(unsafeL2Head.Number)
		checkFeeVaultChanges(operatorFeeVaultInitialBalance, operatorFeeVaultFinalBalance, expectedOperatorFeeIsthmus)
		require.True(t, aliceFinalBalance.Cmp(aliceInitialBalance) < 0, "Alice's balance should decrease")

		// Now wind forward to jovian
		unsafeL2Head = env.Sequencer.ActBuildL2ToFork(t, rollup.Jovian)

		// reset accounting
		aliceInitialBalance, _, _, _, operatorFeeVaultInitialBalance = getCurrentBalances(unsafeL2Head.Number)

		unsafeL2Head, receipt = buildL2BlockWithSingleTx()

		// Check that the scalars are in the receipt
		require.Equal(t, testOperatorFeeScalar, uint32(*receipt.OperatorFeeScalar))
		require.Equal(t, testOperatorFeeConstant, *receipt.OperatorFeeConstant)

		// Check GPO status
		checkGPOStatusAndCall(true, receipt.GasUsed)

		// Assert balance changes
		aliceFinalBalance, _, _, _, operatorFeeVaultFinalBalance = getCurrentBalances(unsafeL2Head.Number)
		checkFeeVaultChanges(operatorFeeVaultInitialBalance, operatorFeeVaultFinalBalance, expectedOperatorFeeJovian)
		require.True(t, aliceFinalBalance.Cmp(aliceInitialBalance) < 0, "Alice's balance should decrease")

		// Batch, mine, sync and run fault proof program
		l2SafeHead := env.BatchMineAndSync(t)
		env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(nil, helpers.NewForkMatrix(helpers.Isthmus), run)
	matrix.Run(gt)
}
