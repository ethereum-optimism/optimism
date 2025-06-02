// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package sysgo

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

// DelegateCallProxyMetaData contains all meta data concerning the DelegateCallProxy contract.
var DelegateCallProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"NotOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxy\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"changeAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"executeDelegateCall\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_proxyAdmin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051610ad4380380610ad4833981810160405281019061003291906100cf565b8073ffffffffffffffffffffffffffffffffffffffff1660808173ffffffffffffffffffffffffffffffffffffffff1681525050506100fc565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b600061009c82610071565b9050919050565b6100ac81610091565b81146100b757600080fd5b50565b6000815190506100c9816100a3565b92915050565b6000602082840312156100e5576100e461006c565b5b60006100f3848285016100ba565b91505092915050565b6080516109b06101246000396000818160d9015281816102a3015261046d01526109b06000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c80631acfd02a146100515780636d4354211461006d5780638da5cb5b14610089578063b68df16d146100a7575b600080fd5b61006b60048036038101906100669190610588565b6100d7565b005b61008760048036038101906100829190610588565b6102a1565b005b61009161046b565b60405161009e91906105d7565b60405180910390f35b6100c160048036038101906100bc9190610738565b61048f565b6040516100ce919061081c565b60405180910390f35b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461015c576040517f30cd747100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008160405160240161016f91906105d7565b6040516020818303038152906040527f8f283970000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050905060008373ffffffffffffffffffffffffffffffffffffffff1682604051610215919061087a565b6000604051808303816000865af19150503d8060008114610252576040519150601f19603f3d011682016040523d82523d6000602084013e610257565b606091505b505090508061029b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610292906108ee565b60405180910390fd5b50505050565b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610326576040517f30cd747100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008160405160240161033991906105d7565b6040516020818303038152906040527ff2fde38b000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050905060008373ffffffffffffffffffffffffffffffffffffffff16826040516103df919061087a565b6000604051808303816000865af19150503d806000811461041c576040519150601f19603f3d011682016040523d82523d6000602084013e610421565b606091505b5050905080610465576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161045c9061095a565b60405180910390fd5b50505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b60606000808473ffffffffffffffffffffffffffffffffffffffff16846040516104b9919061087a565b600060405180830381855af49150503d80600081146104f4576040519150601f19603f3d011682016040523d82523d6000602084013e6104f9565b606091505b50915091508161050b57805160208201fd5b809250505092915050565b6000604051905090565b600080fd5b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006105558261052a565b9050919050565b6105658161054a565b811461057057600080fd5b50565b6000813590506105828161055c565b92915050565b6000806040838503121561059f5761059e610520565b5b60006105ad85828601610573565b92505060206105be85828601610573565b9150509250929050565b6105d18161054a565b82525050565b60006020820190506105ec60008301846105c8565b92915050565b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b610645826105fc565b810181811067ffffffffffffffff821117156106645761066361060d565b5b80604052505050565b6000610677610516565b9050610683828261063c565b919050565b600067ffffffffffffffff8211156106a3576106a261060d565b5b6106ac826105fc565b9050602081019050919050565b82818337600083830152505050565b60006106db6106d684610688565b61066d565b9050828152602081018484840111156106f7576106f66105f7565b5b6107028482856106b9565b509392505050565b600082601f83011261071f5761071e6105f2565b5b813561072f8482602086016106c8565b91505092915050565b6000806040838503121561074f5761074e610520565b5b600061075d85828601610573565b925050602083013567ffffffffffffffff81111561077e5761077d610525565b5b61078a8582860161070a565b9150509250929050565b600081519050919050565b600082825260208201905092915050565b60005b838110156107ce5780820151818401526020810190506107b3565b838111156107dd576000848401525b50505050565b60006107ee82610794565b6107f8818561079f565b93506108088185602086016107b0565b610811816105fc565b840191505092915050565b6000602082019050818103600083015261083681846107e3565b905092915050565b600081905092915050565b600061085482610794565b61085e818561083e565b935061086e8185602086016107b0565b80840191505092915050565b60006108868284610849565b915081905092915050565b600082825260208201905092915050565b7f4368616e676541646d696e3a206661696c656400000000000000000000000000600082015250565b60006108d8601383610891565b91506108e3826108a2565b602082019050919050565b60006020820190508181036000830152610907816108cb565b9050919050565b7f5472616e736665724f776e6572736869703a206661696c656400000000000000600082015250565b6000610944601983610891565b915061094f8261090e565b602082019050919050565b6000602082019050818103600083015261097381610937565b905091905056fea264697066735822122054e9d41a3515a4abdb82b48fd2b0d99d61e5a1444c620bedbe7c4976292015fe64736f6c634300080f0033",
}

// DelegateCallProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegateCallProxyMetaData.ABI instead.
var DelegateCallProxyABI = DelegateCallProxyMetaData.ABI

// DelegateCallProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DelegateCallProxyMetaData.Bin instead.
var DelegateCallProxyBin = DelegateCallProxyMetaData.Bin

// DeployDelegateCallProxy deploys a new Ethereum contract, binding an instance of DelegateCallProxy to it.
func DeployDelegateCallProxy(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address) (common.Address, *types.Transaction, *DelegateCallProxy, error) {
	parsed, err := DelegateCallProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DelegateCallProxyBin), backend, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DelegateCallProxy{DelegateCallProxyCaller: DelegateCallProxyCaller{contract: contract}, DelegateCallProxyTransactor: DelegateCallProxyTransactor{contract: contract}, DelegateCallProxyFilterer: DelegateCallProxyFilterer{contract: contract}}, nil
}

// DelegateCallProxy is an auto generated Go binding around an Ethereum contract.
type DelegateCallProxy struct {
	DelegateCallProxyCaller     // Read-only binding to the contract
	DelegateCallProxyTransactor // Write-only binding to the contract
	DelegateCallProxyFilterer   // Log filterer for contract events
}

// DelegateCallProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegateCallProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegateCallProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegateCallProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateCallProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegateCallProxySession struct {
	Contract     *DelegateCallProxy // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// DelegateCallProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegateCallProxyCallerSession struct {
	Contract *DelegateCallProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// DelegateCallProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegateCallProxyTransactorSession struct {
	Contract     *DelegateCallProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// DelegateCallProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegateCallProxyRaw struct {
	Contract *DelegateCallProxy // Generic contract binding to access the raw methods on
}

// DelegateCallProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegateCallProxyCallerRaw struct {
	Contract *DelegateCallProxyCaller // Generic read-only contract binding to access the raw methods on
}

// DelegateCallProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegateCallProxyTransactorRaw struct {
	Contract *DelegateCallProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegateCallProxy creates a new instance of DelegateCallProxy, bound to a specific deployed contract.
func NewDelegateCallProxy(address common.Address, backend bind.ContractBackend) (*DelegateCallProxy, error) {
	contract, err := bindDelegateCallProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelegateCallProxy{DelegateCallProxyCaller: DelegateCallProxyCaller{contract: contract}, DelegateCallProxyTransactor: DelegateCallProxyTransactor{contract: contract}, DelegateCallProxyFilterer: DelegateCallProxyFilterer{contract: contract}}, nil
}

// NewDelegateCallProxyCaller creates a new read-only instance of DelegateCallProxy, bound to a specific deployed contract.
func NewDelegateCallProxyCaller(address common.Address, caller bind.ContractCaller) (*DelegateCallProxyCaller, error) {
	contract, err := bindDelegateCallProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallProxyCaller{contract: contract}, nil
}

// NewDelegateCallProxyTransactor creates a new write-only instance of DelegateCallProxy, bound to a specific deployed contract.
func NewDelegateCallProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegateCallProxyTransactor, error) {
	contract, err := bindDelegateCallProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateCallProxyTransactor{contract: contract}, nil
}

// NewDelegateCallProxyFilterer creates a new log filterer instance of DelegateCallProxy, bound to a specific deployed contract.
func NewDelegateCallProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegateCallProxyFilterer, error) {
	contract, err := bindDelegateCallProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegateCallProxyFilterer{contract: contract}, nil
}

// bindDelegateCallProxy binds a generic wrapper to an already deployed contract.
func bindDelegateCallProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DelegateCallProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCallProxy *DelegateCallProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCallProxy.Contract.DelegateCallProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCallProxy *DelegateCallProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.DelegateCallProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCallProxy *DelegateCallProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.DelegateCallProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateCallProxy *DelegateCallProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateCallProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateCallProxy *DelegateCallProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateCallProxy *DelegateCallProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateCallProxy *DelegateCallProxyCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelegateCallProxy.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateCallProxy *DelegateCallProxySession) Owner() (common.Address, error) {
	return _DelegateCallProxy.Contract.Owner(&_DelegateCallProxy.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateCallProxy *DelegateCallProxyCallerSession) Owner() (common.Address, error) {
	return _DelegateCallProxy.Contract.Owner(&_DelegateCallProxy.CallOpts)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x1acfd02a.
//
// Solidity: function changeAdmin(address _proxy, address newAdmin) returns()
func (_DelegateCallProxy *DelegateCallProxyTransactor) ChangeAdmin(opts *bind.TransactOpts, _proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.contract.Transact(opts, "changeAdmin", _proxy, newAdmin)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x1acfd02a.
//
// Solidity: function changeAdmin(address _proxy, address newAdmin) returns()
func (_DelegateCallProxy *DelegateCallProxySession) ChangeAdmin(_proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.ChangeAdmin(&_DelegateCallProxy.TransactOpts, _proxy, newAdmin)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x1acfd02a.
//
// Solidity: function changeAdmin(address _proxy, address newAdmin) returns()
func (_DelegateCallProxy *DelegateCallProxyTransactorSession) ChangeAdmin(_proxy common.Address, newAdmin common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.ChangeAdmin(&_DelegateCallProxy.TransactOpts, _proxy, newAdmin)
}

// ExecuteDelegateCall is a paid mutator transaction binding the contract method 0xb68df16d.
//
// Solidity: function executeDelegateCall(address _target, bytes _data) returns(bytes)
func (_DelegateCallProxy *DelegateCallProxyTransactor) ExecuteDelegateCall(opts *bind.TransactOpts, _target common.Address, _data []byte) (*types.Transaction, error) {
	return _DelegateCallProxy.contract.Transact(opts, "executeDelegateCall", _target, _data)
}

// ExecuteDelegateCall is a paid mutator transaction binding the contract method 0xb68df16d.
//
// Solidity: function executeDelegateCall(address _target, bytes _data) returns(bytes)
func (_DelegateCallProxy *DelegateCallProxySession) ExecuteDelegateCall(_target common.Address, _data []byte) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.ExecuteDelegateCall(&_DelegateCallProxy.TransactOpts, _target, _data)
}

// ExecuteDelegateCall is a paid mutator transaction binding the contract method 0xb68df16d.
//
// Solidity: function executeDelegateCall(address _target, bytes _data) returns(bytes)
func (_DelegateCallProxy *DelegateCallProxyTransactorSession) ExecuteDelegateCall(_target common.Address, _data []byte) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.ExecuteDelegateCall(&_DelegateCallProxy.TransactOpts, _target, _data)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0x6d435421.
//
// Solidity: function transferOwnership(address _proxyAdmin, address newOwner) returns()
func (_DelegateCallProxy *DelegateCallProxyTransactor) TransferOwnership(opts *bind.TransactOpts, _proxyAdmin common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.contract.Transact(opts, "transferOwnership", _proxyAdmin, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0x6d435421.
//
// Solidity: function transferOwnership(address _proxyAdmin, address newOwner) returns()
func (_DelegateCallProxy *DelegateCallProxySession) TransferOwnership(_proxyAdmin common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.TransferOwnership(&_DelegateCallProxy.TransactOpts, _proxyAdmin, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0x6d435421.
//
// Solidity: function transferOwnership(address _proxyAdmin, address newOwner) returns()
func (_DelegateCallProxy *DelegateCallProxyTransactorSession) TransferOwnership(_proxyAdmin common.Address, newOwner common.Address) (*types.Transaction, error) {
	return _DelegateCallProxy.Contract.TransferOwnership(&_DelegateCallProxy.TransactOpts, _proxyAdmin, newOwner)
}
