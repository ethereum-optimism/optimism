package proofs

import (
	"math/big"
	"testing"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	unfundedDepositMint = big.NewInt(params.GWei)
)

type depositGasVariant uint8

const (
	l1InfoDeposit depositGasVariant = iota
	ordinaryDepositSuccess
	ordinaryDepositRevert
	// ordinaryDepositUnfunded sends more value than the sender holds on L2 after the mint. The
	// transfer fails before execution, so only the mint and the nonce increment remain. Unlike a
	// deposit that reverts, a deposit that fails this way reports its whole gas limit.
	ordinaryDepositUnfunded
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
		preCanyon := testCfg.Hardfork.Precedence < helpers.Canyon.Precedence
		l2 := env.Engine.EthClient()
		var unfundedDepositValue *big.Int

		if testCfg.Custom == l1InfoDeposit {
			// Every naturally built L2 block starts with an L1-info deposit. Do not force the
			// system flag: derivation sets it only before Regolith.
			env.Sequencer.ActL2EmptyBlock(t)
		} else {
			target := env.Dp.Addresses.Bob
			if testCfg.Custom == ordinaryDepositRevert {
				target = revertDepositTarget
			}

			// A deposit has no nonce of its own. Give Alice a nonzero L2 nonce first, so the
			// receipt's deposit nonce can only match if it is read from her account.
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
			env.Alice.L2.ActMakeTx(t)
			env.Sequencer.ActL2StartBlock(t)
			env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
			env.Sequencer.ActL2EndBlock(t)
			env.Alice.L2.ActCheckReceiptStatusOfLastTx(true)(t)

			// Submit the ordinary deposit through the real portal, mine its L1 receipt, and let
			// the sequencer derive it when adopting the new L1 origin.
			env.Alice.L1.ActResetTxOpts(t)
			env.Alice.L2.ActResetTxOpts(t)
			env.Alice.L2.ActSetTxToAddr(&target)(t)
			env.Alice.L2.ActSetTxGasLimit(userDepositGasLimit)(t)
			if testCfg.Custom == ordinaryDepositUnfunded {
				balance, err := l2.BalanceAt(t.Ctx(), env.Alice.Address(), nil)
				require.NoError(t, err)
				// One mint more than Alice holds once the mint is applied.
				unfundedDepositValue = new(big.Int).Add(balance, new(big.Int).Mul(unfundedDepositMint, big.NewInt(2)))
				env.Alice.L1.ActSetTxValue(unfundedDepositMint)(t)
				env.Alice.L2.ActSetTxValue(unfundedDepositValue)(t)
			}
			env.Alice.ActDeposit(t)
			env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
			env.Miner.ActL1IncludeTx(env.Alice.Address())(t)
			env.Miner.ActL1EndBlock(t)
			env.Sequencer.ActL1HeadSignal(t)
			env.Sequencer.ActBuildToL1HeadUnsafe(t)
			// ActCheckDepositStatus decodes the first log of the L1 receipt. With a mint, that log
			// is the lockbox's, not the deposit event; the receipt checks below cover this case.
			if testCfg.Custom != ordinaryDepositUnfunded {
				env.Alice.ActCheckDepositStatus(true, testCfg.Custom == ordinaryDepositSuccess)(t)
			}
		}

		// Inspect the exact L2 block that contains the deposit under test.
		testedBlockRef := env.Sequencer.L2Unsafe()
		txs, userTxs := env.Engine.BlockTxs(t, testedBlockRef.Hash)
		require.Empty(t, userTxs, "deposit blocks must not contain user transactions")
		testedBlock, receipts, err := env.Engine.SourceClient().FetchReceipts(t.Ctx(), testedBlockRef.Hash)
		require.NoError(t, err)
		require.Len(t, receipts, len(txs), "every deposit must have a receipt")

		if testCfg.Custom == l1InfoDeposit {
			require.Len(t, txs, 1, "empty block must contain only its L1-info deposit")
		} else {
			require.Len(t, txs, 2, "deposit block must contain L1-info and the portal deposit")
		}

		parentBlock := new(big.Int).SetUint64(testedBlockRef.Number - 1)
		testedBlockNum := new(big.Int).SetUint64(testedBlockRef.Number)
		// A deposit receipt carries the deposit nonce from Regolith and the receipt version from
		// Canyon. From Canyon both are in the receipt's consensus encoding, which FetchReceipts
		// checks against the receipts root; before Canyon the nonce is stored and served only. From
		// Regolith the RPC transaction also reports the deposit nonce as its nonce. The nonce is the
		// sender's nonce before the deposit, which is its nonce in the parent block: no block here
		// holds two txs from one sender.
		requireDepositReceiptFields := func(depositTx *optypes.DepositTx, receipt *optypes.Receipt) {
			if preRegolith {
				require.Nil(t, receipt.DepositNonce, "deposit receipts carry no deposit nonce before Regolith")
			} else {
				nonce, err := l2.NonceAt(t.Ctx(), depositTx.From, parentBlock)
				require.NoError(t, err)
				require.NotNil(t, receipt.DepositNonce, "deposit receipts carry the deposit nonce from Regolith")
				require.Equal(t, nonce, *receipt.DepositNonce, "deposit nonce must be the sender's nonce before the deposit")

				// Read as raw JSON: a decoded DepositTx has no nonce.
				var rpcTx struct {
					Nonce *hexutil.Uint64 `json:"nonce"`
				}
				require.NoError(t, env.Engine.RPCClient().CallContext(t.Ctx(), &rpcTx, "eth_getTransactionByHash", receipt.TxHash))
				require.NotNil(t, rpcTx.Nonce, "RPC deposit transactions have a nonce from Regolith")
				require.Equal(t, nonce, uint64(*rpcTx.Nonce), "the RPC nonce of a deposit must be its deposit nonce")
			}
			if preCanyon {
				require.Nil(t, receipt.DepositReceiptVersion, "deposit receipts carry no version before Canyon")
			} else {
				require.NotNil(t, receipt.DepositReceiptVersion, "deposit receipts carry a version from Canyon")
				require.Equal(t, optypes.CanyonDepositReceiptVersion, *receipt.DepositReceiptVersion)
			}
		}

		l1InfoTx := txs[0]
		l1InfoReceipt := receipts[0]
		require.Equal(t, preRegolith, l1InfoTx.IsSystemTransaction, "L1-info system flag must transition at Regolith")
		require.Equal(t, types.ReceiptStatusSuccessful, l1InfoReceipt.Status)
		if preRegolith {
			require.Zero(t, l1InfoReceipt.GasUsed, "Bedrock system deposit must report zero gas")
		} else {
			require.NotZero(t, l1InfoReceipt.GasUsed, "post-Regolith L1-info deposit must report actual gas")
			require.Less(t, l1InfoReceipt.GasUsed, l1InfoTx.Gas, "L1-info gas limit must exceed actual gas")
		}
		requireDepositReceiptFields(l1InfoTx, l1InfoReceipt)

		if testCfg.Custom != l1InfoDeposit {
			depositTx := txs[1]
			depositReceipt := receipts[1]
			require.False(t, depositTx.IsSystemTransaction, "ordinary portal deposit must not be a system transaction")
			require.Equal(t, userDepositGasLimit, depositTx.Gas)
			requireDepositReceiptFields(depositTx, depositReceipt)

			// Gas the deposit reports from Regolith on. Before Regolith every non-system
			// deposit reports its gas limit.
			var expectedReportedGas uint64
			switch testCfg.Custom {
			case ordinaryDepositSuccess:
				require.Equal(t, types.ReceiptStatusSuccessful, depositReceipt.Status)
				require.Equal(t, env.Dp.Addresses.Bob, *depositTx.To)
				expectedReportedGas = params.TxGas
			case ordinaryDepositRevert:
				require.Equal(t, types.ReceiptStatusFailed, depositReceipt.Status)
				require.Equal(t, revertDepositTarget, *depositTx.To)
				expectedReportedGas = revertDepositGas
			case ordinaryDepositUnfunded:
				require.Equal(t, types.ReceiptStatusFailed, depositReceipt.Status)
				require.Equal(t, env.Dp.Addresses.Bob, *depositTx.To)
				require.Zero(t, unfundedDepositMint.Cmp(depositTx.Mint), "deposit must mint %v, minted %v", unfundedDepositMint, depositTx.Mint)
				require.Zero(t, unfundedDepositValue.Cmp(depositTx.Value), "deposit must send %v, sent %v", unfundedDepositValue, depositTx.Value)
				expectedReportedGas = userDepositGasLimit

				// Only the mint stays: the value never reaches the target.
				for addr, delta := range map[common.Address]*big.Int{
					depositTx.From: unfundedDepositMint,
					*depositTx.To:  new(big.Int),
				} {
					before, err := l2.BalanceAt(t.Ctx(), addr, parentBlock)
					require.NoError(t, err)
					after, err := l2.BalanceAt(t.Ctx(), addr, testedBlockNum)
					require.NoError(t, err)
					want := new(big.Int).Add(before, delta)
					require.Zero(t, want.Cmp(after), "balance of %s: want %v, got %v", addr, want, after)
				}
			}
			if testCfg.Custom != ordinaryDepositUnfunded {
				require.Greater(t, userDepositGasLimit, expectedReportedGas, "fixture must leave unused deposit gas")
			}
			if preRegolith {
				expectedReportedGas = userDepositGasLimit
			}
			require.Equal(t, expectedReportedGas, depositReceipt.GasUsed,
				"deposit receipt gas must follow the active fork's accounting rules")

			// Every deposit increments the sender's nonce, whether it succeeds or fails.
			nonceBefore, err := l2.NonceAt(t.Ctx(), depositTx.From, parentBlock)
			require.NoError(t, err)
			require.NotZero(t, nonceBefore, "the sender must have a nonzero nonce before the deposit")
			nonceAfter, err := l2.NonceAt(t.Ctx(), depositTx.From, testedBlockNum)
			require.NoError(t, err)
			require.Equal(t, nonceBefore+1, nonceAfter, "deposit must increment the sender's nonce")
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
			"derivation must reproduce the engine-built block used as the proof claim")

		env.RunFaultProofProgram(t, testedBlockRef.Number, testCfg.CheckResult, testCfg.InputParams...)
	}

	forks := helpers.NewForkMatrix(helpers.Bedrock, helpers.Regolith, helpers.Karst)
	matrix := helpers.NewMatrix[depositGasVariant]()
	matrix.AddDefaultTestCasesWithName("L1InfoDeposit", l1InfoDeposit, forks, runDepositGasTest)
	matrix.AddDefaultTestCasesWithName("OrdinaryDepositSuccess", ordinaryDepositSuccess, forks, runDepositGasTest)
	matrix.AddDefaultTestCasesWithName("OrdinaryDepositRevert", ordinaryDepositRevert, forks, runDepositGasTest)
	matrix.AddDefaultTestCasesWithName("OrdinaryDepositUnfunded", ordinaryDepositUnfunded, forks, runDepositGasTest)
	matrix.Run(gt)
}
