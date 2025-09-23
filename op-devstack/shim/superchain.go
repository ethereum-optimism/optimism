package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type SuperchainConfig struct {
	CommonConfig
	ID         stack.SuperchainID
	Deployment stack.SuperchainDeployment
}

type presetSuperchain struct {
	commonImpl
	id         stack.SuperchainID
	deployment stack.SuperchainDeployment
}

var _ stack.Superchain = (*presetSuperchain)(nil)

func NewSuperchain(cfg SuperchainConfig) stack.Superchain {
	cfg.T = cfg.T.WithCtx(stack.ContextWithID(cfg.T.Ctx(), cfg.ID))
	return &presetSuperchain{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		deployment: cfg.Deployment,
	}
}

func (p *presetSuperchain) ID() stack.SuperchainID {
	return p.id
}

func (p presetSuperchain) Deployment() stack.SuperchainDeployment {
	return p.deployment
}
