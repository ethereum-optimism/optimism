package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAllSupportedGameTypes(t *testing.T) {
	for _, gameType := range SupportedGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			result := new(GameType)
			err := result.Set(gameType.String())
			require.NoError(t, err, "failed to set game type")

			require.Equal(t, gameType, *result)
		})
	}
}

func TestGameTypeFromStringForAllSupportedGameTypes(t *testing.T) {
	for _, gameType := range SupportedGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			result, err := SupportedGameTypeFromString(gameType.String())
			require.NoError(t, err, "failed to get game type from string")

			require.Equal(t, gameType, result)
		})
	}
}

func TestSupportedGameTypeFromString_RejectsSuperCannon(t *testing.T) {
	result, err := SupportedGameTypeFromString("super-cannon")
	require.ErrorIs(t, err, ErrUnknownGameType)
	require.Equal(t, UnknownGameType, result)
}

func TestSupportedGameTypeFromString_RejectsSuperPermissioned(t *testing.T) {
	result, err := SupportedGameTypeFromString(SuperPermissionedGameType.String())
	require.ErrorIs(t, err, ErrUnknownGameType)
	require.Equal(t, UnknownGameType, result)
}

func TestIsPermissioned(t *testing.T) {
	permissioned := []GameType{PermissionedGameType, SuperPermissionedGameType}
	for _, gameType := range permissioned {
		t.Run(gameType.String(), func(t *testing.T) {
			require.True(t, gameType.IsPermissioned())
		})
	}

	notPermissioned := []GameType{CannonGameType, CannonKonaGameType, SuperCannonKonaGameType, ZKDisputeGameType, FastGameType, AlphabetGameType}
	for _, gameType := range notPermissioned {
		t.Run(gameType.String(), func(t *testing.T) {
			require.False(t, gameType.IsPermissioned())
		})
	}
}

func TestKnownStringForAllSupportedGameTypes(t *testing.T) {
	for _, gameType := range SupportedGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			require.NotContains(t, gameType.String(), "invalid")
		})
	}

	t.Run("UnknownGameTypeStringContainsInvalid", func(t *testing.T) {
		// Check that the test above would detect if we hit the unknown case
		require.Contains(t, GameType(4829482).String(), "invalid")
	})
}
