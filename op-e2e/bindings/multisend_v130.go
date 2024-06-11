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

// MultiSendV130MetaData contains all meta data concerning the MultiSendV130 contract.
var MultiSendV130MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"transactions\",\"type\":\"bytes\"}],\"name\":\"multiSend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b503073ffffffffffffffffffffffffffffffffffffffff1660808173ffffffffffffffffffffffffffffffffffffffff1660601b8152505060805160601c6102756100646000398060e052506102756000f3fe60806040526004361061001e5760003560e01c80638d80ff0a14610023575b600080fd5b6100dc6004803603602081101561003957600080fd5b810190808035906020019064010000000081111561005657600080fd5b82018360208201111561006857600080fd5b8035906020019184600183028401116401000000008311171561008a57600080fd5b91908080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f8201169050808301925050505050505091929192905050506100de565b005b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff163073ffffffffffffffffffffffffffffffffffffffff161415610183576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260308152602001806102106030913960400191505060405180910390fd5b805160205b8181101561020a578083015160f81c6001820184015160601c6015830185015160358401860151605585018701600085600081146101cd57600181146101dd576101e8565b6000808585888a5af191506101e8565b6000808585895af491505b5060008114156101f757600080fd5b8260550187019650505050505050610188565b50505056fe4d756c746953656e642073686f756c64206f6e6c792062652063616c6c6564207669612064656c656761746563616c6ca26469706673582212205c784303626eec02b71940b551976170b500a8a36cc5adcbeb2c19751a76d05464736f6c63430007060033",
}

// MultiSendV130ABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiSendV130MetaData.ABI instead.
var MultiSendV130ABI = MultiSendV130MetaData.ABI

// MultiSendV130Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MultiSendV130MetaData.Bin instead.
var MultiSendV130Bin = MultiSendV130MetaData.Bin

// DeployMultiSendV130 deploys a new Ethereum contract, binding an instance of MultiSendV130 to it.
func DeployMultiSendV130(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MultiSendV130, error) {
	parsed, err := MultiSendV130MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MultiSendV130Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiSendV130{MultiSendV130Caller: MultiSendV130Caller{contract: contract}, MultiSendV130Transactor: MultiSendV130Transactor{contract: contract}, MultiSendV130Filterer: MultiSendV130Filterer{contract: contract}}, nil
}

// MultiSendV130 is an auto generated Go binding around an Ethereum contract.
type MultiSendV130 struct {
	MultiSendV130Caller     // Read-only binding to the contract
	MultiSendV130Transactor // Write-only binding to the contract
	MultiSendV130Filterer   // Log filterer for contract events
}

// MultiSendV130Caller is an auto generated read-only Go binding around an Ethereum contract.
type MultiSendV130Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendV130Transactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiSendV130Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendV130Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiSendV130Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendV130Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiSendV130Session struct {
	Contract     *MultiSendV130    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultiSendV130CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiSendV130CallerSession struct {
	Contract *MultiSendV130Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// MultiSendV130TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiSendV130TransactorSession struct {
	Contract     *MultiSendV130Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// MultiSendV130Raw is an auto generated low-level Go binding around an Ethereum contract.
type MultiSendV130Raw struct {
	Contract *MultiSendV130 // Generic contract binding to access the raw methods on
}

// MultiSendV130CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiSendV130CallerRaw struct {
	Contract *MultiSendV130Caller // Generic read-only contract binding to access the raw methods on
}

// MultiSendV130TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiSendV130TransactorRaw struct {
	Contract *MultiSendV130Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiSendV130 creates a new instance of MultiSendV130, bound to a specific deployed contract.
func NewMultiSendV130(address common.Address, backend bind.ContractBackend) (*MultiSendV130, error) {
	contract, err := bindMultiSendV130(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiSendV130{MultiSendV130Caller: MultiSendV130Caller{contract: contract}, MultiSendV130Transactor: MultiSendV130Transactor{contract: contract}, MultiSendV130Filterer: MultiSendV130Filterer{contract: contract}}, nil
}

// NewMultiSendV130Caller creates a new read-only instance of MultiSendV130, bound to a specific deployed contract.
func NewMultiSendV130Caller(address common.Address, caller bind.ContractCaller) (*MultiSendV130Caller, error) {
	contract, err := bindMultiSendV130(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendV130Caller{contract: contract}, nil
}

// NewMultiSendV130Transactor creates a new write-only instance of MultiSendV130, bound to a specific deployed contract.
func NewMultiSendV130Transactor(address common.Address, transactor bind.ContractTransactor) (*MultiSendV130Transactor, error) {
	contract, err := bindMultiSendV130(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendV130Transactor{contract: contract}, nil
}

// NewMultiSendV130Filterer creates a new log filterer instance of MultiSendV130, bound to a specific deployed contract.
func NewMultiSendV130Filterer(address common.Address, filterer bind.ContractFilterer) (*MultiSendV130Filterer, error) {
	contract, err := bindMultiSendV130(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiSendV130Filterer{contract: contract}, nil
}

// bindMultiSendV130 binds a generic wrapper to an already deployed contract.
func bindMultiSendV130(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiSendV130ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendV130 *MultiSendV130Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendV130.Contract.MultiSendV130Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendV130 *MultiSendV130Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendV130.Contract.MultiSendV130Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendV130 *MultiSendV130Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendV130.Contract.MultiSendV130Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendV130 *MultiSendV130CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendV130.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendV130 *MultiSendV130TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendV130.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendV130 *MultiSendV130TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendV130.Contract.contract.Transact(opts, method, params...)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendV130 *MultiSendV130Transactor) MultiSend(opts *bind.TransactOpts, transactions []byte) (*types.Transaction, error) {
	return _MultiSendV130.contract.Transact(opts, "multiSend", transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendV130 *MultiSendV130Session) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendV130.Contract.MultiSend(&_MultiSendV130.TransactOpts, transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendV130 *MultiSendV130TransactorSession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendV130.Contract.MultiSend(&_MultiSendV130.TransactOpts, transactions)
}
