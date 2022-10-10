package metrics

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsNamespace = "indexer"

type Metrics struct {
	SyncHeight *prometheus.GaugeVec

	DepositsCount *prometheus.CounterVec

	WithdrawalsCount *prometheus.CounterVec

	StateBatchesCount prometheus.Counter

	L1CatchingUp prometheus.Gauge

	L2CatchingUp prometheus.Gauge

	SyncPercent *prometheus.GaugeVec

	UpdateDuration *prometheus.SummaryVec

	CachedTokensCount *prometheus.CounterVec

	HTTPRequestsCount prometheus.Counter

	HTTPResponsesCount *prometheus.CounterVec

	HTTPRequestDurationSecs prometheus.Summary

	tokenAddrs map[string]string
}

func NewMetrics(monitoredTokens map[string]string) *Metrics {
	mts := make(map[string]string)
	mts["0x0000000000000000000000000000000000000000"] = "ETH"
	for addr, symbol := range monitoredTokens {
		mts[addr] = symbol
	}

	return &Metrics{
		SyncHeight: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name:      "sync_height",
			Help:      "The max height of the indexer's last batch of L1/L1 blocks.",
			Namespace: metricsNamespace,
		}, []string{
			"chain",
		}),

		DepositsCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:      "deposits_count",
			Help:      "The number of deposits indexed.",
			Namespace: metricsNamespace,
		}, []string{
			"symbol",
		}),

		WithdrawalsCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:      "withdrawals_count",
			Help:      "The number of withdrawals indexed.",
			Namespace: metricsNamespace,
		}, []string{
			"symbol",
		}),

		StateBatchesCount: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "state_batches_count",
			Help:      "The number of state batches indexed.",
			Namespace: metricsNamespace,
		}),

		L1CatchingUp: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "l1_catching_up",
			Help:      "Whether or not L1 is far behind the chain tip.",
			Namespace: metricsNamespace,
		}),

		L2CatchingUp: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "l2_catching_up",
			Help:      "Whether or not L2 is far behind the chain tip.",
			Namespace: metricsNamespace,
		}),

		SyncPercent: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name:      "sync_percent",
			Help:      "Sync percentage for each chain.",
			Namespace: metricsNamespace,
		}, []string{
			"chain",
		}),

		UpdateDuration: promauto.NewSummaryVec(prometheus.SummaryOpts{
			Name:       "update_duration_seconds",
			Help:       "How long each update took.",
			Namespace:  metricsNamespace,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}, []string{
			"chain",
		}),

		CachedTokensCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:      "cached_tokens_count",
			Help:      "How many tokens are in the cache",
			Namespace: metricsNamespace,
		}, []string{
			"chain",
		}),

		HTTPRequestsCount: promauto.NewCounter(prometheus.CounterOpts{
			Name:      "http_requests_count",
			Help:      "How many HTTP requests this instance has seen",
			Namespace: metricsNamespace,
		}),

		HTTPResponsesCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:      "http_responses_count",
			Help:      "How many HTTP responses this instance has served",
			Namespace: metricsNamespace,
		}, []string{
			"status_code",
		}),

		HTTPRequestDurationSecs: promauto.NewSummary(prometheus.SummaryOpts{
			Name:       "http_request_duration_secs",
			Help:       "How long each HTTP request took",
			Namespace:  metricsNamespace,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}),

		tokenAddrs: mts,
	}
}

func (m *Metrics) SetL1SyncHeight(height uint64) {
	m.SyncHeight.WithLabelValues("l1").Set(float64(height))
}

func (m *Metrics) SetL2SyncHeight(height uint64) {
	m.SyncHeight.WithLabelValues("l2").Set(float64(height))
}

func (m *Metrics) RecordDeposit(addr common.Address) {
	sym := m.tokenAddrs[addr.String()]
	if sym == "" {
		sym = "UNKNOWN"
	}

	m.DepositsCount.WithLabelValues(sym).Inc()
}

func (m *Metrics) RecordWithdrawal(addr common.Address) {
	sym := m.tokenAddrs[addr.String()]
	if sym == "" {
		sym = "UNKNOWN"
	}

	m.WithdrawalsCount.WithLabelValues(sym).Inc()
}

func (m *Metrics) RecordStateBatches(count int) {
	m.StateBatchesCount.Add(float64(count))
}

func (m *Metrics) SetL1CatchingUp(state bool) {
	var catchingUp float64
	if state {
		catchingUp = 1
	}
	m.L1CatchingUp.Set(catchingUp)
}

func (m *Metrics) SetL2CatchingUp(state bool) {
	var catchingUp float64
	if state {
		catchingUp = 1
	}
	m.L2CatchingUp.Set(catchingUp)
}

func (m *Metrics) SetL1SyncPercent(height uint64, head uint64) {
	m.SyncPercent.WithLabelValues("l1").Set(float64(height) / float64(head))
}

func (m *Metrics) SetL2SyncPercent(height uint64, head uint64) {
	m.SyncPercent.WithLabelValues("l2").Set(float64(height) / float64(head))
}

func (m *Metrics) IncL1CachedTokensCount() {
	m.CachedTokensCount.WithLabelValues("l1").Inc()
}

func (m *Metrics) IncL2CachedTokensCount() {
	m.CachedTokensCount.WithLabelValues("l2").Inc()
}

func (m *Metrics) RecordHTTPRequest() {
	m.HTTPRequestsCount.Inc()
}

func (m *Metrics) RecordHTTPResponse(code int, dur time.Duration) {
	m.HTTPResponsesCount.WithLabelValues(strconv.Itoa(code)).Inc()
	m.HTTPRequestDurationSecs.Observe(float64(dur) / float64(time.Second))
}

func (m *Metrics) Serve(hostname string, port uint64) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := new(http.Server)
	srv.Addr = net.JoinHostPort(hostname, strconv.FormatUint(port, 10))
	srv.Handler = mux
	err := srv.ListenAndServe()
	return srv, err
}
