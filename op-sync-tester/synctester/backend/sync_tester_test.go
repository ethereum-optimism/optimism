package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/session"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var _ ReadOnlyELBackend = (*MockELReader)(nil)

type MockELReader struct {
	ChainID hexutil.Big

	BlocksByHash   map[common.Hash]*json.RawMessage
	BlocksByNumber map[rpc.BlockNumber]*json.RawMessage

	PayloadsByHash   map[common.Hash]*eth.ExecutionPayloadEnvelope
	PayloadsByNumber map[uint64]*eth.ExecutionPayloadEnvelope

	ReceiptsByHash   map[common.Hash]json.RawMessage
	ReceiptsByNumber map[rpc.BlockNumber]json.RawMessage

	Latest    *json.RawMessage
	Safe      *json.RawMessage
	Finalized *json.RawMessage
}

func NewMockELReader(chainID eth.ChainID) *MockELReader {
	return &MockELReader{
		ChainID:          hexutil.Big(*chainID.ToBig()),
		BlocksByHash:     make(map[common.Hash]*json.RawMessage),
		BlocksByNumber:   make(map[rpc.BlockNumber]*json.RawMessage),
		PayloadsByHash:   make(map[common.Hash]*eth.ExecutionPayloadEnvelope),
		PayloadsByNumber: make(map[uint64]*eth.ExecutionPayloadEnvelope),
		ReceiptsByHash:   make(map[common.Hash]json.RawMessage),
		ReceiptsByNumber: make(map[rpc.BlockNumber]json.RawMessage),
	}
}

func (m *MockELReader) ChainId(ctx context.Context) (hexutil.Big, error) {
	return m.ChainID, nil
}

func (m *MockELReader) GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	raw, ok := m.BlocksByNumber[number]
	if !ok {
		return nil, ethereum.NotFound
	}
	return *raw, nil
}

func (m *MockELReader) GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	raw, ok := m.BlocksByHash[hash]
	if !ok {
		return nil, ethereum.NotFound
	}
	return *raw, nil
}

func (m *MockELReader) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	env, ok := m.PayloadsByNumber[number]
	if !ok {
		return nil, ethereum.NotFound
	}
	return env, nil
}

func (m *MockELReader) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error) {
	env, ok := m.PayloadsByHash[hash]
	if !ok {
		return nil, ethereum.NotFound
	}
	return env, nil
}

// AddPayload makes the block available by hash and number.
func (m *MockELReader) AddPayload(env *eth.ExecutionPayloadEnvelope) {
	m.PayloadsByHash[env.ExecutionPayload.BlockHash] = env
	m.PayloadsByNumber[uint64(env.ExecutionPayload.BlockNumber)] = env
}

func (m *MockELReader) GetBlockReceiptsJSON(ctx context.Context, bnh rpc.BlockNumberOrHash) (json.RawMessage, error) {
	hash, isHash := bnh.Hash()
	if isHash {
		receipts, ok := m.ReceiptsByHash[hash]
		if !ok {
			return nil, ethereum.NotFound
		}
		return receipts, nil
	}
	number, isNumber := bnh.Number()
	if !isNumber {
		// bnh is not a number and not a hash so return not found
		return nil, ethereum.NotFound
	}
	receipts, ok := m.ReceiptsByNumber[number]
	if !ok {
		return nil, ethereum.NotFound
	}
	return receipts, nil
}

func initTestSyncTester(t *testing.T, chainID eth.ChainID, elReader ReadOnlyELBackend) *SyncTester {
	syncTester := NewSyncTester(testlog.Logger(t, log.LevelInfo), nil, sttypes.SyncTesterID("test"), chainID, elReader)
	return syncTester
}

func TestSyncTester_ChainId(t *testing.T) {
	dummySession := &eth.SyncTesterSession{SessionID: uuid.New().String()}
	tests := []struct {
		name            string
		cfgID           eth.ChainID
		elID            eth.ChainID
		session         *eth.SyncTesterSession
		wantErrContains string
	}{
		{
			name:            "no session",
			cfgID:           eth.ChainIDFromUInt64(1),
			elID:            eth.ChainIDFromUInt64(1),
			wantErrContains: "no session",
		},
		{
			name:    "happy path",
			cfgID:   eth.ChainIDFromUInt64(11155111),
			elID:    eth.ChainIDFromUInt64(11155111),
			session: dummySession,
		},
		{
			name:            "mismatch",
			cfgID:           eth.ChainIDFromUInt64(1),
			elID:            eth.ChainIDFromUInt64(11155111),
			session:         dummySession,
			wantErrContains: "chainID mismatch",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockELReader(tc.elID)
			st := initTestSyncTester(t, tc.cfgID, mock)
			ctx := context.Background()
			if tc.session != nil {
				ctx = session.WithSyncTesterSession(ctx, tc.session)
			}
			got, err := st.ChainId(ctx)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, hexutil.Big(*tc.cfgID.ToBig()), got)
		})
	}
}

func makeBlockRaw(num uint64) *json.RawMessage {
	raw := json.RawMessage(fmt.Sprintf(`{"number":"0x%x"}`, num))
	return &raw
}

func TestSyncTester_GetBlockByHash(t *testing.T) {
	hash := common.HexToHash("0xdeadbeef")
	tests := []struct {
		name            string
		sessionLatest   uint64
		rawNumber       uint64 // block.number returned by EL
		session         *eth.SyncTesterSession
		wantErrContains string
	}{
		{
			name:            "no session",
			sessionLatest:   0,
			rawNumber:       0,
			session:         nil,
			wantErrContains: "no session",
		},
		{
			name:            "block number greater than latest",
			sessionLatest:   100,
			rawNumber:       101, // greater than Latest
			session:         &eth.SyncTesterSession{SessionID: uuid.New().String(), CurrentState: eth.FCUState{Latest: 100}},
			wantErrContains: "not found",
		},
		{
			name:          "happy path",
			sessionLatest: 100,
			rawNumber:     99,
			session:       &eth.SyncTesterSession{SessionID: uuid.New().String(), CurrentState: eth.FCUState{Latest: 100}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := NewMockELReader(eth.ChainIDFromUInt64(1))
			block := makeBlockRaw(tc.rawNumber)
			el.BlocksByHash[hash] = block
			st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), el)
			ctx := context.Background()
			if tc.session != nil {
				ctx = session.WithSyncTesterSession(ctx, tc.session)
			}
			raw, err := st.GetBlockByHash(ctx, hash, false)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, raw)

			var header HeaderNumberOnly
			require.NoError(t, json.Unmarshal(raw, &header))
			require.EqualValues(t, tc.rawNumber, bigs.Uint64Strict(header.Number.ToInt()))
		})
	}
}

func TestSyncTester_GetBlockByNumber(t *testing.T) {
	type testCase struct {
		name            string
		session         *eth.SyncTesterSession
		inNumber        rpc.BlockNumber
		wantNum         uint64
		wantErrContains string
	}

	tests := []testCase{
		{
			name:            "no session",
			session:         nil,
			wantErrContains: "no session",
		},
		{
			name: "happy path: numeric less than latest",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			inNumber: rpc.BlockNumber(99),
			wantNum:  99,
		},
		{
			name: "happy path: label latest returns latest",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			inNumber: rpc.LatestBlockNumber,
			wantNum:  100,
		},
		{
			name: "happy path: label safe returns safe",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      97,
					Finalized: 90,
				},
			},
			inNumber: rpc.SafeBlockNumber,
			wantNum:  97,
		},
		{
			name: "happy path: label finalized returns finalized",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      97,
					Finalized: 92,
				},
			},
			inNumber: rpc.FinalizedBlockNumber,
			wantNum:  92,
		},
		{
			name: "pending returns not found",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			inNumber:        rpc.PendingBlockNumber,
			wantErrContains: "not found",
		},
		{
			name: "earliest label returns not found",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			inNumber:        rpc.EarliestBlockNumber,
			wantErrContains: "not found",
		},
		{
			name: "numeric greater than latest returns not found",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			inNumber:        rpc.BlockNumber(101),
			wantErrContains: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := NewMockELReader(eth.ChainIDFromUInt64(1))
			if tc.session != nil {
				el.BlocksByNumber[rpc.BlockNumber(tc.session.CurrentState.Latest)] = makeBlockRaw(tc.session.CurrentState.Latest)
				el.BlocksByNumber[rpc.BlockNumber(tc.session.CurrentState.Safe)] = makeBlockRaw(tc.session.CurrentState.Safe)
				el.BlocksByNumber[rpc.BlockNumber(tc.session.CurrentState.Finalized)] = makeBlockRaw(tc.session.CurrentState.Finalized)
			}
			el.BlocksByNumber[tc.inNumber] = makeBlockRaw(uint64(tc.inNumber.Int64()))
			st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), el)
			ctx := context.Background()
			if tc.session != nil {
				ctx = session.WithSyncTesterSession(ctx, tc.session)
			}
			raw, err := st.GetBlockByNumber(ctx, tc.inNumber, false)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, raw)
			var header HeaderNumberOnly
			require.NoError(t, json.Unmarshal(raw, &header))
			require.EqualValues(t, tc.wantNum, bigs.Uint64Strict(header.Number.ToInt()))
		})
	}
}

func TestSyncTester_GetBlockReceipts(t *testing.T) {
	// The deposit receipt fields are unknown to upstream go-ethereum and must survive the relay.
	makeReceipts := func(n uint64) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`[{"blockNumber":"0x%x","type":"0x7e","depositNonce":"0x5","depositReceiptVersion":"0x1"}]`, n))
	}
	type testCase struct {
		name            string
		session         *eth.SyncTesterSession
		arg             rpc.BlockNumberOrHash
		seedFn          func(el *MockELReader, s *eth.SyncTesterSession)
		wantFirstBN     uint64
		wantEmpty       bool
		wantErrContains string
	}
	hashGood := common.HexToHash("0xabc1")
	hashTooNew := common.HexToHash("0xabc2")
	hashGenesis := common.HexToHash("0xabc3")
	hashUnknown := common.HexToHash("0xabc4")
	tests := []testCase{
		{
			name:            "no session",
			session:         nil,
			arg:             rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
			wantErrContains: "no session",
		},
		{
			name: "happy: via hash, blockNumber less than latest",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashGood, false),
			seedFn: func(el *MockELReader, s *eth.SyncTesterSession) {
				el.ReceiptsByHash[hashGood] = makeReceipts(s.CurrentState.Latest - 1)
			},
			wantFirstBN: 99,
		},
		{
			name: "bad: via hash, blockNumber >= latest returns not found",
			session: &eth.SyncTesterSession{
				SessionID: uuid.New().String(),
				CurrentState: eth.FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashTooNew, false),
			seedFn: func(el *MockELReader, s *eth.SyncTesterSession) {
				// strictly greater than Latest so the post-check triggers NotFound
				el.ReceiptsByHash[hashTooNew] = makeReceipts(s.CurrentState.Latest + 1)
			},
			wantErrContains: "not found",
		},
		{
			name: "happy: via hash, genesis has no receipts",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 95, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashGenesis, false),
			seedFn: func(el *MockELReader, _ *eth.SyncTesterSession) {
				el.ReceiptsByHash[hashGenesis] = json.RawMessage(`[]`)
			},
			wantEmpty: true,
		},
		{
			name: "bad: via hash, EL returns null for an unknown block",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 95, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashUnknown, false),
			seedFn: func(el *MockELReader, _ *eth.SyncTesterSession) {
				el.ReceiptsByHash[hashUnknown] = json.RawMessage(`null`)
			},
			wantErrContains: "not found",
		},
		{
			name: "happy: label latest returns latest",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 95, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
			seedFn: func(el *MockELReader, s *eth.SyncTesterSession) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Latest)] = makeReceipts(s.CurrentState.Latest)
			},
			wantFirstBN: 100,
		},
		{
			name: "happy: label safe returns safe",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber),
			seedFn: func(el *MockELReader, s *eth.SyncTesterSession) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Safe)] = makeReceipts(s.CurrentState.Safe)
			},
			wantFirstBN: 97,
		},
		{
			name: "happy: label finalized returns finalized",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber),
			seedFn: func(el *MockELReader, s *eth.SyncTesterSession) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Finalized)] = makeReceipts(s.CurrentState.Finalized)
			},
			wantFirstBN: 92,
		},
		{
			name: "happy: numeric less than latest",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(99)),
			seedFn: func(el *MockELReader, _ *eth.SyncTesterSession) {
				el.ReceiptsByNumber[rpc.BlockNumber(99)] = makeReceipts(99)
			},
			wantFirstBN: 99,
		},
		{
			name: "bad: numeric greater than latest returns not found",
			session: &eth.SyncTesterSession{
				SessionID:    uuid.New().String(),
				CurrentState: eth.FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			arg:             rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(101)),
			wantErrContains: "not found",
			// No seeding needed: checkBlockNumber should fail before EL call
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := NewMockELReader(eth.ChainIDFromUInt64(1))
			if tc.seedFn != nil && tc.session != nil {
				tc.seedFn(el, tc.session)
			}
			st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), el)
			ctx := context.Background()
			if tc.session != nil {
				ctx = session.WithSyncTesterSession(ctx, tc.session)
			}
			raw, err := st.GetBlockReceipts(ctx, tc.arg)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			if tc.wantEmpty {
				require.JSONEq(t, `[]`, string(raw))
				return
			}
			require.Equal(t, makeReceipts(tc.wantFirstBN), raw)
		})
	}
}

// makeDepositPayload returns an Ecotone block, as served by the read only EL, whose
// first transaction is a deposit.
func makeDepositPayload(t *testing.T, number uint64, parent common.Hash) *eth.ExecutionPayloadEnvelope {
	deposit := &optypes.DepositTx{
		SourceHash: common.HexToHash("0x01"),
		From:       common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001"),
		To:         &common.Address{0x42},
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       []byte{byte(number)},
	}
	depositRaw, err := deposit.MarshalBinary()
	require.NoError(t, err)
	beaconRoot := common.HexToHash("0xbeac")
	env := &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &beaconRoot,
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    parent,
			FeeRecipient:  common.HexToAddress("0x4200000000000000000000000000000000000011"),
			PrevRandao:    eth.Bytes32{0x07},
			BlockNumber:   eth.Uint64Quantity(number),
			GasLimit:      30_000_000,
			Timestamp:     eth.Uint64Quantity(1000 + 2*number),
			Transactions:  []eth.Data{depositRaw},
			Withdrawals:   &types.Withdrawals{},
			BlobGasUsed:   new(eth.Uint64Quantity),
			ExcessBlobGas: new(eth.Uint64Quantity),
		},
	}
	env.ExecutionPayload.BlockHash, _ = env.CheckBlockHash()
	return env
}

func attributesForPayload(env *eth.ExecutionPayloadEnvelope) *eth.PayloadAttributes {
	p := env.ExecutionPayload
	return &eth.PayloadAttributes{
		Timestamp:             p.Timestamp,
		PrevRandao:            p.PrevRandao,
		SuggestedFeeRecipient: p.FeeRecipient,
		Withdrawals:           &types.Withdrawals{},
		ParentBeaconBlockRoot: env.ParentBeaconBlockRoot,
		Transactions:          p.Transactions,
		NoTxPool:              true,
		GasLimit:              &p.GasLimit,
	}
}

func TestSyncTester_BuildAndInsertDepositBlock(t *testing.T) {
	el := NewMockELReader(eth.ChainIDFromUInt64(1))
	head := makeDepositPayload(t, 10, common.HexToHash("0x0a"))
	next := makeDepositPayload(t, 11, head.ExecutionPayload.BlockHash)
	el.AddPayload(head)
	el.AddPayload(next)
	st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), el)
	sess := eth.NewSyncTesterSession(uuid.New().String(), 10, 10, 10, false, nil)
	ctx := session.WithSyncTesterSession(context.Background(), sess)
	state := &eth.ForkchoiceState{HeadBlockHash: head.ExecutionPayload.BlockHash}

	res, err := st.ForkchoiceUpdatedV3(ctx, state, attributesForPayload(next))
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
	require.NotNil(t, res.PayloadID)

	built, err := st.GetPayloadV3(ctx, *res.PayloadID)
	require.NoError(t, err)
	// The read only EL's block is served as the built payload.
	require.NoError(t, built.ExecutionPayload.CheckEqual(makeDepositPayload(t, 11, head.ExecutionPayload.BlockHash).ExecutionPayload))
	require.Equal(t, next.ParentBeaconBlockRoot, built.ParentBeaconBlockRoot)

	payload := *built.ExecutionPayload
	status, err := st.NewPayloadV3(ctx, &payload, []common.Hash{}, built.ParentBeaconBlockRoot)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, status.Status)
	require.Equal(t, next.ExecutionPayload.BlockHash, *status.LatestValidHash)
	require.EqualValues(t, 11, sess.Validated)
}

func requireInvalidAttributes(t *testing.T, err error, reason string) {
	var apiErr *engine.EngineAPIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, engine.InvalidPayloadAttributes.ErrorCode(), apiErr.ErrorCode())
	require.Contains(t, fmt.Sprint(apiErr.ErrorData()), reason)
}

func TestSyncTester_ForkchoiceUpdatedRejectsMismatchingAttributes(t *testing.T) {
	el := NewMockELReader(eth.ChainIDFromUInt64(1))
	head := makeDepositPayload(t, 10, common.HexToHash("0x0a"))
	next := makeDepositPayload(t, 11, head.ExecutionPayload.BlockHash)
	el.AddPayload(head)
	el.AddPayload(next)
	st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), el)
	state := &eth.ForkchoiceState{HeadBlockHash: head.ExecutionPayload.BlockHash}
	newCtx := func() context.Context {
		return session.WithSyncTesterSession(context.Background(), eth.NewSyncTesterSession(uuid.New().String(), 10, 10, 10, false, nil))
	}

	t.Run("tx mismatch", func(t *testing.T) {
		attr := attributesForPayload(next)
		attr.Transactions = makeDepositPayload(t, 12, common.Hash{}).ExecutionPayload.Transactions
		res, err := st.ForkchoiceUpdatedV3(newCtx(), state, attr)
		requireInvalidAttributes(t, err, "tx mismatch")
		require.Equal(t, eth.ExecutionInvalid, res.PayloadStatus.Status)
	})
	t.Run("withdrawals mismatch", func(t *testing.T) {
		// A Canyon block has withdrawals but no beacon root, so only the withdrawals
		// check can tell it apart from Bedrock attributes.
		canyonNext := makeDepositPayload(t, 11, head.ExecutionPayload.BlockHash)
		canyonNext.ParentBeaconBlockRoot = nil
		canyonNext.ExecutionPayload.BlobGasUsed, canyonNext.ExecutionPayload.ExcessBlobGas = nil, nil
		canyonNext.ExecutionPayload.BlockHash, _ = canyonNext.CheckBlockHash()
		canyonEL := NewMockELReader(eth.ChainIDFromUInt64(1))
		canyonEL.AddPayload(head)
		canyonEL.AddPayload(canyonNext)
		st := initTestSyncTester(t, eth.ChainIDFromUInt64(1), canyonEL)
		attr := attributesForPayload(canyonNext)
		attr.Withdrawals = nil
		res, err := st.ForkchoiceUpdatedV1(newCtx(), state, attr)
		requireInvalidAttributes(t, err, "withdrawals mismatch")
		require.Equal(t, eth.ExecutionInvalid, res.PayloadStatus.Status)
	})
}
