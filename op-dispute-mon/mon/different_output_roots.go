package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type DifferentOutputRootMetrics interface {
	RecordDifferentOutputRootGames(count int)
}

type DifferentOutputRootMonitor struct {
	logger  log.Logger
	metrics DifferentOutputRootMetrics
}

func NewDifferentOutputRootMonitor(logger log.Logger, metrics DifferentOutputRootMetrics) *DifferentOutputRootMonitor {
	return &DifferentOutputRootMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *DifferentOutputRootMonitor) CheckDifferentOutputRoots(games []*types.EnrichedGameData) {
	count := 0
	for _, game := range games {
		if game.RollupEndpointDifferentOutputRoots {
			count++
			m.logger.Debug("Different output roots detected",
				"game", game.Proxy,
				"l2BlockNumber", game.L2BlockNumber,
				"rootClaim", game.RootClaim)
		}
	}

	m.metrics.RecordDifferentOutputRootGames(count)

	if count > 0 {
		m.logger.Info("Different output roots summary", "gamesWithDifferentOutputRoots", count, "totalGames", len(games))
	}
}
