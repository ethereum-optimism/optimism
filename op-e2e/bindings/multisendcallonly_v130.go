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

// MultiSendCallOnlyV130MetaData contains all meta data concerning the MultiSendCallOnlyV130 contract.
var MultiSendCallOnlyV130MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"transactions\",\"type\":\"bytes\"}],\"name\":\"multiSend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061019a806100206000396000f3fe60806040526004361061001e5760003560e01c80638d80ff0a14610023575b600080fd5b6100dc6004803603602081101561003957600080fd5b810190808035906020019064010000000081111561005657600080fd5b82018360208201111561006857600080fd5b8035906020019184600183028401116401000000008311171561008a57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506100de565b005b805160205b8181101561015f578083015160f81c6001820184015160601c60158301850151603584018601516055850187016000856000811461012857600181146101385761013d565b6000808585888a5af1915061013d565b600080fd5b50600081141561014c57600080fd5b82605501870196505050505050506100e3565b50505056fea264697066735822122035246402746c96964495cae5b36461fd44dfb89f8e6cf6f6b8d60c0aa89f414864736f6c63430007060033",
}

// MultiSendCallOnlyV130ABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiSendCallOnlyV130MetaData.ABI instead.
var MultiSendCallOnlyV130ABI = MultiSendCallOnlyV130MetaData.ABI

// MultiSendCallOnlyV130Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MultiSendCallOnlyV130MetaData.Bin instead.
var MultiSendCallOnlyV130Bin = MultiSendCallOnlyV130MetaData.Bin

// DeployMultiSendCallOnlyV130 deploys a new Ethereum contract, binding an instance of MultiSendCallOnlyV130 to it.
func DeployMultiSendCallOnlyV130(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MultiSendCallOnlyV130, error) {
	parsed, err := MultiSendCallOnlyV130MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MultiSendCallOnlyV130Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiSendCallOnlyV130{MultiSendCallOnlyV130Caller: MultiSendCallOnlyV130Caller{contract: contract}, MultiSendCallOnlyV130Transactor: MultiSendCallOnlyV130Transactor{contract: contract}, MultiSendCallOnlyV130Filterer: MultiSendCallOnlyV130Filterer{contract: contract}}, nil
}

// MultiSendCallOnlyV130 is an auto generated Go binding around an Ethereum contract.
type MultiSendCallOnlyV130 struct {
	MultiSendCallOnlyV130Caller     // Read-only binding to the contract
	MultiSendCallOnlyV130Transactor // Write-only binding to the contract
	MultiSendCallOnlyV130Filterer   // Log filterer for contract events
}

// MultiSendCallOnlyV130Caller is an auto generated read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyV130Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyV130Transactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyV130Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyV130Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiSendCallOnlyV130Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyV130Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiSendCallOnlyV130Session struct {
	Contract     *MultiSendCallOnlyV130 // Generic contract binding to set the session for
	CallOpts     bind.CallOpts          // Call options to use throughout this session
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyV130CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiSendCallOnlyV130CallerSession struct {
	Contract *MultiSendCallOnlyV130Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                // Call options to use throughout this session
}

// MultiSendCallOnlyV130TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiSendCallOnlyV130TransactorSession struct {
	Contract     *MultiSendCallOnlyV130Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyV130Raw is an auto generated low-level Go binding around an Ethereum contract.
type MultiSendCallOnlyV130Raw struct {
	Contract *MultiSendCallOnlyV130 // Generic contract binding to access the raw methods on
}

// MultiSendCallOnlyV130CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyV130CallerRaw struct {
	Contract *MultiSendCallOnlyV130Caller // Generic read-only contract binding to access the raw methods on
}

// MultiSendCallOnlyV130TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyV130TransactorRaw struct {
	Contract *MultiSendCallOnlyV130Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiSendCallOnlyV130 creates a new instance of MultiSendCallOnlyV130, bound to a specific deployed contract.
func NewMultiSendCallOnlyV130(address common.Address, backend bind.ContractBackend) (*MultiSendCallOnlyV130, error) {
	contract, err := bindMultiSendCallOnlyV130(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyV130{MultiSendCallOnlyV130Caller: MultiSendCallOnlyV130Caller{contract: contract}, MultiSendCallOnlyV130Transactor: MultiSendCallOnlyV130Transactor{contract: contract}, MultiSendCallOnlyV130Filterer: MultiSendCallOnlyV130Filterer{contract: contract}}, nil
}

// NewMultiSendCallOnlyV130Caller creates a new read-only instance of MultiSendCallOnlyV130, bound to a specific deployed contract.
func NewMultiSendCallOnlyV130Caller(address common.Address, caller bind.ContractCaller) (*MultiSendCallOnlyV130Caller, error) {
	contract, err := bindMultiSendCallOnlyV130(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyV130Caller{contract: contract}, nil
}

// NewMultiSendCallOnlyV130Transactor creates a new write-only instance of MultiSendCallOnlyV130, bound to a specific deployed contract.
func NewMultiSendCallOnlyV130Transactor(address common.Address, transactor bind.ContractTransactor) (*MultiSendCallOnlyV130Transactor, error) {
	contract, err := bindMultiSendCallOnlyV130(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyV130Transactor{contract: contract}, nil
}

// NewMultiSendCallOnlyV130Filterer creates a new log filterer instance of MultiSendCallOnlyV130, bound to a specific deployed contract.
func NewMultiSendCallOnlyV130Filterer(address common.Address, filterer bind.ContractFilterer) (*MultiSendCallOnlyV130Filterer, error) {
	contract, err := bindMultiSendCallOnlyV130(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyV130Filterer{contract: contract}, nil
}

// bindMultiSendCallOnlyV130 binds a generic wrapper to an already deployed contract.
func bindMultiSendCallOnlyV130(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiSendCallOnlyV130ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnlyV130.Contract.MultiSendCallOnlyV130Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.MultiSendCallOnlyV130Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.MultiSendCallOnlyV130Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnlyV130.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.contract.Transact(opts, method, params...)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130Transactor) MultiSend(opts *bind.TransactOpts, transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.contract.Transact(opts, "multiSend", transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130Session) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.MultiSend(&_MultiSendCallOnlyV130.TransactOpts, transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnlyV130 *MultiSendCallOnlyV130TransactorSession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnlyV130.Contract.MultiSend(&_MultiSendCallOnlyV130.TransactOpts, transactions)
}
