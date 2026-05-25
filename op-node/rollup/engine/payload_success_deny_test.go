package engine

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestOnPayloadSuccess_ReplacementBlockNotDenied verifies the defensive check
// at payload_success.go:onPayloadSuccess that refuses to forceReset to a
// replacement block whose hash is itself on the deny list. Without the check,
// the next reset would re-apply the cap and the chain would loop.
func TestOnPayloadSuccess_ReplacementBlockNotDenied(t *testing.T) {
	cfg, _, _, payload := buildSimpleCfgAndPayload(t)

	ref := eth.L2BlockRef{
		Hash:       payload.ExecutionPayload.BlockHash,
		Number:     uint64(payload.ExecutionPayload.BlockNumber),
		ParentHash: payload.ExecutionPayload.ParentHash,
		Time:       uint64(payload.ExecutionPayload.Timestamp),
	}

	sa := newMockSuperAuthority()
	sa.denyBlock(ref.Number, ref.Hash)

	engine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	// The defensive check must surface a CriticalErrorEvent (which the
	// supernode handles as a process-exit signal) and must NOT proceed to
	// forceReset / tryUpdateEngine. No engine RPCs are expected.
	emitter.ExpectOnceType("CriticalErrorEvent")

	ec := NewEngineController(
		context.Background(),
		engine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		cfg,
		&sync.Config{},
		&testutils.MockL1Source{},
		emitter,
		sa,
	)

	ec.onPayloadSuccess(context.Background(), PayloadSuccessEvent{
		DerivedFrom: ReplaceBlockSource,
		Envelope:    payload,
		Ref:         ref,
	})

	engine.AssertExpectations(t)
	emitter.AssertExpectations(t)
}
