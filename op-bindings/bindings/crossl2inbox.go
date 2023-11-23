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

// InboxEntry is an auto generated low-level Go binding around an user-defined struct.
type InboxEntry struct {
	Chain  [32]byte
	Output [32]byte
}

// CrossL2InboxMetaData contains all meta data concerning the CrossL2Inbox contract.
var CrossL2InboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_superchainPostie\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"chain\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"output\",\"type\":\"bytes32\"}],\"internalType\":\"structInboxEntry[]\",\"name\":\"mail\",\"type\":\"tuple[]\"}],\"name\":\"deliverMail\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainPostie\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b506040516104cb3803806104cb83398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b60805161043a6100916000396000818160bd015261014f015261043a6000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c806354fd4d5014610051578063c00157da146100a3578063db10b9a9146100e7578063f2c5dcb814610122575b600080fd5b61008d6040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b60405161009a9190610295565b60405180910390f35b60405173ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016815260200161009a565b6101126100f5366004610308565b600060208181529281526040808220909352908152205460ff1681565b604051901515815260200161009a565b61013561013036600461032a565b610137565b005b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614610200576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f43726f73734c32496e626f783a206f6e6c7920706f737469652063616e20646560448201527f6c69766572206d61696c00000000000000000000000000000000000000000000606482015260840160405180910390fd5b60005b818110156102905760016000808585858181106102225761022261039f565b905060400201600001358152602001908152602001600020600085858581811061024e5761024e61039f565b90506040020160200135815260200190815260200160002060006101000a81548160ff0219169083151502179055508080610288906103ce565b915050610203565b505050565b600060208083528351808285015260005b818110156102c2578581018301518582016040015282016102a6565b818111156102d4576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b6000806040838503121561031b57600080fd5b50508035926020909101359150565b6000806020838503121561033d57600080fd5b823567ffffffffffffffff8082111561035557600080fd5b818501915085601f83011261036957600080fd5b81358181111561037857600080fd5b8660208260061b850101111561038d57600080fd5b60209290920196919550909350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610426577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b506001019056fea164736f6c634300080f000a",
}

// CrossL2InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2InboxMetaData.ABI instead.
var CrossL2InboxABI = CrossL2InboxMetaData.ABI

// CrossL2InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2InboxMetaData.Bin instead.
var CrossL2InboxBin = CrossL2InboxMetaData.Bin

// DeployCrossL2Inbox deploys a new Ethereum contract, binding an instance of CrossL2Inbox to it.
func DeployCrossL2Inbox(auth *bind.TransactOpts, backend bind.ContractBackend, _superchainPostie common.Address) (common.Address, *types.Transaction, *CrossL2Inbox, error) {
	parsed, err := CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2InboxBin), backend, _superchainPostie)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// CrossL2Inbox is an auto generated Go binding around an Ethereum contract.
type CrossL2Inbox struct {
	CrossL2InboxCaller     // Read-only binding to the contract
	CrossL2InboxTransactor // Write-only binding to the contract
	CrossL2InboxFilterer   // Log filterer for contract events
}

// CrossL2InboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrossL2InboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrossL2InboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrossL2InboxSession struct {
	Contract     *CrossL2Inbox     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CrossL2InboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrossL2InboxCallerSession struct {
	Contract *CrossL2InboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// CrossL2InboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrossL2InboxTransactorSession struct {
	Contract     *CrossL2InboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CrossL2InboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrossL2InboxRaw struct {
	Contract *CrossL2Inbox // Generic contract binding to access the raw methods on
}

// CrossL2InboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrossL2InboxCallerRaw struct {
	Contract *CrossL2InboxCaller // Generic read-only contract binding to access the raw methods on
}

// CrossL2InboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactorRaw struct {
	Contract *CrossL2InboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrossL2Inbox creates a new instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2Inbox(address common.Address, backend bind.ContractBackend) (*CrossL2Inbox, error) {
	contract, err := bindCrossL2Inbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// NewCrossL2InboxCaller creates a new read-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxCaller(address common.Address, caller bind.ContractCaller) (*CrossL2InboxCaller, error) {
	contract, err := bindCrossL2Inbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxCaller{contract: contract}, nil
}

// NewCrossL2InboxTransactor creates a new write-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxTransactor(address common.Address, transactor bind.ContractTransactor) (*CrossL2InboxTransactor, error) {
	contract, err := bindCrossL2Inbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxTransactor{contract: contract}, nil
}

// NewCrossL2InboxFilterer creates a new log filterer instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxFilterer(address common.Address, filterer bind.ContractFilterer) (*CrossL2InboxFilterer, error) {
	contract, err := bindCrossL2Inbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxFilterer{contract: contract}, nil
}

// bindCrossL2Inbox binds a generic wrapper to an already deployed contract.
func bindCrossL2Inbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrossL2InboxABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.CrossL2InboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transact(opts, method, params...)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCaller) Roots(opts *bind.CallOpts, arg0 [32]byte, arg1 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "roots", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) SuperchainPostie(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "superchainPostie")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) SuperchainPostie() (common.Address, error) {
	return _CrossL2Inbox.Contract.SuperchainPostie(&_CrossL2Inbox.CallOpts)
}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) SuperchainPostie() (common.Address, error) {
	return _CrossL2Inbox.Contract.SuperchainPostie(&_CrossL2Inbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) DeliverMail(opts *bind.TransactOpts, mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "deliverMail", mail)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxSession) DeliverMail(mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMail(&_CrossL2Inbox.TransactOpts, mail)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) DeliverMail(mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMail(&_CrossL2Inbox.TransactOpts, mail)
}
