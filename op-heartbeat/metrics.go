package op_heartbeat

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum-optimism/optimism/op-node/heartbeat"
)

const (
	MetricsNamespace     = "op_heartbeat"
	MinHeartbeatInterval = 10*time.Minute - 10*time.Second
	UsersCacheSize       = 10_000
)

type Metrics interface {
	RecordHeartbeat(payload heartbeat.Payload, ip string)
	RecordVersion(version string)
}

type metrics struct {
	heartbeats *prometheus.CounterVec
	version    *prometheus.GaugeVec
	sameIP     *prometheus.HistogramVec

	// Groups heartbeats per unique IP, version and chain ID combination.
	// string(IP ++ version ++ chainID) -> *heartbeatEntry
	heartbeatUsers *lru.Cache[string, *heartbeatEntry]
}

type heartbeatEntry struct {
	// Count number of heartbeats per interval, atomically updated
	Count uint64
	// Changes once per heartbeat interval
	Time time.Time
}

func NewMetrics(r *prometheus.Registry) Metrics {
	lruCache, _ := lru.New[string, *heartbeatEntry](UsersCacheSize)
	m := &metrics{
		heartbeats: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "heartbeats",
			Help:      "Counts number of heartbeats by chain ID, version and filtered to unique IPs",
		}, []string{
			"chain_id",
			"version",
		}),
		version: promauto.With(r).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "version",
			Help:      "version pseudo-metrics",
		}, []string{
			"version",
		}),
		sameIP: promauto.With(r).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Name:      "heartbeat_same_ip",
			Buckets:   []float64{1, 2, 4, 8, 16, 32, 64, 128},
			Help:      "Histogram of events within same heartbeat interval per unique IP, by chain ID and version",
		}, []string{
			"chain_id",
			"version",
		}),
		heartbeatUsers: lruCache,
	}
	return m
}

func (m *metrics) RecordHeartbeat(payload heartbeat.Payload, ip string) {
	var chainID string
	if AllowedChainIDs[payload.ChainID] {
		chainID = strconv.FormatUint(payload.ChainID, 10)
	} else {
		chainID = "unknown"
	}
	var version string
	if AllowedVersions[payload.Version] {
		version = payload.Version
	} else {
		version = "unknown"
	}

	key := fmt.Sprintf("%s;%s;%s", ip, version, chainID)
	now := time.Now()
	entry, ok, _ := m.heartbeatUsers.PeekOrAdd(key, &heartbeatEntry{Time: now, Count: 1})
	if !ok {
		// if it's a new entry, observe it and exit.
		m.sameIP.WithLabelValues(chainID, version).Observe(1)
		m.heartbeats.WithLabelValues(chainID, version).Inc()
		return
	}

	if now.Sub(entry.Time) < MinHeartbeatInterval {
		// if the span is still going, then add it up
		atomic.AddUint64(&entry.Count, 1)
	} else {
		// if the span ended, then meter it, and reset it
		m.sameIP.WithLabelValues(chainID, version).Observe(float64(atomic.LoadUint64(&entry.Count)))
		entry.Time = now
		atomic.StoreUint64(&entry.Count, 1)

		m.heartbeats.WithLabelValues(chainID, version).Inc()
	}

	// always add, to keep LRU accurate
	m.heartbeatUsers.Add(key, entry)
}

func (m *metrics) RecordVersion(version string) {
	m.version.WithLabelValues(version).Set(1)
}
