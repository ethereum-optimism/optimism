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

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/stretchr/testify/require"
)

type mockLogStoreClient struct {
	sub      mockSubscription
	logs     chan<- types.Log
	subcount int
}

func newMockLogStoreClient() *mockLogStoreClient {
	return &mockLogStoreClient{
		sub: mockSubscription{
			errorChan: make(chan error),
		},
	}
}

func (m *mockLogStoreClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m *mockLogStoreClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, logs chan<- types.Log) (ethereum.Subscription, error) {
	m.subcount = m.subcount + 1
	m.logs = logs
	return m.sub, nil
}

var (
	ErrTestError = errors.New("test error")
)

// errLogStoreClient implements the [ethereum.LogFilter] interface for testing.
type errLogStoreClient struct{}

func (m errLogStoreClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m errLogStoreClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, ErrTestError
}

type mockSubscription struct {
	errorChan chan error
}

func (m mockSubscription) Err() <-chan error {
	return m.errorChan
}

func (m mockSubscription) Unsubscribe() {}

func newLogStore(t *testing.T) (*logStore, *mockLogStoreClient) {
	query := ethereum.FilterQuery{}
	client := newMockLogStoreClient()
	log := testlog.Logger(t, log.LvlError)
	return NewLogStore(query, client, log), client
}

func newErrorLogStore(t *testing.T, client *errLogStoreClient) (*logStore, *errLogStoreClient) {
	query := ethereum.FilterQuery{}
	log := testlog.Logger(t, log.LvlError)
	return NewLogStore(query, client, log), client
}

func TestLogStore_NewLogStore_NotSubscribed(t *testing.T) {
	logStore, _ := newLogStore(t)
	require.False(t, logStore.Subscribed())
}

func TestLogStore_NewLogStore_EmptyLogs(t *testing.T) {
	logStore, _ := newLogStore(t)
	require.Empty(t, logStore.GetLogs())
	require.Empty(t, logStore.GetLogByBlockHash(common.Hash{}))
}

func TestLogStore_Subscribe_EstablishesSubscription(t *testing.T) {
	logStore, client := newLogStore(t)
	defer logStore.Quit()
	require.Equal(t, 0, client.subcount)
	require.False(t, logStore.Subscribed())
	require.NoError(t, logStore.Subscribe(context.Background()))
	require.True(t, logStore.Subscribed())
	require.Equal(t, 1, client.subcount)
}

func TestLogStore_Subscribe_ReceivesLogs(t *testing.T) {
	logStore, client := newLogStore(t)
	defer logStore.Quit()
	require.NoError(t, logStore.Subscribe(context.Background()))

	mockLog := types.Log{
		BlockHash: common.HexToHash("0x1"),
	}
	client.logs <- mockLog

	timeout, tCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer tCancel()
	err := e2eutils.WaitFor(timeout, 500*time.Millisecond, func() (bool, error) {
		result := logStore.GetLogByBlockHash(mockLog.BlockHash)
		return result[0].BlockHash == mockLog.BlockHash, nil
	})
	require.NoError(t, err)
}

func TestLogStore_Subscribe_SubscriptionErrors(t *testing.T) {
	logStore, client := newLogStore(t)
	defer logStore.Quit()
	require.NoError(t, logStore.Subscribe(context.Background()))

	client.sub.errorChan <- ErrTestError

	timeout, tCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer tCancel()
	err := e2eutils.WaitFor(timeout, 500*time.Millisecond, func() (bool, error) {
		subcount := client.subcount == 2
		started := logStore.subscription.Started()
		return subcount && started, nil
	})
	require.NoError(t, err)
}

func TestLogStore_Subscribe_NoClient_Panics(t *testing.T) {
	require.Panics(t, func() {
		logStore, _ := newErrorLogStore(t, nil)
		_ = logStore.Subscribe(context.Background())
	})
}

func TestLogStore_Subscribe_ErrorSubscribing(t *testing.T) {
	logStore, _ := newErrorLogStore(t, &errLogStoreClient{})
	require.False(t, logStore.Subscribed())
	require.EqualError(t, logStore.Subscribe(context.Background()), ErrTestError.Error())
}

func TestLogStore_Quit_ResetsSubscription(t *testing.T) {
	logStore, _ := newLogStore(t)
	require.False(t, logStore.Subscribed())
	require.NoError(t, logStore.Subscribe(context.Background()))
	require.True(t, logStore.Subscribed())
	logStore.Quit()
	require.False(t, logStore.Subscribed())
}

func TestLogStore_Quit_NoSubscription_Panics(t *testing.T) {
	require.Panics(t, func() {
		logStore, _ := newErrorLogStore(t, nil)
		logStore.Quit()
	})
}
