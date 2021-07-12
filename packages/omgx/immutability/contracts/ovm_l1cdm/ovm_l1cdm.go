// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ovm_l1cdm

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Lib_OVMCodecChainBatchHeader is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecChainBatchHeader struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
}

// Lib_OVMCodecChainInclusionProof is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecChainInclusionProof struct {
	Index    *big.Int
	Siblings [][32]byte
}

// iOVM_L1CrossDomainMessengerL2MessageInclusionProof is an auto generated low-level Go binding around an user-defined struct.
type iOVM_L1CrossDomainMessengerL2MessageInclusionProof struct {
	StateRoot            [32]byte
	StateRootBatchHeader Lib_OVMCodecChainBatchHeader
	StateRootProof       Lib_OVMCodecChainInclusionProof
	StateTrieWitness     []byte
	StorageTrieWitness   []byte
}

// OvmL1cdmABI is the input ABI used to generate the binding from.
const OvmL1cdmABI = "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"MessageAllowed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"MessageBlocked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"allowMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"blockMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"blockedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"stateRootBatchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"stateRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"stateTrieWitness\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"storageTrieWitness\",\"type\":\"bytes\"}],\"internalType\":\"structiOVM_L1CrossDomainMessenger.L2MessageInclusionProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"relayedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_gasLimit\",\"type\":\"uint32\"}],\"name\":\"replayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_gasLimit\",\"type\":\"uint32\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"successfulMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// OvmL1cdm is an auto generated Go binding around an Ethereum contract.
type OvmL1cdm struct {
	OvmL1cdmCaller     // Read-only binding to the contract
	OvmL1cdmTransactor // Write-only binding to the contract
	OvmL1cdmFilterer   // Log filterer for contract events
}

// OvmL1cdmCaller is an auto generated read-only Go binding around an Ethereum contract.
type OvmL1cdmCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmL1cdmTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OvmL1cdmTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmL1cdmFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OvmL1cdmFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmL1cdmSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OvmL1cdmSession struct {
	Contract     *OvmL1cdm         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OvmL1cdmCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OvmL1cdmCallerSession struct {
	Contract *OvmL1cdmCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// OvmL1cdmTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OvmL1cdmTransactorSession struct {
	Contract     *OvmL1cdmTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// OvmL1cdmRaw is an auto generated low-level Go binding around an Ethereum contract.
type OvmL1cdmRaw struct {
	Contract *OvmL1cdm // Generic contract binding to access the raw methods on
}

// OvmL1cdmCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OvmL1cdmCallerRaw struct {
	Contract *OvmL1cdmCaller // Generic read-only contract binding to access the raw methods on
}

// OvmL1cdmTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OvmL1cdmTransactorRaw struct {
	Contract *OvmL1cdmTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOvmL1cdm creates a new instance of OvmL1cdm, bound to a specific deployed contract.
func NewOvmL1cdm(address common.Address, backend bind.ContractBackend) (*OvmL1cdm, error) {
	contract, err := bindOvmL1cdm(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdm{OvmL1cdmCaller: OvmL1cdmCaller{contract: contract}, OvmL1cdmTransactor: OvmL1cdmTransactor{contract: contract}, OvmL1cdmFilterer: OvmL1cdmFilterer{contract: contract}}, nil
}

// NewOvmL1cdmCaller creates a new read-only instance of OvmL1cdm, bound to a specific deployed contract.
func NewOvmL1cdmCaller(address common.Address, caller bind.ContractCaller) (*OvmL1cdmCaller, error) {
	contract, err := bindOvmL1cdm(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmCaller{contract: contract}, nil
}

// NewOvmL1cdmTransactor creates a new write-only instance of OvmL1cdm, bound to a specific deployed contract.
func NewOvmL1cdmTransactor(address common.Address, transactor bind.ContractTransactor) (*OvmL1cdmTransactor, error) {
	contract, err := bindOvmL1cdm(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmTransactor{contract: contract}, nil
}

// NewOvmL1cdmFilterer creates a new log filterer instance of OvmL1cdm, bound to a specific deployed contract.
func NewOvmL1cdmFilterer(address common.Address, filterer bind.ContractFilterer) (*OvmL1cdmFilterer, error) {
	contract, err := bindOvmL1cdm(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmFilterer{contract: contract}, nil
}

// bindOvmL1cdm binds a generic wrapper to an already deployed contract.
func bindOvmL1cdm(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OvmL1cdmABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OvmL1cdm *OvmL1cdmRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OvmL1cdm.Contract.OvmL1cdmCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OvmL1cdm *OvmL1cdmRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.OvmL1cdmTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OvmL1cdm *OvmL1cdmRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.OvmL1cdmTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OvmL1cdm *OvmL1cdmCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OvmL1cdm.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OvmL1cdm *OvmL1cdmTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OvmL1cdm *OvmL1cdmTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.contract.Transact(opts, method, params...)
}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCaller) BlockedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "blockedMessages", arg0)
	return *ret0, err
}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmSession) BlockedMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.BlockedMessages(&_OvmL1cdm.CallOpts, arg0)
}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCallerSession) BlockedMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.BlockedMessages(&_OvmL1cdm.CallOpts, arg0)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmL1cdm *OvmL1cdmCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "libAddressManager")
	return *ret0, err
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmL1cdm *OvmL1cdmSession) LibAddressManager() (common.Address, error) {
	return _OvmL1cdm.Contract.LibAddressManager(&_OvmL1cdm.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmL1cdm *OvmL1cdmCallerSession) LibAddressManager() (common.Address, error) {
	return _OvmL1cdm.Contract.LibAddressManager(&_OvmL1cdm.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OvmL1cdm *OvmL1cdmCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OvmL1cdm *OvmL1cdmSession) Owner() (common.Address, error) {
	return _OvmL1cdm.Contract.Owner(&_OvmL1cdm.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OvmL1cdm *OvmL1cdmCallerSession) Owner() (common.Address, error) {
	return _OvmL1cdm.Contract.Owner(&_OvmL1cdm.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OvmL1cdm *OvmL1cdmCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "paused")
	return *ret0, err
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OvmL1cdm *OvmL1cdmSession) Paused() (bool, error) {
	return _OvmL1cdm.Contract.Paused(&_OvmL1cdm.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OvmL1cdm *OvmL1cdmCallerSession) Paused() (bool, error) {
	return _OvmL1cdm.Contract.Paused(&_OvmL1cdm.CallOpts)
}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCaller) RelayedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "relayedMessages", arg0)
	return *ret0, err
}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmSession) RelayedMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.RelayedMessages(&_OvmL1cdm.CallOpts, arg0)
}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCallerSession) RelayedMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.RelayedMessages(&_OvmL1cdm.CallOpts, arg0)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmL1cdm *OvmL1cdmCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "resolve", _name)
	return *ret0, err
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmL1cdm *OvmL1cdmSession) Resolve(_name string) (common.Address, error) {
	return _OvmL1cdm.Contract.Resolve(&_OvmL1cdm.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmL1cdm *OvmL1cdmCallerSession) Resolve(_name string) (common.Address, error) {
	return _OvmL1cdm.Contract.Resolve(&_OvmL1cdm.CallOpts, _name)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "successfulMessages", arg0)
	return *ret0, err
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.SuccessfulMessages(&_OvmL1cdm.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OvmL1cdm *OvmL1cdmCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _OvmL1cdm.Contract.SuccessfulMessages(&_OvmL1cdm.CallOpts, arg0)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OvmL1cdm *OvmL1cdmCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmL1cdm.contract.Call(opts, out, "xDomainMessageSender")
	return *ret0, err
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OvmL1cdm *OvmL1cdmSession) XDomainMessageSender() (common.Address, error) {
	return _OvmL1cdm.Contract.XDomainMessageSender(&_OvmL1cdm.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OvmL1cdm *OvmL1cdmCallerSession) XDomainMessageSender() (common.Address, error) {
	return _OvmL1cdm.Contract.XDomainMessageSender(&_OvmL1cdm.CallOpts)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) AllowMessage(opts *bind.TransactOpts, _xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "allowMessage", _xDomainCalldataHash)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmSession) AllowMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.AllowMessage(&_OvmL1cdm.TransactOpts, _xDomainCalldataHash)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) AllowMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.AllowMessage(&_OvmL1cdm.TransactOpts, _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) BlockMessage(opts *bind.TransactOpts, _xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "blockMessage", _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmSession) BlockMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.BlockMessage(&_OvmL1cdm.TransactOpts, _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) BlockMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.BlockMessage(&_OvmL1cdm.TransactOpts, _xDomainCalldataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) Initialize(opts *bind.TransactOpts, _libAddressManager common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "initialize", _libAddressManager)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OvmL1cdm *OvmL1cdmSession) Initialize(_libAddressManager common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.Initialize(&_OvmL1cdm.TransactOpts, _libAddressManager)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) Initialize(_libAddressManager common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.Initialize(&_OvmL1cdm.TransactOpts, _libAddressManager)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OvmL1cdm *OvmL1cdmTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OvmL1cdm *OvmL1cdmSession) Pause() (*types.Transaction, error) {
	return _OvmL1cdm.Contract.Pause(&_OvmL1cdm.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) Pause() (*types.Transaction, error) {
	return _OvmL1cdm.Contract.Pause(&_OvmL1cdm.TransactOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) RelayMessage(opts *bind.TransactOpts, _target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "relayMessage", _target, _sender, _message, _messageNonce, _proof)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OvmL1cdm *OvmL1cdmSession) RelayMessage(_target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.RelayMessage(&_OvmL1cdm.TransactOpts, _target, _sender, _message, _messageNonce, _proof)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) RelayMessage(_target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.RelayMessage(&_OvmL1cdm.TransactOpts, _target, _sender, _message, _messageNonce, _proof)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OvmL1cdm *OvmL1cdmTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OvmL1cdm *OvmL1cdmSession) RenounceOwnership() (*types.Transaction, error) {
	return _OvmL1cdm.Contract.RenounceOwnership(&_OvmL1cdm.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _OvmL1cdm.Contract.RenounceOwnership(&_OvmL1cdm.TransactOpts)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) ReplayMessage(opts *bind.TransactOpts, _target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "replayMessage", _target, _sender, _message, _queueIndex, _gasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmSession) ReplayMessage(_target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.ReplayMessage(&_OvmL1cdm.TransactOpts, _target, _sender, _message, _queueIndex, _gasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) ReplayMessage(_target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.ReplayMessage(&_OvmL1cdm.TransactOpts, _target, _sender, _message, _queueIndex, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) SendMessage(opts *bind.TransactOpts, _target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "sendMessage", _target, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmSession) SendMessage(_target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.SendMessage(&_OvmL1cdm.TransactOpts, _target, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) SendMessage(_target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.SendMessage(&_OvmL1cdm.TransactOpts, _target, _message, _gasLimit)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OvmL1cdm *OvmL1cdmTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OvmL1cdm *OvmL1cdmSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.TransferOwnership(&_OvmL1cdm.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OvmL1cdm *OvmL1cdmTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OvmL1cdm.Contract.TransferOwnership(&_OvmL1cdm.TransactOpts, newOwner)
}

// OvmL1cdmFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the OvmL1cdm contract.
type OvmL1cdmFailedRelayedMessageIterator struct {
	Event *OvmL1cdmFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmFailedRelayedMessage)
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
		it.Event = new(OvmL1cdmFailedRelayedMessage)
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
func (it *OvmL1cdmFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmFailedRelayedMessage represents a FailedRelayedMessage event raised by the OvmL1cdm contract.
type OvmL1cdmFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts) (*OvmL1cdmFailedRelayedMessageIterator, error) {

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "FailedRelayedMessage")
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmFailedRelayedMessageIterator{contract: _OvmL1cdm.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *OvmL1cdmFailedRelayedMessage) (event.Subscription, error) {

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "FailedRelayedMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmFailedRelayedMessage)
				if err := _OvmL1cdm.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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
// Solidity: event FailedRelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseFailedRelayedMessage(log types.Log) (*OvmL1cdmFailedRelayedMessage, error) {
	event := new(OvmL1cdmFailedRelayedMessage)
	if err := _OvmL1cdm.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmMessageAllowedIterator is returned from FilterMessageAllowed and is used to iterate over the raw logs and unpacked data for MessageAllowed events raised by the OvmL1cdm contract.
type OvmL1cdmMessageAllowedIterator struct {
	Event *OvmL1cdmMessageAllowed // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmMessageAllowedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmMessageAllowed)
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
		it.Event = new(OvmL1cdmMessageAllowed)
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
func (it *OvmL1cdmMessageAllowedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmMessageAllowedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmMessageAllowed represents a MessageAllowed event raised by the OvmL1cdm contract.
type OvmL1cdmMessageAllowed struct {
	XDomainCalldataHash [32]byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterMessageAllowed is a free log retrieval operation binding the contract event 0x52c8a2680a9f4cc0ad0bf88f32096eadbebf0646ea611d93a0ce6a29a0240405.
//
// Solidity: event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterMessageAllowed(opts *bind.FilterOpts, _xDomainCalldataHash [][32]byte) (*OvmL1cdmMessageAllowedIterator, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "MessageAllowed", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmMessageAllowedIterator{contract: _OvmL1cdm.contract, event: "MessageAllowed", logs: logs, sub: sub}, nil
}

// WatchMessageAllowed is a free log subscription operation binding the contract event 0x52c8a2680a9f4cc0ad0bf88f32096eadbebf0646ea611d93a0ce6a29a0240405.
//
// Solidity: event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchMessageAllowed(opts *bind.WatchOpts, sink chan<- *OvmL1cdmMessageAllowed, _xDomainCalldataHash [][32]byte) (event.Subscription, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "MessageAllowed", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmMessageAllowed)
				if err := _OvmL1cdm.contract.UnpackLog(event, "MessageAllowed", log); err != nil {
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

// ParseMessageAllowed is a log parse operation binding the contract event 0x52c8a2680a9f4cc0ad0bf88f32096eadbebf0646ea611d93a0ce6a29a0240405.
//
// Solidity: event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseMessageAllowed(log types.Log) (*OvmL1cdmMessageAllowed, error) {
	event := new(OvmL1cdmMessageAllowed)
	if err := _OvmL1cdm.contract.UnpackLog(event, "MessageAllowed", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmMessageBlockedIterator is returned from FilterMessageBlocked and is used to iterate over the raw logs and unpacked data for MessageBlocked events raised by the OvmL1cdm contract.
type OvmL1cdmMessageBlockedIterator struct {
	Event *OvmL1cdmMessageBlocked // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmMessageBlockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmMessageBlocked)
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
		it.Event = new(OvmL1cdmMessageBlocked)
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
func (it *OvmL1cdmMessageBlockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmMessageBlockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmMessageBlocked represents a MessageBlocked event raised by the OvmL1cdm contract.
type OvmL1cdmMessageBlocked struct {
	XDomainCalldataHash [32]byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterMessageBlocked is a free log retrieval operation binding the contract event 0xf52508d5339edf0d7e5060a416df98db067af561bdc60872d29c0439eaa13a02.
//
// Solidity: event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterMessageBlocked(opts *bind.FilterOpts, _xDomainCalldataHash [][32]byte) (*OvmL1cdmMessageBlockedIterator, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "MessageBlocked", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmMessageBlockedIterator{contract: _OvmL1cdm.contract, event: "MessageBlocked", logs: logs, sub: sub}, nil
}

// WatchMessageBlocked is a free log subscription operation binding the contract event 0xf52508d5339edf0d7e5060a416df98db067af561bdc60872d29c0439eaa13a02.
//
// Solidity: event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchMessageBlocked(opts *bind.WatchOpts, sink chan<- *OvmL1cdmMessageBlocked, _xDomainCalldataHash [][32]byte) (event.Subscription, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "MessageBlocked", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmMessageBlocked)
				if err := _OvmL1cdm.contract.UnpackLog(event, "MessageBlocked", log); err != nil {
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

// ParseMessageBlocked is a log parse operation binding the contract event 0xf52508d5339edf0d7e5060a416df98db067af561bdc60872d29c0439eaa13a02.
//
// Solidity: event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseMessageBlocked(log types.Log) (*OvmL1cdmMessageBlocked, error) {
	event := new(OvmL1cdmMessageBlocked)
	if err := _OvmL1cdm.contract.UnpackLog(event, "MessageBlocked", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the OvmL1cdm contract.
type OvmL1cdmOwnershipTransferredIterator struct {
	Event *OvmL1cdmOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmOwnershipTransferred)
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
		it.Event = new(OvmL1cdmOwnershipTransferred)
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
func (it *OvmL1cdmOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmOwnershipTransferred represents a OwnershipTransferred event raised by the OvmL1cdm contract.
type OvmL1cdmOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*OvmL1cdmOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmOwnershipTransferredIterator{contract: _OvmL1cdm.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *OvmL1cdmOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmOwnershipTransferred)
				if err := _OvmL1cdm.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseOwnershipTransferred(log types.Log) (*OvmL1cdmOwnershipTransferred, error) {
	event := new(OvmL1cdmOwnershipTransferred)
	if err := _OvmL1cdm.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the OvmL1cdm contract.
type OvmL1cdmPausedIterator struct {
	Event *OvmL1cdmPaused // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmPaused)
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
		it.Event = new(OvmL1cdmPaused)
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
func (it *OvmL1cdmPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmPaused represents a Paused event raised by the OvmL1cdm contract.
type OvmL1cdmPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterPaused(opts *bind.FilterOpts) (*OvmL1cdmPausedIterator, error) {

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmPausedIterator{contract: _OvmL1cdm.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *OvmL1cdmPaused) (event.Subscription, error) {

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmPaused)
				if err := _OvmL1cdm.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) ParsePaused(log types.Log) (*OvmL1cdmPaused, error) {
	event := new(OvmL1cdmPaused)
	if err := _OvmL1cdm.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the OvmL1cdm contract.
type OvmL1cdmRelayedMessageIterator struct {
	Event *OvmL1cdmRelayedMessage // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmRelayedMessage)
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
		it.Event = new(OvmL1cdmRelayedMessage)
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
func (it *OvmL1cdmRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmRelayedMessage represents a RelayedMessage event raised by the OvmL1cdm contract.
type OvmL1cdmRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterRelayedMessage(opts *bind.FilterOpts) (*OvmL1cdmRelayedMessageIterator, error) {

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "RelayedMessage")
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmRelayedMessageIterator{contract: _OvmL1cdm.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *OvmL1cdmRelayedMessage) (event.Subscription, error) {

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "RelayedMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmRelayedMessage)
				if err := _OvmL1cdm.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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
// Solidity: event RelayedMessage(bytes32 msgHash)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseRelayedMessage(log types.Log) (*OvmL1cdmRelayedMessage, error) {
	event := new(OvmL1cdmRelayedMessage)
	if err := _OvmL1cdm.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the OvmL1cdm contract.
type OvmL1cdmSentMessageIterator struct {
	Event *OvmL1cdmSentMessage // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmSentMessage)
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
		it.Event = new(OvmL1cdmSentMessage)
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
func (it *OvmL1cdmSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmSentMessage represents a SentMessage event raised by the OvmL1cdm contract.
type OvmL1cdmSentMessage struct {
	Message []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326.
//
// Solidity: event SentMessage(bytes message)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterSentMessage(opts *bind.FilterOpts) (*OvmL1cdmSentMessageIterator, error) {

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "SentMessage")
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmSentMessageIterator{contract: _OvmL1cdm.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326.
//
// Solidity: event SentMessage(bytes message)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *OvmL1cdmSentMessage) (event.Subscription, error) {

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "SentMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmSentMessage)
				if err := _OvmL1cdm.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326.
//
// Solidity: event SentMessage(bytes message)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseSentMessage(log types.Log) (*OvmL1cdmSentMessage, error) {
	event := new(OvmL1cdmSentMessage)
	if err := _OvmL1cdm.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmL1cdmUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the OvmL1cdm contract.
type OvmL1cdmUnpausedIterator struct {
	Event *OvmL1cdmUnpaused // Event containing the contract specifics and raw log

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
func (it *OvmL1cdmUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmL1cdmUnpaused)
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
		it.Event = new(OvmL1cdmUnpaused)
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
func (it *OvmL1cdmUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmL1cdmUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmL1cdmUnpaused represents a Unpaused event raised by the OvmL1cdm contract.
type OvmL1cdmUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) FilterUnpaused(opts *bind.FilterOpts) (*OvmL1cdmUnpausedIterator, error) {

	logs, sub, err := _OvmL1cdm.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &OvmL1cdmUnpausedIterator{contract: _OvmL1cdm.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *OvmL1cdmUnpaused) (event.Subscription, error) {

	logs, sub, err := _OvmL1cdm.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmL1cdmUnpaused)
				if err := _OvmL1cdm.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_OvmL1cdm *OvmL1cdmFilterer) ParseUnpaused(log types.Log) (*OvmL1cdmUnpaused, error) {
	event := new(OvmL1cdmUnpaused)
	if err := _OvmL1cdm.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	return event, nil
}
