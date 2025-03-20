package system2

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// L1NetworkID identifies a L1Network by name and chainID, is type-safe, and can be value-copied and used as map key.
type L1NetworkID idWithChain

const L1NetworkKind Kind = "L1Network"

func (id L1NetworkID) String() string {
	return idWithChain(id).string(L1NetworkKind)
}

func (id L1NetworkID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(L1NetworkKind)
}

func (id *L1NetworkID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(L1NetworkKind, data)
}

func SortL1NetworkIDs(ids []L1NetworkID) []L1NetworkID {
	return copyAndSort(ids, func(a, b L1NetworkID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L1Network represents a L1 chain, a collection of configuration and node resources.
type L1Network interface {
	Network
	ID() L1NetworkID

	L1ELNode(id L1ELNodeID) L1ELNode
	L1CLNode(id L1CLNodeID) L1CLNode

	L1ELNodes() []L1ELNodeID
	L1CLNodes() []L1CLNodeID
}

type ExtensibleL1Network interface {
	ExtensibleNetwork
	L1Network
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}

type L1NetworkConfig struct {
	NetworkConfig
	ID L1NetworkID
}

type presetL1Network struct {
	presetNetwork
	id L1NetworkID

	els locks.RWMap[L1ELNodeID, L1ELNode]
	cls locks.RWMap[L1CLNodeID, L1CLNode]
}

var _ ExtensibleL1Network = (*presetL1Network)(nil)

func NewL1Network(cfg L1NetworkConfig) ExtensibleL1Network {
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.NetworkConfig.ChainConfig.ChainID), "chain config must match expected chain")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetL1Network{
		id:            cfg.ID,
		presetNetwork: newNetwork(cfg.NetworkConfig),
	}
}

func (p *presetL1Network) ID() L1NetworkID {
	return p.id
}

func (p *presetL1Network) L1ELNode(id L1ELNodeID) L1ELNode {
	v, ok := p.els.Get(id)
	p.require().True(ok, "l1 EL node %s must exist", id)
	return v
}

func (p *presetL1Network) AddL1ELNode(v L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l1 EL node %s must not already exist", id)
}

func (p *presetL1Network) L1CLNode(id L1CLNodeID) L1CLNode {
	v, ok := p.cls.Get(id)
	p.require().True(ok, "l1 CL node %s must exist", id)
	return v
}

func (p *presetL1Network) AddL1CLNode(v L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l1 CL node %s must not already exist", id)
}

func (p *presetL1Network) L1ELNodes() []L1ELNodeID {
	return SortL1ELNodeIDs(p.els.Keys())
}

func (p *presetL1Network) L1CLNodes() []L1CLNodeID {
	return SortL1CLNodeIDs(p.cls.Keys())
}
