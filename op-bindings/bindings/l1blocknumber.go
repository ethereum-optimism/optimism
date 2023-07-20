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
	_ = abi.ConvertType
)

// L1BlockNumberMetaData contains all meta data concerning the L1BlockNumber contract.
var L1BlockNumberMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"getL1BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60e060405234801561001057600080fd5b5060016080819052600060a081905260c08290528161059461004a8339600061018d015260006101640152600061013b01526105946000f3fe60806040526004361061002d5760003560e01c806354fd4d5014610052578063b9b3efe91461007d57610048565b3661004857600061003c6100a0565b90508060005260206000f35b600061003c6100a0565b34801561005e57600080fd5b50610067610134565b6040516100749190610344565b60405180910390f35b34801561008957600080fd5b506100926100a0565b604051908152602001610074565b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16638381f58a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610101573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906101259190610395565b67ffffffffffffffff16905090565b606061015f7f00000000000000000000000000000000000000000000000000000000000000006101d7565b6101887f00000000000000000000000000000000000000000000000000000000000000006101d7565b6101b17f00000000000000000000000000000000000000000000000000000000000000006101d7565b6040516020016101c3939291906103c6565b604051602081830303815290604052905090565b60608160000361021a57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610244578061022e8161046b565b915061023d9050600a836104d2565b915061021e565b60008167ffffffffffffffff81111561025f5761025f6104e6565b6040519080825280601f01601f191660200182016040528015610289576020820181803683370190505b5090505b841561030c5761029e600183610515565b91506102ab600a8661052c565b6102b6906030610540565b60f81b8183815181106102cb576102cb610558565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610305600a866104d2565b945061028d565b949350505050565b60005b8381101561032f578181015183820152602001610317565b8381111561033e576000848401525b50505050565b6020815260008251806020840152610363816040850160208701610314565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000602082840312156103a757600080fd5b815167ffffffffffffffff811681146103bf57600080fd5b9392505050565b600084516103d8818460208901610314565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610414816001850160208a01610314565b6001920191820152835161042f816002840160208801610314565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361049c5761049c61043c565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826104e1576104e16104a3565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156105275761052761043c565b500390565b60008261053b5761053b6104a3565b500690565b600082198211156105535761055361043c565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// L1BlockNumberABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BlockNumberMetaData.ABI instead.
var L1BlockNumberABI = L1BlockNumberMetaData.ABI

// L1BlockNumberBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1BlockNumberMetaData.Bin instead.
var L1BlockNumberBin = L1BlockNumberMetaData.Bin

// DeployL1BlockNumber deploys a new Ethereum contract, binding an instance of L1BlockNumber to it.
func DeployL1BlockNumber(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1BlockNumber, error) {
	parsed, err := L1BlockNumberMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1BlockNumberBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1BlockNumber{L1BlockNumberCaller: L1BlockNumberCaller{contract: contract}, L1BlockNumberTransactor: L1BlockNumberTransactor{contract: contract}, L1BlockNumberFilterer: L1BlockNumberFilterer{contract: contract}}, nil
}

// L1BlockNumber is an auto generated Go binding around an Ethereum contract.
type L1BlockNumber struct {
	L1BlockNumberCaller     // Read-only binding to the contract
	L1BlockNumberTransactor // Write-only binding to the contract
	L1BlockNumberFilterer   // Log filterer for contract events
}

// L1BlockNumberCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlockNumberCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockNumberTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlockNumberTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockNumberFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BlockNumberFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockNumberSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BlockNumberSession struct {
	Contract     *L1BlockNumber    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BlockNumberCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BlockNumberCallerSession struct {
	Contract *L1BlockNumberCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// L1BlockNumberTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BlockNumberTransactorSession struct {
	Contract     *L1BlockNumberTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// L1BlockNumberRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BlockNumberRaw struct {
	Contract *L1BlockNumber // Generic contract binding to access the raw methods on
}

// L1BlockNumberCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BlockNumberCallerRaw struct {
	Contract *L1BlockNumberCaller // Generic read-only contract binding to access the raw methods on
}

// L1BlockNumberTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BlockNumberTransactorRaw struct {
	Contract *L1BlockNumberTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1BlockNumber creates a new instance of L1BlockNumber, bound to a specific deployed contract.
func NewL1BlockNumber(address common.Address, backend bind.ContractBackend) (*L1BlockNumber, error) {
	contract, err := bindL1BlockNumber(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1BlockNumber{L1BlockNumberCaller: L1BlockNumberCaller{contract: contract}, L1BlockNumberTransactor: L1BlockNumberTransactor{contract: contract}, L1BlockNumberFilterer: L1BlockNumberFilterer{contract: contract}}, nil
}

// NewL1BlockNumberCaller creates a new read-only instance of L1BlockNumber, bound to a specific deployed contract.
func NewL1BlockNumberCaller(address common.Address, caller bind.ContractCaller) (*L1BlockNumberCaller, error) {
	contract, err := bindL1BlockNumber(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockNumberCaller{contract: contract}, nil
}

// NewL1BlockNumberTransactor creates a new write-only instance of L1BlockNumber, bound to a specific deployed contract.
func NewL1BlockNumberTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlockNumberTransactor, error) {
	contract, err := bindL1BlockNumber(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockNumberTransactor{contract: contract}, nil
}

// NewL1BlockNumberFilterer creates a new log filterer instance of L1BlockNumber, bound to a specific deployed contract.
func NewL1BlockNumberFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BlockNumberFilterer, error) {
	contract, err := bindL1BlockNumber(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BlockNumberFilterer{contract: contract}, nil
}

// bindL1BlockNumber binds a generic wrapper to an already deployed contract.
func bindL1BlockNumber(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1BlockNumberMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BlockNumber *L1BlockNumberRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BlockNumber.Contract.L1BlockNumberCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BlockNumber *L1BlockNumberRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.L1BlockNumberTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BlockNumber *L1BlockNumberRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.L1BlockNumberTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1BlockNumber *L1BlockNumberCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1BlockNumber.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1BlockNumber *L1BlockNumberTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1BlockNumber *L1BlockNumberTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.contract.Transact(opts, method, params...)
}

// GetL1BlockNumber is a free data retrieval call binding the contract method 0xb9b3efe9.
//
// Solidity: function getL1BlockNumber() view returns(uint256)
func (_L1BlockNumber *L1BlockNumberCaller) GetL1BlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1BlockNumber.contract.Call(opts, &out, "getL1BlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1BlockNumber is a free data retrieval call binding the contract method 0xb9b3efe9.
//
// Solidity: function getL1BlockNumber() view returns(uint256)
func (_L1BlockNumber *L1BlockNumberSession) GetL1BlockNumber() (*big.Int, error) {
	return _L1BlockNumber.Contract.GetL1BlockNumber(&_L1BlockNumber.CallOpts)
}

// GetL1BlockNumber is a free data retrieval call binding the contract method 0xb9b3efe9.
//
// Solidity: function getL1BlockNumber() view returns(uint256)
func (_L1BlockNumber *L1BlockNumberCallerSession) GetL1BlockNumber() (*big.Int, error) {
	return _L1BlockNumber.Contract.GetL1BlockNumber(&_L1BlockNumber.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1BlockNumber *L1BlockNumberCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1BlockNumber.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1BlockNumber *L1BlockNumberSession) Version() (string, error) {
	return _L1BlockNumber.Contract.Version(&_L1BlockNumber.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1BlockNumber *L1BlockNumberCallerSession) Version() (string, error) {
	return _L1BlockNumber.Contract.Version(&_L1BlockNumber.CallOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_L1BlockNumber *L1BlockNumberTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _L1BlockNumber.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_L1BlockNumber *L1BlockNumberSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.Fallback(&_L1BlockNumber.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_L1BlockNumber *L1BlockNumberTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _L1BlockNumber.Contract.Fallback(&_L1BlockNumber.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1BlockNumber *L1BlockNumberTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1BlockNumber.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1BlockNumber *L1BlockNumberSession) Receive() (*types.Transaction, error) {
	return _L1BlockNumber.Contract.Receive(&_L1BlockNumber.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1BlockNumber *L1BlockNumberTransactorSession) Receive() (*types.Transaction, error) {
	return _L1BlockNumber.Contract.Receive(&_L1BlockNumber.TransactOpts)
}
