package derive

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// stubBatchProvider is a minimal SingularBatchProvider for testing.
// It only needs to satisfy the interface; its methods are not called
// in the race condition test path (except FlushChannel via DepositsOnlyAttributes).
type stubBatchProvider struct {
	origin eth.L1BlockRef
}

func (s *stubBatchProvider) Origin() eth.L1BlockRef { return s.origin }
func (s *stubBatchProvider) FlushChannel()          {}
func (s *stubBatchProvider) Reset(_ context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	return nil
}
func (s *stubBatchProvider) NextBatch(_ context.Context, _ eth.L2BlockRef) (*SingularBatch, bool, error) {
	return nil, false, nil
}

// makeTestAttribs creates test AttributesWithParent for the race condition tests.
func makeTestAttribs(rng *rand.Rand) *AttributesWithParent {
	parent := testutils.RandomL2BlockRef(rng)
	derivedFrom := testutils.RandomBlockRef(rng)

	return &AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:    eth.Uint64Quantity(parent.Time + 2),
			Transactions: []eth.Data{eth.Data("deposit-tx")},
			NoTxPool:     true,
		},
		Parent:      parent,
		Concluding:  false,
		DerivedFrom: derivedFrom,
	}
}

// TestDepositsOnlyRace_AttributesQueueResetClearsLastAttribs verifies the
// low-level precondition for the race: when the AttributesQueue is reset,
// lastAttribs becomes nil and DepositsOnlyAttributes fails with
// "no attributes generated yet".
//
// In production this happens when a pipeline reset interleaves between a
// BuildStartEvent (which used the previously-derived attributes) and a
// PayloadProcessEvent that deems the payload invalid and requests
// deposits-only attributes via DepositsOnlyPayloadAttributesRequestEvent.
func TestDepositsOnlyRace_AttributesQueueResetClearsLastAttribs(t *testing.T) {
	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		DepositContractAddress: common.Address{0xbb},
		L1SystemConfigAddress:  common.Address{0xcc},
	}

	rng := rand.New(rand.NewSource(9999))
	attribs := makeTestAttribs(rng)

	stub := &stubBatchProvider{origin: attribs.DerivedFrom}
	aq := NewAttributesQueue(testlog.Logger(t, log.LevelError), cfg, nil, stub)

	// Step 1: Simulate that the pipeline has derived attributes (sets lastAttribs).
	// In production, this happens when PipelineStepEvent → pipeline.Step() →
	// AttributesQueue.NextAttributes() successfully returns attributes.
	aq.lastAttribs = attribs

	parent := attribs.Parent.ID()
	derivedFrom := attribs.DerivedFrom

	// Sanity: DepositsOnlyAttributes works when lastAttribs is set.
	depositsOnly, err := aq.DepositsOnlyAttributes(parent, derivedFrom)
	require.NoError(t, err)
	require.NotNil(t, depositsOnly)

	// Step 2: Pipeline reset clears lastAttribs.
	// In production this is triggered by ResetEvent → ResetEngineRequestEvent →
	// onResetEngineRequest → forceReset → Pipeline.Reset() → stage resets →
	// AttributesQueue.Reset() → aq.reset() → lastAttribs = nil
	aq.reset()
	require.Nil(t, aq.lastAttribs, "lastAttribs should be nil after reset")

	// Step 3: DepositsOnlyAttributes now fails — this is the race condition.
	// In production, the DepositsOnlyPayloadAttributesRequestEvent arrives after
	// the reset has cleared lastAttribs, causing a CriticalErrorEvent that kills the node.
	_, err = aq.DepositsOnlyAttributes(parent, derivedFrom)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no attributes generated yet",
		"The race condition: DepositsOnlyAttributes fails after pipeline reset clears lastAttribs")
}

// TestDepositsOnlyRace_PipelineDeriverEmitsCriticalError demonstrates the
// end-to-end consequence: when PipelineDeriver handles a
// DepositsOnlyPayloadAttributesRequestEvent after a pipeline reset, it emits
// a CriticalErrorEvent that terminates the node.
//
// The event sequence that triggers this in production:
//  1. Pipeline derives attributes → lastAttribs is set
//  2. BuildStartEvent → onBuildStart → queues ForkchoiceUpdateEvent + BuildStartedEvent
//  3. ForkchoiceUpdateEvent processing triggers a ResetEvent (due to head state change)
//  4. ResetEvent → pipeline reset → stages reset → lastAttribs = nil
//  5. BuildStartedEvent → BuildSealEvent → BuildSealedEvent → PayloadProcessEvent
//  6. Payload is deemed invalid (ExecutionInvalid + Holocene active)
//  7. DepositsOnlyPayloadAttributesRequestEvent is emitted
//  8. PipelineDeriver calls DepositsOnlyAttributes() → lastAttribs == nil → error
//  9. PipelineDeriver emits CriticalErrorEvent → node exits
func TestDepositsOnlyRace_PipelineDeriverEmitsCriticalError(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)

	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		DepositContractAddress: common.Address{0xbb},
		L1SystemConfigAddress:  common.Address{0xcc},
	}

	rng := rand.New(rand.NewSource(7777))
	attribs := makeTestAttribs(rng)

	parent := attribs.Parent.ID()
	derivedFrom := attribs.DerivedFrom

	// Build a minimal pipeline with a real AttributesQueue.
	stub := &stubBatchProvider{origin: derivedFrom}
	aq := NewAttributesQueue(logger, cfg, nil, stub)

	// Simulate: attributes were derived, then reset cleared them.
	aq.lastAttribs = attribs
	aq.reset() // <-- the race: reset happens between derivation and deposits-only request

	// Create a DerivationPipeline with just the attributes queue wired in.
	pipeline := &DerivationPipeline{
		log:       logger,
		rollupCfg: cfg,
		attrib:    aq,
		resetting: 100, // pretend all stages are reset (> len(stages))
	}

	deriver := NewPipelineDeriver(context.Background(), pipeline)

	// Set up the event system.
	ctx := context.Background()
	executor := event.NewGlobalSynchronous(ctx)
	sys := event.NewSystem(logger, executor)
	defer sys.Stop()

	// Register the pipeline deriver.
	sys.Register("pipeline", deriver)

	// Register a spy to capture the CriticalErrorEvent.
	var gotCriticalError bool
	var criticalErr error
	spy := event.DeriverFunc(func(ctx context.Context, ev event.Event) bool {
		if ce, ok := ev.(event.CriticalErrorEvent); ok {
			gotCriticalError = true
			criticalErr = ce.Err
			return true
		}
		return false
	})
	collectorEmitter := sys.Register("spy", spy)

	// Emit the DepositsOnlyPayloadAttributesRequestEvent — this is what happens
	// when the engine deems a payload invalid under Holocene.
	collectorEmitter.Emit(ctx, DepositsOnlyPayloadAttributesRequestEvent{
		Parent:      parent,
		DerivedFrom: derivedFrom,
	})

	// Drain the event queue to process everything synchronously.
	require.NoError(t, executor.Drain())

	// Verify: the PipelineDeriver should have emitted a CriticalErrorEvent because
	// lastAttribs was nil (cleared by the reset) when DepositsOnlyAttributes was called.
	require.True(t, gotCriticalError,
		"Expected CriticalErrorEvent when DepositsOnlyPayloadAttributesRequestEvent "+
			"is processed after a pipeline reset clears lastAttribs")
	require.Contains(t, criticalErr.Error(), "no attributes generated yet",
		"The critical error should indicate that no attributes were available")
	require.Contains(t, criticalErr.Error(), "deriving deposits-only attributes",
		"The error should be wrapped with the deposits-only context")

	// This test demonstrates that the race condition results in a CriticalErrorEvent
	// which terminates the node. The fix should handle this case gracefully — either
	// by re-deriving the attributes or by deferring the deposits-only request until
	// the pipeline has re-derived attributes after the reset.
	t.Log("Race condition confirmed: pipeline reset between attribute derivation and " +
		"DepositsOnlyPayloadAttributesRequestEvent causes CriticalErrorEvent (node crash)")
}
