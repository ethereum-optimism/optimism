package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type L1NetworkConfig struct {
	NetworkConfig
	ID stack.L1NetworkID
}

type presetL1Network struct {
	presetNetwork
	id stack.L1NetworkID

	els locks.RWMap[stack.L1ELNodeID, stack.L1ELNode]
	cls locks.RWMap[stack.L1CLNodeID, stack.L1CLNode]
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

func (p *presetL1Network) ID() stack.L1NetworkID {
	return p.id
}

func (p *presetL1Network) L1ELNode(m stack.L1ELMatcher) stack.L1ELNode {
	v, ok := findMatch(m, p.els.Get, p.L1ELNodes)
	p.require().True(ok, "must find L1 EL %s", m)
	return v
}

func (p *presetL1Network) AddL1ELNode(v stack.L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l1 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l1 EL node %s must not already exist", id)
}

func (p *presetL1Network) L1CLNode(m stack.L1CLMatcher) stack.L1CLNode {
	v, ok := findMatch(m, p.cls.Get, p.L1CLNodes)
	p.require().True(ok, "must find L1 CL %s", m)
	return v
}

func (p *presetL1Network) AddL1CLNode(v stack.L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "l1 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l1 CL node %s must not already exist", id)
}

func (p *presetL1Network) L1ELNodeIDs() []stack.L1ELNodeID {
	return stack.SortL1ELNodeIDs(p.els.Keys())
}

func (p *presetL1Network) L1ELNodes() []stack.L1ELNode {
	return stack.SortL1ELNodes(p.els.Values())
}

func (p *presetL1Network) L1CLNodeIDs() []stack.L1CLNodeID {
	return stack.SortL1CLNodeIDs(p.cls.Keys())
}

func (p *presetL1Network) L1CLNodes() []stack.L1CLNode {
	return stack.SortL1CLNodes(p.cls.Values())
}
