package types

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPreimageOracleData(t *testing.T) {
	t.Run("LocalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{1, 2, 3}, []byte{4, 5, 6}, 7)
		require.True(t, data.IsLocal)
		require.Equal(t, []byte{1, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
		require.Equal(t, uint32(7), data.OracleOffset)
	})

	t.Run("GlobalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{0, 2, 3}, []byte{4, 5, 6}, 7)
		require.False(t, data.IsLocal)
		require.Equal(t, []byte{0, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
		require.Equal(t, uint32(7), data.OracleOffset)
	})
}

func TestIsRootPosition(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		expected bool
	}{
		{
			name:     "ZeroRoot",
			position: NewPositionFromGIndex(big.NewInt(0)),
			expected: true,
		},
		{
			name:     "ValidRoot",
			position: NewPositionFromGIndex(big.NewInt(1)),
			expected: true,
		},
		{
			name:     "NotRoot",
			position: NewPositionFromGIndex(big.NewInt(2)),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.position.IsRootPosition())
		})
	}
}

func buildClaim(gindex *big.Int, parentGIndex *big.Int) Claim {
	return Claim{
		ClaimData: ClaimData{
			Position: NewPositionFromGIndex(gindex),
		},
		Parent: ClaimData{
			Position: NewPositionFromGIndex(parentGIndex),
		},
	}
}

func TestDefendsParent(t *testing.T) {
	tests := []struct {
		name     string
		claim    Claim
		expected bool
	}{
		{
			name:     "LeftChildAttacks",
			claim:    buildClaim(big.NewInt(2), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightChildDoesntDefend",
			claim:    buildClaim(big.NewInt(3), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubChildDoesntDefend",
			claim:    buildClaim(big.NewInt(4), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "SubSecondChildDoesntDefend",
			claim:    buildClaim(big.NewInt(5), big.NewInt(1)),
			expected: false,
		},
		{
			name:     "RightLeftChildDefendsParent",
			claim:    buildClaim(big.NewInt(6), big.NewInt(1)),
			expected: true,
		},
		{
			name:     "SubThirdChildDefends",
			claim:    buildClaim(big.NewInt(7), big.NewInt(1)),
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.claim.DefendsParent())
		})
	}
}
