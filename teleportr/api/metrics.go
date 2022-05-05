package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const TeleportrAPINamespace = "teleportr_api"

var (
	rpcRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: TeleportrAPINamespace,
		Name:      "rpc_requests_total",
		Help:      "Count of total client RPC requests.",
	})
	httpResponseCodesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: TeleportrAPINamespace,
		Name:      "http_response_codes_total",
		Help:      "Count of total HTTP response codes.",
	}, []string{
		"status_code",
	})
	httpRequestDurationSumm = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:  TeleportrAPINamespace,
		Name:       "http_request_duration_seconds",
		Help:       "Summary of HTTP request durations, in seconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})
	databaseErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: TeleportrAPINamespace,
		Name:      "database_errors_total",
		Help:      "Count of total database failures.",
	}, []string{
		"method",
	})
	rpcErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: TeleportrAPINamespace,
		Name:      "rpc_errors_total",
		Help:      "Count of total L1 rpc failures.",
	}, []string{
		"method",
	})
)
