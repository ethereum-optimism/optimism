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

// CrossDomainMessengerMetaData contains all meta data concerning the CrossDomainMessenger contract.
var CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"MESSAGE_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_CALLDATA_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OTHER_MESSENGER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_CALL_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_CONSTANT_OVERHEAD\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_GAS_CHECK_BUFFER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RELAY_RESERVED_GAS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseGas\",\"inputs\":[{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"failedMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messageNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"relayMessage\",\"inputs\":[{\"name\":\"_nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"sendMessage\",\"inputs\":[{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_minGasLimit\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"successfulMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"xDomainMessageSender\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FailedRelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentMessage\",\"inputs\":[{\"name\":\"target\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sender\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"message\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"messageNonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentMessageExtension1\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

// CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossDomainMessengerMetaData.ABI instead.
var CrossDomainMessengerABI = CrossDomainMessengerMetaData.ABI

// CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type CrossDomainMessenger struct {
	CrossDomainMessengerCaller     // Read-only binding to the contract
	CrossDomainMessengerTransactor // Write-only binding to the contract
	CrossDomainMessengerFilterer   // Log filterer for contract events
}

// CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrossDomainMessengerSession struct {
	Contract     *CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts         // Call options to use throughout this session
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrossDomainMessengerCallerSession struct {
	Contract *CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts               // Call options to use throughout this session
}

// CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrossDomainMessengerTransactorSession struct {
	Contract     *CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts               // Transaction auth options to use throughout this session
}

// CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrossDomainMessengerRaw struct {
	Contract *CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrossDomainMessengerCallerRaw struct {
	Contract *CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrossDomainMessengerTransactorRaw struct {
	Contract *CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrossDomainMessenger creates a new instance of CrossDomainMessenger, bound to a specific deployed contract.
func NewCrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*CrossDomainMessenger, error) {
	contract, err := bindCrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessenger{CrossDomainMessengerCaller: CrossDomainMessengerCaller{contract: contract}, CrossDomainMessengerTransactor: CrossDomainMessengerTransactor{contract: contract}, CrossDomainMessengerFilterer: CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewCrossDomainMessengerCaller creates a new read-only instance of CrossDomainMessenger, bound to a specific deployed contract.
func NewCrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*CrossDomainMessengerCaller, error) {
	contract, err := bindCrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerCaller{contract: contract}, nil
}

// NewCrossDomainMessengerTransactor creates a new write-only instance of CrossDomainMessenger, bound to a specific deployed contract.
func NewCrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*CrossDomainMessengerTransactor, error) {
	contract, err := bindCrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewCrossDomainMessengerFilterer creates a new log filterer instance of CrossDomainMessenger, bound to a specific deployed contract.
func NewCrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*CrossDomainMessengerFilterer, error) {
	contract, err := bindCrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindCrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindCrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossDomainMessenger *CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossDomainMessenger.Contract.CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossDomainMessenger *CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossDomainMessenger *CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossDomainMessenger *CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossDomainMessenger *CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossDomainMessenger *CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_CrossDomainMessenger *CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _CrossDomainMessenger.Contract.MESSAGEVERSION(&_CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _CrossDomainMessenger.Contract.MESSAGEVERSION(&_CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) MINGASCALLDATAOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_CALLDATA_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADDENOMINATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADNUMERATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_CrossDomainMessenger.CallOpts)
}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) OTHERMESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "OTHER_MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerSession) OTHERMESSENGER() (common.Address, error) {
	return _CrossDomainMessenger.Contract.OTHERMESSENGER(&_CrossDomainMessenger.CallOpts)
}

// OTHERMESSENGER is a free data retrieval call binding the contract method 0x9fce812c.
//
// Solidity: function OTHER_MESSENGER() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) OTHERMESSENGER() (common.Address, error) {
	return _CrossDomainMessenger.Contract.OTHERMESSENGER(&_CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) RELAYCALLOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CALL_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) RELAYCONSTANTOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CONSTANT_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) RELAYGASCHECKBUFFER(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "RELAY_GAS_CHECK_BUFFER")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) RELAYRESERVEDGAS(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "RELAY_RESERVED_GAS")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_CrossDomainMessenger.CallOpts)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) BaseGas(opts *bind.CallOpts, _message []byte, _minGasLimit uint32) (uint64, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "baseGas", _message, _minGasLimit)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _CrossDomainMessenger.Contract.BaseGas(&_CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _CrossDomainMessenger.Contract.BaseGas(&_CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) FailedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "failedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _CrossDomainMessenger.Contract.FailedMessages(&_CrossDomainMessenger.CallOpts, arg0)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _CrossDomainMessenger.Contract.FailedMessages(&_CrossDomainMessenger.CallOpts, arg0)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_CrossDomainMessenger *CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _CrossDomainMessenger.Contract.MessageNonce(&_CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _CrossDomainMessenger.Contract.MessageNonce(&_CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerSession) Paused() (bool, error) {
	return _CrossDomainMessenger.Contract.Paused(&_CrossDomainMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) Paused() (bool, error) {
	return _CrossDomainMessenger.Contract.Paused(&_CrossDomainMessenger.CallOpts)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _CrossDomainMessenger.Contract.SuccessfulMessages(&_CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _CrossDomainMessenger.Contract.SuccessfulMessages(&_CrossDomainMessenger.CallOpts, arg0)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossDomainMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerSession) XDomainMessageSender() (common.Address, error) {
	return _CrossDomainMessenger.Contract.XDomainMessageSender(&_CrossDomainMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_CrossDomainMessenger *CrossDomainMessengerCallerSession) XDomainMessageSender() (common.Address, error) {
	return _CrossDomainMessenger.Contract.XDomainMessageSender(&_CrossDomainMessenger.CallOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _CrossDomainMessenger.contract.Transact(opts, "relayMessage", _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.RelayMessage(&_CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd764ad0b.
//
// Solidity: function relayMessage(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerTransactorSession) RelayMessage(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.RelayMessage(&_CrossDomainMessenger.TransactOpts, _nonce, _sender, _target, _value, _minGasLimit, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _CrossDomainMessenger.contract.Transact(opts, "sendMessage", _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.SendMessage(&_CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_CrossDomainMessenger *CrossDomainMessengerTransactorSession) SendMessage(_target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _CrossDomainMessenger.Contract.SendMessage(&_CrossDomainMessenger.TransactOpts, _target, _message, _minGasLimit)
}

// CrossDomainMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the CrossDomainMessenger contract.
type CrossDomainMessengerFailedRelayedMessageIterator struct {
	Event *CrossDomainMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *CrossDomainMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossDomainMessengerFailedRelayedMessage)
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
		it.Event = new(CrossDomainMessengerFailedRelayedMessage)
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
func (it *CrossDomainMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossDomainMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossDomainMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the CrossDomainMessenger contract.
type CrossDomainMessengerFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*CrossDomainMessengerFailedRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerFailedRelayedMessageIterator{contract: _CrossDomainMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *CrossDomainMessengerFailedRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossDomainMessengerFailedRelayedMessage)
				if err := _CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*CrossDomainMessengerFailedRelayedMessage, error) {
	event := new(CrossDomainMessengerFailedRelayedMessage)
	if err := _CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CrossDomainMessengerInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the CrossDomainMessenger contract.
type CrossDomainMessengerInitializedIterator struct {
	Event *CrossDomainMessengerInitialized // Event containing the contract specifics and raw log

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
func (it *CrossDomainMessengerInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossDomainMessengerInitialized)
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
		it.Event = new(CrossDomainMessengerInitialized)
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
func (it *CrossDomainMessengerInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossDomainMessengerInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossDomainMessengerInitialized represents a Initialized event raised by the CrossDomainMessenger contract.
type CrossDomainMessengerInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) FilterInitialized(opts *bind.FilterOpts) (*CrossDomainMessengerInitializedIterator, error) {

	logs, sub, err := _CrossDomainMessenger.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerInitializedIterator{contract: _CrossDomainMessenger.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *CrossDomainMessengerInitialized) (event.Subscription, error) {

	logs, sub, err := _CrossDomainMessenger.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossDomainMessengerInitialized)
				if err := _CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) ParseInitialized(log types.Log) (*CrossDomainMessengerInitialized, error) {
	event := new(CrossDomainMessengerInitialized)
	if err := _CrossDomainMessenger.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the CrossDomainMessenger contract.
type CrossDomainMessengerRelayedMessageIterator struct {
	Event *CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossDomainMessengerRelayedMessage)
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
		it.Event = new(CrossDomainMessengerRelayedMessage)
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
func (it *CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the CrossDomainMessenger contract.
type CrossDomainMessengerRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*CrossDomainMessengerRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerRelayedMessageIterator{contract: _CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *CrossDomainMessengerRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossDomainMessengerRelayedMessage)
				if err := _CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*CrossDomainMessengerRelayedMessage, error) {
	event := new(CrossDomainMessengerRelayedMessage)
	if err := _CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CrossDomainMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the CrossDomainMessenger contract.
type CrossDomainMessengerSentMessageIterator struct {
	Event *CrossDomainMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *CrossDomainMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossDomainMessengerSentMessage)
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
		it.Event = new(CrossDomainMessengerSentMessage)
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
func (it *CrossDomainMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossDomainMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossDomainMessengerSentMessage represents a SentMessage event raised by the CrossDomainMessenger contract.
type CrossDomainMessengerSentMessage struct {
	Target       common.Address
	Sender       common.Address
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, target []common.Address) (*CrossDomainMessengerSentMessageIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.FilterLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerSentMessageIterator{contract: _CrossDomainMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *CrossDomainMessengerSentMessage, target []common.Address) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.WatchLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossDomainMessengerSentMessage)
				if err := _CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0xcb0f7ffd78f9aee47a248fae8db181db6eee833039123e026dcbff529522e52a.
//
// Solidity: event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) ParseSentMessage(log types.Log) (*CrossDomainMessengerSentMessage, error) {
	event := new(CrossDomainMessengerSentMessage)
	if err := _CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CrossDomainMessengerSentMessageExtension1Iterator is returned from FilterSentMessageExtension1 and is used to iterate over the raw logs and unpacked data for SentMessageExtension1 events raised by the CrossDomainMessenger contract.
type CrossDomainMessengerSentMessageExtension1Iterator struct {
	Event *CrossDomainMessengerSentMessageExtension1 // Event containing the contract specifics and raw log

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
func (it *CrossDomainMessengerSentMessageExtension1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossDomainMessengerSentMessageExtension1)
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
		it.Event = new(CrossDomainMessengerSentMessageExtension1)
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
func (it *CrossDomainMessengerSentMessageExtension1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossDomainMessengerSentMessageExtension1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossDomainMessengerSentMessageExtension1 represents a SentMessageExtension1 event raised by the CrossDomainMessenger contract.
type CrossDomainMessengerSentMessageExtension1 struct {
	Sender common.Address
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterSentMessageExtension1 is a free log retrieval operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) FilterSentMessageExtension1(opts *bind.FilterOpts, sender []common.Address) (*CrossDomainMessengerSentMessageExtension1Iterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.FilterLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return &CrossDomainMessengerSentMessageExtension1Iterator{contract: _CrossDomainMessenger.contract, event: "SentMessageExtension1", logs: logs, sub: sub}, nil
}

// WatchSentMessageExtension1 is a free log subscription operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) WatchSentMessageExtension1(opts *bind.WatchOpts, sink chan<- *CrossDomainMessengerSentMessageExtension1, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _CrossDomainMessenger.contract.WatchLogs(opts, "SentMessageExtension1", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossDomainMessengerSentMessageExtension1)
				if err := _CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
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

// ParseSentMessageExtension1 is a log parse operation binding the contract event 0x8ebb2ec2465bdb2a06a66fc37a0963af8a2a6a1479d81d56fdb8cbb98096d546.
//
// Solidity: event SentMessageExtension1(address indexed sender, uint256 value)
func (_CrossDomainMessenger *CrossDomainMessengerFilterer) ParseSentMessageExtension1(log types.Log) (*CrossDomainMessengerSentMessageExtension1, error) {
	event := new(CrossDomainMessengerSentMessageExtension1)
	if err := _CrossDomainMessenger.contract.UnpackLog(event, "SentMessageExtension1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
