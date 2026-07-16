package proofs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestAwaitMinVerifiedTimestampRetriesAndReturnsSuccessfulResponse(t *testing.T) {
	const timestamp = uint64(1234)
	expected := eth.SuperRootAtTimestampResponse{
		CurrentSafeTimestamp: 5678,
		Data:                 &eth.SuperRootResponseData{},
	}
	calls := 0
	query := func(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error) {
		calls++
		if calls == 1 {
			return eth.SuperRootAtTimestampResponse{}, errors.New("transient query failure")
		}
		return expected, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	actual, err := awaitMinVerifiedTimestamp(ctx, timestamp, time.Millisecond, query)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, 2, calls)
}
