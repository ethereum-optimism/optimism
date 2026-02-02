package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type MixedSuperAvailabilityMetrics interface {
	RecordMixedSuperAvailabilityGames(count int)
}

type MixedSuperAvailability struct {
	logger  log.Logger
	metrics MixedSuperAvailabilityMetrics
}

func NewMixedSuperAvailability(logger log.Logger, metrics MixedSuperAvailabilityMetrics) *MixedSuperAvailability {
	return &MixedSuperAvailability{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *MixedSuperAvailability) CheckMixedSuperAvailability(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.HasMixedSuperAvailability() {
			count++
			m.logger.Debug("Mixed super availability detected",
				"game", game.Proxy,
				"totalEndpoints", game.SuperNodeEndpointTotalCount,
				"notFoundCount", game.SuperNodeEndpointNotFoundCount,
				"errorCount", game.SuperNodeEndpointErrorCount)
		}
	}

	m.metrics.RecordMixedSuperAvailabilityGames(count)

	if count > 0 {
		m.logger.Info("Mixed super availability summary", "gamesWithMixedSuperAvailability", count, "totalGames", len(games))
	}
}
