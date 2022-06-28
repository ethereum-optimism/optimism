package bridge

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var zeroAddr common.Address

// managedContract represents a contract that is managed
// by the AddressManager
type managedContract struct {
	name     string
	required bool
}

var standardContracts = []managedContract{
	{"Proxy__OVM_L1CrossDomainMessenger", false},
	{"Proxy__OVM_L1StandardBridge", false},
	{"StateCommitmentChain", false},
	{"CanonicalTransactionChain", false},
	{"BondManager", false},
}

type Addresses struct {
	addrs map[string]common.Address
}

func NewAddresses(client bind.ContractBackend, addrMgrAddr common.Address) (*Addresses, error) {
	ret := &Addresses{
		addrs: make(map[string]common.Address),
	}
	ret.addrs["AddressManager"] = addrMgrAddr

	mgr, err := bindings.NewLibAddressManager(addrMgrAddr, client)
	if err != nil {
		return nil, err
	}

	for _, contract := range standardContracts {
		contractAddr, err := mgr.GetAddress(nil, contract.name)
		if err != nil {
			return nil, fmt.Errorf("error getting contract %s: %v", contract.name, err)
		}
		if contractAddr == zeroAddr && contract.required {
			return nil, fmt.Errorf("contract %s is not deployed", contract.name)
		}
		ret.addrs[contract.name] = contractAddr
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
