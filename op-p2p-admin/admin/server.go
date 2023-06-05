package admin

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-p2p-admin/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"sort"
)

type SourceNode struct {
	API   p2p.API
	Self  *p2p.PeerInfo
	Peers *p2p.PeerDump
}

type Server struct {
	log     log.Logger
	m       metrics.Metricer
	sources []*SourceNode
}

func NewServer(log log.Logger, m metrics.Metricer) *Server {
	return &Server{log: log, m: m}
}

func (s *Server) Serve(ctx context.Context) error {
	// TODO
	return nil
}

// TODO: more tabulator customization: https://tabulator.info/examples/5.5

type tabulatorColumn struct {
	Title   string            `json:"title"`
	Field   string            `json:"field"`
	Width   int               `json:"width"`
	Columns []tabulatorColumn `json:"columns"`
}

func makeColumns(peers []peer.ID) []tabulatorColumn {
	byPeer := func(field string) []tabulatorColumn {
		peerColumns := make([]tabulatorColumn, len(peers))
		for i, id := range peers {
			peerColumns[i].Title = id.String()
			peerColumns[i].Field = field + "_" + id.String()
		}
		return peerColumns
	}
	return []tabulatorColumn{
		{Title: "Peer ID", Field: "peerID", Width: 200},
		{Title: "Node ID", Field: "nodeID"},
		{Title: "User Agent", Field: "userAgent"},
		{Title: "Protocol Version", Field: "protocolVersion"},
		{Title: "ENR", Field: "ENR"},
		{Title: "Addresses", Field: "addresses"},
		{Title: "Protocols", Field: "protocols"},
		{Title: "Connectedness", Columns: byPeer("connectedness")},
		{Title: "Direction", Columns: byPeer("direction")},
		{Title: "Protected", Field: "protected"},
		{Title: "ChainID", Field: "chainID"},
		{Title: "Latency", Columns: byPeer("latency")},
		{Title: "Gossip blocks", Columns: byPeer("gossipBlocks")},
		{Title: "Score", Columns: byPeer("score")},
		{Title: "Conflicting", Field: "conflicting"}, // when we perceive different values per peer source, for attributes that should match (e.g. chain ID)
	}
}

func addPeer(source peer.ID, row map[string]any, info *p2p.PeerInfo) {
	plural := func(field string, v any) {
		row[field+"_"+source.String()] = v
	}
	singular := func(field string, v any) {
		if known, ok := row[field]; ok {
			knownV := fmt.Sprintf("%v", known)
			newV := fmt.Sprintf("%v", v)
			if knownV != newV {
				row["conflicting"] = true
			}
			return
		}
		row[field] = v
	}
	singular("peerID", info.PeerID)
	singular("nodeID", info.NodeID)
	singular("userAgent", info.UserAgent)
	singular("protocolVersion", info.ProtocolVersion)
	plural("ENR", info.ENR)
	plural("addresses", info.Addresses)
	singular("protocols", info.Protocols)
	plural("direction", info.Direction.String())
	singular("chainID", info.ChainID)
	plural("gossipBlocks", info.GossipBlocks)
	plural("score", info.PeerScores.Gossip)
	plural("connectedness", info.Connectedness.String())
	plural("latency", info.Latency.Milliseconds())
}

func (s *Server) HandlePeerlist() {
	peers := make(map[peer.ID]struct{})
	for _, src := range s.sources {
		for _, v := range src.Peers.Peers {
			peers[v.PeerID] = struct{}{}
		}
	}
	peerList := make([]map[string]any, 0, len(peers))
	for id, _ := range peers {
		m := make(map[string]any)
		for _, src := range s.sources {
			if srcDat, ok := src.Peers.Peers[id.String()]; ok {
				addPeer(src.Self.PeerID, m, srcDat)
			}
		}
		peerList = append(peerList, m)
	}
	sort.Slice(peerList, func(i, j int) bool {
		return peerList[i]["peerID"].(string) < peerList[j]["peerID"].(string)
	})
}
