package metrics

import (
	"io"

	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_challenger"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer

	// Record Tx metrics
	txmetrics.TxMetricer

	// Record cache metrics
	caching.Metrics

	RecordGameStep()
	RecordGameMove()
	RecordCannonExecutionTime(t float64)

	RecordGamesStatus(inProgress, defenderWon, challengerWon int)

	RecordGameUpdateScheduled()
	RecordGameUpdateCompleted()

	IncActiveExecutors()
	DecActiveExecutors()
	IncIdleExecutors()
	DecIdleExecutors()
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	txmetrics.TxMetrics

	*opmetrics.CacheMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	executors prometheus.GaugeVec

	moves prometheus.Counter
	steps prometheus.Counter

	cannonExecutionTime prometheus.Histogram

	trackedGames  prometheus.GaugeVec
	inflightGames prometheus.Gauge
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

		CacheMetrics: opmetrics.NewCacheMetrics(factory, Namespace, "provider_cache", "Provider cache"),

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
		executors: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "executors",
			Help:      "Number of active and idle executors",
		}, []string{
			"status",
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
			Buckets: append(
				[]float64{1.0, 10.0},
				prometheus.ExponentialBuckets(30.0, 2.0, 14)...),
		}),
		trackedGames: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "tracked_games",
			Help:      "Number of games being tracked by the challenger",
		}, []string{
			"status",
		}),
		inflightGames: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "inflight_games",
			Help:      "Number of games being tracked by the challenger",
		}),
	}
}

func (m *Metrics) Start(host string, port int) (*httputil.HTTPServer, error) {
	return opmetrics.StartServer(m.registry, host, port)
}

func (m *Metrics) StartBalanceMetrics(
	l log.Logger,
	client *ethclient.Client,
	account common.Address,
) io.Closer {
	return opmetrics.LaunchBalanceMetrics(l, m.registry, m.ns, client, account)
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

func (m *Metrics) IncActiveExecutors() {
	m.executors.WithLabelValues("active").Inc()
}

func (m *Metrics) DecActiveExecutors() {
	m.executors.WithLabelValues("active").Dec()
}

func (m *Metrics) IncIdleExecutors() {
	m.executors.WithLabelValues("idle").Inc()
}

func (m *Metrics) DecIdleExecutors() {
	m.executors.WithLabelValues("idle").Dec()
}

func (m *Metrics) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	m.trackedGames.WithLabelValues("in_progress").Set(float64(inProgress))
	m.trackedGames.WithLabelValues("defender_won").Set(float64(defenderWon))
	m.trackedGames.WithLabelValues("challenger_won").Set(float64(challengerWon))
}

func (m *Metrics) RecordGameUpdateScheduled() {
	m.inflightGames.Add(1)
}

func (m *Metrics) RecordGameUpdateCompleted() {
	m.inflightGames.Sub(1)
}
