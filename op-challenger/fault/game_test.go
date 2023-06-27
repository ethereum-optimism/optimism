package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func createTestClaims() (Claim, Claim, Claim) {
	top := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
			Position: NewPosition(0, 0),
		},
		Parent: ClaimData{},
	}

	middle := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: NewPosition(1, 1),
		},
		Parent: top.ClaimData,
	}

	bottom := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(2, 2),
		},
		Parent: middle.ClaimData,
	}

	return top, middle, bottom
}

// TestGame_Put_RootAlreadyExists tests the [Game.Put] method using a [gameState]
// instance errors when the root claim already exists in state.
func TestGame_Put_RootAlreadyExists(t *testing.T) {
	// Setup the game state.
	top, _, _ := createTestClaims()
	g := NewGameState(top)

	// Try to put the root claim into the game state again.
	err := g.Put(top)
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_Put_AlreadyExists tests the [Game.Put] method using a [gameState]
// instance errors when the given claim already exists in state.
func TestGame_Put_AlreadyExists(t *testing.T) {
	// Setup the game state.
	top, middle, _ := createTestClaims()
	g := NewGameState(top)

	// Put the next claim into state.
	err := g.Put(middle)
	require.NoError(t, err)

	// Put the claim into the game state again.
	err = g.Put(middle)
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_Put_ParentsAndChildren tests the [Game.Put] method using a [gameState] instance.
func TestGame_Put_ParentsAndChildren(t *testing.T) {
	// Setup the game state.
	top, middle, bottom := createTestClaims()
	g := NewGameState(top)

	// We should not be able to get the parent of the root claim.
	parent, err := g.getParent(top)
	require.ErrorIs(t, err, ErrClaimNotFound)
	require.Equal(t, parent, Claim{})

	// Put the middle claim into the game state.
	// We should expect no parent to exist, yet.
	err = g.Put(middle)
	require.NoError(t, err)
	parent, err = g.getParent(middle)
	require.NoError(t, err)
	require.Equal(t, parent, top)

	// Put the bottom claim into the game state.
	// We should expect the parent to be the claim we just added.
	err = g.Put(bottom)
	require.NoError(t, err)
	parent, err = g.getParent(bottom)
	require.NoError(t, err)
	require.Equal(t, parent, middle)
}

// TestGame_ClaimPairs tests the [Game.ClaimPairs] method using a [gameState] instance.
func TestGame_ClaimPairs(t *testing.T) {
	// Setup the game state.
	top, middle, bottom := createTestClaims()
	g := NewGameState(top)

	// Add middle claim to the game state.
	err := g.Put(middle)
	require.NoError(t, err)

	// Add the bottom claim to the game state.
	err = g.Put(bottom)
	require.NoError(t, err)

	// Validate claim pairs.
	expected := []struct{ claim, parent Claim }{
		{middle, top},
		{bottom, middle},
	}
	pairs := g.ClaimPairs()
	require.ElementsMatch(t, expected, pairs)
}
