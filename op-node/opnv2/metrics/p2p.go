package metrics

import (
	"context"
	"time"

	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	libp2pmetrics "github.com/libp2p/go-libp2p/core/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type P2PMetricer interface {
	RecordGossipEvent(evType int32)
	IncPeerCount()
	DecPeerCount()
	IncStreamCount()
	DecStreamCount()
	RecordBandwidth(ctx context.Context, bwc *libp2pmetrics.BandwidthCounter)
	SetPeerScores(allScores []store.PeerScores)
	RecordPeerUnban()
	RecordIPUnban()
	RecordDial(allow bool)
	RecordAccept(allow bool)
}

type P2PMetrics struct {
	PeerCount         prometheus.Gauge
	StreamCount       prometheus.Gauge
	GossipEventsTotal *prometheus.CounterVec
	BandwidthTotal    *prometheus.GaugeVec
	PeerUnbans        prometheus.Counter
	IPUnbans          prometheus.Counter
	Dials             *prometheus.CounterVec
	Accepts           *prometheus.CounterVec
	PeerScores        *prometheus.HistogramVec
}

var _ P2PMetricer = (*P2PMetrics)(nil)

func NewP2PMetrics(ns string, factory metrics.Factory) *P2PMetrics {
	return &P2PMetrics{
		PeerCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "peer_count",
			Help:      "Count of currently connected p2p peers",
		}),
		PeerScores: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "peer_scores",
			Help:      "Histogram of currently connected peer scores",
			Buckets:   []float64{-100, -40, -20, -10, -5, -2, -1, -0.5, -0.05, 0, 0.05, 0.5, 1, 2, 5, 10, 20, 40},
		}, []string{"type"}),
		StreamCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "stream_count",
			Help:      "Count of currently connected p2p streams",
		}),
		GossipEventsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "gossip_events_total",
			Help:      "Count of gossip events by type",
		}, []string{
			"type",
		}),
		BandwidthTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "bandwidth_bytes_total",
			Help:      "P2P bandwidth by direction",
		}, []string{
			"direction",
		}),
		PeerUnbans: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "peer_unbans",
			Help:      "Count of peer unbans",
		}),
		IPUnbans: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "ip_unbans",
			Help:      "Count of IP unbans",
		}),
		Dials: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "dials",
			Help:      "Count of outgoing dial attempts, with label to filter to allowed attempts",
		}, []string{"allow"}),
		Accepts: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Subsystem: "p2p",
			Name:      "accepts",
			Help:      "Count of incoming dial attempts to accept, with label to filter to allowed attempts",
		}, []string{"allow"}),
	}
}

// SetPeerScores updates the peer score metrics.
// Accepts a slice of peer scores in any order.
func (m *P2PMetrics) SetPeerScores(allScores []store.PeerScores) {
	for _, scores := range allScores {
		m.PeerScores.WithLabelValues("total").Observe(scores.Gossip.Total)
		m.PeerScores.WithLabelValues("ipColocation").Observe(scores.Gossip.IPColocationFactor)
		m.PeerScores.WithLabelValues("behavioralPenalty").Observe(scores.Gossip.BehavioralPenalty)
		m.PeerScores.WithLabelValues("blocksFirstMessage").Observe(scores.Gossip.Blocks.FirstMessageDeliveries)
		m.PeerScores.WithLabelValues("blocksTimeInMesh").Observe(scores.Gossip.Blocks.TimeInMesh)
		m.PeerScores.WithLabelValues("blocksMessageDeliveries").Observe(scores.Gossip.Blocks.MeshMessageDeliveries)
		m.PeerScores.WithLabelValues("blocksInvalidMessageDeliveries").Observe(scores.Gossip.Blocks.InvalidMessageDeliveries)

		m.PeerScores.WithLabelValues("reqRespValidResponses").Observe(scores.ReqResp.ValidResponses)
		m.PeerScores.WithLabelValues("reqRespErrorResponses").Observe(scores.ReqResp.ErrorResponses)
		m.PeerScores.WithLabelValues("reqRespRejectedPayloads").Observe(scores.ReqResp.RejectedPayloads)
	}
}

func (m *P2PMetrics) RecordGossipEvent(evType int32) {
	m.GossipEventsTotal.WithLabelValues(pb.TraceEvent_Type_name[evType]).Inc()
}

func (m *P2PMetrics) IncPeerCount() {
	m.PeerCount.Inc()
}

func (m *P2PMetrics) DecPeerCount() {
	m.PeerCount.Dec()
}

func (m *P2PMetrics) IncStreamCount() {
	m.StreamCount.Inc()
}

func (m *P2PMetrics) DecStreamCount() {
	m.StreamCount.Dec()
}

func (m *P2PMetrics) RecordBandwidth(ctx context.Context, bwc *libp2pmetrics.BandwidthCounter) {
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			bwTotals := bwc.GetBandwidthTotals()
			m.BandwidthTotal.WithLabelValues("in").Set(float64(bwTotals.TotalIn))
			m.BandwidthTotal.WithLabelValues("out").Set(float64(bwTotals.TotalOut))
		case <-ctx.Done():
			return
		}
	}
}

func (m *P2PMetrics) RecordPeerUnban() {
	m.PeerUnbans.Inc()
}

func (m *P2PMetrics) RecordIPUnban() {
	m.IPUnbans.Inc()
}

func (m *P2PMetrics) RecordDial(allow bool) {
	if allow {
		m.Dials.WithLabelValues("true").Inc()
	} else {
		m.Dials.WithLabelValues("false").Inc()
	}
}

func (m *P2PMetrics) RecordAccept(allow bool) {
	if allow {
		m.Accepts.WithLabelValues("true").Inc()
	} else {
		m.Accepts.WithLabelValues("false").Inc()
	}
}

type NoopP2PMetrics struct{}

var _ P2PMetricer = NoopP2PMetrics{}

func (NoopP2PMetrics) RecordGossipEvent(evType int32) {}

func (NoopP2PMetrics) SetPeerScores(allScores []store.PeerScores) {}

func (NoopP2PMetrics) IncPeerCount() {}

func (NoopP2PMetrics) DecPeerCount() {}

func (NoopP2PMetrics) IncStreamCount() {}

func (NoopP2PMetrics) DecStreamCount() {}

func (NoopP2PMetrics) RecordBandwidth(ctx context.Context, bwc *libp2pmetrics.BandwidthCounter) {}

func (NoopP2PMetrics) RecordPeerUnban() {}

func (NoopP2PMetrics) RecordIPUnban() {}

func (NoopP2PMetrics) RecordDial(allow bool) {}

func (NoopP2PMetrics) RecordAccept(allow bool) {}
