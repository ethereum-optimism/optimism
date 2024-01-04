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

func alphabetClaim(index *big.Int, claim *big.Int) common.Hash {
	return alphabetStateHash(BuildAlphabetPreimage(index, claim))
}

func TestAlphabetProvider_Step(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)

	ap := NewTraceProvider(startingL2BlockNumber, depth)

	// Start at the absolute prestate as the stateData
	claim := BuildAlphabetPreimage(big.NewInt(0), absolutePrestateInt)
	claim = ap.step(claim)
	startingTraceIndex := new(big.Int).Lsh(startingL2BlockNumber, 4)
	startingClaim := new(big.Int).Add(absolutePrestateInt, startingTraceIndex)
	require.Equal(t, BuildAlphabetPreimage(startingTraceIndex, startingClaim), claim)

	// Step again, which should increment both the claim and trace index by 1
	claim = ap.step(claim)
	nextTraceIndex := new(big.Int).Add(startingTraceIndex, big.NewInt(1))
	nextClaim := new(big.Int).Add(startingClaim, big.NewInt(1))
	require.Equal(t, BuildAlphabetPreimage(nextTraceIndex, nextClaim), claim)

	// Step again, which should increment both the claim and trace index by 1
	claim = ap.step(claim)
	nextTraceIndex = new(big.Int).Add(nextTraceIndex, big.NewInt(1))
	nextClaim = new(big.Int).Add(nextClaim, big.NewInt(1))
	require.Equal(t, BuildAlphabetPreimage(nextTraceIndex, nextClaim), claim)
}

// TestAlphabetProvider_Get_ClaimsByTraceIndex tests the [fault.AlphabetProvider] Get function.
func TestAlphabetProvider_Get_ClaimsByTraceIndex(t *testing.T) {
	// Create a new alphabet provider.
	depth := types.Depth(3)
	startingL2BlockNumber := big.NewInt(1)
	sbn := new(big.Int).Lsh(startingL2BlockNumber, 4)
	startingTraceIndex := new(big.Int).Add(absolutePrestateInt, sbn)
	canonicalProvider := NewTraceProvider(startingL2BlockNumber, depth)

	// Build a list of traces.
	traces := []struct {
		traceIndex   types.Position
		expectedHash common.Hash
	}{
		{
			types.NewPosition(depth, big.NewInt(7)),
			alphabetClaim(new(big.Int).Add(sbn, big.NewInt(6)), new(big.Int).Add(startingTraceIndex, big.NewInt(6))),
		},
		{
			types.NewPosition(depth, big.NewInt(3)),
			alphabetClaim(new(big.Int).Add(sbn, big.NewInt(2)), new(big.Int).Add(startingTraceIndex, big.NewInt(2))),
		},
		{
			types.NewPosition(depth, big.NewInt(5)),
			alphabetClaim(new(big.Int).Add(sbn, big.NewInt(4)), new(big.Int).Add(startingTraceIndex, big.NewInt(4))),
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
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	expected := BuildAlphabetPreimage(big.NewInt(0), absolutePrestateInt)
	pos := types.NewPosition(depth, big.NewInt(1))
	retrieved, proof, data, err := ap.GetStepData(context.Background(), pos)
	require.NoError(t, err)
	require.Equal(t, expected, retrieved)
	require.Empty(t, proof)
	key := preimage.LocalIndexKey(L2ClaimBlockNumberLocalIndex).PreimageKey()
	expectedLocalContextData := types.NewPreimageOracleData(key[:], startingL2BlockNumber.Bytes(), 0)
	require.Equal(t, expectedLocalContextData, data)
}

// TestGetPreimage_TooLargeIndex_Fails tests the GetPreimage
// function errors if the index is too large.
func TestGetStepData_TooLargeIndex_Fails(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	pos := types.NewPosition(depth, big.NewInt(5))
	_, _, _, err := ap.GetStepData(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Succeeds tests the Get function.
func TestGet_Succeeds(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	pos := types.NewPosition(depth, big.NewInt(0))
	claim, err := ap.Get(context.Background(), pos)
	require.NoError(t, err)
	expected := alphabetClaim(big.NewInt(0), absolutePrestateInt)
	require.Equal(t, expected, claim)
}

// TestGet_IndexTooLarge tests the Get function with an index
// greater than the number of indices: 2^depth - 1.
func TestGet_IndexTooLarge(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	pos := types.NewPosition(depth, big.NewInt(4))
	_, err := ap.Get(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

func TestGet_DepthTooLarge(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	pos := types.NewPosition(depth+1, big.NewInt(0))
	_, err := ap.Get(context.Background(), pos)
	require.ErrorIs(t, err, ErrIndexTooLarge)
}

// TestGet_Extends tests the Get function with an index that is larger
// than the trace, but smaller than the maximum depth.
func TestGet_Extends(t *testing.T) {
	depth := types.Depth(2)
	startingL2BlockNumber := big.NewInt(1)
	sbn := new(big.Int).Lsh(startingL2BlockNumber, 4)
	startingTraceIndex := new(big.Int).Add(absolutePrestateInt, sbn)
	ap := NewTraceProvider(startingL2BlockNumber, depth)
	pos := types.NewPosition(depth, big.NewInt(3))
	claim, err := ap.Get(context.Background(), pos)
	require.NoError(t, err)
	expected := alphabetClaim(new(big.Int).Add(sbn, big.NewInt(2)), new(big.Int).Add(startingTraceIndex, big.NewInt(2)))
	require.Equal(t, expected, claim)
}
