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

// SafeSingletonFactoryMetaData contains all meta data concerning the SafeSingletonFactory contract.
var SafeSingletonFactoryMetaData = &bind.MetaData{
	ABI: "[{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\",\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"creationCode\",\"type\":\"bytes\"}]}]",
	Bin: "0x604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3",
}

// SafeSingletonFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use SafeSingletonFactoryMetaData.ABI instead.
var SafeSingletonFactoryABI = SafeSingletonFactoryMetaData.ABI

// SafeSingletonFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SafeSingletonFactoryMetaData.Bin instead.
var SafeSingletonFactoryBin = SafeSingletonFactoryMetaData.Bin

// DeploySafeSingletonFactory deploys a new Ethereum contract, binding an instance of SafeSingletonFactory to it.
func DeploySafeSingletonFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SafeSingletonFactory, error) {
	parsed, err := SafeSingletonFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SafeSingletonFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SafeSingletonFactory{SafeSingletonFactoryCaller: SafeSingletonFactoryCaller{contract: contract}, SafeSingletonFactoryTransactor: SafeSingletonFactoryTransactor{contract: contract}, SafeSingletonFactoryFilterer: SafeSingletonFactoryFilterer{contract: contract}}, nil
}

// SafeSingletonFactory is an auto generated Go binding around an Ethereum contract.
type SafeSingletonFactory struct {
	SafeSingletonFactoryCaller     // Read-only binding to the contract
	SafeSingletonFactoryTransactor // Write-only binding to the contract
	SafeSingletonFactoryFilterer   // Log filterer for contract events
}

// SafeSingletonFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type SafeSingletonFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeSingletonFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SafeSingletonFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeSingletonFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SafeSingletonFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SafeSingletonFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SafeSingletonFactorySession struct {
	Contract     *SafeSingletonFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// SafeSingletonFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SafeSingletonFactoryCallerSession struct {
	Contract *SafeSingletonFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// SafeSingletonFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SafeSingletonFactoryTransactorSession struct {
	Contract     *SafeSingletonFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// SafeSingletonFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type SafeSingletonFactoryRaw struct {
	Contract *SafeSingletonFactory // Generic contract binding to access the raw methods on
}

// SafeSingletonFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SafeSingletonFactoryCallerRaw struct {
	Contract *SafeSingletonFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// SafeSingletonFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SafeSingletonFactoryTransactorRaw struct {
	Contract *SafeSingletonFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSafeSingletonFactory creates a new instance of SafeSingletonFactory, bound to a specific deployed contract.
func NewSafeSingletonFactory(address common.Address, backend bind.ContractBackend) (*SafeSingletonFactory, error) {
	contract, err := bindSafeSingletonFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SafeSingletonFactory{SafeSingletonFactoryCaller: SafeSingletonFactoryCaller{contract: contract}, SafeSingletonFactoryTransactor: SafeSingletonFactoryTransactor{contract: contract}, SafeSingletonFactoryFilterer: SafeSingletonFactoryFilterer{contract: contract}}, nil
}

// NewSafeSingletonFactoryCaller creates a new read-only instance of SafeSingletonFactory, bound to a specific deployed contract.
func NewSafeSingletonFactoryCaller(address common.Address, caller bind.ContractCaller) (*SafeSingletonFactoryCaller, error) {
	contract, err := bindSafeSingletonFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SafeSingletonFactoryCaller{contract: contract}, nil
}

// NewSafeSingletonFactoryTransactor creates a new write-only instance of SafeSingletonFactory, bound to a specific deployed contract.
func NewSafeSingletonFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*SafeSingletonFactoryTransactor, error) {
	contract, err := bindSafeSingletonFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SafeSingletonFactoryTransactor{contract: contract}, nil
}

// NewSafeSingletonFactoryFilterer creates a new log filterer instance of SafeSingletonFactory, bound to a specific deployed contract.
func NewSafeSingletonFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*SafeSingletonFactoryFilterer, error) {
	contract, err := bindSafeSingletonFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SafeSingletonFactoryFilterer{contract: contract}, nil
}

// bindSafeSingletonFactory binds a generic wrapper to an already deployed contract.
func bindSafeSingletonFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SafeSingletonFactoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeSingletonFactory *SafeSingletonFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeSingletonFactory.Contract.SafeSingletonFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeSingletonFactory *SafeSingletonFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.SafeSingletonFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeSingletonFactory *SafeSingletonFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.SafeSingletonFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SafeSingletonFactory *SafeSingletonFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SafeSingletonFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SafeSingletonFactory *SafeSingletonFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SafeSingletonFactory *SafeSingletonFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeSingletonFactory *SafeSingletonFactoryTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _SafeSingletonFactory.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeSingletonFactory *SafeSingletonFactorySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.Fallback(&_SafeSingletonFactory.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_SafeSingletonFactory *SafeSingletonFactoryTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _SafeSingletonFactory.Contract.Fallback(&_SafeSingletonFactory.TransactOpts, calldata)
}
