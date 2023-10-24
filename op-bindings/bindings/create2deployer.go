// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Create2DeployerMetaData contains all meta data concerning the Create2Deployer contract.
var Create2DeployerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"codeHash\",\"type\":\"bytes32\"}],\"name\":\"computeAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"codeHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"}],\"name\":\"computeAddressWithDeployer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"code\",\"type\":\"bytes\"}],\"name\":\"deploy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"name\":\"deployERC1820Implementer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"payoutAddress\",\"type\":\"address\"}],\"name\":\"killCreate2Deployer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x6080604052600436106100a05760003560e01c80636447045411610064578063644704541461016a57806366cfa0571461018a578063715018a6146101aa5780638456cb59146101bf5780638da5cb5b146101d4578063f2fde38b146101f257600080fd5b8063076c37b2146100ac5780633f4ba83a146100ce578063481286e6146100e357806356299481146101205780635c975abb1461014057600080fd5b366100a757005b600080fd5b3480156100b857600080fd5b506100cc6100c736600461077b565b610212565b005b3480156100da57600080fd5b506100cc610277565b3480156100ef57600080fd5b506101036100fe36600461077b565b6102ab565b6040516001600160a01b0390911681526020015b60405180910390f35b34801561012c57600080fd5b5061010361013b3660046107b2565b610311565b34801561014c57600080fd5b50600054600160a01b900460ff166040519015158152602001610117565b34801561017657600080fd5b506100cc6101853660046107eb565b610372565b34801561019657600080fd5b506100cc6101a536600461081e565b6103de565b3480156101b657600080fd5b506100cc610419565b3480156101cb57600080fd5b506100cc61044d565b3480156101e057600080fd5b506000546001600160a01b0316610103565b3480156101fe57600080fd5b506100cc61020d3660046107eb565b61047f565b600054600160a01b900460ff16156102455760405162461bcd60e51b815260040161023c906108e2565b60405180910390fd5b61027282826040518060200161025a9061076e565b601f1982820381018352601f9091011660405261051a565b505050565b6000546001600160a01b031633146102a15760405162461bcd60e51b815260040161023c9061090c565b6102a961061c565b565b600061030a8383604080516001600160f81b03196020808301919091526bffffffffffffffffffffffff193060601b16602183015260358201859052605580830185905283518084039091018152607590920190925280519101206000905b9392505050565b604080516001600160f81b03196020808301919091526bffffffffffffffffffffffff19606085901b16602183015260358201869052605580830186905283518084039091018152607590920190925280519101206000905b949350505050565b6000546001600160a01b0316331461039c5760405162461bcd60e51b815260040161023c9061090c565b6040516001600160a01b038216904780156108fc02916000818181858888f193505050501580156103d1573d6000803e3d6000fd5b50806001600160a01b0316ff5b600054600160a01b900460ff16156104085760405162461bcd60e51b815260040161023c906108e2565b61041383838361051a565b50505050565b6000546001600160a01b031633146104435760405162461bcd60e51b815260040161023c9061090c565b6102a960006106b9565b6000546001600160a01b031633146104775760405162461bcd60e51b815260040161023c9061090c565b6102a9610709565b6000546001600160a01b031633146104a95760405162461bcd60e51b815260040161023c9061090c565b6001600160a01b03811661050e5760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b606482015260840161023c565b610517816106b9565b50565b6000808447101561056d5760405162461bcd60e51b815260206004820152601d60248201527f437265617465323a20696e73756666696369656e742062616c616e6365000000604482015260640161023c565b82516105bb5760405162461bcd60e51b815260206004820181905260248201527f437265617465323a2062797465636f6465206c656e677468206973207a65726f604482015260640161023c565b8383516020850187f590506001600160a01b03811661036a5760405162461bcd60e51b815260206004820152601960248201527f437265617465323a204661696c6564206f6e206465706c6f7900000000000000604482015260640161023c565b600054600160a01b900460ff1661066c5760405162461bcd60e51b815260206004820152601460248201527314185d5cd8589b194e881b9bdd081c185d5cd95960621b604482015260640161023c565b6000805460ff60a01b191690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa335b6040516001600160a01b03909116815260200160405180910390a1565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600054600160a01b900460ff16156107335760405162461bcd60e51b815260040161023c906108e2565b6000805460ff60a01b1916600160a01b1790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a25861069c3390565b6101348061094283390190565b6000806040838503121561078e57600080fd5b50508035926020909101359150565b6001600160a01b038116811461051757600080fd5b6000806000606084860312156107c757600080fd5b833592506020840135915060408401356107e08161079d565b809150509250925092565b6000602082840312156107fd57600080fd5b813561030a8161079d565b634e487b7160e01b600052604160045260246000fd5b60008060006060848603121561083357600080fd5b8335925060208401359150604084013567ffffffffffffffff8082111561085957600080fd5b818601915086601f83011261086d57600080fd5b81358181111561087f5761087f610808565b604051601f8201601f19908116603f011681019083821181831017156108a7576108a7610808565b816040528281528960208487010111156108c057600080fd5b8260208601602083013760006020848301015280955050505050509250925092565b60208082526010908201526f14185d5cd8589b194e881c185d5cd95960821b604082015260600190565b6020808252818101527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260408201526060019056fe608060405234801561001057600080fd5b50610114806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063249cb3fa14602d575b600080fd5b603c603836600460a4565b604e565b60405190815260200160405180910390f35b6000828152602081815260408083206001600160a01b038516845290915281205460ff16607b576000609d565b7fa2ef4600d742022d532d4747cb3547474667d6f13804902513b2ec01c848f4b45b9392505050565b6000806040838503121560b657600080fd5b8235915060208301356001600160a01b038116811460d357600080fd5b80915050925092905056fea2646970667358221220a5a496558254ee0cf3c67a46f475274d2a4e7c3fcd0a6926c382539e9f4e747f64736f6c63430008090033a264697066735822122058b32e980f80f9510cb90f6ad481aa6ca33b3d34a29adff4a2381aa52879574b64736f6c63430008090033",
}

// Create2DeployerABI is the input ABI used to generate the binding from.
// Deprecated: Use Create2DeployerMetaData.ABI instead.
var Create2DeployerABI = Create2DeployerMetaData.ABI

// Create2DeployerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Create2DeployerMetaData.Bin instead.
var Create2DeployerBin = Create2DeployerMetaData.Bin

// DeployCreate2Deployer deploys a new Ethereum contract, binding an instance of Create2Deployer to it.
func DeployCreate2Deployer(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Create2Deployer, error) {
	parsed, err := Create2DeployerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Create2DeployerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Create2Deployer{Create2DeployerCaller: Create2DeployerCaller{contract: contract}, Create2DeployerTransactor: Create2DeployerTransactor{contract: contract}, Create2DeployerFilterer: Create2DeployerFilterer{contract: contract}}, nil
}

// Create2Deployer is an auto generated Go binding around an Ethereum contract.
type Create2Deployer struct {
	Create2DeployerCaller     // Read-only binding to the contract
	Create2DeployerTransactor // Write-only binding to the contract
	Create2DeployerFilterer   // Log filterer for contract events
}

// Create2DeployerCaller is an auto generated read-only Go binding around an Ethereum contract.
type Create2DeployerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Create2DeployerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type Create2DeployerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Create2DeployerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Create2DeployerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Create2DeployerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Create2DeployerSession struct {
	Contract     *Create2Deployer  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Create2DeployerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Create2DeployerCallerSession struct {
	Contract *Create2DeployerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// Create2DeployerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Create2DeployerTransactorSession struct {
	Contract     *Create2DeployerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// Create2DeployerRaw is an auto generated low-level Go binding around an Ethereum contract.
type Create2DeployerRaw struct {
	Contract *Create2Deployer // Generic contract binding to access the raw methods on
}

// Create2DeployerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Create2DeployerCallerRaw struct {
	Contract *Create2DeployerCaller // Generic read-only contract binding to access the raw methods on
}

// Create2DeployerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Create2DeployerTransactorRaw struct {
	Contract *Create2DeployerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCreate2Deployer creates a new instance of Create2Deployer, bound to a specific deployed contract.
func NewCreate2Deployer(address common.Address, backend bind.ContractBackend) (*Create2Deployer, error) {
	contract, err := bindCreate2Deployer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Create2Deployer{Create2DeployerCaller: Create2DeployerCaller{contract: contract}, Create2DeployerTransactor: Create2DeployerTransactor{contract: contract}, Create2DeployerFilterer: Create2DeployerFilterer{contract: contract}}, nil
}

// NewCreate2DeployerCaller creates a new read-only instance of Create2Deployer, bound to a specific deployed contract.
func NewCreate2DeployerCaller(address common.Address, caller bind.ContractCaller) (*Create2DeployerCaller, error) {
	contract, err := bindCreate2Deployer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Create2DeployerCaller{contract: contract}, nil
}

// NewCreate2DeployerTransactor creates a new write-only instance of Create2Deployer, bound to a specific deployed contract.
func NewCreate2DeployerTransactor(address common.Address, transactor bind.ContractTransactor) (*Create2DeployerTransactor, error) {
	contract, err := bindCreate2Deployer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Create2DeployerTransactor{contract: contract}, nil
}

// NewCreate2DeployerFilterer creates a new log filterer instance of Create2Deployer, bound to a specific deployed contract.
func NewCreate2DeployerFilterer(address common.Address, filterer bind.ContractFilterer) (*Create2DeployerFilterer, error) {
	contract, err := bindCreate2Deployer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Create2DeployerFilterer{contract: contract}, nil
}

// bindCreate2Deployer binds a generic wrapper to an already deployed contract.
func bindCreate2Deployer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(Create2DeployerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Create2Deployer *Create2DeployerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Create2Deployer.Contract.Create2DeployerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Create2Deployer *Create2DeployerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.Contract.Create2DeployerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Create2Deployer *Create2DeployerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Create2Deployer.Contract.Create2DeployerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Create2Deployer *Create2DeployerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Create2Deployer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Create2Deployer *Create2DeployerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Create2Deployer *Create2DeployerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Create2Deployer.Contract.contract.Transact(opts, method, params...)
}

// ComputeAddress is a free data retrieval call binding the contract method 0x481286e6.
//
// Solidity: function computeAddress(bytes32 salt, bytes32 codeHash) view returns(address)
func (_Create2Deployer *Create2DeployerCaller) ComputeAddress(opts *bind.CallOpts, salt [32]byte, codeHash [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Create2Deployer.contract.Call(opts, &out, "computeAddress", salt, codeHash)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ComputeAddress is a free data retrieval call binding the contract method 0x481286e6.
//
// Solidity: function computeAddress(bytes32 salt, bytes32 codeHash) view returns(address)
func (_Create2Deployer *Create2DeployerSession) ComputeAddress(salt [32]byte, codeHash [32]byte) (common.Address, error) {
	return _Create2Deployer.Contract.ComputeAddress(&_Create2Deployer.CallOpts, salt, codeHash)
}

// ComputeAddress is a free data retrieval call binding the contract method 0x481286e6.
//
// Solidity: function computeAddress(bytes32 salt, bytes32 codeHash) view returns(address)
func (_Create2Deployer *Create2DeployerCallerSession) ComputeAddress(salt [32]byte, codeHash [32]byte) (common.Address, error) {
	return _Create2Deployer.Contract.ComputeAddress(&_Create2Deployer.CallOpts, salt, codeHash)
}

// ComputeAddressWithDeployer is a free data retrieval call binding the contract method 0x56299481.
//
// Solidity: function computeAddressWithDeployer(bytes32 salt, bytes32 codeHash, address deployer) pure returns(address)
func (_Create2Deployer *Create2DeployerCaller) ComputeAddressWithDeployer(opts *bind.CallOpts, salt [32]byte, codeHash [32]byte, deployer common.Address) (common.Address, error) {
	var out []interface{}
	err := _Create2Deployer.contract.Call(opts, &out, "computeAddressWithDeployer", salt, codeHash, deployer)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ComputeAddressWithDeployer is a free data retrieval call binding the contract method 0x56299481.
//
// Solidity: function computeAddressWithDeployer(bytes32 salt, bytes32 codeHash, address deployer) pure returns(address)
func (_Create2Deployer *Create2DeployerSession) ComputeAddressWithDeployer(salt [32]byte, codeHash [32]byte, deployer common.Address) (common.Address, error) {
	return _Create2Deployer.Contract.ComputeAddressWithDeployer(&_Create2Deployer.CallOpts, salt, codeHash, deployer)
}

// ComputeAddressWithDeployer is a free data retrieval call binding the contract method 0x56299481.
//
// Solidity: function computeAddressWithDeployer(bytes32 salt, bytes32 codeHash, address deployer) pure returns(address)
func (_Create2Deployer *Create2DeployerCallerSession) ComputeAddressWithDeployer(salt [32]byte, codeHash [32]byte, deployer common.Address) (common.Address, error) {
	return _Create2Deployer.Contract.ComputeAddressWithDeployer(&_Create2Deployer.CallOpts, salt, codeHash, deployer)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Create2Deployer *Create2DeployerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Create2Deployer.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Create2Deployer *Create2DeployerSession) Owner() (common.Address, error) {
	return _Create2Deployer.Contract.Owner(&_Create2Deployer.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Create2Deployer *Create2DeployerCallerSession) Owner() (common.Address, error) {
	return _Create2Deployer.Contract.Owner(&_Create2Deployer.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Create2Deployer *Create2DeployerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Create2Deployer.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Create2Deployer *Create2DeployerSession) Paused() (bool, error) {
	return _Create2Deployer.Contract.Paused(&_Create2Deployer.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Create2Deployer *Create2DeployerCallerSession) Paused() (bool, error) {
	return _Create2Deployer.Contract.Paused(&_Create2Deployer.CallOpts)
}

// Deploy is a paid mutator transaction binding the contract method 0x66cfa057.
//
// Solidity: function deploy(uint256 value, bytes32 salt, bytes code) returns()
func (_Create2Deployer *Create2DeployerTransactor) Deploy(opts *bind.TransactOpts, value *big.Int, salt [32]byte, code []byte) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "deploy", value, salt, code)
}

// Deploy is a paid mutator transaction binding the contract method 0x66cfa057.
//
// Solidity: function deploy(uint256 value, bytes32 salt, bytes code) returns()
func (_Create2Deployer *Create2DeployerSession) Deploy(value *big.Int, salt [32]byte, code []byte) (*types.Transaction, error) {
	return _Create2Deployer.Contract.Deploy(&_Create2Deployer.TransactOpts, value, salt, code)
}

// Deploy is a paid mutator transaction binding the contract method 0x66cfa057.
//
// Solidity: function deploy(uint256 value, bytes32 salt, bytes code) returns()
func (_Create2Deployer *Create2DeployerTransactorSession) Deploy(value *big.Int, salt [32]byte, code []byte) (*types.Transaction, error) {
	return _Create2Deployer.Contract.Deploy(&_Create2Deployer.TransactOpts, value, salt, code)
}

// DeployERC1820Implementer is a paid mutator transaction binding the contract method 0x076c37b2.
//
// Solidity: function deployERC1820Implementer(uint256 value, bytes32 salt) returns()
func (_Create2Deployer *Create2DeployerTransactor) DeployERC1820Implementer(opts *bind.TransactOpts, value *big.Int, salt [32]byte) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "deployERC1820Implementer", value, salt)
}

// DeployERC1820Implementer is a paid mutator transaction binding the contract method 0x076c37b2.
//
// Solidity: function deployERC1820Implementer(uint256 value, bytes32 salt) returns()
func (_Create2Deployer *Create2DeployerSession) DeployERC1820Implementer(value *big.Int, salt [32]byte) (*types.Transaction, error) {
	return _Create2Deployer.Contract.DeployERC1820Implementer(&_Create2Deployer.TransactOpts, value, salt)
}

// DeployERC1820Implementer is a paid mutator transaction binding the contract method 0x076c37b2.
//
// Solidity: function deployERC1820Implementer(uint256 value, bytes32 salt) returns()
func (_Create2Deployer *Create2DeployerTransactorSession) DeployERC1820Implementer(value *big.Int, salt [32]byte) (*types.Transaction, error) {
	return _Create2Deployer.Contract.DeployERC1820Implementer(&_Create2Deployer.TransactOpts, value, salt)
}

// KillCreate2Deployer is a paid mutator transaction binding the contract method 0x64470454.
//
// Solidity: function killCreate2Deployer(address payoutAddress) returns()
func (_Create2Deployer *Create2DeployerTransactor) KillCreate2Deployer(opts *bind.TransactOpts, payoutAddress common.Address) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "killCreate2Deployer", payoutAddress)
}

// KillCreate2Deployer is a paid mutator transaction binding the contract method 0x64470454.
//
// Solidity: function killCreate2Deployer(address payoutAddress) returns()
func (_Create2Deployer *Create2DeployerSession) KillCreate2Deployer(payoutAddress common.Address) (*types.Transaction, error) {
	return _Create2Deployer.Contract.KillCreate2Deployer(&_Create2Deployer.TransactOpts, payoutAddress)
}

// KillCreate2Deployer is a paid mutator transaction binding the contract method 0x64470454.
//
// Solidity: function killCreate2Deployer(address payoutAddress) returns()
func (_Create2Deployer *Create2DeployerTransactorSession) KillCreate2Deployer(payoutAddress common.Address) (*types.Transaction, error) {
	return _Create2Deployer.Contract.KillCreate2Deployer(&_Create2Deployer.TransactOpts, payoutAddress)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Create2Deployer *Create2DeployerTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Create2Deployer *Create2DeployerSession) Pause() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Pause(&_Create2Deployer.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Create2Deployer *Create2DeployerTransactorSession) Pause() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Pause(&_Create2Deployer.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Create2Deployer *Create2DeployerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Create2Deployer *Create2DeployerSession) RenounceOwnership() (*types.Transaction, error) {
	return _Create2Deployer.Contract.RenounceOwnership(&_Create2Deployer.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Create2Deployer *Create2DeployerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Create2Deployer.Contract.RenounceOwnership(&_Create2Deployer.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Create2Deployer *Create2DeployerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Create2Deployer *Create2DeployerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Create2Deployer.Contract.TransferOwnership(&_Create2Deployer.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Create2Deployer *Create2DeployerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Create2Deployer.Contract.TransferOwnership(&_Create2Deployer.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Create2Deployer *Create2DeployerTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Create2Deployer *Create2DeployerSession) Unpause() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Unpause(&_Create2Deployer.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Create2Deployer *Create2DeployerTransactorSession) Unpause() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Unpause(&_Create2Deployer.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Create2Deployer *Create2DeployerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Create2Deployer.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Create2Deployer *Create2DeployerSession) Receive() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Receive(&_Create2Deployer.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Create2Deployer *Create2DeployerTransactorSession) Receive() (*types.Transaction, error) {
	return _Create2Deployer.Contract.Receive(&_Create2Deployer.TransactOpts)
}

// Create2DeployerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Create2Deployer contract.
type Create2DeployerOwnershipTransferredIterator struct {
	Event *Create2DeployerOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Create2DeployerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Create2DeployerOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Create2DeployerOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Create2DeployerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Create2DeployerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Create2DeployerOwnershipTransferred represents a OwnershipTransferred event raised by the Create2Deployer contract.
type Create2DeployerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Create2Deployer *Create2DeployerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*Create2DeployerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Create2Deployer.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &Create2DeployerOwnershipTransferredIterator{contract: _Create2Deployer.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Create2Deployer *Create2DeployerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *Create2DeployerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Create2Deployer.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Create2DeployerOwnershipTransferred)
				if err := _Create2Deployer.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Create2Deployer *Create2DeployerFilterer) ParseOwnershipTransferred(log types.Log) (*Create2DeployerOwnershipTransferred, error) {
	event := new(Create2DeployerOwnershipTransferred)
	if err := _Create2Deployer.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Create2DeployerPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Create2Deployer contract.
type Create2DeployerPausedIterator struct {
	Event *Create2DeployerPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Create2DeployerPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Create2DeployerPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Create2DeployerPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Create2DeployerPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Create2DeployerPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Create2DeployerPaused represents a Paused event raised by the Create2Deployer contract.
type Create2DeployerPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Create2Deployer *Create2DeployerFilterer) FilterPaused(opts *bind.FilterOpts) (*Create2DeployerPausedIterator, error) {

	logs, sub, err := _Create2Deployer.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &Create2DeployerPausedIterator{contract: _Create2Deployer.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Create2Deployer *Create2DeployerFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *Create2DeployerPaused) (event.Subscription, error) {

	logs, sub, err := _Create2Deployer.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Create2DeployerPaused)
				if err := _Create2Deployer.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_Create2Deployer *Create2DeployerFilterer) ParsePaused(log types.Log) (*Create2DeployerPaused, error) {
	event := new(Create2DeployerPaused)
	if err := _Create2Deployer.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Create2DeployerUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the Create2Deployer contract.
type Create2DeployerUnpausedIterator struct {
	Event *Create2DeployerUnpaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *Create2DeployerUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Create2DeployerUnpaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(Create2DeployerUnpaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *Create2DeployerUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Create2DeployerUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Create2DeployerUnpaused represents a Unpaused event raised by the Create2Deployer contract.
type Create2DeployerUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Create2Deployer *Create2DeployerFilterer) FilterUnpaused(opts *bind.FilterOpts) (*Create2DeployerUnpausedIterator, error) {

	logs, sub, err := _Create2Deployer.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &Create2DeployerUnpausedIterator{contract: _Create2Deployer.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Create2Deployer *Create2DeployerFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *Create2DeployerUnpaused) (event.Subscription, error) {

	logs, sub, err := _Create2Deployer.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Create2DeployerUnpaused)
				if err := _Create2Deployer.contract.UnpackLog(event, "Unpaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_Create2Deployer *Create2DeployerFilterer) ParseUnpaused(log types.Log) (*Create2DeployerUnpaused, error) {
	event := new(Create2DeployerUnpaused)
	if err := _Create2Deployer.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
