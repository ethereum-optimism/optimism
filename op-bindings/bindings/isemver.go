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

// ISemverMetaData contains all meta data concerning the ISemver contract.
var ISemverMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"}]",
}

// ISemverABI is the input ABI used to generate the binding from.
// Deprecated: Use ISemverMetaData.ABI instead.
var ISemverABI = ISemverMetaData.ABI

// ISemver is an auto generated Go binding around an Ethereum contract.
type ISemver struct {
	ISemverCaller     // Read-only binding to the contract
	ISemverTransactor // Write-only binding to the contract
	ISemverFilterer   // Log filterer for contract events
}

// ISemverCaller is an auto generated read-only Go binding around an Ethereum contract.
type ISemverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISemverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ISemverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISemverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ISemverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ISemverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ISemverSession struct {
	Contract     *ISemver          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ISemverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ISemverCallerSession struct {
	Contract *ISemverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ISemverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ISemverTransactorSession struct {
	Contract     *ISemverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ISemverRaw is an auto generated low-level Go binding around an Ethereum contract.
type ISemverRaw struct {
	Contract *ISemver // Generic contract binding to access the raw methods on
}

// ISemverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ISemverCallerRaw struct {
	Contract *ISemverCaller // Generic read-only contract binding to access the raw methods on
}

// ISemverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ISemverTransactorRaw struct {
	Contract *ISemverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewISemver creates a new instance of ISemver, bound to a specific deployed contract.
func NewISemver(address common.Address, backend bind.ContractBackend) (*ISemver, error) {
	contract, err := bindISemver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ISemver{ISemverCaller: ISemverCaller{contract: contract}, ISemverTransactor: ISemverTransactor{contract: contract}, ISemverFilterer: ISemverFilterer{contract: contract}}, nil
}

// NewISemverCaller creates a new read-only instance of ISemver, bound to a specific deployed contract.
func NewISemverCaller(address common.Address, caller bind.ContractCaller) (*ISemverCaller, error) {
	contract, err := bindISemver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ISemverCaller{contract: contract}, nil
}

// NewISemverTransactor creates a new write-only instance of ISemver, bound to a specific deployed contract.
func NewISemverTransactor(address common.Address, transactor bind.ContractTransactor) (*ISemverTransactor, error) {
	contract, err := bindISemver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ISemverTransactor{contract: contract}, nil
}

// NewISemverFilterer creates a new log filterer instance of ISemver, bound to a specific deployed contract.
func NewISemverFilterer(address common.Address, filterer bind.ContractFilterer) (*ISemverFilterer, error) {
	contract, err := bindISemver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ISemverFilterer{contract: contract}, nil
}

// bindISemver binds a generic wrapper to an already deployed contract.
func bindISemver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ISemverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISemver *ISemverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISemver.Contract.ISemverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISemver *ISemverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISemver.Contract.ISemverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISemver *ISemverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISemver.Contract.ISemverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ISemver *ISemverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ISemver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ISemver *ISemverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ISemver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ISemver *ISemverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ISemver.Contract.contract.Transact(opts, method, params...)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ISemver *ISemverCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ISemver.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ISemver *ISemverSession) Version() (string, error) {
	return _ISemver.Contract.Version(&_ISemver.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ISemver *ISemverCallerSession) Version() (string, error) {
	return _ISemver.Contract.Version(&_ISemver.CallOpts)
}
