package metrics

import (
	fmt "fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MetricsNamespace = "ufm"
)

var (
	Debug bool

	errorsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "errors_total",
		Help:      "Count of errors",
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

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z ]+`)

func RecordError(provider string, errorLabel string) {
	if Debug {
		log.Debug("metric inc",
			"m", "errors_total",
			"provider", provider,
			"error", errorLabel)
	}
	errorsTotal.WithLabelValues(provider, errorLabel).Inc()
}

// RecordErrorDetails concats the error message to the label removing non-alpha chars
func RecordErrorDetails(provider string, label string, err error) {
	errClean := nonAlphanumericRegex.ReplaceAllString(err.Error(), "")
	errClean = strings.ReplaceAll(errClean, " ", "_")
	errClean = strings.ReplaceAll(errClean, "__", "_")
	label = fmt.Sprintf("%s.%s", label, errClean)
	RecordError(provider, label)
}

func RecordRPCLatency(provider string, client string, method string, latency time.Duration) {
	if Debug {
		log.Debug("metric set",
			"m", "rpc_latency",
			"provider", provider,
			"client", client,
			"method", method,
			"latency", latency)
	}
	rpcLatency.WithLabelValues(provider, client, method).Set(float64(latency.Milliseconds()))
}

func RecordRoundTripLatency(provider string, latency time.Duration) {
	if Debug {
		log.Debug("metric set",
			"m", "roundtrip_latency",
			"provider", provider,
			"latency", latency)
	}
	roundTripLatency.WithLabelValues(provider).Set(float64(latency.Milliseconds()))
}

func RecordGasUsed(provider string, val uint64) {
	if Debug {
		log.Debug("metric add",
			"m", "gas_used",
			"provider", provider,
			"val", val)
	}
	gasUsed.WithLabelValues(provider).Set(float64(val))
}

func RecordFirstSeenLatency(providerSource string, providerSeen string, latency time.Duration) {
	if Debug {
		log.Debug("metric set",
			"m", "first_seen_latency",
			"provider_source", providerSource,
			"provider_seen", providerSeen,
			"latency", latency)
	}
	firstSeenLatency.WithLabelValues(providerSource, providerSeen).Set(float64(latency.Milliseconds()))
}

func RecordProviderToProviderLatency(providerSource string, providerSeen string, latency time.Duration) {
	if Debug {
		log.Debug("metric set",
			"m", "provider_to_provider_latency",
			"provider_source", providerSource,
			"provider_seen", providerSeen,
			"latency", latency)
	}
	providerToProviderLatency.WithLabelValues(providerSource, providerSeen).Set(float64(latency.Milliseconds()))
}

func RecordTransactionsInFlight(network string, count int) {
	if Debug {
		log.Debug("metric set",
			"m", "transactions_inflight",
			"network", network,
			"count", count)
	}
	networkTransactionsInFlight.WithLabelValues(network).Set(float64(count))
}
