package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type DifferentSuperRootMetrics interface {
	RecordDifferentSuperRootGames(count int)
}

type DifferentSuperRootMonitor struct {
	logger  log.Logger
	metrics DifferentSuperRootMetrics
}

func NewDifferentSuperRootMonitor(logger log.Logger, metrics DifferentSuperRootMetrics) *DifferentSuperRootMonitor {
	return &DifferentSuperRootMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *DifferentSuperRootMonitor) CheckDifferentSuperRoots(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.SuperNodeEndpointDifferentSuperRoots {
			count++
			m.logger.Debug("Different super roots detected",
				"game", game.Proxy,
				"l2SequenceNumber", game.L2SequenceNumber,
				"rootClaim", game.RootClaim)
		}
	}

	m.metrics.RecordDifferentSuperRootGames(count)

	if count > 0 {
		m.logger.Info("Different super roots summary", "gamesWithDifferentSuperRoots", count, "totalGames", len(games))
	}
}
