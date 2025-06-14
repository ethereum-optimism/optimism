package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// DeprecatedMetrics is unused. It is here to document old metrics that used to exist.
type DeprecatedMetrics struct {
	// req-resp sync, no longer used
	PayloadsQuarantineTotal prometheus.Gauge
	P2PReqDurationSeconds   *prometheus.HistogramVec
	P2PReqTotal             *prometheus.CounterVec
	P2PPayloadByNumber      *prometheus.GaugeVec

	// Protocol version reporting, no longer used
	// Delta = params.ProtocolVersionComparison
	ProtocolVersionDelta *prometheus.GaugeVec
	// ProtocolVersions is pseudo-metric to report the exact protocol version info
	ProtocolVersions *prometheus.GaugeVec
}
