package p2p

import (
	"errors"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/metrics"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// Prepared provides a p2p host and discv5 service that is already set up.
// This implements SetupP2P.
type Prepared struct {
	HostP2P   host.Host
	LocalNode *enode.LocalNode
	UDPv5     *discover.UDPv5

	EnableReqRespSync   bool
	EnablePeerBanning   bool
	PeerScoreParams     pubsub.PeerScoreParams
	TopicScoreParams    pubsub.TopicScoreParams
	BandScoreThresholds *BandScoreThresholds
	Gater               connmgr.ConnectionGater
	ConnMgr             connmgr.ConnManager
}

var _ SetupP2P = (*Prepared)(nil)

func (p *Prepared) TargetPeers() uint {
	return 20
}

func (p *Prepared) Check() error {
	if (p.LocalNode == nil) != (p.UDPv5 == nil) {
		return fmt.Errorf("inconsistent discv5 setup: %v <> %v", p.LocalNode, p.UDPv5)
	}
	if p.LocalNode != nil && p.HostP2P == nil {
		return errors.New("cannot provide discovery without p2p host")
	}
	return nil
}

// Host creates a libp2p host service. Returns nil, nil if p2p is disabled.
func (p *Prepared) Host(log log.Logger, reporter metrics.Reporter) (host.Host, error) {
	h := &extrasHost{
		Host: p.HostP2P,

		ConnMgr: p.ConnMgr,
	}
	// Only add the connection gater if it offers the full interface we're looking for.
	if g, ok := p.Gater.(ConnectionGater); ok {
		h.Gater = g
	}
	return h, nil
}

// Discovery creates a disc-v5 service. Returns nil, nil, nil if discovery is disabled.
func (p *Prepared) Discovery(log log.Logger, rollupCfg *rollup.Config, tcpPort uint16) (*enode.LocalNode, *discover.UDPv5, error) {
	if p.LocalNode != nil {
		dat := OpStackENRData{
			chainID: rollupCfg.L2ChainID.Uint64(),
			version: 0,
		}
		p.LocalNode.Set(&dat)
		if tcpPort != 0 {
			p.LocalNode.Set(enr.TCP(tcpPort))
		}
	}
	return p.LocalNode, p.UDPv5, nil
}

func (p *Prepared) ConfigureGossip(rollupCfg *rollup.Config) []pubsub.Option {
	return []pubsub.Option{
		pubsub.WithGossipSubParams(BuildGlobalGossipParams(rollupCfg)),
	}
}

func (p *Prepared) PeerScoringParams() *pubsub.PeerScoreParams {
	return &p.PeerScoreParams
}

func (p *Prepared) PeerBandScorer() *BandScoreThresholds {
	return p.BandScoreThresholds
}

func (p *Prepared) BanPeers() bool {
	return p.EnablePeerBanning
}

func (p *Prepared) TopicScoringParams() *pubsub.TopicScoreParams {
	return &p.TopicScoreParams
}

func (p *Prepared) Disabled() bool {
	return false
}

func (p *Prepared) ReqRespSyncEnabled() bool {
	return p.EnableReqRespSync
}

type extrasHost struct {
	host.Host
	Gater   ConnectionGater
	ConnMgr connmgr.ConnManager
}

func (h *extrasHost) ConnectionGater() ConnectionGater {
	return h.Gater
}

func (h *extrasHost) ConnectionManager() connmgr.ConnManager {
	return h.ConnMgr
}
