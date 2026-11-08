package broadcaster

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmocks "github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
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
		{name: "lowest allowed", interval: 50 * time.Millisecond, wantWarn: true},
		{name: "faster than the default", interval: time.Second - 1, wantWarn: true},
		{name: "below minimum", interval: 50*time.Millisecond - 1, wantErr: true},
		{name: "above maximum", interval: time.Second + 1, wantErr: true},
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
				require.ErrorContains(t, err, "receipt query interval must be 0 or between 50ms and 1s")
				return
			}
			require.NoError(t, err)
			warned := logs.FindLog(testlog.NewMessageContainsFilter(warning)) != nil
			require.Equal(t, tt.wantWarn, warned)
		})
	}
}

// The receipt appears on the tenth poll. At the one-second default the tx manager's ticker can't get
// there before ten seconds on any machine, and at 50ms a slow host still has plenty of slack.
func TestKeyedBroadcasterPollsAtConfiguredInterval(t *testing.T) {
	const receiptOnPoll = 10
	require.Less(t, timeUntilPoll(t, 50*time.Millisecond, receiptOnPoll), receiptOnPoll*time.Second)
}

// Real deployments set no interval, so faster polling here would apply to every one of them.
func TestKeyedBroadcasterPollsEverySecondByDefault(t *testing.T) {
	for _, interval := range []time.Duration{0, time.Second} {
		t.Run(interval.String(), func(t *testing.T) {
			require.GreaterOrEqual(t, timeUntilPoll(t, interval, 1), time.Second)
		})
	}
}

// timeUntilPoll broadcasts a transaction whose receipt appears on the given poll, and returns how long that took.
func timeUntilPoll(t *testing.T, interval time.Duration, receiptOnPoll int32) time.Duration {
	t.Helper()
	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		var result any
		switch req.Method {
		case "eth_getBlockByNumber":
			result = &types.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1), Difficulty: new(big.Int)}
		case "eth_maxPriorityFeePerGas", "eth_blobBaseFee":
			result = "0x1"
		case "eth_estimateGas":
			result = "0x5208"
		case "eth_getTransactionCount":
			result = "0x0"
		case "eth_sendRawTransaction":
			result = common.Hash{'T'}
		case "eth_getTransactionReceipt":
			if polls.Add(1) >= receiptOnPoll {
				result = &types.Receipt{
					Status:      types.ReceiptStatusSuccessful,
					Logs:        []*types.Log{},
					BlockNumber: big.NewInt(1),
					BlockHash:   common.Hash{'B'},
				}
			}
		}
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}))
	}))
	t.Cleanup(srv.Close)
	client, err := ethclient.Dial(srv.URL)
	require.NoError(t, err)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)
	bcaster, err := NewKeyedBroadcaster(KeyedBroadcasterOpts{
		Logger:               testlog.Logger(t, log.LevelError),
		ChainID:              big.NewInt(1),
		Client:               client,
		Signer:               opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(key, big.NewInt(1))),
		From:                 from,
		ReceiptQueryInterval: interval,
	})
	require.NoError(t, err)
	bcaster.Hook(Broadcast{Type: BroadcastCall, From: from, To: common.Address{'T'}, Value: (*hexutil.U256)(new(uint256.Int))})

	start := time.Now()
	_, err = bcaster.Broadcast(t.Context())
	require.NoError(t, err)
	elapsed := time.Since(start)
	require.EqualValues(t, receiptOnPoll, polls.Load())
	return elapsed
}
