package sysgo

import (
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
