package challenger

import (
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

// TestLogStore_NewLogStore tests the NewLogStore method on a [logStore].
func TestLogStore_NewLogStore(t *testing.T) {
	query := ethereum.FilterQuery{}
	logStore := NewLogStore(query)
	require.Equal(t, query, logStore.query)
	require.Equal(t, []types.Log{}, logStore.logList)
	require.Equal(t, make(map[common.Hash]types.Log), logStore.logMap)
	require.Equal(t, SubscriptionId(0), logStore.currentSubId)
	require.Equal(t, make(map[SubscriptionId]Subscription), logStore.subMap)
}

// TestLogStore_NewSubscription tests the newSubscription method on a [logStore].
func TestLogStore_NewSubscription(t *testing.T) {
	query := ethereum.FilterQuery{}
	logStore := NewLogStore(query)
	require.Equal(t, 0, len(logStore.subMap))
	require.Equal(t, 0, len(logStore.subEscapes))
	require.Equal(t, SubscriptionId(0), logStore.currentSubId)

	logStore.client = &mockLogFilterClient{}

	// Now create the new subscription.
	subscriptionId, err := logStore.newSubscription(query)
	require.NoError(t, err)
	require.Equal(t, SubscriptionId(1), subscriptionId)
	require.Equal(t, 1, len(logStore.subMap))
	require.Equal(t, 1, len(logStore.subEscapes))
}
