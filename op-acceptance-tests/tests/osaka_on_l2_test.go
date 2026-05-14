package tests

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

var modexpPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000005")
var p256VerifyPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000100")

// buildModExpInput constructs input data for the MODEXP precompile (address 0x05).
// Format: <Bsize (32 bytes)> <Esize (32 bytes)> <Msize (32 bytes)> <B> <E> <M>
func buildModExpInput(base, exp, mod []byte) []byte {
	input := make([]byte, 0, 96+len(base)+len(exp)+len(mod))
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(base))).Bytes(), 32)...)
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(exp))).Bytes(), 32)...)
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(mod))).Bytes(), 32)...)
	input = append(input, base...)
	input = append(input, exp...)
	input = append(input, mod...)
	return input
}

func TestEIP7823UpperBoundModExp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Modexp input exceeding EIP-7823 limits: modulus length is 1025 bytes (limit is 1024)
	oversizeMod := make([]byte, 1025)
	oversizeMod[1024] = 5
	planOversized := txplan.Combine(
		txplan.WithTo(&modexpPrecompile),
		txplan.WithData(buildModExpInput([]byte{2}, []byte{3}, oversizeMod)),
		txplan.WithGasLimit(2_000_000),
	)
	withinLimitInput := buildModExpInput([]byte{2}, []byte{3}, []byte{5})

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		receipt, err := txplan.NewPlannedTx(eoa.Plan(), planOversized).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status)
	})

	t.Run("post-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Make sure the chain is past genesis before submitting txs, so the agreed
		// block we feed kona below is always >= 1 (genesis output is not reliably
		// served by OutputAtBlock).
		sys.L2EL.WaitForBlockNumber(1)

		// Post-Karst: oversized modulus is rejected, so the tx is included but reverts.
		oversizedReceipt, err := txplan.NewPlannedTx(eoa.Plan(), planOversized).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusFailed, oversizedReceipt.Status)

		// Post-Karst: within-limit modulus still works.
		withinLimitReceipt, err := txplan.NewPlannedTx(
			eoa.Plan(),
			txplan.WithTo(&modexpPrecompile),
			txplan.WithData(withinLimitInput),
			txplan.WithGasLimit(200_000),
		).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, withinLimitReceipt.Status)

		// Cross-check kona-host agrees with the live chain over a range that
		// covers both the rejected oversized call and the within-limit success.
		agreedBlock := bigs.Uint64Strict(oversizedReceipt.BlockNumber) - 1
		claimBlock := bigs.Uint64Strict(withinLimitReceipt.BlockNumber)
		t.Require().True(sys.RunKonaNative(agreedBlock, claimBlock))
	})
}

func TestEIP7883ModExpGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Call modexp with empty calldata. The precompile pads missing bytes with
	// zeros, giving Bsize=0, Esize=0, Msize=0. This hits exactly the gas floor:
	//   EIP-2565 (pre-Karst):  max(200, floor(0*0/3)) = 200 gas
	//   EIP-7883 (post-Karst): max(500, floor(0*0))   = 500 gas
	// Empty calldata also avoids EIP-7623 calldata cost inflation, so intrinsic
	// gas is exactly 21,000 and we can precisely control execution gas via the tx gas limit.
	planUnderGas := txplan.Combine(
		txplan.WithTo(&modexpPrecompile),
		txplan.WithGasLimit(21_300),
	)

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Pre-Karst: 21,000 + 300 execution gas is enough for 200-gas floor.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(), planUnderGas).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status)
	})

	t.Run("post-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Make sure the chain is past genesis before submitting txs, so the agreed
		// block we feed kona below is always >= 1 (genesis output is not reliably
		// served by OutputAtBlock).
		sys.L2EL.WaitForBlockNumber(1)

		// Post-Karst: 21,000 + 300 execution gas is NOT enough for 500-gas floor.
		underGasReceipt, err := txplan.NewPlannedTx(eoa.Plan(), planUnderGas).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusFailed, underGasReceipt.Status)

		// Post-Karst: 21,000 + 600 execution gas is enough for 500-gas floor.
		sufficientReceipt, err := txplan.NewPlannedTx(
			eoa.Plan(),
			txplan.WithTo(&modexpPrecompile),
			txplan.WithGasLimit(21_600),
		).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, sufficientReceipt.Status)

		// Cross-check kona-host agrees with the live chain over a range that
		// covers both the OOG case and the within-floor success.
		agreedBlock := bigs.Uint64Strict(underGasReceipt.BlockNumber) - 1
		claimBlock := bigs.Uint64Strict(sufficientReceipt.BlockNumber)
		t.Require().True(sys.RunKonaNative(agreedBlock, claimBlock))
	})
}

func TestEIP7825TxGasLimitCap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// EIP-7825 caps transaction gas at 2^24 = 16,777,216.
	// This is a tx validity rule enforced at the txpool/block level, not by the
	// EVM, so eth_call and eth_simulateV1 don't enforce it.

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()

		// Live chain is on Jovian (no cap), so a tx with gas > 2^24 lands
		// successfully. Run kona two ways to verify it honors its rollup
		// config: with a matching Jovian config it accepts the block, with
		// Karst forced at genesis it rejects the block on EIP-7825 grounds.
		cases := []struct {
			name        string
			konaOpts    []presets.Option
			konaAccepts bool
		}{
			{
				name:        "kona-with-jovian",
				konaAccepts: true,
			},
			{
				name:        "kona-with-karst",
				konaOpts:    []presets.Option{presets.WithKonaKarstAtGenesis()},
				konaAccepts: false,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t devtest.T) {
				t.Parallel()
				opts := append(
					[]presets.Option{presets.WithDeployerOptions(sysgo.WithJovianAtGenesis)},
					tc.konaOpts...,
				)
				sys := presets.NewMinimalWithKona(t, opts...)
				sys.L2EL.WaitForBlockNumber(1)
				eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

				receipt, err := txplan.NewPlannedTx(
					eoa.Plan(),
					txplan.WithTo(&common.Address{}),
					txplan.WithGasLimit(params.MaxTxGas+1),
				).Included.Eval(t.Ctx())
				t.Require().NoError(err)
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status)

				agreedBlock := bigs.Uint64Strict(receipt.BlockNumber) - 1
				claimBlock := bigs.Uint64Strict(receipt.BlockNumber)
				t.Require().Equal(tc.konaAccepts, sys.RunKonaNative(agreedBlock, claimBlock))
			})
		}
	})

	t.Run("post-karst", func(t devtest.T) {
		t.Parallel()
		// Live chain is on Karst — op-reth's RPC rejects a tx with gas > 2^24
		// at submission time, so it never lands on-chain. Nothing for kona to
		// validate.
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		_, err := txplan.NewPlannedTx(
			eoa.Plan(),
			txplan.WithTo(&common.Address{}),
			txplan.WithGasLimit(params.MaxTxGas+1),
		).Included.Eval(t.Ctx())
		t.Require().Error(err)
	})
}

func TestEIP7951P256VerifyGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Call P256VERIFY with empty calldata. The precompile charges its full gas
	// cost regardless of input length, then returns empty (input != 160 bytes).
	// Empty calldata avoids EIP-7623 calldata cost inflation, so intrinsic gas
	// is just 21,000 and we can precisely control execution gas via gas limit.
	//   RIP-7212 (pre-Karst):  P256VERIFY costs 3,450 gas
	//   EIP-7951 (post-Karst): P256VERIFY costs 6,900 gas
	planUnderGas := txplan.Combine(
		txplan.WithTo(&p256VerifyPrecompile),
		txplan.WithGasLimit(24_500),
	)

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()

		// Live chain is on Jovian (RIP-7212, P256VERIFY costs 3,450 gas), so
		// 21,000 + 3,500 execution gas is enough for the call to succeed. Run
		// kona two ways to verify it honors its rollup config: with a matching
		// Jovian config it accepts the block, with Karst forced at genesis it
		// disagrees because under EIP-7951 the call costs 6,900 gas and would
		// OOG instead.
		cases := []struct {
			name        string
			konaOpts    []presets.Option
			konaAccepts bool
		}{
			{
				name:        "kona-with-jovian",
				konaAccepts: true,
			},
			{
				name:        "kona-with-karst",
				konaOpts:    []presets.Option{presets.WithKonaKarstAtGenesis()},
				konaAccepts: false,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t devtest.T) {
				t.Parallel()
				opts := append(
					[]presets.Option{presets.WithDeployerOptions(sysgo.WithJovianAtGenesis)},
					tc.konaOpts...,
				)
				sys := presets.NewMinimalWithKona(t, opts...)
				sys.L2EL.WaitForBlockNumber(1)
				eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

				// Pre-Karst: 21,000 + 3,500 execution gas is enough for 3,450-gas precompile.
				receipt, err := txplan.NewPlannedTx(eoa.Plan(), planUnderGas).Included.Eval(t.Ctx())
				t.Require().NoError(err)
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status)

				agreedBlock := bigs.Uint64Strict(receipt.BlockNumber) - 1
				claimBlock := bigs.Uint64Strict(receipt.BlockNumber)
				t.Require().Equal(tc.konaAccepts, sys.RunKonaNative(agreedBlock, claimBlock))
			})
		}
	})

	t.Run("post-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Make sure the chain is past genesis before submitting txs, so the agreed
		// block we feed kona below is always >= 1 (genesis output is not reliably
		// served by OutputAtBlock).
		sys.L2EL.WaitForBlockNumber(1)

		// Post-Karst: 21,000 + 3,500 execution gas is NOT enough for 6,900-gas precompile.
		underGasReceipt, err := txplan.NewPlannedTx(eoa.Plan(), planUnderGas).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusFailed, underGasReceipt.Status)

		// Post-Karst: 21,000 + 7,000 execution gas is enough for 6,900-gas precompile.
		sufficientReceipt, err := txplan.NewPlannedTx(
			eoa.Plan(),
			txplan.WithTo(&p256VerifyPrecompile),
			txplan.WithGasLimit(28_000),
		).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, sufficientReceipt.Status)

		// Cross-check kona-host agrees with the live chain over a range that
		// covers both the OOG case and the within-cost success.
		agreedBlock := bigs.Uint64Strict(underGasReceipt.BlockNumber) - 1
		claimBlock := bigs.Uint64Strict(sufficientReceipt.BlockNumber)
		t.Require().True(sys.RunKonaNative(agreedBlock, claimBlock))
	})
}

func TestEIP7939CLZ(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// EVM init code that computes CLZ(1) and returns the 32-byte result.
	// CLZ(1) = 255 because 1 has 255 leading zero bits in a uint256.
	clzCode := []byte{
		byte(vm.PUSH1), 1, // stack: [1]
		byte(vm.CLZ),      // stack: [255] (1 has 255 leading zeros)
		byte(vm.PUSH1), 0, // stack: [0, 255]
		byte(vm.MSTORE),    // mem[0:32] = 255
		byte(vm.PUSH1), 32, // stack: [32]
		byte(vm.PUSH1), 0, // stack: [0, 32]
		byte(vm.RETURN), // return mem[0:32]
	}
	deployPlan := txplan.Combine(
		txplan.WithData(clzCode),
		txplan.WithGasLimit(100_000),
	)

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Pre-Karst: CLZ opcode (0x1e) is invalid; the init code aborts and
		// deployment fails.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(), deployPlan).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status)
	})

	t.Run("post-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Make sure the chain is past genesis before submitting txs, so the agreed
		// block we feed kona below is always >= 1 (genesis output is not reliably
		// served by OutputAtBlock).
		sys.L2EL.WaitForBlockNumber(1)

		// Post-Karst: CLZ executes; init code returns 32 bytes; deployment succeeds.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(), deployPlan).Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status)

		// The deployed code IS the 32-byte CLZ(1) result.
		deployedCode, err := sys.L2EL.EthClient().CodeAtHash(t.Ctx(), receipt.ContractAddress, receipt.BlockHash)
		t.Require().NoError(err)
		t.Require().Equal(common.LeftPadBytes([]byte{0xff}, 32), deployedCode, "CLZ(1) should equal 255")

		// Cross-check kona-host agrees with the live chain over the block where
		// CLZ was executed.
		agreedBlock := bigs.Uint64Strict(receipt.BlockNumber) - 1
		claimBlock := bigs.Uint64Strict(receipt.BlockNumber)
		t.Require().True(sys.RunKonaNative(agreedBlock, claimBlock))
	})
}

// TestEIP7825DepositBypassesTxGasLimitCap proves that deposit transactions are not
// subject to the EIP-7825 2^24 gas cap introduced by Karst. Deposits are forced onto
// L2 by the derivation pipeline rather than passing through the txpool, so the cap
// — which is a tx validity rule — must not apply to them; otherwise an attacker could
// trivially brick the rollup by submitting deposits that can never be included.
func TestEIP7825DepositBypassesTxGasLimitCap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
	sys.L1Network.WaitForOnline()
	sys.L2EL.WaitForBlockNumber(1)

	alice := sys.FunderL1.NewFundedEOA(eth.OneEther)
	alicel2 := alice.AsEL(sys.L2EL)

	portalAddr := sys.L2Chain.Escape().RollupConfig().DepositContractAddress
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(sys.L1EL.EthClient()),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t),
	)

	// Deposit with gas limit above the EIP-7825 cap of 2^24 = 16,777,216.
	depositGasLimit := params.MaxTxGas + 1
	depositAmount := eth.OneHundredthEther
	args := portal.DepositTransaction(alice.Address(), depositAmount, depositGasLimit, false, []byte{})
	// Skip eth_estimateGas: the estimator in txplan caps its binary search at
	// params.MaxTxGas, but ResourceMetering's Burn.gas inside depositTransaction
	// needs to burn ~depositGasLimit gas on L1, so estimation would run out of gas.
	l1Receipt := contract.Write(alice, args,
		txplan.WithValue(depositAmount),
		txplan.WithGasLimit(depositGasLimit+1_000_000),
	)
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l1Receipt.Status)

	var l2DepositTx *ethtypes.DepositTx
	for _, log := range l1Receipt.Logs {
		var err error
		if l2DepositTx, err = derive.UnmarshalDepositLogEvent(log); err == nil {
			break
		}
	}
	t.Require().NotNil(l2DepositTx, "no TransactionDeposited event in L1 receipt")
	t.Require().Equal(depositGasLimit, l2DepositTx.Gas, "L2 deposit tx gas should match the requested gas limit")

	sys.L2EL.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(l1Receipt.BlockNumber), 120)
	l2Receipt := sys.L2EL.WaitForReceipt(ethtypes.NewTx(l2DepositTx).Hash())
	t.Require().Equal(ethtypes.ReceiptStatusSuccessful, l2Receipt.Status, "deposit should be included and succeed on L2")

	alicel2.WaitForBalance(depositAmount)

	// Cross-check kona-host agrees with the live chain on the block where the
	// high-gas deposit landed — proves kona implements the deposit-side bypass
	// of EIP-7825 (deposits are not subject to the 2^24 cap).
	agreedBlock := bigs.Uint64Strict(l2Receipt.BlockNumber) - 1
	claimBlock := bigs.Uint64Strict(l2Receipt.BlockNumber)
	t.Require().True(sys.RunKonaNative(agreedBlock, claimBlock))
}

// TestEIP7934BlockSizeLimitDisabled proves that EIP-7934 is disabled by building a single block
// whose transaction data alone exceeds the max block size.
func TestEIP7934BlockSizeLimitDisabled(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// EIP-7623 inflates zero-byte calldata cost to 10 gas/byte, so packing
	// 12 MB into one block requires ~120M gas.
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(
		sysgo.WithKarstAtGenesis,
		sysgo.WithL2GasLimit(120_000_000),
	))

	spamTxs(sys)

	// Find a block whose total transaction data exceeds 10 MiB.
	l2Client := sys.L2EL.EthClient()
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	for {
		select {
		case <-time.After(l2BlockTime):
			info, blockTxs, err := l2Client.InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
			t.Require().NoError(err)

			var totalTxSize int
			for _, tx := range blockTxs {
				bin, err := tx.MarshalBinary()
				t.Require().NoError(err)
				totalTxSize += len(bin)
			}

			t.Logger().Info("Checking L2 block...", "number", info.NumberU64(), "size", totalTxSize, "gasUsed", info.GasUsed())

			// We use tx data size instead of the total block size since we don't have a client
			// capable of deserializing block responses.
			if totalTxSize > params.MaxBlockSize {
				return
			}
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err())
		}
	}
}

func spamTxs(sys *presets.Minimal) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	eoas := loadtest.FundEOAs(sys.T, eth.HundredEther, 50, l2BlockTime, sys.L2EL, sys.Wallet, sys.FaucetL2)
	eoasRR := loadtest.NewRoundRobin(eoas)
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		// Max tx size in op-geth and op-reth mempools is 128 kB per tx.
		// We leave an 8 kB buffer for tx data outside the calldata.
		const calldataSize = 120 * 1024
		_, err := eoasRR.Get().Include(t,
			txplan.WithTo(&predeploys.L1BlockAddr),
			txplan.WithData(make([]byte, calldataSize)),
			txplan.WithGasLimit(1_250_000),
		)
		return err
	})
	schedule := loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(50))

	ctx, cancel := context.WithCancel(sys.T.Ctx())
	var wg sync.WaitGroup
	wg.Add(1)
	sys.T.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	go func() {
		defer wg.Done()
		schedule.Run(sys.T.WithCtx(ctx), spammer)
	}()
}
