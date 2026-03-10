package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
)

// collectingEmitter records all emitted events in order.
type collectingEmitter struct {
	events []event.Event
}

func (c *collectingEmitter) Emit(_ context.Context, ev event.Event) {
	c.events = append(c.events, ev)
}

func (c *collectingEmitter) reset() {
	c.events = nil
}

func (c *collectingEmitter) eventsOfType(typ string) []event.Event {
	var result []event.Event
	for _, ev := range c.events {
		if ev.String() == typ {
			result = append(result, ev)
		}
	}
	return result
}

func TestInteropInvalidateBlockEvent_DeferredDuringBuild(t *testing.T) {
	cfg := &rollup.Config{}
	emitter := &collectingEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	interopAttribs := &derive.AttributesWithParent{
		DerivedFrom: ReplaceBlockSource,
		Parent:      eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 10},
		Attributes:  &eth.PayloadAttributes{},
	}
	interopEvent := InteropInvalidateBlockEvent{
		Invalidated: eth.BlockRef{Hash: common.Hash{0xbb}, Number: 11},
		Attributes:  interopAttribs,
	}

	// When no build is in progress, the event should emit BuildStartEvent immediately
	ec.OnEvent(context.Background(), interopEvent)
	require.Len(t, emitter.eventsOfType("build-start"), 1)
	require.False(t, ec.buildInProgress)
	require.Nil(t, ec.pendingInteropInvalidation)

	emitter.reset()

	// Simulate a build in progress
	ec.buildInProgress = true

	// Now the event should be deferred
	ec.OnEvent(context.Background(), interopEvent)
	require.Len(t, emitter.events, 0, "no events should be emitted while build is in progress")
	require.NotNil(t, ec.pendingInteropInvalidation)
	require.Equal(t, interopEvent.Invalidated, ec.pendingInteropInvalidation.Invalidated)
}

func TestInteropInvalidateBlockEvent_ReplayedOnBuildComplete(t *testing.T) {
	cfg := &rollup.Config{}
	emitter := &collectingEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	interopAttribs := &derive.AttributesWithParent{
		DerivedFrom: ReplaceBlockSource,
		Parent:      eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 10},
		Attributes:  &eth.PayloadAttributes{},
	}

	// Set up deferred invalidation
	ec.buildInProgress = true
	ec.pendingInteropInvalidation = &InteropInvalidateBlockEvent{
		Invalidated: eth.BlockRef{Hash: common.Hash{0xbb}, Number: 11},
		Attributes:  interopAttribs,
	}

	// Clear build in progress should replay the deferred event
	ec.clearBuildInProgress(context.Background())

	require.False(t, ec.buildInProgress)
	require.Nil(t, ec.pendingInteropInvalidation)
	// Should have re-emitted an InteropInvalidateBlockEvent
	require.Len(t, emitter.events, 1)
	replayedEv, ok := emitter.events[0].(InteropInvalidateBlockEvent)
	require.True(t, ok, "replayed event should be InteropInvalidateBlockEvent")
	require.Equal(t, common.Hash{0xbb}, replayedEv.Invalidated.Hash)
}

func TestInteropInvalidateBlockEvent_NotReplayedWithoutPending(t *testing.T) {
	cfg := &rollup.Config{}
	emitter := &collectingEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	ec.buildInProgress = true
	// No pending interop invalidation
	ec.clearBuildInProgress(context.Background())

	require.False(t, ec.buildInProgress)
	require.Len(t, emitter.events, 0)
}

func TestBuildLifecycle_SetsAndClearsBuildInProgress(t *testing.T) {
	// Test that a successful build start sets buildInProgress,
	// and onPayloadProcess clears it.
	cfg := &rollup.Config{}
	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	parent := eth.L2BlockRef{Hash: common.Hash{0x01}, Number: 5}
	derivedFrom := eth.L1BlockRef{Hash: common.Hash{0x02}, Number: 100, Time: 1000}

	attribs := &derive.AttributesWithParent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
		Attributes:  &eth.PayloadAttributes{},
	}

	payloadID := eth.PayloadID{0x01}

	// Mock startPayload success
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{HeadBlockHash: parent.Hash},
		attribs.Attributes,
		&eth.ForkchoiceUpdatedResult{
			PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
			PayloadID:     &payloadID,
		},
		nil,
	)

	emitter.Mock.On("Emit", mock.Anything).Maybe()

	require.False(t, ec.buildInProgress)
	ec.OnEvent(context.Background(), BuildStartEvent{Attributes: attribs})
	require.True(t, ec.buildInProgress, "buildInProgress should be true after successful build start")
}

func TestBuildCancel_ClearsBuildInProgress(t *testing.T) {
	cfg := &rollup.Config{}
	mockEngine := &testutils.MockEngine{}
	emitter := &collectingEmitter{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	ec.buildInProgress = true

	info := eth.PayloadInfo{ID: eth.PayloadID{0x01}}
	mockEngine.ExpectGetPayload(info.ID, nil, nil)

	ec.OnEvent(context.Background(), BuildCancelEvent{Info: info, Force: true})
	require.False(t, ec.buildInProgress)
}

func TestBuildSealError_ClearsBuildInProgress(t *testing.T) {
	cfg := &rollup.Config{}
	mockEngine := &testutils.MockEngine{}
	emitter := &collectingEmitter{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	ec.buildInProgress = true

	info := eth.PayloadInfo{ID: eth.PayloadID{0x01}}
	mockEngine.ExpectGetPayload(info.ID, nil, errMockGetPayload)

	ec.OnEvent(context.Background(), BuildSealEvent{
		Info:        info,
		DerivedFrom: eth.L1BlockRef{Hash: common.Hash{0x02}, Number: 100},
	})
	require.False(t, ec.buildInProgress, "buildInProgress should be cleared on seal error")
	require.Len(t, emitter.eventsOfType("payload-seal-expired-error"), 1)
}

var errMockGetPayload = &mockRPCError{code: -32001, msg: "unknown payload"}

type mockRPCError struct {
	code int
	msg  string
}

func (e *mockRPCError) Error() string  { return e.msg }
func (e *mockRPCError) ErrorCode() int { return e.code }

func TestInteropInvalidateBlockEvent_FullRaceScenario(t *testing.T) {
	// Simulate the exact race condition from the bug report:
	// 1. Build starts for attribsN
	// 2. InteropInvalidateBlockEvent arrives during the build
	// 3. Build completes (PayloadProcessEvent with success)
	// 4. Deferred interop event replays
	//
	// Verify that lastBuildAttribs is NOT overwritten before PayloadProcess reads it.

	holoceneTime := uint64(0)
	cfg := &rollup.Config{HoloceneTime: &holoceneTime}
	emitter := &collectingEmitter{}
	mockEngine := &testutils.MockEngine{}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	parent := eth.L2BlockRef{Hash: common.Hash{0x01}, Number: 5}
	derivedFrom := eth.L1BlockRef{Hash: common.Hash{0x02}, Number: 100, Time: 1000}

	originalAttribs := &derive.AttributesWithParent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
		Attributes:  &eth.PayloadAttributes{},
	}

	interopAttribs := &derive.AttributesWithParent{
		DerivedFrom: ReplaceBlockSource,
		Parent:      parent,
		Attributes:  &eth.PayloadAttributes{},
	}

	// Step 1: Simulate build in progress with originalAttribs
	ec.buildInProgress = true
	ec.lastBuildAttribs = originalAttribs

	// Step 2: InteropInvalidateBlockEvent arrives
	ec.OnEvent(context.Background(), InteropInvalidateBlockEvent{
		Invalidated: eth.BlockRef{Hash: common.Hash{0xbb}, Number: 6},
		Attributes:  interopAttribs,
	})

	// Verify it was deferred
	require.NotNil(t, ec.pendingInteropInvalidation)
	require.True(t, ec.buildInProgress)
	// Crucially: lastBuildAttribs is still the original
	require.Equal(t, originalAttribs, ec.lastBuildAttribs)

	// Step 3: PayloadProcessEvent arrives (invalid payload triggers deposits-only)
	ref := eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 6, ParentHash: parent.Hash}
	envelope := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash:   ref.Hash,
			BlockNumber: eth.Uint64Quantity(ref.Number),
		},
	}

	mockEngine.ExpectNewPayload(envelope.ExecutionPayload, nil,
		&eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, nil)

	emitter.reset()
	ec.OnEvent(context.Background(), PayloadProcessEvent{
		Concluding:  true,
		DerivedFrom: derivedFrom,
		Envelope:    envelope,
		Ref:         ref,
	})

	// After PayloadProcess:
	// - buildInProgress should be cleared
	// - The deferred interop event should be replayed
	require.False(t, ec.buildInProgress)
	require.Nil(t, ec.pendingInteropInvalidation)

	// Should have emitted deposits-only request (from the invalid payload) AND
	// replayed the interop invalidation event
	depositsOnlyEvents := emitter.eventsOfType("deposits-only-payload-attributes-request")
	interopEvents := emitter.eventsOfType("interop-invalidate-block")
	require.Len(t, depositsOnlyEvents, 1, "should emit deposits-only request for invalid payload")
	require.Len(t, interopEvents, 1, "should replay the deferred interop invalidation")
}
