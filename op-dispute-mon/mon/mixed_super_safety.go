package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type MixedSuperSafetyMetrics interface {
	RecordMixedSuperSafetyGames(count int)
}

type MixedSuperSafetyMonitor struct {
	logger  log.Logger
	metrics MixedSuperSafetyMetrics
}

func NewMixedSuperSafetyMonitor(logger log.Logger, metrics MixedSuperSafetyMetrics) *MixedSuperSafetyMonitor {
	return &MixedSuperSafetyMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *MixedSuperSafetyMonitor) CheckMixedSuperSafety(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.HasMixedSuperSafety() {
			count++
			m.logger.Debug("Mixed super safety detected",
				"game", game.Proxy,
				"safeCount", game.SuperNodeEndpointSafeCount,
				"unsafeCount", game.SuperNodeEndpointUnsafeCount)
		}
	}

	m.metrics.RecordMixedSuperSafetyGames(count)

	if count > 0 {
		m.logger.Info("Mixed super safety summary", "gamesWithMixedSuperSafety", count, "totalGames", len(games))
	}
}
