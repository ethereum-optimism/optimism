package bridge

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/address_manager"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var zeroAddr common.Address

var standardContracts = []string{
	"Proxy__OVM_L1CrossDomainMessenger",
	"Proxy__OVM_L1StandardBridge",
	"StateCommitmentChain",
	"CanonicalTransactionChain",
	"BondManager",
}

type Addresses struct {
	addrs map[string]common.Address
}

func NewAddresses(client bind.ContractBackend, addrMgrAddr common.Address) (*Addresses, error) {
	ret := &Addresses{
		addrs: make(map[string]common.Address),
	}
	ret.addrs["AddressManager"] = addrMgrAddr

	mgr, err := address_manager.NewAddressManager(addrMgrAddr, client)
	if err != nil {
		return nil, err
	}

	for _, contractName := range standardContracts {
		contractAddr, err := mgr.GetAddress(nil, contractName)
		if err != nil {
			return nil, fmt.Errorf("error getting contract %s: %v", contractName, err)
		}
		if contractAddr == zeroAddr {
			return nil, fmt.Errorf("contract %s is not deployed", contractName)
		}
		ret.addrs[contractName] = contractAddr
	}

	return ret, nil
}

func (a *Addresses) AddressManager() common.Address {
	return a.addrs["AddressManager"]
}

func (a *Addresses) L1CrossDomainMessenger() common.Address {
	return a.addrs["Proxy__OVM_L1CrossDomainMessenger"]
}

func (a *Addresses) L1StandardBridge() common.Address {
	return a.addrs["Proxy__OVM_L1StandardBridge"]
}

func (a *Addresses) StateCommitmentChain() common.Address {
	return a.addrs["StateCommitmentChain"]
}

func (a *Addresses) CanonicalTransactionChain() common.Address {
	return a.addrs["CanonicalTransactionChain"]
}

func (a *Addresses) BondManager() common.Address {
	return a.addrs["BondManager"]
}
