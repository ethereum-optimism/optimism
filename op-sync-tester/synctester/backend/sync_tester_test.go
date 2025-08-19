package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var _ ReadOnlyELBackend = (*MockELReader)(nil)

type MockELReader struct {
	ChainID hexutil.Big

	BlocksByHash   map[common.Hash]*json.RawMessage
	BlocksByNumber map[rpc.BlockNumber]*json.RawMessage

	ReceiptsByHash   map[common.Hash][]*types.Receipt
	ReceiptsByNumber map[rpc.BlockNumber][]*types.Receipt

	Latest    *json.RawMessage
	Safe      *json.RawMessage
	Finalized *json.RawMessage
}

func NewMockELReader(chainID eth.ChainID) *MockELReader {
	return &MockELReader{
		ChainID:          hexutil.Big(*chainID.ToBig()),
		BlocksByHash:     make(map[common.Hash]*json.RawMessage),
		BlocksByNumber:   make(map[rpc.BlockNumber]*json.RawMessage),
		ReceiptsByHash:   make(map[common.Hash][]*types.Receipt),
		ReceiptsByNumber: make(map[rpc.BlockNumber][]*types.Receipt),
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

func (m *MockELReader) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}

func (m *MockELReader) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}

func (m *MockELReader) GetBlockReceipts(ctx context.Context, bnh rpc.BlockNumberOrHash) ([]*types.Receipt, error) {
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
	dummySession := &Session{SessionID: uuid.New().String()}
	tests := []struct {
		name            string
		cfgID           eth.ChainID
		elID            eth.ChainID
		session         *Session
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
				ctx = WithSession(ctx, tc.session)
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
		session         *Session
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
			session:         &Session{SessionID: uuid.New().String(), CurrentState: FCUState{Latest: 100}},
			wantErrContains: "not found",
		},
		{
			name:          "happy path",
			sessionLatest: 100,
			rawNumber:     99,
			session:       &Session{SessionID: uuid.New().String(), CurrentState: FCUState{Latest: 100}},
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
				ctx = WithSession(ctx, tc.session)
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
			require.EqualValues(t, tc.rawNumber, header.Number.ToInt().Uint64())
		})
	}
}

func TestSyncTester_GetBlockByNumber(t *testing.T) {
	type testCase struct {
		name            string
		session         *Session
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
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
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
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
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
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
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
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
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
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			inNumber:        rpc.PendingBlockNumber,
			wantErrContains: "not found",
		},
		{
			name: "earliest label returns not found",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			inNumber:        rpc.EarliestBlockNumber,
			wantErrContains: "not found",
		},
		{
			name: "numeric greater than latest returns not found",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
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
				ctx = WithSession(ctx, tc.session)
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
			require.EqualValues(t, tc.wantNum, header.Number.ToInt().Uint64())
		})
	}
}

func TestSyncTester_GetBlockReceipts(t *testing.T) {
	makeReceipts := func(n uint64) []*types.Receipt {
		r := new(types.Receipt)
		r.BlockNumber = new(big.Int).SetUint64(n)
		return []*types.Receipt{r}
	}
	type testCase struct {
		name            string
		session         *Session
		arg             rpc.BlockNumberOrHash
		seedFn          func(el *MockELReader, s *Session)
		wantFirstBN     uint64
		wantErrContains string
	}
	hashGood := common.HexToHash("0xabc1")
	hashTooNew := common.HexToHash("0xabc2")
	tests := []testCase{
		{
			name:            "no session",
			session:         nil,
			arg:             rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
			wantErrContains: "no session",
		},
		{
			name: "happy: via hash, blockNumber less than latest",
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashGood, false),
			seedFn: func(el *MockELReader, s *Session) {
				el.ReceiptsByHash[hashGood] = makeReceipts(s.CurrentState.Latest - 1)
			},
			wantFirstBN: 99,
		},
		{
			name: "bad: via hash, blockNumber >= latest returns not found",
			session: &Session{
				SessionID: uuid.New().String(),
				CurrentState: FCUState{
					Latest:    100,
					Safe:      95,
					Finalized: 90,
				},
			},
			arg: rpc.BlockNumberOrHashWithHash(hashTooNew, false),
			seedFn: func(el *MockELReader, s *Session) {
				// strictly greater than Latest so the post-check triggers NotFound
				el.ReceiptsByHash[hashTooNew] = makeReceipts(s.CurrentState.Latest + 1)
			},
			wantErrContains: "not found",
		},
		{
			name: "happy: label latest returns latest",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 95, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
			seedFn: func(el *MockELReader, s *Session) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Latest)] = makeReceipts(s.CurrentState.Latest)
			},
			wantFirstBN: 100,
		},
		{
			name: "happy: label safe returns safe",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 90},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.SafeBlockNumber),
			seedFn: func(el *MockELReader, s *Session) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Safe)] = makeReceipts(s.CurrentState.Safe)
			},
			wantFirstBN: 97,
		},
		{
			name: "happy: label finalized returns finalized",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.FinalizedBlockNumber),
			seedFn: func(el *MockELReader, s *Session) {
				el.ReceiptsByNumber[rpc.BlockNumber(s.CurrentState.Finalized)] = makeReceipts(s.CurrentState.Finalized)
			},
			wantFirstBN: 92,
		},
		{
			name: "happy: numeric less than latest",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
			},
			arg: rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(99)),
			seedFn: func(el *MockELReader, _ *Session) {
				el.ReceiptsByNumber[rpc.BlockNumber(99)] = makeReceipts(99)
			},
			wantFirstBN: 99,
		},
		{
			name: "bad: numeric greater than latest returns not found",
			session: &Session{
				SessionID:    uuid.New().String(),
				CurrentState: FCUState{Latest: 100, Safe: 97, Finalized: 92},
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
				ctx = WithSession(ctx, tc.session)
			}
			recs, err := st.GetBlockReceipts(ctx, tc.arg)
			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, recs)
			require.GreaterOrEqual(t, len(recs), 1)
			require.EqualValues(t, tc.wantFirstBN, recs[0].BlockNumber.Uint64())
		})
	}
}
