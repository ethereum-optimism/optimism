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
