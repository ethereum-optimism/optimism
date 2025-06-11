package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment struct {
	protocolVersionsAddr common.Address
	superchainConfigAddr common.Address
}

var _ stack.SuperchainDeployment = &SuperchainDeployment{}

func (d *SuperchainDeployment) SuperchainConfigAddr() common.Address {
	return d.superchainConfigAddr
}

func (d *SuperchainDeployment) ProtocolVersionsAddr() common.Address {
	return d.protocolVersionsAddr
}

type Superchain struct {
	id         stack.SuperchainID
	deployment *SuperchainDeployment
}

func (s *Superchain) hydrate(system stack.ExtensibleSystem) {
	sysSuperchain := shim.NewSuperchain(shim.SuperchainConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           s.id,
		Deployment:   s.deployment,
	})
	system.AddSuperchain(sysSuperchain)
}
