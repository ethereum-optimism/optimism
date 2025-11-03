package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type NodeEndpointErrorsMetrics interface {
	RecordNodeEndpointErrors(count int)
}

type NodeEndpointErrorsMonitor struct {
	logger  log.Logger
	metrics NodeEndpointErrorsMetrics
}

func NewNodeEndpointErrorsMonitor(logger log.Logger, metrics NodeEndpointErrorsMetrics) *NodeEndpointErrorsMonitor {
	return &NodeEndpointErrorsMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *NodeEndpointErrorsMonitor) CheckNodeEndpointErrors(games []*types.EnrichedGameData) {
	// Use a set to track unique endpoint errors across all games
	uniqueEndpointErrors := make(map[string]bool)

	for _, game := range games {
		if len(game.RollupEndpointErrors) != 0 {
			for endpointID := range game.RollupEndpointErrors {
				uniqueEndpointErrors[endpointID] = true
			}
		}
	}

	errorCount := len(uniqueEndpointErrors)
	m.metrics.RecordNodeEndpointErrors(errorCount)
}
