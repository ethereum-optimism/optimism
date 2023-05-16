package p2pstub

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

type MocknetGater struct {
	self peer.ID
	*conngater.BasicConnectionGater
	mn mocknet.Mocknet
}

func NewMocknetGater(self peer.ID, gater *conngater.BasicConnectionGater, mn mocknet.Mocknet) *MocknetGater {
	return &MocknetGater{
		self:                 self,
		BasicConnectionGater: gater,
		mn:                   mn,
	}
}

func (g *MocknetGater) BlockPeer(p peer.ID) error {
	err := g.mn.UnlinkPeers(g.self, p)
	if err != nil {
		return fmt.Errorf("unlink peers %v and %v: %w", g.self, p, err)
	}
	return g.BasicConnectionGater.BlockPeer(p)
}

func (g *MocknetGater) UnblockPeer(p peer.ID) error {
	_, err := g.mn.LinkPeers(g.self, p)
	if err != nil {
		return fmt.Errorf("link peers %v and %v: %w", g.self, p, err)
	}
	return g.BasicConnectionGater.UnblockPeer(p)
}

var _ p2p.ConnectionGater = (*MocknetGater)(nil)
