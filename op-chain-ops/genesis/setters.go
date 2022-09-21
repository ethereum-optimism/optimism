package genesis

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// FundDevAccounts will fund each of the development accounts.
func FundDevAccounts(db vm.StateDB) {
	for _, account := range DevAccounts {
		db.CreateAccount(account)
		db.AddBalance(account, devBalance)
	}
}

// SetL2Proxies will set each of the proxies in the state. It requires
// a Proxy and ProxyAdmin deployment present so that the Proxy bytecode
// can be set in state and the ProxyAdmin can be set as the admin of the
// Proxy.
func SetL2Proxies(db vm.StateDB, proxyAdminAddr common.Address) error {
	return setProxies(db, proxyAdminAddr, bigL2PredeployNamespace, 2048)
}

// SetL1Proxies will set each of the proxies in the state. It requires
// a Proxy and ProxyAdmin deployment present so that the Proxy bytecode
// can be set in state and the ProxyAdmin can be set as the admin of the
// Proxy.
func SetL1Proxies(db vm.StateDB, proxyAdminAddr common.Address) error {
	return setProxies(db, proxyAdminAddr, bigL1PredeployNamespace, 2048)
}

func setProxies(db vm.StateDB, proxyAdminAddr common.Address, namespace *big.Int, count uint64) error {
	depBytecode, err := bindings.GetDeployedBytecode("Proxy")
	if err != nil {
		return err
	}

	for i := uint64(0); i <= count; i++ {
		bigAddr := new(big.Int).Or(namespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)

		// There is no proxy at the governance token address
		if addr == predeploys.GovernanceTokenAddr {
			continue
		}

		db.CreateAccount(addr)
		db.SetCode(addr, depBytecode)
		db.SetState(addr, AdminSlot, proxyAdminAddr.Hash())
	}
	return nil
}

// SetImplementations will set the implmentations of the contracts in the state
// and configure the proxies to point to the implementations. It also sets
// the appropriate storage values for each contract at the proxy address.
func SetImplementations(db vm.StateDB, storage state.StorageConfig, immutable immutables.ImmutableConfig) error {
	deployResults, err := immutables.BuildOptimism(immutable)
	if err != nil {
		return err
	}

	for name, address := range predeploys.Predeploys {
		// Convert the address to the code address
		var addr common.Address
		switch *address {
		case predeploys.GovernanceTokenAddr:
			addr = predeploys.GovernanceTokenAddr
		case predeploys.LegacyERC20ETHAddr:
			addr = predeploys.LegacyERC20ETHAddr
		default:
			addr, err = AddressToCodeNamespace(*address)
			if err != nil {
				return err
			}
			// Set the implementation slot in the predeploy proxy
			db.SetState(*address, ImplementationSlot, addr.Hash())
		}

		// Create the account
		db.CreateAccount(addr)

		// Use the genrated bytecode when there are immutables
		// otherwise use the artifact deployed bytecode
		if bytecode, ok := deployResults[name]; ok {
			db.SetCode(addr, bytecode)
		} else {
			depBytecode, err := bindings.GetDeployedBytecode(name)
			if err != nil {
				return err
			}
			db.SetCode(addr, depBytecode)
		}

		// Set the storage values
		if storageConfig, ok := storage[name]; ok {
			if err := state.SetStorage(name, *address, storageConfig, db); err != nil {
				return err
			}
		}

		code := db.GetCode(addr)
		if len(code) == 0 {
			return fmt.Errorf("code not set for %s", name)
		}
	}
	return nil
}

// Get the storage layout of the LegacyMessagePasser
// Iterate over the storage layout to know which storage slots to ignore
// Iterate over each storage slot, compute the migration
func MigrateDepositHashes(db vm.StateDB) error {
	layout, err := bindings.GetStorageLayout("LegacyMessagePasser")
	if err != nil {
		return err
	}

	// Build a list of storage slots to ignore. The values in the
	// mapping are guaranteed to not be in this list because they are
	// hashes.
	ignore := make(map[common.Hash]bool)
	for _, entry := range layout.Storage {
		encoded, err := state.EncodeUintValue(entry.Slot, 0)
		if err != nil {
			return err
		}
		ignore[encoded] = true
	}

	return db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		if _, ok := ignore[key]; ok {
			return true
		}
		// TODO(tynes): Do the value migration here
		return true
	})
}

// SetPrecompileBalances will set a single wei at each precompile address.
// This is an optimization to make calling them cheaper. This should only
// be used for devnets.
func SetPrecompileBalances(db vm.StateDB) {
	for i := 0; i < 256; i++ {
		addr := common.BytesToAddress([]byte{byte(i)})
		db.CreateAccount(addr)
		db.AddBalance(addr, common.Big1)
	}
}
