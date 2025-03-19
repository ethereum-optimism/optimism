package system2

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// L1ChainID identifies a L1Chain by name and chainID, is type-safe, and can be value-copied and used as map key.
type L1ChainID idWithChain

func (id L1ChainID) String() string {
	return idWithChain(id).string("L1Chain")
}

func (id L1ChainID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText("L1Chain")
}

func (id *L1ChainID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText("L1Chain", data)
}

func SortL1ChainIDs(ids []L1ChainID) []L1ChainID {
	return copyAndSort(ids, func(a, b L1ChainID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// L1Chain represents a L1 chain, a collection of configuration and node resources.
type L1Chain interface {
	Chain
	ID() L1ChainID

	L1ELNode(id L1ELNodeID) L1ELNode
	L1CLNode(id L1CLNodeID) L1CLNode

	L1ELNodes() []L1ELNodeID
	L1CLNodes() []L1CLNodeID
}

type ExtensibleL1Chain interface {
	L1Chain
	AddL1ELNode(v L1ELNode)
	AddL1CLNode(v L1CLNode)
}

type L1ChainConfig struct {
	ChainConfig
	ID L1ChainID
}

type presetL1Chain struct {
	presetChain
	id L1ChainID

	els locks.RWMap[L1ELNodeID, L1ELNode]
	cls locks.RWMap[L1CLNodeID, L1CLNode]
}

var _ ExtensibleL1Chain = (*presetL1Chain)(nil)

func NewL1Chain(cfg L1ChainConfig) ExtensibleL1Chain {
	require.Equal(cfg.T, cfg.ID.ChainID, eth.ChainIDFromBig(cfg.ChainConfig.ChainCfg.ChainID), "chain config must match expected chain")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetL1Chain{
		id:          cfg.ID,
		presetChain: newChain(cfg.ChainConfig),
	}
}

func (p *presetL1Chain) ID() L1ChainID {
	return p.id
}

func (p *presetL1Chain) L1ELNode(id L1ELNodeID) L1ELNode {
	v, ok := p.els.Get(id)
	p.require().True(ok, "l1 EL node %s must exist", id)
	return v
}

func (p *presetL1Chain) AddL1ELNode(v L1ELNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 EL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.els.SetIfMissing(id, v), "l1 EL node %s must not already exist", id)
}

func (p *presetL1Chain) L1CLNode(id L1CLNodeID) L1CLNode {
	v, ok := p.cls.Get(id)
	p.require().True(ok, "l1 CL node %s must exist", id)
	return v
}

func (p *presetL1Chain) AddL1CLNode(v L1CLNode) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID, "l1 CL node %s must be on chain %s", id, p.chainID)
	p.require().True(p.cls.SetIfMissing(id, v), "l1 CL node %s must not already exist", id)
}

func (p *presetL1Chain) L1ELNodes() []L1ELNodeID {
	return SortL1ELNodeIDs(p.els.Keys())
}

func (p *presetL1Chain) L1CLNodes() []L1CLNodeID {
	return SortL1CLNodeIDs(p.cls.Keys())
}
