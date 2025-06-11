package proofs

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_OperatorFeeConstistency(gt *testing.T) {
	type testCase int64

	const (
		NormalTx testCase = iota
		DepositTx
		StateRefund
		IsthmusTransitionBlock
	)

	const testOperatorFeeScalar = uint32(20000)
	const testOperatorFeeConstant = uint64(500)
	const testDepositValue = uint64(10000)
	testStorageUpdateContractAddress := common.HexToAddress("0xffffffff")
	// contract TestSetter {
	//   uint x;
	//   function set(uint _x) public { x = _x; }
	// }
	// The deployed bytecode below is from the contract above
	testStorageUpdateContractCode := common.FromHex("0x6080604052348015600e575f80fd5b50600436106026575f3560e01c806360fe47b114602a575b5f80fd5b60406004803603810190603c9190607d565b6042565b005b805f8190555050565b5f80fd5b5f819050919050565b605f81604f565b81146068575f80fd5b50565b5f813590506077816058565b92915050565b5f60208284031215608f57608e604b565b5b5f609a84828501606b565b9150509291505056fea26469706673582212201712a1e6e9c5e2ba1f8f7403f5d6e00090c6fa2b70c632beea4be8009331bd2064736f6c63430008190033")

	runIsthmusDerivationTest := func(gt *testing.T, testCfg *helpers.TestCfg[testCase]) {
		t := actionsHelpers.NewDefaultTesting(gt)
		deployConfigOverrides := func(dp *genesis.DeployConfig) {}

		if testCfg.Custom == StateRefund {
			testCfg.Allocs = actionsHelpers.DefaultAlloc
			testCfg.Allocs.L2Alloc = make(map[common.Address]types.Account)
			testCfg.Allocs.L2Alloc[testStorageUpdateContractAddress] = types.Account{
				Code:    testStorageUpdateContractCode,
				Nonce:   1,
				Balance: new(big.Int),
			}
		}

		if testCfg.Custom == IsthmusTransitionBlock {
			deployConfigOverrides = func(dp *genesis.DeployConfig) {
				dp.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
				dp.L2GenesisIsthmusTimeOffset = ptr(hexutil.Uint64(13))
			}
		}

		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), deployConfigOverrides)

		balanceAt := func(a common.Address) *big.Int {
			t.Helper()
			bal, err := env.Engine.EthClient().BalanceAt(t.Ctx(), a, nil)
			require.NoError(t, err)
			return bal
		}

		getCurrentBalances := func() (alice *big.Int, l1FeeVault *big.Int, baseFeeVault *big.Int, sequencerFeeVault *big.Int, operatorFeeVault *big.Int) {
			alice = balanceAt(env.Alice.Address())
			l1FeeVault = balanceAt(predeploys.L1FeeVaultAddr)
			baseFeeVault = balanceAt(predeploys.BaseFeeVaultAddr)
			sequencerFeeVault = balanceAt(predeploys.SequencerFeeVaultAddr)
			operatorFeeVault = balanceAt(predeploys.OperatorFeeVaultAddr)

			return alice, l1FeeVault, baseFeeVault, sequencerFeeVault, operatorFeeVault
		}

		setStorageInUpdateContractTo := func(value byte) {
			t.Helper()
			input := common.RightPadBytes(common.FromHex("0x60fe47b1"), 36)
			input[35] = value
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&testStorageUpdateContractAddress)(t)
			env.Alice.L2.ActSetTxCalldata(input)(t)
			env.Alice.L2.ActMakeTx(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
			r := env.Alice.L2.LastTxReceipt(t)
			require.Equal(t, types.ReceiptStatusSuccessful, r.Status, "tx unsuccessful")
		}

		t.Logf("L2 Genesis Time: %d, IsthmusTime: %d ", env.Sequencer.RollupCfg.Genesis.L2Time, *env.Sequencer.RollupCfg.IsthmusTime)

		sysCfgContract, err := bindings.NewSystemConfig(env.Sd.RollupCfg.L1SystemConfigAddress, env.Miner.EthClient())
		require.NoError(t, err)

		sysCfgOwner, err := bind.NewKeyedTransactorWithChainID(env.Dp.Secrets.Deployer, env.Sd.RollupCfg.L1ChainID)
		require.NoError(t, err)

		// Update the operator fee parameters
		_, err = sysCfgContract.SetOperatorFeeScalars(sysCfgOwner, testOperatorFeeScalar, testOperatorFeeConstant)
		require.NoError(t, err)

		env.Miner.ActL1StartBlock(12)(t)
		env.Miner.ActL1IncludeTx(env.Dp.Addresses.Deployer)(t)
		env.Miner.ActL1EndBlock(t)

		// sequence L2 blocks, and submit with new batcher
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActBuildToL1Head(t)
		env.BatchAndMine(t)

		env.Sequencer.ActL1HeadSignal(t)

		var aliceInitialBalance *big.Int
		var baseFeeVaultInitialBalance *big.Int
		var l1FeeVaultInitialBalance *big.Int
		var sequencerFeeVaultInitialBalance *big.Int
		var operatorFeeVaultInitialBalance *big.Int

		var receipt *types.Receipt

		switch testCfg.Custom {
		case NormalTx, IsthmusTransitionBlock:
			aliceInitialBalance, l1FeeVaultInitialBalance, baseFeeVaultInitialBalance, sequencerFeeVaultInitialBalance, operatorFeeVaultInitialBalance = getCurrentBalances()

			require.Equal(t, operatorFeeVaultInitialBalance.Sign(), 0)

			// Send an L2 tx
			env.Sequencer.ActL2StartBlock(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
			env.Alice.L2.ActMakeTx(t)
			// we usually don't include txs in the transition block, so we force-include it
			env.Engine.ActL2IncludeTxIgnoreForcedEmpty(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)

			if testCfg.Custom == IsthmusTransitionBlock {
				require.True(t, env.Sd.RollupCfg.IsIsthmusActivationBlock(env.Sequencer.L2Unsafe().Time))
			}

		case StateRefund:
			setStorageInUpdateContractTo(1)
			rSet := env.Alice.L2.LastTxReceipt(t)
			require.Equal(t, uint64(43696), rSet.GasUsed)
			aliceInitialBalance, l1FeeVaultInitialBalance, baseFeeVaultInitialBalance, sequencerFeeVaultInitialBalance, operatorFeeVaultInitialBalance = getCurrentBalances()
			setStorageInUpdateContractTo(0)
			rUnset := env.Alice.L2.LastTxReceipt(t)
			// we assert on the exact gas used to show that a refund is happening
			require.Equal(t, uint64(21784), rUnset.GasUsed)

		case DepositTx:
			aliceInitialBalance, l1FeeVaultInitialBalance, baseFeeVaultInitialBalance, sequencerFeeVaultInitialBalance, operatorFeeVaultInitialBalance = getCurrentBalances()

			bobInitialBalance := balanceAt(env.Bob.Address())

			// regular Deposit, in new L1 block
			env.Alice.L1.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			env.Alice.L2.ActSetTxValue(new(big.Int).SetUint64(testDepositValue))(t)
			env.Alice.ActDeposit(t)
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
			env.Miner.ActL1EndBlock(t)

			// sync sequencer build enough blocks to adopt latest L1 origin
			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActBuildToL1HeadUnsafe(t)

			env.Alice.ActCheckDepositStatus(true, true)(t)

			bobFinalBalance := balanceAt(env.Bob.Address())

			require.Equal(t, bobInitialBalance.Uint64()+testDepositValue, bobFinalBalance.Uint64())

			receipt, err = env.Alice.GetLastDepositL2Receipt(t)
			require.NoError(t, err)
		}

		aliceFinalBalance, l1FeeVaultFinalBalance, baseFeeVaultFinalBalance, sequencerFeeVaultFinalBalance, operatorFeeVaultFinalBalance := getCurrentBalances()

		if receipt == nil {
			receipt = env.Alice.L2.LastTxReceipt(t)
		}

		if testCfg.Custom == DepositTx || testCfg.Custom == IsthmusTransitionBlock {
			require.Nil(t, receipt.OperatorFeeScalar)
			require.Nil(t, receipt.OperatorFeeConstant)

			// Nothing should has been sent to operator fee vault
			require.Equal(t, operatorFeeVaultInitialBalance, operatorFeeVaultFinalBalance)
		} else {
			// Check that the operator fee was applied
			require.Equal(t, testOperatorFeeScalar, uint32(*receipt.OperatorFeeScalar))
			require.Equal(t, testOperatorFeeConstant, *receipt.OperatorFeeConstant)

			// Check that the operator fee sent to the vault is correct
			require.Equal(t,
				new(big.Int).Add(
					new(big.Int).Div(
						new(big.Int).Mul(
							new(big.Int).SetUint64(receipt.GasUsed),
							new(big.Int).SetUint64(uint64(testOperatorFeeScalar)),
						),
						new(big.Int).SetUint64(1e6),
					),
					new(big.Int).SetUint64(testOperatorFeeConstant),
				),
				new(big.Int).Sub(operatorFeeVaultFinalBalance, operatorFeeVaultInitialBalance),
			)
		}
		require.True(t, aliceFinalBalance.Cmp(aliceInitialBalance) < 0, "Alice's balance should decrease")

		// Check that no Ether has been minted or burned
		finalTotalBalance := new(big.Int).Add(
			aliceFinalBalance,
			new(big.Int).Add(
				new(big.Int).Add(
					new(big.Int).Sub(l1FeeVaultFinalBalance, l1FeeVaultInitialBalance),
					new(big.Int).Sub(sequencerFeeVaultFinalBalance, sequencerFeeVaultInitialBalance),
				),
				new(big.Int).Add(
					new(big.Int).Sub(operatorFeeVaultFinalBalance, operatorFeeVaultInitialBalance),
					new(big.Int).Sub(baseFeeVaultFinalBalance, baseFeeVaultInitialBalance),
				),
			),
		)

		if testCfg.Custom == DepositTx {
			// Minus the deposit value that was sent to Bob
			require.Equal(t, aliceInitialBalance.Uint64()-testDepositValue, finalTotalBalance.Uint64())
		} else {
			require.Equal(t, aliceInitialBalance, finalTotalBalance)
		}

		l2UnsafeHead := env.Engine.L2Chain().CurrentHeader()

		env.BatchAndMine(t)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

		require.Equal(t, eth.HeaderBlockID(l2SafeHead), eth.HeaderBlockID(l2UnsafeHead), "derivation leads to the same block")

		env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[testCase]()
	matrix.AddDefaultTestCasesWithName("NormalTx", NormalTx, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("DepositTx", DepositTx, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("StateRefund", StateRefund, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("IsthmusTransitionBlock", IsthmusTransitionBlock, helpers.NewForkMatrix(helpers.Holocene), runIsthmusDerivationTest)
	matrix.Run(gt)
}

func ptr[T any](v T) *T {
	return &v
}
