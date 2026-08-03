package mon

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/stretchr/testify/require"
)

func TestGameTypeMonitorZerosAgedOutTypes(t *testing.T) {
	metrics := &stubGameTypeMetrics{}
	monitor := NewGameTypeMonitor(metrics)
	monitor.CheckGameTypes([]*monTypes.CommonGameData{{
		GameMetadata: gameTypes.GameMetadata{GameType: uint32(gameTypes.ZKDisputeGameType)},
	}})
	require.Equal(t, 1, metrics.counts[gameTypes.ZKDisputeGameType.String()])

	monitor.CheckGameTypes(nil)
	require.Equal(t, 0, metrics.counts[gameTypes.ZKDisputeGameType.String()])
	for _, gameType := range gameTypes.SupportedLifecycleGameTypes {
		require.Contains(t, metrics.counts, gameType.String())
	}
}

type stubGameTypeMetrics struct {
	counts map[string]int
}

func (s *stubGameTypeMetrics) RecordGameTypes(counts map[string]int) {
	s.counts = counts
}
