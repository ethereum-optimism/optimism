package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type MixedAvailabilityMetrics interface {
	RecordMixedAvailabilityGames(count int)
}

type MixedAvailability struct {
	logger  log.Logger
	metrics MixedAvailabilityMetrics
}

func NewMixedAvailability(logger log.Logger, metrics MixedAvailabilityMetrics) *MixedAvailability {
	return &MixedAvailability{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *MixedAvailability) CheckMixedAvailability(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.HasMixedAvailability() {
			count++
			m.logger.Debug("Mixed availability detected",
				"game", game.Proxy,
				"totalEndpoints", game.RollupEndpointTotalCount,
				"notFoundCount", game.RollupEndpointNotFoundCount,
				"errorCount", game.RollupEndpointErrorCount)
		}
	}

	m.metrics.RecordMixedAvailabilityGames(count)

	if count > 0 {
		m.logger.Warn("Mixed availability summary", "gamesWithMixedAvailability", count, "totalGames", len(games))
	}
}
