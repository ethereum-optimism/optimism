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
		Parent:              root.ClaimData,
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	middle := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: NewPosition(2, big.NewInt(2)),
		},
		Parent:              top.ClaimData,
		ContractIndex:       2,
		ParentContractIndex: 1,
	}

	bottom := Claim{
		ClaimData: ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: NewPosition(3, big.NewInt(4)),
		},
		Parent:              middle.ClaimData,
		ContractIndex:       3,
		ParentContractIndex: 2,
	}

	return root, top, middle, bottom
}

func TestIsDuplicate(t *testing.T) {
	root, top, middle, bottom := createTestClaims()
	g := NewGameState(false, []Claim{root, top}, testMaxDepth)

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
	g := NewGameState(false, expected, testMaxDepth)

	// Validate claim pairs.
	actual := g.Claims()
	require.ElementsMatch(t, expected, actual)
}
