package tests

import (
	"context"
	"encoding/binary"
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
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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

var karstSubtests = []struct{
	name string
	opt  sysgo.DeployerOption
{
	{name: "pre-karst", opt: sysgo.WithJovianAtGenesis},
	{name: "post-karst", opt: sysgo.WithKarstAtGenesis},
}

// initCodeStaticCall returns contract-creation init-code that:
//   - copies `input` from the code section into mem[0:len(input)]
//   - STATICCALLs `addr` with that input, forwarding `gas` (or all remaining
//     gas via the GAS opcode if gas == 0)
//   - REVERTs if the call returns 0 (failed); otherwise RETURNs zero bytes.
//
// As a consequence the deployment receipt's status reflects whether the
// precompile call succeeded — making it a usable EVM-behavior assertion
// from a state-changing transaction.
func initCodeStaticCall(t devtest.T, addr common.Address, gas uint64, input []byte) []byte {
	push1 := func(v byte) []byte { return []byte{byte(vm.PUSH1), v} }
	push8 := func(v uint64) []byte {
		b := make([]byte, 9)
		b[0] = byte(vm.PUSH8)
		binary.BigEndian.PutUint64(b[1:], v)
		return b
	}
	push20 := func(a common.Address) []byte { return append([]byte{byte(vm.PUSH20)}, a.Bytes()...) }

	gasPart := 9 // PUSH8 + 8 bytes
	if gas == 0 {
		gasPart = 1 // GAS opcode
	}
	// Sum the byte sizes of every code-section instruction below before the data.
	codeLen := uint64(9 + 9 + 2 + 1 + 9 + 9 + 9 + 9 + 21 + gasPart + 1 + 1 + 2 + 1 + 2 + 2 + 1 + 1 + 2 + 2 + 1)
	revertJD := codeLen - 6 // points to the JUMPDEST near the end

	inLen := uint64(len(input))

	var code []byte
	// CODECOPY input -> mem[0:inLen]
	code = append(code, push8(inLen)...)   // size
	code = append(code, push8(codeLen)...) // codeOff
	code = append(code, push1(0)...)       // memDest
	code = append(code, byte(vm.CODECOPY)) // (1)
	// STATICCALL args (push retLen, retOff, argsLen, argsOff, addr, gas)
	code = append(code, push8(0)...)     // retLen
	code = append(code, push8(0)...)     // retOff
	code = append(code, push8(inLen)...) // argsLen
	code = append(code, push8(0)...)     // argsOff
	code = append(code, push20(addr)...) // addr
	if gas == 0 {
		code = append(code, byte(vm.GAS))
	} else {
		code = append(code, push8(gas)...)
	}
	code = append(code, byte(vm.STATICCALL))
	// Branch on success
	code = append(code, byte(vm.ISZERO))
	code = append(code, push1(byte(revertJD))...)
	code = append(code, byte(vm.JUMPI))
	// Success: RETURN 0 0
	code = append(code, push1(0)...)
	code = append(code, push1(0)...)
	code = append(code, byte(vm.RETURN))
	// Revert path
	code = append(code, byte(vm.JUMPDEST))
	code = append(code, push1(0)...)
	code = append(code, push1(0)...)
	code = append(code, byte(vm.REVERT))

	t.Require().Equal(uint64(len(code)), codeLen)

	return append(code, input...)
}

func TestEIP7823UpperBoundModExp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Modexp input exceeding EIP-7823 limits: modulus length is 1025 bytes (limit is 1024).
	oversizeMod := make([]byte, 1025)
	oversizeMod[1024] = 5
	exceedingLimitInput := buildModExpInput([]byte{2}, []byte{3}, oversizeMod)

	// Init-code that STATICCALLs modexp with the oversize input and forwards
	// all remaining gas — pre-Karst the precompile accepts it (deployment
	// succeeds), post-Karst EIP-7823 rejects it (STATICCALL returns 0 →
	// init-code reverts → deployment fails).
	deployData := initCodeStaticCall(t, modexpPrecompile, 0, exceedingLimitInput)

	for _, sub := range karstSubtests {
		t.Run(sub.name, func(t devtest.T) {
			t.Parallel()
			sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sub.opt))
			eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

			tx := txplan.NewPlannedTx(
				eoa.Plan(),
				txplan.WithData(deployData),
				txplan.WithGasLimit(500_000),
			)
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "deployment tx should land on chain")

			if sub.name == "post-karst" {
				t.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status,
					"post-karst: oversized modexp must be rejected (EIP-7823)")
			} else {
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status,
					"pre-karst: oversized modexp should be accepted")
			}

			claimBlock := receipt.BlockNumber.Uint64()
			sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
			t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the modexp-7823 chain")
		})
	}
}

func TestEIP7883ModExpGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Init-code that STATICCALLs modexp with empty input and forwards exactly
	// 300 gas. The modexp gas floor is 200 pre-Karst (EIP-2565) and 500
	// post-Karst (EIP-7883), so 300 succeeds pre-fork and OOGs post-fork.
	deployData := initCodeStaticCall(t, modexpPrecompile, 300, nil)

	for _, sub := range karstSubtests {
		t.Run(sub.name, func(t devtest.T) {
			t.Parallel()
			sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sub.opt))
			eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

			tx := txplan.NewPlannedTx(
				eoa.Plan(),
				txplan.WithData(deployData),
				txplan.WithGasLimit(500_000),
			)
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "deployment tx should land on chain")

			if sub.name == "post-karst" {
				t.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status,
					"post-karst: modexp must OOG at 300 gas (floor=500)")
			} else {
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status,
					"pre-karst: modexp should succeed at 300 gas (floor=200)")
			}

			claimBlock := receipt.BlockNumber.Uint64()
			sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
			t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the modexp-7883 chain")
		})
	}
}

func TestEIP7825TxGasLimitCap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	testCases := map[string]struct {
		opt       sysgo.DeployerOption
		expectErr bool
	}{
		"pre-karst": {
			opt: sysgo.WithJovianAtGenesis,
		},
		"post-karst": {
			opt:       sysgo.WithKarstAtGenesis,
			expectErr: true,
		},
	}

	// EIP-7825 caps transaction gas at 2^24 = 16,777,216.
	// This is a tx validity rule enforced at the txpool/block level, not by the
	// EVM, so eth_call and eth_simulateV1 don't enforce it. We must send a real
	// transaction and verify the RPC rejects it.
	for name, testCase := range testCases {
		t.Run(name, func(t devtest.T) {
			t.Parallel()
			sys := presets.NewMinimal(t, presets.WithDeployerOptions(testCase.opt))

			eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

			planWithGasLimit := func(gas uint64) txplan.Option {
				return txplan.Combine(
					eoa.Plan(),
					txplan.WithGasLimit(gas),
					txplan.WithTo(&common.Address{}),
				)
			}

			_, err := txplan.NewPlannedTx(planWithGasLimit(params.MaxTxGas)).Success.Eval(t.Ctx())
			t.Require().NoError(err, "tx with gas at 2^24 should succeed")

			tx := txplan.NewPlannedTx(planWithGasLimit(params.MaxTxGas + 1))
			if testCase.expectErr {
				_, err := tx.Included.Eval(t.Ctx())
				t.Require().Error(err, "tx with gas above 2^24 should be rejected")
			} else {
				_, err := tx.Success.Eval(t.Ctx())
				t.Require().NoError(err, "tx with gas above 2^24 should succeed")
			}
		})
	}
}

func TestEIP7951P256VerifyGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// Init-code that STATICCALLs P256VERIFY forwarding exactly 3,500 gas.
	// P256VERIFY cost is 3,450 pre-Karst (RIP-7212) and 6,900 post-Karst
	// (EIP-7951), so 3,500 succeeds pre-fork and OOGs post-fork.
	deployData := initCodeStaticCall(t, p256VerifyPrecompile, 3_500, nil)

	for _, sub := range karstSubtests {
		t.Run(sub.name, func(t devtest.T) {
			t.Parallel()
			sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sub.opt))
			eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

			tx := txplan.NewPlannedTx(
				eoa.Plan(),
				txplan.WithData(deployData),
				txplan.WithGasLimit(500_000),
			)
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "deployment tx should land on chain")

			if sub.name == "post-karst" {
				t.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status,
					"post-karst: P256VERIFY must OOG at 3,500 gas (cost=6,900)")
			} else {
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status,
					"pre-karst: P256VERIFY should succeed at 3,500 gas (cost=3,450)")
			}

			claimBlock := receipt.BlockNumber.Uint64()
			sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
			t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the p256-7951 chain")
		})
	}
}

func TestEIP7939CLZ(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// EVM init-code that computes CLZ(1) and RETURNs the 32-byte result.
	// CLZ(1) = 255 because 1 has 255 leading zero bits in a uint256.
	// Pre-Karst the CLZ opcode (0x1e) is invalid → halts with all gas consumed
	// → deployment fails. Post-Karst CLZ is defined → init-code RETURNs and
	// the runtime bytecode (32 bytes ending in 0xff) is deployed.
	clzCode := []byte{
		byte(vm.PUSH1), 1,
		byte(vm.CLZ),
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	for _, sub := range karstSubtests {
		t.Run(sub.name, func(t devtest.T) {
			t.Parallel()
			sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(sub.opt))
			eoa := sys.FunderL2.NewFundedEOA(eth.OneEther)

			tx := txplan.NewPlannedTx(
				eoa.Plan(),
				txplan.WithData(clzCode),
				txplan.WithGasLimit(200_000),
			)
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "deployment tx should land on chain")

			if sub.name == "post-karst" {
				t.Require().Equal(ethtypes.ReceiptStatusSuccessful, receipt.Status,
					"post-karst: CLZ opcode should be defined")
			} else {
				t.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status,
					"pre-karst: CLZ opcode should be invalid")
			}

			claimBlock := receipt.BlockNumber.Uint64()
			sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
			t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the CLZ-7939 chain")
		})
	}
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

	claimBlock := l2Receipt.BlockNumber.Uint64()
	sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
	t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the deposit-inclusion block")
}

// TestEIP7934BlockSizeLimitDisabled proves that EIP-7934 is disabled by building a single block
// whose transaction data alone exceeds the max block size.
func TestEIP7934BlockSizeLimitDisabled(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	// EIP-7623 inflates zero-byte calldata cost to 10 gas/byte, so packing
	// 12 MB into one block requires ~120M gas.
	sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(
		sysgo.WithKarstAtGenesis,
		sysgo.WithL2GasLimit(120_000_000),
	))

	spamTxs(sys.Minimal)

	// Find a block whose total transaction data exceeds 10 MiB.
	l2Client := sys.L2EL.EthClient()
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	var claimBlock uint64
	for claimBlock == 0 {
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
				claimBlock = info.NumberU64()
			}
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err())
		}
	}

	sys.L2CL.Reached(types.LocalSafe, claimBlock, 60)
	t.Require().NoError(sys.RunKona(t, 1, claimBlock), "kona should agree on the oversize block")
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
