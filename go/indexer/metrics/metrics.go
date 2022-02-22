package metrics

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const metricsNamespace = "indexer"

type Metrics struct {
	L1SyncHeight prometheus.Gauge

	L2SyncHeight prometheus.Gauge

	DepositsCount *prometheus.CounterVec

	WithdrawalsCount *prometheus.CounterVec

	L1CatchingUp prometheus.Gauge

	L2CatchingUp prometheus.Gauge

	SyncPercent *prometheus.GaugeVec

	UpdateDuration *prometheus.SummaryVec

	tokenAddrs map[string]string
}

func NewMetrics(monitoredTokens map[string]string) *Metrics {
	mts := make(map[string]string)
	mts["0x0000000000000000000000000000000000000000"] = "ETH"
	for addr, symbol := range monitoredTokens {
		mts[addr] = symbol
	}

	return &Metrics{
		L1SyncHeight: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "l1_sync_height",
			Help:      "The max height of the indexer's last batch of L1 blocks.",
			Namespace: metricsNamespace,
		}),

		L2SyncHeight: promauto.NewGauge(prometheus.GaugeOpts{
			Name:      "l2_sync_height",
			Help:      "The max height of the indexer's last batch of L2 blocks.",
			Namespace: metricsNamespace,
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
			Name: "sync_percent",
			Help: "Sync percentage for each chain.",
			Namespace: metricsNamespace,
		}, []string{
			"chain",
		}),

		UpdateDuration: promauto.NewSummaryVec(prometheus.SummaryOpts{
			Name: "update_duration",
			Help: "How long each update took.",
		}, []string{
			"chain",
		}),

		tokenAddrs: mts,
	}
}

func (m *Metrics) SetL1SyncHeight(height uint64) {
	m.L1SyncHeight.Set(float64(height))
}

func (m *Metrics) SetL2SyncHeight(height uint64) {
	m.L2SyncHeight.Set(float64(height))
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

func (m * Metrics) SetL1SyncPercent(height uint64, head uint64) {
	m.SyncPercent.WithLabelValues("l1").Set(float64(height) / float64(head))
}

func (m * Metrics) SetL2SyncPercent(height uint64, head uint64) {
	m.SyncPercent.WithLabelValues("l2").Set(float64(height) / float64(head))
}

func (m *Metrics) Serve(hostname string, port uint64) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := new(http.Server)
	srv.Addr = fmt.Sprintf("%s:%d", hostname, port)
	srv.Handler = mux
	err := srv.ListenAndServe()
	return srv, err
}
