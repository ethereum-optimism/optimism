package engine_controller

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// unified mock covers both payload/output paths and BlockAtTimestamp path

func TestOutputV0AtBlockNumber_UsesPayloadWhenAvailable(t *testing.T) {
	t.Parallel()
	l2 := &mockL2{
		ref: eth.L2BlockRef{Number: 100, Time: 123},
		payload: &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
			StateRoot:       eth.Bytes32{0xaa},
			WithdrawalsRoot: func() *common.Hash { h := common.Hash{}; h[0] = 0xbb; return &h }(),
			BlockHash:       func() common.Hash { h := common.Hash{}; h[0] = 0xcc; return h }(),
		}},
	}
	ec := &simpleEngineController{l2: l2, rollup: &rollup.Config{}, log: gethlog.New()}
	out, err := ec.OutputV0AtBlockNumber(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, 1, l2.payloadCalls)
	require.Equal(t, 0, l2.outputCalls) // no fallback
}

func TestOutputV0AtBlockNumber_FallsBackWithoutWithdrawalsRoot(t *testing.T) {
	t.Parallel()
	l2 := &mockL2{
		ref: eth.L2BlockRef{Number: 100, Time: 123},
		// payload without withdrawals root forces fallback
		payload: &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{}},
		output:  &eth.OutputV0{StateRoot: eth.Bytes32{0x01}, MessagePasserStorageRoot: eth.Bytes32{0x02}, BlockHash: func() common.Hash { var h common.Hash; h[0] = 0x03; return h }()},
	}
	ec := &simpleEngineController{l2: l2, rollup: &rollup.Config{}, log: gethlog.New()}
	out, err := ec.OutputV0AtBlockNumber(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, 1, l2.payloadCalls)
	require.Equal(t, 1, l2.outputCalls)
}

type mockL2 struct {
	// Block ref path
	lastNum uint64
	ref     eth.L2BlockRef
	refErr  error

	// Output/payload path
	payload      *eth.ExecutionPayloadEnvelope
	payloadErr   error
	output       *eth.OutputV0
	outputErr    error
	payloadCalls int
	outputCalls  int

	// Block ref by label support
	refsByLabel         map[eth.BlockLabel]eth.L2BlockRef
	refsByLabelAfterFCU map[eth.BlockLabel]eth.L2BlockRef // state after FCU calls
	refByLabelErr       error
	labelCallCount      int

	// Block ref by number support (map for multiple blocks)
	refsByNumber map[uint64]eth.L2BlockRef

	// Payload by number support (map for multiple blocks)
	payloadsByNumber map[uint64]*eth.ExecutionPayloadEnvelope

	// NewPayload tracking
	newPayloadCalls  int
	newPayloadStatus *eth.PayloadStatusV1
	newPayloadErr    error
	lastNewPayload   *eth.ExecutionPayload

	// ForkchoiceUpdate tracking
	fcuCalls     int
	fcuResult    *eth.ForkchoiceUpdatedResult
	fcuErr       error
	lastFCUState *eth.ForkchoiceState
	fcuCompleted bool // tracks if FCU sequence is complete (used to switch label state)
}

func (m *mockL2) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	m.labelCallCount++
	if m.refByLabelErr != nil {
		return eth.L2BlockRef{}, m.refByLabelErr
	}
	// After FCU is complete, use the post-FCU state for verification
	if m.fcuCompleted && m.refsByLabelAfterFCU != nil {
		if ref, ok := m.refsByLabelAfterFCU[label]; ok {
			return ref, nil
		}
	}
	if m.refsByLabel != nil {
		if ref, ok := m.refsByLabel[label]; ok {
			return ref, nil
		}
	}
	return eth.L2BlockRef{Number: 999}, nil
}
func (m *mockL2) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	if m.refErr != nil {
		return eth.L2BlockRef{}, m.refErr
	}
	m.lastNum = num
	if m.refsByNumber != nil {
		if ref, ok := m.refsByNumber[num]; ok {
			return ref, nil
		}
	}
	return m.ref, nil
}
func (m *mockL2) OutputV0AtBlockNumber(ctx context.Context, blockNum uint64) (*eth.OutputV0, error) {
	m.outputCalls++
	return m.output, m.outputErr
}
func (m *mockL2) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	m.payloadCalls++
	if m.payloadsByNumber != nil {
		if payload, ok := m.payloadsByNumber[number]; ok {
			return payload, nil
		}
	}
	return m.payload, m.payloadErr
}
func (m *mockL2) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, payloadAttributes *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	m.fcuCalls++
	m.lastFCUState = state
	if m.fcuErr != nil {
		return nil, m.fcuErr
	}
	if m.fcuResult != nil {
		return m.fcuResult, nil
	}
	// Mark FCU as completed after second call (synthetic + target)
	if m.fcuCalls >= 2 {
		m.fcuCompleted = true
	}
	return &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil
}
func (m *mockL2) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}
func (m *mockL2) Close() {
}
func (m *mockL2) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	m.newPayloadCalls++
	m.lastNewPayload = payload
	if m.newPayloadErr != nil {
		return nil, m.newPayloadErr
	}
	if m.newPayloadStatus != nil {
		return m.newPayloadStatus, nil
	}
	return &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil
}

var _ l2Provider = (*mockL2)(nil)

func TestEngineController_TargetBlockNumber(t *testing.T) {
	t.Parallel()
	rcfg := &rollup.Config{Genesis: rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: 1_000}, BlockTime: 2, L2ChainID: big.NewInt(420)}
	m := &mockL2{ref: eth.L2BlockRef{Number: 0, Time: 0}}
	ec := &simpleEngineController{l2: m, rollup: rcfg, log: gethlog.New()}

	// ts = genesis + 2*3 => block #3, with safe head above target
	numRef, err := ec.BlockAtTimestamp(context.Background(), 1_000+2*3, eth.Safe)
	require.NoError(t, err)
	require.Equal(t, uint64(3), m.lastNum)
	require.Equal(t, m.ref, numRef)
	// ts = genesis + 2*1000 => block #1000, with safe head now below target
	_, err = ec.BlockAtTimestamp(context.Background(), 1_000+2*1000, eth.Safe)
	require.ErrorIs(t, err, ethereum.NotFound)
}

func TestEngineController_SentinelErrors(t *testing.T) {
	t.Parallel()
	ec := &simpleEngineController{l2: nil, rollup: nil}
	_, err := ec.BlockAtTimestamp(context.Background(), 0, eth.Safe)
	require.ErrorIs(t, err, ErrNoEngineClient)

	ec = &simpleEngineController{l2: &mockL2{}, rollup: nil}
	_, err = ec.BlockAtTimestamp(context.Background(), 0, eth.Safe)
	require.ErrorIs(t, err, ErrNoRollupConfig)
}
