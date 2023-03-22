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

// OptimismModuleMetaData contains all meta data concerning the OptimismModule contract.
var OptimismModuleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"getSequencer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"sequencers\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"stake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// OptimismModuleABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismModuleMetaData.ABI instead.
var OptimismModuleABI = OptimismModuleMetaData.ABI

// OptimismModule is an auto generated Go binding around an Ethereum contract.
type OptimismModule struct {
	OptimismModuleCaller     // Read-only binding to the contract
	OptimismModuleTransactor // Write-only binding to the contract
	OptimismModuleFilterer   // Log filterer for contract events
}

// OptimismModuleCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismModuleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismModuleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismModuleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismModuleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismModuleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismModuleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismModuleSession struct {
	Contract     *OptimismModule   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismModuleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismModuleCallerSession struct {
	Contract *OptimismModuleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OptimismModuleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismModuleTransactorSession struct {
	Contract     *OptimismModuleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OptimismModuleRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismModuleRaw struct {
	Contract *OptimismModule // Generic contract binding to access the raw methods on
}

// OptimismModuleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismModuleCallerRaw struct {
	Contract *OptimismModuleCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismModuleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismModuleTransactorRaw struct {
	Contract *OptimismModuleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismModule creates a new instance of OptimismModule, bound to a specific deployed contract.
func NewOptimismModule(address common.Address, backend bind.ContractBackend) (*OptimismModule, error) {
	contract, err := bindOptimismModule(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismModule{OptimismModuleCaller: OptimismModuleCaller{contract: contract}, OptimismModuleTransactor: OptimismModuleTransactor{contract: contract}, OptimismModuleFilterer: OptimismModuleFilterer{contract: contract}}, nil
}

// NewOptimismModuleCaller creates a new read-only instance of OptimismModule, bound to a specific deployed contract.
func NewOptimismModuleCaller(address common.Address, caller bind.ContractCaller) (*OptimismModuleCaller, error) {
	contract, err := bindOptimismModule(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismModuleCaller{contract: contract}, nil
}

// NewOptimismModuleTransactor creates a new write-only instance of OptimismModule, bound to a specific deployed contract.
func NewOptimismModuleTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismModuleTransactor, error) {
	contract, err := bindOptimismModule(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismModuleTransactor{contract: contract}, nil
}

// NewOptimismModuleFilterer creates a new log filterer instance of OptimismModule, bound to a specific deployed contract.
func NewOptimismModuleFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismModuleFilterer, error) {
	contract, err := bindOptimismModule(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismModuleFilterer{contract: contract}, nil
}

// bindOptimismModule binds a generic wrapper to an already deployed contract.
func bindOptimismModule(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OptimismModuleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismModule *OptimismModuleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismModule.Contract.OptimismModuleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismModule *OptimismModuleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismModule.Contract.OptimismModuleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismModule *OptimismModuleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismModule.Contract.OptimismModuleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismModule *OptimismModuleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismModule.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismModule *OptimismModuleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismModule.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismModule *OptimismModuleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismModule.Contract.contract.Transact(opts, method, params...)
}

// GetSequencer is a free data retrieval call binding the contract method 0x4d96a90a.
//
// Solidity: function getSequencer() view returns(address)
func (_OptimismModule *OptimismModuleCaller) GetSequencer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismModule.contract.Call(opts, &out, "getSequencer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetSequencer is a free data retrieval call binding the contract method 0x4d96a90a.
//
// Solidity: function getSequencer() view returns(address)
func (_OptimismModule *OptimismModuleSession) GetSequencer() (common.Address, error) {
	return _OptimismModule.Contract.GetSequencer(&_OptimismModule.CallOpts)
}

// GetSequencer is a free data retrieval call binding the contract method 0x4d96a90a.
//
// Solidity: function getSequencer() view returns(address)
func (_OptimismModule *OptimismModuleCallerSession) GetSequencer() (common.Address, error) {
	return _OptimismModule.Contract.GetSequencer(&_OptimismModule.CallOpts)
}

// Sequencers is a free data retrieval call binding the contract method 0x6ba7ccff.
//
// Solidity: function sequencers(uint256 ) view returns(address)
func (_OptimismModule *OptimismModuleCaller) Sequencers(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _OptimismModule.contract.Call(opts, &out, "sequencers", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Sequencers is a free data retrieval call binding the contract method 0x6ba7ccff.
//
// Solidity: function sequencers(uint256 ) view returns(address)
func (_OptimismModule *OptimismModuleSession) Sequencers(arg0 *big.Int) (common.Address, error) {
	return _OptimismModule.Contract.Sequencers(&_OptimismModule.CallOpts, arg0)
}

// Sequencers is a free data retrieval call binding the contract method 0x6ba7ccff.
//
// Solidity: function sequencers(uint256 ) view returns(address)
func (_OptimismModule *OptimismModuleCallerSession) Sequencers(arg0 *big.Int) (common.Address, error) {
	return _OptimismModule.Contract.Sequencers(&_OptimismModule.CallOpts, arg0)
}

// Stake is a paid mutator transaction binding the contract method 0x3a4b66f1.
//
// Solidity: function stake() returns()
func (_OptimismModule *OptimismModuleTransactor) Stake(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismModule.contract.Transact(opts, "stake")
}

// Stake is a paid mutator transaction binding the contract method 0x3a4b66f1.
//
// Solidity: function stake() returns()
func (_OptimismModule *OptimismModuleSession) Stake() (*types.Transaction, error) {
	return _OptimismModule.Contract.Stake(&_OptimismModule.TransactOpts)
}

// Stake is a paid mutator transaction binding the contract method 0x3a4b66f1.
//
// Solidity: function stake() returns()
func (_OptimismModule *OptimismModuleTransactorSession) Stake() (*types.Transaction, error) {
	return _OptimismModule.Contract.Stake(&_OptimismModule.TransactOpts)
}
