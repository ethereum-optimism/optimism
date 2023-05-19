package challenger

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

// mockLogFilterClient implements the [ethereum.LogFilter] interface for testing.
type mockLogFilterClient struct{}

func (m mockLogFilterClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m mockLogFilterClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

// FuzzSubscriptionId_Increment tests the Increment method on a [SubscriptionId].
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

// TestSubscription_Subscribe_MissingClient tests the Subscribe
// method on a [Subscription] fails when the client is missing.
func TestSubscription_Subscribe_MissingClient(t *testing.T) {
	query := ethereum.FilterQuery{}
	subscription := Subscription{
		query: query,
	}
	err := subscription.Subscribe()
	require.EqualError(t, err, ErrMissingClient.Error())
}

// TestSubscription_Subscribe tests the Subscribe method on a [Subscription].
func TestSubscription_Subscribe(t *testing.T) {
	query := ethereum.FilterQuery{}
	subscription := Subscription{
		query:  query,
		client: mockLogFilterClient{},
	}
	require.Nil(t, subscription.logs)
	err := subscription.Subscribe()
	require.NoError(t, err)
	require.NotNil(t, subscription.logs)
}

var ErrSubscriptionFailed = errors.New("failed to subscribe to logs")

type errLogFilterClient struct{}

func (m errLogFilterClient) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	panic("this should not be called by the Subscription.Subscribe method")
}

func (m errLogFilterClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, ErrSubscriptionFailed
}

// TestSubscription_Subscribe_Errors tests the Subscribe
// method on a [Subscription] errors if the LogFilter client
// returns an error.
func TestSubscription_Subscribe_Errors(t *testing.T) {
	query := ethereum.FilterQuery{}
	subscription := Subscription{
		query:  query,
		client: errLogFilterClient{},
	}
	err := subscription.Subscribe()
	require.EqualError(t, err, ErrSubscriptionFailed.Error())
}
