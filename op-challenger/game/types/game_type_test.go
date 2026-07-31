package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedLifecycleGameTypes(t *testing.T) {
	expected := []GameType{
		AlphabetGameType,
		CannonGameType,
		CannonKonaGameType,
		PermissionedGameType,
		SuperPermissionedGameType,
		FastGameType,
		SuperCannonKonaGameType,
		ZKDisputeGameType,
	}
	require.ElementsMatch(t, expected, SupportedLifecycleGameTypes)
}

func TestPlayableGameTypes(t *testing.T) {
	expected := []GameType{
		AlphabetGameType,
		CannonGameType,
		CannonKonaGameType,
		PermissionedGameType,
		FastGameType,
		SuperCannonKonaGameType,
		ZKDisputeGameType,
	}
	require.ElementsMatch(t, expected, PlayableGameTypes)
	require.NotContains(t, PlayableGameTypes, SuperPermissionedGameType)
}

func TestPlayableGameTypesAreLifecycleSupported(t *testing.T) {
	for _, gameType := range PlayableGameTypes {
		require.Contains(t, SupportedLifecycleGameTypes, gameType)
	}
}

func TestSetAllPlayableGameTypes(t *testing.T) {
	for _, gameType := range PlayableGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			result := new(GameType)
			err := result.Set(gameType.String())
			require.NoError(t, err)
			require.Equal(t, gameType, *result)
		})
	}
}

func TestSetRejectsSuperPermissioned(t *testing.T) {
	result := new(GameType)
	err := result.Set(SuperPermissionedGameType.String())
	require.ErrorIs(t, err, ErrUnknownGameType)
}

func TestPlayableGameTypeFromStringAcceptsAllPlayableGameTypes(t *testing.T) {
	for _, gameType := range PlayableGameTypes {
		t.Run(gameType.String(), func(t *testing.T) {
			result, err := PlayableGameTypeFromString(gameType.String())
			require.NoError(t, err)
			require.Equal(t, gameType, result)
		})
	}
}

func TestPlayableGameTypeFromStringRejectsNonPlayableGameTypes(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "super-permissioned", value: SuperPermissionedGameType.String()},
		{name: "legacy super-cannon", value: "super-cannon"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := PlayableGameTypeFromString(test.value)
			require.ErrorIs(t, err, ErrUnknownGameType)
			require.Equal(t, UnknownGameType, result)
		})
	}
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

func TestStableStringForAllLifecycleGameTypes(t *testing.T) {
	expected := map[GameType]string{
		AlphabetGameType:          "alphabet",
		CannonGameType:            "cannon",
		CannonKonaGameType:        "cannon-kona",
		PermissionedGameType:      "permissioned",
		SuperPermissionedGameType: "super-permissioned",
		FastGameType:              "fast",
		SuperCannonKonaGameType:   "super-cannon-kona",
		ZKDisputeGameType:         "zk",
	}
	require.Len(t, SupportedLifecycleGameTypes, len(expected))

	for _, gameType := range SupportedLifecycleGameTypes {
		expectedString, ok := expected[gameType]
		require.True(t, ok, "missing stable string assertion for lifecycle game type %d", gameType)
		t.Run(expectedString, func(t *testing.T) {
			require.Equal(t, expectedString, gameType.String())
		})
	}
}
