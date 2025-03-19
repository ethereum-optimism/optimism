package system2

import (
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Chain is an ethereum chain, common between L1 and L2.
// For L1 or L2 specifics, see L1Chain and L2Chain extensions.
// A chain hosts configuration resources and tracks participating nodes.
type Chain interface {
	Common

	ChainID() eth.ChainID

	ChainConfig() *params.ChainConfig

	Faucet() Faucet
}

type ChainConfig struct {
	CommonConfig
	ChainCfg *params.ChainConfig
}

type presetChain struct {
	commonImpl
	faucet   Faucet
	chainCfg *params.ChainConfig
	chainID  eth.ChainID
}

var _ Chain = (*presetChain)(nil)

// newChain creates a new chain, safe to embed in other structs
func newChain(cfg ChainConfig) presetChain {
	return presetChain{
		commonImpl: newCommon(cfg.CommonConfig),
		chainCfg:   cfg.ChainCfg,
		chainID:    eth.ChainIDFromBig(cfg.ChainCfg.ChainID),
	}
}

func (p *presetChain) ChainID() eth.ChainID {
	return p.chainID
}

func (p *presetChain) ChainConfig() *params.ChainConfig {
	return p.chainCfg
}

func (p *presetChain) Faucet() Faucet {
	p.require().NotNil(p.faucet, "faucet not available")
	return p.faucet
}
