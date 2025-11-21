package shim

import (
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type NetworkConfig struct {
	CommonConfig
	ChainConfig *params.ChainConfig
}

type presetNetwork struct {
	commonImpl
	chainCfg *params.ChainConfig
	chainID  eth.ChainID

	faucets     locks.RWMap[stack.ComponentID, stack.Faucet]
	syncTesters locks.RWMap[stack.ComponentID, stack.SyncTester]
}

var _ stack.Network = (*presetNetwork)(nil)

// newNetwork creates a new network, safe to embed in other structs
func newNetwork(cfg NetworkConfig) presetNetwork {
	return presetNetwork{
		commonImpl: newCommon(cfg.CommonConfig),
		chainCfg:   cfg.ChainConfig,
		chainID:    eth.ChainIDFromBig(cfg.ChainConfig.ChainID),
	}
}

func (p *presetNetwork) ID() eth.ChainID {
	return p.chainID
}

func (p *presetNetwork) ChainConfig() *params.ChainConfig {
	return p.chainCfg
}

func (p *presetNetwork) FaucetIDs() []stack.ComponentID {
	return p.faucets.Keys()
}

func (p *presetNetwork) Faucets() []stack.Faucet {
	return stack.SortFaucets(p.faucets.Values())
}

func (p *presetNetwork) Faucet(m stack.FaucetMatcher) stack.Faucet {
	v, ok := findMatch(m, p.faucets.Values())
	p.require().True(ok, "must find faucet %s", m)
	return v
}

func (p *presetNetwork) AddFaucet(v stack.Faucet) {
	id := v.ID()
	p.require().Equal(p.chainID, v.ChainID(), "faucet %s must be on chain %s", id, p.chainID)
	p.require().True(p.faucets.SetIfMissing(id, v), "faucet %s must not already exist", id)
}

func (p *presetNetwork) SyncTesterIDs() []stack.ComponentID {
	return p.syncTesters.Keys()
}

func (p *presetNetwork) SyncTesters() []stack.SyncTester {
	return stack.SortSyncTesters(p.syncTesters.Values())
}

func (p *presetNetwork) SyncTester(m stack.SyncTesterMatcher) stack.SyncTester {
	v, ok := findMatch(m, p.syncTesters.Values())
	p.require().True(ok, "must find sync tester %s", m)
	return v
}

func (p *presetNetwork) AddSyncTester(v stack.SyncTester) {
	id := v.ID()
	p.require().Equal(p.chainID, v.ChainID(), "sync tester %s must be on chain %s", id, p.chainID)
	p.require().True(p.syncTesters.SetIfMissing(id, v), "sync tester %s must not already exist", id)
}
