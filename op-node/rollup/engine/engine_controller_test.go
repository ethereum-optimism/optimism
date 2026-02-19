package engine

import (
	"context"
	"errors"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestInvalidPayloadDropsHead(t *testing.T) {
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	payload := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash: common.Hash{0x01},
	}}

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})

	// Add an unsafe payload requests a forkchoice update via engine controller
	ec.AddUnsafePayload(context.Background(), payload)

	require.NotNil(t, ec.unsafePayloads.Peek())

	// Mark it invalid; it should be dropped if it matches the queue head
	ec.OnEvent(context.Background(), PayloadInvalidEvent{Envelope: payload})
	require.Nil(t, ec.unsafePayloads.Peek())
}

// buildSimpleCfgAndPayload creates a minimal rollup config and a valid payload (A1) on top of A0.
func buildSimpleCfgAndPayload(t *testing.T) (*rollup.Config, eth.L2BlockRef, eth.L2BlockRef, *eth.ExecutionPayloadEnvelope) {
	t.Helper()
	rng := mrand.New(mrand.NewSource(1234))
	refA := testutils.RandomBlockRef(rng)

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}

	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}

	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	// Populate necessary L1 info fields
	aL1Info := &testutils.MockBlockInfo{
		InfoParentHash:  refA.ParentHash,
		InfoNum:         refA.Number,
		InfoTime:        refA.Time,
		InfoHash:        refA.Hash,
		InfoBaseFee:     big.NewInt(1),
		InfoBlobBaseFee: big.NewInt(1),
		InfoReceiptRoot: gethtypes.EmptyRootHash,
		InfoRoot:        testutils.RandomHash(rng),
		InfoGasUsed:     rng.Uint64(),
	}
	a1L1Info, err := derive.L1InfoDepositBytes(cfg, params.SepoliaChainConfig, cfg.Genesis.SystemConfig, refA1.SequenceNumber, aL1Info, refA1.Time)
	require.NoError(t, err)

	payloadA1 := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		ParentHash:   refA1.ParentHash,
		BlockNumber:  eth.Uint64Quantity(refA1.Number),
		Timestamp:    eth.Uint64Quantity(refA1.Time),
		BlockHash:    refA1.Hash,
		Transactions: []eth.Data{a1L1Info},
	}}
	return cfg, refA0, refA1, payloadA1
}

func TestOnUnsafePayload_EnqueueEmit(t *testing.T) {
	cfg, _, _, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, nil)

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})

	ec.AddUnsafePayload(context.Background(), payloadA1)

	got := ec.unsafePayloads.Peek()
	require.NotNil(t, got)
	require.Equal(t, payloadA1, got)
}

func TestOnForkchoiceUpdate_ProcessRetryAndPop(t *testing.T) {
	cfg, refA0, refA1, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	mockEngine := &testutils.MockEngine{}
	cl := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, false, &testutils.MockL1Source{}, emitter, nil)

	// queue payload A1
	emitter.ExpectOnceType("UnsafeUpdateEvent")
	emitter.ExpectOnceType("PayloadInvalidEvent")
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")
	cl.AddUnsafePayload(context.Background(), payloadA1)

	// applicable forkchoice -> process once
	mockEngine.ExpectGetPayload(eth.PayloadID{}, payloadA1, nil)
	mockEngine.ExpectNewPayload(payloadA1.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil)
	mockEngine.ExpectForkchoiceUpdate(&eth.ForkchoiceState{HeadBlockHash: refA1.Hash, SafeBlockHash: common.Hash{}, FinalizedBlockHash: common.Hash{}}, nil, &eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil)
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.NotNil(t, cl.unsafePayloads.Peek(), "should not pop yet")

	// same forkchoice -> retry
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA0, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.NotNil(t, cl.unsafePayloads.Peek(), "still pending")

	// after applied (unsafe head == A1) -> pop
	cl.OnEvent(context.Background(), ForkchoiceUpdateEvent{UnsafeL2Head: refA1, SafeL2Head: refA0, FinalizedL2Head: refA0})
	require.Nil(t, cl.unsafePayloads.Peek())
}

func TestPeekUnsafePayload(t *testing.T) {
	cfg, _, _, payloadA1 := buildSimpleCfgAndPayload(t)

	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync}, false, &testutils.MockL1Source{}, emitter, nil)

	// empty -> zero
	_, ref := ec.PeekUnsafePayload()
	require.Equal(t, eth.L2BlockRef{}, ref)

	// queue -> returns derived ref
	_ = ec.unsafePayloads.Push(payloadA1)
	want, err := derive.PayloadToBlockRef(cfg, payloadA1.ExecutionPayload)
	require.NoError(t, err)

	_, ref = ec.PeekUnsafePayload()
	require.Equal(t, want, ref)
}

func TestPeekUnsafePayload_OnDeriveErrorReturnsZero(t *testing.T) {
	// missing L1-info in txs will cause derive error
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{SyncMode: sync.CLSync}, false, &testutils.MockL1Source{}, emitter, nil)

	bad := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{BlockNumber: 1, BlockHash: common.Hash{0xaa}}}
	_ = ec.unsafePayloads.Push(bad)
	_, ref := ec.PeekUnsafePayload()
	require.Equal(t, eth.L2BlockRef{}, ref)
}

func TestInvalidPayloadForNonHead_NoDrop(t *testing.T) {
	emitter := &testutils.MockEmitter{}
	ec := NewEngineController(context.Background(), nil, testlog.Logger(t, 0), metrics.NoopMetrics, &rollup.Config{}, &sync.Config{SyncMode: sync.CLSync}, false, &testutils.MockL1Source{}, emitter, nil)

	// Head payload (lower block number)
	head := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 1,
		BlockHash:   common.Hash{0x01},
	}}
	// Non-head payload (higher block number)
	other := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockNumber: 2,
		BlockHash:   common.Hash{0x02},
	}}

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})
	ec.AddUnsafePayload(context.Background(), head)

	emitter.ExpectOnce(PayloadInvalidEvent{})
	emitter.ExpectOnce(ForkchoiceUpdateEvent{})
	ec.AddUnsafePayload(context.Background(), other)

	// Invalidate non-head should not drop head
	ec.OnEvent(context.Background(), PayloadInvalidEvent{Envelope: other})
	require.Equal(t, 2, ec.unsafePayloads.Len())
	require.Equal(t, head, ec.unsafePayloads.Peek())
}

// note: nil-envelope behavior is not tested to match current implementation

// TestEngineController_SafeL2Head tests SafeL2Head behavior with various configurations
func TestEngineController_SafeL2Head(t *testing.T) {
	tests := []struct {
		name              string
		supervisorEnabled bool
		setupSuperAuth    func() *mockSuperAuthority
		setupLocalSafe    *eth.L2BlockRef
		setupDeprecated   *eth.L2BlockRef
		setupEngine       func(*testutils.MockEngine)
		expectPanic       string
		expectResult      *eth.L2BlockRef
	}{
		{
			name:              "with SuperAuthority returns verified block",
			supervisorEnabled: true,
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					fullyVerifiedL2Head: eth.BlockID{Hash: common.Hash{0xbb}, Number: 50},
				}
			},
			setupLocalSafe: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByHash(common.Hash{0xbb}, eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}, nil)
			},
			expectResult: &eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50},
		},
		{
			name:              "with SuperAuthority empty BlockID returns empty",
			supervisorEnabled: true,
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{fullyVerifiedL2Head: eth.BlockID{}}
			},
			expectResult: &eth.L2BlockRef{},
		},
		{
			name:              "without SuperAuthority but supervisor enabled uses deprecated",
			supervisorEnabled: true,
			setupSuperAuth:    func() *mockSuperAuthority { return nil },
			setupDeprecated:   &eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 200},
			expectResult:      &eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 200},
		},
		{
			name:              "without SuperAuthority and supervisor disabled uses local safe",
			supervisorEnabled: false,
			setupSuperAuth:    func() *mockSuperAuthority { return nil },
			setupLocalSafe:    &eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 300},
			expectResult:      &eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 300},
		},
		{
			name:              "falls back to local safe when SuperAuthority block ahead of local safe",
			supervisorEnabled: true,
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					fullyVerifiedL2Head: eth.BlockID{Hash: common.Hash{0xff}, Number: 200},
				}
			},
			setupLocalSafe: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			expectResult:   &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
		},
		{
			name:              "panics when SuperAuthority block unknown to engine",
			supervisorEnabled: true,
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					fullyVerifiedL2Head: eth.BlockID{Hash: common.Hash{0x99}, Number: 50},
				}
			},
			setupLocalSafe: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByHash(common.Hash{0x99}, eth.L2BlockRef{}, errors.New("block not found"))
			},
			expectPanic: "superAuthority supplied an identifier for the safe head which is not known to the engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockEngine *testutils.MockEngine
			if tt.setupEngine != nil {
				mockEngine = &testutils.MockEngine{}
			}

			cfg := &rollup.Config{}
			emitter := &testutils.MockEmitter{}
			var superAuthority rollup.SuperAuthority
			if tt.setupSuperAuth != nil {
				if sa := tt.setupSuperAuth(); sa != nil {
					superAuthority = sa
				}
			}
			ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, tt.supervisorEnabled, &testutils.MockL1Source{}, emitter, superAuthority)
			if tt.setupLocalSafe != nil {
				ec.SetLocalSafeHead(*tt.setupLocalSafe)
			}
			if tt.setupDeprecated != nil {
				ec.SetSafeHead(*tt.setupDeprecated)
			}

			if tt.setupEngine != nil {
				tt.setupEngine(mockEngine)
			}

			if tt.expectPanic != "" {
				require.PanicsWithValue(t, tt.expectPanic, func() {
					ec.SafeL2Head()
				})
			} else {
				result := ec.SafeL2Head()
				require.Equal(t, *tt.expectResult, result)
			}
		})
	}
}

// TestEngineController_ForkchoiceUpdateUsesSuperAuthority tests that forkchoice
// updates use SafeL2Head() which respects SuperAuthority
func TestEngineController_ForkchoiceUpdateUsesSuperAuthority(t *testing.T) {
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 1000,
		},
		BlockTime: 2,
	}

	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}
	// Set SuperAuthority with verified head at block 60
	verifiedRef := eth.L2BlockRef{
		Hash:   common.Hash{0xdd},
		Number: 60,
	}
	mockSA := &mockSuperAuthority{
		fullyVerifiedL2Head: verifiedRef.ID(),
	}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, true, &testutils.MockL1Source{}, emitter, mockSA)

	// Set heads
	unsafeRef := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localSafeRef := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 80}
	finalizedRef := eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 50}

	ec.unsafeHead = unsafeRef
	ec.SetLocalSafeHead(localSafeRef)
	ec.SetFinalizedHead(finalizedRef)

	// Mock initializeUnknowns calls
	mockEngine.ExpectL2BlockRefByLabel(eth.Unsafe, unsafeRef, nil)
	// SafeL2Head is called multiple times during initialization and forkchoice - be generous
	for i := 0; i < 10; i++ {
		mockEngine.ExpectL2BlockRefByHash(verifiedRef.Hash, verifiedRef, nil)
	}
	mockEngine.ExpectL2BlockRefByLabel(eth.Safe, localSafeRef, nil)
	mockEngine.ExpectL2BlockRefByLabel(eth.Finalized, finalizedRef, nil)

	// Expect emitter events
	emitter.ExpectOnceType("ForkchoiceUpdateEvent")

	// Expect forkchoice update with SuperAuthority's safe head
	expectedFC := eth.ForkchoiceState{
		HeadBlockHash:      unsafeRef.Hash,
		SafeBlockHash:      verifiedRef.Hash, // from SuperAuthority
		FinalizedBlockHash: finalizedRef.Hash,
	}
	mockEngine.ExpectForkchoiceUpdate(&expectedFC, nil, &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
	}, nil)

	// Trigger forkchoice update
	ec.needFCUCall = true
	err := ec.tryUpdateEngineInternal(context.Background())
	require.NoError(t, err)
}

// SuperAuthority tests are in super_authority_deny_test.go
