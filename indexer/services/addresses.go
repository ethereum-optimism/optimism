package services

import (
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type AddressManager interface {
	L1StandardBridge() (common.Address, *bindings.L1StandardBridge)
	OptimismPortal() (common.Address, *bindings.OptimismPortal)
}

type BedrockAddresses struct {
	l1SB       *bindings.L1StandardBridge
	l1SBAddr   common.Address
	portal     *bindings.OptimismPortal
	portalAddr common.Address
}

var _ AddressManager = (*BedrockAddresses)(nil)

func NewBedrockAddresses(client bind.ContractBackend, l1SBAddr, portalAddr common.Address) (AddressManager, error) {
	l1SB, err := bindings.NewL1StandardBridge(l1SBAddr, client)
	if err != nil {
		return nil, err
	}
	portal, err := bindings.NewOptimismPortal(portalAddr, client)
	if err != nil {
		return nil, err
	}

	return &BedrockAddresses{
		l1SB:       l1SB,
		l1SBAddr:   l1SBAddr,
		portal:     portal,
		portalAddr: portalAddr,
	}, nil
}

func (b *BedrockAddresses) L1StandardBridge() (common.Address, *bindings.L1StandardBridge) {
	return b.l1SBAddr, b.l1SB
}

func (b *BedrockAddresses) OptimismPortal() (common.Address, *bindings.OptimismPortal) {
	return b.portalAddr, b.portal
}
