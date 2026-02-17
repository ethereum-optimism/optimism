package interop

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestSupernodeInteropBidirectionalMessages tests sending messages in both directions
// (A->B and B->A) to verify the supernode handles bidirectional interop correctly.
// All messages are valid, and no interruptions to the chains are expected.
func TestSupernodeInteropBidirectionalMessages(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

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
	initTriggerAtoB := randomInitTrigger(rng, eventLoggerA, 2, 10)
	initTxAtoB, initReceiptAtoB := alice.SendInitMessage(initTriggerAtoB)
	sys.L2B.WaitForBlock()
	_, execReceiptAtoB := bob.SendExecMessage(initTxAtoB, 0)

	t.Logger().Info("A->B message sent",
		"init_block", initReceiptAtoB.BlockNumber,
		"exec_block", execReceiptAtoB.BlockNumber,
	)

	// Send B -> A message
	initTriggerBtoA := randomInitTrigger(rng, eventLoggerB, 2, 10)
	initTxBtoA, initReceiptBtoA := bob.SendInitMessage(initTriggerBtoA)
	sys.L2A.WaitForBlock()
	_, execReceiptBtoA := alice.SendExecMessage(initTxBtoA, 0)

	t.Logger().Info("B->A message sent",
		"init_block", initReceiptBtoA.BlockNumber,
		"exec_block", execReceiptBtoA.BlockNumber,
	)

	// Wait for all messages to become safe
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	timeout := time.Duration(blockTime*25+60) * time.Second

	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		// All blocks should be safe
		return statusA.SafeL2.Number > bigs.Uint64Strict(initReceiptAtoB.BlockNumber) &&
			statusA.SafeL2.Number > bigs.Uint64Strict(execReceiptBtoA.BlockNumber) &&
			statusB.SafeL2.Number > bigs.Uint64Strict(execReceiptAtoB.BlockNumber) &&
			statusB.SafeL2.Number > bigs.Uint64Strict(initReceiptBtoA.BlockNumber)
	}, timeout, time.Second, "bidirectional messages should become safe")

	t.Logger().Info("bidirectional messages processed successfully")
}

// randomInitTrigger creates a random init trigger for testing.
func randomInitTrigger(rng *rand.Rand, eventLoggerAddress common.Address, topicCount, dataLen int) *txintent.InitTrigger {
	if topicCount > 4 {
		topicCount = 4 // Max 4 topics in EVM logs
	}
	if topicCount < 1 {
		topicCount = 1
	}
	if dataLen < 1 {
		dataLen = 1
	}

	topics := make([][32]byte, topicCount)
	for i := range topics {
		copy(topics[i][:], testutils.RandomData(rng, 32))
	}

	return &txintent.InitTrigger{
		Emitter:    eventLoggerAddress,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, dataLen),
	}
}
