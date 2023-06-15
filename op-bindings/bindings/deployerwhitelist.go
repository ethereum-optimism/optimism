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

// DeployerWhitelistMetaData contains all meta data concerning the DeployerWhitelist contract.
var DeployerWhitelistMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnerChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"}],\"name\":\"WhitelistDisabled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"whitelisted\",\"type\":\"bool\"}],\"name\":\"WhitelistStatusChanged\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"enableArbitraryContractDeployment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_deployer\",\"type\":\"address\"}],\"name\":\"isDeployerAllowed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"setOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_deployer\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"_isWhitelisted\",\"type\":\"bool\"}],\"name\":\"setWhitelistedDeployer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b5060016080819052600060a081905260c081905280610b3761004a823960006105450152600061051c015260006104f30152610b376000f3fe608060405234801561001057600080fd5b506004361061007d5760003560e01c80638da5cb5b1161005b5780638da5cb5b146100c85780639b19251a1461010d578063b1540a0114610140578063bdc7b54f1461015357600080fd5b806308fd63221461008257806313af40351461009757806354fd4d50146100aa575b600080fd5b61009561009036600461088a565b61015b565b005b6100956100a53660046108c6565b6102bb565b6100b26104ec565b6040516100bf9190610918565b60405180910390f35b6000546100e89073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100bf565b61013061011b3660046108c6565b60016020526000908152604090205460ff1681565b60405190151581526020016100bf565b61013061014e3660046108c6565b61058f565b6100956105e0565b60005473ffffffffffffffffffffffffffffffffffffffff16331461022d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff821660008181526001602090815260409182902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00168515159081179091558251938452908301527f8daaf060c3306c38e068a75c054bf96ecd85a3db1252712c4d93632744c42e0d910160405180910390a15050565b60005473ffffffffffffffffffffffffffffffffffffffff163314610388576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a401610224565b73ffffffffffffffffffffffffffffffffffffffff8116610451576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f4465706c6f79657257686974656c6973743a2063616e206f6e6c79206265206460448201527f697361626c65642076696120656e61626c65417262697472617279436f6e747260648201527f6163744465706c6f796d656e7400000000000000000000000000000000000000608482015260a401610224565b6000546040805173ffffffffffffffffffffffffffffffffffffffff928316815291831660208301527fb532073b38c83145e3e5135377a08bf9aab55bc0fd7c1179cd4fb995d2a5159c910160405180910390a1600080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60606105177f0000000000000000000000000000000000000000000000000000000000000000610724565b6105407f0000000000000000000000000000000000000000000000000000000000000000610724565b6105697f0000000000000000000000000000000000000000000000000000000000000000610724565b60405160200161057b93929190610969565b604051602081830303815290604052905090565b6000805473ffffffffffffffffffffffffffffffffffffffff1615806105da575073ffffffffffffffffffffffffffffffffffffffff821660009081526001602052604090205460ff165b92915050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146106ad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f4465706c6f79657257686974656c6973743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c656420627920746865206f776e6572206f66207460648201527f68697320636f6e74726163740000000000000000000000000000000000000000608482015260a401610224565b60005460405173ffffffffffffffffffffffffffffffffffffffff90911681527fc0e106cf568e50698fdbde1eff56f5a5c966cc7958e37e276918e9e4ccdf8cd49060200160405180910390a1600080547fffffffffffffffffffffffff0000000000000000000000000000000000000000169055565b60608160000361076757505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610791578061077b81610a0e565b915061078a9050600a83610a75565b915061076b565b60008167ffffffffffffffff8111156107ac576107ac610a89565b6040519080825280601f01601f1916602001820160405280156107d6576020820181803683370190505b5090505b8415610859576107eb600183610ab8565b91506107f8600a86610acf565b610803906030610ae3565b60f81b81838151811061081857610818610afb565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610852600a86610a75565b94506107da565b949350505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461088557600080fd5b919050565b6000806040838503121561089d57600080fd5b6108a683610861565b9150602083013580151581146108bb57600080fd5b809150509250929050565b6000602082840312156108d857600080fd5b6108e182610861565b9392505050565b60005b838110156109035781810151838201526020016108eb565b83811115610912576000848401525b50505050565b60208152600082518060208401526109378160408501602087016108e8565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000845161097b8184602089016108e8565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516109b7816001850160208a016108e8565b600192019182015283516109d28160028401602088016108e8565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610a3f57610a3f6109df565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610a8457610a84610a46565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082821015610aca57610aca6109df565b500390565b600082610ade57610ade610a46565b500690565b60008219821115610af657610af66109df565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
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
	parsed, err := DeployerWhitelistMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
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
