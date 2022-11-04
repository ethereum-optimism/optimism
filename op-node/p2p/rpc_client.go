package p2p

import (
	"context"
	"net"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

var NamespaceRPC = "opp2p"

type Client struct {
	c *rpc.Client
}

var _ API = (*Client)(nil)

func NewClient(c *rpc.Client) *Client {
	return &Client{c: c}
}

func prefixRPC(method string) string {
	return NamespaceRPC + "_" + method
}

func (c *Client) Self(ctx context.Context) (*PeerInfo, error) {
	var out *PeerInfo
	err := c.c.CallContext(ctx, &out, prefixRPC("self"))
	return out, err
}

func (c *Client) Peers(ctx context.Context, connected bool) (*PeerDump, error) {
	var out *PeerDump
	err := c.c.CallContext(ctx, &out, prefixRPC("peers"), connected)
	return out, err
}

func (c *Client) PeerStats(ctx context.Context) (*PeerStats, error) {
	var out *PeerStats
	err := c.c.CallContext(ctx, &out, prefixRPC("peerStats"))
	return out, err
}

func (c *Client) DiscoveryTable(ctx context.Context) ([]*enode.Node, error) {
	var out []*enode.Node
	err := c.c.CallContext(ctx, &out, prefixRPC("discoveryTable"))
	return out, err
}

func (c *Client) BlockPeer(ctx context.Context, p peer.ID) error {
	return c.c.CallContext(ctx, nil, prefixRPC("blockPeer"), p)
}

func (c *Client) UnblockPeer(ctx context.Context, p peer.ID) error {
	return c.c.CallContext(ctx, nil, prefixRPC("unblockPeer"), p)
}

func (c *Client) ListBlockedPeers(ctx context.Context) ([]peer.ID, error) {
	var out []peer.ID
	err := c.c.CallContext(ctx, &out, prefixRPC("listBlockedPeers"))
	return out, err
}

func (c *Client) BlockAddr(ctx context.Context, ip net.IP) error {
	return c.c.CallContext(ctx, nil, prefixRPC("blockAddr"), ip)
}

func (c *Client) UnblockAddr(ctx context.Context, ip net.IP) error {
	return c.c.CallContext(ctx, nil, prefixRPC("unblockAddr"), ip)
}

func (c *Client) ListBlockedAddrs(ctx context.Context) ([]net.IP, error) {
	var out []net.IP
	err := c.c.CallContext(ctx, &out, prefixRPC("listBlockedAddrs"))
	return out, err
}

func (c *Client) BlockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return c.c.CallContext(ctx, nil, prefixRPC("blockSubnet"), ipnet)
}

func (c *Client) UnblockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return c.c.CallContext(ctx, nil, prefixRPC("unblockSubnet"), ipnet)
}

func (c *Client) ListBlockedSubnets(ctx context.Context) ([]*net.IPNet, error) {
	var out []*net.IPNet
	err := c.c.CallContext(ctx, &out, prefixRPC("listBlockedSubnets"))
	return out, err
}

func (c *Client) ProtectPeer(ctx context.Context, p peer.ID) error {
	return c.c.CallContext(ctx, nil, prefixRPC("protectPeer"), p)
}

func (c *Client) UnprotectPeer(ctx context.Context, p peer.ID) error {
	return c.c.CallContext(ctx, nil, prefixRPC("unprotectPeer"), p)
}

func (c *Client) ConnectPeer(ctx context.Context, addr string) error {
	return c.c.CallContext(ctx, nil, prefixRPC("connectPeer"), addr)
}

func (c *Client) DisconnectPeer(ctx context.Context, id peer.ID) error {
	return c.c.CallContext(ctx, nil, prefixRPC("disconnectPeer"), id)
}
