package sources

import (
	"context"
	"net"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
)

type P2PClient struct {
	client client.RPC
}

var P2PNamespaceRPC = "opp2p"

var _ apis.P2PClient = (*P2PClient)(nil)

func prefixP2PRPC(method string) string {
	return P2PNamespaceRPC + "_" + method
}

func NewP2PClient(client client.RPC) *P2PClient {
	return &P2PClient{
		client: client,
	}
}

func (pc *P2PClient) Self(ctx context.Context) (*apis.PeerInfo, error) {
	output := &apis.PeerInfo{}
	err := pc.client.CallContext(ctx, output, prefixP2PRPC("self"))
	return output, err
}

func (pc *P2PClient) Peers(ctx context.Context, connected bool) (*apis.PeerDump, error) {
	output := &apis.PeerDump{}
	err := pc.client.CallContext(ctx, &output, prefixP2PRPC("peers"), connected)
	return output, err
}

func (pc *P2PClient) PeerStats(ctx context.Context) (*apis.PeerStats, error) {
	output := &apis.PeerStats{}
	err := pc.client.CallContext(ctx, output, prefixP2PRPC("peerStats"))
	return output, err
}

func (pc *P2PClient) DiscoveryTable(ctx context.Context) ([]*enode.Node, error) {
	output := []*enode.Node{}
	err := pc.client.CallContext(ctx, &output, prefixP2PRPC("discoveryTable"))
	return output, err
}

func (pc *P2PClient) BlockPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("blockPeer"), p)
}

func (pc *P2PClient) UnblockPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("unblockPeer"), p)
}

func (pc *P2PClient) ListBlockedPeers(ctx context.Context) ([]peer.ID, error) {
	output := []peer.ID{}
	err := pc.client.CallContext(ctx, &output, prefixP2PRPC("listBlockedPeers"))
	return output, err
}

func (pc *P2PClient) BlockAddr(ctx context.Context, ip net.IP) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("blockAddr"), ip)
}

func (pc *P2PClient) UnblockAddr(ctx context.Context, ip net.IP) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("unblockAddr"), ip)
}

func (pc *P2PClient) ListBlockedAddrs(ctx context.Context) ([]net.IP, error) {
	output := []net.IP{}
	err := pc.client.CallContext(ctx, &output, prefixP2PRPC("listBlockedAddrs"))
	return output, err
}

func (pc *P2PClient) BlockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("blockSubnet"), ipnet)
}

func (pc *P2PClient) UnblockSubnet(ctx context.Context, ipnet *net.IPNet) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("unblockSubnet"), ipnet)
}

func (pc *P2PClient) ListBlockedSubnets(ctx context.Context) ([]*net.IPNet, error) {
	output := []*net.IPNet{}
	err := pc.client.CallContext(ctx, &output, prefixP2PRPC("listBlockedSubnets"))
	return output, err
}

func (pc *P2PClient) ProtectPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("protectPeer"), p)
}

func (pc *P2PClient) UnprotectPeer(ctx context.Context, p peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("unprotectPeer"), p)
}

func (pc *P2PClient) ConnectPeer(ctx context.Context, addr string) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("connectPeer"), addr)
}

func (pc *P2PClient) DisconnectPeer(ctx context.Context, id peer.ID) error {
	return pc.client.CallContext(ctx, nil, prefixP2PRPC("disconnectPeer"), id)
}
