package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func createTestClaims() []Claim {
	top := Claim{
		Value:         common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position:      NewPosition(0, 0),
		Parent:        nil,
		DefendsParent: false,
	}

	middle := Claim{
		Value:         common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
		Position:      NewPosition(1, 1),
		Parent:        &top,
		DefendsParent: true,
	}

	bottom := Claim{
		Value:         common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
		Position:      NewPosition(2, 2),
		Parent:        &middle,
		DefendsParent: false,
	}

	return []Claim{top, middle, bottom}
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
	err := g.Put(claim)
	require.NoError(t, err)

	// Put the claim into the game state again.
	err = g.Put(claim)
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_Put_ParentsAndChildren tests the [Game.Put] method using a [gameState] instance.
func TestGame_Put_ParentsAndChildren(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create test claims
	claims := createTestClaims()
	top := claims[0]
	middle := claims[1]
	bottom := claims[2]

	// Put the middle claim into the game state.
	// We should expect no parent to exist, yet.
	err := g.Put(middle)
	require.NoError(t, err)
	parent, err := g.getParent(middle)
	require.NoError(t, err)
	require.Equal(t, parent, top)

	// Put the bottom claim into the game state.
	// We should expect the parent to be the claim we just added.
	err = g.Put(bottom)
	require.NoError(t, err)
	parent, err = g.getParent(bottom)
	require.NoError(t, err)
	require.Equal(t, parent, middle)

	// Put the top claim into the game state.
	// We should expect the highest parent to not have a parent.
	// And the first claim to now have the highest parent as its parent.
	err = g.Put(top)
	require.NoError(t, err)
	parent, err = g.getParent(top)
	require.ErrorIs(t, err, ErrClaimNotFound)
	require.Equal(t, parent, Claim{})
}

// TestGame_ClaimPairs tests the [Game.ClaimPairs] method using a [gameState] instance.
func TestGame_ClaimPairs(t *testing.T) {
	// Create a new game state.
	g := NewGameState()

	// Create test claims
	claims := createTestClaims()

	// Create a map of claim gindex to their parent.
	parents := map[uint64]Claim{
		0: {},
		3: claims[0],
		6: claims[1],
	}

	// Add the claims to the game state.
	for _, claim := range claims {
		err := g.Put(claim)
		require.NoError(t, err)
	}

	// Get the list of claim pairs.
	pairs := g.ClaimPairs()

	// Validate the pairs parents are correct.
	for _, pair := range pairs {
		require.Equal(t, parents[pair.claim.ToGIndex()], pair.parent)
	}
}
