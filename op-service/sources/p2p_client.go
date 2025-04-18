package sources

import (
	"context"
	"net"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
)

type P2PClient struct {
	client client.RPC
}

var NamespaceRPC = "opp2p"

var _ p2p.API = (*P2PClient)(nil)

func prefixRPC(method string) string {
	return NamespaceRPC + "_" + method
}

func NewP2PClient(client client.RPC) *P2PClient {
	return &P2PClient{
		client: client,
	}
}

func (pc *P2PClient) Self(ctx context.Context) (*p2p.PeerInfo, error) {
	output := &p2p.PeerInfo{}
	err := pc.client.CallContext(ctx, output, prefixRPC("self"))
	return output, err
}

func (pc *P2PClient) Peers(ctx context.Context, connected bool) (*p2p.PeerDump, error) {
	output := &p2p.PeerDump{}
	err := pc.client.CallContext(ctx, &output, prefixRPC("peers"), connected)
	return output, err
}

func (pc *P2PClient) PeerStats(ctx context.Context) (*p2p.PeerStats, error) {
	output := &p2p.PeerStats{}
	err := pc.client.CallContext(ctx, output, prefixRPC("peerStats"))
	return output, err
}

func (pc *P2PClient) DiscoveryTable(ctx context.Context) ([]*enode.Node, error) {
	output := []*enode.Node{}
	err := pc.client.CallContext(ctx, &output, prefixRPC("discoveryTable"))
	return output, err
}

func (pc *P2PClient) BlockPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("blockPeer"), p)
}

func (pc *P2PClient) UnblockPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("unblockPeer"), p)
}

func (pc *P2PClient) ListBlockedPeers(ctx context.Context) ([]peer.ID, error) {
	output := []peer.ID{}
	err := pc.client.CallContext(ctx, &output, prefixRPC("listBlockedPeers"))
	return output, err
}

func (pc *P2PClient) BlockAddr(ctx context.Context, ip net.IP) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("blockAddr"), ip)
}

func (pc *P2PClient) UnblockAddr(ctx context.Context, ip net.IP) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("unblockAddr"), ip)
}

func (pc *P2PClient) ListBlockedAddrs(ctx context.Context) ([]net.IP, error) {
	output := []net.IP{}
	err := pc.client.CallContext(ctx, &output, prefixRPC("listBlockedAddrs"))
	return output, err
}

func (pc *P2PClient) BlockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("blockSubnet"), ipnet)
}

func (pc *P2PClient) UnblockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("unblockSubnet"), ipnet)
}

func (pc *P2PClient) ListBlockedSubnets(ctx context.Context) ([]*net.IPNet, error) {
	output := []*net.IPNet{}
	err := pc.client.CallContext(ctx, &output, prefixRPC("listBlockedSubnets"))
	return output, err
}

func (pc *P2PClient) ProtectPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("protectPeer"), p)
}

func (pc *P2PClient) UnprotectPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("unprotectPeer"), p)
}

func (pc *P2PClient) ConnectPeer(ctx context.Context, addr string) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("connectPeer"), addr)
}

func (pc *P2PClient) DisconnectPeer(ctx context.Context, id peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixRPC("disconnectPeer"), id)
}
