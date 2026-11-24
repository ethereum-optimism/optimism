package superfaultproofs

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// lagoonBLSPrecompile is a BLS12-381 precompile whose input limit Lagoon lowers. kona-client runs
// these precompiles through its own accelerated code, with its own copies of the limits.
type lagoonBLSPrecompile struct {
	name string
	// addr is the low byte of the precompile's address.
	addr        byte
	elementSize int
	// maxElements is the most input elements the precompile takes from Lagoon on.
	maxElements int
	// gas lets a call with one element more succeed under the earlier limits, so only the Lagoon
	// limit can fail it. It covers the precompile's cost, the 1/64 that CALL keeps back, the 53,000
	// intrinsic gas of a creation and ~45,000 gas of memory.
	gas uint64
}

var lagoonBLSPrecompiles = []lagoonBLSPrecompile{
	// 818 points cost 818 * 12000 * 519 / 1000 = 5,094,504 gas.
	{name: "G1MSM", addr: 0x0c, elementSize: 160, maxElements: 817, gas: 6_000_000},
	// 455 points cost 455 * 22500 * 524 / 1000 = 5,364,450 gas.
	{name: "G2MSM", addr: 0x0e, elementSize: 288, maxElements: 454, gas: 6_200_000},
	// 341 pairs cost 32600 * 341 + 37700 = 11,154,300 gas.
	{name: "Pairing", addr: 0x0f, elementSize: 384, maxElements: 340, gas: 12_000_000},
}

// blsCallInitCode returns init code that calls the precompile at addr with size zero bytes of
// memory as input and reverts if the call fails. Zero bytes encode points at infinity, which are
// valid inputs. The input is built in memory because a transaction that carries an input over a
// Lagoon limit exceeds the transaction pool's 128 KiB limit on transaction size.
func blsCallInitCode(addr byte, size int) []byte {
	return []byte{
		byte(vm.PUSH0), byte(vm.PUSH0), // retLength, retOffset
		byte(vm.PUSH3), byte(size >> 16), byte(size >> 8), byte(size), // argsLength
		byte(vm.PUSH0), byte(vm.PUSH0), // argsOffset, value
		byte(vm.PUSH1), addr,
		byte(vm.GAS), byte(vm.CALL),
		byte(vm.PUSH1), 18, byte(vm.JUMPI),
		byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT),
		byte(vm.JUMPDEST), byte(vm.STOP), // offset 18
	}
}

// blsCall is one call of a BLS12-381 precompile with elements input elements.
type blsCall struct {
	precompile lagoonBLSPrecompile
	elements   int
	tx         *txplan.PlannedTx
}

// RunSingleChainSuperFaultProofLagoonBLSLimitsTest verifies the single-chain super-root transition
// into a block that calls each BLS12-381 precompile at its Lagoon input limit, which succeeds, and
// one element over it, which fails.
func RunSingleChainSuperFaultProofLagoonBLSLimitsTest(t devtest.T, sys *presets.SingleChainInterop, runners ...ProofRunner) {
	runSingleChainSuperFaultProofSmokeTest(t, sys, prepareLagoonBLSLimitsTarget, runners...)
}

// prepareLagoonBLSLimitsTarget sends every call and returns the block that holds them all.
func prepareLagoonBLSLimitsTarget(t devtest.T, sys *presets.SingleChainInterop, c *chain, _ []*chain) (uint64, uint64, uint64) {
	t.Require().True(c.Cfg.IsLagoon(c.EL.BlockRefByLabel(eth.Unsafe).Time), "Lagoon must be active")

	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		eoa := sys.FunderA.NewFundedEOA(eth.OneEther)
		nonce := eoa.PendingNonce()
		var calls []blsCall
		for _, p := range lagoonBLSPrecompiles {
			for _, elements := range []int{p.maxElements, p.maxElements + 1} {
				tx := submitTxWithoutWait(t, eoa, nonce,
					txplan.WithData(blsCallInitCode(p.addr, elements*p.elementSize)),
					txplan.WithGasLimit(p.gas),
				)
				nonce++
				calls = append(calls, blsCall{precompile: p, elements: elements, tx: tx})
			}
		}

		blocks := make(map[uint64]struct{})
		var target uint64
		for _, call := range calls {
			receipt, err := call.tx.Included.Eval(t.Ctx())
			t.Require().NoError(err, "%s call with %d elements must be included", call.precompile.name, call.elements)
			if call.elements <= call.precompile.maxElements {
				t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status, "%s call with %d elements must succeed", call.precompile.name, call.elements)
			} else {
				t.Require().Equal(types.ReceiptStatusFailed, receipt.Status, "%s call with %d elements must fail", call.precompile.name, call.elements)
			}
			target = bigs.Uint64Strict(receipt.BlockNumber)
			blocks[target] = struct{}{}
		}
		if len(blocks) != 1 {
			t.Logger().Warn("Calls landed in different blocks, retrying", "attempt", attempt, "blocks", len(blocks))
			continue
		}

		endTimestamp := c.Cfg.TimestampForBlock(target)
		return endTimestamp - 1, endTimestamp, target
	}
	t.Require().FailNowf("calls did not land in one block", "after %d attempts", maxAttempts)
	return 0, 0, 0
}
