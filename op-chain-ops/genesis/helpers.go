package genesis

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/immutables"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// codeNamespace represents the namespace of implementations of predeploys
	codeNamespace = common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000")
	// predeployNamespace represents the namespace of predeploys
	predeployNamespace = common.HexToAddress("0x4200000000000000000000000000000000000000")
	// bigPredeployNamespace represents the predeploy namespace as a big.Int
	bigPredeployNamespace = new(big.Int).SetBytes(predeployNamespace.Bytes())
	// bigCodeNamespace represents the predeploy namespace as a big.Int
	bigCodeNameSpace = new(big.Int).SetBytes(codeNamespace.Bytes())
	// implementationSlot represents the EIP 1967 implementation storage slot
	ImplementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	// implementationSlot represents the EIP 1967 admin storage slot
	AdminSlot = common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")
)

// DevAccounts represent the standard hardhat development accounts.
// These are funded if the deploy config has funding development
// accounts enabled.
var DevAccounts = []common.Address{
	common.HexToAddress("0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"),
	common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"),
	common.HexToAddress("0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec"),
	common.HexToAddress("0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f"),
	common.HexToAddress("0x2546BcD3c84621e976D8185a91A922aE77ECEc30"),
	common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"),
	common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	common.HexToAddress("0x71bE63f3384f5fb98995898A86B02Fb2426c5788"),
	common.HexToAddress("0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"),
	common.HexToAddress("0x90F79bf6EB2c4f870365E785982E1f101E93b906"),
	common.HexToAddress("0x976EA74026E726554dB657fA54763abd0C3a0aa9"),
	common.HexToAddress("0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"),
	common.HexToAddress("0xBcd4042DE499D14e55001CcbB24a551F3b954096"),
	common.HexToAddress("0xFABB0ac9d68B0B445fB7357272Ff202C5651694a"),
	common.HexToAddress("0xa0Ee7A142d267C1f36714E4a8F75612F20a79720"),
	common.HexToAddress("0xbDA5747bFD65F08deb54cb465eB87D40e51B197E"),
	common.HexToAddress("0xcd3B766CCDd6AE721141F452C550Ca635964ce71"),
	common.HexToAddress("0xdD2FD4581271e230360230F9337D5c0430Bf44C0"),
	common.HexToAddress("0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097"),
	common.HexToAddress("0xde3829a23df1479438622a08a116e8eb3f620bb5"),
	common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
}

// The devBalance is the amount of wei that a dev account is funded with.
var devBalance = hexutil.MustDecodeBig("0x200000000000000000000000000000000000000000000000000000000000000")

// FundDevAccounts will fund each of the development accounts.
func FundDevAccounts(db vm.StateDB) {
	for _, account := range DevAccounts {
		db.CreateAccount(account)
		db.AddBalance(account, devBalance)
	}
}

// SetProxies will set each of the proxies in the state. It requires
// a Proxy and ProxyAdmin deployment present so that the Proxy bytecode
// can be set in state and the ProxyAdmin can be set as the admin of the
// Proxy.
func SetProxies(hh *hardhat.Hardhat, db vm.StateDB) error {
	proxy, err := hh.GetArtifact("Proxy")
	if err != nil {
		return err
	}
	proxyAdmin, err := hh.GetDeployment("ProxyAdmin")
	if err != nil {
		return err
	}

	for i := uint64(0); i <= 2048; i++ {
		bigAddr := new(big.Int).Or(bigPredeployNamespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)

		// There is no proxy at the governance token address
		if addr == predeploys.GovernanceTokenAddr {
			continue
		}

		db.CreateAccount(addr)
		db.SetCode(addr, proxy.DeployedBytecode)
		db.SetState(addr, AdminSlot, proxyAdmin.Address.Hash())
	}
	return nil
}

// SetImplementations will set the implmentations of the contracts in the state
// and configure the proxies to point to the implementations. It also sets
// the appropriate storage values for each contract at the proxy address.
func SetImplementations(hh *hardhat.Hardhat, db vm.StateDB, storage StorageConfig) error {
	deployResults, err := immutables.BuildOptimism()
	if err != nil {
		return err
	}

	for name, address := range predeploys.Predeploys {
		// Get the hardhat artifact to access the deployed bytecode
		artifact, err := hh.GetArtifact(name)
		if err != nil {
			return err
		}

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
			// Set the implmentation slot in the predeploy proxy
			db.SetState(*address, ImplementationSlot, addr.Hash())
		}

		// Create the account
		db.CreateAccount(addr)

		// Use the genrated bytecode when there are immutables
		// otherwise use the artifact deployed bytecode
		if bytecode, ok := deployResults[name]; ok {
			db.SetCode(addr, bytecode)
		} else {
			db.SetCode(addr, artifact.DeployedBytecode)
		}

		// Set the storage values
		if storageConfig, ok := storage[name]; ok {
			layout, err := hh.GetStorageLayout(name)
			if err != nil {
				return err
			}
			slots, err := state.ComputeStorageSlots(layout, storageConfig)
			if err != nil {
				return err
			}
			// The storage values must go in the proxy address
			for _, slot := range slots {
				db.SetState(*address, slot.Key, slot.Value)
			}
		}

		code := db.GetCode(addr)
		if len(code) == 0 {
			return fmt.Errorf("code not set for %s", name)
		}
	}
	return nil
}

// AddressToCodeNamespace takes a predeploy address and computes
// the implmentation address that the implementation should be deployed at
func AddressToCodeNamespace(addr common.Address) (common.Address, error) {
	bytesAddr := addr.Bytes()
	if !bytes.Equal(bytesAddr[0:2], []byte{0x42, 0x00}) {
		return common.Address{}, fmt.Errorf("cannot handle non predeploy: %s", addr)
	}
	bigAddress := new(big.Int).SetBytes(bytesAddr[18:])
	num := new(big.Int).Or(bigCodeNameSpace, bigAddress)
	return common.BigToAddress(num), nil
}

// Get the storage layout of the L2ToL1MessagePasser
// Iterate over the storage layout to know which storage slots to ignore
// Iterate over each storage slot, compute the migration
func MigrateDepositHashes(hh *hardhat.Hardhat, db vm.StateDB) error {
	layout, err := hh.GetStorageLayout("L2ToL1MessagePasser")
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

	db.ForEachStorage(predeploys.L2ToL1MessagePasserAddr, func(key, value common.Hash) bool {
		if _, ok := ignore[key]; ok {
			return true
		}
		// TODO(tynes): Do the value migration here
		return true
	})
	return nil
}

// getBlockFromTag will resolve a Block given an rpc block tag
func getBlockFromTag(chain ethereum.ChainReader, tag rpc.BlockNumberOrHash) (*types.Block, error) {
	if hash, ok := tag.Hash(); ok {
		block, err := chain.BlockByHash(context.Background(), hash)
		if err != nil {
			return nil, err
		}
		return block, nil
	} else if num, ok := tag.Number(); ok {
		blockNumber := new(big.Int).SetInt64(num.Int64())
		block, err := chain.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			return nil, err
		}
		return block, nil
	} else {
		return nil, fmt.Errorf("invalid block tag: %v", tag)
	}
}
