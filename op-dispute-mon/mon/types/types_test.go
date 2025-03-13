package types

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/stretchr/testify/require"
)

func TestEnrichedGameData_UsesOutputRoots(t *testing.T) {
	for _, gameType := range outputRootGameTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := EnrichedGameData{
				GameMetadata: types.GameMetadata{GameType: gameType},
			}
			require.True(t, data.UsesOutputRoots())
		})
	}

	nonOutputRootTypes := []uint32{4, 5, 9, 42982, 20013130}
	for _, gameType := range nonOutputRootTypes {
		gameType := gameType
		t.Run(fmt.Sprintf("GameType-%v", gameType), func(t *testing.T) {
			data := EnrichedGameData{
				GameMetadata: types.GameMetadata{GameType: gameType},
			}
			require.False(t, data.UsesOutputRoots())
		})
	}
}
