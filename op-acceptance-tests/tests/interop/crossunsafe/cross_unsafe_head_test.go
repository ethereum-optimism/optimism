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

// TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessage verifies that op-reth's runtime
// cross-unsafe head advances past a block containing an executing message once the initiating
// message has been validated against the source chain.
//
// Two-chain interop, with chain B's op-reth started with --rollup.cross-unsafe-head-source-rpc
// pointing at chain A (see WithCrossUnsafeHeadSourceFromPeer). Alice initiates a message on chain
// A; Bob executes it on chain B.
//
// To make the result unambiguous, chain B's batcher is paused first: with no batch submission its
// safe head cannot advance to the executing block, so the cross-unsafe head can only reach that
// block by runtime-validating it as an unsafe block against chain A — not because the safe head
// caught up to it.
func TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "eth_crossUnsafeHead is an op-reth feature")

	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithCrossUnsafeHeadSourceFromPeer())
	require := t.Require()

	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	// Alice initiates a message on chain A (the source chain).
	eventLoggerAddress := alice.DeployEventLogger()
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 20))
	sys.L2A.WaitForBlock()

	// Pin chain B's safe head below the executing message by pausing its batcher.
	sys.L2BatcherB.Stop()

	// Bob executes the message on chain B (the chain running eth_crossUnsafeHead).
	execMsg := bob.SendExecMessage(initMsg)
	execBlock := execMsg.Receipt.BlockNumber.Uint64()

	// The cross-unsafe head must reach the executing block, which requires op-reth to validate
	// the initiating message against chain A.
	dsl.CheckAll(t, sys.L2ELB.CrossUnsafeHeadReachedFn(execBlock, 45))

	// With the batcher paused the safe head cannot have reached the executing block, so the
	// advance above is attributable to runtime validation rather than safe-head promotion.
	require.Less(sys.L2ELB.BlockRefByLabel(eth.Safe).Number, execBlock,
		"chain B safe head advanced to the executing block despite the paused batcher; cannot attribute the cross-unsafe head to validation")
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

// TestCrossUnsafeHeadAdvancesPastSelfReferencingMessage verifies that op-reth validates an
// intra-chain (self-referencing) executing message against its own local provider rather than a
// source RPC.
//
// Both the initiating and executing messages are on chain B, so the executing message's identifier
// chain ID equals chain B's own ID. There is no source RPC configured for the local chain, so the
// head can only reach the executing block if op-reth takes the local-provider validation path.
// As in the cross-chain test, chain B's batcher is paused so reaching the block (while it is still
// above the safe head) is proof of runtime validation rather than safe-head promotion.
func TestCrossUnsafeHeadAdvancesPastSelfReferencingMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "eth_crossUnsafeHead is an op-reth feature")

	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithCrossUnsafeHeadSourceFromPeer())
	require := t.Require()

	rng := rand.New(rand.NewSource(1234))
	// Both EOAs are on chain B: the executing message references an initiating message on the same
	// chain.
	alice := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 20))
	sys.L2B.WaitForBlock()

	// Pin chain B's safe head below the executing block.
	sys.L2BatcherB.Stop()

	execMsg := bob.SendExecMessage(initMsg)
	execBlock := execMsg.Receipt.BlockNumber.Uint64()

	// Reaching the executing block requires validating the same-chain initiating message against
	// the local provider.
	dsl.CheckAll(t, sys.L2ELB.CrossUnsafeHeadReachedFn(execBlock, 45))

	require.Less(sys.L2ELB.BlockRefByLabel(eth.Safe).Number, execBlock,
		"chain B safe head advanced to the executing block despite the paused batcher; cannot attribute the cross-unsafe head to validation")
}
