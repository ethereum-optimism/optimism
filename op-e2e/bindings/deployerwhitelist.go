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

// DeployerWhitelistMetaData contains all meta data concerning the DeployerWhitelist contract.
var DeployerWhitelistMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"enableArbitraryContractDeployment\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isDeployerAllowed\",\"inputs\":[{\"name\":\"_deployer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setOwner\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setWhitelistedDeployer\",\"inputs\":[{\"name\":\"_deployer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_isWhitelisted\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"whitelist\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"OwnerChanged\",\"inputs\":[{\"name\":\"oldOwner\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WhitelistDisabled\",\"inputs\":[{\"name\":\"oldOwner\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WhitelistStatusChanged\",\"inputs\":[{\"name\":\"deployer\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"whitelisted\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false}]",
	Bin: "0x608060405234801561001057600080fd5b506107bc806100206000396000f3fe608060405234801561001057600080fd5b506004361061007d5760003560e01c80638da5cb5b1161005b5780638da5cb5b146100fc5780639b19251a14610141578063b1540a0114610174578063bdc7b54f1461018757600080fd5b806308fd63221461008257806313af40351461009757806354fd4d50146100aa575b600080fd5b6100956100903660046106de565b61018f565b005b6100956100a536600461071a565b6102ef565b6100e66040518060400160405280600581526020017f312e312e3000000000000000000000000000000000000000000000000000000081525081565b6040516100f3919061073c565b60405180910390f35b60005461011c9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100f3565b61016461014f36600461071a565b60016020526000908152604090205460ff1681565b60405190151581526020016100f3565b61016461018236600461071a565b610520565b610095610571565b60005473ffffffffffffffffffffffffffffffffffffffff163314610261576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff821660008181526001602090815260409182902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00168515159081179091558251938452908301527f8daaf060c3306c38e068a75c054bf96ecd85a3db1252712c4d93632744c42e0d910160405180910390a15050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146103bc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a401610258565b73ffffffffffffffffffffffffffffffffffffffff8116610485576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f4465706c6f79657257686974656c6973743a2063616e206f6e6c79206265206460448201527f697361626c65642076696120656e61626c65417262697472617279436f6e747260648201527f6163744465706c6f796d656e7400000000000000000000000000000000000000608482015260a401610258565b6000546040805173ffffffffffffffffffffffffffffffffffffffff928316815291831660208301527fb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c910160405180910390a1600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6000805473ffffffffffffffffffffffffffffffffffffffff16158061056b575073ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604090205460ff165b92915050565b60005473ffffffffffffffffffffffffffffffffffffffff16331461063e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a401610258565b60005460405173ffffffffffffffffffffffffffffffffffffffff90911681527fc0e106cf568e50698fdbde1eff56f5a5c966cc7958e37e276918e9e4ccdf8cd49060200160405180910390a1600080547fffffffffffffffffffffffff0000000000000000000000000000000000000000169055565b803573ffffffffffffffffffffffffffffffffffffffff811681146106d957600080fd5b919050565b600080604083850312156106f157600080fd5b6106fa836106b5565b91506020830135801515811461070f57600080fd5b809150509250929050565b60006020828403121561072c57600080fd5b610735826106b5565b9392505050565b600060208083528351808285015260005b818110156107695785810183015185820160400152820161074d565b8181111561077b576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01692909201604001939250505056fea164736f6c634300080f000a",
}

// DeployerWhitelistABI is the input ABI used to generate the binding from.
// Deprecated: Use DeployerWhitelistMetaData.ABI instead.
var DeployerWhitelistABI = DeployerWhitelistMetaData.ABI

// DeployerWhitelistBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DeployerWhitelistMetaData.Bin instead.
var DeployerWhitelistBin = DeployerWhitelistMetaData.Bin

// DeployDeployerWhitelist deploys a new Ethereum contract, binding an instance of DeployerWhitelist to it.
func DeployDeployerWhitelist(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DeployerWhitelist, error) {
	parsed, err := DeployerWhitelistMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DeployerWhitelistBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DeployerWhitelist{DeployerWhitelistCaller: DeployerWhitelistCaller{contract: contract}, DeployerWhitelistTransactor: DeployerWhitelistTransactor{contract: contract}, DeployerWhitelistFilterer: DeployerWhitelistFilterer{contract: contract}}, nil
}

// DeployerWhitelist is an auto generated Go binding around an Ethereum contract.
type DeployerWhitelist struct {
	DeployerWhitelistCaller     // Read-only binding to the contract
	DeployerWhitelistTransactor // Write-only binding to the contract
	DeployerWhitelistFilterer   // Log filterer for contract events
}

// DeployerWhitelistCaller is an auto generated read-only Go binding around an Ethereum contract.
type DeployerWhitelistCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeployerWhitelistTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DeployerWhitelistTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeployerWhitelistFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DeployerWhitelistFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DeployerWhitelistSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DeployerWhitelistSession struct {
	Contract     *DeployerWhitelist // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// DeployerWhitelistCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DeployerWhitelistCallerSession struct {
	Contract *DeployerWhitelistCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// DeployerWhitelistTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DeployerWhitelistTransactorSession struct {
	Contract     *DeployerWhitelistTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// DeployerWhitelistRaw is an auto generated low-level Go binding around an Ethereum contract.
type DeployerWhitelistRaw struct {
	Contract *DeployerWhitelist // Generic contract binding to access the raw methods on
}

// DeployerWhitelistCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DeployerWhitelistCallerRaw struct {
	Contract *DeployerWhitelistCaller // Generic read-only contract binding to access the raw methods on
}

// DeployerWhitelistTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DeployerWhitelistTransactorRaw struct {
	Contract *DeployerWhitelistTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDeployerWhitelist creates a new instance of DeployerWhitelist, bound to a specific deployed contract.
func NewDeployerWhitelist(address common.Address, backend bind.ContractBackend) (*DeployerWhitelist, error) {
	contract, err := bindDeployerWhitelist(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelist{DeployerWhitelistCaller: DeployerWhitelistCaller{contract: contract}, DeployerWhitelistTransactor: DeployerWhitelistTransactor{contract: contract}, DeployerWhitelistFilterer: DeployerWhitelistFilterer{contract: contract}}, nil
}

// NewDeployerWhitelistCaller creates a new read-only instance of DeployerWhitelist, bound to a specific deployed contract.
func NewDeployerWhitelistCaller(address common.Address, caller bind.ContractCaller) (*DeployerWhitelistCaller, error) {
	contract, err := bindDeployerWhitelist(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistCaller{contract: contract}, nil
}

// NewDeployerWhitelistTransactor creates a new write-only instance of DeployerWhitelist, bound to a specific deployed contract.
func NewDeployerWhitelistTransactor(address common.Address, transactor bind.ContractTransactor) (*DeployerWhitelistTransactor, error) {
	contract, err := bindDeployerWhitelist(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistTransactor{contract: contract}, nil
}

// NewDeployerWhitelistFilterer creates a new log filterer instance of DeployerWhitelist, bound to a specific deployed contract.
func NewDeployerWhitelistFilterer(address common.Address, filterer bind.ContractFilterer) (*DeployerWhitelistFilterer, error) {
	contract, err := bindDeployerWhitelist(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistFilterer{contract: contract}, nil
}

// bindDeployerWhitelist binds a generic wrapper to an already deployed contract.
func bindDeployerWhitelist(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DeployerWhitelistABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DeployerWhitelist *DeployerWhitelistRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DeployerWhitelist.Contract.DeployerWhitelistCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DeployerWhitelist *DeployerWhitelistRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.DeployerWhitelistTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DeployerWhitelist *DeployerWhitelistRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.DeployerWhitelistTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DeployerWhitelist *DeployerWhitelistCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DeployerWhitelist.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DeployerWhitelist *DeployerWhitelistTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DeployerWhitelist *DeployerWhitelistTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.contract.Transact(opts, method, params...)
}

// IsDeployerAllowed is a free data retrieval call binding the contract method 0xb1540a01.
//
// Solidity: function isDeployerAllowed(address _deployer) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistCaller) IsDeployerAllowed(opts *bind.CallOpts, _deployer common.Address) (bool, error) {
	var out []interface{}
	err := _DeployerWhitelist.contract.Call(opts, &out, "isDeployerAllowed", _deployer)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsDeployerAllowed is a free data retrieval call binding the contract method 0xb1540a01.
//
// Solidity: function isDeployerAllowed(address _deployer) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistSession) IsDeployerAllowed(_deployer common.Address) (bool, error) {
	return _DeployerWhitelist.Contract.IsDeployerAllowed(&_DeployerWhitelist.CallOpts, _deployer)
}

// IsDeployerAllowed is a free data retrieval call binding the contract method 0xb1540a01.
//
// Solidity: function isDeployerAllowed(address _deployer) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistCallerSession) IsDeployerAllowed(_deployer common.Address) (bool, error) {
	return _DeployerWhitelist.Contract.IsDeployerAllowed(&_DeployerWhitelist.CallOpts, _deployer)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DeployerWhitelist *DeployerWhitelistCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DeployerWhitelist.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DeployerWhitelist *DeployerWhitelistSession) Owner() (common.Address, error) {
	return _DeployerWhitelist.Contract.Owner(&_DeployerWhitelist.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DeployerWhitelist *DeployerWhitelistCallerSession) Owner() (common.Address, error) {
	return _DeployerWhitelist.Contract.Owner(&_DeployerWhitelist.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DeployerWhitelist *DeployerWhitelistCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DeployerWhitelist.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DeployerWhitelist *DeployerWhitelistSession) Version() (string, error) {
	return _DeployerWhitelist.Contract.Version(&_DeployerWhitelist.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DeployerWhitelist *DeployerWhitelistCallerSession) Version() (string, error) {
	return _DeployerWhitelist.Contract.Version(&_DeployerWhitelist.CallOpts)
}

// Whitelist is a free data retrieval call binding the contract method 0x9b19251a.
//
// Solidity: function whitelist(address ) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistCaller) Whitelist(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _DeployerWhitelist.contract.Call(opts, &out, "whitelist", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Whitelist is a free data retrieval call binding the contract method 0x9b19251a.
//
// Solidity: function whitelist(address ) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistSession) Whitelist(arg0 common.Address) (bool, error) {
	return _DeployerWhitelist.Contract.Whitelist(&_DeployerWhitelist.CallOpts, arg0)
}

// Whitelist is a free data retrieval call binding the contract method 0x9b19251a.
//
// Solidity: function whitelist(address ) view returns(bool)
func (_DeployerWhitelist *DeployerWhitelistCallerSession) Whitelist(arg0 common.Address) (bool, error) {
	return _DeployerWhitelist.Contract.Whitelist(&_DeployerWhitelist.CallOpts, arg0)
}

// EnableArbitraryContractDeployment is a paid mutator transaction binding the contract method 0xbdc7b54f.
//
// Solidity: function enableArbitraryContractDeployment() returns()
func (_DeployerWhitelist *DeployerWhitelistTransactor) EnableArbitraryContractDeployment(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DeployerWhitelist.contract.Transact(opts, "enableArbitraryContractDeployment")
}

// EnableArbitraryContractDeployment is a paid mutator transaction binding the contract method 0xbdc7b54f.
//
// Solidity: function enableArbitraryContractDeployment() returns()
func (_DeployerWhitelist *DeployerWhitelistSession) EnableArbitraryContractDeployment() (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.EnableArbitraryContractDeployment(&_DeployerWhitelist.TransactOpts)
}

// EnableArbitraryContractDeployment is a paid mutator transaction binding the contract method 0xbdc7b54f.
//
// Solidity: function enableArbitraryContractDeployment() returns()
func (_DeployerWhitelist *DeployerWhitelistTransactorSession) EnableArbitraryContractDeployment() (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.EnableArbitraryContractDeployment(&_DeployerWhitelist.TransactOpts)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_DeployerWhitelist *DeployerWhitelistTransactor) SetOwner(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _DeployerWhitelist.contract.Transact(opts, "setOwner", _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_DeployerWhitelist *DeployerWhitelistSession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.SetOwner(&_DeployerWhitelist.TransactOpts, _owner)
}

// SetOwner is a paid mutator transaction binding the contract method 0x13af4035.
//
// Solidity: function setOwner(address _owner) returns()
func (_DeployerWhitelist *DeployerWhitelistTransactorSession) SetOwner(_owner common.Address) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.SetOwner(&_DeployerWhitelist.TransactOpts, _owner)
}

// SetWhitelistedDeployer is a paid mutator transaction binding the contract method 0x08fd6322.
//
// Solidity: function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) returns()
func (_DeployerWhitelist *DeployerWhitelistTransactor) SetWhitelistedDeployer(opts *bind.TransactOpts, _deployer common.Address, _isWhitelisted bool) (*types.Transaction, error) {
	return _DeployerWhitelist.contract.Transact(opts, "setWhitelistedDeployer", _deployer, _isWhitelisted)
}

// SetWhitelistedDeployer is a paid mutator transaction binding the contract method 0x08fd6322.
//
// Solidity: function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) returns()
func (_DeployerWhitelist *DeployerWhitelistSession) SetWhitelistedDeployer(_deployer common.Address, _isWhitelisted bool) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.SetWhitelistedDeployer(&_DeployerWhitelist.TransactOpts, _deployer, _isWhitelisted)
}

// SetWhitelistedDeployer is a paid mutator transaction binding the contract method 0x08fd6322.
//
// Solidity: function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) returns()
func (_DeployerWhitelist *DeployerWhitelistTransactorSession) SetWhitelistedDeployer(_deployer common.Address, _isWhitelisted bool) (*types.Transaction, error) {
	return _DeployerWhitelist.Contract.SetWhitelistedDeployer(&_DeployerWhitelist.TransactOpts, _deployer, _isWhitelisted)
}

// DeployerWhitelistOwnerChangedIterator is returned from FilterOwnerChanged and is used to iterate over the raw logs and unpacked data for OwnerChanged events raised by the DeployerWhitelist contract.
type DeployerWhitelistOwnerChangedIterator struct {
	Event *DeployerWhitelistOwnerChanged // Event containing the contract specifics and raw log

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
func (it *DeployerWhitelistOwnerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DeployerWhitelistOwnerChanged)
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
		it.Event = new(DeployerWhitelistOwnerChanged)
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
func (it *DeployerWhitelistOwnerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DeployerWhitelistOwnerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DeployerWhitelistOwnerChanged represents a OwnerChanged event raised by the DeployerWhitelist contract.
type DeployerWhitelistOwnerChanged struct {
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOwnerChanged is a free log retrieval operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address oldOwner, address newOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) FilterOwnerChanged(opts *bind.FilterOpts) (*DeployerWhitelistOwnerChangedIterator, error) {

	logs, sub, err := _DeployerWhitelist.contract.FilterLogs(opts, "OwnerChanged")
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistOwnerChangedIterator{contract: _DeployerWhitelist.contract, event: "OwnerChanged", logs: logs, sub: sub}, nil
}

// WatchOwnerChanged is a free log subscription operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address oldOwner, address newOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) WatchOwnerChanged(opts *bind.WatchOpts, sink chan<- *DeployerWhitelistOwnerChanged) (event.Subscription, error) {

	logs, sub, err := _DeployerWhitelist.contract.WatchLogs(opts, "OwnerChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DeployerWhitelistOwnerChanged)
				if err := _DeployerWhitelist.contract.UnpackLog(event, "OwnerChanged", log); err != nil {
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

// ParseOwnerChanged is a log parse operation binding the contract event 0xb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c.
//
// Solidity: event OwnerChanged(address oldOwner, address newOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) ParseOwnerChanged(log types.Log) (*DeployerWhitelistOwnerChanged, error) {
	event := new(DeployerWhitelistOwnerChanged)
	if err := _DeployerWhitelist.contract.UnpackLog(event, "OwnerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DeployerWhitelistWhitelistDisabledIterator is returned from FilterWhitelistDisabled and is used to iterate over the raw logs and unpacked data for WhitelistDisabled events raised by the DeployerWhitelist contract.
type DeployerWhitelistWhitelistDisabledIterator struct {
	Event *DeployerWhitelistWhitelistDisabled // Event containing the contract specifics and raw log

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
func (it *DeployerWhitelistWhitelistDisabledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DeployerWhitelistWhitelistDisabled)
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
		it.Event = new(DeployerWhitelistWhitelistDisabled)
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
func (it *DeployerWhitelistWhitelistDisabledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DeployerWhitelistWhitelistDisabledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DeployerWhitelistWhitelistDisabled represents a WhitelistDisabled event raised by the DeployerWhitelist contract.
type DeployerWhitelistWhitelistDisabled struct {
	OldOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterWhitelistDisabled is a free log retrieval operation binding the contract event 0xc0e106cf568e50698fdbde1eff56f5a5c966cc7958e37e276918e9e4ccdf8cd4.
//
// Solidity: event WhitelistDisabled(address oldOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) FilterWhitelistDisabled(opts *bind.FilterOpts) (*DeployerWhitelistWhitelistDisabledIterator, error) {

	logs, sub, err := _DeployerWhitelist.contract.FilterLogs(opts, "WhitelistDisabled")
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistWhitelistDisabledIterator{contract: _DeployerWhitelist.contract, event: "WhitelistDisabled", logs: logs, sub: sub}, nil
}

// WatchWhitelistDisabled is a free log subscription operation binding the contract event 0xc0e106cf568e50698fdbde1eff56f5a5c966cc7958e37e276918e9e4ccdf8cd4.
//
// Solidity: event WhitelistDisabled(address oldOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) WatchWhitelistDisabled(opts *bind.WatchOpts, sink chan<- *DeployerWhitelistWhitelistDisabled) (event.Subscription, error) {

	logs, sub, err := _DeployerWhitelist.contract.WatchLogs(opts, "WhitelistDisabled")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DeployerWhitelistWhitelistDisabled)
				if err := _DeployerWhitelist.contract.UnpackLog(event, "WhitelistDisabled", log); err != nil {
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

// ParseWhitelistDisabled is a log parse operation binding the contract event 0xc0e106cf568e50698fdbde1eff56f5a5c966cc7958e37e276918e9e4ccdf8cd4.
//
// Solidity: event WhitelistDisabled(address oldOwner)
func (_DeployerWhitelist *DeployerWhitelistFilterer) ParseWhitelistDisabled(log types.Log) (*DeployerWhitelistWhitelistDisabled, error) {
	event := new(DeployerWhitelistWhitelistDisabled)
	if err := _DeployerWhitelist.contract.UnpackLog(event, "WhitelistDisabled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DeployerWhitelistWhitelistStatusChangedIterator is returned from FilterWhitelistStatusChanged and is used to iterate over the raw logs and unpacked data for WhitelistStatusChanged events raised by the DeployerWhitelist contract.
type DeployerWhitelistWhitelistStatusChangedIterator struct {
	Event *DeployerWhitelistWhitelistStatusChanged // Event containing the contract specifics and raw log

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
func (it *DeployerWhitelistWhitelistStatusChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DeployerWhitelistWhitelistStatusChanged)
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
		it.Event = new(DeployerWhitelistWhitelistStatusChanged)
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
func (it *DeployerWhitelistWhitelistStatusChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DeployerWhitelistWhitelistStatusChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DeployerWhitelistWhitelistStatusChanged represents a WhitelistStatusChanged event raised by the DeployerWhitelist contract.
type DeployerWhitelistWhitelistStatusChanged struct {
	Deployer    common.Address
	Whitelisted bool
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWhitelistStatusChanged is a free log retrieval operation binding the contract event 0x8daaf060c3306c38e068a75c054bf96ecd85a3db1252712c4d93632744c42e0d.
//
// Solidity: event WhitelistStatusChanged(address deployer, bool whitelisted)
func (_DeployerWhitelist *DeployerWhitelistFilterer) FilterWhitelistStatusChanged(opts *bind.FilterOpts) (*DeployerWhitelistWhitelistStatusChangedIterator, error) {

	logs, sub, err := _DeployerWhitelist.contract.FilterLogs(opts, "WhitelistStatusChanged")
	if err != nil {
		return nil, err
	}
	return &DeployerWhitelistWhitelistStatusChangedIterator{contract: _DeployerWhitelist.contract, event: "WhitelistStatusChanged", logs: logs, sub: sub}, nil
}

// WatchWhitelistStatusChanged is a free log subscription operation binding the contract event 0x8daaf060c3306c38e068a75c054bf96ecd85a3db1252712c4d93632744c42e0d.
//
// Solidity: event WhitelistStatusChanged(address deployer, bool whitelisted)
func (_DeployerWhitelist *DeployerWhitelistFilterer) WatchWhitelistStatusChanged(opts *bind.WatchOpts, sink chan<- *DeployerWhitelistWhitelistStatusChanged) (event.Subscription, error) {

	logs, sub, err := _DeployerWhitelist.contract.WatchLogs(opts, "WhitelistStatusChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DeployerWhitelistWhitelistStatusChanged)
				if err := _DeployerWhitelist.contract.UnpackLog(event, "WhitelistStatusChanged", log); err != nil {
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

// ParseWhitelistStatusChanged is a log parse operation binding the contract event 0x8daaf060c3306c38e068a75c054bf96ecd85a3db1252712c4d93632744c42e0d.
//
// Solidity: event WhitelistStatusChanged(address deployer, bool whitelisted)
func (_DeployerWhitelist *DeployerWhitelistFilterer) ParseWhitelistStatusChanged(log types.Log) (*DeployerWhitelistWhitelistStatusChanged, error) {
	event := new(DeployerWhitelistWhitelistStatusChanged)
	if err := _DeployerWhitelist.contract.UnpackLog(event, "WhitelistStatusChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
