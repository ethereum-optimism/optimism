package tests

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
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

// setupKarstForkTest creates a minimal system with Karst activated at block offset 3
// and returns the L2 client along with pre-fork and post-fork block numbers.
func setupKarstForkTest(t devtest.T) (l2Client apis.EthClient, preForkBlockNum, postForkBlockNum uint64) {
	karstOffset := uint64(3)
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtOffset(&karstOffset)))

	activationBlock := sys.L2Chain.AwaitActivation(t, forks.Karst)
	t.Require().Greater(activationBlock.Number, uint64(0), "karst must not activate at genesis")
	preForkBlockNum = activationBlock.Number - 1
	postForkBlockNum = activationBlock.Number + 1
	sys.L2EL.WaitForBlockNumber(postForkBlockNum)

	l2Client = sys.L2EL.EthClient()
	return
}

func TestEIP7823UpperBoundModExp(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	l2Client, preForkBlockNum, postForkBlockNum := setupKarstForkTest(t)

	// Modexp input exceeding EIP-7823 limits: modulus length is 1025 bytes (limit is 1024)
	oversizeMod := make([]byte, 1025)
	oversizeMod[1024] = 5
	exceedingLimitInput := buildModExpInput([]byte{2}, []byte{3}, oversizeMod)

	// Pre-fork: oversized modexp input should succeed (EIP-7823 not yet active)
	result, err := l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:   &modexpPrecompile,
		Data: exceedingLimitInput,
	}, rpc.BlockNumber(preForkBlockNum))
	t.Require().NoError(err)
	t.Require().Len(result, 1025, "pre-fork: modexp with oversized input should return 1025-byte result")

	// Post-fork: oversized modexp input should fail (EIP-7823 enforced)
	result, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:   &modexpPrecompile,
		Data: exceedingLimitInput,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().Error(err)
	t.Require().Empty(result, "post-fork: modexp with oversized input should return empty result due to EIP-7823")

	// Post-fork: within-limit modexp input should still succeed
	result, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:   &modexpPrecompile,
		Data: buildModExpInput([]byte{2}, []byte{3}, []byte{5}),
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().NoError(err)
	t.Require().Equal([]byte{3}, result, "2^3 mod 5 should equal 3")
}

func TestEIP7883ModExpGasCostIncrease(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	l2Client, preForkBlockNum, postForkBlockNum := setupKarstForkTest(t)

	// Call modexp with empty calldata. The precompile pads missing bytes with
	// zeros, giving Bsize=0, Esize=0, Msize=0. This hits exactly the gas floor:
	//   EIP-2565 (pre-Karst):  max(200, floor(0*0/3)) = 200 gas
	//   EIP-7883 (post-Karst): max(500, floor(0*0))   = 500 gas
	// Empty calldata also avoids EIP-7623 calldata cost inflation, so intrinsic
	// gas is just 21,000 and we can precisely control execution gas via Gas limit.

	// Pre-fork: 21,000 + 300 execution gas is enough for 200-gas floor.
	_, err := l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &modexpPrecompile,
		Gas: 21_300,
	}, rpc.BlockNumber(preForkBlockNum))
	t.Require().NoError(err, "pre-fork: modexp should succeed with 300 execution gas (floor is 200)")

	// Post-fork: 21,000 + 300 execution gas is NOT enough for 500-gas floor.
	_, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &modexpPrecompile,
		Gas: 21_300,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().Error(err, "post-fork: modexp should fail with 300 execution gas (floor is 500)")

	// Post-fork: 21,000 + 600 execution gas is enough for 500-gas floor.
	_, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &modexpPrecompile,
		Gas: 21_600,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().NoError(err, "post-fork: modexp should succeed with 600 execution gas (floor is 500)")
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

	l2Client, preForkBlockNum, postForkBlockNum := setupKarstForkTest(t)

	// Call P256VERIFY with empty calldata. The precompile charges its full gas
	// cost regardless of input length, then returns empty (input != 160 bytes).
	// Empty calldata avoids EIP-7623 calldata cost inflation, so intrinsic gas
	// is just 21,000 and we can precisely control execution gas via gas limit.
	//   RIP-7212 (pre-Karst):  P256VERIFY costs 3,450 gas
	//   EIP-7951 (post-Karst): P256VERIFY costs 6,900 gas

	// Pre-fork: 21,000 + 3,500 execution gas is enough for 3,450-gas precompile.
	_, err := l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &p256VerifyPrecompile,
		Gas: 24_500,
	}, rpc.BlockNumber(preForkBlockNum))
	t.Require().NoError(err, "pre-fork: P256VERIFY should succeed with 3,500 execution gas (cost is 3,450)")

	// Post-fork: 21,000 + 3,500 execution gas is NOT enough for 6,900-gas precompile.
	_, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &p256VerifyPrecompile,
		Gas: 24_500,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().Error(err, "post-fork: P256VERIFY should fail with 3,500 execution gas (cost is 6,900)")

	// Post-fork: 21,000 + 7,000 execution gas is enough for 6,900-gas precompile.
	_, err = l2Client.Call(t.Ctx(), ethereum.CallMsg{
		To:  &p256VerifyPrecompile,
		Gas: 28_000,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().NoError(err, "post-fork: P256VERIFY should succeed with 7,000 execution gas (cost is 6,900)")
}

func TestEIP7939CLZ(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	l2Client, preForkBlockNum, postForkBlockNum := setupKarstForkTest(t)

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

	// Pre-fork: CLZ opcode (0x1e) is not yet valid, so execution should fail.
	_, err := l2Client.Call(t.Ctx(), ethereum.CallMsg{
		Data: clzCode,
	}, rpc.BlockNumber(preForkBlockNum))
	t.Require().Error(err, "pre-fork: CLZ opcode should not be available")

	// Post-fork: CLZ opcode is valid, execution should succeed.
	result, err := l2Client.Call(t.Ctx(), ethereum.CallMsg{
		Data: clzCode,
	}, rpc.BlockNumber(postForkBlockNum))
	t.Require().NoError(err, "post-fork: CLZ opcode should be available")
	expected := common.LeftPadBytes([]byte{0xff}, 32) // 255 as uint256
	t.Require().Equal(expected, result, "CLZ(1) should equal 255")
}

// TestEIP7825DepositBypassesTxGasLimitCap proves that deposit transactions are not
// subject to the EIP-7825 2^24 gas cap introduced by Karst. Deposits are forced onto
// L2 by the derivation pipeline rather than passing through the txpool, so the cap
// — which is a tx validity rule — must not apply to them; otherwise an attacker could
// trivially brick the rollup by submitting deposits that can never be included.
func TestEIP7825DepositBypassesTxGasLimitCap(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))
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
