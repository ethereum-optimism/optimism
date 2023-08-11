package alphabet

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func alphabetClaim(index uint64, letter string) common.Hash {
	return crypto.Keccak256Hash(BuildAlphabetPreimage(index, letter))
}

// TestAlphabetProvider_Get_ClaimsByTraceIndex tests the [fault.AlphabetProvider] Get function.
func TestAlphabetProvider_Get_ClaimsByTraceIndex(t *testing.T) {
	// Create a new alphabet provider.
	canonicalProvider := NewTraceProvider("abcdefgh", uint64(3))

	// Build a list of traces.
	traces := []struct {
		traceIndex   uint64
		expectedHash common.Hash
	}{
		{
			7,
			alphabetClaim(7, "h"),
		},
		{
			3,
			alphabetClaim(3, "d"),
		},
		{
			5,
			alphabetClaim(5, "f"),
		},
	}

	// Execute each trace and check the alphabet provider returns the expected hash.
	for _, trace := range traces {
		expectedHash, err := canonicalProvider.Get(context.Background(), trace.traceIndex)
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

// TestGetPreimage_Succeeds tests the GetPreimage function
// returns the correct pre-image for a index.
func TestGetPreimage_Succeeds(t *testing.T) {
	ap := NewTraceProvider("abc", 2)
	expected := BuildAlphabetPreimage(0, "a'")
	retrieved, proof, err := ap.GetPreimage(context.Background(), uint64(0))
	require.NoError(t, err)
	require.Equal(t, expected, retrieved)
	require.Empty(t, proof)
}

// TestGetPreimage_TooLargeIndex_Fails tests the GetPreimage
// function errors if the index is too large.
func TestGetPreimage_TooLargeIndex_Fails(t *testing.T) {
	ap := NewTraceProvider("abc", 2)
	_, _, err := ap.GetPreimage(context.Background(), 4)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Succeeds tests the Get function.
func TestGet_Succeeds(t *testing.T) {
	ap := NewTraceProvider("abc", 2)
	claim, err := ap.Get(context.Background(), 0)
	require.NoError(t, err)
	expected := alphabetClaim(0, "a")
	require.Equal(t, expected, claim)
}

// TestGet_IndexTooLarge tests the Get function with an index
// greater than the number of indices: 2^depth - 1.
func TestGet_IndexTooLarge(t *testing.T) {
	ap := NewTraceProvider("abc", 2)
	_, err := ap.Get(context.Background(), 4)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Extends tests the Get function with an index that is larger
// than the trace, but smaller than the maximum depth.
func TestGet_Extends(t *testing.T) {
	ap := NewTraceProvider("abc", 2)
	claim, err := ap.Get(context.Background(), 3)
	require.NoError(t, err)
	expected := alphabetClaim(2, "c")
	require.Equal(t, expected, claim)
}
