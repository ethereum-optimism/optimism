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

// StorageSetterSlot is an auto generated low-level Go binding around an user-defined struct.
type StorageSetterSlot struct {
	Key   [32]byte
	Value [32]byte
}

// StorageSetterMetaData contains all meta data concerning the StorageSetter contract.
var StorageSetterMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"}],\"name\":\"getAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"addr_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"}],\"name\":\"getBool\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"value_\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"}],\"name\":\"getBytes32\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"value_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"}],\"name\":\"getUint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"value_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"setAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"_value\",\"type\":\"bool\"}],\"name\":\"setBool\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"internalType\":\"structStorageSetter.Slot[]\",\"name\":\"slots\",\"type\":\"tuple[]\"}],\"name\":\"setBytes32\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_value\",\"type\":\"bytes32\"}],\"name\":\"setBytes32\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_slot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"setUint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// StorageSetterABI is the input ABI used to generate the binding from.
// Deprecated: Use StorageSetterMetaData.ABI instead.
var StorageSetterABI = StorageSetterMetaData.ABI

// StorageSetter is an auto generated Go binding around an Ethereum contract.
type StorageSetter struct {
	StorageSetterCaller     // Read-only binding to the contract
	StorageSetterTransactor // Write-only binding to the contract
	StorageSetterFilterer   // Log filterer for contract events
}

// StorageSetterCaller is an auto generated read-only Go binding around an Ethereum contract.
type StorageSetterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageSetterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StorageSetterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageSetterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StorageSetterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StorageSetterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StorageSetterSession struct {
	Contract     *StorageSetter    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StorageSetterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StorageSetterCallerSession struct {
	Contract *StorageSetterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// StorageSetterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StorageSetterTransactorSession struct {
	Contract     *StorageSetterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// StorageSetterRaw is an auto generated low-level Go binding around an Ethereum contract.
type StorageSetterRaw struct {
	Contract *StorageSetter // Generic contract binding to access the raw methods on
}

// StorageSetterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StorageSetterCallerRaw struct {
	Contract *StorageSetterCaller // Generic read-only contract binding to access the raw methods on
}

// StorageSetterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StorageSetterTransactorRaw struct {
	Contract *StorageSetterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStorageSetter creates a new instance of StorageSetter, bound to a specific deployed contract.
func NewStorageSetter(address common.Address, backend bind.ContractBackend) (*StorageSetter, error) {
	contract, err := bindStorageSetter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StorageSetter{StorageSetterCaller: StorageSetterCaller{contract: contract}, StorageSetterTransactor: StorageSetterTransactor{contract: contract}, StorageSetterFilterer: StorageSetterFilterer{contract: contract}}, nil
}

// NewStorageSetterCaller creates a new read-only instance of StorageSetter, bound to a specific deployed contract.
func NewStorageSetterCaller(address common.Address, caller bind.ContractCaller) (*StorageSetterCaller, error) {
	contract, err := bindStorageSetter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StorageSetterCaller{contract: contract}, nil
}

// NewStorageSetterTransactor creates a new write-only instance of StorageSetter, bound to a specific deployed contract.
func NewStorageSetterTransactor(address common.Address, transactor bind.ContractTransactor) (*StorageSetterTransactor, error) {
	contract, err := bindStorageSetter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StorageSetterTransactor{contract: contract}, nil
}

// NewStorageSetterFilterer creates a new log filterer instance of StorageSetter, bound to a specific deployed contract.
func NewStorageSetterFilterer(address common.Address, filterer bind.ContractFilterer) (*StorageSetterFilterer, error) {
	contract, err := bindStorageSetter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StorageSetterFilterer{contract: contract}, nil
}

// bindStorageSetter binds a generic wrapper to an already deployed contract.
func bindStorageSetter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StorageSetterABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageSetter *StorageSetterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageSetter.Contract.StorageSetterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageSetter *StorageSetterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageSetter.Contract.StorageSetterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageSetter *StorageSetterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageSetter.Contract.StorageSetterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StorageSetter *StorageSetterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StorageSetter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StorageSetter *StorageSetterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StorageSetter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StorageSetter *StorageSetterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StorageSetter.Contract.contract.Transact(opts, method, params...)
}

// GetAddress is a free data retrieval call binding the contract method 0x21f8a721.
//
// Solidity: function getAddress(bytes32 _slot) view returns(address addr_)
func (_StorageSetter *StorageSetterCaller) GetAddress(opts *bind.CallOpts, _slot [32]byte) (common.Address, error) {
	var out []interface{}
	err := _StorageSetter.contract.Call(opts, &out, "getAddress", _slot)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAddress is a free data retrieval call binding the contract method 0x21f8a721.
//
// Solidity: function getAddress(bytes32 _slot) view returns(address addr_)
func (_StorageSetter *StorageSetterSession) GetAddress(_slot [32]byte) (common.Address, error) {
	return _StorageSetter.Contract.GetAddress(&_StorageSetter.CallOpts, _slot)
}

// GetAddress is a free data retrieval call binding the contract method 0x21f8a721.
//
// Solidity: function getAddress(bytes32 _slot) view returns(address addr_)
func (_StorageSetter *StorageSetterCallerSession) GetAddress(_slot [32]byte) (common.Address, error) {
	return _StorageSetter.Contract.GetAddress(&_StorageSetter.CallOpts, _slot)
}

// GetBool is a free data retrieval call binding the contract method 0x7ae1cfca.
//
// Solidity: function getBool(bytes32 _slot) view returns(bool value_)
func (_StorageSetter *StorageSetterCaller) GetBool(opts *bind.CallOpts, _slot [32]byte) (bool, error) {
	var out []interface{}
	err := _StorageSetter.contract.Call(opts, &out, "getBool", _slot)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetBool is a free data retrieval call binding the contract method 0x7ae1cfca.
//
// Solidity: function getBool(bytes32 _slot) view returns(bool value_)
func (_StorageSetter *StorageSetterSession) GetBool(_slot [32]byte) (bool, error) {
	return _StorageSetter.Contract.GetBool(&_StorageSetter.CallOpts, _slot)
}

// GetBool is a free data retrieval call binding the contract method 0x7ae1cfca.
//
// Solidity: function getBool(bytes32 _slot) view returns(bool value_)
func (_StorageSetter *StorageSetterCallerSession) GetBool(_slot [32]byte) (bool, error) {
	return _StorageSetter.Contract.GetBool(&_StorageSetter.CallOpts, _slot)
}

// GetBytes32 is a free data retrieval call binding the contract method 0xa6ed563e.
//
// Solidity: function getBytes32(bytes32 _slot) view returns(bytes32 value_)
func (_StorageSetter *StorageSetterCaller) GetBytes32(opts *bind.CallOpts, _slot [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _StorageSetter.contract.Call(opts, &out, "getBytes32", _slot)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBytes32 is a free data retrieval call binding the contract method 0xa6ed563e.
//
// Solidity: function getBytes32(bytes32 _slot) view returns(bytes32 value_)
func (_StorageSetter *StorageSetterSession) GetBytes32(_slot [32]byte) ([32]byte, error) {
	return _StorageSetter.Contract.GetBytes32(&_StorageSetter.CallOpts, _slot)
}

// GetBytes32 is a free data retrieval call binding the contract method 0xa6ed563e.
//
// Solidity: function getBytes32(bytes32 _slot) view returns(bytes32 value_)
func (_StorageSetter *StorageSetterCallerSession) GetBytes32(_slot [32]byte) ([32]byte, error) {
	return _StorageSetter.Contract.GetBytes32(&_StorageSetter.CallOpts, _slot)
}

// GetUint is a free data retrieval call binding the contract method 0xbd02d0f5.
//
// Solidity: function getUint(bytes32 _slot) view returns(uint256 value_)
func (_StorageSetter *StorageSetterCaller) GetUint(opts *bind.CallOpts, _slot [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _StorageSetter.contract.Call(opts, &out, "getUint", _slot)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetUint is a free data retrieval call binding the contract method 0xbd02d0f5.
//
// Solidity: function getUint(bytes32 _slot) view returns(uint256 value_)
func (_StorageSetter *StorageSetterSession) GetUint(_slot [32]byte) (*big.Int, error) {
	return _StorageSetter.Contract.GetUint(&_StorageSetter.CallOpts, _slot)
}

// GetUint is a free data retrieval call binding the contract method 0xbd02d0f5.
//
// Solidity: function getUint(bytes32 _slot) view returns(uint256 value_)
func (_StorageSetter *StorageSetterCallerSession) GetUint(_slot [32]byte) (*big.Int, error) {
	return _StorageSetter.Contract.GetUint(&_StorageSetter.CallOpts, _slot)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_StorageSetter *StorageSetterCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _StorageSetter.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_StorageSetter *StorageSetterSession) Version() (string, error) {
	return _StorageSetter.Contract.Version(&_StorageSetter.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_StorageSetter *StorageSetterCallerSession) Version() (string, error) {
	return _StorageSetter.Contract.Version(&_StorageSetter.CallOpts)
}

// SetAddress is a paid mutator transaction binding the contract method 0xca446dd9.
//
// Solidity: function setAddress(bytes32 _slot, address _address) returns()
func (_StorageSetter *StorageSetterTransactor) SetAddress(opts *bind.TransactOpts, _slot [32]byte, _address common.Address) (*types.Transaction, error) {
	return _StorageSetter.contract.Transact(opts, "setAddress", _slot, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0xca446dd9.
//
// Solidity: function setAddress(bytes32 _slot, address _address) returns()
func (_StorageSetter *StorageSetterSession) SetAddress(_slot [32]byte, _address common.Address) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetAddress(&_StorageSetter.TransactOpts, _slot, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0xca446dd9.
//
// Solidity: function setAddress(bytes32 _slot, address _address) returns()
func (_StorageSetter *StorageSetterTransactorSession) SetAddress(_slot [32]byte, _address common.Address) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetAddress(&_StorageSetter.TransactOpts, _slot, _address)
}

// SetBool is a paid mutator transaction binding the contract method 0xabfdcced.
//
// Solidity: function setBool(bytes32 _slot, bool _value) returns()
func (_StorageSetter *StorageSetterTransactor) SetBool(opts *bind.TransactOpts, _slot [32]byte, _value bool) (*types.Transaction, error) {
	return _StorageSetter.contract.Transact(opts, "setBool", _slot, _value)
}

// SetBool is a paid mutator transaction binding the contract method 0xabfdcced.
//
// Solidity: function setBool(bytes32 _slot, bool _value) returns()
func (_StorageSetter *StorageSetterSession) SetBool(_slot [32]byte, _value bool) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBool(&_StorageSetter.TransactOpts, _slot, _value)
}

// SetBool is a paid mutator transaction binding the contract method 0xabfdcced.
//
// Solidity: function setBool(bytes32 _slot, bool _value) returns()
func (_StorageSetter *StorageSetterTransactorSession) SetBool(_slot [32]byte, _value bool) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBool(&_StorageSetter.TransactOpts, _slot, _value)
}

// SetBytes32 is a paid mutator transaction binding the contract method 0x0528afe2.
//
// Solidity: function setBytes32((bytes32,bytes32)[] slots) returns()
func (_StorageSetter *StorageSetterTransactor) SetBytes32(opts *bind.TransactOpts, slots []StorageSetterSlot) (*types.Transaction, error) {
	return _StorageSetter.contract.Transact(opts, "setBytes32", slots)
}

// SetBytes32 is a paid mutator transaction binding the contract method 0x0528afe2.
//
// Solidity: function setBytes32((bytes32,bytes32)[] slots) returns()
func (_StorageSetter *StorageSetterSession) SetBytes32(slots []StorageSetterSlot) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBytes32(&_StorageSetter.TransactOpts, slots)
}

// SetBytes32 is a paid mutator transaction binding the contract method 0x0528afe2.
//
// Solidity: function setBytes32((bytes32,bytes32)[] slots) returns()
func (_StorageSetter *StorageSetterTransactorSession) SetBytes32(slots []StorageSetterSlot) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBytes32(&_StorageSetter.TransactOpts, slots)
}

// SetBytes320 is a paid mutator transaction binding the contract method 0x4e91db08.
//
// Solidity: function setBytes32(bytes32 _slot, bytes32 _value) returns()
func (_StorageSetter *StorageSetterTransactor) SetBytes320(opts *bind.TransactOpts, _slot [32]byte, _value [32]byte) (*types.Transaction, error) {
	return _StorageSetter.contract.Transact(opts, "setBytes320", _slot, _value)
}

// SetBytes320 is a paid mutator transaction binding the contract method 0x4e91db08.
//
// Solidity: function setBytes32(bytes32 _slot, bytes32 _value) returns()
func (_StorageSetter *StorageSetterSession) SetBytes320(_slot [32]byte, _value [32]byte) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBytes320(&_StorageSetter.TransactOpts, _slot, _value)
}

// SetBytes320 is a paid mutator transaction binding the contract method 0x4e91db08.
//
// Solidity: function setBytes32(bytes32 _slot, bytes32 _value) returns()
func (_StorageSetter *StorageSetterTransactorSession) SetBytes320(_slot [32]byte, _value [32]byte) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetBytes320(&_StorageSetter.TransactOpts, _slot, _value)
}

// SetUint is a paid mutator transaction binding the contract method 0xe2a4853a.
//
// Solidity: function setUint(bytes32 _slot, uint256 _value) returns()
func (_StorageSetter *StorageSetterTransactor) SetUint(opts *bind.TransactOpts, _slot [32]byte, _value *big.Int) (*types.Transaction, error) {
	return _StorageSetter.contract.Transact(opts, "setUint", _slot, _value)
}

// SetUint is a paid mutator transaction binding the contract method 0xe2a4853a.
//
// Solidity: function setUint(bytes32 _slot, uint256 _value) returns()
func (_StorageSetter *StorageSetterSession) SetUint(_slot [32]byte, _value *big.Int) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetUint(&_StorageSetter.TransactOpts, _slot, _value)
}

// SetUint is a paid mutator transaction binding the contract method 0xe2a4853a.
//
// Solidity: function setUint(bytes32 _slot, uint256 _value) returns()
func (_StorageSetter *StorageSetterTransactorSession) SetUint(_slot [32]byte, _value *big.Int) (*types.Transaction, error) {
	return _StorageSetter.Contract.SetUint(&_StorageSetter.TransactOpts, _slot, _value)
}
