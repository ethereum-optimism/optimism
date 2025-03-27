package system2

import (
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
)

// Network is an interface to an ethereum chain and its resources, with common properties between L1 and L2.
// For L1 or L2 specifics, see L1Network and L2Network extensions.
// A network hosts configuration resources and tracks participating nodes.
type Network interface {
	Common

	ChainID() eth.ChainID

	ChainConfig() *params.ChainConfig

	Faucet() Faucet

	User(id UserID) User
	Users() []UserID
}

type ExtensibleNetwork interface {
	Network
	AddUser(v User)
}

type NetworkConfig struct {
	CommonConfig
	ChainConfig *params.ChainConfig
}

type presetNetwork struct {
	commonImpl
	faucet   Faucet
	chainCfg *params.ChainConfig
	chainID  eth.ChainID

	users locks.RWMap[UserID, User]
}

var _ Network = (*presetNetwork)(nil)

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

func (p *presetNetwork) Faucet() Faucet {
	p.require().NotNil(p.faucet, "faucet not available")
	return p.faucet
}

func (p *presetNetwork) User(id UserID) User {
	v, ok := p.users.Get(id)
	p.require().True(ok, "user %s must exist", id)
	return v
}

func (p *presetNetwork) AddUser(v User) {
	p.require().True(p.users.SetIfMissing(v.ID(), v), "user %s must not already exist", v.ID())
}

func (p *presetNetwork) Users() []UserID {
	return SortUserIDs(p.users.Keys())
}
