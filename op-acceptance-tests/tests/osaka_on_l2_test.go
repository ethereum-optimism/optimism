package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-karst/karsttest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestEIP7823UpperBoundModExp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Pre-Karst: the oversized modulus is accepted by the modexp precompile
		// and the call succeeds.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(),
			txplan.WithTo(&karsttest.ModExpPrecompile),
			txplan.WithData(karsttest.NewEIP7823OversizedModExpInput()),
			txplan.WithGasLimit(karsttest.EIP7823OversizedGasLimit),
		).Included.Eval(t.Ctx())
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

		agreedBlockChild, claimBlock, err := karsttest.CheckEIP7823(t.Ctx(), t.Logger(), eoa.Plan())
		t.Require().NoError(err)
		t.Require().True(sys.RunKonaNative(agreedBlockChild-1, claimBlock))
	})
}

func TestEIP7883ModExpGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Empty MODEXP calldata pads to Bsize=Esize=Msize=0, which hits exactly the
		// precompile gas floor — 200 gas under EIP-2565 (pre-Karst), 500 under
		// EIP-7883 (post-Karst). Empty calldata also avoids EIP-7623 calldata cost
		// inflation, so intrinsic gas is exactly 21,000 and tx gas limit minus
		// 21,000 is the execution budget.
		//
		// Pre-Karst: 21,000 + 300 execution gas is enough for the 200-gas floor.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(),
			txplan.WithTo(&karsttest.ModExpPrecompile),
			txplan.WithGasLimit(karsttest.EIP7883BoundaryGas),
		).Included.Eval(t.Ctx())
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

		agreedBlockChild, claimBlock, err := karsttest.CheckEIP7883(t.Ctx(), t.Logger(), eoa.Plan())
		t.Require().NoError(err)
		t.Require().True(sys.RunKonaNative(agreedBlockChild-1, claimBlock))
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
		t.Require().NoError(karsttest.CheckEIP7825(t.Ctx(), t.Logger(), eoa.Plan()))
	})
}

func TestEIP7951P256VerifyGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()

		// Empty P256VERIFY calldata: the precompile charges its full gas cost
		// regardless of input length and returns empty (input != 160 bytes).
		// Empty calldata avoids EIP-7623 calldata cost inflation, so intrinsic
		// gas is exactly 21,000 and tx gas limit minus 21,000 is the execution
		// budget.
		//   RIP-7212 (pre-Karst):  P256VERIFY costs 3,450 gas
		//   EIP-7951 (post-Karst): P256VERIFY costs 6,900 gas
		//
		// Live chain is on Jovian, so 21,000 + 3,500 execution gas is enough
		// for the call to succeed. Run kona two ways to verify it honors its
		// rollup config: with a matching Jovian config it accepts the block;
		// with Karst forced at genesis it disagrees because under EIP-7951
		// the call costs 6,900 gas and would OOG.
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

				receipt, err := txplan.NewPlannedTx(eoa.Plan(),
					txplan.WithTo(&karsttest.P256VerifyPrecompile),
					txplan.WithGasLimit(karsttest.EIP7951BoundaryGas),
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
		sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Make sure the chain is past genesis before submitting txs, so the agreed
		// block we feed kona below is always >= 1 (genesis output is not reliably
		// served by OutputAtBlock).
		sys.L2EL.WaitForBlockNumber(1)

		agreedBlockChild, claimBlock, err := karsttest.CheckEIP7951(t.Ctx(), t.Logger(), eoa.Plan())
		t.Require().NoError(err)
		t.Require().True(sys.RunKonaNative(agreedBlockChild-1, claimBlock))
	})
}

func TestEIP7939CLZ(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	t.Run("pre-karst", func(t devtest.T) {
		t.Parallel()
		sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithJovianAtGenesis))
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

		// Pre-Karst: CLZ opcode (0x1e) is invalid; the init code aborts and
		// deployment fails.
		receipt, err := txplan.NewPlannedTx(eoa.Plan(),
			txplan.WithData(karsttest.CLZBytecode),
			txplan.WithGasLimit(100_000),
		).Included.Eval(t.Ctx())
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

		claimBlock, err := karsttest.CheckEIP7939(t.Ctx(), t.Logger(), sys.L2EL.EthClient(), eoa.Plan())
		t.Require().NoError(err)
		t.Require().True(sys.RunKonaNative(claimBlock-1, claimBlock))
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
	portalAddr := sys.L2Chain.Escape().RollupConfig().DepositContractAddress

	claimBlock, err := karsttest.CheckEIP7825DepositBypass(
		t.Ctx(), t.Logger(),
		sys.L2EL.EthClient(),
		portalAddr, alice.Address(),
		alice.Plan(),
		eth.OneHundredthEther,
	)
	t.Require().NoError(err)
	t.Require().True(sys.RunKonaNative(claimBlock-1, claimBlock))
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

	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	t.Require().NoError(karsttest.CheckEIP7934BlockSizeDisabled(t.Ctx(), t.Logger(), sys.L2EL.EthClient(), l2BlockTime))
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
