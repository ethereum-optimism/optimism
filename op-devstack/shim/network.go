package shim

import (
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type NetworkConfig struct {
	CommonConfig
	ChainConfig *params.ChainConfig
}

type presetNetwork struct {
	commonImpl
	chainCfg *params.ChainConfig
	chainID  eth.ChainID

	// Unified component registry for generic access
	registry *stack.Registry
}

var _ stack.Network = (*presetNetwork)(nil)

// newNetwork creates a new network, safe to embed in other structs
func newNetwork(cfg NetworkConfig) presetNetwork {
	return presetNetwork{
		commonImpl: newCommon(cfg.CommonConfig),
		chainCfg:   cfg.ChainConfig,
		chainID:    eth.ChainIDFromBig(cfg.ChainConfig.ChainID),
		registry:   stack.NewRegistry(),
	}
}

// --- ComponentRegistry interface implementation ---

func (p *presetNetwork) Component(id stack.ComponentID) (any, bool) {
	return p.registry.Get(id)
}

func (p *presetNetwork) Components(kind stack.ComponentKind) []any {
	ids := p.registry.IDsByKind(kind)
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		if comp, ok := p.registry.Get(id); ok {
			result = append(result, comp)
		}
	}
	return result
}

func (p *presetNetwork) ComponentIDs(kind stack.ComponentKind) []stack.ComponentID {
	return p.registry.IDsByKind(kind)
}

func (p *presetNetwork) ChainID() eth.ChainID {
	return p.chainID
}

func (p *presetNetwork) ChainConfig() *params.ChainConfig {
	return p.chainCfg
}

func (p *presetNetwork) FaucetIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindFaucet))
}

func (p *presetNetwork) Faucets() []stack.Faucet {
	ids := p.registry.IDsByKind(stack.KindFaucet)
	result := make([]stack.Faucet, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.Faucet))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetNetwork) Faucet(m stack.FaucetMatcher) stack.Faucet {
	getter := func(id stack.ComponentID) (stack.Faucet, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.Faucet), true
	}
	v, ok := findMatch(m, getter, p.Faucets)
	p.require().True(ok, "must find faucet %s", m)
	return v
}

func (p *presetNetwork) AddFaucet(v stack.Faucet) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "faucet %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "faucet %s must not already exist", id)
	p.registry.Register(id, v)
}

func (p *presetNetwork) SyncTesterIDs() []stack.ComponentID {
	return sortByID(p.registry.IDsByKind(stack.KindSyncTester))
}

func (p *presetNetwork) SyncTesters() []stack.SyncTester {
	ids := p.registry.IDsByKind(stack.KindSyncTester)
	result := make([]stack.SyncTester, 0, len(ids))
	for _, id := range ids {
		if v, ok := p.registry.Get(id); ok {
			result = append(result, v.(stack.SyncTester))
		}
	}
	return sortByIDFunc(result)
}

func (p *presetNetwork) SyncTester(m stack.SyncTesterMatcher) stack.SyncTester {
	getter := func(id stack.ComponentID) (stack.SyncTester, bool) {
		v, ok := p.registry.Get(id)
		if !ok {
			return nil, false
		}
		return v.(stack.SyncTester), true
	}
	v, ok := findMatch(m, getter, p.SyncTesters)
	p.require().True(ok, "must find sync tester %s", m)
	return v
}

func (p *presetNetwork) AddSyncTester(v stack.SyncTester) {
	id := v.ID()
	p.require().Equal(p.chainID, id.ChainID(), "sync tester %s must be on chain %s", id, p.chainID)
	_, exists := p.registry.Get(id)
	p.require().False(exists, "sync tester %s must not already exist", id)
	p.registry.Register(id, v)
}
