package p2p

import (
	"context"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
)

type PeerInfo struct {
	PeerID          peer.ID  `json:"peerID"`
	NodeID          enode.ID `json:"nodeID"`
	UserAgent       string   `json:"userAgent"`
	ProtocolVersion string   `json:"protocolVersion"`
	ENR             string   `json:"ENR"`       // might not always be known, e.g. if the peer connected us instead of us discovering them
	Addresses       []string `json:"addresses"` // multi-addresses. may be mix of LAN / docker / external IPs. All of them are communicated.
	Protocols       []string `json:"protocols"` // negotiated protocols list
	// GossipScore float64
	// PeerScore float64
	Connectedness network.Connectedness `json:"connectedness"` // "NotConnected", "Connected", "CanConnect" (gracefully disconnected), or "CannotConnect" (tried but failed)
	Direction     network.Direction     `json:"direction"`     // "Unknown", "Inbound" (if the peer contacted us), "Outbound" (if we connected to them)
	Protected     bool                  `json:"protected"`     // Protected peers do not get
	ChainID       uint64                `json:"chainID"`       // some peers might try to connect, but we figure out they are on a different chain later. This may be 0 if the peer is not an optimism node at all.
	Latency       time.Duration         `json:"latency"`

	GossipBlocks bool `json:"gossipBlocks"` // if the peer is in our gossip topic

	PeerScores store.PeerScores `json:"scores"`
}

type PeerDump struct {
	TotalConnected uint                 `json:"totalConnected"`
	Peers          map[string]*PeerInfo `json:"peers"`
	BannedPeers    []peer.ID            `json:"bannedPeers"`
	BannedIPS      []net.IP             `json:"bannedIPS"`
	BannedSubnets  []*net.IPNet         `json:"bannedSubnets"`
}

//go:generate mockery --name API --output mocks/ --with-expecter=true
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
