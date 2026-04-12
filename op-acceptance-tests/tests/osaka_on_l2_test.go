package tests

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

var modexpPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000005")

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

	karstOffset := uint64(3)
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtOffset(&karstOffset)))

	activationBlock := sys.L2Chain.AwaitActivation(t, forks.Karst)
	t.Require().Greater(activationBlock.Number, uint64(0), "karst must not activate at genesis")
	preForkBlockNum := activationBlock.Number - 1
	postForkBlockNum := activationBlock.Number + 1
	sys.L2EL.WaitForBlockNumber(postForkBlockNum)

	l2Client := sys.L2EL.EthClient()

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

func TestEIP7939CLZ(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "osaka is not supported in op-geth")

	karstOffset := uint64(3)
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtOffset(&karstOffset)))

	activationBlock := sys.L2Chain.AwaitActivation(t, forks.Karst)
	t.Require().Greater(activationBlock.Number, uint64(0), "karst must not activate at genesis")
	preForkBlockNum := activationBlock.Number - 1
	postForkBlockNum := activationBlock.Number + 1
	sys.L2EL.WaitForBlockNumber(postForkBlockNum)

	l2Client := sys.L2EL.EthClient()

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
