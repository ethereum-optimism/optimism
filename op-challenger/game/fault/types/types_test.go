package types

import (
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
			position: NewPositionFromGIndex(0),
			expected: true,
		},
		{
			name:     "ValidRoot",
			position: NewPositionFromGIndex(1),
			expected: true,
		},
		{
			name:     "NotRoot",
			position: NewPositionFromGIndex(2),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.position.IsRootPosition())
		})
	}
}

func buildClaim(gindex uint64, parentGIndex uint64) Claim {
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
			claim:    buildClaim(2, 1),
			expected: false,
		},
		{
			name:     "RightChildDoesntDefend",
			claim:    buildClaim(3, 1),
			expected: false,
		},
		{
			name:     "SubChildDoesntDefend",
			claim:    buildClaim(4, 1),
			expected: false,
		},
		{
			name:     "SubSecondChildDoesntDefend",
			claim:    buildClaim(5, 1),
			expected: false,
		},
		{
			name:     "RightLeftChildDefendsParent",
			claim:    buildClaim(6, 1),
			expected: true,
		},
		{
			name:     "SubThirdChildDefends",
			claim:    buildClaim(7, 1),
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.claim.DefendsParent())
		})
	}
}
