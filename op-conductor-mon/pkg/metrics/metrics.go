package metrics

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MetricsNamespace = "op-conductor-mon"
)

var (
	Debug                bool
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z ]+`)

	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "errors_total",
		Help:      "Count of errors",
	}, []string{
		"error",
		"method",
		"node",
	})

	rpcLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "rpc_latency",
		Help:      "RPC latency per network, node and method (ms)",
	}, []string{
		"node",
		"method",
	})

	nodeState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "node_state",
		Help:      "State per node (bool)",
	}, []string{
		"node",
		"state",
	})

	nodeLeader = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "node_leader",
		Help:      "Leader according to a node",
	}, []string{
		"node",
		"leader",
	})

	clusterMembershipCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Name:      "cluster_membership_count",
		Help:      "Cluster membership counnt according to a node",
	}, []string{
		"node",
	})
)

func errLabel(err error) string {
	errClean := nonAlphanumericRegex.ReplaceAllString(err.Error(), "")
	errClean = strings.ReplaceAll(errClean, " ", "_")
	errClean = strings.ReplaceAll(errClean, "__", "_")
	return errClean
}

func RecordError(error string) {
	if Debug {
		log.Debug("metric inc",
			"m", "errors_total",
			"error", error)
	}
	errorsTotal.WithLabelValues(error).Inc()
}

// RecordErrorDetails concats the error message to the label removing non-alpha chars
func RecordErrorDetails(label string, err error) {
	label = fmt.Sprintf("%s.%s", label, errLabel(err))
	RecordError(label)
}

func RecordNetworkErrorDetails(node string, method string, err error) {
	if Debug {
		log.Debug("metric inc",
			"m", "errors_total",
			"node", node,
			"error", errLabel(err))
	}
	errorsTotal.WithLabelValues(errLabel(err), method, node).Inc()
}

func RecordRPCLatency(node string, method string, latency time.Duration) {
	if Debug {
		log.Debug("metric set",
			"m", "rpc_latency",
			"node", node,
			"method", method,
			"latency", latency)
	}
	rpcLatency.WithLabelValues(node, method).Set(float64(latency.Milliseconds()))
}

func RecordNodeState(node string, state string, val bool) {
	if Debug {
		log.Debug("metric set",
			"m", "node_state",
			"node", node,
			"state", state,
			"val", val)
	}
	nodeState.WithLabelValues(node, state).Set(boolToFloat64(val))
}

func ReportNodeLeader(node string, leader string) {
	if Debug {
		log.Debug("metric set",
			"m", "node_leader",
			"node", node,
			"leader", leader)
	}
	nodeLeader.WithLabelValues(node, leader).Set(1)
}

func ReportClusterMembershipCount(node string, len int) {
	if Debug {
		log.Debug("metric set",
			"m", "cluster_membership_count",
			"node", node,
			"len", len)
	}
	clusterMembershipCount.WithLabelValues(node).Set(float64(len))
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
