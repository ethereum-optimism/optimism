package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum/go-ethereum/common"
)

type SuperchainDeployment struct {
	superchainConfigAddr common.Address
}

var _ stack.SuperchainDeployment = &SuperchainDeployment{}

func (d *SuperchainDeployment) SuperchainConfigAddr() common.Address {
	return d.superchainConfigAddr
}
