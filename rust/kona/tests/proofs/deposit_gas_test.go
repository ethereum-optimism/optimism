package proofs

import (
	"math/big"
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

const (
	userDepositGasLimit = uint64(100_000)
	revertDepositGas    = uint64(params.TxGas + 6) // PUSH1 0; PUSH1 0; REVERT
)

var (
	revertDepositTarget = common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")
	revertDepositCode   = common.FromHex("0x60006000fd")
)

type depositGasVariant uint8

const (
	l1InfoDeposit depositGasVariant = iota
	ordinaryDepositSuccess
	ordinaryDepositRevert
)

func TestDepositGasParity(gt *testing.T) {
	runDepositGasTest := func(gt *testing.T, testCfg *helpers.TestCfg[depositGasVariant]) {
		t := actionsHelpers.NewDefaultTesting(gt)

		if testCfg.Custom == ordinaryDepositRevert {
			// Each matrix entry owns its allocation map: honest/junk claim cases and forks may run
			// concurrently. The runtime uses only pre-Shanghai opcodes so it behaves identically at
			// every fork under test.
			allocs := *actionsHelpers.DefaultAlloc
			allocs.L2Alloc = map[common.Address]types.Account{
				revertDepositTarget: {
					Code:    revertDepositCode,
					Nonce:   1,
					Balance: new(big.Int),
				},
			}
			testCfg.Allocs = &allocs
		}

		batcherCfg := helpers.NewBatcherCfg(func(c *actionsHelpers.BatcherCfg) {
			// Calldata is the only batch transport available at every fork in this matrix.
			c.DataAvailabilityType = batcherFlags.CalldataType
		})
		env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), batcherCfg)
		preRegolith := testCfg.Hardfork.Precedence < helpers.Regolith.Precedence

		if testCfg.Custom == l1InfoDeposit {
			// Every naturally built L2 block starts with an L1-info deposit. Do not force the
			// system flag: derivation sets it only before Regolith.
			env.Sequencer.ActL2EmptyBlock(t)
		} else {
			target := env.Dp.Addresses.Bob
			if testCfg.Custom == ordinaryDepositRevert {
				target = revertDepositTarget
			}

			// Submit the ordinary deposit through the real portal, mine its L1 receipt, and let
			// the sequencer derive it when adopting the new L1 origin.
			env.Alice.L1.ActResetTxOpts(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&target)(t)
			env.Alice.L2.ActSetTxGasLimit(userDepositGasLimit)(t)
			env.Alice.ActDeposit(t)
			env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
			env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
			env.Miner.ActL1EndBlock(t)
			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActBuildToL1HeadUnsafe(t)
			env.Alice.ActCheckDepositStatus(true, testCfg.Custom == ordinaryDepositSuccess)(t)
		}

		// Inspect the exact op-geth block that contains the deposit under test.
		testedBlockRef := env.Sequencer.L2Unsafe()
		testedBlock := env.Engine.L2Chain().GetBlockByHash(testedBlockRef.Hash)
		require.NotNil(t, testedBlock)
		txs := testedBlock.Transactions()
		receipts := env.Engine.L2Chain().GetReceiptsByHash(testedBlockRef.Hash)
		require.Len(t, receipts, len(txs), "every transaction must have a receipt")

		if testCfg.Custom == l1InfoDeposit {
			require.Len(t, txs, 1, "empty block must contain only its L1-info deposit")
		} else {
			require.Len(t, txs, 2, "deposit block must contain L1-info and the portal deposit")
		}

		l1InfoTx := txs[0]
		l1InfoReceipt := receipts[0]
		require.True(t, l1InfoTx.IsDepositTx(), "L1-info transaction must be a deposit")
		require.Equal(t, preRegolith, l1InfoTx.IsSystemTx(), "L1-info system flag must transition at Regolith")
		require.Equal(t, types.ReceiptStatusSuccessful, l1InfoReceipt.Status)
		if preRegolith {
			require.Zero(t, l1InfoReceipt.GasUsed, "Bedrock system deposit must report zero gas")
		} else {
			require.NotZero(t, l1InfoReceipt.GasUsed, "post-Regolith L1-info deposit must report actual gas")
			require.Less(t, l1InfoReceipt.GasUsed, l1InfoTx.Gas(), "L1-info gas limit must exceed actual gas")
		}

		if testCfg.Custom != l1InfoDeposit {
			depositTx := txs[1]
			depositReceipt := receipts[1]
			require.True(t, depositTx.IsDepositTx(), "portal transaction must derive to a deposit")
			require.False(t, depositTx.IsSystemTx(), "ordinary portal deposit must not be a system transaction")
			require.Equal(t, userDepositGasLimit, depositTx.Gas())

			expectedStatus := uint64(types.ReceiptStatusSuccessful)
			expectedActualGas := uint64(params.TxGas)
			if testCfg.Custom == ordinaryDepositRevert {
				expectedStatus = types.ReceiptStatusFailed
				expectedActualGas = revertDepositGas
				require.Equal(t, revertDepositTarget, *depositTx.To())
			} else {
				require.Equal(t, env.Dp.Addresses.Bob, *depositTx.To())
			}
			require.Greater(t, userDepositGasLimit, expectedActualGas, "fixture must leave unused deposit gas")
			require.Equal(t, expectedStatus, depositReceipt.Status)

			expectedReportedGas := expectedActualGas
			if preRegolith {
				expectedReportedGas = userDepositGasLimit
			}
			require.Equal(t, expectedReportedGas, depositReceipt.GasUsed,
				"deposit receipt gas must follow the active fork's accounting rules")
		}

		var cumulativeGas uint64
		for i, receipt := range receipts {
			require.Equal(t, txs[i].Hash(), receipt.TxHash)
			cumulativeGas += receipt.GasUsed
			require.Equal(t, cumulativeGas, receipt.CumulativeGasUsed,
				"receipt cumulative gas must equal the sum of reported transaction gas")
		}
		require.Equal(t, cumulativeGas, testedBlock.GasUsed(),
			"header gas used must equal the final receipt cumulative gas")

		// Publish the unsafe chain, derive it back as safe, and prove only the transition into
		// the block containing the tested deposit (rather than every preceding empty block).
		l2SafeHead := env.BatchMineAndSync(t)
		require.Equal(t, testedBlockRef.Number, l2SafeHead.Number,
			"safe head must be the tested deposit block")
		require.Equal(t, testedBlockRef.Hash, l2SafeHead.Hash,
			"derivation must reproduce the op-geth block used as the proof claim")

		env.RunFaultProofProgram(t, testedBlockRef.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	forks := helpers.NewForkMatrix(helpers.Bedrock, helpers.Regolith, helpers.Karst)
	matrix := helpers.NewMatrix[depositGasVariant]()
	matrix.AddDefaultTestCasesWithName("L1InfoDeposit", l1InfoDeposit, forks, runDepositGasTest)
	matrix.AddDefaultTestCasesWithName("OrdinaryDepositSuccess", ordinaryDepositSuccess, forks, runDepositGasTest)
	matrix.AddDefaultTestCasesWithName("OrdinaryDepositRevert", ordinaryDepositRevert, forks, runDepositGasTest)
	matrix.Run(gt)
}
