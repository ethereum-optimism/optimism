package same_timestamp_invalid

/*
SPECIFICATION: Same-Timestamp Interop Tests

Same-timestamp interop allows executing messages to reference initiating messages
from the SAME timestamp. This enables cross-chain communication within the same
block time across the interop set.

Validation rules:
- An executing message MAY reference an initiating message from the same timestamp (VALID)
- An executing message MAY reference an initiating message from a prior timestamp (VALID)
- An executing message MUST NOT reference an initiating message from a future timestamp (INVALID)
- The initiating message must exist and have a valid checksum (INVALID if not found)
- Same-timestamp messages MUST NOT form circular dependencies (INVALID if cycle detected)

Test 1 (TestSupernodeSameTimestampExecMessage) verifies:
- When an executing message references an initiating message from the same timestamp,
  the interop activity accepts this as VALID
- Neither chain's block is replaced
- Both transactions remain in their respective blocks

Test 2 (TestSupernodeSameTimestampInvalidTransitive) verifies:
- Invalid messages (e.g., bad log index) still cause block replacement
- Transitive invalidation works: if Chain B is replaced, Chain A's exec referencing
  B's (now-gone) init is also invalidated and replaced
- Note: The invalidation is due to bad log index, NOT same-timestamp

Test 3 (TestSupernodeSameTimestampCycle) verifies:
- When two chains have mutual same-timestamp exec messages (A executes B, B executes A),
  this creates a circular dependency that is detected and causes both blocks to be replaced
- This validates the cycle detection algorithm (Kahn's topological sort)

Test flow:
1. Pause interop to control the validation timing
2. Stop both sequencers to precisely control block timestamps
3. Calculate the target timestamp and pre-compute message identifiers
4. Include messages at timestamp T
5. Verify all transactions are at the same timestamp
6. Resume interop and wait for validation
7. Verify expected outcomes (valid/replaced based on test scenario)
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
// referencing initiating messages from the same timestamp are accepted as VALID.
//
// Scenario:
// - Chain A emits initiating message at timestamp T
// - Chain B executes that message at timestamp T (same timestamp - VALID!)
// - Interop validates both messages successfully
// - Neither chain's block is replaced
// - Both transactions remain in their respective blocks
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

	t.Logger().Info("pre-computed init message (same timestamp - now valid)",
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

	// === STEP 4: Resume interop and validate ===
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

	// ASSERTION: Chain B's block should NOT be replaced (same-timestamp exec is now VALID)
	// Since the block hash is unchanged, the exec transaction is guaranteed to still exist.
	require.Equal(t, originalExecBlockHash, currentBlockB.Hash,
		"Chain B's block should NOT be replaced - same-timestamp exec message is valid")

	t.Logger().Info("test complete: same-timestamp exec message was correctly validated and accepted",
		"timestamp", execBlockTimestamp,
		"initBlockHash", initBlockHash,
		"execBlockHash", currentBlockB.Hash,
	)
}

// TestSupernodeSameTimestampInvalidTransitive tests transitive invalidation:
// when one chain's block is replaced due to an invalid message, other chains
// that depended on messages from that block are also replaced.
//
// NOTE: This test uses same-timestamp messages, but the invalidation is NOT caused
// by same-timestamp (which is now VALID). The invalidation is caused by an INVALID
// LOG INDEX (9999) in Chain B's exec message.
//
// Scenario at timestamp T:
// - Chain A: emits init(IA), executes IB (valid reference to B's init, same-timestamp OK)
// - Chain B: emits init(IB), executes IA (INVALID - bad log index 9999, NOT same-timestamp)
//
// Expected outcome over two rounds of verification:
// 1. Round 1: B is replaced because B's exec(IA) has an invalid log index (9999)
// 2. Round 2: A is replaced because B's init(IB) no longer exists after B was replaced
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

// TestSupernodeSameTimestampCycle tests that mutual same-timestamp executing messages
// are detected as a circular dependency and cause both blocks to be replaced.
//
// This validates the cycle detection algorithm (Kahn's topological sort) in
// the supernode's interop verification.
//
// Scenario at timestamp T:
// - Chain A: emits init(IA) at log 0, executes IB at log 1 (references B's init at log 0)
// - Chain B: emits init(IB) at log 0, executes IA at log 1 (references A's init at log 0)
//
// Dependency graph:
// - A's exec(IB) depends on B's init(IB) which depends on B's exec(IA)
// - B's exec(IA) depends on A's init(IA) which depends on A's exec(IB)
// - This creates a cycle: A:1 → B:0 → B:1 → A:0 → A:1
//
// Expected outcome:
// - Cycle detection identifies the circular dependency
// - Both chains' blocks are marked invalid and replaced
func TestSupernodeSameTimestampCycle(gt *testing.T) {
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

	t.Logger().Info("target timestamp for cycle detection test",
		"currentTimestamp", unsafeA.Time,
		"nextTimestamp", nextTimestamp,
		"expectedBlockA", expectedBlockNumA,
		"expectedBlockB", expectedBlockNumB,
	)

	// Prepare init messages for BOTH chains with deterministic content
	rng := rand.New(rand.NewSource(55555)) // Unique seed for this test
	initTriggerA := createDeterministicInitTrigger(rng, eventLoggerA, 2, 10)
	initTriggerB := createDeterministicInitTrigger(rng, eventLoggerB, 2, 10)

	// Pre-compute message identifiers for BOTH init messages
	// IA: Chain A's init message at log index 0
	precomputedMsgIA := precomputeInitMessage(
		eventLoggerA,
		expectedBlockNumA,
		0, // LogIndex: first log in Chain A's block (init)
		nextTimestamp,
		sys.L2ELA.ChainID(),
		initTriggerA,
	)

	// IB: Chain B's init message at log index 0
	precomputedMsgIB := precomputeInitMessage(
		eventLoggerB,
		expectedBlockNumB,
		0, // LogIndex: first log in Chain B's block (init)
		nextTimestamp,
		sys.L2ELB.ChainID(),
		initTriggerB,
	)

	t.Logger().Info("pre-computed messages for cycle test",
		"IA_origin", precomputedMsgIA.Identifier.Origin,
		"IA_blockNum", precomputedMsgIA.Identifier.BlockNumber,
		"IA_logIdx", precomputedMsgIA.Identifier.LogIndex,
		"IB_origin", precomputedMsgIB.Identifier.Origin,
		"IB_blockNum", precomputedMsgIB.Identifier.BlockNumber,
		"IB_logIdx", precomputedMsgIB.Identifier.LogIndex,
	)

	// === STEP 1: Submit all four transactions to mempools ===
	// Chain A: init(IA) at log 0 + exec(IB) at log 1 - VALID references, but creates cycle
	// Chain B: init(IB) at log 0 + exec(IA) at log 1 - VALID references, but creates cycle

	// 1. Chain A: init(IA) - will be at log index 0
	initTxA := submitInitMessage(ctx, t, alice, initTriggerA)
	t.Logger().Info("submitted init(IA) to Chain A", "txHash", initTxA.Signed.Value().Hash())

	// 2. Chain A: exec(IB) - will be at log index 1, references B's init at log 0
	execTxA := submitExecMessageWithPrecomputedMsg(ctx, t, alice, precomputedMsgIB)
	t.Logger().Info("submitted exec(IB) to Chain A", "txHash", execTxA.Signed.Value().Hash())

	// 3. Chain B: init(IB) - will be at log index 0
	initTxB := submitInitMessage(ctx, t, bob, initTriggerB)
	t.Logger().Info("submitted init(IB) to Chain B", "txHash", initTxB.Signed.Value().Hash())

	// 4. Chain B: exec(IA) - will be at log index 1, references A's init at log 0
	execTxB := submitExecMessageWithPrecomputedMsg(ctx, t, bob, precomputedMsgIA)
	t.Logger().Info("submitted exec(IA) to Chain B", "txHash", execTxB.Signed.Value().Hash())

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
	require.NoError(t, err, "exec(IA) should be included")

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

	t.Logger().Info("original blocks recorded - expecting cycle detection to invalidate both",
		"blockA_hash", originalBlockHashA,
		"blockB_hash", originalBlockHashB,
		"timestamp", targetTimestamp,
	)

	// === STEP 4: Resume interop validation ===
	t.Logger().Info("resuming interop validation", "targetTimestamp", targetTimestamp)
	sys.Supernode.ResumeInterop()

	// Wait for validation to complete
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

	// ASSERTION: BOTH chains should be replaced due to cycle detection
	// The mutual same-timestamp exec messages create a circular dependency
	chainAReplaced := originalBlockHashA != currentBlockA.Hash
	chainBReplaced := originalBlockHashB != currentBlockB.Hash

	require.True(t, chainBReplaced,
		"Chain B should be replaced - cycle detected in same-timestamp messages")
	require.True(t, chainAReplaced,
		"Chain A should be replaced - cycle detected in same-timestamp messages")

	// Verify the original transactions are no longer in the replacement blocks
	sys.L2ELA.AssertTxNotInBlock(blockNumberA, initReceiptA.TxHash)
	sys.L2ELA.AssertTxNotInBlock(blockNumberA, execReceiptA.TxHash)
	sys.L2ELB.AssertTxNotInBlock(blockNumberB, initReceiptB.TxHash)
	sys.L2ELB.AssertTxNotInBlock(blockNumberB, execReceiptB.TxHash)

	t.Logger().Info("test complete: cycle detection caused both chains to be replaced",
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
