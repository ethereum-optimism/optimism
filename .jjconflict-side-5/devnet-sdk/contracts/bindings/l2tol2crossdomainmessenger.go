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

// Identifier is an auto generated low-level Go binding around an user-defined struct.
type Identifier struct {
	Origin      common.Address
	BlockNumber *big.Int
	LogIndex    *big.Int
	Timestamp   *big.Int
	ChainId     *big.Int
}

// L2ToL2CrossDomainMessengerMetaData contains all meta data concerning the L2ToL2CrossDomainMessenger contract.
var L2ToL2CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"crossDomainMessageContext\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"sender_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"source_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"crossDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"sender_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"crossDomainMessageSource\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"source_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageNonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageVersion\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"origin\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"logIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"internalType\":\"structIdentifier\",\"name\":\"_id\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"_sentMessage\",\"type\":\"bytes\"}],\"name\":\"relayMessage\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"returnData_\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_destination\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"sendMessage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"successfulMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"source\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"destination\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"EventPayloadNotSentMessage\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"IdOriginNotL2ToL2CrossDomainMessenger\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidChainId\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MessageAlreadyRelayed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MessageDestinationNotRelayChain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MessageDestinationSameChain\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MessageTargetCrossL2Inbox\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MessageTargetL2ToL2CrossDomainMessenger\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotEntered\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ReentrantCall\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TargetCallFailed\",\"type\":\"error\"}]",
}

// L2ToL2CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ToL2CrossDomainMessengerMetaData.ABI instead.
var L2ToL2CrossDomainMessengerABI = L2ToL2CrossDomainMessengerMetaData.ABI

// L2ToL2CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessenger struct {
	L2ToL2CrossDomainMessengerCaller     // Read-only binding to the contract
	L2ToL2CrossDomainMessengerTransactor // Write-only binding to the contract
	L2ToL2CrossDomainMessengerFilterer   // Log filterer for contract events
}

// L2ToL2CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2ToL2CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2ToL2CrossDomainMessengerSession struct {
	Contract     *L2ToL2CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2ToL2CrossDomainMessengerCallerSession struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// L2ToL2CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2ToL2CrossDomainMessengerTransactorSession struct {
	Contract     *L2ToL2CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerRaw struct {
	Contract *L2ToL2CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCallerRaw struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactorRaw struct {
	Contract *L2ToL2CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2ToL2CrossDomainMessenger creates a new instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*L2ToL2CrossDomainMessenger, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessenger{L2ToL2CrossDomainMessengerCaller: L2ToL2CrossDomainMessengerCaller{contract: contract}, L2ToL2CrossDomainMessengerTransactor: L2ToL2CrossDomainMessengerTransactor{contract: contract}, L2ToL2CrossDomainMessengerFilterer: L2ToL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewL2ToL2CrossDomainMessengerCaller creates a new read-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*L2ToL2CrossDomainMessengerCaller, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerCaller{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerTransactor creates a new write-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ToL2CrossDomainMessengerTransactor, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerFilterer creates a new log filterer instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L2ToL2CrossDomainMessengerFilterer, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindL2ToL2CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindL2ToL2CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ToL2CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// CrossDomainMessageContext is a free data retrieval call binding the contract method 0x7936cbee.
//
// Solidity: function crossDomainMessageContext() view returns(address sender_, uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossDomainMessageContext(opts *bind.CallOpts) (struct {
	Sender common.Address
	Source *big.Int
}, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossDomainMessageContext")

	outstruct := new(struct {
		Sender common.Address
		Source *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Sender = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Source = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// CrossDomainMessageContext is a free data retrieval call binding the contract method 0x7936cbee.
//
// Solidity: function crossDomainMessageContext() view returns(address sender_, uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossDomainMessageContext() (struct {
	Sender common.Address
	Source *big.Int
}, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageContext(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageContext is a free data retrieval call binding the contract method 0x7936cbee.
//
// Solidity: function crossDomainMessageContext() view returns(address sender_, uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossDomainMessageContext() (struct {
	Sender common.Address
	Source *big.Int
}, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageContext(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address sender_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address sender_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossDomainMessageSender() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSender(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address sender_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossDomainMessageSender() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSender(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossDomainMessageSource(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossDomainMessageSource")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossDomainMessageSource() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSource(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 source_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossDomainMessageSource() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSource(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageVersion is a free data retrieval call binding the contract method 0x52617f3c.
//
// Solidity: function messageVersion() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MessageVersion(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "messageVersion")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MessageVersion is a free data retrieval call binding the contract method 0x52617f3c.
//
// Solidity: function messageVersion() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MessageVersion() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageVersion(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageVersion is a free data retrieval call binding the contract method 0x52617f3c.
//
// Solidity: function messageVersion() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MessageVersion() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageVersion(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) Version() (string, error) {
	return _L2ToL2CrossDomainMessenger.Contract.Version(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) Version() (string, error) {
	return _L2ToL2CrossDomainMessenger.Contract.Version(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8d1d298f.
//
// Solidity: function relayMessage((address,uint256,uint256,uint256,uint256) _id, bytes _sentMessage) payable returns(bytes returnData_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _id Identifier, _sentMessage []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "relayMessage", _id, _sentMessage)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8d1d298f.
//
// Solidity: function relayMessage((address,uint256,uint256,uint256,uint256) _id, bytes _sentMessage) payable returns(bytes returnData_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) RelayMessage(_id Identifier, _sentMessage []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _id, _sentMessage)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8d1d298f.
//
// Solidity: function relayMessage((address,uint256,uint256,uint256,uint256) _id, bytes _sentMessage) payable returns(bytes returnData_)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) RelayMessage(_id Identifier, _sentMessage []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _id, _sentMessage)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "sendMessage", _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}

// L2ToL2CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerRelayedMessageIterator struct {
	Event *L2ToL2CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL2CrossDomainMessengerRelayedMessage)
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
		it.Event = new(L2ToL2CrossDomainMessengerRelayedMessage)
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
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL2CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerRelayedMessage struct {
	Source       *big.Int
	MessageNonce *big.Int
	MessageHash  [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x5948076590932b9d173029c7df03fe386e755a61c86c7fe2671011a2faa2a379.
//
// Solidity: event RelayedMessage(uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, source []*big.Int, messageNonce []*big.Int, messageHash [][32]byte) (*L2ToL2CrossDomainMessengerRelayedMessageIterator, error) {

	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", sourceRule, messageNonceRule, messageHashRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerRelayedMessageIterator{contract: _L2ToL2CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x5948076590932b9d173029c7df03fe386e755a61c86c7fe2671011a2faa2a379.
//
// Solidity: event RelayedMessage(uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *L2ToL2CrossDomainMessengerRelayedMessage, source []*big.Int, messageNonce []*big.Int, messageHash [][32]byte) (event.Subscription, error) {

	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", sourceRule, messageNonceRule, messageHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL2CrossDomainMessengerRelayedMessage)
				if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x5948076590932b9d173029c7df03fe386e755a61c86c7fe2671011a2faa2a379.
//
// Solidity: event RelayedMessage(uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*L2ToL2CrossDomainMessengerRelayedMessage, error) {
	event := new(L2ToL2CrossDomainMessengerRelayedMessage)
	if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2ToL2CrossDomainMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerSentMessageIterator struct {
	Event *L2ToL2CrossDomainMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *L2ToL2CrossDomainMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL2CrossDomainMessengerSentMessage)
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
		it.Event = new(L2ToL2CrossDomainMessengerSentMessage)
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
func (it *L2ToL2CrossDomainMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL2CrossDomainMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL2CrossDomainMessengerSentMessage represents a SentMessage event raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerSentMessage struct {
	Destination  *big.Int
	Target       common.Address
	MessageNonce *big.Int
	Sender       common.Address
	Message      []byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320.
//
// Solidity: event SentMessage(uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, destination []*big.Int, target []common.Address, messageNonce []*big.Int) (*L2ToL2CrossDomainMessengerSentMessageIterator, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}
	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.FilterLogs(opts, "SentMessage", destinationRule, targetRule, messageNonceRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerSentMessageIterator{contract: _L2ToL2CrossDomainMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320.
//
// Solidity: event SentMessage(uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *L2ToL2CrossDomainMessengerSentMessage, destination []*big.Int, target []common.Address, messageNonce []*big.Int) (event.Subscription, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}
	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.WatchLogs(opts, "SentMessage", destinationRule, targetRule, messageNonceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL2CrossDomainMessengerSentMessage)
				if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320.
//
// Solidity: event SentMessage(uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) ParseSentMessage(log types.Log) (*L2ToL2CrossDomainMessengerSentMessage, error) {
	event := new(L2ToL2CrossDomainMessengerSentMessage)
	if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
