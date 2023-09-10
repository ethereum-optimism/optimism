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

func TestGrouped(t *testing.T) {
	// base cases
	require.Nil(t, Grouped(One, Zero, 1))
	require.Nil(t, Grouped(Zero, One, 0))

	// Same Start/End
	group := Grouped(One, One, 1)
	require.Len(t, group, 1)
	require.Equal(t, One, group[0].Start)
	require.Equal(t, One, group[0].End)

	Three, Five := big.NewInt(3), big.NewInt(5)

	// One at a time
	group = Grouped(One, Three, 1)
	require.Equal(t, One, group[0].End)
	require.Equal(t, int64(1), group[0].End.Int64())
	require.Equal(t, int64(2), group[1].Start.Int64())
	require.Equal(t, int64(2), group[1].End.Int64())
	require.Equal(t, int64(3), group[2].Start.Int64())
	require.Equal(t, int64(3), group[2].End.Int64())

	// Split groups
	group = Grouped(One, Five, 3)
	require.Len(t, group, 2)
	require.Equal(t, One, group[0].Start)
	require.Equal(t, int64(3), group[0].End.Int64())
	require.Equal(t, int64(4), group[1].Start.Int64())
	require.Equal(t, Five, group[1].End)

	// Encompasses the range
	group = Grouped(One, Five, 5)
	require.Len(t, group, 1)
	require.Equal(t, One, group[0].Start, Zero)
	require.Equal(t, Five, group[0].End)

	// Size larger than the entire range
	group = Grouped(One, Five, 100)
	require.Len(t, group, 1)
	require.Equal(t, One, group[0].Start, Zero)
	require.Equal(t, Five, group[0].End)
}
