package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type NodeEndpointErrorCountMetrics interface {
	RecordNodeEndpointErrorCount(count int)
}

type NodeEndpointErrorCountMonitor struct {
	logger  log.Logger
	metrics NodeEndpointErrorCountMetrics
}

func NewNodeEndpointErrorCountMonitor(logger log.Logger, metrics NodeEndpointErrorCountMetrics) *NodeEndpointErrorCountMonitor {
	return &NodeEndpointErrorCountMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *NodeEndpointErrorCountMonitor) CheckNodeEndpointErrorCount(games []*types.EnrichedGameData) {
	totalErrors := 0

	for _, game := range games {
		totalErrors += game.RollupEndpointErrorCount
	}

	m.metrics.RecordNodeEndpointErrorCount(totalErrors)
}

// countGamesWithErrors returns the number of games that have at least one error
func countGamesWithErrors(games []*types.EnrichedGameData) int {
	count := 0
	for _, game := range games {
		if game.RollupEndpointErrorCount > 0 {
			count++
		}
	}
	return count
}
