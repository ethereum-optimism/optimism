package challenger

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/stretchr/testify/require"
)

type mockLogFilterClient struct{}

func (m mockLogFilterClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m mockLogFilterClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

func newSubscription(t *testing.T, client *mockLogFilterClient) (*Subscription, *mockLogFilterClient) {
	query := ethereum.FilterQuery{}
	log := testlog.Logger(t, log.LvlError)
	return NewSubscription(query, client, log), client
}

func FuzzSubscriptionId_Increment(f *testing.F) {
	maxUint64 := uint64(math.MaxUint64)
	f.Fuzz(func(t *testing.T, id uint64) {
		if id >= maxUint64 {
			t.Skip("skipping due to overflow")
		} else {
			subId := SubscriptionId(id)
			require.Equal(t, subId.Increment(), SubscriptionId(id+1))
		}
	})
}

func TestSubscription_Subscribe_NilClient_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected nil client to panic")
		}
	}()
	subscription, _ := newSubscription(t, nil)
	require.NoError(t, subscription.Subscribe())
}

func TestSubscription_Subscribe(t *testing.T) {
	subscription, _ := newSubscription(t, &mockLogFilterClient{})
	require.NoError(t, subscription.Subscribe())
	require.True(t, subscription.Started())
}

var ErrSubscriptionFailed = errors.New("failed to subscribe to logs")

type errLogFilterClient struct{}

func (m errLogFilterClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m errLogFilterClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, ErrSubscriptionFailed
}

func TestSubscription_Subscribe_SubscriptionErrors(t *testing.T) {
	query := ethereum.FilterQuery{}
	log := testlog.Logger(t, log.LvlError)
	subscription := Subscription{
		query:  query,
		client: errLogFilterClient{},
		log:    log,
	}
	require.EqualError(t, subscription.Subscribe(), ErrSubscriptionFailed.Error())
}
