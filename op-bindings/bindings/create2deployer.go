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
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"codeHash\",\"type\":\"bytes32\"}],\"name\":\"computeAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"codeHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"}],\"name\":\"computeAddressWithDeployer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"code\",\"type\":\"bytes\"}],\"name\":\"deploy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"}],\"name\":\"deployERC1820Implementer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610630806100206000396000f3fe6080604052600436106100435760003560e01c8063076c37b21461004f578063481286e61461007157806356299481146100ba57806366cfa057146100da57600080fd5b3661004a57005b600080fd5b34801561005b57600080fd5b5061006f61006a366004610327565b6100fa565b005b34801561007d57600080fd5b5061009161008c366004610327565b61014a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b3480156100c657600080fd5b506100916100d5366004610349565b61015d565b3480156100e657600080fd5b5061006f6100f53660046103ca565b610172565b61014582826040518060200161010f9061031a565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe082820381018352601f90910116604052610183565b505050565b600061015683836102e7565b9392505050565b600061016a8484846102f0565b949350505050565b61017d838383610183565b50505050565b6000834710156101f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f437265617465323a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b815160000361025f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f437265617465323a2062797465636f6465206c656e677468206973207a65726f60448201526064016101eb565b8282516020840186f5905073ffffffffffffffffffffffffffffffffffffffff8116610156576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f437265617465323a204661696c6564206f6e206465706c6f790000000000000060448201526064016101eb565b60006101568383305b6000604051836040820152846020820152828152600b8101905060ff815360559020949350505050565b61014e806104ad83390190565b6000806040838503121561033a57600080fd5b50508035926020909101359150565b60008060006060848603121561035e57600080fd5b8335925060208401359150604084013573ffffffffffffffffffffffffffffffffffffffff8116811461039057600080fd5b809150509250925092565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000806000606084860312156103df57600080fd5b8335925060208401359150604084013567ffffffffffffffff8082111561040557600080fd5b818601915086601f83011261041957600080fd5b81358181111561042b5761042b61039b565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156104715761047161039b565b8160405282815289602084870101111561048a57600080fd5b826020860160208301376000602084830101528095505050505050925092509256fe608060405234801561001057600080fd5b5061012e806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063249cb3fa14602d575b600080fd5b603c603836600460b1565b604e565b60405190815260200160405180910390f35b60008281526020818152604080832073ffffffffffffffffffffffffffffffffffffffff8516845290915281205460ff16608857600060aa565b7fa2ef4600d742022d532d4747cb3547474667d6f13804902513b2ec01c848f4b45b9392505050565b6000806040838503121560c357600080fd5b82359150602083013573ffffffffffffffffffffffffffffffffffffffff8116811460ed57600080fd5b80915050925092905056fea26469706673582212205ffd4e6cede7d06a5daf93d48d0541fc68189eeb16608c1999a82063b666eb1164736f6c63430008130033a2646970667358221220fdc4a0fe96e3b21c108ca155438d37c9143fb01278a3c1d274948bad89c564ba64736f6c63430008130033",
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
