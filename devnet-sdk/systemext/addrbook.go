package systemext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum/go-ethereum/common"
)

const (
	ProtocolVersionsAddressName = "protocolVersionsProxy"
	SuperchainConfigAddressName = "superchainConfigProxy"

	SystemConfigAddressName = "systemConfigProxy"
)

type l1AddressBook struct {
	protocolVersions common.Address
	superchainConfig common.Address
}

func newL1AddressBook(setup *system2.Setup, addresses descriptors.AddressMap) *l1AddressBook {
	protocolVersions, ok := addresses[ProtocolVersionsAddressName]
	setup.Require.True(ok)
	superchainConfig, ok := addresses[SuperchainConfigAddressName]
	setup.Require.True(ok)

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

var _ system2.SuperchainDeployment = (*l1AddressBook)(nil)

type l2AddressBook struct {
	systemConfig common.Address
}

func newL2AddressBook(setup *system2.Setup, l1Addresses descriptors.AddressMap) *l2AddressBook {
	systemConfig, ok := l1Addresses[SystemConfigAddressName]
	setup.Require.True(ok)

	return &l2AddressBook{
		systemConfig: systemConfig,
	}
}

func (a *l2AddressBook) SystemConfigProxyAddr() common.Address {
	return a.systemConfig
}

var _ system2.L2Deployment = (*l2AddressBook)(nil)
