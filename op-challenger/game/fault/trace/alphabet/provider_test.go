package alphabet

import (
	"context"
	"math/big"
	"testing"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func alphabetClaim(index *big.Int, letter string) common.Hash {
	return alphabetStateHash(BuildAlphabetPreimage(index, letter))
}

// TestAlphabetProvider_Get_ClaimsByTraceIndex tests the [fault.AlphabetProvider] Get function.
func TestAlphabetProvider_Get_ClaimsByTraceIndex(t *testing.T) {
	// Create a new alphabet provider.
	depth := 3
	canonicalProvider := NewTraceProvider("abcdefgh", uint64(depth))

	// Build a list of traces.
	traces := []struct {
		traceIndex   types.Position
		expectedHash common.Hash
	}{
		{
			types.NewPosition(depth, big.NewInt(7)),
			alphabetClaim(big.NewInt(7), "h"),
		},
		{
			types.NewPosition(depth, big.NewInt(3)),
			alphabetClaim(big.NewInt(3), "d"),
		},
		{
			types.NewPosition(depth, big.NewInt(5)),
			alphabetClaim(big.NewInt(5), "f"),
		},
	}

	// Execute each trace and check the alphabet provider returns the expected hash.
	for _, trace := range traces {
		expectedHash, err := canonicalProvider.Get(context.Background(), trace.traceIndex)
		require.NoError(t, err)
		require.Equal(t, trace.expectedHash, expectedHash)
	}
}

// TestGetPreimage_Succeeds tests the GetPreimage function
// returns the correct pre-image for a index.
func TestGetStepData_Succeeds(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	expected := BuildAlphabetPreimage(big.NewInt(0), "a")
	pos := types.NewPosition(depth, big.NewInt(1))
	retrieved, proof, data, err := ap.GetStepData(context.Background(), pos)
	require.NoError(t, err)
	require.Equal(t, expected, retrieved)
	require.Empty(t, proof)
	key := preimage.LocalIndexKey(L2ClaimBlockNumberLocalIndex).PreimageKey()
	expectedLocalContextData := types.NewPreimageOracleData(key[:], nil, 0)
	require.Equal(t, expectedLocalContextData, data)
}

// TestGetPreimage_TooLargeIndex_Fails tests the GetPreimage
// function errors if the index is too large.
func TestGetStepData_TooLargeIndex_Fails(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	pos := types.NewPosition(depth, big.NewInt(5))
	_, _, _, err := ap.GetStepData(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Succeeds tests the Get function.
func TestGet_Succeeds(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	pos := types.NewPosition(depth, big.NewInt(0))
	claim, err := ap.Get(context.Background(), pos)
	require.NoError(t, err)
	expected := alphabetClaim(big.NewInt(0), "a")
	require.Equal(t, expected, claim)
}

// TestGet_IndexTooLarge tests the Get function with an index
// greater than the number of indices: 2^depth - 1.
func TestGet_IndexTooLarge(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	pos := types.NewPosition(depth, big.NewInt(4))
	_, err := ap.Get(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

func TestGet_DepthTooLarge(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	pos := types.NewPosition(depth+1, big.NewInt(0))
	_, err := ap.Get(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Extends tests the Get function with an index that is larger
// than the trace, but smaller than the maximum depth.
func TestGet_Extends(t *testing.T) {
	depth := 2
	ap := NewTraceProvider("abc", uint64(depth))
	pos := types.NewPosition(depth, big.NewInt(3))
	claim, err := ap.Get(context.Background(), pos)
	require.NoError(t, err)
	expected := alphabetClaim(big.NewInt(2), "c")
	require.Equal(t, expected, claim)
}
