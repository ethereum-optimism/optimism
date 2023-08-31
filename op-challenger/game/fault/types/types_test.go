package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var validGameStatuses = []GameStatus{
	GameStatusInProgress,
	GameStatusChallengerWon,
	GameStatusDefenderWon,
}

func TestGameStatusFromUint8(t *testing.T) {
	for _, status := range validGameStatuses {
		t.Run(fmt.Sprintf("Valid Game Status %v", status), func(t *testing.T) {
			parsed, err := GameStatusFromUint8(uint8(status))
			require.NoError(t, err)
			require.Equal(t, status, parsed)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		status, err := GameStatusFromUint8(3)
		require.Error(t, err)
		require.Equal(t, GameStatus(3), status)
	})
}

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
