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

// L1BlockNumberMetaData contains all meta data concerning the L1BlockNumber contract.
var L1BlockNumberMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"fallback\",\"stateMutability\":\"payable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"getL1BlockNumber\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610219806100206000396000f3fe60806040526004361061002d5760003560e01c806354fd4d5014610052578063b9b3efe9146100b157610048565b3661004857600061003c6100d4565b90508060005260206000f35b600061003c6100d4565b34801561005e57600080fd5b5061009b6040518060400160405280600581526020017f312e312e3000000000000000000000000000000000000000000000000000000081525081565b6040516100a89190610168565b60405180910390f35b3480156100bd57600080fd5b506100c66100d4565b6040519081526020016100a8565b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16638381f58a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610135573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061015991906101db565b67ffffffffffffffff16905090565b600060208083528351808285015260005b8181101561019557858101830151858201604001528201610179565b818111156101a7576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b6000602082840312156101ed57600080fd5b815167ffffffffffffffff8116811461020557600080fd5b939250505056fea164736f6c634300080f000a",
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
	parsed, err := abi.JSON(strings.NewReader(L1BlockNumberABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
