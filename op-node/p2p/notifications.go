package p2p

import (
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/log"
)

// TODO: add metrics here as well

type notifications struct {
	log log.Logger
	m   *metrics.Metrics
}

func (notif *notifications) Listen(n network.Network, a ma.Multiaddr) {
	notif.log.Info("started listening network address", "addr", a)
}
func (notif *notifications) ListenClose(n network.Network, a ma.Multiaddr) {
	notif.log.Info("stopped listening network address", "addr", a)
}
func (notif *notifications) Connected(n network.Network, v network.Conn) {
	notif.m.PeerCount.Inc()
	notif.log.Info("connected to peer", "peer", v.RemotePeer(), "addr", v.RemoteMultiaddr())
}
func (notif *notifications) Disconnected(n network.Network, v network.Conn) {
	notif.m.PeerCount.Dec()
	notif.log.Info("disconnected from peer", "peer", v.RemotePeer(), "addr", v.RemoteMultiaddr())
}
func (notif *notifications) OpenedStream(n network.Network, v network.Stream) {
	notif.m.StreamCount.Inc()
	c := v.Conn()
	notif.log.Trace("opened stream", "protocol", v.Protocol(), "peer", c.RemotePeer(), "addr", c.RemoteMultiaddr())
}
func (notif *notifications) ClosedStream(n network.Network, v network.Stream) {
	notif.m.StreamCount.Dec()
	c := v.Conn()
	notif.log.Trace("opened stream", "protocol", v.Protocol(), "peer", c.RemotePeer(), "addr", c.RemoteMultiaddr())
}

func NewNetworkNotifier(log log.Logger, metrics *metrics.Metrics) network.Notifiee {
	return &notifications{log: log, m: metrics}
}
