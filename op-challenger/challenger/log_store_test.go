package challenger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/stretchr/testify/require"
)

// mockLogStoreClient implements the [ethereum.LogFilter] interface for testing.
type mockLogStoreClient struct {
	sub mockSubscription
}

func newMockLogStoreClient() mockLogStoreClient {
	return mockLogStoreClient{
		sub: mockSubscription{
			errorChan: make(chan error),
		},
	}
}

func (m mockLogStoreClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m mockLogStoreClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return m.sub, nil
}

type mockSubscription struct {
	errorChan chan error
}

func (m mockSubscription) Err() <-chan error {
	return m.errorChan
}

func (m mockSubscription) Unsubscribe() {}

// TestLogStore_NewLogStore tests the NewLogStore method on a [logStore].
func TestLogStore_NewLogStore(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)
	require.Equal(t, query, logStore.query)
	require.Equal(t, []types.Log{}, logStore.logList)
	require.Equal(t, make(map[common.Hash][]types.Log), logStore.logMap)
	require.Equal(t, SubscriptionId(0), logStore.subscription.id)
	require.Equal(t, client, logStore.client)
}

// TestLogStore_Subscribe tests the [Subscribe] method on a [logStore].
func TestLogStore_Subscribe(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// The subscription should not be started by default.
	require.False(t, logStore.subscription.Started())

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)
	require.True(t, logStore.subscription.Started())
}

// TestLogStore_Subscribe_MissingClient tests the [Subscribe] method on a [logStore]
// fails when the client is missing.
func TestLogStore_Subscribe_MissingClient(t *testing.T) {
	query := ethereum.FilterQuery{}
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, nil, log)
	err := logStore.Subscribe()
	require.EqualError(t, err, ErrMissingClient.Error())
}

// TestLogStore_Quit tests the [Quit] method on a [logStore].
func TestLogStore_Quit(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// A nil subscription should not cause a panic.
	logStore.subscription = nil
	logStore.Quit()

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)

	// Quit the subscription
	logStore.Quit()
	require.Nil(t, logStore.subscription)
}

// TestLogStore_Resubsribe tests the [Resubscribe] method on a [logStore].
func TestLogStore_Resubscribe(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)

	// Resubscribe to the logStore.
	err = logStore.resubscribe()
	require.NoError(t, err)
}

// TestLogStore_Logs tests log methods on a [logStore].
func TestLogStore_Logs(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	require.Equal(t, []types.Log{}, logStore.GetLogs())
	require.Equal(t, []types.Log(nil), logStore.GetLogByBlockHash(common.HexToHash("0x1")))

	// Insert logs.
	logStore.insertLog(types.Log{
		BlockHash: common.HexToHash("0x1"),
	})
	logStore.insertLog(types.Log{
		BlockHash: common.HexToHash("0x1"),
	})

	// Validate log insertion.
	require.Equal(t, 2, len(logStore.GetLogs()))
	require.Equal(t, 2, len(logStore.GetLogByBlockHash(common.HexToHash("0x1"))))
}

// TestLogStore_DispatchLogs tests the [DispatchLogs] method on the [logStore].
func TestLogStore_DispatchLogs(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)

	// Dispatch logs on the logStore.
	go logStore.dispatchLogs()
	time.Sleep(1 * time.Second)

	// Send logs through the subscription.
	logStore.subscription.logs <- types.Log{
		BlockHash: common.HexToHash("0x1"),
	}
	time.Sleep(1 * time.Second)
	logStore.subscription.logs <- types.Log{
		BlockHash: common.HexToHash("0x1"),
	}
	time.Sleep(1 * time.Second)

	// Verify that the log was inserted correctly.
	require.Equal(t, 2, len(logStore.logList))
	require.Equal(t, 2, len(logStore.GetLogByBlockHash(common.HexToHash("0x1"))))
	require.Equal(t, 2, len(logStore.GetLogs()))

	// Quit the subscription.
	logStore.Quit()
}

// TestLogStore_DispatchLogs_SubscriptionError tests the [DispatchLogs] method on the [logStore]
// when the subscription returns an error.
func TestLogStore_DispatchLogs_SubscriptionError(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)

	// Dispatch logs on the logStore.
	go logStore.dispatchLogs()
	time.Sleep(1 * time.Second)

	// Send an error through the subscription.
	client.sub.errorChan <- errors.New("test error")
	time.Sleep(1 * time.Second)

	// Check that the subscription was restarted.
	require.True(t, logStore.subscription.Started())

	// Quit the subscription.
	logStore.Quit()
}

// TestLogStore_Start tests the [Start] method on the [logStore].
func TestLogStore_Start(t *testing.T) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	logStore := NewLogStore(query, client, log)

	// Subscribe to the logStore.
	err := logStore.Subscribe()
	require.NoError(t, err)

	// Start the logStore.
	logStore.Start()
	time.Sleep(1 * time.Second)

	// Quit the subscription.
	logStore.Quit()
}
