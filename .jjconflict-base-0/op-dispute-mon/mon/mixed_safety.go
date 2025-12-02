package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type MixedSafetyMetrics interface {
	RecordMixedSafetyGames(count int)
}

type MixedSafetyMonitor struct {
	logger  log.Logger
	metrics MixedSafetyMetrics
}

func NewMixedSafetyMonitor(logger log.Logger, metrics MixedSafetyMetrics) *MixedSafetyMonitor {
	return &MixedSafetyMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *MixedSafetyMonitor) CheckMixedSafety(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.HasMixedSafety() {
			count++
			m.logger.Debug("Mixed safety detected",
				"game", game.Proxy,
				"safeCount", game.RollupEndpointSafeCount,
				"unsafeCount", game.RollupEndpointUnsafeCount)
		}
	}

	m.metrics.RecordMixedSafetyGames(count)

	if count > 0 {
		m.logger.Info("Mixed safety summary", "gamesWithMixedSafety", count, "totalGames", len(games))
	}
}
