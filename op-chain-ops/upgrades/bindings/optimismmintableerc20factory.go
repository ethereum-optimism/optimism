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

// OptimismMintableERC20FactoryMetaData contains all meta data concerning the OptimismMintableERC20Factory contract.
var OptimismMintableERC20FactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BRIDGE\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"name\":\"createOptimismMintableERC20\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"}],\"name\":\"createOptimismMintableERC20WithDecimals\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"name\":\"createStandardL2Token\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"}],\"name\":\"OptimismMintableERC20Created\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"}],\"name\":\"StandardL2TokenCreated\",\"type\":\"event\"}]",
}

// OptimismMintableERC20FactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismMintableERC20FactoryMetaData.ABI instead.
var OptimismMintableERC20FactoryABI = OptimismMintableERC20FactoryMetaData.ABI

// OptimismMintableERC20Factory is an auto generated Go binding around an Ethereum contract.
type OptimismMintableERC20Factory struct {
	OptimismMintableERC20FactoryCaller     // Read-only binding to the contract
	OptimismMintableERC20FactoryTransactor // Write-only binding to the contract
	OptimismMintableERC20FactoryFilterer   // Log filterer for contract events
}

// OptimismMintableERC20FactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismMintableERC20FactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismMintableERC20FactorySession struct {
	Contract     *OptimismMintableERC20Factory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// OptimismMintableERC20FactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismMintableERC20FactoryCallerSession struct {
	Contract *OptimismMintableERC20FactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// OptimismMintableERC20FactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismMintableERC20FactoryTransactorSession struct {
	Contract     *OptimismMintableERC20FactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// OptimismMintableERC20FactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryRaw struct {
	Contract *OptimismMintableERC20Factory // Generic contract binding to access the raw methods on
}

// OptimismMintableERC20FactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryCallerRaw struct {
	Contract *OptimismMintableERC20FactoryCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismMintableERC20FactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryTransactorRaw struct {
	Contract *OptimismMintableERC20FactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismMintableERC20Factory creates a new instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20Factory(address common.Address, backend bind.ContractBackend) (*OptimismMintableERC20Factory, error) {
	contract, err := bindOptimismMintableERC20Factory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20Factory{OptimismMintableERC20FactoryCaller: OptimismMintableERC20FactoryCaller{contract: contract}, OptimismMintableERC20FactoryTransactor: OptimismMintableERC20FactoryTransactor{contract: contract}, OptimismMintableERC20FactoryFilterer: OptimismMintableERC20FactoryFilterer{contract: contract}}, nil
}

// NewOptimismMintableERC20FactoryCaller creates a new read-only instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryCaller(address common.Address, caller bind.ContractCaller) (*OptimismMintableERC20FactoryCaller, error) {
	contract, err := bindOptimismMintableERC20Factory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryCaller{contract: contract}, nil
}

// NewOptimismMintableERC20FactoryTransactor creates a new write-only instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismMintableERC20FactoryTransactor, error) {
	contract, err := bindOptimismMintableERC20Factory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryTransactor{contract: contract}, nil
}

// NewOptimismMintableERC20FactoryFilterer creates a new log filterer instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismMintableERC20FactoryFilterer, error) {
	contract, err := bindOptimismMintableERC20Factory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryFilterer{contract: contract}, nil
}

// bindOptimismMintableERC20Factory binds a generic wrapper to an already deployed contract.
func bindOptimismMintableERC20Factory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismMintableERC20FactoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20Factory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.contract.Transact(opts, method, params...)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) BRIDGE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "BRIDGE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) BRIDGE() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.BRIDGE(&_OptimismMintableERC20Factory.CallOpts)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) BRIDGE() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.BRIDGE(&_OptimismMintableERC20Factory.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Bridge() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.Bridge(&_OptimismMintableERC20Factory.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) Bridge() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.Bridge(&_OptimismMintableERC20Factory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Version() (string, error) {
	return _OptimismMintableERC20Factory.Contract.Version(&_OptimismMintableERC20Factory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) Version() (string, error) {
	return _OptimismMintableERC20Factory.Contract.Version(&_OptimismMintableERC20Factory.CallOpts)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateOptimismMintableERC20(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createOptimismMintableERC20", _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateOptimismMintableERC20(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateOptimismMintableERC20(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateOptimismMintableERC20WithDecimals(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createOptimismMintableERC20WithDecimals", _remoteToken, _name, _symbol, _decimals)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateOptimismMintableERC20WithDecimals(_remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20WithDecimals(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol, _decimals)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateOptimismMintableERC20WithDecimals(_remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20WithDecimals(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol, _decimals)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateStandardL2Token(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createStandardL2Token", _remoteToken, _name, _symbol)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateStandardL2Token(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateStandardL2Token(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateStandardL2Token(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateStandardL2Token(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "initialize", _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.Initialize(&_OptimismMintableERC20Factory.TransactOpts, _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.Initialize(&_OptimismMintableERC20Factory.TransactOpts, _bridge)
}

// OptimismMintableERC20FactoryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryInitializedIterator struct {
	Event *OptimismMintableERC20FactoryInitialized // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryInitialized)
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
		it.Event = new(OptimismMintableERC20FactoryInitialized)
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
func (it *OptimismMintableERC20FactoryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryInitialized represents a Initialized event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterInitialized(opts *bind.FilterOpts) (*OptimismMintableERC20FactoryInitializedIterator, error) {

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryInitializedIterator{contract: _OptimismMintableERC20Factory.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryInitialized) (event.Subscription, error) {

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryInitialized)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseInitialized(log types.Log) (*OptimismMintableERC20FactoryInitialized, error) {
	event := new(OptimismMintableERC20FactoryInitialized)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator is returned from FilterOptimismMintableERC20Created and is used to iterate over the raw logs and unpacked data for OptimismMintableERC20Created events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator struct {
	Event *OptimismMintableERC20FactoryOptimismMintableERC20Created // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
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
		it.Event = new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
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
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryOptimismMintableERC20Created represents a OptimismMintableERC20Created event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryOptimismMintableERC20Created struct {
	LocalToken  common.Address
	RemoteToken common.Address
	Deployer    common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterOptimismMintableERC20Created is a free log retrieval operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterOptimismMintableERC20Created(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address) (*OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "OptimismMintableERC20Created", localTokenRule, remoteTokenRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator{contract: _OptimismMintableERC20Factory.contract, event: "OptimismMintableERC20Created", logs: logs, sub: sub}, nil
}

// WatchOptimismMintableERC20Created is a free log subscription operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchOptimismMintableERC20Created(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryOptimismMintableERC20Created, localToken []common.Address, remoteToken []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "OptimismMintableERC20Created", localTokenRule, remoteTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "OptimismMintableERC20Created", log); err != nil {
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

// ParseOptimismMintableERC20Created is a log parse operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseOptimismMintableERC20Created(log types.Log) (*OptimismMintableERC20FactoryOptimismMintableERC20Created, error) {
	event := new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "OptimismMintableERC20Created", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20FactoryStandardL2TokenCreatedIterator is returned from FilterStandardL2TokenCreated and is used to iterate over the raw logs and unpacked data for StandardL2TokenCreated events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryStandardL2TokenCreatedIterator struct {
	Event *OptimismMintableERC20FactoryStandardL2TokenCreated // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryStandardL2TokenCreated)
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
		it.Event = new(OptimismMintableERC20FactoryStandardL2TokenCreated)
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
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryStandardL2TokenCreated represents a StandardL2TokenCreated event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryStandardL2TokenCreated struct {
	RemoteToken common.Address
	LocalToken  common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterStandardL2TokenCreated is a free log retrieval operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterStandardL2TokenCreated(opts *bind.FilterOpts, remoteToken []common.Address, localToken []common.Address) (*OptimismMintableERC20FactoryStandardL2TokenCreatedIterator, error) {

	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "StandardL2TokenCreated", remoteTokenRule, localTokenRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryStandardL2TokenCreatedIterator{contract: _OptimismMintableERC20Factory.contract, event: "StandardL2TokenCreated", logs: logs, sub: sub}, nil
}

// WatchStandardL2TokenCreated is a free log subscription operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchStandardL2TokenCreated(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryStandardL2TokenCreated, remoteToken []common.Address, localToken []common.Address) (event.Subscription, error) {

	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "StandardL2TokenCreated", remoteTokenRule, localTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryStandardL2TokenCreated)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "StandardL2TokenCreated", log); err != nil {
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

// ParseStandardL2TokenCreated is a log parse operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseStandardL2TokenCreated(log types.Log) (*OptimismMintableERC20FactoryStandardL2TokenCreated, error) {
	event := new(OptimismMintableERC20FactoryStandardL2TokenCreated)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "StandardL2TokenCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
