package node

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricsNamespace = "op_indexer_rpc"
	batchMethod      = "<batch>"
)

type Metricer interface {
	RecordRPCClientRequest(method string) func(err error)
	RecordRPCClientBatchRequest(b []rpc.BatchElem) func(err error)
}

type clientMetrics struct {
	rpcClientRequestsTotal          *prometheus.CounterVec
	rpcClientRequestDurationSeconds *prometheus.HistogramVec
	rpcClientResponsesTotal         *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry, subsystem string) Metricer {
	factory := metrics.With(registry)
	return &clientMetrics{
		rpcClientRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total RPC requests initiated by the RPC client",
		}, []string{
			"method",
		}),
		rpcClientRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC client request durations",
		}, []string{
			"method",
		}),
		rpcClientResponsesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      "responses_total",
			Help:      "Total RPC request responses received by the RPC client",
		}, []string{
			"method",
			"error",
		}),
	}
}

func (m *clientMetrics) RecordRPCClientRequest(method string) func(err error) {
	m.rpcClientRequestsTotal.WithLabelValues(method).Inc()
	timer := prometheus.NewTimer(m.rpcClientRequestDurationSeconds.WithLabelValues(method))
	return func(err error) {
		m.recordRPCClientResponse(method, err)
		timer.ObserveDuration()
	}
}

func (m *clientMetrics) RecordRPCClientBatchRequest(b []rpc.BatchElem) func(err error) {
	m.rpcClientRequestsTotal.WithLabelValues(batchMethod).Add(float64(len(b)))
	for _, elem := range b {
		m.rpcClientRequestsTotal.WithLabelValues(elem.Method).Inc()
	}

	timer := prometheus.NewTimer(m.rpcClientRequestDurationSeconds.WithLabelValues(batchMethod))
	return func(err error) {
		m.recordRPCClientResponse(batchMethod, err)
		timer.ObserveDuration()

		// Record errors for individual requests
		for _, elem := range b {
			m.recordRPCClientResponse(elem.Method, elem.Error)
		}
	}
}

func (m *clientMetrics) recordRPCClientResponse(method string, err error) {
	var errStr string
	var rpcErr rpc.Error
	var httpErr rpc.HTTPError
	if err == nil {
		errStr = "<nil>"
	} else if errors.As(err, &rpcErr) {
		errStr = fmt.Sprintf("rpc_%d", rpcErr.ErrorCode())
	} else if errors.As(err, &httpErr) {
		errStr = fmt.Sprintf("http_%d", httpErr.StatusCode)
	} else if errors.Is(err, ethereum.NotFound) {
		errStr = "<not found>"
	} else {
		errStr = "<unknown>"
	}
	m.rpcClientResponsesTotal.WithLabelValues(method, errStr).Inc()
}
