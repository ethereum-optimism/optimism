package sysext

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

const (
	ProtocolVersionsAddressName = "ProtocolVersionsProxy"
	SuperchainConfigAddressName = "SuperchainConfigProxy"

	SystemConfigAddressName = "SystemConfigProxy"
	DisputeGameFactoryName  = "DisputeGameFactoryProxy"
)

type l1AddressBook struct {
	protocolVersions common.Address
	superchainConfig common.Address
}

func newL1AddressBook(t devtest.T, addresses descriptors.AddressMap) *l1AddressBook {
	protocolVersions, ok := addresses[ProtocolVersionsAddressName]
	t.Require().True(ok)
	superchainConfig, ok := addresses[SuperchainConfigAddressName]
	t.Require().True(ok)

	book := &l1AddressBook{
		protocolVersions: protocolVersions,
		superchainConfig: superchainConfig,
	}

	return book
}

func (a *l1AddressBook) ProtocolVersionsAddr() common.Address {
	return a.protocolVersions
}

func (a *l1AddressBook) SuperchainConfigAddr() common.Address {
	return a.superchainConfig
}

var _ stack.SuperchainDeployment = (*l1AddressBook)(nil)

type l2AddressBook struct {
	systemConfig       common.Address
	disputeGameFactory common.Address
}

func newL2AddressBook(t devtest.T, l1Addresses descriptors.AddressMap) *l2AddressBook {
	systemConfig, ok := l1Addresses[SystemConfigAddressName]
	t.Require().True(ok)
	disputeGameFactory, ok := l1Addresses[DisputeGameFactoryName]
	t.Require().True(ok)

	return &l2AddressBook{
		systemConfig:       systemConfig,
		disputeGameFactory: disputeGameFactory,
	}
}

func (a *l2AddressBook) SystemConfigProxyAddr() common.Address {
	return a.systemConfig
}

func (a *l2AddressBook) DisputeGameFactoryProxyAddr() common.Address {
	return a.disputeGameFactory
}

var _ stack.L2Deployment = (*l2AddressBook)(nil)
