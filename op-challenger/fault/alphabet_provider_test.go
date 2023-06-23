package fault

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestAlphabetProvider_Get_ClaimsByTraceIndex tests the [fault.AlphabetProvider] Get function.
func TestAlphabetProvider_Get_ClaimsByTraceIndex(t *testing.T) {
	// Create a new alphabet provider.
	canonicalProvider := NewAlphabetProvider("abcdefgh", uint64(3))

	// Build a list of traces.
	traces := []struct {
		traceIndex   uint64
		expectedHash common.Hash
	}{
		{
			7,
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		},
		{
			3,
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
		},
		{
			5,
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000566"),
		},
	}

	// Execute each trace and check the alphabet provider returns the expected hash.
	for _, trace := range traces {
		expectedHash, err := canonicalProvider.Get(trace.traceIndex)
		require.NoError(t, err)
		require.Equal(t, trace.expectedHash, expectedHash)
	}
}

// FuzzIndexToBytes tests the IndexToBytes function.
func FuzzIndexToBytes(f *testing.F) {
	f.Fuzz(func(t *testing.T, index uint64) {
		translated := IndexToBytes(index)
		original := new(big.Int)
		original.SetBytes(translated)
		require.Equal(t, original.Uint64(), index)
	})
}

// TestComputeAlphabetClaim tests the ComputeAlphabetClaim function.
func TestComputeAlphabetClaim(t *testing.T) {
	ap := NewAlphabetProvider("abc", 2)
	claim := ap.ComputeAlphabetClaim(0)
	concatenated := append(IndexToBytes(0), []byte("a")...)
	expected := common.BytesToHash(concatenated)
	require.Equal(t, expected, claim)
}

// TestGet tests the Get function.
func TestGet(t *testing.T) {
	ap := NewAlphabetProvider("abc", 2)
	claim, err := ap.Get(0)
	require.NoError(t, err)
	concatenated := append(IndexToBytes(0), []byte("a")...)
	expected := common.BytesToHash(concatenated)
	require.Equal(t, expected, claim)
}

// TestGet_IndexTooLarge tests the Get function with an index
// greater than the number of indices: 2^depth - 1.
func TestGet_IndexTooLarge(t *testing.T) {
	ap := NewAlphabetProvider("abc", 2)
	_, err := ap.Get(4)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Extends tests the Get function with an index that is larger
// than the trace, but smaller than the maximum depth.
func TestGet_Extends(t *testing.T) {
	ap := NewAlphabetProvider("abc", 2)
	claim, err := ap.Get(3)
	require.NoError(t, err)
	concatenated := append(IndexToBytes(2), []byte("c")...)
	expected := common.BytesToHash(concatenated)
	require.Equal(t, expected, claim)
}
