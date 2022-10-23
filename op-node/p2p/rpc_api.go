package p2p

import (
	"context"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

type PeerInfo struct {
	ENR             string                `json:"ENR"` // might not always be known, e.g. if the peer connected us instead of us discovering them
	PeerID          peer.ID               `json:"peerID"`
	UserAgent       string                `json:"userAgent"`
	ProtocolVersion string                `json:"protocolVersion"`
	Protocols       []string              `json:"protocols"`     // negotiated protocols list
	Addresses       []string              `json:"addresses"`     // multi-addresses. may be mix of LAN / docker / external IPs. All of them are communicated.
	Connectedness   network.Connectedness `json:"connectedness"` // "NotConnected", "Connected", "CanConnect" (gracefully disconnected), or "CannotConnect" (tried but failed)
	Direction       network.Direction     `json:"direction"`     // "Unknown", "Inbound" (if the peer contacted us), "Outbound" (if we connected to them)
	ChainID         uint64                `json:"chainID"`       // some peers might try to connect, but we figure out they are on a different chain later. This may be 0 if the peer is not an optimism node at all.
	Latency         time.Duration         `json:"latency"`
	NodeID          enode.ID              `json:"nodeID"`
	Protected       bool                  `json:"protected"`    // Protected peers do not get
	GossipBlocks    bool                  `json:"gossipBlocks"` // if the peer is in our gossip topic
	//GossipScore float64
	//PeerScore float64
}

type PeerDump struct {
	Peers          map[string]*PeerInfo `json:"peers"`
	BannedPeers    []peer.ID            `json:"bannedPeers"`
	BannedIPS      []net.IP             `json:"bannedIPS"`
	BannedSubnets  []*net.IPNet         `json:"bannedSubnets"`
	TotalConnected uint                 `json:"totalConnected"`
}

type API interface {
	Self(ctx context.Context) (*PeerInfo, error)
	Peers(ctx context.Context, connected bool) (*PeerDump, error)
	PeerStats(ctx context.Context) (*PeerStats, error)
	DiscoveryTable(ctx context.Context) ([]*enode.Node, error)
	BlockPeer(ctx context.Context, p peer.ID) error
	UnblockPeer(ctx context.Context, p peer.ID) error
	ListBlockedPeers(ctx context.Context) ([]peer.ID, error)
	BlockAddr(ctx context.Context, ip net.IP) error
	UnblockAddr(ctx context.Context, ip net.IP) error
	ListBlockedAddrs(ctx context.Context) ([]net.IP, error)
	BlockSubnet(ctx context.Context, ipnet *net.IPNet) error
	UnblockSubnet(ctx context.Context, ipnet *net.IPNet) error
	ListBlockedSubnets(ctx context.Context) ([]*net.IPNet, error)
	ProtectPeer(ctx context.Context, p peer.ID) error
	UnprotectPeer(ctx context.Context, p peer.ID) error
	ConnectPeer(ctx context.Context, addr string) error
	DisconnectPeer(ctx context.Context, id peer.ID) error
}
