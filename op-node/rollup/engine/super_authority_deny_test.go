package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// mockSuperAuthority implements SuperAuthority for testing.
type mockSuperAuthority struct {
	deniedBlocks map[uint64]common.Hash
	shouldError  bool
}

func newMockSuperAuthority() *mockSuperAuthority {
	return &mockSuperAuthority{
		deniedBlocks: make(map[uint64]common.Hash),
	}
}

func (m *mockSuperAuthority) denyBlock(blockNumber uint64, hash common.Hash) {
	m.deniedBlocks[blockNumber] = hash
}

func (m *mockSuperAuthority) IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error) {
	if m.shouldError {
		return false, fmt.Errorf("superauthority check failed")
	}
	deniedHash, exists := m.deniedBlocks[blockNumber]
	if exists && deniedHash == payloadHash {
		return true, nil
	}
	return false, nil
}

// superAuthorityTestCase defines a test scenario for SuperAuthority behavior
type superAuthorityTestCase struct {
	name string
	// setup is called to configure the test scenario
	// Returns: engine (nil if not needed), superAuthority (nil if testing nil case), derivedFrom
	setup func(payload *eth.ExecutionPayloadEnvelope) (*testutils.MockEngine, rollup.SuperAuthority, eth.L1BlockRef)
	// expectations sets up expected calls on the emitter and engine
	expectations func(emitter *testutils.MockEmitter, engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope)
}

func TestSuperAuthority(t *testing.T) {
	tests := []superAuthorityTestCase{
		{
			name: "DeniedPayload_EmitsDepositsOnlyRequest",
			setup: func(payload *eth.ExecutionPayloadEnvelope) (*testutils.MockEngine, rollup.SuperAuthority, eth.L1BlockRef) {
				sa := newMockSuperAuthority()
				sa.denyBlock(uint64(payload.ExecutionPayload.BlockNumber), payload.ExecutionPayload.BlockHash)
				// Need DerivedFrom for Holocene path
				return nil, sa, eth.L1BlockRef{Number: 1}
			},
			expectations: func(emitter *testutils.MockEmitter, engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope) {
				emitter.ExpectOnceType("DepositsOnlyPayloadAttributesRequestEvent")
			},
		},
		{
			name: "AllowedPayload_Proceeds",
			setup: func(payload *eth.ExecutionPayloadEnvelope) (*testutils.MockEngine, rollup.SuperAuthority, eth.L1BlockRef) {
				sa := newMockSuperAuthority()
				// Do NOT deny the payload
				return &testutils.MockEngine{}, sa, eth.L1BlockRef{}
			},
			expectations: func(emitter *testutils.MockEmitter, engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope) {
				engine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
				emitter.ExpectOnceType("PayloadSuccessEvent")
			},
		},
		{
			name: "Error_ProceedsWithPayload",
			setup: func(payload *eth.ExecutionPayloadEnvelope) (*testutils.MockEngine, rollup.SuperAuthority, eth.L1BlockRef) {
				sa := newMockSuperAuthority()
				sa.shouldError = true
				return &testutils.MockEngine{}, sa, eth.L1BlockRef{}
			},
			expectations: func(emitter *testutils.MockEmitter, engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope) {
				// Despite error, expect NewPayload (graceful degradation)
				engine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
				emitter.ExpectOnceType("PayloadSuccessEvent")
			},
		},
		{
			name: "NilAuthority_Proceeds",
			setup: func(payload *eth.ExecutionPayloadEnvelope) (*testutils.MockEngine, rollup.SuperAuthority, eth.L1BlockRef) {
				return &testutils.MockEngine{}, nil, eth.L1BlockRef{}
			},
			expectations: func(emitter *testutils.MockEmitter, engine *testutils.MockEngine, payload *eth.ExecutionPayloadEnvelope) {
				engine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
				emitter.ExpectOnceType("PayloadSuccessEvent")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runSuperAuthorityTest(t, tc)
		})
	}
}

func runSuperAuthorityTest(t *testing.T, tc superAuthorityTestCase) {
	cfg, _, _, payload := buildSimpleCfgAndPayload(t)
	emitter := &testutils.MockEmitter{}

	engine, sa, derivedFrom := tc.setup(payload)
	tc.expectations(emitter, engine, payload)

	ec := NewEngineController(
		context.Background(),
		engine,
		testlog.Logger(t, 0),
		metrics.NoopMetrics,
		cfg,
		&sync.Config{},
		false,
		&testutils.MockL1Source{},
		emitter,
		sa,
	)

	blockRef := eth.L2BlockRef{
		Hash:       payload.ExecutionPayload.BlockHash,
		Number:     uint64(payload.ExecutionPayload.BlockNumber),
		ParentHash: payload.ExecutionPayload.ParentHash,
		Time:       uint64(payload.ExecutionPayload.Timestamp),
	}

	ec.onPayloadProcess(context.Background(), PayloadProcessEvent{
		Envelope:    payload,
		Ref:         blockRef,
		DerivedFrom: derivedFrom,
	})

	if engine != nil {
		engine.AssertExpectations(t)
	}
	emitter.AssertExpectations(t)
}

// Ensure derive.DepositsOnlyPayloadAttributesRequestEvent is referenced to verify import
var _ = derive.DepositsOnlyPayloadAttributesRequestEvent{}

// Ensure rollup is imported (used by buildSimpleCfgAndPayload)
var _ *rollup.Config
