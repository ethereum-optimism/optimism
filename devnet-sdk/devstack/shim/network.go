package shim

import (
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

type NetworkConfig struct {
	CommonConfig
	ChainConfig *params.ChainConfig
}

type presetNetwork struct {
	commonImpl
	faucet   stack.Faucet
	chainCfg *params.ChainConfig
	chainID  eth.ChainID

	users locks.RWMap[stack.UserID, stack.User]
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

func (p *presetNetwork) ChainID() eth.ChainID {
	return p.chainID
}

func (p *presetNetwork) ChainConfig() *params.ChainConfig {
	return p.chainCfg
}

func (p *presetNetwork) Faucet() stack.Faucet {
	p.require().NotNil(p.faucet, "faucet not available")
	return p.faucet
}

func (p *presetNetwork) User(m stack.UserMatcher) stack.User {
	v, ok := findMatch(m, p.users.Get, p.Users)
	p.require().True(ok, "must find user %s", m)
	return v
}

func (p *presetNetwork) AddUser(v stack.User) {
	p.require().True(p.users.SetIfMissing(v.ID(), v), "user %s must not already exist", v.ID())
}

func (p *presetNetwork) Users() []stack.User {
	return stack.SortUsers(p.users.Values())
}

func (p *presetNetwork) UserIDs() []stack.UserID {
	return stack.SortUserIDs(p.users.Keys())
}
