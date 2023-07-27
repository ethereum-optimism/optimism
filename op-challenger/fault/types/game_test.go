package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const testMaxDepth = 3

func createTestClaims() (Claim, Claim, Claim, Claim) {
	// root & middle are from the trace "abcdexyz"
	// top & bottom are from the trace  "abcdefgh"
	root := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: NewPosition(0, 0),
		},
		// Root claim has no parent
	}
	top := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: NewPosition(1, 0),
		},
		Parent: root.ClaimData,
	}
	middle := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: NewPosition(2, 2),
		},
		Parent: top.ClaimData,
	}

	bottom := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(3, 4),
		},
		Parent: middle.ClaimData,
	}

	return root, top, middle, bottom
}

func TestIsDuplicate(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)
	require.NoError(t, g.Put(top))

	// Root + Top should be duplicates
	require.True(t, g.IsDuplicate(root))
	require.True(t, g.IsDuplicate(top))

	// Middle + Bottom should not be a duplicate
	require.False(t, g.IsDuplicate(middle))
	require.False(t, g.IsDuplicate(bottom))
}

// TestGame_Put_RootAlreadyExists tests the [Game.Put] method using a [gameState]
// instance errors when the root claim already exists in state.
func TestGame_Put_RootAlreadyExists(t *testing.T) {
	// Setup the game state.
	top, _, _, _ := createTestClaims()
	g := NewGameState(false, top, testMaxDepth)

	// Try to put the root claim into the game state again.
	err := g.Put(top)
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_PutAll_RootAlreadyExists tests the [Game.PutAll] method using a [gameState]
// instance errors when the root claim already exists in state.
func TestGame_PutAll_RootAlreadyExists(t *testing.T) {
	// Setup the game state.
	root, _, _, _ := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)

	// Try to put the root claim into the game state again.
	err := g.PutAll([]Claim{root})
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_PutAll_AlreadyExists tests the [Game.PutAll] method using a [gameState]
// instance errors when the given claim already exists in state.
func TestGame_PutAll_AlreadyExists(t *testing.T) {
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)

	err := g.PutAll([]Claim{top, middle})
	require.NoError(t, err)

	err = g.PutAll([]Claim{middle, bottom})
	require.ErrorIs(t, err, ErrClaimExists)
}

// TestGame_PutAll_ParentsAndChildren tests the [Game.PutAll] method using a [gameState] instance.
func TestGame_PutAll_ParentsAndChildren(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)

	// We should not be able to get the parent of the root claim.
	parent, err := g.getParent(root)
	require.ErrorIs(t, err, ErrClaimNotFound)
	require.Equal(t, parent, Claim{})

	// Put the rest of the claims in the state.
	err = g.PutAll([]Claim{top, middle, bottom})
	require.NoError(t, err)
	parent, err = g.getParent(top)
	require.NoError(t, err)
	require.Equal(t, parent, root)
	parent, err = g.getParent(middle)
	require.NoError(t, err)
	require.Equal(t, parent, top)
	parent, err = g.getParent(bottom)
	require.NoError(t, err)
	require.Equal(t, parent, middle)
}

// TestGame_Put_AlreadyExists tests the [Game.Put] method using a [gameState]
// instance errors when the given claim already exists in state.
func TestGame_Put_AlreadyExists(t *testing.T) {
	// Setup the game state.
	top, middle, _, _ := createTestClaims()
	g := NewGameState(false, top, testMaxDepth)

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
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)

	// We should not be able to get the parent of the root claim.
	parent, err := g.getParent(root)
	require.ErrorIs(t, err, ErrClaimNotFound)
	require.Equal(t, parent, Claim{})

	// Put + Check Top
	err = g.Put(top)
	require.NoError(t, err)
	parent, err = g.getParent(top)
	require.NoError(t, err)
	require.Equal(t, parent, root)

	// Put + Check Top Middle
	err = g.Put(middle)
	require.NoError(t, err)
	parent, err = g.getParent(middle)
	require.NoError(t, err)
	require.Equal(t, parent, top)

	// Put + Check Top Bottom
	err = g.Put(bottom)
	require.NoError(t, err)
	parent, err = g.getParent(bottom)
	require.NoError(t, err)
	require.Equal(t, parent, middle)
}

// TestGame_ClaimPairs tests the [Game.ClaimPairs] method using a [gameState] instance.
func TestGame_ClaimPairs(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)

	// Add top claim to the game state.
	err := g.Put(top)
	require.NoError(t, err)

	// Add the middle claim to the game state.
	err = g.Put(middle)
	require.NoError(t, err)

	// Add the bottom claim to the game state.
	err = g.Put(bottom)
	require.NoError(t, err)

	// Validate claim pairs.
	expected := []Claim{root, top, middle, bottom}
	claims := g.Claims()
	require.ElementsMatch(t, expected, claims)
}

func TestAgreeWithClaimLevelDisagreeWithOutput(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, root, testMaxDepth)
	require.NoError(t, g.PutAll([]Claim{top, middle, bottom}))

	require.True(t, g.AgreeWithClaimLevel(root))
	require.False(t, g.AgreeWithClaimLevel(top))
	require.True(t, g.AgreeWithClaimLevel(middle))
	require.False(t, g.AgreeWithClaimLevel(bottom))
}

func TestAgreeWithClaimLevelAgreeWithOutput(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(true, root, testMaxDepth)
	require.NoError(t, g.PutAll([]Claim{top, middle, bottom}))

	require.False(t, g.AgreeWithClaimLevel(root))
	require.True(t, g.AgreeWithClaimLevel(top))
	require.False(t, g.AgreeWithClaimLevel(middle))
	require.True(t, g.AgreeWithClaimLevel(bottom))
}
