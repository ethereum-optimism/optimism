package shim

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
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
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.NetworkConfig.ChainConfig.ChainID), "chain config must match expected chain")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetL1Network{
		id:            cfg.ID,
		presetNetwork: newNetwork(cfg.NetworkConfig),
	}
}

func (p *presetL1Network) ID() stack.L1NetworkID {
	return p.id
}

func (p *presetL1Network) L1ELNode(id stack.L1ELNodeID) stack.L1ELNode {
	v, ok := p.els.Get(id)
	p.require().True(ok, "l1 EL node %s must exist", id)
	return v
}

func (p *presetL1Network) AddL1ELNode(v stack.L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l1 EL node %s must not already exist", id)
}

func (p *presetL1Network) L1CLNode(id stack.L1CLNodeID) stack.L1CLNode {
	v, ok := p.cls.Get(id)
	p.require().True(ok, "l1 CL node %s must exist", id)
	return v
}

func (p *presetL1Network) AddL1CLNode(v stack.L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l1 CL node %s must not already exist", id)
}

func (p *presetL1Network) L1ELNodes() []stack.L1ELNodeID {
	return stack.SortL1ELNodeIDs(p.els.Keys())
}

func (p *presetL1Network) L1CLNodes() []stack.L1CLNodeID {
	return stack.SortL1CLNodeIDs(p.cls.Keys())
}
