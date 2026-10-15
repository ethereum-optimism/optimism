package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// PeerWith connects el to peer and registers the peering as trusted on both sides, so the session
// survives reputation penalties, bans and eviction.
func (el *L2ELNode) PeerWith(peer *L2ELNode) {
	el.log.Info("Peering with a trusted peer", "node", el, "peer", peer)
	sysgo.ConnectP2P(el.ctx, el.require, el.rpc(), peer.rpc())
}

// PeerWithBasic connects el to peer as a basic peer: el dials peer and keeps it as a static entry,
// and nothing exempts the session from reputation penalties, bans or eviction on either side, so
// a test can observe peer management, which PeerWith masks.
func (el *L2ELNode) PeerWithBasic(peer *L2ELNode) {
	el.log.Info("Peering with a basic peer", "node", el, "peer", peer)
	sysgo.ConnectP2PBasic(el.ctx, el.require, el.rpc(), peer.rpc())
}

// DisconnectPeerWith undoes a PeerWith peering: it drops the trusted entries and the peer on both
// sides and waits for the session to end. It requires the trusted entries, so it is not the
// inverse of PeerWithBasic.
func (el *L2ELNode) DisconnectPeerWith(peer *L2ELNode) {
	el.log.Info("Disconnecting from a peer", "node", el, "peer", peer)
	sysgo.DisconnectP2P(el.ctx, el.require, el.rpc(), peer.rpc())
}

// VerifyPeeredWith requires peer to be among el's connected peers, waiting for it to show up.
func (el *L2ELNode) VerifyPeeredWith(peer *L2ELNode) {
	el.log.Info("Expecting a peer to be connected", "node", el, "peer", peer)
	sysgo.WaitP2PConnected(el.ctx, el.require, el.rpc(), peer.rpc())
}

// PeerInfoOf returns what el reports about its session with peer, once the session has
// negotiated its eth protocol version.
func (el *L2ELNode) PeerInfoOf(peer *L2ELNode) sysgo.ELPeerInfo {
	el.log.Info("Reading peer info", "node", el, "peer", peer)
	info := sysgo.PeerInfo(el.ctx, el.require, el.rpc(), peer.rpc())
	el.log.Info("Read peer info", "node", el, "peer", peer, "eth_version", info.EthVersion, "caps", info.Caps,
		"inbound", info.Inbound, "static", info.Static, "trusted", info.Trusted, "remote_address", info.RemoteAddress)
	return info
}

// Peers returns what el reports about every peer it is connected to, with an EthVersion of 0 for
// a session whose eth handshake is still pending.
func (el *L2ELNode) Peers() []sysgo.ELPeerInfo {
	peers, err := sysgo.ListPeers(el.ctx, el.rpc())
	el.require.NoError(err, "failed to list the peers of %s", el)
	return peers
}

func (el *L2ELNode) rpc() client.RPC {
	return el.inner.L2EthClient().RPC()
}
