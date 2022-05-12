package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

// TODO: dynamic peering
// - req-resp protocol to ensure peers from a different chain learn they shouldn't be connected
// - banning peers based on score
// - store enode in peerstore in dynamic-peering background process
// - peers must be tagged with the "optimism" tag and marked with high value if the chain ID matches

var (
	DisabledP2P         = errors.New("p2p is disabled")
	DisabledDiscovery   = errors.New("discovery disabled")
	NoConnectionManager = errors.New("no connection manager")
	NoConnectionGater   = errors.New("no connection gater")
)

type Node interface {
	// Host returns the libp2p host, nil if disabled
	Host() host.Host
	// Dv5Local returns the control over the Discv5 data of the local node, nil if disabled
	Dv5Local() *enode.LocalNode
	// Dv5Udp returns the control over the Discv5 network, nil if disabled
	Dv5Udp() *discover.UDPv5
	// GossipSub returns the gossip router
	GossipSub() *pubsub.PubSub
	// GossipTopicInfo returns the gossip topic info handle
	GossipTopicInfo() GossipTopicInfo
	// ConnectionGater returns the connection gater, to ban/unban peers with, may be nil
	ConnectionGater() ConnectionGater
	// ConnectionManager returns the connection manager, to protect peers with, may be nil
	ConnectionManager() connmgr.ConnManager
}

type APIBackend struct {
	node Node
	log  log.Logger
}

func NewP2PAPIBackend(node Node, log log.Logger) *APIBackend {
	return &APIBackend{
		node: node,
		log:  log,
	}
}

type PeerInfo struct {
	PeerID          peer.ID        `json:"peerID"`
	NodeID          enode.ID       `json:"nodeID"`
	UserAgent       string         `json:"userAgent"`
	ProtocolVersion string         `json:"protocolVersion"`
	ENR             string         `json:"ENR"`       // might not always be known, e.g. if the peer connected us instead of us discovering them
	Addresses       []ma.Multiaddr `json:"addresses"` // may be mix of LAN / docker / external IPs. All of them are communicated.
	Protocols       []string       `json:"protocols"` // negotiated protocols list
	//GossipScore float64
	//PeerScore float64
	Connectedness network.Connectedness `json:"connectedness"` // "NotConnected", "Connected", "CanConnect" (gracefully disconnected), or "CannotConnect" (tried but failed)
	Direction     network.Direction     `json:"direction"`     // "Unknown", "Inbound" (if the peer contacted us), "Outbound" (if we connected to them)
	BannedID      bool                  `json:"bannedID"`      // If the peer has been banned by peer ID
	BannedIP      bool                  `json:"bannedIP"`      // If the peer has been banned by IP address
	BannedSubnet  bool                  `json:"bannedSubnet"`  // If the peer has been banned as part of a whole IP subnet
	Protected     bool                  `json:"protected"`     // Protected peers do not get
	ChainID       uint64                `json:"chainID"`       // some peers might try to connect, but we figure out they are on a different chain later. This may be 0 if the peer is not an optimism node at all.
	Latency       time.Duration         `json:"latency"`

	GossipBlocks bool `json:"gossipBlocks"` // if the peer is in our gossip topic
}

func dumpPeer(id peer.ID, nw network.Network, pstore peerstore.Peerstore, connMgr connmgr.ConnManager) (*PeerInfo, error) {
	info := &PeerInfo{
		PeerID: id,
	}

	// we might not have the pubkey if it's from a multi-addr and if we never discovered/connected them
	pub := pstore.PubKey(id)
	if pub != nil {
		typedPub, ok := pub.(*crypto.Secp256k1PublicKey)
		if !ok {
			return nil, fmt.Errorf("unexpected pubkey type: %T", pub)
		}
		info.NodeID = enode.PubkeyToIDV4((*ecdsa.PublicKey)(typedPub))
	}
	if dat, err := pstore.Get(id, "ProtocolVersion"); err != nil {
		protocolVersion, ok := dat.(string)
		if ok {
			info.ProtocolVersion = protocolVersion
		}
	}
	if dat, err := pstore.Get(id, "AgentVersion"); err != nil {
		agentVersion, ok := dat.(string)
		if ok {
			info.UserAgent = agentVersion
		}
	}
	if dat, err := pstore.Get(id, "ENR"); err != nil {
		enodeData, ok := dat.(*enode.Node)
		if ok {
			info.ENR = enodeData.String()
		}
	}
	info.Addresses = pstore.Addrs(id)
	info.Connectedness = nw.Connectedness(id)
	if protocols, err := pstore.GetProtocols(id); err != nil {
		info.Protocols = protocols
	}
	// get the first connection direction, if any (will default to unknown when there are no connections)
	for _, c := range nw.ConnsToPeer(id) {
		info.Direction = c.Stat().Direction
		break
	}
	if dat, err := pstore.Get(id, "optimismChainID"); err != nil {
		chID, ok := dat.(uint64)
		if ok {
			info.ChainID = chID
		}
	}
	info.Latency = pstore.LatencyEWMA(id)
	if connMgr != nil {
		info.Protected = connMgr.IsProtected(id, "")
	}

	return info, nil
}

type PeerDump struct {
	TotalConnected uint                  `json:"totalConnected"`
	Peers          map[peer.ID]*PeerInfo `json:"peers"`
	BannedPeers    []peer.ID             `json:"bannedPeers"`
	BannedIPS      []net.IP              `json:"bannedIPS"`
	BannedSubnets  []*net.IPNet          `json:"bannedSubnets"`
}

// Peers lists information of peers. Optionally filter to only retrieve connected peers.
func (s *APIBackend) Peers(ctx context.Context, connected bool) (*PeerDump, error) {
	h := s.node.Host()
	if h == nil {
		return nil, DisabledP2P
	}

	nw := h.Network()
	pstore := h.Peerstore()
	var peers []peer.ID
	if connected {
		peers = nw.Peers()
	} else {
		peers = pstore.Peers()
	}

	dump := &PeerDump{Peers: make(map[peer.ID]*PeerInfo)}
	for _, id := range peers {
		peerInfo, err := dumpPeer(id, nw, pstore, s.node.ConnectionManager())
		if err != nil {
			dump.Peers[id] = peerInfo
		}
		if peerInfo.Connectedness == network.Connected {
			dump.TotalConnected += 1
		}
	}
	for _, id := range s.node.GossipTopicInfo().BlocksTopicPeers() {
		if p, ok := dump.Peers[id]; ok {
			p.GossipBlocks = true
		}
	}
	if gater := s.node.ConnectionGater(); gater != nil {
		dump.BannedPeers = gater.ListBlockedPeers()
		dump.BannedSubnets = gater.ListBlockedSubnets()
		dump.BannedIPS = gater.ListBlockedAddrs()
	}
	return dump, nil
}

type PeerStats struct {
	Connected   uint `json:"connected"`
	Table       uint `json:"table"`
	BlocksTopic uint `json:"blocksTopic"`
	Banned      uint `json:"banned"`
	Known       uint `json:"known"`
}

func (s *APIBackend) PeerStats() (*PeerStats, error) {
	h := s.node.Host()
	if h == nil {
		return nil, DisabledP2P
	}

	nw := h.Network()
	pstore := h.Peerstore()

	stats := &PeerStats{
		Connected:   uint(len(nw.Peers())),
		Table:       0,
		BlocksTopic: uint(len(s.node.GossipTopicInfo().BlocksTopicPeers())),
		Banned:      0,
		Known:       uint(len(pstore.Peers())),
	}
	if gater := s.node.ConnectionGater(); gater != nil {
		stats.Banned = uint(len(gater.ListBlockedPeers()))
	}
	if dv5 := s.node.Dv5Udp(); dv5 != nil {
		stats.Table = uint(len(dv5.AllNodes()))
	}
	return stats, nil
}

func (s *APIBackend) DiscoveryTable() ([]*enode.Node, error) {
	if dv5 := s.node.Dv5Udp(); dv5 != nil {
		return dv5.AllNodes(), nil
	} else {
		return nil, DisabledDiscovery
	}
}

func (s *APIBackend) BlockPeer(p peer.ID) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.BlockPeer(p)
	}
}

func (s *APIBackend) UnblockPeer(p peer.ID) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.UnblockPeer(p)
	}
}

func (s *APIBackend) ListBlockedPeers() ([]peer.ID, error) {
	if gater := s.node.ConnectionGater(); gater == nil {
		return nil, NoConnectionGater
	} else {
		return gater.ListBlockedPeers(), nil
	}
}

// BlockAddr adds an IP address to the set of blocked addresses.
// Note: active connections to the IP address are not automatically closed.
func (s *APIBackend) BlockAddr(ip net.IP) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.BlockAddr(ip)
	}
}

func (s *APIBackend) UnblockAddr(ip net.IP) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.UnblockAddr(ip)
	}
}

func (s *APIBackend) ListBlockedAddrs() ([]net.IP, error) {
	if gater := s.node.ConnectionGater(); gater == nil {
		return nil, NoConnectionGater
	} else {
		return gater.ListBlockedAddrs(), nil
	}
}

// BlockSubnet adds an IP subnet to the set of blocked addresses.
// Note: active connections to the IP subnet are not automatically closed.
func (s *APIBackend) BlockSubnet(ipnet *net.IPNet) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.BlockSubnet(ipnet)
	}
}

func (s *APIBackend) UnblockSubnet(ipnet *net.IPNet) error {
	if gater := s.node.ConnectionGater(); gater == nil {
		return NoConnectionGater
	} else {
		return gater.UnblockSubnet(ipnet)
	}
}

func (s *APIBackend) ListBlockedSubnets() ([]*net.IPNet, error) {
	if gater := s.node.ConnectionGater(); gater == nil {
		return nil, NoConnectionGater
	} else {
		return gater.ListBlockedSubnets(), nil
	}
}

func (s *APIBackend) ProtectPeer(p peer.ID) error {
	if manager := s.node.ConnectionManager(); manager == nil {
		return NoConnectionManager
	} else {
		manager.Protect(p, "api-protected")
		return nil
	}
}

func (s *APIBackend) UnprotectPeer(p peer.ID) error {
	if manager := s.node.ConnectionManager(); manager == nil {
		return NoConnectionManager
	} else {
		manager.Unprotect(p, "api-protected")
		return nil
	}
}

// ConnectPeer connects to a given peer address, and wait for protocol negotiation & identification of the peer
func (s *APIBackend) ConnectPeer(ctx context.Context, addr ma.Multiaddr) error {
	h := s.node.Host()
	if h == nil {
		return DisabledP2P
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("bad peer address: %v", err)
	}
	// Put a sanity limit on the connection time
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	return h.Connect(ctx, *addrInfo)
}

func (s *APIBackend) DisconnectPeer(id peer.ID) error {
	h := s.node.Host()
	if h == nil {
		return DisabledP2P
	}
	return h.Network().ClosePeer(id)
}
