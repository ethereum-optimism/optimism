package genesis

import (
	"fmt"
	"math/big"

	"github.com/bobanetwork/boba/boba-bindings/bindings"
	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/bobanetwork/boba/boba-chain-ops/chain"
	"github.com/bobanetwork/boba/boba-chain-ops/immutables"
	"github.com/bobanetwork/boba/boba-chain-ops/state"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"

	"github.com/ledgerwatch/log/v3"
)

// UntouchableCodeHashes contains code hashes of all the contracts
// that should not be touched by the migration process.
type ChainHashMap map[uint64]common.Hash

var (
	// UntouchablePredeploys are addresses in the predeploy namespace
	// that should not be touched by the migration process.
	UntouchablePredeploys = map[common.Address]bool{
		predeploys.BobaL2Addr: true,
		predeploys.WETH9Addr:  true,
	}

	// UntouchableCodeHashes represent the bytecode hashes of contracts
	// that should not be touched by the migration process.
	UntouchableCodeHashes = map[common.Address]ChainHashMap{
		predeploys.BobaL2Addr: {
			28882: common.HexToHash("0x536465c3460a5849f66be041a130eedbac32f223f6990db22988bd6db9e156f4"),
		},
		predeploys.WETH9Addr: {
			288:   common.HexToHash("0x5b4b51d84d1f4b5bff7e20e96ed0771857d01c15aee81ff1eb34cf75c25e725e"),
			28882: common.HexToHash("0x5b4b51d84d1f4b5bff7e20e96ed0771857d01c15aee81ff1eb34cf75c25e725e"),
		},
	}

	// FrozenStoragePredeploys represents the set of predeploys that
	// will not have their storage wiped during the migration process.
	// It is very explicitly set in its own mapping to ensure that
	// changes elsewhere in the codebase do no alter the predeploys
	// that do not have their storage wiped. It is safe for all other
	// predeploys to have their storage wiped.
	FrozenStoragePredeploys = map[common.Address]bool{
		predeploys.WETH9Addr:               true,
		predeploys.LegacyMessagePasserAddr: true,
		predeploys.LegacyERC20ETHAddr:      true,
		predeploys.DeployerWhitelistAddr:   true,
		// Boba
		predeploys.BobaL2Addr: true,
	}
)

// SetL2Proxies will set each of the proxies in the state. It requires
// a Proxy and ProxyAdmin deployment present so that the Proxy bytecode
// can be set in state and the ProxyAdmin can be set as the admin of the
// Proxy.
func SetL2Proxies(g *types.Genesis) error {
	return setProxies(g, predeploys.ProxyAdminAddr, BigL2PredeployNamespace, 2048)
}

// WipePredeployStorage will wipe the storage of all L2 predeploys expect
// for predeploys that must not have their storage altered.
func WipePredeployStorage(g *types.Genesis) error {
	for name, addr := range predeploys.Predeploys {
		if addr == nil {
			return fmt.Errorf("nil address in predeploys mapping for %s", name)
		}

		if FrozenStoragePredeploys[*addr] && (*addr != predeploys.BobaL2Addr || chain.IsBobaTokenPredeploy(g.Config.ChainID)) {
			log.Trace("skipping wiping of storage", "name", name, "address", *addr)
			continue
		}

		log.Info("wiping storage", "name", name, "address", *addr)

		genesisAccount := types.GenesisAccount{
			Constructor: g.Alloc[*addr].Constructor,
			Code:        g.Alloc[*addr].Code,
			Storage:     map[common.Hash]common.Hash{},
			Balance:     g.Alloc[*addr].Balance, // This should be zero
			Nonce:       g.Alloc[*addr].Nonce,
		}
		g.Alloc[*addr] = genesisAccount
	}

	return nil
}

func setProxies(g *types.Genesis, proxyAdminAddr common.Address, namespace *big.Int, count uint64) error {
	depBytecode, err := bindings.GetDeployedBytecode("Proxy")
	if err != nil {
		return err
	}

	for i := uint64(0); i <= count; i++ {
		bigAddr := new(big.Int).Or(namespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)

		if UntouchablePredeploys[addr] && (addr != predeploys.BobaL2Addr || chain.IsBobaTokenPredeploy(g.Config.ChainID)) {
			log.Info("Skipping setting proxy", "address", addr)
			continue
		}

		balance := g.Alloc[addr].Balance
		if balance == nil {
			balance = big.NewInt(0)
		}

		var genesisAccount types.GenesisAccount
		if g.Alloc[addr].Storage == nil {
			genesisAccount = types.GenesisAccount{
				Constructor: g.Alloc[addr].Constructor,
				Code:        depBytecode,
				Storage: map[common.Hash]common.Hash{
					AdminSlot: proxyAdminAddr.Hash(),
				},
				Balance: balance,
				Nonce:   g.Alloc[addr].Nonce,
			}
		} else {
			g.Alloc[addr].Storage[AdminSlot] = proxyAdminAddr.Hash()
			genesisAccount = types.GenesisAccount{
				Constructor: g.Alloc[addr].Constructor,
				Code:        depBytecode,
				Storage:     g.Alloc[addr].Storage,
				Balance:     balance,
				Nonce:       g.Alloc[addr].Nonce,
			}
		}
		g.Alloc[addr] = genesisAccount

		log.Trace("Set proxy", "address", addr, "admin", proxyAdminAddr)
	}

	return nil
}

func SetLegacyETH(g *types.Genesis, storage state.StorageConfig, immutable immutables.ImmutableConfig) error {
	deployResults, err := immutables.BuildOptimism(immutable)
	if err != nil {
		return err
	}

	return setupPredeploy(g, deployResults, storage, "LegacyERC20ETH", predeploys.LegacyERC20ETHAddr, predeploys.LegacyERC20ETHAddr)
}

// SetImplementations will set the implementations of the contracts in the state
// and configure the proxies to point to the implementations. It also sets
// the appropriate storage values for each contract at the proxy address.
func SetImplementations(g *types.Genesis, storage state.StorageConfig, immutable immutables.ImmutableConfig) error {
	deployResults, err := immutables.BuildOptimism(immutable)
	if err != nil {
		return err
	}

	for name, address := range predeploys.Predeploys {
		if UntouchablePredeploys[*address] {
			continue
		}

		if *address == predeploys.LegacyERC20ETHAddr {
			continue
		}

		var (
			codeAddr        common.Address
			withoutImplSlot bool
		)
		switch name {
		case "Create2Deployer", "DeterministicDeploymentProxy":
			codeAddr = *address
			withoutImplSlot = true
		default:
			codeAddr, err = AddressToCodeNamespace(*address)
			if err != nil {
				return fmt.Errorf("error converting to code namespace: %w", err)
			}
		}

		balance := g.Alloc[*address].Balance
		if balance == nil {
			balance = big.NewInt(0)
		}

		if g.Alloc[*address].Storage == nil {
			genesisAccount := types.GenesisAccount{
				Constructor: g.Alloc[*address].Constructor,
				Code:        g.Alloc[*address].Code,
				Storage:     map[common.Hash]common.Hash{},
				Balance:     balance,
				Nonce:       g.Alloc[*address].Nonce,
			}
			g.Alloc[*address] = genesisAccount
		}
		if !withoutImplSlot {
			g.Alloc[*address].Storage[ImplementationSlot] = codeAddr.Hash()
		}

		if err := setupPredeploy(g, deployResults, storage, name, *address, codeAddr); err != nil {
			return err
		}
	}
	return nil
}

func SetDevOnlyL2Implementations(g *types.Genesis, storage state.StorageConfig, immutable immutables.ImmutableConfig) error {
	deployResults, err := immutables.BuildOptimism(immutable)
	if err != nil {
		return err
	}

	for name, address := range predeploys.Predeploys {
		if !UntouchablePredeploys[*address] {
			continue
		}

		if *address == predeploys.BobaL2Addr {
			continue
		}

		g.Alloc[*address] = types.GenesisAccount{
			Balance: big.NewInt(0),
			Storage: map[common.Hash]common.Hash{},
		}

		if err := setupPredeploy(g, deployResults, storage, name, *address, *address); err != nil {
			return err
		}

		code := g.Alloc[*address].Code
		if len(code) == 0 {
			return fmt.Errorf("code not set for %s", name)
		}
	}

	g.Alloc[predeploys.LegacyERC20ETHAddr] = types.GenesisAccount{
		Balance: big.NewInt(0),
		Storage: map[common.Hash]common.Hash{},
	}
	if err := setupPredeploy(g, deployResults, storage, "LegacyERC20ETH", predeploys.LegacyERC20ETHAddr, predeploys.LegacyERC20ETHAddr); err != nil {
		return fmt.Errorf("error setting up legacy eth: %w", err)
	}

	// This is to handle a very special case in L2GovernaceContract
	// Although we put _decimals as an input when we deploy the contract,
	// _decimals is not in the storage. It's set as the constant value, so we have to remove
	// this key from storage when we build the genesis state.
	g.Alloc[predeploys.BobaL2Addr] = types.GenesisAccount{
		Balance: big.NewInt(0),
		Storage: map[common.Hash]common.Hash{},
	}
	var _decimals interface{}
	if storageConfig, ok := storage["BobaL2"]; ok {
		_decimals = storageConfig["_decimals"]
		delete(storageConfig, "_decimals")
	} else {
		return fmt.Errorf("storage config not found for BobaL2")
	}
	if err := setupPredeploy(g, deployResults, storage, "BobaL2", predeploys.BobaL2Addr, predeploys.BobaL2Addr); err != nil {
		return fmt.Errorf("error setting up boba l2: %w", err)
	}
	storage["BobaL2"]["_decimals"] = _decimals

	return nil
}

func setupPredeploy(g *types.Genesis, deployResults immutables.DeploymentResults, storage state.StorageConfig, name string, proxyAddr common.Address, implAddr common.Address) error {
	balance := g.Alloc[implAddr].Balance
	if balance == nil {
		balance = big.NewInt(0)
	}
	// Use the generated bytecode when there are immutables
	// otherwise use the artifact deployed bytecode
	if bytecode, ok := deployResults[name]; ok {
		log.Info("Setting deployed bytecode with immutables", "name", name, "address", implAddr)
		genesisAccount := types.GenesisAccount{
			Constructor: g.Alloc[implAddr].Constructor,
			Code:        hexutil.MustDecode(bytecode),
			Storage:     g.Alloc[implAddr].Storage,
			Balance:     balance,
			Nonce:       g.Alloc[implAddr].Nonce,
		}
		g.Alloc[implAddr] = genesisAccount
	} else {
		depBytecode, err := bindings.GetDeployedBytecode(name)
		if err != nil {
			return err
		}
		log.Info("Setting deployed bytecode from solc compiler output", "name", name, "address", implAddr)
		genesisAccount := types.GenesisAccount{
			Constructor: g.Alloc[implAddr].Constructor,
			Code:        depBytecode,
			Storage:     g.Alloc[implAddr].Storage,
			Balance:     balance,
			Nonce:       g.Alloc[implAddr].Nonce,
		}
		g.Alloc[implAddr] = genesisAccount
	}

	// Set the storage values
	if storageConfig, ok := storage[name]; ok {
		log.Info("Setting storage", "name", name, "address", proxyAddr)
		if err := state.SetStorage(name, proxyAddr, storageConfig, g); err != nil {
			return err
		}
	}

	return nil
}

// Set balance field in genesis to zero for addresses
func SetBalanceToZero(g *types.Genesis) {
	for addr, account := range g.Alloc {
		account.Balance = big.NewInt(0)
		g.Alloc[addr] = account
	}
}
