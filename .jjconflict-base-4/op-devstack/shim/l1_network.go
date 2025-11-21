package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type L1NetworkConfig struct {
	NetworkConfig
	ID eth.ChainID
}

type presetL1Network struct {
	presetNetwork
	id eth.ChainID

	els locks.RWMap[stack.ComponentID, stack.L1ELNode]
	cls locks.RWMap[stack.ComponentID, stack.L1CLNode]
}

var _ stack.ExtensibleL1Network = (*presetL1Network)(nil)

func NewL1Network(cfg L1NetworkConfig) stack.ExtensibleL1Network {
	cfg.T = cfg.T.WithCtx(stack.ContextWithChainID(cfg.T.Ctx(), cfg.ID))
	return &presetL1Network{
		id:            cfg.ID,
		presetNetwork: newNetwork(cfg.NetworkConfig),
	}
}

func (p *presetL1Network) ID() eth.ChainID {
	return p.id
}

func (p *presetL1Network) L1ELNode(m stack.L1ELMatcher) stack.L1ELNode {
	v, ok := findMatch(m, p.els.Values())
	p.require().True(ok, "must find L1 EL %s", m)
	return v
}

func (p *presetL1Network) AddL1ELNode(v stack.L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, v.ChainID(), "l1 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l1 EL node %s must not already exist", id)
}

func (p *presetL1Network) L1CLNode(m stack.L1CLMatcher) stack.L1CLNode {
	v, ok := findMatch(m, p.cls.Values())
	p.require().True(ok, "must find L1 CL %s", m)
	return v
}

func (p *presetL1Network) AddL1CLNode(v stack.L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, v.ChainID(), "l1 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l1 CL node %s must not already exist", id)
}

func (p *presetL1Network) L1ELNodes() []stack.L1ELNode {
	return stack.SortL1ELNodes(p.els.Values())
}

func (p *presetL1Network) L1CLNodes() []stack.L1CLNode {
	return stack.SortL1CLNodes(p.cls.Values())
}
