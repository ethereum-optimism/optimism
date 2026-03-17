package interop

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSupernodeInteropBidirectionalMessages tests sending messages in both directions
// (A->B and B->A) to verify the supernode handles bidirectional interop correctly.
// All messages are valid, and no interruptions to the chains are expected.
func TestSupernodeInteropBidirectionalMessages(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)

	// Create funded EOAs on both chains
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event loggers on both chains
	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()

	// Sync chains
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	rng := rand.New(rand.NewSource(54321))

	// Send A -> B message
	initMsgAtoB := alice.SendRandomInitMessage(rng, eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsgAtoB := bob.SendExecMessage(initMsgAtoB)

	t.Logger().Info("A->B message sent", "msg", execMsgAtoB)

	// Send B -> A message
	initMsgBtoA := bob.SendRandomInitMessage(rng, eventLoggerB, 2, 10)
	sys.L2A.WaitForBlock()
	execMsgBtoA := alice.SendExecMessage(initMsgBtoA)

	t.Logger().Info("B->A message sent", "msg", execMsgBtoA)

	// Wait for all messages to become safe
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	timeout := time.Duration(blockTime*25+60) * time.Second

	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		// All blocks should be safe
		return statusA.SafeL2.Number > bigs.Uint64Strict(initMsgAtoB.BlockNumber()) &&
			statusA.SafeL2.Number > bigs.Uint64Strict(execMsgBtoA.BlockNumber()) &&
			statusB.SafeL2.Number > bigs.Uint64Strict(execMsgAtoB.BlockNumber()) &&
			statusB.SafeL2.Number > bigs.Uint64Strict(initMsgBtoA.BlockNumber())
	}, timeout, time.Second, "bidirectional messages should become safe")

	t.Logger().Info("bidirectional messages processed successfully")
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()
	for _, s := range []eth.L2BlockRef{finalStatusA.SafeL2, finalStatusB.SafeL2} {
		t.Require().NotZero(t, s.Time, "SafeL2.Time was zero")
		t.Require().NotZero(t, s.L1Origin, "SafeL2.L1Origin was zero")
		t.Require().NotZero(t, s.ParentHash, "SafeL2.ParentHash was zero")
		t.Require().NotZero(t, s.Hash, "SafeL2.Hash was zero")
	}
}
