package broadcaster

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmocks "github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestKeyedBroadcasterSendsSequentiallyWithGasEstimation(t *testing.T) {
	bcasts := []script.Broadcast{
		{
			Type:  script.BroadcastCreate,
			Input: []byte{0x60, 0x00},
			Value: (*hexutil.U256)(new(uint256.Int)),
		},
		{
			Type:  script.BroadcastCall,
			To:    common.Address{'T'},
			Input: []byte{0x12, 0x34},
			Value: (*hexutil.U256)(new(uint256.Int)),
		},
	}

	mgr := txmocks.NewTxManager(t)
	next := 0
	mgr.On("Send", mock.Anything, mock.Anything).Return(
		func(_ context.Context, candidate txmgr.TxCandidate) (*types.Receipt, error) {
			require.Less(t, next, len(bcasts))
			require.Zero(t, candidate.GasLimit)
			require.Equal(t, []byte(bcasts[next].Input), candidate.TxData)
			if bcasts[next].Type == script.BroadcastCreate {
				require.Nil(t, candidate.To)
			} else {
				require.Equal(t, bcasts[next].To, *candidate.To)
			}

			next++
			return &types.Receipt{
				Status: types.ReceiptStatusSuccessful,
				TxHash: common.Hash{byte(next)},
			}, nil
		},
	)

	broadcaster := &KeyedBroadcaster{
		lgr:    testlog.Logger(t, log.LevelError),
		mgr:    mgr,
		bcasts: bcasts,
	}
	results, err := broadcaster.Broadcast(context.Background())
	require.NoError(t, err)
	require.Len(t, results, len(bcasts))
	require.Equal(t, len(bcasts), next)
	for i, result := range results {
		require.Equal(t, bcasts[i], result.Broadcast)
		require.Equal(t, common.Hash{byte(i + 1)}, result.TxHash)
		require.NoError(t, result.Err)
	}
}

func TestKeyedBroadcasterStopsAfterFailure(t *testing.T) {
	bcasts := []script.Broadcast{
		{Type: script.BroadcastCall, To: common.Address{'A'}, Value: (*hexutil.U256)(new(uint256.Int))},
		{Type: script.BroadcastCall, To: common.Address{'B'}, Value: (*hexutil.U256)(new(uint256.Int))},
		{Type: script.BroadcastCall, To: common.Address{'C'}, Value: (*hexutil.U256)(new(uint256.Int))},
	}
	sendErr := errors.New("send failed")

	mgr := txmocks.NewTxManager(t)
	calls := 0
	mgr.On("Send", mock.Anything, mock.Anything).Return(
		func(_ context.Context, _ txmgr.TxCandidate) (*types.Receipt, error) {
			calls++
			if calls == 2 {
				return nil, sendErr
			}
			return &types.Receipt{
				Status: types.ReceiptStatusSuccessful,
				TxHash: common.Hash{byte(calls)},
			}, nil
		},
	)

	broadcaster := &KeyedBroadcaster{
		lgr:    testlog.Logger(t, log.LevelError),
		mgr:    mgr,
		bcasts: bcasts,
	}
	results, err := broadcaster.Broadcast(context.Background())
	require.ErrorIs(t, err, sendErr)
	require.Len(t, results, 2)
	require.Equal(t, 2, calls)
	require.ErrorIs(t, results[1].Err, sendErr)
}
