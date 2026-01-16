package bigs

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqual(t *testing.T) {
	require.True(t, Equal(big.NewInt(0), big.NewInt(0)))
	require.True(t, Equal(big.NewInt(1), big.NewInt(1)))
	require.True(t, Equal(big.NewInt(1900), big.NewInt(1900)))
	require.True(t, Equal(big.NewInt(-1), big.NewInt(-1)))
	require.True(t, Equal(big.NewInt(-1900), big.NewInt(-1900)))

	require.False(t, Equal(big.NewInt(0), big.NewInt(1)))
	require.False(t, Equal(big.NewInt(1), big.NewInt(0)))
	require.False(t, Equal(big.NewInt(1), big.NewInt(2)))
	require.False(t, Equal(big.NewInt(-1900), big.NewInt(1900)))
}

func TestIsZero(t *testing.T) {
	require.True(t, IsZero(big.NewInt(0)))
	require.False(t, IsZero(big.NewInt(1)))
	require.False(t, IsZero(big.NewInt(-1)))
}

func TestIsPositive(t *testing.T) {
	require.True(t, IsPositive(big.NewInt(1)))
	require.True(t, IsPositive(big.NewInt(2)))

	require.False(t, IsPositive(big.NewInt(0)))
	require.False(t, IsPositive(big.NewInt(-1)))
}

func TestIsNegative(t *testing.T) {
	require.True(t, IsNegative(big.NewInt(-1)))
	require.True(t, IsNegative(big.NewInt(-2)))

	require.False(t, IsNegative(big.NewInt(0)))
	require.False(t, IsNegative(big.NewInt(1)))
}

func TestUint64Strict(t *testing.T) {
	require.Equal(t, uint64(0), Uint64Strict(big.NewInt(0)))
	require.Equal(t, uint64(42), Uint64Strict(big.NewInt(42)))

	max := new(big.Int).SetUint64(math.MaxUint64)
	require.Equal(t, uint64(math.MaxUint64), Uint64Strict(max))

	require.Panics(t, func() {
		Uint64Strict(big.NewInt(-1))
	})
	require.Panics(t, func() {
		Uint64Strict(nil)
	})
	require.Panics(t, func() {
		tooLarge := new(big.Int).Add(max, big.NewInt(1))
		Uint64Strict(tooLarge)
	})
}
