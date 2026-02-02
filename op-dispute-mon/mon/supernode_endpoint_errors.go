package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type SuperNodeEndpointErrorsMetrics interface {
	RecordSuperNodeEndpointErrors(count int)
}

type SuperNodeEndpointErrorsMonitor struct {
	logger  log.Logger
	metrics SuperNodeEndpointErrorsMetrics
}

func NewSuperNodeEndpointErrorsMonitor(logger log.Logger, metrics SuperNodeEndpointErrorsMetrics) *SuperNodeEndpointErrorsMonitor {
	return &SuperNodeEndpointErrorsMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *SuperNodeEndpointErrorsMonitor) CheckSuperNodeEndpointErrors(games []*types.EnrichedGameData) {
	// Use a set to track unique endpoint errors across all games
	uniqueEndpointErrors := make(map[string]bool)

	for _, game := range games {
		if len(game.SuperNodeEndpointErrors) != 0 {
			for endpointID := range game.SuperNodeEndpointErrors {
				uniqueEndpointErrors[endpointID] = true
			}
		}
	}

	errorCount := len(uniqueEndpointErrors)
	m.metrics.RecordSuperNodeEndpointErrors(errorCount)
}
