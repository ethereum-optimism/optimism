package proxyd

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	MetricsNamespace = "proxyd"

	RPCRequestSourceHTTP = "http"
	RPCRequestSourceWS   = "ws"

	SourceClient  = "client"
	SourceBackend = "backend"
	SourceProxyd  = "proxyd"
)

var (
	rpcRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "rpc_requests_total",
		Help:      "Count of total client RPC requests.",
	})

	rpcForwardsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "rpc_backend_requests_total",
		Help:      "Count of total RPC requests forwarded to each backend.",
	}, []string{
		"backend_name",
		"method_name",
		"source",
	})

	rpcErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "rpc_errors_total",
		Help:      "Count of total RPC errors.",
	}, []string{
		"source",
		"error_code",
	})

	rpcBackendRequestDurationSumm = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  MetricsNamespace,
		Name:       "rpc_backend_request_duration_seconds",
		Help:       "Summary of backend response times broken down by backend and method name.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	}, []string{
		"backend_name",
		"method_name",
	})

	activeClientWsConnsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "active_client_ws_conns",
		Help:      "Gauge of active client WS connections.",
	})

	activeBackendWsConnsGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "active_backend_ws_conns",
		Help:      "Gauge of active backend WS connections.",
	}, []string{
		"backend_name",
	})

	unserviceableRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "unserviceable_requests_total",
		Help:      "Count of total requests that were rejected due to no backends being available.",
	}, []string{
		"source",
	})

	httpRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "http_requests_total",
		Help:      "Count of total HTTP requests.",
	})

	httpRequestDurationSumm = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:  MetricsNamespace,
		Name:       "http_request_duration_seconds",
		Help:       "Summary of HTTP request durations, in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})

	wsMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "ws_messages_total",
		Help:      "Count of total websocket messages including protocol control.",
	}, []string{
		"backend_name",
		"source",
	})

	redisErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "redis_errors_total",
		Help:      "Count of total Redis errors.",
	}, []string{
		"source",
	})
)

func RecordRPCError(source string, err error) {
	rpcErr, ok := err.(*RPCErr)
	var code int
	if ok {
		code = rpcErr.Code
	} else {
		code = -1
	}

	rpcErrorsTotal.WithLabelValues(source, strconv.Itoa(code)).Inc()
}

func RecordRedisError(source string) {
	redisErrorsTotal.WithLabelValues(source).Inc()
}

func RecordWSMessage(backendName, source string) {
	wsMessagesTotal.WithLabelValues(backendName, source).Inc()
}

func RecordUnserviceableRequest(source string) {
	unserviceableRequestsTotal.WithLabelValues(source).Inc()
}
