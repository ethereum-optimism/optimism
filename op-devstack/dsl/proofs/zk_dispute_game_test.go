package proofs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAwaitClaimDataRetriesReadErrors(t *testing.T) {
	expected := ZKClaimData{ParentIndex: 7}
	attempts := 0
	read := func(context.Context) (ZKClaimData, error) {
		attempts++
		if attempts < 3 {
			return ZKClaimData{}, errors.New("rpc unavailable")
		}
		return expected, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	actual, err := awaitClaimData(ctx, time.Millisecond, read)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, 3, attempts)
}

func TestAwaitClaimDataReturnsLastReadErrorWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := awaitClaimData(ctx, time.Millisecond, func(context.Context) (ZKClaimData, error) {
		return ZKClaimData{}, errors.New("rpc unavailable")
	})

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "rpc unavailable")
}
