package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type DBMetricer interface {
	RecordDBEntryCount(chainID eth.ChainID, kind string, count int64)
	RecordDBSearchEntriesRead(chainID eth.ChainID, count int64)
}

type DBMetrics struct {
	DBEntryCountVec        *prometheus.GaugeVec
	DBSearchEntriesReadVec *prometheus.HistogramVec
}

var _ DBMetricer = (*DBMetrics)(nil)

func NewDBMetrics(ns string, factory metrics.Factory) *DBMetrics {
	return &DBMetrics{
		DBEntryCountVec: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "logdb_entries_current",
			Help:      "Current number of entries in the database of specified kind and chain ID",
		}, []string{
			"chain",
			"kind",
		}),
		DBSearchEntriesReadVec: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "logdb_search_entries_read",
			Help:      "Entries read per search of the log database",
			Buckets:   []float64{1, 2, 5, 10, 100, 200, 256},
		}, []string{
			"chain",
		}),
	}
}

func (m *DBMetrics) RecordDBEntryCount(chainID eth.ChainID, kind string, count int64) {
	m.DBEntryCountVec.WithLabelValues(chainIDLabel(chainID), kind).Set(float64(count))
}

func (m *DBMetrics) RecordDBSearchEntriesRead(chainID eth.ChainID, count int64) {
	m.DBSearchEntriesReadVec.WithLabelValues(chainIDLabel(chainID)).Observe(float64(count))
}

type NoopDBMetrics struct{}

var _ DBMetricer = NoopDBMetrics{}

func (NoopDBMetrics) RecordDBEntryCount(_ eth.ChainID, _ string, _ int64) {}
func (NoopDBMetrics) RecordDBSearchEntriesRead(_ eth.ChainID, _ int64)    {}
