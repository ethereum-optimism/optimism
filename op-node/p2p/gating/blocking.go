package gating

import (
	"net"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
)

//go:generate mockery --name BlockingConnectionGater --output mocks/ --with-expecter=true
type BlockingConnectionGater interface {
	connmgr.ConnectionGater

	// BlockPeer adds a peer to the set of blocked peers.
	// Note: active connections to the peer are not automatically closed.
	BlockPeer(p peer.ID) error
	UnblockPeer(p peer.ID) error
	ListBlockedPeers() []peer.ID

	// BlockAddr adds an IP address to the set of blocked addresses.
	// Note: active connections to the IP address are not automatically closed.
	BlockAddr(ip net.IP) error
	UnblockAddr(ip net.IP) error
	ListBlockedAddrs() []net.IP

	// BlockSubnet adds an IP subnet to the set of blocked addresses.
	// Note: active connections to the IP subnet are not automatically closed.
	BlockSubnet(ipnet *net.IPNet) error
	UnblockSubnet(ipnet *net.IPNet) error
	ListBlockedSubnets() []*net.IPNet
}

func NewBlockingConnectionGater(store ds.Batching) (BlockingConnectionGater, error) {
	return conngater.NewBasicConnectionGater(store)
}
