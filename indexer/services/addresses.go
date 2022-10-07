package services

import (
	"github.com/ethereum-optimism/optimism/indexer/bindings/legacy/scc"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type AddressManager interface {
	L1StandardBridge() (common.Address, *bindings.L1StandardBridge)
	StateCommitmentChain() (common.Address, *scc.StateCommitmentChain)
	OptimismPortal() (common.Address, *bindings.OptimismPortal)
}

type LegacyAddresses struct {
	l1SB     *bindings.L1StandardBridge
	l1SBAddr common.Address
	l2SB     *bindings.L2StandardBridge
	scc      *scc.StateCommitmentChain
	sccAddr  common.Address
}

func NewLegacyAddresses(client bind.ContractBackend, addrMgrAddr common.Address) (AddressManager, error) {
	mgr, err := bindings.NewAddressManager(addrMgrAddr, client)
	if err != nil {
		return nil, err
	}

	l1SBAddr, err := mgr.GetAddress(nil, "Proxy__OVM_L1StandardBridge")
	if err != nil {
		return nil, err
	}
	sccAddr, err := mgr.GetAddress(nil, "StateCommitmentChain")
	if err != nil {
		return nil, err
	}
	l1SB, err := bindings.NewL1StandardBridge(l1SBAddr, client)
	if err != nil {
		return nil, err
	}
	l2SB, err := bindings.NewL2StandardBridge(predeploys.L2StandardBridgeAddr, client)
	if err != nil {
		return nil, err
	}
	sccContract, err := scc.NewStateCommitmentChain(sccAddr, client)
	if err != nil {
		return nil, err
	}

	return &LegacyAddresses{
		l1SB:     l1SB,
		l1SBAddr: l1SBAddr,
		l2SB:     l2SB,
		scc:      sccContract,
		sccAddr:  sccAddr,
	}, nil
}

func (a *LegacyAddresses) L1StandardBridge() (common.Address, *bindings.L1StandardBridge) {
	return a.l1SBAddr, a.l1SB
}

func (a *LegacyAddresses) StateCommitmentChain() (common.Address, *scc.StateCommitmentChain) {
	return a.sccAddr, a.scc
}

func (a *LegacyAddresses) OptimismPortal() (common.Address, *bindings.OptimismPortal) {
	panic("OptimismPortal not configured on legacy networks - this is a programmer error")
}

type BedrockAddresses struct {
	l1SB       *bindings.L1StandardBridge
	l1SBAddr   common.Address
	portal     *bindings.OptimismPortal
	portalAddr common.Address
}

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

func (b *BedrockAddresses) StateCommitmentChain() (common.Address, *scc.StateCommitmentChain) {
	panic("SCC not configured on legacy networks - this is a programmer error")
}

func (b *BedrockAddresses) OptimismPortal() (common.Address, *bindings.OptimismPortal) {
	return b.portalAddr, b.portal
}
