package engine

import (
	"context"
	"errors"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/mock"
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
			name:              "with SuperAuthority empty BlockID returns genesis",
			supervisorEnabled: true,
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{fullyVerifiedL2Head: eth.BlockID{}}
			},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByNumber(0, eth.L2BlockRef{Hash: common.Hash{0x00}, Number: 0}, nil)
			},
			expectResult: &eth.L2BlockRef{Hash: common.Hash{0x00}, Number: 0},
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
	finalizedRef := eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 50}
	mockSA := &mockSuperAuthority{
		fullyVerifiedL2Head: verifiedRef.ID(),
		finalizedL2Head:     finalizedRef.ID(),
	}
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, true, &testutils.MockL1Source{}, emitter, mockSA)

	// Set heads
	unsafeRef := eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100}
	localSafeRef := eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 80}

	ec.unsafeHead = unsafeRef
	ec.SetLocalSafeHead(localSafeRef)
	ec.SetFinalizedHead(finalizedRef)

	// Mock initializeUnknowns calls
	mockEngine.ExpectL2BlockRefByLabel(eth.Unsafe, unsafeRef, nil)
	// SafeL2Head is called multiple times during initialization and forkchoice - be generous
	for i := 0; i < 10; i++ {
		mockEngine.ExpectL2BlockRefByHash(verifiedRef.Hash, verifiedRef, nil)
	}
	// FinalizedHead is also called and will look up the finalized block by hash
	for i := 0; i < 10; i++ {
		mockEngine.ExpectL2BlockRefByHash(finalizedRef.Hash, finalizedRef, nil)
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

// TestFollowSource_DivergentLocalSafeAndCrossSafe verifies that FollowSource correctly handles
// the case where external cross-safe and local-safe values diverge. After the fix:
//   - Consolidation/reorg checks use eLocalSafeRef (not eSafeBlockRef)
//   - PromoteSafe injects the external cross-safe head
//   - promoteFinalized succeeds because cross-safe is set before finalized is promoted
func TestFollowSource_DivergentLocalSafeAndCrossSafe(t *testing.T) {
	rng := mrand.New(mrand.NewSource(9999))

	// Create block refs for a simple chain: block1 → block2 → block3 → block4 → block5
	l1Origin := testutils.RandomBlockRef(rng)

	block1 := eth.L2BlockRef{
		Hash: testutils.RandomHash(rng), Number: 1,
		ParentHash: testutils.RandomHash(rng), Time: l1Origin.Time + 1,
		L1Origin: l1Origin.ID(), SequenceNumber: 1,
	}
	block2 := eth.L2BlockRef{
		Hash: testutils.RandomHash(rng), Number: 2,
		ParentHash: block1.Hash, Time: l1Origin.Time + 2,
		L1Origin: l1Origin.ID(), SequenceNumber: 2,
	}
	block3 := eth.L2BlockRef{
		Hash: testutils.RandomHash(rng), Number: 3,
		ParentHash: block2.Hash, Time: l1Origin.Time + 3,
		L1Origin: l1Origin.ID(), SequenceNumber: 3,
	}
	block4 := eth.L2BlockRef{
		Hash: testutils.RandomHash(rng), Number: 4,
		ParentHash: block3.Hash, Time: l1Origin.Time + 4,
		L1Origin: l1Origin.ID(), SequenceNumber: 4,
	}
	block5 := eth.L2BlockRef{
		Hash: testutils.RandomHash(rng), Number: 5,
		ParentHash: block4.Hash, Time: l1Origin.Time + 5,
		L1Origin: l1Origin.ID(), SequenceNumber: 5,
	}

	interopTime := uint64(0)
	cfg := &rollup.Config{InteropTime: &interopTime}
	mockEngine := &testutils.MockEngine{}
	emitter := &testutils.MockEmitter{}

	// FollowSourceEnabled=true with no superAuthority:
	//   SafeL2Head() returns deprecatedSafeHead (cross-safe)
	//   FinalizedHead() returns deprecatedFinalizedHead
	// This lets us observe cross-safe independently from local-safe.
	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
		metrics.NoopMetrics, cfg, &sync.Config{L2FollowSourceEndpoint: "http://localhost"}, false, &testutils.MockL1Source{}, emitter, nil)

	// Initial state: unsafe=block5, localSafe=block2, crossSafe=block2, finalized=block1
	ec.unsafeHead = block5
	ec.SetLocalSafeHead(block2)
	ec.SetSafeHead(block2)      // deprecatedSafeHead = block2
	ec.SetFinalizedHead(block1) // deprecatedFinalizedHead = block1
	ec.needFCUCall = false      // reset after setup

	// Mock expectations:
	// Allow any events from the emitter (LocalSafeUpdateEvent, SafeDerivedEvent, etc.)
	emitter.Mock.On("Emit", mock.Anything).Maybe()

	// Consolidation lookup: after fix, uses eLocalSafeRef.Number (5)
	mockEngine.ExpectL2BlockRefByNumber(5, block5, nil)

	// FCU from PromoteSafe's tryUpdateEngine: safe=block4, finalized still block1
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{
			HeadBlockHash:      block5.Hash,
			SafeBlockHash:      block4.Hash,
			FinalizedBlockHash: block1.Hash,
		}, nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil,
	)
	// FCU from promoteFinalized's tryUpdateEngine: finalized now block3
	mockEngine.ExpectForkchoiceUpdate(
		&eth.ForkchoiceState{
			HeadBlockHash:      block5.Hash,
			SafeBlockHash:      block4.Hash,
			FinalizedBlockHash: block3.Hash,
		}, nil,
		&eth.ForkchoiceUpdatedResult{PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid}}, nil,
	)

	// Call FollowSource with divergent cross-safe (block4) and local-safe (block5)
	ec.FollowSource(block4, block5, block3)

	// Assert the final head state
	require.Equal(t, block5, ec.localSafeHead, "localSafeHead should be updated to block5")
	require.Equal(t, block4, ec.deprecatedSafeHead, "deprecatedSafeHead (cross-safe) should be updated to block4")
	require.Equal(t, block3, ec.deprecatedFinalizedHead, "deprecatedFinalizedHead (cross-finalized) should be updated to block3")

	// Assert the invariant: cross-safe <= local-safe
	require.LessOrEqual(t, ec.deprecatedSafeHead.Number, ec.localSafeHead.Number,
		"invariant: cross-safe (deprecatedSafeHead) must not exceed local-safe")
}

// TestEngineController_FinalizedHead tests FinalizedHead behavior with various configurations
func TestEngineController_FinalizedHead(t *testing.T) {
	tests := []struct {
		name            string
		setupSuperAuth  func() *mockSuperAuthority
		setupLocalSafe  *eth.L2BlockRef
		setupLocalFinal *eth.L2BlockRef
		setupEngine     func(*testutils.MockEngine)
		expectPanic     string
		expectResult    *eth.L2BlockRef
	}{
		{
			name: "with SuperAuthority returns finalized block",
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					finalizedL2Head: eth.BlockID{Hash: common.Hash{0xbb}, Number: 50},
				}
			},
			setupLocalSafe:  &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupLocalFinal: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByHash(common.Hash{0xbb}, eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50}, nil)
			},
			expectResult: &eth.L2BlockRef{Hash: common.Hash{0xbb}, Number: 50},
		},
		{
			name: "with SuperAuthority empty BlockID fallback to genesis",
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{finalizedL2Head: eth.BlockID{}}
			},
			setupLocalFinal: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByNumber(0, eth.L2BlockRef{Hash: common.Hash{0x00}, Number: 0}, nil)
			},
			expectResult: &eth.L2BlockRef{Hash: common.Hash{0x00}, Number: 0},
		},
		{
			name: "with SuperAuthority ahead of local safe uses local safe",
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					finalizedL2Head: eth.BlockID{Hash: common.Hash{0xbb}, Number: 50},
				}
			},
			setupLocalSafe:  &eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 40},
			setupLocalFinal: &eth.L2BlockRef{Hash: common.Hash{0xdd}, Number: 30},
			expectResult:    &eth.L2BlockRef{Hash: common.Hash{0xcc}, Number: 40},
		},
		{
			name:           "without SuperAuthority returns zero value",
			setupSuperAuth: func() *mockSuperAuthority { return nil },
			expectResult:   &eth.L2BlockRef{},
		},
		{
			name: "returns empty block when genesis lookup fails",
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{finalizedL2Head: eth.BlockID{}}
			},
			setupLocalFinal: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByNumber(0, eth.L2BlockRef{}, errors.New("genesis not found"))
			},
			expectResult: &eth.L2BlockRef{},
		},
		{
			name: "panics when SuperAuthority block unknown to engine",
			setupSuperAuth: func() *mockSuperAuthority {
				return &mockSuperAuthority{
					finalizedL2Head: eth.BlockID{Hash: common.Hash{0x99}, Number: 50},
				}
			},
			setupLocalSafe:  &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupLocalFinal: &eth.L2BlockRef{Hash: common.Hash{0xaa}, Number: 100},
			setupEngine: func(m *testutils.MockEngine) {
				m.ExpectL2BlockRefByHash(common.Hash{0x99}, eth.L2BlockRef{}, errors.New("block not found"))
			},
			expectPanic: "superAuthority supplied an identifier for the finalized head which is not known to the engine",
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
			ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics, cfg, &sync.Config{}, false, &testutils.MockL1Source{}, emitter, superAuthority)
			if tt.setupLocalSafe != nil {
				ec.SetLocalSafeHead(*tt.setupLocalSafe)
			}
			if tt.setupLocalFinal != nil {
				ec.SetFinalizedHead(*tt.setupLocalFinal)
			}

			if tt.setupEngine != nil {
				tt.setupEngine(mockEngine)
			}

			if tt.expectPanic != "" {
				require.PanicsWithValue(t, tt.expectPanic, func() {
					ec.FinalizedHead()
				})
			} else {
				result := ec.FinalizedHead()
				require.Equal(t, *tt.expectResult, result)
			}
		})
	}
}
