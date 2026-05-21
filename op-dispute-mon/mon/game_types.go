package mon

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

type GameTypeMetrics interface {
	RecordGameTypes(gameTypeCounts map[string]int)
}

type GameTypeMonitor struct {
	metrics GameTypeMetrics
}

func NewGameTypeMonitor(metrics GameTypeMetrics) *GameTypeMonitor {
	return &GameTypeMonitor{metrics: metrics}
}

func (m *GameTypeMonitor) CheckGameTypes(games []*types.EnrichedGameData) {
	counts := make(map[string]int)
	for _, game := range games {
		gt := gameTypes.GameType(game.GameType).String()
		counts[gt]++
	}
	m.metrics.RecordGameTypes(counts)
}
