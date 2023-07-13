package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MetricsNamespace = "ufm"
)

var (
	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "errors_total",
		Help:      "Count of errors.",
	}, []string{
		"provider",
		"error",
	})

	rpcLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "rpc_latency",
		Help:      "RPC latency per provider, client and method (ms)",
	}, []string{
		"provider",
		"client",
		"method",
	})

	roundTripLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "roundtrip_latency",
		Help:      "Round trip latency per provider (ms)",
	}, []string{
		"provider",
	})

	gasUsed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "gas_used",
		Help:      "Gas used per provider",
	}, []string{
		"provider",
	})

	firstSeenLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "first_seen_latency",
		Help:      "First seen latency latency per provider (ms)",
	}, []string{
		"provider_source",
		"provider_seen",
	})

	providerToProviderLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "provider_to_provider_latency",
		Help:      "Provider to provider latency (ms)",
	}, []string{
		"provider_source",
		"provider_seen",
	})

	networkTransactionsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "transactions_inflight",
		Help:      "Transactions in flight, per network",
	}, []string{
		"network",
	})
)

func RecordError(provider string, errorLabel string) {
	errorsTotal.WithLabelValues(provider, errorLabel).Inc()
}

func RecordRPCLatency(provider string, client string, method string, latency time.Duration) {
	rpcLatency.WithLabelValues(provider, client, method).Set(float64(latency.Milliseconds()))
}

func RecordRoundTripLatency(provider string, latency time.Duration) {
	roundTripLatency.WithLabelValues(provider).Set(float64(latency.Milliseconds()))
}

func RecordGasUsed(provider string, val uint64) {
	gasUsed.WithLabelValues(provider).Set(float64(val))
}

func RecordFirstSeenLatency(provider_source string, provider_seen string, latency time.Duration) {
	firstSeenLatency.WithLabelValues(provider_source, provider_seen).Set(float64(latency.Milliseconds()))
}

func RecordProviderToProviderLatency(provider_source string, provider_seen string, latency time.Duration) {
	firstSeenLatency.WithLabelValues(provider_source, provider_seen).Set(float64(latency.Milliseconds()))
}

func RecordTransactionsInFlight(network string, count int) {
	networkTransactionsInFlight.WithLabelValues(network).Set(float64(count))
}
