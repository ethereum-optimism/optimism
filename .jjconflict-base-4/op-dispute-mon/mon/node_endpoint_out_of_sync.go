package mon

import (
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum/go-ethereum/log"
)

type NodeEndpointOutOfSyncMetrics interface {
	RecordNodeEndpointOutOfSyncCount(count int)
}

type NodeEndpointOutOfSyncMonitor struct {
	logger  log.Logger
	metrics NodeEndpointOutOfSyncMetrics
}

func NewNodeEndpointOutOfSyncMonitor(logger log.Logger, metrics NodeEndpointOutOfSyncMetrics) *NodeEndpointOutOfSyncMonitor {
	return &NodeEndpointOutOfSyncMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

func (m *NodeEndpointOutOfSyncMonitor) CheckNodeEndpointOutOfSync(games []*types.EnrichedGameData) {
	totalOutOfSync := 0

	for _, game := range games {
		totalOutOfSync += game.RollupEndpointOutOfSyncCount
	}

	m.metrics.RecordNodeEndpointOutOfSyncCount(totalOutOfSync)
}
