package crossunsafe

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessages verifies that op-reth's runtime
// cross-unsafe head advances past blocks containing validated executing messages — exercising both
// validation paths:
//   - a cross-chain message (initiated on chain A, the configured source chain), validated via the
//     source RPC; and
//   - an intra-chain / self-reference (initiated on chain B itself), validated against op-reth's
//     own local provider (there is no source RPC for the local chain).
//
// Chain B's batcher is paused first, pinning its safe head below the executing messages, so the
// head reaching them is proof of runtime validation rather than safe-head promotion.
func TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessages(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "eth_crossUnsafeHead is an op-reth feature")

	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithCrossUnsafeHeadSourceFromPeer())
	require := t.Require()

	rng := rand.New(rand.NewSource(1234))

	// A cross-chain initiating message on chain A (the configured source chain).
	sourceA := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	crossInit := sourceA.SendInitMessage(interop.RandomInitTrigger(rng, sourceA.DeployEventLogger(), 2, 20))
	// An intra-chain (self-reference) initiating message on chain B itself.
	sourceB := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	selfInit := sourceB.SendInitMessage(interop.RandomInitTrigger(rng, sourceB.DeployEventLogger(), 2, 20))

	sys.L2A.WaitForBlock()
	sys.L2B.WaitForBlock()

	// Pin chain B's safe head below the executing messages by pausing its batcher.
	sys.L2BatcherB.Stop()

	// Execute both messages on chain B: one references chain A (validated via the source RPC), the
	// other references chain B itself (validated via the local provider).
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	crossExec := bob.SendExecMessage(crossInit)
	selfExec := bob.SendExecMessage(selfInit)
	target := max(crossExec.Receipt.BlockNumber.Uint64(), selfExec.Receipt.BlockNumber.Uint64())

	// Reaching `target` requires validating every executing message at or below it — both the
	// cross-chain message (against chain A) and the self-reference (against the local provider).
	dsl.CheckAll(t, sys.L2ELB.CrossUnsafeHeadReachedFn(target, 45))

	// With the batcher paused the safe head cannot have reached the executing messages, so the
	// advance above is attributable to runtime validation rather than safe-head promotion.
	require.Less(sys.L2ELB.BlockRefByLabel(eth.Safe).Number, target,
		"chain B safe head advanced to the executing messages despite the paused batcher; cannot attribute the cross-unsafe head to validation")
}

// TestCrossUnsafeHeadStopsAtInvalidExecutingMessage verifies that op-reth's runtime cross-unsafe
// head refuses to advance past a block whose executing message cannot be validated against the
// source chain.
//
// Bob includes a genuinely-invalid executing message on chain B (its identifier references a
// non-existent source log). Chain B's batcher is stopped first, so the block is only ever on the
// unsafe chain and is never derived from L1 — the supernode therefore does not reorg it out from
// under us, giving a stable assertion window. op-reth, fetching the source logs from chain A,
// finds no log at the claimed index and must stop the cross-unsafe walk at that block.
func TestCrossUnsafeHeadStopsAtInvalidExecutingMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "eth_crossUnsafeHead is an op-reth feature")

	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithCrossUnsafeHeadSourceFromPeer())

	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 20))
	sys.L2A.WaitForBlock()

	// Keep chain B on the unsafe chain only: with no batch submission the invalid block is never
	// derived from L1, so the supernode does not reorg it away during the assertion window.
	sys.L2BatcherB.Stop()

	// Bob includes an executing message whose identifier points at a non-existent source log.
	invalid := bob.SendInvalidExecMessage(initMsg)
	invalidBlock := invalid.Receipt.BlockNumber.Uint64()

	// The walk should reach the block immediately before the invalid one, then never include the
	// invalid block itself, even as later unsafe blocks are produced.
	dsl.CheckAll(t, sys.L2ELB.CrossUnsafeHeadReachedFn(invalidBlock-1, 45))
	dsl.CheckAll(t, sys.L2ELB.CrossUnsafeHeadStaysBelowFn(invalidBlock, 10))
}
