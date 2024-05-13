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

// L1ERC721BridgeMetaData contains all meta data concerning the L1ERC721Bridge contract.
var L1ERC721BridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_BRIDGE\",\"outputs\":[{\"internalType\":\"contractStandardBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC721\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC721To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeERC721\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"contractSuperchainConfig\",\"name\":\"_superchainConfig\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherBridge\",\"outputs\":[{\"internalType\":\"contractStandardBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainConfig\",\"outputs\":[{\"internalType\":\"contractSuperchainConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC721BridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC721BridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"}]",
}

// L1ERC721BridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use L1ERC721BridgeMetaData.ABI instead.
var L1ERC721BridgeABI = L1ERC721BridgeMetaData.ABI

// L1ERC721Bridge is an auto generated Go binding around an Ethereum contract.
type L1ERC721Bridge struct {
	L1ERC721BridgeCaller     // Read-only binding to the contract
	L1ERC721BridgeTransactor // Write-only binding to the contract
	L1ERC721BridgeFilterer   // Log filterer for contract events
}

// L1ERC721BridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1ERC721BridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC721BridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1ERC721BridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC721BridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1ERC721BridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC721BridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1ERC721BridgeSession struct {
	Contract     *L1ERC721Bridge   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1ERC721BridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1ERC721BridgeCallerSession struct {
	Contract *L1ERC721BridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L1ERC721BridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1ERC721BridgeTransactorSession struct {
	Contract     *L1ERC721BridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L1ERC721BridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1ERC721BridgeRaw struct {
	Contract *L1ERC721Bridge // Generic contract binding to access the raw methods on
}

// L1ERC721BridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1ERC721BridgeCallerRaw struct {
	Contract *L1ERC721BridgeCaller // Generic read-only contract binding to access the raw methods on
}

// L1ERC721BridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1ERC721BridgeTransactorRaw struct {
	Contract *L1ERC721BridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1ERC721Bridge creates a new instance of L1ERC721Bridge, bound to a specific deployed contract.
func NewL1ERC721Bridge(address common.Address, backend bind.ContractBackend) (*L1ERC721Bridge, error) {
	contract, err := bindL1ERC721Bridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1ERC721Bridge{L1ERC721BridgeCaller: L1ERC721BridgeCaller{contract: contract}, L1ERC721BridgeTransactor: L1ERC721BridgeTransactor{contract: contract}, L1ERC721BridgeFilterer: L1ERC721BridgeFilterer{contract: contract}}, nil
}

// NewL1ERC721BridgeCaller creates a new read-only instance of L1ERC721Bridge, bound to a specific deployed contract.
func NewL1ERC721BridgeCaller(address common.Address, caller bind.ContractCaller) (*L1ERC721BridgeCaller, error) {
	contract, err := bindL1ERC721Bridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeCaller{contract: contract}, nil
}

// NewL1ERC721BridgeTransactor creates a new write-only instance of L1ERC721Bridge, bound to a specific deployed contract.
func NewL1ERC721BridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*L1ERC721BridgeTransactor, error) {
	contract, err := bindL1ERC721Bridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeTransactor{contract: contract}, nil
}

// NewL1ERC721BridgeFilterer creates a new log filterer instance of L1ERC721Bridge, bound to a specific deployed contract.
func NewL1ERC721BridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*L1ERC721BridgeFilterer, error) {
	contract, err := bindL1ERC721Bridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeFilterer{contract: contract}, nil
}

// bindL1ERC721Bridge binds a generic wrapper to an already deployed contract.
func bindL1ERC721Bridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1ERC721BridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ERC721Bridge *L1ERC721BridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ERC721Bridge.Contract.L1ERC721BridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ERC721Bridge *L1ERC721BridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.L1ERC721BridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ERC721Bridge *L1ERC721BridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.L1ERC721BridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ERC721Bridge *L1ERC721BridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ERC721Bridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ERC721Bridge *L1ERC721BridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ERC721Bridge *L1ERC721BridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.contract.Transact(opts, method, params...)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeSession) MESSENGER() (common.Address, error) {
	return _L1ERC721Bridge.Contract.MESSENGER(&_L1ERC721Bridge.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) MESSENGER() (common.Address, error) {
	return _L1ERC721Bridge.Contract.MESSENGER(&_L1ERC721Bridge.CallOpts)
}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) OTHERBRIDGE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "OTHER_BRIDGE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeSession) OTHERBRIDGE() (common.Address, error) {
	return _L1ERC721Bridge.Contract.OTHERBRIDGE(&_L1ERC721Bridge.CallOpts)
}

// OTHERBRIDGE is a free data retrieval call binding the contract method 0x7f46ddb2.
//
// Solidity: function OTHER_BRIDGE() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) OTHERBRIDGE() (common.Address, error) {
	return _L1ERC721Bridge.Contract.OTHERBRIDGE(&_L1ERC721Bridge.CallOpts)
}

// Deposits is a free data retrieval call binding the contract method 0x5d93a3fc.
//
// Solidity: function deposits(address , address , uint256 ) view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) Deposits(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address, arg2 *big.Int) (bool, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "deposits", arg0, arg1, arg2)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Deposits is a free data retrieval call binding the contract method 0x5d93a3fc.
//
// Solidity: function deposits(address , address , uint256 ) view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeSession) Deposits(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (bool, error) {
	return _L1ERC721Bridge.Contract.Deposits(&_L1ERC721Bridge.CallOpts, arg0, arg1, arg2)
}

// Deposits is a free data retrieval call binding the contract method 0x5d93a3fc.
//
// Solidity: function deposits(address , address , uint256 ) view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) Deposits(arg0 common.Address, arg1 common.Address, arg2 *big.Int) (bool, error) {
	return _L1ERC721Bridge.Contract.Deposits(&_L1ERC721Bridge.CallOpts, arg0, arg1, arg2)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeSession) Messenger() (common.Address, error) {
	return _L1ERC721Bridge.Contract.Messenger(&_L1ERC721Bridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) Messenger() (common.Address, error) {
	return _L1ERC721Bridge.Contract.Messenger(&_L1ERC721Bridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) OtherBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "otherBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeSession) OtherBridge() (common.Address, error) {
	return _L1ERC721Bridge.Contract.OtherBridge(&_L1ERC721Bridge.CallOpts)
}

// OtherBridge is a free data retrieval call binding the contract method 0xc89701a2.
//
// Solidity: function otherBridge() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) OtherBridge() (common.Address, error) {
	return _L1ERC721Bridge.Contract.OtherBridge(&_L1ERC721Bridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeSession) Paused() (bool, error) {
	return _L1ERC721Bridge.Contract.Paused(&_L1ERC721Bridge.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) Paused() (bool, error) {
	return _L1ERC721Bridge.Contract.Paused(&_L1ERC721Bridge.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeSession) SuperchainConfig() (common.Address, error) {
	return _L1ERC721Bridge.Contract.SuperchainConfig(&_L1ERC721Bridge.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) SuperchainConfig() (common.Address, error) {
	return _L1ERC721Bridge.Contract.SuperchainConfig(&_L1ERC721Bridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1ERC721Bridge *L1ERC721BridgeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1ERC721Bridge.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1ERC721Bridge *L1ERC721BridgeSession) Version() (string, error) {
	return _L1ERC721Bridge.Contract.Version(&_L1ERC721Bridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1ERC721Bridge *L1ERC721BridgeCallerSession) Version() (string, error) {
	return _L1ERC721Bridge.Contract.Version(&_L1ERC721Bridge.CallOpts)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactor) BridgeERC721(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.contract.Transact(opts, "bridgeERC721", _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeSession) BridgeERC721(_localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.BridgeERC721(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721 is a paid mutator transaction binding the contract method 0x3687011a.
//
// Solidity: function bridgeERC721(address _localToken, address _remoteToken, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactorSession) BridgeERC721(_localToken common.Address, _remoteToken common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.BridgeERC721(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactor) BridgeERC721To(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.contract.Transact(opts, "bridgeERC721To", _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeSession) BridgeERC721To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.BridgeERC721To(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// BridgeERC721To is a paid mutator transaction binding the contract method 0xaa557452.
//
// Solidity: function bridgeERC721To(address _localToken, address _remoteToken, address _to, uint256 _tokenId, uint32 _minGasLimit, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactorSession) BridgeERC721To(_localToken common.Address, _remoteToken common.Address, _to common.Address, _tokenId *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.BridgeERC721To(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _to, _tokenId, _minGasLimit, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactor) FinalizeBridgeERC721(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.contract.Transact(opts, "finalizeBridgeERC721", _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeSession) FinalizeBridgeERC721(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.FinalizeBridgeERC721(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// FinalizeBridgeERC721 is a paid mutator transaction binding the contract method 0x761f4493.
//
// Solidity: function finalizeBridgeERC721(address _localToken, address _remoteToken, address _from, address _to, uint256 _tokenId, bytes _extraData) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactorSession) FinalizeBridgeERC721(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _tokenId *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.FinalizeBridgeERC721(&_L1ERC721Bridge.TransactOpts, _localToken, _remoteToken, _from, _to, _tokenId, _extraData)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _superchainConfig) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _L1ERC721Bridge.contract.Transact(opts, "initialize", _messenger, _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _superchainConfig) returns()
func (_L1ERC721Bridge *L1ERC721BridgeSession) Initialize(_messenger common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.Initialize(&_L1ERC721Bridge.TransactOpts, _messenger, _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _superchainConfig) returns()
func (_L1ERC721Bridge *L1ERC721BridgeTransactorSession) Initialize(_messenger common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _L1ERC721Bridge.Contract.Initialize(&_L1ERC721Bridge.TransactOpts, _messenger, _superchainConfig)
}

// L1ERC721BridgeERC721BridgeFinalizedIterator is returned from FilterERC721BridgeFinalized and is used to iterate over the raw logs and unpacked data for ERC721BridgeFinalized events raised by the L1ERC721Bridge contract.
type L1ERC721BridgeERC721BridgeFinalizedIterator struct {
	Event *L1ERC721BridgeERC721BridgeFinalized // Event containing the contract specifics and raw log

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
func (it *L1ERC721BridgeERC721BridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ERC721BridgeERC721BridgeFinalized)
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
		it.Event = new(L1ERC721BridgeERC721BridgeFinalized)
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
func (it *L1ERC721BridgeERC721BridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ERC721BridgeERC721BridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ERC721BridgeERC721BridgeFinalized represents a ERC721BridgeFinalized event raised by the L1ERC721Bridge contract.
type L1ERC721BridgeERC721BridgeFinalized struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	TokenId     *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC721BridgeFinalized is a free log retrieval operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) FilterERC721BridgeFinalized(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L1ERC721BridgeERC721BridgeFinalizedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L1ERC721Bridge.contract.FilterLogs(opts, "ERC721BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeERC721BridgeFinalizedIterator{contract: _L1ERC721Bridge.contract, event: "ERC721BridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchERC721BridgeFinalized is a free log subscription operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) WatchERC721BridgeFinalized(opts *bind.WatchOpts, sink chan<- *L1ERC721BridgeERC721BridgeFinalized, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L1ERC721Bridge.contract.WatchLogs(opts, "ERC721BridgeFinalized", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ERC721BridgeERC721BridgeFinalized)
				if err := _L1ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeFinalized", log); err != nil {
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

// ParseERC721BridgeFinalized is a log parse operation binding the contract event 0x1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac.
//
// Solidity: event ERC721BridgeFinalized(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) ParseERC721BridgeFinalized(log types.Log) (*L1ERC721BridgeERC721BridgeFinalized, error) {
	event := new(L1ERC721BridgeERC721BridgeFinalized)
	if err := _L1ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ERC721BridgeERC721BridgeInitiatedIterator is returned from FilterERC721BridgeInitiated and is used to iterate over the raw logs and unpacked data for ERC721BridgeInitiated events raised by the L1ERC721Bridge contract.
type L1ERC721BridgeERC721BridgeInitiatedIterator struct {
	Event *L1ERC721BridgeERC721BridgeInitiated // Event containing the contract specifics and raw log

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
func (it *L1ERC721BridgeERC721BridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ERC721BridgeERC721BridgeInitiated)
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
		it.Event = new(L1ERC721BridgeERC721BridgeInitiated)
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
func (it *L1ERC721BridgeERC721BridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ERC721BridgeERC721BridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ERC721BridgeERC721BridgeInitiated represents a ERC721BridgeInitiated event raised by the L1ERC721Bridge contract.
type L1ERC721BridgeERC721BridgeInitiated struct {
	LocalToken  common.Address
	RemoteToken common.Address
	From        common.Address
	To          common.Address
	TokenId     *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC721BridgeInitiated is a free log retrieval operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) FilterERC721BridgeInitiated(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address, from []common.Address) (*L1ERC721BridgeERC721BridgeInitiatedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L1ERC721Bridge.contract.FilterLogs(opts, "ERC721BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeERC721BridgeInitiatedIterator{contract: _L1ERC721Bridge.contract, event: "ERC721BridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchERC721BridgeInitiated is a free log subscription operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) WatchERC721BridgeInitiated(opts *bind.WatchOpts, sink chan<- *L1ERC721BridgeERC721BridgeInitiated, localToken []common.Address, remoteToken []common.Address, from []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _L1ERC721Bridge.contract.WatchLogs(opts, "ERC721BridgeInitiated", localTokenRule, remoteTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ERC721BridgeERC721BridgeInitiated)
				if err := _L1ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeInitiated", log); err != nil {
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

// ParseERC721BridgeInitiated is a log parse operation binding the contract event 0xb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a5.
//
// Solidity: event ERC721BridgeInitiated(address indexed localToken, address indexed remoteToken, address indexed from, address to, uint256 tokenId, bytes extraData)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) ParseERC721BridgeInitiated(log types.Log) (*L1ERC721BridgeERC721BridgeInitiated, error) {
	event := new(L1ERC721BridgeERC721BridgeInitiated)
	if err := _L1ERC721Bridge.contract.UnpackLog(event, "ERC721BridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ERC721BridgeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1ERC721Bridge contract.
type L1ERC721BridgeInitializedIterator struct {
	Event *L1ERC721BridgeInitialized // Event containing the contract specifics and raw log

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
func (it *L1ERC721BridgeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ERC721BridgeInitialized)
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
		it.Event = new(L1ERC721BridgeInitialized)
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
func (it *L1ERC721BridgeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ERC721BridgeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ERC721BridgeInitialized represents a Initialized event raised by the L1ERC721Bridge contract.
type L1ERC721BridgeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1ERC721BridgeInitializedIterator, error) {

	logs, sub, err := _L1ERC721Bridge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1ERC721BridgeInitializedIterator{contract: _L1ERC721Bridge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1ERC721BridgeInitialized) (event.Subscription, error) {

	logs, sub, err := _L1ERC721Bridge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ERC721BridgeInitialized)
				if err := _L1ERC721Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L1ERC721Bridge *L1ERC721BridgeFilterer) ParseInitialized(log types.Log) (*L1ERC721BridgeInitialized, error) {
	event := new(L1ERC721BridgeInitialized)
	if err := _L1ERC721Bridge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
