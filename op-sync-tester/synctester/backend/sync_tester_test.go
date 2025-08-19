package backend

import (
	"context"
	"encoding/json"
	"fmt"
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

	BlocksByHashData map[common.Hash]*json.RawMessage
	BlocksByNumber   map[rpc.BlockNumber]*json.RawMessage
	Latest           *json.RawMessage
	Safe             *json.RawMessage
	Finalized        *json.RawMessage
}

func NewMockELReader(chainID eth.ChainID) *MockELReader {
	return &MockELReader{
		ChainID:          hexutil.Big(*chainID.ToBig()),
		BlocksByHashData: make(map[common.Hash]*json.RawMessage),
		BlocksByNumber:   make(map[rpc.BlockNumber]*json.RawMessage),
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
	raw, ok := m.BlocksByHashData[hash]
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
	return nil, nil
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
			name:            "block number larger than session latest",
			sessionLatest:   100,
			rawNumber:       101, // larger than Latest
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
			el.BlocksByHashData[hash] = block
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
			name: "happy path: label latest returns CurrentState.Latest",
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
			name: "happy path: label safe returns CurrentState.Safe",
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
			name: "happy path: label finalized returns CurrentState.Finalized",
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
