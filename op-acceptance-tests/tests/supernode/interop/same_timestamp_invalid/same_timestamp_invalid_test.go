package same_timestamp_invalid

/*
SPECIFICATION: Same-Timestamp Invalid Executing Message Test

Per the current strict timestamp validation rules:
- An executing message MUST reference an initiating message from a PRIOR timestamp
- Executing messages that reference initiating messages from the same timestamp are invalid

This test verifies that:
- When an executing message references an initiating message from the same timestamp,
  the interop activity detects this as invalid
- The chain containing the invalid executing message has its block replaced
- After replacement, the executing message transaction no longer exists in the chain

Test flow:
1. Pause interop to control the validation timing
2. Stop both sequencers to precisely control block timestamps
3. Calculate the target timestamp and pre-compute message identifiers
4. Include initiating message on Chain A at timestamp T
5. Include executing message on Chain B at timestamp T (same timestamp - invalid!)
6. Verify both transactions are at the same timestamp
7. Resume interop and wait for validation
8. Verify: Chain B's block was replaced, executing message no longer exists
*/

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TestSupernodeSameTimestampExecMessage tests that executing messages
// referencing initiating messages from the same timestamp are detected as invalid
// and the containing block is replaced.
//
// Scenario:
// - Chain A emits initiating message at timestamp T
// - Chain B executes that message at timestamp T (same timestamp - INVALID)
// - Interop detects the invalid executing message
// - Chain B's block is replaced with a deposits-only block
// - The executing message transaction no longer exists
func TestSupernodeSameTimestampExecMessage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	ctx := t.Ctx()

	// Create funded EOAs on both chains
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event logger on chain A (for initiating message)
	eventLoggerA := alice.DeployEventLogger()

	// Sync chains before pausing interop
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	// Pause interop and verify it has stopped cleanly
	// Uses max local safe timestamp from both chains, pauses at +10, awaits validation at +9
	pausedTimestamp := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "pausedTimestamp", pausedTimestamp)

	// Get rollup config for block time
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Stop both sequencers to control timestamps precisely
	t.Logger().Info("stopping both sequencers")
	sys.L2ACL.StopSequencer()
	sys.L2BCL.StopSequencer()

	// Get current state of both chains after pause
	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// Synchronize chains to the same timestamp if needed
	unsafeA, unsafeB = synchronizeChainsToSameTimestamp(t, sys, unsafeA, unsafeB)

	// Calculate expected block numbers for the NEXT blocks
	// Use the paused timestamp as our target (it's already unverified)
	nextTimestamp := unsafeA.Time + blockTime
	expectedBlockNumA := unsafeA.Number + 1

	t.Logger().Info("target timestamp for same-timestamp invalid test",
		"currentTimestamp", unsafeA.Time,
		"nextTimestamp", nextTimestamp,
		"pausedTimestamp", pausedTimestamp,
		"expectedBlockA", expectedBlockNumA,
		"chainA_currentBlock", unsafeA.Number,
		"chainB_currentBlock", unsafeB.Number,
	)

	// Prepare Alice's init message with deterministic content
	rng := rand.New(rand.NewSource(99999)) // Different seed for this test
	initTrigger := createDeterministicInitTrigger(rng, eventLoggerA, 2, 10)

	// Pre-compute the message identifier for Chain B's exec message
	// This is what the init message will look like once included at nextTimestamp
	precomputedMsg := precomputeInitMessage(
		eventLoggerA,        // Origin: the event logger address
		expectedBlockNumA,   // BlockNumber: next block on Chain A
		0,                   // LogIndex: first log in the block
		nextTimestamp,       // Timestamp: the SAME timestamp (this makes it invalid!)
		sys.L2ELA.ChainID(), // ChainID: Chain A's chain ID
		initTrigger,         // InitTrigger: to compute payload hash
	)

	t.Logger().Info("pre-computed init message (same timestamp - will be invalid)",
		"origin", precomputedMsg.Identifier.Origin,
		"blockNumber", precomputedMsg.Identifier.BlockNumber,
		"logIndex", precomputedMsg.Identifier.LogIndex,
		"timestamp", precomputedMsg.Identifier.Timestamp,
		"chainID", precomputedMsg.Identifier.ChainID,
		"payloadHash", precomputedMsg.PayloadHash,
	)

	// === STEP 1: Submit BOTH transactions to mempools BEFORE resuming sequencers ===
	// This is critical: we must submit both before resuming either sequencer,
	// so that both chains produce blocks at the same timestamp.
	// If we resumed sequencers sequentially, wall-clock time would advance between
	// inclusions and the timestamps would differ.

	// Submit Alice's init message to Chain A mempool
	initTx := submitInitMessage(ctx, t, alice, initTrigger)
	t.Logger().Info("submitted init message to Chain A mempool", "txHash", initTx.Signed.Value().Hash())

	// Submit Bob's exec message to Chain B mempool
	execTx := submitExecMessageWithPrecomputedMsg(ctx, t, bob, precomputedMsg)
	t.Logger().Info("submitted exec message to Chain B mempool", "txHash", execTx.Signed.Value().Hash())

	// === STEP 2: Resume BOTH sequencers simultaneously ===
	// Both chains will produce blocks at the same next timestamp since they're
	// currently synchronized and we're resuming them together.
	t.Logger().Info("resuming both sequencers simultaneously")
	sys.L2ACL.StartSequencer()
	sys.L2BCL.StartSequencer()

	// === STEP 3: Wait for BOTH transactions to be included ===
	t.Logger().Info("waiting for both transactions to be included")

	initReceipt, err := initTx.Included.Eval(ctx)
	require.NoError(t, err, "init tx should be included")

	execReceipt, err := execTx.Included.Eval(ctx)
	require.NoError(t, err, "exec tx should be included")

	// Get block info for both
	initBlock := sys.L2ELA.BlockRefByHash(initReceipt.BlockHash)
	execBlock := sys.L2ELB.BlockRefByHash(execReceipt.BlockHash)
	execBlockNumber := execBlock.Number
	execBlockHash := execBlock.Hash
	execBlockTimestamp := execBlock.Time
	execTxHash := execReceipt.TxHash

	t.Logger().Info("init message included on Chain A",
		"block", initBlock.Number,
		"timestamp", initBlock.Time,
		"hash", initBlock.Hash,
	)

	t.Logger().Info("exec message included on Chain B",
		"block", execBlockNumber,
		"timestamp", execBlockTimestamp,
		"hash", execBlockHash,
		"txHash", execTxHash,
	)

	// === CRITICAL ASSERTION: Both must be at the same timestamp ===
	require.Equal(t, initBlock.Time, execBlock.Time,
		"init and exec MUST be at the same timestamp for this test to be valid")

	t.Logger().Info("CONFIRMED: both messages are at the same timestamp",
		"timestamp", initBlock.Time,
		"initBlock", initBlock.Number,
		"execBlock", execBlock.Number,
	)

	// Record state before validation
	originalExecBlockHash := execBlockHash
	initBlockHash := initBlock.Hash

	// === STEP 4: Resume interop and observe replacement ===
	t.Logger().Info("resuming interop validation", "targetTimestamp", initBlock.Time)
	sys.Supernode.ResumeInterop()

	// Wait for validation to complete at this timestamp
	sys.Supernode.AwaitValidatedTimestamp(execBlockTimestamp)

	// === STEP 5: Verify the results ===
	// Get the current blocks at the same heights
	currentBlockA := sys.L2ELA.BlockRefByNumber(initBlock.Number)
	currentBlockB := sys.L2ELB.BlockRefByNumber(execBlockNumber)

	t.Logger().Info("blocks after validation",
		"initBlock_original", initBlockHash,
		"initBlock_current", currentBlockA.Hash,
		"initBlock_replaced", initBlockHash != currentBlockA.Hash,
		"execBlock_original", originalExecBlockHash,
		"execBlock_current", currentBlockB.Hash,
		"execBlock_replaced", originalExecBlockHash != currentBlockB.Hash,
	)

	// ASSERTION: Chain A's block should NOT be replaced (init message is valid)
	require.Equal(t, initBlockHash, currentBlockA.Hash,
		"Chain A's block should NOT be replaced - init message is valid")

	// ASSERTION: Chain B's block SHOULD be replaced (exec message is invalid due to same timestamp)
	require.NotEqual(t, originalExecBlockHash, currentBlockB.Hash,
		"Chain B's block SHOULD be replaced - exec message references same-timestamp init (invalid)")

	// ASSERTION: The invalid exec transaction no longer exists in the replacement block
	sys.L2ELB.AssertTxNotInBlock(execBlockNumber, execTxHash)

	t.Logger().Info("test complete: same-timestamp exec message was correctly invalidated and replaced",
		"timestamp", execBlockTimestamp,
		"initBlockHash", initBlockHash,
		"originalExecBlockHash", originalExecBlockHash,
		"replacementExecBlockHash", currentBlockB.Hash,
	)
}

// TestSupernodeSameTimestampInvalidTransitive tests transitive invalidation:
// when one chain's block is replaced due to an invalid message, other chains
// that depended on messages from that block are also replaced.
//
// Scenario at timestamp T:
// - Chain A: emits init(IA), executes IB (valid reference to B's init)
// - Chain B: emits init(IB), executes IA (INVALID - bad log index)
//
// Expected outcome over two rounds of verification:
// 1. Round 1: B is replaced because B's exec(IA) has an invalid log index
// 2. Round 2: A is replaced because B's init(IB) no longer exists after B was replaced
// (And actually both chains are replaced immediately because they use the same timestamp as the executing message, currently disabled)
//
// This demonstrates transitive/cascading invalidation through the interop system.
func TestSupernodeSameTimestampInvalidTransitive(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	ctx := t.Ctx()

	// Create funded EOAs on both chains
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event loggers on BOTH chains (for mutual messaging)
	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()

	// Sync chains before pausing interop
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)

	// Pause interop and verify it has stopped cleanly
	pausedTimestamp := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "pausedTimestamp", pausedTimestamp)

	// Get rollup config for block time
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Stop both sequencers to control timestamps precisely
	t.Logger().Info("stopping both sequencers")
	sys.L2ACL.StopSequencer()
	sys.L2BCL.StopSequencer()

	// Get current state of both chains after pause
	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// Synchronize chains to the same timestamp if needed
	unsafeA, unsafeB = synchronizeChainsToSameTimestamp(t, sys, unsafeA, unsafeB)

	// Calculate expected block numbers for the NEXT blocks
	nextTimestamp := unsafeA.Time + blockTime
	expectedBlockNumA := unsafeA.Number + 1
	expectedBlockNumB := unsafeB.Number + 1

	t.Logger().Info("target timestamp for transitive invalidation test",
		"currentTimestamp", unsafeA.Time,
		"nextTimestamp", nextTimestamp,
		"expectedBlockA", expectedBlockNumA,
		"expectedBlockB", expectedBlockNumB,
	)

	// Prepare init messages for BOTH chains with deterministic content
	rng := rand.New(rand.NewSource(77777)) // Different seed for this test
	initTriggerA := createDeterministicInitTrigger(rng, eventLoggerA, 2, 10)
	initTriggerB := createDeterministicInitTrigger(rng, eventLoggerB, 2, 10)

	// Pre-compute message identifiers for BOTH init messages
	// IA: Chain A's init message (will be referenced by B's INVALID exec)
	precomputedMsgIA := precomputeInitMessage(
		eventLoggerA,
		expectedBlockNumA,
		0, // LogIndex: first log in Chain A's block
		nextTimestamp,
		sys.L2ELA.ChainID(),
		initTriggerA,
	)

	// IB: Chain B's init message (will be referenced by A's VALID exec)
	precomputedMsgIB := precomputeInitMessage(
		eventLoggerB,
		expectedBlockNumB,
		0, // LogIndex: first log in Chain B's block
		nextTimestamp,
		sys.L2ELB.ChainID(),
		initTriggerB,
	)

	// Create INVALID version of IA for Chain B's exec (bad log index)
	// This makes B's exec(IA) invalid, while A's exec(IB) remains valid
	invalidMsgIA := precomputedMsgIA
	invalidMsgIA.Identifier.LogIndex = 9999 // Invalid log index - this log doesn't exist

	t.Logger().Info("pre-computed messages",
		"IA_origin", precomputedMsgIA.Identifier.Origin,
		"IA_blockNum", precomputedMsgIA.Identifier.BlockNumber,
		"IB_origin", precomputedMsgIB.Identifier.Origin,
		"IB_blockNum", precomputedMsgIB.Identifier.BlockNumber,
		"invalidIA_logIndex", invalidMsgIA.Identifier.LogIndex,
	)

	// === STEP 1: Submit all four transactions to mempools ===
	// Chain A: init(IA) + exec(IB) - both valid
	// Chain B: init(IB) + exec(IA) - init valid, exec INVALID

	// 1. Chain A: init(IA)
	initTxA := submitInitMessage(ctx, t, alice, initTriggerA)
	t.Logger().Info("submitted init(IA) to Chain A", "txHash", initTxA.Signed.Value().Hash())

	// 2. Chain A: exec(IB) - valid reference to B's init
	execTxA := submitExecMessageWithPrecomputedMsg(ctx, t, alice, precomputedMsgIB)
	t.Logger().Info("submitted exec(IB) to Chain A (VALID)", "txHash", execTxA.Signed.Value().Hash())

	// 3. Chain B: init(IB)
	initTxB := submitInitMessage(ctx, t, bob, initTriggerB)
	t.Logger().Info("submitted init(IB) to Chain B", "txHash", initTxB.Signed.Value().Hash())

	// 4. Chain B: exec(IA) - INVALID reference (bad log index)
	execTxB := submitExecMessageWithPrecomputedMsg(ctx, t, bob, invalidMsgIA)
	t.Logger().Info("submitted exec(IA) to Chain B (INVALID - bad log index)", "txHash", execTxB.Signed.Value().Hash())

	// === STEP 2: Resume BOTH sequencers simultaneously ===
	t.Logger().Info("resuming both sequencers simultaneously")
	sys.L2ACL.StartSequencer()
	sys.L2BCL.StartSequencer()

	// === STEP 3: Wait for all transactions to be included ===
	t.Logger().Info("waiting for all transactions to be included")

	initReceiptA, err := initTxA.Included.Eval(ctx)
	require.NoError(t, err, "init(IA) should be included")
	execReceiptA, err := execTxA.Included.Eval(ctx)
	require.NoError(t, err, "exec(IB) should be included")
	initReceiptB, err := initTxB.Included.Eval(ctx)
	require.NoError(t, err, "init(IB) should be included")
	execReceiptB, err := execTxB.Included.Eval(ctx)
	require.NoError(t, err, "invalid exec(IA) should be included")

	// Get the actual blocks
	blockA := sys.L2ELA.BlockRefByHash(initReceiptA.BlockHash)
	blockB := sys.L2ELB.BlockRefByHash(initReceiptB.BlockHash)

	t.Logger().Info("transaction inclusion results",
		"blockA_number", blockA.Number,
		"blockA_timestamp", blockA.Time,
		"blockA_hash", blockA.Hash,
		"blockB_number", blockB.Number,
		"blockB_timestamp", blockB.Time,
		"blockB_hash", blockB.Hash,
	)

	// Verify all txs are in the expected blocks at the same timestamp
	require.Equal(t, blockA.Time, blockB.Time,
		"both blocks must be at the same timestamp for this test to be valid")
	require.Equal(t, initReceiptA.BlockHash, execReceiptA.BlockHash,
		"Chain A's init and exec should be in the same block")
	require.Equal(t, initReceiptB.BlockHash, execReceiptB.BlockHash,
		"Chain B's init and exec should be in the same block")

	// Record original block hashes before validation
	originalBlockHashA := blockA.Hash
	originalBlockHashB := blockB.Hash
	blockNumberA := blockA.Number
	blockNumberB := blockB.Number
	targetTimestamp := blockA.Time

	t.Logger().Info("original blocks recorded",
		"blockA_hash", originalBlockHashA,
		"blockB_hash", originalBlockHashB,
		"timestamp", targetTimestamp,
	)

	// === STEP 4: Resume interop validation ===
	t.Logger().Info("resuming interop validation", "targetTimestamp", targetTimestamp)
	sys.Supernode.ResumeInterop()

	// Wait for validation to complete - this may take multiple iterations
	// as both blocks get replaced through transitive invalidation:
	// Round 1: B replaced (invalid exec)
	// Round 2: A replaced (B's init no longer exists)
	sys.Supernode.AwaitValidatedTimestamp(targetTimestamp)

	// === STEP 5: Verify the results ===
	// Get the current blocks at the same heights
	currentBlockA := sys.L2ELA.BlockRefByNumber(blockNumberA)
	currentBlockB := sys.L2ELB.BlockRefByNumber(blockNumberB)

	t.Logger().Info("blocks after validation",
		"originalA", originalBlockHashA,
		"currentA", currentBlockA.Hash,
		"replacedA", originalBlockHashA != currentBlockA.Hash,
		"originalB", originalBlockHashB,
		"currentB", currentBlockB.Hash,
		"replacedB", originalBlockHashB != currentBlockB.Hash,
	)

	// ASSERTION: BOTH chains should be replaced due to transitive invalidation
	// Chain B: replaced because exec(IA) has invalid log index
	// Chain A: replaced because exec(IB) references B's init which no longer exists
	chainAReplaced := originalBlockHashA != currentBlockA.Hash
	chainBReplaced := originalBlockHashB != currentBlockB.Hash

	require.True(t, chainBReplaced,
		"Chain B should be replaced - exec(IA) has invalid log index")
	require.True(t, chainAReplaced,
		"Chain A should be replaced - exec(IB) references B's init which no longer exists after B was replaced")

	// Verify the original transactions are no longer in the replacement blocks
	sys.L2ELA.AssertTxNotInBlock(blockNumberA, initReceiptA.TxHash)
	sys.L2ELA.AssertTxNotInBlock(blockNumberA, execReceiptA.TxHash)
	sys.L2ELB.AssertTxNotInBlock(blockNumberB, initReceiptB.TxHash)
	sys.L2ELB.AssertTxNotInBlock(blockNumberB, execReceiptB.TxHash)

	t.Logger().Info("test complete: transitive invalidation caused both chains to be replaced",
		"timestamp", targetTimestamp,
		"originalBlockA", originalBlockHashA,
		"replacementBlockA", currentBlockA.Hash,
		"originalBlockB", originalBlockHashB,
		"replacementBlockB", currentBlockB.Hash,
	)
}

// --- Helper Functions ---

// createDeterministicInitTrigger creates an init trigger with deterministic content
func createDeterministicInitTrigger(rng *rand.Rand, eventLogger common.Address, topicCount, dataLen int) *txintent.InitTrigger {
	if topicCount > 4 {
		topicCount = 4
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
		Emitter:    eventLogger,
		Topics:     topics,
		OpaqueData: testutils.RandomData(rng, dataLen),
	}
}

// precomputeInitMessage creates a Message with pre-computed identifier and payload hash
// This allows creating an exec message before the init message is actually included
func precomputeInitMessage(
	origin common.Address,
	blockNumber uint64,
	logIndex uint32,
	timestamp uint64,
	chainID eth.ChainID,
	trigger *txintent.InitTrigger,
) suptypes.Message {
	// Compute the payload as it will appear in the log
	// The EventLogger emits: emit Log(topics, data)
	// The log topics are the trigger's topics, and data is the opaque data
	payload := computeLogPayload(trigger.Topics, trigger.OpaqueData)
	payloadHash := crypto.Keccak256Hash(payload)

	return suptypes.Message{
		Identifier: suptypes.Identifier{
			Origin:      origin,
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   timestamp,
			ChainID:     chainID,
		},
		PayloadHash: payloadHash,
	}
}

// computeLogPayload computes the payload as LogToMessagePayload would
// For EventLogger, the log topics are the input topics and data is the opaque data
func computeLogPayload(topics [][32]byte, data []byte) []byte {
	payload := make([]byte, 0)
	for _, topic := range topics {
		payload = append(payload, topic[:]...)
	}
	payload = append(payload, data...)
	return payload
}

// submitInitMessage submits an init message to the mempool without waiting for inclusion
func submitInitMessage(ctx context.Context, t devtest.T, alice *dsl.EOA, trigger *txintent.InitTrigger) *txplan.PlannedTx {
	tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](alice.Plan())
	tx.Content.Set(trigger)

	// Submit to mempool (don't wait for inclusion)
	_, err := tx.PlannedTx.Submitted.Eval(ctx)
	require.NoError(t, err, "failed to submit init tx to mempool")

	return tx.PlannedTx
}

// submitExecMessageWithPrecomputedMsg submits an exec message with a pre-computed identifier
func submitExecMessageWithPrecomputedMsg(ctx context.Context, t devtest.T, bob *dsl.EOA, msg suptypes.Message) *txplan.PlannedTx {
	execTrigger := &txintent.ExecTrigger{
		Executor: constants.CrossL2Inbox,
		Msg:      msg,
	}

	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
	tx.Content.Set(execTrigger)

	// Submit to mempool (don't wait for inclusion)
	_, err := tx.PlannedTx.Submitted.Eval(ctx)
	require.NoError(t, err, "failed to submit exec tx to mempool")

	return tx.PlannedTx
}

// synchronizeChainsToSameTimestamp ensures both chains are at the exact same timestamp.
// If they differ, advances the chain that's behind until they match.
// Returns the updated block refs for both chains.
func synchronizeChainsToSameTimestamp(t devtest.T, sys *presets.TwoL2SupernodeInterop, unsafeA, unsafeB eth.L2BlockRef) (eth.L2BlockRef, eth.L2BlockRef) {
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		t.Logger().Info("synchronization check",
			"iteration", i,
			"chainA_time", unsafeA.Time,
			"chainA_block", unsafeA.Number,
			"chainB_time", unsafeB.Time,
			"chainB_block", unsafeB.Number,
		)

		if unsafeA.Time == unsafeB.Time {
			t.Logger().Info("chains synchronized at same timestamp",
				"timestamp", unsafeA.Time,
				"chainA_block", unsafeA.Number,
				"chainB_block", unsafeB.Number,
			)
			return unsafeA, unsafeB
		}

		// Advance the chain that's behind
		if unsafeA.Time < unsafeB.Time {
			t.Logger().Info("advancing chain A to catch up", "from", unsafeA.Time, "to", unsafeB.Time)
			sys.L2ACL.StartSequencer()
			sys.L2ELA.WaitForTime(unsafeB.Time)
			sys.L2ACL.StopSequencer()
			unsafeA = sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		} else {
			t.Logger().Info("advancing chain B to catch up", "from", unsafeB.Time, "to", unsafeA.Time)
			sys.L2BCL.StartSequencer()
			sys.L2ELB.WaitForTime(unsafeA.Time)
			sys.L2BCL.StopSequencer()
			unsafeB = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		}
	}

	// If we get here, synchronization failed
	require.Equal(t, unsafeA.Time, unsafeB.Time,
		"failed to synchronize chains after %d iterations: chainA=%d, chainB=%d",
		maxIterations, unsafeA.Time, unsafeB.Time)
	return unsafeA, unsafeB
}
