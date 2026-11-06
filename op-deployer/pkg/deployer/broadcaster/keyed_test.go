package broadcaster

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmocks "github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestKeyedBroadcasterSendsSequentiallyWithGasEstimation(t *testing.T) {
	bcasts := []Broadcast{
		{
			Type:  BroadcastCreate,
			Input: []byte{0x60, 0x00},
			Value: (*hexutil.U256)(new(uint256.Int)),
		},
		{
			Type:  BroadcastCall,
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
			if bcasts[next].Type == BroadcastCreate {
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
	bcasts := []Broadcast{
		{Type: BroadcastCall, To: common.Address{'A'}, Value: (*hexutil.U256)(new(uint256.Int))},
		{Type: BroadcastCall, To: common.Address{'B'}, Value: (*hexutil.U256)(new(uint256.Int))},
		{Type: BroadcastCall, To: common.Address{'C'}, Value: (*hexutil.U256)(new(uint256.Int))},
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

func TestNewKeyedBroadcasterReceiptQueryInterval(t *testing.T) {
	const warning = "using non-default receipt query interval"
	tests := []struct {
		name     string
		interval time.Duration
		wantErr  bool
		wantWarn bool
	}{
		{name: "zero is the one second default", interval: 0},
		{name: "explicit default", interval: time.Second},
		{name: "lowest allowed", interval: 10 * time.Millisecond, wantWarn: true},
		{name: "test value", interval: 50 * time.Millisecond, wantWarn: true},
		{name: "highest allowed", interval: 12 * time.Second, wantWarn: true},
		{name: "below minimum", interval: 10*time.Millisecond - 1, wantErr: true},
		{name: "above maximum", interval: 12*time.Second + 1, wantErr: true},
		{name: "negative", interval: -time.Second, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantErr, CheckReceiptQueryInterval(tt.interval) != nil)

			lgr, logs := testlog.CaptureLogger(t, log.LevelWarn)
			_, err := NewKeyedBroadcaster(KeyedBroadcasterOpts{
				Logger:               lgr,
				ChainID:              big.NewInt(1),
				Client:               ethclient.NewClient(nil),
				Signer:               func(context.Context, common.Address, *types.Transaction) (*types.Transaction, error) { return nil, nil },
				ReceiptQueryInterval: tt.interval,
			})
			if tt.wantErr {
				require.ErrorContains(t, err, "receipt query interval must be 0 or between 10ms and 12s")
				return
			}
			require.NoError(t, err)
			warned := logs.FindLog(testlog.NewMessageContainsFilter(warning)) != nil
			require.Equal(t, tt.wantWarn, warned)
		})
	}
}
