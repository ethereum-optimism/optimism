package metrics

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	RPCServerSubsystem = "rpc_server"
	RPCClientSubsystem = "rpc_client"

	BatchMethod = "<batch>"
)

type RPCClientMetricer interface {
	RecordRPCClientRequest(method string) func(err error)
	RecordRPCClientResponse(method string, err error)
}

type RPCServerMetricer interface {
	RecordRPCServerRequest(method string) func()
}

type RPCMetricer interface {
	RPCClientMetricer
	RPCServerMetricer
}

// RPCMetrics tracks client-only RPC metrics
type RPCClientMetrics struct {
	RPCClientRequestsTotal          *prometheus.CounterVec
	RPCClientRequestDurationSeconds *prometheus.HistogramVec
	RPCClientResponsesTotal         *prometheus.CounterVec
}

// RPCMetrics tracks server-only RPC metrics
type RPCServerMetrics struct {
	RPCServerRequestsTotal          *prometheus.CounterVec
	RPCServerRequestDurationSeconds *prometheus.HistogramVec
}

// RPCMetrics tracks all the RPC metrics, both client & server
type RPCMetrics struct {
	RPCClientMetrics
	RPCServerMetrics
}

// MakeRPCMetrics creates a new RPCMetrics with the given namespace
func MakeRPCMetrics(ns string, factory Factory) RPCMetrics {
	return RPCMetrics{
		RPCClientMetrics: MakeRPCClientMetrics(ns, factory),
		RPCServerMetrics: MakeRPCServerMetrics(ns, factory),
	}
}

// MakeRPCClientMetrics creates a new RPCServerMetrics instance with the given namespace
func MakeRPCClientMetrics(ns string, factory Factory) RPCClientMetrics {
	return RPCClientMetrics{
		RPCClientRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "requests_total",
			Help:      "Total RPC requests initiated by the opnode's RPC client",
		}, []string{
			"method",
		}),
		RPCClientRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC client request durations",
		}, []string{
			"method",
		}),
		RPCClientResponsesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "responses_total",
			Help:      "Total RPC request responses received by the opnode's RPC client",
		}, []string{
			"method",
			"error",
		}),
	}
}

// RecordRPCClientRequest is a helper method to record an RPC client
// request. It bumps the requests metric, tracks the response
// duration, and records the response's error code.
func (m *RPCClientMetrics) RecordRPCClientRequest(method string) func(err error) {
	m.RPCClientRequestsTotal.WithLabelValues(method).Inc()
	timer := prometheus.NewTimer(m.RPCClientRequestDurationSeconds.WithLabelValues(method))
	return func(err error) {
		m.RecordRPCClientResponse(method, err)
		timer.ObserveDuration()
	}
}

// RecordRPCClientResponse records an RPC response. It will
// convert the passed-in error into something metrics friendly.
// Nil errors get converted into <nil>, RPC errors are converted
// into rpc_<error code>, HTTP errors are converted into
// http_<status code>, and everything else is converted into
// <unknown>.
func (m *RPCClientMetrics) RecordRPCClientResponse(method string, err error) {
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
	m.RPCClientResponsesTotal.WithLabelValues(method, errStr).Inc()
}

// MakeRPCServerMetrics creates a new RPCServerMetrics instance with the given namespace
func MakeRPCServerMetrics(ns string, factory Factory) RPCServerMetrics {
	return RPCServerMetrics{
		RPCServerRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "requests_total",
			Help:      "Total requests to the RPC server",
		}, []string{
			"method",
		}),
		RPCServerRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC server request durations",
		}, []string{
			"method",
		}),
	}
}

// RecordRPCServerRequest is a helper method to record an incoming RPC
// call to the opnode's RPC server. It bumps the requests metric,
// and tracks how long it takes to serve a response.
func (m *RPCServerMetrics) RecordRPCServerRequest(method string) func() {
	m.RPCServerRequestsTotal.WithLabelValues(method).Inc()
	timer := prometheus.NewTimer(m.RPCServerRequestDurationSeconds.WithLabelValues(method))
	return func() {
		timer.ObserveDuration()
	}
}

type NoopRPCMetrics struct{}

func (n *NoopRPCMetrics) RecordRPCServerRequest(method string) func() {
	return func() {}
}

func (n *NoopRPCMetrics) RecordRPCClientRequest(method string) func(err error) {
	return func(err error) {}
}
func (n *NoopRPCMetrics) RecordRPCClientResponse(method string, err error) {
}

var _ RPCMetricer = (*NoopRPCMetrics)(nil)
