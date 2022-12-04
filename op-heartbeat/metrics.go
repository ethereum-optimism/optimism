package op_heartbeat

import (
	"strconv"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const MetricsNamespace = "op_heartbeat"

type Metrics interface {
	RecordHeartbeat(payload heartbeat.Payload)
	RecordVersion(version string)
}

type metrics struct {
	heartbeats *prometheus.CounterVec
	version    *prometheus.GaugeVec
}

func NewMetrics(r *prometheus.Registry) Metrics {
	m := &metrics{
		heartbeats: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "heartbeats",
			Help:      "Counts number of heartbeats by chain ID",
		}, []string{
			"chain_id",
			"version",
		}),
		version: promauto.With(r).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "version",
			Help:      "version pseudo-metrics",
		}, []string{
			"version",
		}),
	}
	return m
}

func (m *metrics) RecordHeartbeat(payload heartbeat.Payload) {
	var chainID string
	if AllowedChainIDs[payload.ChainID] {
		chainID = strconv.FormatUint(payload.ChainID, 10)
	} else {
		chainID = "unknown"
	}
	var version string
	if AllowedVersions[payload.Version] {
		version = payload.Version
	} else {
		version = "unknown"
	}
	m.heartbeats.WithLabelValues(chainID, version).Inc()
}

func (m *metrics) RecordVersion(version string) {
	m.version.WithLabelValues(version).Set(1)
}
