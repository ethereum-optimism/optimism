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

	BlockByHashData map[common.Hash]*json.RawMessage
}

func NewMockELReader(chainID eth.ChainID) *MockELReader {
	return &MockELReader{
		ChainID:         hexutil.Big(*chainID.ToBig()),
		BlockByHashData: make(map[common.Hash]*json.RawMessage),
	}
}

func (m *MockELReader) ChainId(ctx context.Context) (hexutil.Big, error) {
	return m.ChainID, nil
}

func (m *MockELReader) GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	return nil, nil
}

func (m *MockELReader) GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	raw, ok := m.BlockByHashData[hash]
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

func TestSyncTester_GetBlockByHash(t *testing.T) {
	makeBlockRaw := func(num uint64) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{"number":"0x%x"}`, num))
	}

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
			el.BlockByHashData[hash] = &block
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
