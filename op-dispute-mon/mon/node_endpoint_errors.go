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
	if errorCount > 0 {
		m.logger.Warn("Found rollup node endpoint errors",
			"unique_endpoint_count", errorCount,
			"endpoints", getEndpointList(uniqueEndpointErrors))
	} else {
		m.logger.Debug("No rollup node endpoint errors found")
	}

	m.metrics.RecordNodeEndpointErrors(errorCount)
}

// getEndpointList converts the map keys to a slice for logging
func getEndpointList(endpointErrors map[string]bool) []string {
	endpoints := make([]string, 0, len(endpointErrors))
	for endpointID := range endpointErrors {
		endpoints = append(endpoints, endpointID)
	}
	return endpoints
}
