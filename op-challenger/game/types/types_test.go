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
