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
)

const Namespace = "op_dispute_mon"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
	RecordGameAgreement(status string, count int)

	caching.Metrics
}

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	*opmetrics.CacheMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	trackedGames   prometheus.GaugeVec
	gamesAgreement prometheus.GaugeVec
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics() *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

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
		trackedGames: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "tracked_games",
			Help:      "Number of games being tracked by the challenger",
		}, []string{
			"status",
		}),
		gamesAgreement: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "games_agreement",
			Help:      "Number of games broken down by whether the result agrees with the reference node",
		}, []string{
			"status",
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

func (m *Metrics) RecordGamesStatus(inProgress, defenderWon, challengerWon int) {
	m.trackedGames.WithLabelValues("in_progress").Set(float64(inProgress))
	m.trackedGames.WithLabelValues("defender_won").Set(float64(defenderWon))
	m.trackedGames.WithLabelValues("challenger_won").Set(float64(challengerWon))
}

func (m *Metrics) RecordGameAgreement(status string, count int) {
	m.gamesAgreement.WithLabelValues(status).Set(float64(count))
}
