package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	disputeGames = []struct {
		name     string
		gameType GameType
	}{
		{"attestation", AttestationDisputeGameType},
		{"fault", FaultDisputeGameType},
		{"validity", ValidityDisputeGameType},
	}
)

// TestDefaultGameType returns the default dispute game type.
func TestDefaultGameType(t *testing.T) {
	defaultGameType := disputeGames[0].gameType
	require.Equal(t, defaultGameType, DefaultGameType())
}

// TestGameType_Valid tests the Valid function with valid inputs.
func TestGameType_Valid(t *testing.T) {
	for _, game := range disputeGames {
		require.True(t, game.gameType.Valid())
	}
}

// TestGameType_Invalid tests the Valid function with an invalid input.
func TestGameType_Invalid(t *testing.T) {
	invalid := disputeGames[len(disputeGames)-1].gameType + 1
	require.False(t, GameType(invalid).Valid())
}

// FuzzGameType_Invalid checks that invalid game types are correctly
// returned as invalid by the validation [Valid] function.
func FuzzGameType_Invalid(f *testing.F) {
	maxCount := len(DisputeGameTypes)
	f.Fuzz(func(t *testing.T, number uint8) {
		if number >= uint8(maxCount) {
			require.False(t, GameType(number).Valid())
		} else {
			require.True(t, GameType(number).Valid())
		}
	})
}

// TestGameType_Default tests the default value of the DisputeGameType.
func TestGameType_Default(t *testing.T) {
	d := NewDisputeGameType()
	require.Equal(t, DefaultGameType(), d.selected)
	require.Equal(t, DefaultGameType(), d.Type())
}

// TestGameType_String tests the Set and String function on the DisputeGameType.
func TestGameType_String(t *testing.T) {
	for _, dg := range disputeGames {
		t.Run(dg.name, func(t *testing.T) {
			d := NewDisputeGameType()
			require.Equal(t, dg.name, dg.gameType.String())
			require.NoError(t, d.Set(dg.name))
			require.Equal(t, dg.name, d.String())
			require.Equal(t, dg.gameType, d.selected)
		})
	}
}

// TestGameType_Type tests the Type function on the DisputeGameType.
func TestGameType_Type(t *testing.T) {
	for _, dg := range disputeGames {
		t.Run(dg.name, func(t *testing.T) {
			d := NewDisputeGameType()
			require.Equal(t, dg.name, dg.gameType.String())
			require.NoError(t, d.Set(dg.name))
			require.Equal(t, dg.gameType, d.Type())
			require.Equal(t, dg.gameType, d.selected)
		})
	}
}
