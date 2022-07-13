package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	Namespace = "op_node"

	RPCServerSubsystem = "rpc_server"
	RPCClientSubsystem = "rpc_client"

	BatchMethod = "<batch>"
)

type Metrics struct {
	Info                            *prometheus.GaugeVec
	Up                              prometheus.Gauge
	RPCServerRequestsTotal          *prometheus.CounterVec
	RPCServerRequestDurationSeconds *prometheus.HistogramVec
	RPCClientRequestsTotal          *prometheus.CounterVec
	RPCClientRequestDurationSeconds *prometheus.HistogramVec
	RPCClientResponsesTotal         *prometheus.CounterVec

	registry *prometheus.Registry
}

func NewMetrics(procName string) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())
	return &Metrics{
		Info: promauto.With(registry).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		Up: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op node has finished starting up",
		}),
		RPCServerRequestsTotal: promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "requests_total",
			Help:      "Total requests to the RPC server",
		}, []string{
			"method",
		}),
		RPCServerRequestDurationSeconds: promauto.With(registry).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC server request durations",
		}, []string{
			"method",
		}),
		RPCClientRequestsTotal: promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "requests_total",
			Help:      "Total RPC requests initiated by the opnode's RPC client",
		}, []string{
			"method",
		}),
		RPCClientRequestDurationSeconds: promauto.With(registry).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC client request durations",
		}, []string{
			"method",
		}),
		RPCClientResponsesTotal: promauto.With(registry).NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "responses_total",
			Help:      "Total RPC request responses received by the opnode's RPC client",
		}, []string{
			"method",
			"error",
		}),
		registry: registry,
	}
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the opnode.
func (m *Metrics) RecordInfo(version string) {
	m.Info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.Up.Set(1)
}

// RecordRPCServerRequest is a helper method to record an incoming RPC
// call to the opnode's RPC server. It bumps the requests metric,
// and tracks how long it takes to serve a response.
func (m *Metrics) RecordRPCServerRequest(method string) func() {
	m.RPCServerRequestsTotal.WithLabelValues(method).Inc()
	timer := prometheus.NewTimer(m.RPCServerRequestDurationSeconds.WithLabelValues(method))
	return func() {
		timer.ObserveDuration()
	}
}

// RecordRPCClientRequest is a helper method to record an RPC client
// request. It bumps the requests metric, tracks the response
// duration, and records the response's error code.
func (m *Metrics) RecordRPCClientRequest(method string) func(err error) {
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
func (m *Metrics) RecordRPCClientResponse(method string, err error) {
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

// Serve starts the metrics server on the given hostname and port.
// The server will be closed when the passed-in context is cancelled.
func (m *Metrics) Serve(ctx context.Context, hostname string, port int) error {
	addr := net.JoinHostPort(hostname, strconv.Itoa(port))
	server := &http.Server{
		Addr: addr,
		Handler: promhttp.InstrumentMetricHandler(
			m.registry, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}),
		),
	}
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	return server.ListenAndServe()
}
