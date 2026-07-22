package reorg

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSupernodeDoesNotReadoptInvalidatedBlock covers invalidation recovery when the
// sequencer is far ahead on the invalid branch — as on a live network, where the
// sequencer does not follow the supernode's deny list and keeps extending the
// invalid chain that unsafe sync then delivers back to the supernode's VNs.
//
// WHEN: an invalid Executing Message is included on chain B, and the light sequencer
// builds many more blocks on top of it before the supernode invalidates it
// THEN:
//   - The invalid block is replaced (deposits-only) as usual
//   - The far-ahead invalid branch must NOT be re-adopted from unsafe sync: the
//     replacement stays canonical while verification durably passes it
func TestSupernodeDoesNotReadoptInvalidatedBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	// op-reth only: the light-sequencer invalid-message path does not recover on op-geth (#21119).
	sysgo.SkipOnOpGeth(t, "light-sequencer invalid-message path is op-reth only (#21119)")
	// Short sequencing window: with the chain B sequencer stalled below, the chain only
	// advances once the window expires and derivation force-fills deposits-only blocks.
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithSupernodeVNSequencerForBootstrap(),
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
	)
	sys.BootstrapLightSequencersViaVNHandoff()

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)

	rng := rand.New(rand.NewSource(12345))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	invalidBlockNumber := bigs.Uint64Strict(execMsg.BlockNumber())
	invalidBlockHash := execMsg.BlockHash()
	invalidBlockTimestamp := sys.L2B.TimestampForBlockNum(invalidBlockNumber)

	// Before invalidation begins: the invalid block must be derived (its replacement is
	// requested from the derivation path), and the sequencer must be far ahead on the
	// invalid branch with the supernode VN following it.
	sys.L2BSupernodeCL.Reached(safety.LocalSafe, invalidBlockNumber, 30)
	const farAhead = 20
	sys.L2ELB.Reached(eth.Unsafe, invalidBlockNumber+farAhead, 60)
	sys.L2BSupernodeEL.Reached(eth.Unsafe, invalidBlockNumber+farAhead, 60)

	// Capture the far-ahead tip of the invalid branch while it is still canonical, then
	// stop the chain B sequencer. On a live network the sequencer does not honor the
	// supernode's deny list and keeps extending the invalid branch; the local light
	// sequencer instead heals within seconds by following the supernode's safe head,
	// which would mask the recovery behavior under test. Stopping it and re-delivering
	// the captured tip after the replacement models the live-network condition exactly.
	farTipOnInvalidBranch := sys.L2ELB.PayloadByNumber(invalidBlockNumber + farAhead)
	sys.L2BCL.StopSequencer()

	sys.Supernode.ResumeInterop()

	sys.L2BSupernodeEL.AwaitBlockReplaced(invalidBlockNumber, invalidBlockHash, 90)
	sys.L2BSupernodeCL.PostUnsafePayload(farTipOnInvalidBranch)

	// Verification must get well past the replacement while the just-delivered invalid
	// branch is never re-adopted as canonical.
	sys.Supernode.AwaitValidatedTimestampWithoutReadoption(
		invalidBlockTimestamp+10, sys.L2BSupernodeEL, invalidBlockNumber, invalidBlockHash)

	// The settled replacement must not contain the invalid exec-message tx.
	sys.L2BSupernodeEL.Reached(eth.Safe, invalidBlockNumber, 30)
	sys.L2BSupernodeEL.AssertTxNotInBlock(invalidBlockNumber, execMsg.Receipt.TxHash)
}
