package p2p

import (
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/log"
)

// TODO: add metrics here as well

type notifications struct {
	log log.Logger
}

func (notif *notifications) Listen(n network.Network, a ma.Multiaddr) {
	notif.log.Info("started listening network address", "addr", a)
}
func (notif *notifications) ListenClose(n network.Network, a ma.Multiaddr) {
	notif.log.Info("stopped listening network address", "addr", a)
}
func (notif *notifications) Connected(n network.Network, v network.Conn) {
	notif.log.Info("connected to peer", "peer", v.RemotePeer(), "addr", v.RemoteMultiaddr())
}
func (notif *notifications) Disconnected(n network.Network, v network.Conn) {
	notif.log.Info("disconnected from peer", "peer", v.RemotePeer(), "addr", v.RemoteMultiaddr())
}
func (notif *notifications) OpenedStream(n network.Network, v network.Stream) {
	c := v.Conn()
	notif.log.Trace("opened stream", "protocol", v.Protocol(), "peer", c.RemotePeer(), "addr", c.RemoteMultiaddr())
}
func (notif *notifications) ClosedStream(n network.Network, v network.Stream) {
	c := v.Conn()
	notif.log.Trace("opened stream", "protocol", v.Protocol(), "peer", c.RemotePeer(), "addr", c.RemoteMultiaddr())
}

func NewNetworkNotifier(log log.Logger) network.Notifiee {
	return &notifications{log: log}
}
