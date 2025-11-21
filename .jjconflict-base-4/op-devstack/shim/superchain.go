package shim

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type SuperchainConfig struct {
	CommonConfig
	ID            stack.ComponentID
	Deployment    stack.SuperchainDeployment
	DependencySet depset.DependencySet
}

type presetSuperchain struct {
	commonImpl
	id            stack.ComponentID
	deployment    stack.SuperchainDeployment
	dependencySet depset.DependencySet
}

var _ stack.Superchain = (*presetSuperchain)(nil)

func NewSuperchain(cfg SuperchainConfig) stack.Superchain {
	cfg.T = cfg.T.WithCtx(stack.ContextWithComponentID(cfg.T.Ctx(), cfg.ID))
	return &presetSuperchain{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		deployment: cfg.Deployment,
	}
}

func (p *presetSuperchain) ID() stack.ComponentID {
	return p.id
}

func (p presetSuperchain) Deployment() stack.SuperchainDeployment {
	return p.deployment
}

func (p *presetSuperchain) DependencySet() depset.DependencySet {
	return p.dependencySet
}
