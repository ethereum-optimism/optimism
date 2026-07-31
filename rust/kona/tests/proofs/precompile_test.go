package proofs

import (
	"testing"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/invoker"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestPrecompiles(gt *testing.T) {
	matrix := helpers.NewMatrix[PrecompileTestFixture]()
	for _, test := range PrecompileTestFixtures {
		testCase := test
		matrix.AddTestCase(
			testCase.Name,
			testCase,
			helpers.FaultProofForks(),
			runPrecompileTest,
			helpers.ExpectNoError(),
		)
	}
	matrix.Run(gt)
}

func runPrecompileTest(gt *testing.T, testCfg *helpers.TestCfg[PrecompileTestFixture]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())
	testCase := testCfg.Custom

	// deploy invoker contract
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxCalldata(common.FromHex(invoker.InvokerMetaData.Bin))(t)
	env.Alice.L2.ActMakeTx(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
	env.Sequencer.ActL2EndBlock(t)
	env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

	invokerContract := env.Alice.L2.LastTxReceipt(t).ContractAddress
	require.NotZero(t, invokerContract, "invoker contract address is zero")
	abi, err := invoker.InvokerMetaData.GetAbi()
	require.NoError(t, err)
	invokeCalldata, err := abi.Pack("invokePrecompile", testCase.Address, testCase.Input)
	require.NoError(t, err)

	// call precompile via invoker
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&invokerContract)(t)
	env.Alice.L2.ActSetTxCalldata(invokeCalldata)(t)
	env.Alice.L2.ActMakeTx(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
	env.Sequencer.ActL2EndBlock(t)
	env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

	receipt := env.Alice.L2.LastTxReceipt(t)
	receiptBlockTime := env.Engine.L2Chain().GetBlockByHash(receipt.BlockHash).Time()
	rules := env.Engine.L2Chain().Config().Rules(receipt.BlockNumber, true, receiptBlockTime)
	expectedResult := make([]byte, 0)
	precompile, ok := vm.ActivePrecompiledContracts(rules)[testCase.Address]
	if ok {
		expectedResult, err = precompile.Run(testCase.Input)
		require.NoError(t, err)
	}

	// sanity check Invoker precompile call
	require.Equal(t, receipt.Status, uint64(1), "transaction should succeed")
	require.Len(t, receipt.Logs, 1)
	require.Equal(t, receipt.Logs[0].Address, invokerContract)
	require.Len(t, receipt.Logs[0].Topics, 2)
	precompileAddress := receipt.Logs[0].Topics[1]
	var out struct {
		Result             []byte
		DelegateCallResult []byte
	}
	err = abi.UnpackIntoInterface(&out, "PrecompileInvoked", receipt.Logs[0].Data)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(precompileAddress.Hex()), testCase.Address)
	require.Equal(t, expectedResult, out.Result)
	require.Equal(t, expectedResult, out.DelegateCallResult)

	// instruct the batcher to submit the Invoker precompile tx to l1, and include the transaction.
	env.Batcher.ActSubmitAll(t)
	env.Miner.ActL1StartBlock(12)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)

	// Finalize the block with the batch on L1.
	env.Miner.ActL1SafeNext(t)
	env.Miner.ActL1FinalizeNext(t)

	// Instruct the sequencer to derive the L2 chain from the data on L1 that the batcher just posted.
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l1Head := env.Miner.L1Chain().CurrentBlock()
	l2SafeHead := env.Engine.L2Chain().CurrentSafeBlock()

	require.Equal(t, uint64(1), bigs.Uint64Strict(l1Head.Number))
	// Ensure the block is marked as safe before we attempt to fault prove it.
	require.Equal(t, uint64(2), bigs.Uint64Strict(l2SafeHead.Number))

	defaultParam := helpers.WithPreInteropDefaults(t, bigs.Uint64Strict(l2SafeHead.Number), env.Sequencer.L2Verifier, env.Engine)
	fixtureInputParams := []helpers.FixtureInputParam{defaultParam, helpers.WithL1Head(l1Head.Hash())}
	var fixtureInputs helpers.FixtureInputs
	for _, apply := range fixtureInputParams {
		apply(&fixtureInputs)
	}

	env.RunFaultProofProgram(t, bigs.Uint64Strict(l2SafeHead.Number), helpers.ExpectNoError(), fixtureInputParams...)
}
