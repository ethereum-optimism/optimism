package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestGame_GetClaimParent tests the [Game.GetClaimParent] method using a [gameState] instance.
func TestGame_GetClaimParent(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create claims.
	child := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: NewPosition(1, 1),
	}
	parent := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: NewPosition(0, 0),
	}

	// Put the claims into the game state.
	_, err := g.Put(child)
	require.NoError(t, err)
	_, err = g.Put(parent)
	require.NoError(t, err)

	// Get the parent of the child.
	retrieved := g.GetClaimParent(child)
	require.Equal(t, parent, retrieved)

	// The parent should not have a parent.
	retrieved = g.GetClaimParent(parent)
	require.Equal(t, Claim{}, retrieved)
}

// TestGame_Put_AlreadyExists tests the [Game.Put] method using a [gameState] instance
// errors when a claim already exists.
func TestGame_Put_AlreadyExists(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create a new claim.
	claim := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: NewPosition(1, 1),
	}

	// Put the claim into the game state.
	_, err := g.Put(claim)
	require.NoError(t, err)

	// Put the claim into the game state again.
	_, err = g.Put(claim)
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_Put_ParentsAndChildren tests the [Game.Put] method using a [gameState] instance.
func TestGame_Put_ParentsAndChildren(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create a new claim.
	middle := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: NewPosition(1, 1),
	}

	// Put the middle claim into the game state.
	// We should expect no parent to exist, yet.
	parent, err := g.Put(middle)
	require.NoError(t, err)
	require.Equal(t, parent, Claim{})

	// Add the lowest claim to the game state.
	bottom := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
		Position: NewPosition(2, 2),
	}

	// Put the bottom claim into the game state.
	// We should expect the parent to be the claim we just added.
	parent, err = g.Put(bottom)
	require.NoError(t, err)
	require.Equal(t, parent, middle)

	// Construct the highest level parent.
	top := Claim{
		Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position: NewPosition(0, 0),
	}

	// Put the top claim into the game state.
	// We should expect the highest parent to not have a parent.
	// And the first claim to now have the highest parent as its parent.
	parent, err = g.Put(top)
	require.NoError(t, err)
	require.Equal(t, parent, Claim{})
	require.Equal(t, g.GetClaimParent(middle), top)
}

// TestGame_ClaimPairs tests the [Game.ClaimPairs] method using a [gameState] instance.
func TestGame_ClaimPairs(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create a list of claims to add to the game state.
	claims := []Claim{
		{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(2, 2),
		},
		{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
			Position: NewPosition(1, 1),
		},
		{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
			Position: NewPosition(0, 0),
		},
	}

	// Create a map of claim gindex to their parent.
	parents := map[uint64]Claim{
		0: {},
		3: claims[2],
		6: claims[1],
	}

	// Add the claims to the game state.
	for _, claim := range claims {
		_, err := g.Put(claim)
		require.NoError(t, err)
	}

	// Get the list of claim pairs.
	pairs := g.ClaimPairs()

	// Validate the pairs parents are correct.
	for _, pair := range pairs {
		require.Equal(t, parents[pair.claim.ToGIndex()], pair.parent)
	}
}
