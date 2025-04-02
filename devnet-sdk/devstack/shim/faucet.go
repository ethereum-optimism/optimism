package shim

import "github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"

type FaucetConfig struct {
	CommonConfig
	ID stack.FaucetID
}

type presetFaucet struct {
	commonImpl
	id stack.FaucetID
}

var _ stack.Faucet = (*presetFaucet)(nil)

func NewFaucet(cfg FaucetConfig) stack.Faucet {
	return &presetFaucet{
		id:         cfg.ID,
		commonImpl: newCommon(cfg.CommonConfig),
	}
}

func (p *presetFaucet) ID() stack.FaucetID {
	return p.id
}

func (p *presetFaucet) NewUser() stack.User {
	p.require().Fail("not implemented")
	return nil
}
