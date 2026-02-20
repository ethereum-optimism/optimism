package same_timestamp_invalid

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

// =============================================================================
// Test Harness
// =============================================================================

type sameTimestampHarness struct {
	t   devtest.T
	sys *presets.TwoL2SupernodeInterop
	ctx context.Context

	alice *dsl.EOA
	bob   *dsl.EOA

	eventLoggerA common.Address
	eventLoggerB common.Address

	nextTimestamp     uint64
	expectedBlockNumA uint64
	expectedBlockNumB uint64
}

func newSameTimestampHarness(gt *testing.T) *sameTimestampHarness {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	ctx := t.Ctx()

	// Create funded EOAs
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event loggers
	eventLoggerA := alice.DeployEventLogger()
	eventLoggerB := bob.DeployEventLogger()

	// Sync chains and pause interop
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)

	// Stop sequencers
	sys.L2ACL.StopSequencer()
	sys.L2BCL.StopSequencer()

	// Get current state and synchronize timestamps
	unsafeA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	unsafeB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	unsafeA, unsafeB = synchronizeChainsToSameTimestamp(t, sys, unsafeA, unsafeB)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	return &sameTimestampHarness{
		t:                 t,
		sys:               sys,
		ctx:               ctx,
		alice:             alice,
		bob:               bob,
		eventLoggerA:      eventLoggerA.Address(),
		eventLoggerB:      eventLoggerB.Address(),
		nextTimestamp:     unsafeA.Time + blockTime,
		expectedBlockNumA: unsafeA.Number + 1,
		expectedBlockNumB: unsafeB.Number + 1,
	}
}

// precomputeInit creates a precomputed init message identifier
func (h *sameTimestampHarness) precomputeInit(rng *rand.Rand, origin common.Address, blockNum uint64, logIdx uint32, chainID eth.ChainID) (suptypes.Message, *txintent.InitTrigger) {
	trigger := createDeterministicInitTrigger(rng, origin, 2, 10)
	msg := precomputeInitMessage(origin, blockNum, logIdx, h.nextTimestamp, chainID, trigger)
	return msg, trigger
}

// submitAndInclude submits all transactions then resumes sequencers and waits for inclusion
func (h *sameTimestampHarness) submitAndInclude(txsA, txsB []*txplan.PlannedTx) (blockA, blockB eth.L2BlockRef) {
	// Resume sequencers
	h.sys.L2ACL.StartSequencer()
	h.sys.L2BCL.StartSequencer()

	// Wait for all transactions
	for _, tx := range txsA {
		receipt, err := tx.Included.Eval(h.ctx)
		require.NoError(h.t, err)
		blockA = h.sys.L2ELA.BlockRefByHash(receipt.BlockHash)
	}
	for _, tx := range txsB {
		receipt, err := tx.Included.Eval(h.ctx)
		require.NoError(h.t, err)
		blockB = h.sys.L2ELB.BlockRefByHash(receipt.BlockHash)
	}

	require.Equal(h.t, blockA.Time, blockB.Time, "blocks must be at same timestamp")
	return blockA, blockB
}

// validateAndCheck resumes interop, waits for validation, and checks replacement
func (h *sameTimestampHarness) validateAndCheck(blockA, blockB eth.L2BlockRef, expectReplacedA, expectReplacedB bool) {
	h.sys.Supernode.ResumeInterop()
	h.sys.Supernode.AwaitValidatedTimestamp(blockA.Time)

	currentA := h.sys.L2ELA.BlockRefByNumber(blockA.Number)
	currentB := h.sys.L2ELB.BlockRefByNumber(blockB.Number)

	if expectReplacedA {
		require.NotEqual(h.t, blockA.Hash, currentA.Hash, "Chain A should be replaced")
	} else {
		require.Equal(h.t, blockA.Hash, currentA.Hash, "Chain A should NOT be replaced")
	}

	if expectReplacedB {
		require.NotEqual(h.t, blockB.Hash, currentB.Hash, "Chain B should be replaced")
	} else {
		require.Equal(h.t, blockB.Hash, currentB.Hash, "Chain B should NOT be replaced")
	}
}

// =============================================================================
// Tests
// =============================================================================

// TestSupernodeSameTimestampExecMessage: Chain B executes Chain A's init at same timestamp - VALID
func TestSupernodeSameTimestampExecMessage(gt *testing.T) {
	h := newSameTimestampHarness(gt)
	rng := rand.New(rand.NewSource(99999))

	// Chain A: init message
	msgIA, triggerA := h.precomputeInit(rng, h.eventLoggerA, h.expectedBlockNumA, 0, h.sys.L2ELA.ChainID())

	// Submit: A=init, B=exec(A's init)
	initTxA := submitInitMessage(h.ctx, h.t, h.alice, triggerA)
	execTxB := submitExecMessage(h.ctx, h.t, h.bob, msgIA)

	blockA, blockB := h.submitAndInclude([]*txplan.PlannedTx{initTxA}, []*txplan.PlannedTx{execTxB})
	h.validateAndCheck(blockA, blockB, false, false) // Neither replaced
}

// TestSupernodeSameTimestampInvalidTransitive: Bad log index causes transitive invalidation
func TestSupernodeSameTimestampInvalidTransitive(gt *testing.T) {
	h := newSameTimestampHarness(gt)
	rng := rand.New(rand.NewSource(77777))

	// Chain A: init + exec(B's init)
	msgIA, triggerA := h.precomputeInit(rng, h.eventLoggerA, h.expectedBlockNumA, 0, h.sys.L2ELA.ChainID())
	msgIB, triggerB := h.precomputeInit(rng, h.eventLoggerB, h.expectedBlockNumB, 0, h.sys.L2ELB.ChainID())

	// Make B's exec INVALID - bad log index
	invalidMsgIA := msgIA
	invalidMsgIA.Identifier.LogIndex = 9999

	// Submit: A=init+exec(B), B=init+exec(A with bad logIdx)
	initTxA := submitInitMessage(h.ctx, h.t, h.alice, triggerA)
	execTxA := submitExecMessage(h.ctx, h.t, h.alice, msgIB)
	initTxB := submitInitMessage(h.ctx, h.t, h.bob, triggerB)
	execTxB := submitExecMessage(h.ctx, h.t, h.bob, invalidMsgIA)

	blockA, blockB := h.submitAndInclude(
		[]*txplan.PlannedTx{initTxA, execTxA},
		[]*txplan.PlannedTx{initTxB, execTxB},
	)
	h.validateAndCheck(blockA, blockB, true, true) // Both replaced (transitive)
}

// TestSupernodeSameTimestampCycle: Mutual exec messages create cycle - both replaced
func TestSupernodeSameTimestampCycle(gt *testing.T) {
	h := newSameTimestampHarness(gt)
	rng := rand.New(rand.NewSource(55555))

	// Chain A: init at log 0, exec(B) at log 1
	// Chain B: init at log 0, exec(A) at log 1
	msgIA, triggerA := h.precomputeInit(rng, h.eventLoggerA, h.expectedBlockNumA, 0, h.sys.L2ELA.ChainID())
	msgIB, triggerB := h.precomputeInit(rng, h.eventLoggerB, h.expectedBlockNumB, 0, h.sys.L2ELB.ChainID())

	// Submit: A=init+exec(B), B=init+exec(A) - creates cycle
	initTxA := submitInitMessage(h.ctx, h.t, h.alice, triggerA)
	execTxA := submitExecMessage(h.ctx, h.t, h.alice, msgIB)
	initTxB := submitInitMessage(h.ctx, h.t, h.bob, triggerB)
	execTxB := submitExecMessage(h.ctx, h.t, h.bob, msgIA)

	blockA, blockB := h.submitAndInclude(
		[]*txplan.PlannedTx{initTxA, execTxA},
		[]*txplan.PlannedTx{initTxB, execTxB},
	)
	h.validateAndCheck(blockA, blockB, true, true) // Both replaced (cycle)
}

// =============================================================================
// Helpers
// =============================================================================

func createDeterministicInitTrigger(rng *rand.Rand, eventLogger common.Address, topicCount, dataLen int) *txintent.InitTrigger {
	if topicCount > 4 {
		topicCount = 4
	}
	if topicCount < 1 {
		topicCount = 1
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

func precomputeInitMessage(origin common.Address, blockNumber uint64, logIndex uint32, timestamp uint64, chainID eth.ChainID, trigger *txintent.InitTrigger) suptypes.Message {
	payload := make([]byte, 0)
	for _, topic := range trigger.Topics {
		payload = append(payload, topic[:]...)
	}
	payload = append(payload, trigger.OpaqueData...)
	return suptypes.Message{
		Identifier: suptypes.Identifier{
			Origin:      origin,
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   timestamp,
			ChainID:     chainID,
		},
		PayloadHash: crypto.Keccak256Hash(payload),
	}
}

func submitInitMessage(ctx context.Context, t devtest.T, eoa *dsl.EOA, trigger *txintent.InitTrigger) *txplan.PlannedTx {
	tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](eoa.Plan())
	tx.Content.Set(trigger)
	_, err := tx.PlannedTx.Submitted.Eval(ctx)
	require.NoError(t, err)
	return tx.PlannedTx
}

func submitExecMessage(ctx context.Context, t devtest.T, eoa *dsl.EOA, msg suptypes.Message) *txplan.PlannedTx {
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](eoa.Plan())
	tx.Content.Set(&txintent.ExecTrigger{Executor: constants.CrossL2Inbox, Msg: msg})
	_, err := tx.PlannedTx.Submitted.Eval(ctx)
	require.NoError(t, err)
	return tx.PlannedTx
}

func synchronizeChainsToSameTimestamp(t devtest.T, sys *presets.TwoL2SupernodeInterop, unsafeA, unsafeB eth.L2BlockRef) (eth.L2BlockRef, eth.L2BlockRef) {
	for i := 0; i < 10; i++ {
		if unsafeA.Time == unsafeB.Time {
			return unsafeA, unsafeB
		}
		if unsafeA.Time < unsafeB.Time {
			sys.L2ACL.StartSequencer()
			sys.L2ELA.WaitForTime(unsafeB.Time)
			sys.L2ACL.StopSequencer()
			unsafeA = sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		} else {
			sys.L2BCL.StartSequencer()
			sys.L2ELB.WaitForTime(unsafeA.Time)
			sys.L2BCL.StopSequencer()
			unsafeB = sys.L2ELB.BlockRefByLabel(eth.Unsafe)
		}
	}
	require.Equal(t, unsafeA.Time, unsafeB.Time, "failed to synchronize chains")
	return unsafeA, unsafeB
}
