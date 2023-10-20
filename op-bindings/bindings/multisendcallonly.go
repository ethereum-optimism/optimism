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

// MultiSendCallOnlyMetaData contains all meta data concerning the MultiSendCallOnly contract.
var MultiSendCallOnlyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"transactions\",\"type\":\"bytes\"}],\"name\":\"multiSend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061019a806100206000396000f3fe60806040526004361061001e5760003560e01c80638d80ff0a14610023575b600080fd5b6100dc6004803603602081101561003957600080fd5b810190808035906020019064010000000081111561005657600080fd5b82018360208201111561006857600080fd5b8035906020019184600183028401116401000000008311171561008a57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506100de565b005b805160205b8181101561015f578083015160f81c6001820184015160601c60158301850151603584018601516055850187016000856000811461012857600181146101385761013d565b6000808585888a5af1915061013d565b600080fd5b50600081141561014c57600080fd5b82605501870196505050505050506100e3565b50505056fea264697066735822122035246402746c96964495cae5b36461fd44dfb89f8e6cf6f6b8d60c0aa89f414864736f6c63430007060033",
}

// MultiSendCallOnlyABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiSendCallOnlyMetaData.ABI instead.
var MultiSendCallOnlyABI = MultiSendCallOnlyMetaData.ABI

// MultiSendCallOnlyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MultiSendCallOnlyMetaData.Bin instead.
var MultiSendCallOnlyBin = MultiSendCallOnlyMetaData.Bin

// DeployMultiSendCallOnly deploys a new Ethereum contract, binding an instance of MultiSendCallOnly to it.
func DeployMultiSendCallOnly(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MultiSendCallOnly, error) {
	parsed, err := MultiSendCallOnlyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MultiSendCallOnlyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiSendCallOnly{MultiSendCallOnlyCaller: MultiSendCallOnlyCaller{contract: contract}, MultiSendCallOnlyTransactor: MultiSendCallOnlyTransactor{contract: contract}, MultiSendCallOnlyFilterer: MultiSendCallOnlyFilterer{contract: contract}}, nil
}

// MultiSendCallOnly is an auto generated Go binding around an Ethereum contract.
type MultiSendCallOnly struct {
	MultiSendCallOnlyCaller     // Read-only binding to the contract
	MultiSendCallOnlyTransactor // Write-only binding to the contract
	MultiSendCallOnlyFilterer   // Log filterer for contract events
}

// MultiSendCallOnlyCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiSendCallOnlyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiSendCallOnlySession struct {
	Contract     *MultiSendCallOnly // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiSendCallOnlyCallerSession struct {
	Contract *MultiSendCallOnlyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// MultiSendCallOnlyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiSendCallOnlyTransactorSession struct {
	Contract     *MultiSendCallOnlyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultiSendCallOnlyRaw struct {
	Contract *MultiSendCallOnly // Generic contract binding to access the raw methods on
}

// MultiSendCallOnlyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyCallerRaw struct {
	Contract *MultiSendCallOnlyCaller // Generic read-only contract binding to access the raw methods on
}

// MultiSendCallOnlyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyTransactorRaw struct {
	Contract *MultiSendCallOnlyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiSendCallOnly creates a new instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnly(address common.Address, backend bind.ContractBackend) (*MultiSendCallOnly, error) {
	contract, err := bindMultiSendCallOnly(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnly{MultiSendCallOnlyCaller: MultiSendCallOnlyCaller{contract: contract}, MultiSendCallOnlyTransactor: MultiSendCallOnlyTransactor{contract: contract}, MultiSendCallOnlyFilterer: MultiSendCallOnlyFilterer{contract: contract}}, nil
}

// NewMultiSendCallOnlyCaller creates a new read-only instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyCaller(address common.Address, caller bind.ContractCaller) (*MultiSendCallOnlyCaller, error) {
	contract, err := bindMultiSendCallOnly(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyCaller{contract: contract}, nil
}

// NewMultiSendCallOnlyTransactor creates a new write-only instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyTransactor(address common.Address, transactor bind.ContractTransactor) (*MultiSendCallOnlyTransactor, error) {
	contract, err := bindMultiSendCallOnly(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyTransactor{contract: contract}, nil
}

// NewMultiSendCallOnlyFilterer creates a new log filterer instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyFilterer(address common.Address, filterer bind.ContractFilterer) (*MultiSendCallOnlyFilterer, error) {
	contract, err := bindMultiSendCallOnly(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyFilterer{contract: contract}, nil
}

// bindMultiSendCallOnly binds a generic wrapper to an already deployed contract.
func bindMultiSendCallOnly(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiSendCallOnlyABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnly *MultiSendCallOnlyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnly.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.contract.Transact(opts, method, params...)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlyTransactor) MultiSend(opts *bind.TransactOpts, transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.contract.Transact(opts, "multiSend", transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlySession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSend(&_MultiSendCallOnly.TransactOpts, transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorSession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSend(&_MultiSendCallOnly.TransactOpts, transactions)
}
