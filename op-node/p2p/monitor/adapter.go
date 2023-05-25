package monitor

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerManagerAdapter implements the PeerManager interface by delegating to a variety of different p2p components
type PeerManagerAdapter struct {
	n       network.Network
	connMgr connmgr.ConnManager
	scores  store.ScoreDatastore
	// TODO: something to do banning but its not merged yet...
}

func NewPeerManagerAdapter(n network.Network, connMgr connmgr.ConnManager, scores store.ScoreDatastore) *PeerManagerAdapter {
	return &PeerManagerAdapter{
		n:       n,
		connMgr: connMgr,
		scores:  scores,
	}
}

func (p *PeerManagerAdapter) Peers() []peer.ID {
	return p.n.Peers()
}

func (p *PeerManagerAdapter) GetPeerScore(id peer.ID) (float64, error) {
	scores, err := p.scores.GetPeerScores(id)
	if err != nil {
		return 0, err
	}
	return scores.Gossip.Total, nil
}

func (p *PeerManagerAdapter) IsProtected(id peer.ID) bool {
	if p.connMgr == nil {
		return false
	}
	// TODO: Need a constant for the tag somewhere
	return p.connMgr.IsProtected(id, "static")
}

func (p *PeerManagerAdapter) ClosePeer(id peer.ID) error {
	return p.n.ClosePeer(id)
}

func (p *PeerManagerAdapter) BanPeer(id peer.ID, banDuration time.Time) error {
	//TODO implement me
	return fmt.Errorf("peer banning not implemented")
}

var _ PeerManager = (*PeerManagerAdapter)(nil)
