package metrics

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type noopMetrics struct {
	opmetrics.NoopRPCMetrics
	event.NoopMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) Document() []opmetrics.DocumentedMetric { return nil }

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (m *noopMetrics) RecordCrossUnsafe(_ eth.ChainID, _ types.BlockSeal) {}
func (m *noopMetrics) RecordCrossSafe(_ eth.ChainID, _ types.BlockSeal)   {}
func (m *noopMetrics) RecordLocalSafe(_ eth.ChainID, _ types.BlockSeal)   {}
func (m *noopMetrics) RecordLocalUnsafe(_ eth.ChainID, _ types.BlockSeal) {}

func (m *noopMetrics) CacheAdd(_ eth.ChainID, _ string, _ int, _ bool) {}
func (m *noopMetrics) CacheGet(_ eth.ChainID, _ string, _ bool)        {}

func (m *noopMetrics) RecordDBEntryCount(_ eth.ChainID, _ string, _ int64) {}
func (m *noopMetrics) RecordDBSearchEntriesRead(_ eth.ChainID, _ int64)    {}

func (m *noopMetrics) RecordAccessListVerifyFailure(_ eth.ChainID) {}
