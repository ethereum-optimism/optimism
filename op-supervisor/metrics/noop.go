package metrics

import (
	"math/big"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

type noopMetrics struct {
	opmetrics.NoopRPCMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) Document() []opmetrics.DocumentedMetric { return nil }

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (m *noopMetrics) CacheAdd(_ *big.Int, _ string, _ int, _ bool) {}
func (m *noopMetrics) CacheGet(_ *big.Int, _ string, _ bool)        {}
