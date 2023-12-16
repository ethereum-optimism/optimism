package types

import (
	"math/big"
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
			Position: NewPosition(0, common.Big0),
		},
		// Root claim has no parent
	}
	top := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: NewPosition(1, common.Big0),
		},
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	middle := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: NewPosition(2, big.NewInt(2)),
		},
		ContractIndex:       2,
		ParentContractIndex: 1,
	}

	bottom := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(3, big.NewInt(4)),
		},
		ContractIndex:       3,
		ParentContractIndex: 2,
	}

	return root, top, middle, bottom
}

func TestIsDuplicate(t *testing.T) {
	root, top, middle, bottom := createTestClaims()
	g := NewGameState([]Claim{root, top}, testMaxDepth)

	// Root + Top should be duplicates
	require.True(t, g.IsDuplicate(root))
	require.True(t, g.IsDuplicate(top))

	// Middle + Bottom should not be a duplicate
	require.False(t, g.IsDuplicate(middle))
	require.False(t, g.IsDuplicate(bottom))
}

func TestGame_Claims(t *testing.T) {
	// Setup the game state.
	root, top, middle, bottom := createTestClaims()
	expected := []Claim{root, top, middle, bottom}
	g := NewGameState(expected, testMaxDepth)

	// Validate claim pairs.
	actual := g.Claims()
	require.ElementsMatch(t, expected, actual)
}

func TestGame_DefendsParent(t *testing.T) {
	tests := []struct {
		name     string
		game     *gameState
		expected bool
	}{
		{
			name:     "LeftChildAttacks",
			game:     buildGameWithClaim(big.NewInt(2), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(3), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(4), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubSecondChildDoesntDefend",
			game:     buildGameWithClaim(big.NewInt(5), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightLeftChildDefendsParent",
			game:     buildGameWithClaim(big.NewInt(6), big.NewInt(1)),
			expected: true,
		},
		{
			name:     "SubThirdChildDefends",
			game:     buildGameWithClaim(big.NewInt(7), big.NewInt(1)),
			expected: true,
		},
		{
			name: "RootDoesntDefend",
			game: NewGameState([]Claim{
				{
					ClaimData: ClaimData{
						Position: NewPositionFromGIndex(big.NewInt(0)),
					},
					ContractIndex: 0,
				},
			}, testMaxDepth),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := test.game.Claims()
			require.Equal(t, test.expected, test.game.DefendsParent(claims[len(claims)-1]))
		})
	}
}

func buildGameWithClaim(claimGIndex *big.Int, parentGIndex *big.Int) *gameState {
	parentClaim := Claim{
		ClaimData: ClaimData{
			Position: NewPositionFromGIndex(parentGIndex),
		},
		ContractIndex: 0,
	}
	claim := Claim{
		ClaimData: ClaimData{
			Position: NewPositionFromGIndex(claimGIndex),
		},
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	return NewGameState([]Claim{parentClaim, claim}, testMaxDepth)
}
