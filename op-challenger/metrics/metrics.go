package metrics

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_challenger"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	// Record Tx metrics
	txmetrics.TxMetricer

	RecordGameStep()
	RecordGameMove()
	RecordCannonExecutionTime(t float64)
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	txmetrics.TxMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	moves               prometheus.Counter
	steps               prometheus.Counter
	cannonExecutionTime prometheus.Histogram
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics() *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

		TxMetrics: txmetrics.MakeTxMetrics(Namespace, factory),

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "up",
			Help:      "1 if the op-challenger has finished starting up",
		}),
		moves: factory.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "moves",
			Help:      "Number of game moves made by the challenge agent",
		}),
		steps: factory.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "steps",
			Help:      "Number of game steps made by the challenge agent",
		}),
		cannonExecutionTime: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "cannon_execution_time",
			Help:      "Time (in seconds) to execute cannon",
			Buckets:   append([]float64{1.0, 10.0}, prometheus.ExponentialBuckets(30.0, 2.0, 14)...),
		}),
	}
}

func (m *Metrics) Serve(ctx context.Context, host string, port int) error {
	return opmetrics.ListenAndServe(ctx, m.registry, host, port)
}

func (m *Metrics) StartBalanceMetrics(ctx context.Context, l log.Logger, client *ethclient.Client, account common.Address) {
	opmetrics.LaunchBalanceMetrics(ctx, l, m.registry, m.ns, client, account)
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the op-proposer.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.up.Set(1)
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

func (m *Metrics) RecordGameMove() {
	m.moves.Add(1)
}

func (m *Metrics) RecordGameStep() {
	m.steps.Add(1)
}

func (m *Metrics) RecordCannonExecutionTime(t float64) {
	m.cannonExecutionTime.Observe(t)
}
