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

// DeterministicDeploymentProxyMetaData contains all meta data concerning the DeterministicDeploymentProxy contract.
var DeterministicDeploymentProxyMetaData = &bind.MetaData{
	ABI: "[{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\",\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"salt\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"creationCode\",\"type\":\"bytes\"}]}]",
	Bin: "0x604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3",
}

// DeterministicDeploymentProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use DeterministicDeploymentProxyMetaData.ABI instead.
var DeterministicDeploymentProxyABI = DeterministicDeploymentProxyMetaData.ABI

// DeterministicDeploymentProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DeterministicDeploymentProxyMetaData.Bin instead.
var DeterministicDeploymentProxyBin = DeterministicDeploymentProxyMetaData.Bin

// DeployDeterministicDeploymentProxy deploys a new Ethereum contract, binding an instance of DeterministicDeploymentProxy to it.
func DeployDeterministicDeploymentProxy(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DeterministicDeploymentProxy, error) {
	parsed, err := DeterministicDeploymentProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DeterministicDeploymentProxyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DeterministicDeploymentProxy{DeterministicDeploymentProxyCaller: DeterministicDeploymentProxyCaller{contract: contract}, DeterministicDeploymentProxyTransactor: DeterministicDeploymentProxyTransactor{contract: contract}, DeterministicDeploymentProxyFilterer: DeterministicDeploymentProxyFilterer{contract: contract}}, nil
}

// DeterministicDeploymentProxy is an auto generated Go binding around an Ethereum contract.
type DeterministicDeploymentProxy struct {
	DeterministicDeploymentProxyCaller     // Read-only binding to the contract
	DeterministicDeploymentProxyTransactor // Write-only binding to the contract
	DeterministicDeploymentProxyFilterer   // Log filterer for contract events
}

// DeterministicDeploymentProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type DeterministicDeploymentProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeterministicDeploymentProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DeterministicDeploymentProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeterministicDeploymentProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DeterministicDeploymentProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeterministicDeploymentProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DeterministicDeploymentProxySession struct {
	Contract     *DeterministicDeploymentProxy // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// DeterministicDeploymentProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DeterministicDeploymentProxyCallerSession struct {
	Contract *DeterministicDeploymentProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// DeterministicDeploymentProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DeterministicDeploymentProxyTransactorSession struct {
	Contract     *DeterministicDeploymentProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// DeterministicDeploymentProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type DeterministicDeploymentProxyRaw struct {
	Contract *DeterministicDeploymentProxy // Generic contract binding to access the raw methods on
}

// DeterministicDeploymentProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DeterministicDeploymentProxyCallerRaw struct {
	Contract *DeterministicDeploymentProxyCaller // Generic read-only contract binding to access the raw methods on
}

// DeterministicDeploymentProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DeterministicDeploymentProxyTransactorRaw struct {
	Contract *DeterministicDeploymentProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDeterministicDeploymentProxy creates a new instance of DeterministicDeploymentProxy, bound to a specific deployed contract.
func NewDeterministicDeploymentProxy(address common.Address, backend bind.ContractBackend) (*DeterministicDeploymentProxy, error) {
	contract, err := bindDeterministicDeploymentProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DeterministicDeploymentProxy{DeterministicDeploymentProxyCaller: DeterministicDeploymentProxyCaller{contract: contract}, DeterministicDeploymentProxyTransactor: DeterministicDeploymentProxyTransactor{contract: contract}, DeterministicDeploymentProxyFilterer: DeterministicDeploymentProxyFilterer{contract: contract}}, nil
}

// NewDeterministicDeploymentProxyCaller creates a new read-only instance of DeterministicDeploymentProxy, bound to a specific deployed contract.
func NewDeterministicDeploymentProxyCaller(address common.Address, caller bind.ContractCaller) (*DeterministicDeploymentProxyCaller, error) {
	contract, err := bindDeterministicDeploymentProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DeterministicDeploymentProxyCaller{contract: contract}, nil
}

// NewDeterministicDeploymentProxyTransactor creates a new write-only instance of DeterministicDeploymentProxy, bound to a specific deployed contract.
func NewDeterministicDeploymentProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*DeterministicDeploymentProxyTransactor, error) {
	contract, err := bindDeterministicDeploymentProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DeterministicDeploymentProxyTransactor{contract: contract}, nil
}

// NewDeterministicDeploymentProxyFilterer creates a new log filterer instance of DeterministicDeploymentProxy, bound to a specific deployed contract.
func NewDeterministicDeploymentProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*DeterministicDeploymentProxyFilterer, error) {
	contract, err := bindDeterministicDeploymentProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DeterministicDeploymentProxyFilterer{contract: contract}, nil
}

// bindDeterministicDeploymentProxy binds a generic wrapper to an already deployed contract.
func bindDeterministicDeploymentProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DeterministicDeploymentProxyABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DeterministicDeploymentProxy.Contract.DeterministicDeploymentProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.DeterministicDeploymentProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.DeterministicDeploymentProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DeterministicDeploymentProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.contract.Transact(opts, method, params...)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.Fallback(&_DeterministicDeploymentProxy.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DeterministicDeploymentProxy *DeterministicDeploymentProxyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DeterministicDeploymentProxy.Contract.Fallback(&_DeterministicDeploymentProxy.TransactOpts, calldata)
}
