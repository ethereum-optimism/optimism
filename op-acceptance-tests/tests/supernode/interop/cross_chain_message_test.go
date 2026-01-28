package interop

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// TestSupernodeInteropValidInitExecMessage tests that a valid initiating message
// followed by a valid executing message are correctly processed by the supernode's
// interop verification activity.
//
// This test confirms:
// - An init message can be sent on Chain A
// - An exec message referencing the init can be sent on Chain B
// - Both chains continue to advance their safe heads after the messages
// - The supernode's interop activity processes both timestamps correctly
func TestSupernodeInteropValidInitExecMessage(gt *testing.T) {
	t := devtest.SerialT(gt) // Serial because we need precise ordering
	sys := presets.NewTwoL2SupernodeInterop(t)

	// Create funded EOAs for sending transactions
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	// Deploy event logger contract on chain A
	eventLoggerAddress := alice.DeployEventLogger()

	// Wait for chain B to catch up to chain A if necessary
	sys.L2B.CatchUpTo(sys.L2A)

	// Record initial safe heads
	initialSafeA := sys.L2ACL.SyncStatus().SafeL2.Number
	initialSafeB := sys.L2BCL.SyncStatus().SafeL2.Number

	t.Logger().Info("initial state",
		"chainA_safe", initialSafeA,
		"chainB_safe", initialSafeB,
		"event_logger", eventLoggerAddress,
	)

	// Send initiating message on chain A
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initTrigger := randomInitTrigger(rng, eventLoggerAddress, rng.Intn(3)+1, rng.Intn(10)+1)
	initTx, initReceipt := alice.SendInitMessage(initTrigger)

	t.Logger().Info("init message sent",
		"block_number", initReceipt.BlockNumber,
		"block_hash", initReceipt.BlockHash,
	)

	// Wait for at least one block between init and exec
	sys.L2B.WaitForBlock()

	// Send executing message on chain B
	_, execReceipt := bob.SendExecMessage(initTx, 0) // 0 = first log index

	t.Logger().Info("exec message sent",
		"block_number", execReceipt.BlockNumber,
		"block_hash", execReceipt.BlockHash,
	)

	// Wait for safe heads to advance past both message blocks
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	timeout := time.Duration(blockTime*20+60) * time.Second

	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		initBlockNum := initReceipt.BlockNumber.Uint64()
		execBlockNum := execReceipt.BlockNumber.Uint64()

		t.Logger().Debug("waiting for safe heads to pass message blocks",
			"chainA_safe", statusA.SafeL2.Number,
			"chainB_safe", statusB.SafeL2.Number,
			"init_block", initBlockNum,
			"exec_block", execBlockNum,
		)

		return statusA.SafeL2.Number > initBlockNum && statusB.SafeL2.Number > execBlockNum
	}, timeout, time.Second, "safe heads should advance past message blocks")

	// Verify the chains are still healthy and advancing
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("test complete",
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
	)

	// Both chains should have advanced significantly
	t.Require().Greater(finalStatusA.SafeL2.Number, initialSafeA, "chain A safe head should advance")
	t.Require().Greater(finalStatusB.SafeL2.Number, initialSafeB, "chain B safe head should advance")
}

// TestSupernodeInteropMultipleMessages tests sending multiple init/exec message pairs
// and verifying the supernode processes them all correctly.
func TestSupernodeInteropMultipleMessages(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	// Create funded EOAs
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event logger
	eventLoggerAddress := alice.DeployEventLogger()

	// Sync chains
	sys.L2B.CatchUpTo(sys.L2A)

	// Send multiple message pairs
	numPairs := 3
	rng := rand.New(rand.NewSource(12345))

	type messagePair struct {
		initBlockNum uint64
		execBlockNum uint64
	}
	pairs := make([]messagePair, 0, numPairs)

	for i := 0; i < numPairs; i++ {
		// Send init message
		initTrigger := randomInitTrigger(rng, eventLoggerAddress, 2, 10)
		initTx, initReceipt := alice.SendInitMessage(initTrigger)

		// Wait for a block
		sys.L2B.WaitForBlock()

		// Send exec message
		_, execReceipt := bob.SendExecMessage(initTx, 0)

		pairs = append(pairs, messagePair{
			initBlockNum: initReceipt.BlockNumber.Uint64(),
			execBlockNum: execReceipt.BlockNumber.Uint64(),
		})

		t.Logger().Info("message pair sent",
			"pair", i+1,
			"init_block", initReceipt.BlockNumber,
			"exec_block", execReceipt.BlockNumber,
		)
	}

	// Wait for all messages to become safe
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	timeout := time.Duration(blockTime*30+90) * time.Second

	lastPair := pairs[len(pairs)-1]
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number > lastPair.initBlockNum &&
			statusB.SafeL2.Number > lastPair.execBlockNum
	}, timeout, time.Second, "all message pairs should become safe")

	t.Logger().Info("all message pairs processed successfully", "num_pairs", numPairs)
}

// TestSupernodeInteropBidirectionalMessages tests sending messages in both directions
// (A->B and B->A) to verify the supernode handles bidirectional interop correctly.
func TestSupernodeInteropBidirectionalMessages(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

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
		return statusA.SafeL2.Number > initReceiptAtoB.BlockNumber.Uint64() &&
			statusA.SafeL2.Number > execReceiptBtoA.BlockNumber.Uint64() &&
			statusB.SafeL2.Number > execReceiptAtoB.BlockNumber.Uint64() &&
			statusB.SafeL2.Number > initReceiptBtoA.BlockNumber.Uint64()
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
