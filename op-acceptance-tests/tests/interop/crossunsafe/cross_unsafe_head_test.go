package crossunsafe

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// crossUnsafeHead is the JSON shape returned by op-reth's eth_crossUnsafeHead.
type crossUnsafeHead struct {
	Number hexutil.Uint64 `json:"number"`
	Hash   common.Hash    `json:"hash"`
}

// callCrossUnsafeHead invokes eth_crossUnsafeHead on the given EL node.
func callCrossUnsafeHead(t devtest.T, el *dsl.L2ELNode) (crossUnsafeHead, error) {
	var head crossUnsafeHead
	err := el.EthClient().RPC().CallContext(t.Ctx(), &head, "eth_crossUnsafeHead")
	return head, err
}

// TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessage verifies that op-reth's runtime
// cross-unsafe head advances past a block containing an executing message once the initiating
// message has been validated against the source chain.
//
// Two-chain interop, with chain B's op-reth started with --rollup.cross-unsafe-head-source-rpc
// pointing at chain A (see WithCrossUnsafeHeadSourceFromPeer). Alice initiates a message on chain
// A; Bob executes it on chain B.
//
// Crucially we require the head to reach the executing block *while that block is still above
// chain B's safe head*. The endpoint seeds its walk from the local safe head and only validates
// blocks above it, so reaching an above-safe block is proof that op-reth ran the runtime
// validation (fetched and checked the initiating message from chain A). Without this guard the
// assertion could be satisfied by the safe head simply advancing past the block on its own.
func TestCrossUnsafeHeadAdvancesPastValidatedExecutingMessage(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "eth_crossUnsafeHead is an op-reth feature")

	sys := presets.NewTwoL2SupernodeInterop(t, 0, presets.WithCrossUnsafeHeadSourceFromPeer())
	require := t.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	// Alice initiates a message on chain A (the source chain).
	eventLoggerAddress := alice.DeployEventLogger()
	initMsg := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 20))
	sys.L2A.WaitForBlock()

	// Bob executes the message on chain B (the chain running eth_crossUnsafeHead).
	execMsg := bob.SendExecMessage(initMsg)
	execBlock := execMsg.Receipt.BlockNumber.Uint64()
	logger.Info("executing message included on chain B", "block", execBlock)

	require.Eventually(func() bool {
		head, err := callCrossUnsafeHead(t, sys.L2ELB)
		if err != nil {
			logger.Warn("eth_crossUnsafeHead call failed", "err", err)
			return false
		}
		safe := sys.L2ELB.BlockRefByLabel(eth.Safe).Number
		logger.Info("cross-unsafe head", "number", uint64(head.Number), "safe", safe, "target", execBlock)
		// Proof of runtime validation: the head reached an executing-message block that is
		// still unsafe (strictly above the safe head).
		return uint64(head.Number) >= execBlock && safe < execBlock && head.Hash != (common.Hash{})
	}, 90*time.Second, 1*time.Second,
		"cross-unsafe head should validate and advance past the executing-message block while it is still unsafe")
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
	require := t.Require()
	logger := t.Logger()

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
	logger.Info("invalid executing message included on chain B (unsafe only)", "block", invalidBlock)

	// The walk should reach the block immediately before the invalid one...
	require.Eventually(func() bool {
		head, err := callCrossUnsafeHead(t, sys.L2ELB)
		if err != nil {
			logger.Warn("eth_crossUnsafeHead call failed", "err", err)
			return false
		}
		logger.Info("cross-unsafe head (approaching invalid block)", "number", uint64(head.Number), "invalidBlock", invalidBlock)
		return uint64(head.Number) >= invalidBlock-1
	}, 90*time.Second, 1*time.Second,
		"cross-unsafe head should advance up to the block before the invalid executing message")

	// ...and must never include the invalid block, even as later unsafe blocks are produced.
	require.Never(func() bool {
		head, err := callCrossUnsafeHead(t, sys.L2ELB)
		if err != nil {
			return false
		}
		if uint64(head.Number) >= invalidBlock {
			logger.Error("cross-unsafe head advanced past the invalid block", "number", uint64(head.Number), "invalidBlock", invalidBlock)
			return true
		}
		return false
	}, 20*time.Second, 1*time.Second,
		"cross-unsafe head must not advance past a block with an unvalidatable executing message")
}
