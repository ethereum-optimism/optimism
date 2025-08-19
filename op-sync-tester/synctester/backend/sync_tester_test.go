package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
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
}

func NewMockELReader(chainID eth.ChainID) *MockELReader {
	return &MockELReader{ChainID: hexutil.Big(*chainID.ToBig())}
}

func (m *MockELReader) ChainId(ctx context.Context) (hexutil.Big, error) {
	return m.ChainID, nil
}

func (m *MockELReader) GetBlockByNumberJSON(ctx context.Context, number rpc.BlockNumber, fullTx bool) (json.RawMessage, error) {
	return nil, nil
}

func (m *MockELReader) GetBlockByHashJSON(ctx context.Context, hash common.Hash, fullTx bool) (json.RawMessage, error) {
	return nil, nil
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
			session:         nil,
			wantErrContains: "session",
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
