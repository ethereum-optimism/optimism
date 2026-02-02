package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type SuperNodeEndpointErrorCountMetrics interface {
	RecordSuperNodeEndpointErrorCount(count int)
}

type SuperNodeEndpointErrorCountMonitor struct {
	logger  log.Logger
	metrics SuperNodeEndpointErrorCountMetrics
}

func NewSuperNodeEndpointErrorCountMonitor(logger log.Logger, metrics SuperNodeEndpointErrorCountMetrics) *SuperNodeEndpointErrorCountMonitor {
	return &SuperNodeEndpointErrorCountMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *SuperNodeEndpointErrorCountMonitor) CheckSuperNodeEndpointErrorCount(games []*types.EnrichedGameData) {
	totalErrors := 0

	for _, game := range games {
		totalErrors += game.SuperNodeEndpointErrorCount
	}

	m.metrics.RecordSuperNodeEndpointErrorCount(totalErrors)
}
