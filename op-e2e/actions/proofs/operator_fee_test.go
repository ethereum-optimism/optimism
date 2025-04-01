package proofs

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func Test_ProgramAction_OperatorFeeConsistency(gt *testing.T) {
	type testCase int64

	const (
		NormalTx testCase = iota
		DepositTx
		StateRefund
		NotEnoughFundsInBatchMissingOpFee
		IsthmusTransitionBlock
	)

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

		var testOperatorFeeScalar uint32
		var testOperatorFeeConstant uint64
		if testCfg.Custom == NotEnoughFundsInBatchMissingOpFee {
			testOperatorFeeScalar = 0
			testOperatorFeeConstant = 0xffff
		} else {
			testOperatorFeeScalar = 100e6
			testOperatorFeeConstant = 500
		}

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
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
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

			// regular Deposit, in new L1 block
			env.Alice.L1.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			env.Alice.L2.ActSetTxGasLimit(2e6)(t)
			env.Alice.ActDeposit(t)
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
			env.Miner.ActL1EndBlock(t)

			// sync sequencer build enough blocks to adopt latest L1 origin
			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActBuildToL1HeadUnsafe(t)

			env.Alice.ActCheckDepositStatus(true, true)(t)

			receipt, err = env.Alice.GetLastDepositL2Receipt(t)
			require.NoError(t, err)
		case NotEnoughFundsInBatchMissingOpFee:
			pkey, err := crypto.GenerateKey()
			require.NoError(t, err)
			address := crypto.PubkeyToAddress(pkey.PublicKey)

			// Send `address` just enough ETH to cover the gas costs of the transaction, sans the operator fee.
			// Since we're just doing a simple call to `Bob`, that should be 21000 gas at the current gas price
			// plus the L1 data fee.

			// Craft a transaction from Alice -> Bob (just to compute L1 cost, not to send.)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			tx := env.Alice.L2.MakeTransaction(t)

			rlp, err := tx.MarshalBinary()
			require.NoError(t, err, "failed to RLP encode transaction")

			unsafeHeader := env.Engine.L2Chain().CurrentHeader()
			unsafeBlock := env.Engine.L2Chain().GetBlockByHash(unsafeHeader.Hash())
			nextBaseFee := eip1559.CalcBaseFee(
				env.Sd.L2Cfg.Config,
				unsafeHeader,
				unsafeHeader.Time+env.Sd.RollupCfg.BlockTime,
			)

			l1BlockInfo, err := derive.L1BlockInfoFromBytes(env.Sd.RollupCfg, unsafeHeader.Time, unsafeBlock.Transactions()[0].Data())
			require.NoError(t, err)

			daCost := fjordL1Cost(l1BlockInfo, types.NewRollupCostData(rlp))
			expectedFeePreIsthmus := nextBaseFee.Mul(nextBaseFee, big.NewInt(int64(params.TxGas)))
			expectedFeePreIsthmus.Add(expectedFeePreIsthmus, daCost)

			// Include an L2 tx, from Bob -> mock signer
			env.Bob.L2.ActResetTxOpts(t)
			env.Bob.L2.ActSetTxToAddr(&address)(t)
			env.Bob.L2.ActSetTxValue(expectedFeePreIsthmus)(t)
			env.Bob.L2.ActMakeTx(t)

			env.Sequencer.ActL2StartBlock(t)
			env.Engine.ActL2IncludeTx(env.Bob.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
			env.Bob.L2.ActCheckReceiptStatusOfLastTx(true)(t)

			// Ensure the mock signer received the funds
			require.Equal(t, expectedFeePreIsthmus, balanceAt(address))

			// Buffer the L2 block we just included
			env.Batcher.ActL2BatchBuffer(t)

			aliceInitialBalance, l1FeeVaultInitialBalance, baseFeeVaultInitialBalance, sequencerFeeVaultInitialBalance, operatorFeeVaultInitialBalance = getCurrentBalances()
			if testCfg.Custom != NotEnoughFundsInBatchMissingOpFee {
				require.Equal(t, operatorFeeVaultInitialBalance.Sign(), 0)
			}

			// Craft a transaction from Alice -> Bob
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			env.Alice.L2.ActSetTxGasLimit(params.TxGas)(t)
			env.Alice.L2.ActSetGasFeeCap(big.NewInt(1))(t)
			env.Alice.L2.ActSetGasTipCap(big.NewInt(1))(t)
			env.Alice.L2.ActMakeTx(t)

			// Include an L2 tx, from Alice -> Bob
			env.Sequencer.ActL2StartBlock(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
			env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

			// Instruct the batcher to submit a faulty channel, with Alice's tx re-signed by a new private key.
			// This key will have 0 balance.
			env.Batcher.ActL2BatchBuffer(t, func(block *types.Block) *types.Block {
				txs := block.Transactions()

				// Skip over any L2 blocks that don't contain user-space txs.
				if len(txs) == 1 {
					return block
				}

				// Re-sign Alice's tx with a random key.
				require.NoError(t, err, "error generating random private key")
				signer := types.LatestSigner(env.Sd.L2Cfg.Config)
				newSignedTx, err := types.SignTx(txs[1], signer, pkey)
				require.NoError(t, err, "error re-signing Alice's transaction")

				// Replace Alice's tx with the re-signed one.
				txs[1] = newSignedTx
				return block
			})
			env.Batcher.ActL2ChannelClose(t)
			env.Batcher.ActL2BatchSubmit(t)

			// Include the batcher transaction.
			env.Miner.ActL1StartBlock(12)(t)
			env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
			env.Miner.ActL1EndBlock(t)
		}

		aliceFinalBalance, l1FeeVaultFinalBalance, baseFeeVaultFinalBalance, sequencerFeeVaultFinalBalance, operatorFeeVaultFinalBalance := getCurrentBalances()
		l2UnsafeHead := env.Engine.L2Chain().CurrentHeader()

		if receipt == nil {
			receipt = env.Alice.L2.LastTxReceipt(t)
		}

		if testCfg.Custom == DepositTx || testCfg.Custom == IsthmusTransitionBlock {
			require.Nil(t, receipt.OperatorFeeScalar)
			require.Nil(t, receipt.OperatorFeeConstant)

			// Nothing should has been sent to operator fee vault
			require.Equal(t, operatorFeeVaultInitialBalance, operatorFeeVaultFinalBalance)
		} else if env.Sd.RollupCfg.IsIsthmus(l2UnsafeHead.Time) {
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

		if testCfg.Custom == DepositTx {
			require.Equal(t, aliceInitialBalance, aliceFinalBalance, "Alice's balance shouldn't have changed")
		} else {
			require.True(t, aliceFinalBalance.Cmp(aliceInitialBalance) < 0, "Alice's balance should decrease")
		}

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

		require.Equal(t, aliceInitialBalance, finalTotalBalance)

		// The NotEnoughFundsInBatchMissingOpFee case is special, as it submits its own invalid batch.
		if testCfg.Custom != NotEnoughFundsInBatchMissingOpFee {
			env.BatchAndMine(t)
		}
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)

		l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

		if testCfg.Custom == NotEnoughFundsInBatchMissingOpFee {
			// The unsafe block prior to derivation should be different from the safe block after derivation. The
			// batcher posted the block but with a different transaction, signed by a key that has no balance. This
			// should cause a reorg in the unsafe chain, and the original block should be reduced to deposits only
			// if Isthmus is active.
			require.NotEqual(t, eth.HeaderBlockID(l2SafeHead), eth.HeaderBlockID(l2UnsafeHead), "derivation should not lead to the same block")

			reorgedUnsafe := env.Engine.L2Chain().CurrentHeader()
			require.Equal(t, eth.HeaderBlockID(l2SafeHead), eth.HeaderBlockID(reorgedUnsafe), "reorged unsafe block is the same")

			safeHeadBlock := env.Engine.L2Chain().GetBlockByHash(l2SafeHead.Hash())
			if env.Sd.RollupCfg.IsIsthmus(l2SafeHead.Time) {
				require.Equal(t, len(safeHeadBlock.Transactions()), 1)

				// Ensure that the logs contain a mention of the block being replaced _due to the signer not having enough
				// balance_.
				require.NotNil(t, env.Logs.FindLog(testlog.NewAttributesContainsFilter("err", "insufficient funds for gas * price + value")))
				require.NotNil(t, env.Logs.FindLog(testlog.NewAttributesContainsFilter("err", "have 1400000021000 want 1400000086535")))
			} else {
				require.Equal(t, len(safeHeadBlock.Transactions()), 2)
			}
		} else {
			require.Equal(t, eth.HeaderBlockID(l2SafeHead), eth.HeaderBlockID(l2UnsafeHead), "derivation leads to the same block")
		}

		env.RunFaultProofProgram(t, l2SafeHead.Number.Uint64(), testCfg.CheckResult, testCfg.InputParams...)
	}

	matrix := helpers.NewMatrix[testCase]()
	matrix.AddDefaultTestCasesWithName("NormalTx", NormalTx, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("DepositTx", DepositTx, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("StateRefund", StateRefund, helpers.NewForkMatrix(helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("NotEnoughFundsInBatchMissingOpFee", NotEnoughFundsInBatchMissingOpFee, helpers.NewForkMatrix(helpers.Holocene, helpers.Isthmus), runIsthmusDerivationTest)
	matrix.AddDefaultTestCasesWithName("IsthmusTransitionBlock", IsthmusTransitionBlock, helpers.NewForkMatrix(helpers.Holocene), runIsthmusDerivationTest)
	matrix.Run(gt)
}

func fjordL1Cost(l1BlockInfo *derive.L1BlockInfo, rollupCostData types.RollupCostData) *big.Int {
	costFunc := types.NewL1CostFuncFjord(
		l1BlockInfo.BaseFee,
		l1BlockInfo.BlobBaseFee,
		new(big.Int).SetUint64(uint64(l1BlockInfo.BaseFeeScalar)),
		new(big.Int).SetUint64(uint64(l1BlockInfo.BlobBaseFeeScalar)))

	fee, _ := costFunc(rollupCostData)
	return fee
}

func ptr[T any](v T) *T {
	return &v
}
