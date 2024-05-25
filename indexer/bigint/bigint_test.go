package bigint

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClamp(t *testing.T) {
	start := big.NewInt(1)
	end := big.NewInt(10)

	// When the (start, end) bounds are within range
	// the same end pointer should be returned

	// larger range
	result := Clamp(start, end, 20)
	require.True(t, end == result)

	// exact range
	result = Clamp(start, end, 10)
	require.True(t, end == result)

	// smaller range
	result = Clamp(start, end, 5)
	require.False(t, end == result)
	require.Equal(t, uint64(5), result.Uint64())
}
