package managed

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// mockEventStream implements ManagedEventStream for testing
type mockEventStream struct {
	events []*supervisortypes.ManagedEvent
}

func (m *mockEventStream) Send(event *supervisortypes.ManagedEvent) {
	m.events = append(m.events, event)
}

func (m *mockEventStream) Serve() (*supervisortypes.ManagedEvent, error) {
	panic("not implemented")
}

func (m *mockEventStream) Subscribe(ctx context.Context) (*gethrpc.Subscription, error) {
	panic("not implemented")
}

func (m *mockEventStream) drainEvents() []*supervisortypes.ManagedEvent {
	events := m.events
	m.events = nil
	return events
}

func TestManagedMode_OnEvent_Deduplication(t *testing.T) {
	logger, logs := testlog.CaptureLogger(t, log.LevelDebug)
	cfg := &rollup.Config{
		L2ChainID:   big.NewInt(123),
		InteropTime: new(uint64), // Interop active from genesis
	}

	mockStream := &mockEventStream{}

	// Create ManagedMode with only the necessary fields for testing
	mm := &ManagedMode{
		log:    logger,
		cfg:    cfg,
		events: mockStream,
		// Initialize event timestamp trackers with short TTLs for testing
		lastReset:         newEventTimestamp[struct{}](50 * time.Millisecond),
		lastUnsafe:        newEventTimestamp[eth.BlockID](50 * time.Millisecond),
		lastSafe:          newEventTimestamp[eth.BlockID](50 * time.Millisecond),
		lastL1Traversal:   newEventTimestamp[eth.BlockID](50 * time.Millisecond),
		lastExhaustedL1:   newEventTimestamp[eth.BlockID](50 * time.Millisecond),
		lastReplacedBlock: newEventTimestamp[eth.BlockID](50 * time.Millisecond),
	}

	// Common test data used across multiple sub-tests
	l1Ref1 := eth.L1BlockRef{Hash: common.Hash{1}, Number: 50}
	l1Ref2 := eth.L1BlockRef{Hash: common.Hash{2}, Number: 51}

	l2Ref1 := eth.L2BlockRef{Hash: common.Hash{1}, Number: 100, Time: 1000}
	l2Ref2 := eth.L2BlockRef{Hash: common.Hash{2}, Number: 101, Time: 1000}

	t.Run("ResetEvent", func(t *testing.T) {
		logs.Clear()

		// First reset event should be sent
		resetErr := errors.New("test reset error")
		resetEvent := rollup.ResetEvent{Err: resetErr}

		result := mm.OnEvent(resetEvent)
		require.True(t, result, "ResetEvent should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First reset event should be sent")
		require.NotNil(t, events[0].Reset)
		require.Equal(t, resetErr.Error(), *events[0].Reset)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate reset request")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Immediate duplicate should be skipped (within TTL)
		result = mm.OnEvent(resetEvent)
		require.True(t, result, "ResetEvent should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate reset event should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Wait for TTL to expire and try again
		time.Sleep(60 * time.Millisecond) // TTL is 50ms
		logs.Clear()

		result = mm.OnEvent(resetEvent)
		require.True(t, result, "ResetEvent should be handled after TTL")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "Reset event after TTL should be sent")
		require.NotNil(t, events[0].Reset)

		// Check that no skipping log was generated after TTL
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log after TTL")
	})

	t.Run("UnsafeUpdateEvent", func(t *testing.T) {
		logs.Clear()

		// First unsafe update should be sent
		unsafeEvent1 := engine.UnsafeUpdateEvent{Ref: l2Ref1}
		result := mm.OnEvent(unsafeEvent1)
		require.True(t, result, "UnsafeUpdateEvent should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First unsafe update should be sent")
		require.NotNil(t, events[0].UnsafeBlock)
		require.Equal(t, l2Ref1.BlockRef(), *events[0].UnsafeBlock)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate local unsafe update event")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Duplicate should be skipped
		result = mm.OnEvent(unsafeEvent1)
		require.True(t, result, "Duplicate UnsafeUpdateEvent should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate unsafe update should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Different reference should be sent
		logs.Clear()
		unsafeEvent2 := engine.UnsafeUpdateEvent{Ref: l2Ref2}
		result = mm.OnEvent(unsafeEvent2)
		require.True(t, result, "UnsafeUpdateEvent with different ref should be handled")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "Unsafe update with different ref should be sent")
		require.NotNil(t, events[0].UnsafeBlock)
		require.Equal(t, l2Ref2.BlockRef(), *events[0].UnsafeBlock)

		// Check that no skipping log was generated for different ref
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for different ref")
	})

	t.Run("LocalSafeUpdateEvent", func(t *testing.T) {
		logs.Clear()

		// First safe update should be sent
		safeEvent1 := engine.LocalSafeUpdateEvent{Source: l1Ref1, Ref: l2Ref1}
		result := mm.OnEvent(safeEvent1)
		require.True(t, result, "LocalSafeUpdateEvent should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First safe update should be sent")
		require.NotNil(t, events[0].DerivationUpdate)
		require.Equal(t, l1Ref1, events[0].DerivationUpdate.Source)
		require.Equal(t, l2Ref1.BlockRef(), events[0].DerivationUpdate.Derived)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate derivation update (new local safe)")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Duplicate should be skipped
		result = mm.OnEvent(safeEvent1)
		require.True(t, result, "Duplicate LocalSafeUpdateEvent should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate safe update should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Different reference should be sent
		logs.Clear()
		safeEvent2 := engine.LocalSafeUpdateEvent{Source: l1Ref1, Ref: l2Ref2}
		result = mm.OnEvent(safeEvent2)
		require.True(t, result, "LocalSafeUpdateEvent with different ref should be handled")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "Safe update with different ref should be sent")
		require.NotNil(t, events[0].DerivationUpdate)
		require.Equal(t, l2Ref2.BlockRef(), events[0].DerivationUpdate.Derived)

		// Check that no skipping log was generated for different ref
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for different ref")
	})

	t.Run("DeriverL1StatusEvent", func(t *testing.T) {
		logs.Clear()

		// First L1 status event should be sent
		l1StatusEvent1 := derive.DeriverL1StatusEvent{Origin: l1Ref1, LastL2: l2Ref1}
		result := mm.OnEvent(l1StatusEvent1)
		require.True(t, result, "DeriverL1StatusEvent should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First L1 status event should be sent")
		require.NotNil(t, events[0].DerivationUpdate)
		require.NotNil(t, events[0].DerivationOriginUpdate)
		require.Equal(t, l1Ref1, events[0].DerivationUpdate.Source)
		require.Equal(t, l1Ref1, *events[0].DerivationOriginUpdate)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate derivation update (L1 traversal)")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Duplicate should be skipped
		result = mm.OnEvent(l1StatusEvent1)
		require.True(t, result, "Duplicate DeriverL1StatusEvent should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate L1 status event should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Different origin should be sent
		logs.Clear()
		l1StatusEvent2 := derive.DeriverL1StatusEvent{Origin: l1Ref2, LastL2: l2Ref1}
		result = mm.OnEvent(l1StatusEvent2)
		require.True(t, result, "DeriverL1StatusEvent with different origin should be handled")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "L1 status event with different origin should be sent")
		require.NotNil(t, events[0].DerivationUpdate)
		require.Equal(t, l1Ref2, events[0].DerivationUpdate.Source)

		// Check that no skipping log was generated for different origin
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for different origin")
	})

	t.Run("ExhaustedL1Event", func(t *testing.T) {
		logs.Clear()

		// First exhausted L1 event should be sent
		exhaustedEvent1 := derive.ExhaustedL1Event{L1Ref: l1Ref1, LastL2: l2Ref1}
		result := mm.OnEvent(exhaustedEvent1)
		require.True(t, result, "ExhaustedL1Event should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First exhausted L1 event should be sent")
		require.NotNil(t, events[0].ExhaustL1)
		require.Equal(t, l1Ref1, events[0].ExhaustL1.Source)
		require.Equal(t, l2Ref1.BlockRef(), events[0].ExhaustL1.Derived)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate exhausted L1 event")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Duplicate should be skipped
		result = mm.OnEvent(exhaustedEvent1)
		require.True(t, result, "Duplicate ExhaustedL1Event should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate exhausted L1 event should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Different L1 ref should be sent
		logs.Clear()
		exhaustedEvent2 := derive.ExhaustedL1Event{L1Ref: l1Ref2, LastL2: l2Ref1}
		result = mm.OnEvent(exhaustedEvent2)
		require.True(t, result, "ExhaustedL1Event with different L1Ref should be handled")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "Exhausted L1 event with different L1Ref should be sent")
		require.NotNil(t, events[0].ExhaustL1)
		require.Equal(t, l1Ref2, events[0].ExhaustL1.Source)
		require.Equal(t, l2Ref1.BlockRef(), events[0].ExhaustL1.Derived)

		// Check that no skipping log was generated for different L1 ref
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for different L1 ref")
	})

	t.Run("InteropReplacedBlockEvent", func(t *testing.T) {
		logs.Clear()

		// Create a valid replacement block with system tx
		outputRoot := &eth.OutputV0{
			StateRoot:                eth.Bytes32{1},
			MessagePasserStorageRoot: eth.Bytes32{2},
			BlockHash:                common.Hash{3},
		}
		systemTx := InvalidatedBlockSourceDepositTx(outputRoot.Marshal())
		encodedTx, err := systemTx.MarshalBinary()
		require.NoError(t, err)

		envelope := &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				Transactions: []eth.Data{encodedTx},
			},
		}

		// First replaced block event should be sent
		replacedEvent1 := engine.InteropReplacedBlockEvent{Ref: l2Ref1.BlockRef(), Envelope: envelope}
		result := mm.OnEvent(replacedEvent1)
		require.True(t, result, "InteropReplacedBlockEvent should be handled")

		events := mockStream.drainEvents()
		require.Len(t, events, 1, "First replaced block event should be sent")
		require.NotNil(t, events[0].ReplaceBlock)
		require.Equal(t, l2Ref1.BlockRef(), events[0].ReplaceBlock.Replacement)
		require.Equal(t, outputRoot.BlockHash, events[0].ReplaceBlock.Invalidated)

		// Check that no skipping log was generated
		hasSkipLog := testlog.NewMessageContainsFilter("Skipped sending duplicate replaced block event")
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for first event")

		// Duplicate should be skipped
		result = mm.OnEvent(replacedEvent1)
		require.True(t, result, "Duplicate InteropReplacedBlockEvent should be handled but skipped")

		events = mockStream.drainEvents()
		require.Len(t, events, 0, "Duplicate replaced block event should be skipped")

		// Check that skipping log was generated
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "Skip log should be present")

		// Different reference should be sent
		logs.Clear()
		replacedEvent2 := engine.InteropReplacedBlockEvent{Ref: l2Ref2.BlockRef(), Envelope: envelope}
		result = mm.OnEvent(replacedEvent2)
		require.True(t, result, "InteropReplacedBlockEvent with different ref should be handled")

		events = mockStream.drainEvents()
		require.Len(t, events, 1, "Replaced block event with different ref should be sent")
		require.NotNil(t, events[0].ReplaceBlock)
		require.Equal(t, l2Ref2.BlockRef(), events[0].ReplaceBlock.Replacement)

		// Check that no skipping log was generated for different ref
		require.Nil(t, logs.FindLog(testlog.NewLevelFilter(log.LevelWarn), hasSkipLog), "No skip log for different ref")
	})

	t.Run("NonInteropEvents", func(t *testing.T) {
		// Create a config with Interop activation at a future time
		interopTime := uint64(2000)
		preInteropCfg := &rollup.Config{
			L2ChainID:   big.NewInt(123),
			InteropTime: &interopTime,
		}

		preInteropMM := &ManagedMode{
			log:        logger,
			cfg:        preInteropCfg,
			events:     &mockEventStream{},
			lastUnsafe: newEventTimestamp[eth.BlockID](50 * time.Millisecond),
		}

		ref := eth.L2BlockRef{
			Hash:   common.Hash{1},
			Number: 100,
			Time:   1000,
		}

		unsafeEvent := engine.UnsafeUpdateEvent{Ref: ref}
		result := preInteropMM.OnEvent(unsafeEvent)
		require.False(t, result, "Pre-Interop UnsafeUpdateEvent should not be handled")

		events := preInteropMM.events.(*mockEventStream).drainEvents()
		require.Len(t, events, 0, "Pre-Interop events should not be sent")
	})
}
