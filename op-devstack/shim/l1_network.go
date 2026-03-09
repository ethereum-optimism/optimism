package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1NetworkConfig struct {
	NetworkConfig
	ID stack.ComponentID
}

type presetL1Network struct {
	presetNetwork
	id stack.ComponentID
}

var _ stack.ExtensibleL1Network = (*presetL1Network)(nil)

func NewL1Network(cfg L1NetworkConfig) stack.ExtensibleL1Network {
	require.Equal(cfg.T, cfg.ID.ChainID(), eth.ChainIDFromBig(cfg.NetworkConfig.ChainConfig.ChainID), "chain config must match expected chain")
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &presetL1Network{
		id:            cfg.ID,
		presetNetwork: newNetwork(cfg.NetworkConfig),
	}
}

func (p *presetL1Network) ID() stack.ComponentID {
	return p.id
}

func (p *presetL1Network) L1ELNode(m stack.L1ELMatcher) stack.L1ELNode {
	getter := func(id stack.ComponentID) (stack.L1ELNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L1ELNode), true
	}
	v, ok := findMatch(m, getter, p.L1ELNodes)
	p.require().True(ok, "must find L1 EL %s", m)
	return v
}

func (p *presetL1Network) AddL1ELNode(v stack.L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l1 EL node %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l1 EL node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL1Network) L1CLNode(m stack.L1CLMatcher) stack.L1CLNode {
	getter := func(id stack.ComponentID) (stack.L1CLNode, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.L1CLNode), true
	}
	v, ok := findMatch(m, getter, p.L1CLNodes)
	p.require().True(ok, "must find L1 CL %s", m)
	return v
}

func (p *presetL1Network) AddL1CLNode(v stack.L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l1 CL node %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "l1 CL node %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetL1Network) L1ELNodeIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL1ELNode))
}

func (p *presetL1Network) L1ELNodes() []stack.L1ELNode {
	ids := p.registry.IDsByKind(stack.KindL1ELNode)
	result := make([]stack.L1ELNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L1ELNode))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetL1Network) L1CLNodeIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindL1CLNode))
}

func (p *presetL1Network) L1CLNodes() []stack.L1CLNode {
	ids := p.registry.IDsByKind(stack.KindL1CLNode)
	result := make([]stack.L1CLNode, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.L1CLNode))
		}
	}
	return sortByIDFunc(result)
}
