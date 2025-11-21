package metrics

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	RPCServerSubsystem = "rpc_server"
	RPCClientSubsystem = "rpc_client"
)

type RPCMetricer interface {
	NewRecorder(name string) rpc.Recorder
}

// RPCMetrics tracks all the RPC metrics, both client & server.
type RPCMetrics struct {
	// Legacy client metrics. Do not remove labels or change settings for backward compatibility.
	clientRequestsTotal          *prometheus.CounterVec
	clientRequestDurationSeconds *prometheus.HistogramVec
	clientResponsesTotal         *prometheus.CounterVec

	// Legacy server metrics. Do not remove labels or change settings for backward compatibility.
	serverRequestsTotal          *prometheus.CounterVec
	serverRequestDurationSeconds *prometheus.HistogramVec

	// New metrics
	serverResponsesTotal       *prometheus.CounterVec
	notificationsReceivedTotal *prometheus.CounterVec
	notificationsSentTotal     *prometheus.CounterVec
	clientParamsSizeTotal      *prometheus.CounterVec
	clientResultsSizeTotal     *prometheus.CounterVec
	serverParamsSizeTotal      *prometheus.CounterVec
	serverResultsSizeTotal     *prometheus.CounterVec
}

func (m *RPCMetrics) NewRecorder(name string) rpc.Recorder {
	return &rpcRecorder{m: m, name: name}
}

var _ RPCMetricer = (*RPCMetrics)(nil)

// MakeRPCMetrics creates a new RPCMetrics with the given namespace.
// This struct is intended to be embedded into the larger metrics struct.
func MakeRPCMetrics(ns string, factory Factory) RPCMetrics {
	return RPCMetrics{
		clientRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "requests_total",
			Help:      "Total RPC requests initiated",
		}, []string{
			"rpc",
			"method",
		}),
		clientRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC client request durations",
		}, []string{
			"rpc",
			"method",
		}),
		clientResponsesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "responses_total",
			Help:      "Total RPC request responses received",
		}, []string{
			"rpc",
			"method",
			"error",
		}),
		serverRequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "requests_total",
			Help:      "Total requests to the RPC server",
		}, []string{
			"rpc",
			"method",
		}),
		serverRequestDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "request_duration_seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			Help:      "Histogram of RPC server request durations",
		}, []string{
			"rpc",
			"method",
		}),
		serverResponsesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "responses_total",
			Help:      "Total RPC request responses served",
		}, []string{
			"rpc",
			"method",
			"error",
		}),
		notificationsReceivedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "notifications_received_total",
			Help:      "Total RPC notifications received",
		}, []string{
			"rpc",
			"method",
		}),
		notificationsSentTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "notifications_sent_total",
			Help:      "Total RPC notifications sent",
		}, []string{
			"rpc",
			"method",
		}),
		clientParamsSizeTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "params_size_total",
			Help:      "Total bytes of RPC params sent",
		}, []string{
			"rpc",
			"method",
		}),
		clientResultsSizeTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCClientSubsystem,
			Name:      "results_size_total",
			Help:      "Total bytes of RPC results received",
		}, []string{
			"rpc",
			"method",
		}),
		serverParamsSizeTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "params_size_total",
			Help:      "Total bytes of RPC params received",
		}, []string{
			"rpc",
			"method",
		}),
		serverResultsSizeTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: RPCServerSubsystem,
			Name:      "results_size_total",
			Help:      "Total bytes of RPC results sent back",
		}, []string{
			"rpc",
			"method",
		}),
	}
}

type rpcRecorder struct {
	m    *RPCMetrics
	name string
}

func (rec *rpcRecorder) RecordOutgoing(ctx context.Context, msg rpc.RecordedMsg) rpc.RecordDone {
	if msg.MsgIsNotification() {
		rec.m.notificationsSentTotal.WithLabelValues(rec.name, msg.MsgMethod()).Inc()
		return nil
	}
	rec.m.clientRequestsTotal.WithLabelValues(rec.name, msg.MsgMethod()).Inc()
	rec.m.clientParamsSizeTotal.WithLabelValues(rec.name, msg.MsgMethod()).Add(float64(len(msg.MsgParams())))
	timer := prometheus.NewTimer(rec.m.clientRequestDurationSeconds.WithLabelValues(rec.name, msg.MsgMethod()))
	return func(ctx context.Context, input, output rpc.RecordedMsg) {
		timer.ObserveDuration()
		if output != nil {
			errStr := "<nil>"
			if msgErr := output.MsgError(); msgErr != nil {
				errStr = fmt.Sprintf("rpc_%d", msgErr.ErrorCode())
			} else {
				rec.m.clientResultsSizeTotal.WithLabelValues(rec.name, input.MsgMethod()).Add(float64(len(output.MsgResult())))
			}
			rec.m.clientResponsesTotal.WithLabelValues(rec.name, input.MsgMethod(), errStr).Inc()
		}
	}
}

func (rec *rpcRecorder) RecordIncoming(ctx context.Context, msg rpc.RecordedMsg) rpc.RecordDone {
	if msg.MsgIsNotification() {
		rec.m.notificationsReceivedTotal.WithLabelValues(rec.name, msg.MsgMethod()).Inc()
		return nil
	}
	rec.m.serverRequestsTotal.WithLabelValues(rec.name, msg.MsgMethod()).Inc()
	rec.m.serverParamsSizeTotal.WithLabelValues(rec.name, msg.MsgMethod()).Add(float64(len(msg.MsgParams())))
	timer := prometheus.NewTimer(rec.m.serverRequestDurationSeconds.WithLabelValues(rec.name, msg.MsgMethod()))
	return func(ctx context.Context, input, output rpc.RecordedMsg) {
		timer.ObserveDuration()
		if output != nil {
			errStr := "<nil>"
			if msgErr := output.MsgError(); msgErr != nil {
				errStr = fmt.Sprintf("rpc_%d", msgErr.ErrorCode())
			} else {
				rec.m.serverResultsSizeTotal.WithLabelValues(rec.name, input.MsgMethod()).Add(float64(len(output.MsgResult())))
			}
			rec.m.serverResponsesTotal.WithLabelValues(rec.name, input.MsgMethod(), errStr).Inc()
		}
	}
}

type NoopRPCMetrics struct{}

func (n *NoopRPCMetrics) NewRecorder(name string) rpc.Recorder {
	return &NoopRPCRecorder{}
}

type NoopRPCRecorder struct{}

func (n *NoopRPCRecorder) RecordIncoming(ctx context.Context, msg rpc.RecordedMsg) rpc.RecordDone {
	return nil
}

func (n *NoopRPCRecorder) RecordOutgoing(ctx context.Context, msg rpc.RecordedMsg) rpc.RecordDone {
	return nil
}

var _ RPCMetricer = (*NoopRPCMetrics)(nil)
