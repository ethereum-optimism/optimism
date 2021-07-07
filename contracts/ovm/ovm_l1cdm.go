// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package OVM_L1CDM

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

// OVML1CDMABI is the input ABI used to generate the binding from.
const OVML1CDMABI = "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"MessageAllowed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"MessageBlocked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"allowMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_xDomainCalldataHash\",\"type\":\"bytes32\"}],\"name\":\"blockMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"blockedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_messageNonce\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"stateRootBatchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"stateRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"stateTrieWitness\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"storageTrieWitness\",\"type\":\"bytes\"}],\"internalType\":\"structiOVM_L1CrossDomainMessenger.L2MessageInclusionProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"relayedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_gasLimit\",\"type\":\"uint32\"}],\"name\":\"replayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_gasLimit\",\"type\":\"uint32\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"successfulMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// OVML1CDM is an auto generated Go binding around an Ethereum contract.
type OVML1CDM struct {
	OVML1CDMCaller     // Read-only binding to the contract
	OVML1CDMTransactor // Write-only binding to the contract
	OVML1CDMFilterer   // Log filterer for contract events
}

// OVML1CDMCaller is an auto generated read-only Go binding around an Ethereum contract.
type OVML1CDMCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVML1CDMTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OVML1CDMTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVML1CDMFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OVML1CDMFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVML1CDMSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OVML1CDMSession struct {
	Contract     *OVML1CDM         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OVML1CDMCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OVML1CDMCallerSession struct {
	Contract *OVML1CDMCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// OVML1CDMTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OVML1CDMTransactorSession struct {
	Contract     *OVML1CDMTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// OVML1CDMRaw is an auto generated low-level Go binding around an Ethereum contract.
type OVML1CDMRaw struct {
	Contract *OVML1CDM // Generic contract binding to access the raw methods on
}

// OVML1CDMCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OVML1CDMCallerRaw struct {
	Contract *OVML1CDMCaller // Generic read-only contract binding to access the raw methods on
}

// OVML1CDMTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OVML1CDMTransactorRaw struct {
	Contract *OVML1CDMTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOVML1CDM creates a new instance of OVML1CDM, bound to a specific deployed contract.
func NewOVML1CDM(address common.Address, backend bind.ContractBackend) (*OVML1CDM, error) {
	contract, err := bindOVML1CDM(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OVML1CDM{OVML1CDMCaller: OVML1CDMCaller{contract: contract}, OVML1CDMTransactor: OVML1CDMTransactor{contract: contract}, OVML1CDMFilterer: OVML1CDMFilterer{contract: contract}}, nil
}

// NewOVML1CDMCaller creates a new read-only instance of OVML1CDM, bound to a specific deployed contract.
func NewOVML1CDMCaller(address common.Address, caller bind.ContractCaller) (*OVML1CDMCaller, error) {
	contract, err := bindOVML1CDM(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMCaller{contract: contract}, nil
}

// NewOVML1CDMTransactor creates a new write-only instance of OVML1CDM, bound to a specific deployed contract.
func NewOVML1CDMTransactor(address common.Address, transactor bind.ContractTransactor) (*OVML1CDMTransactor, error) {
	contract, err := bindOVML1CDM(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMTransactor{contract: contract}, nil
}

// NewOVML1CDMFilterer creates a new log filterer instance of OVML1CDM, bound to a specific deployed contract.
func NewOVML1CDMFilterer(address common.Address, filterer bind.ContractFilterer) (*OVML1CDMFilterer, error) {
	contract, err := bindOVML1CDM(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMFilterer{contract: contract}, nil
}

// bindOVML1CDM binds a generic wrapper to an already deployed contract.
func bindOVML1CDM(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OVML1CDMABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVML1CDM *OVML1CDMRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVML1CDM.Contract.OVML1CDMCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVML1CDM *OVML1CDMRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVML1CDM.Contract.OVML1CDMTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVML1CDM *OVML1CDMRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVML1CDM.Contract.OVML1CDMTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVML1CDM *OVML1CDMCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVML1CDM.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVML1CDM *OVML1CDMTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVML1CDM.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVML1CDM *OVML1CDMTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVML1CDM.Contract.contract.Transact(opts, method, params...)
}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCaller) BlockedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "blockedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMSession) BlockedMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.BlockedMessages(&_OVML1CDM.CallOpts, arg0)
}

// BlockedMessages is a free data retrieval call binding the contract method 0xc6b94ab0.
//
// Solidity: function blockedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCallerSession) BlockedMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.BlockedMessages(&_OVML1CDM.CallOpts, arg0)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVML1CDM *OVML1CDMCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVML1CDM *OVML1CDMSession) LibAddressManager() (common.Address, error) {
	return _OVML1CDM.Contract.LibAddressManager(&_OVML1CDM.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVML1CDM *OVML1CDMCallerSession) LibAddressManager() (common.Address, error) {
	return _OVML1CDM.Contract.LibAddressManager(&_OVML1CDM.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OVML1CDM *OVML1CDMCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OVML1CDM *OVML1CDMSession) Owner() (common.Address, error) {
	return _OVML1CDM.Contract.Owner(&_OVML1CDM.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OVML1CDM *OVML1CDMCallerSession) Owner() (common.Address, error) {
	return _OVML1CDM.Contract.Owner(&_OVML1CDM.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OVML1CDM *OVML1CDMCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OVML1CDM *OVML1CDMSession) Paused() (bool, error) {
	return _OVML1CDM.Contract.Paused(&_OVML1CDM.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OVML1CDM *OVML1CDMCallerSession) Paused() (bool, error) {
	return _OVML1CDM.Contract.Paused(&_OVML1CDM.CallOpts)
}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCaller) RelayedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "relayedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMSession) RelayedMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.RelayedMessages(&_OVML1CDM.CallOpts, arg0)
}

// RelayedMessages is a free data retrieval call binding the contract method 0x21d800ec.
//
// Solidity: function relayedMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCallerSession) RelayedMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.RelayedMessages(&_OVML1CDM.CallOpts, arg0)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVML1CDM *OVML1CDMCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVML1CDM *OVML1CDMSession) Resolve(_name string) (common.Address, error) {
	return _OVML1CDM.Contract.Resolve(&_OVML1CDM.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVML1CDM *OVML1CDMCallerSession) Resolve(_name string) (common.Address, error) {
	return _OVML1CDM.Contract.Resolve(&_OVML1CDM.CallOpts, _name)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.SuccessfulMessages(&_OVML1CDM.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_OVML1CDM *OVML1CDMCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _OVML1CDM.Contract.SuccessfulMessages(&_OVML1CDM.CallOpts, arg0)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OVML1CDM *OVML1CDMCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVML1CDM.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OVML1CDM *OVML1CDMSession) XDomainMessageSender() (common.Address, error) {
	return _OVML1CDM.Contract.XDomainMessageSender(&_OVML1CDM.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_OVML1CDM *OVML1CDMCallerSession) XDomainMessageSender() (common.Address, error) {
	return _OVML1CDM.Contract.XDomainMessageSender(&_OVML1CDM.CallOpts)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMTransactor) AllowMessage(opts *bind.TransactOpts, _xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "allowMessage", _xDomainCalldataHash)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMSession) AllowMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.Contract.AllowMessage(&_OVML1CDM.TransactOpts, _xDomainCalldataHash)
}

// AllowMessage is a paid mutator transaction binding the contract method 0x81ada46c.
//
// Solidity: function allowMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) AllowMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.Contract.AllowMessage(&_OVML1CDM.TransactOpts, _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMTransactor) BlockMessage(opts *bind.TransactOpts, _xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "blockMessage", _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMSession) BlockMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.Contract.BlockMessage(&_OVML1CDM.TransactOpts, _xDomainCalldataHash)
}

// BlockMessage is a paid mutator transaction binding the contract method 0x0ecf2eea.
//
// Solidity: function blockMessage(bytes32 _xDomainCalldataHash) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) BlockMessage(_xDomainCalldataHash [32]byte) (*types.Transaction, error) {
	return _OVML1CDM.Contract.BlockMessage(&_OVML1CDM.TransactOpts, _xDomainCalldataHash)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OVML1CDM *OVML1CDMTransactor) Initialize(opts *bind.TransactOpts, _libAddressManager common.Address) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "initialize", _libAddressManager)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OVML1CDM *OVML1CDMSession) Initialize(_libAddressManager common.Address) (*types.Transaction, error) {
	return _OVML1CDM.Contract.Initialize(&_OVML1CDM.TransactOpts, _libAddressManager)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _libAddressManager) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) Initialize(_libAddressManager common.Address) (*types.Transaction, error) {
	return _OVML1CDM.Contract.Initialize(&_OVML1CDM.TransactOpts, _libAddressManager)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OVML1CDM *OVML1CDMTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OVML1CDM *OVML1CDMSession) Pause() (*types.Transaction, error) {
	return _OVML1CDM.Contract.Pause(&_OVML1CDM.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_OVML1CDM *OVML1CDMTransactorSession) Pause() (*types.Transaction, error) {
	return _OVML1CDM.Contract.Pause(&_OVML1CDM.TransactOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OVML1CDM *OVML1CDMTransactor) RelayMessage(opts *bind.TransactOpts, _target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "relayMessage", _target, _sender, _message, _messageNonce, _proof)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OVML1CDM *OVML1CDMSession) RelayMessage(_target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OVML1CDM.Contract.RelayMessage(&_OVML1CDM.TransactOpts, _target, _sender, _message, _messageNonce, _proof)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xd7fd19dd.
//
// Solidity: function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, (bytes32,(uint256,bytes32,uint256,uint256,bytes),(uint256,bytes32[]),bytes,bytes) _proof) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) RelayMessage(_target common.Address, _sender common.Address, _message []byte, _messageNonce *big.Int, _proof iOVM_L1CrossDomainMessengerL2MessageInclusionProof) (*types.Transaction, error) {
	return _OVML1CDM.Contract.RelayMessage(&_OVML1CDM.TransactOpts, _target, _sender, _message, _messageNonce, _proof)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OVML1CDM *OVML1CDMTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OVML1CDM *OVML1CDMSession) RenounceOwnership() (*types.Transaction, error) {
	return _OVML1CDM.Contract.RenounceOwnership(&_OVML1CDM.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OVML1CDM *OVML1CDMTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _OVML1CDM.Contract.RenounceOwnership(&_OVML1CDM.TransactOpts)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMTransactor) ReplayMessage(opts *bind.TransactOpts, _target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "replayMessage", _target, _sender, _message, _queueIndex, _gasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMSession) ReplayMessage(_target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.Contract.ReplayMessage(&_OVML1CDM.TransactOpts, _target, _sender, _message, _queueIndex, _gasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x706ceab6.
//
// Solidity: function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) ReplayMessage(_target common.Address, _sender common.Address, _message []byte, _queueIndex *big.Int, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.Contract.ReplayMessage(&_OVML1CDM.TransactOpts, _target, _sender, _message, _queueIndex, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMTransactor) SendMessage(opts *bind.TransactOpts, _target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "sendMessage", _target, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMSession) SendMessage(_target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.Contract.SendMessage(&_OVML1CDM.TransactOpts, _target, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0x3dbb202b.
//
// Solidity: function sendMessage(address _target, bytes _message, uint32 _gasLimit) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) SendMessage(_target common.Address, _message []byte, _gasLimit uint32) (*types.Transaction, error) {
	return _OVML1CDM.Contract.SendMessage(&_OVML1CDM.TransactOpts, _target, _message, _gasLimit)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OVML1CDM *OVML1CDMTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _OVML1CDM.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OVML1CDM *OVML1CDMSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OVML1CDM.Contract.TransferOwnership(&_OVML1CDM.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OVML1CDM *OVML1CDMTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OVML1CDM.Contract.TransferOwnership(&_OVML1CDM.TransactOpts, newOwner)
}

// OVML1CDMFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the OVML1CDM contract.
type OVML1CDMFailedRelayedMessageIterator struct {
	Event *OVML1CDMFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *OVML1CDMFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMFailedRelayedMessage)
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
		it.Event = new(OVML1CDMFailedRelayedMessage)
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
func (it *OVML1CDMFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMFailedRelayedMessage represents a FailedRelayedMessage event raised by the OVML1CDM contract.
type OVML1CDMFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 msgHash)
func (_OVML1CDM *OVML1CDMFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts) (*OVML1CDMFailedRelayedMessageIterator, error) {

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "FailedRelayedMessage")
	if err != nil {
		return nil, err
	}
	return &OVML1CDMFailedRelayedMessageIterator{contract: _OVML1CDM.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 msgHash)
func (_OVML1CDM *OVML1CDMFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *OVML1CDMFailedRelayedMessage) (event.Subscription, error) {

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "FailedRelayedMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMFailedRelayedMessage)
				if err := _OVML1CDM.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseFailedRelayedMessage(log types.Log) (*OVML1CDMFailedRelayedMessage, error) {
	event := new(OVML1CDMFailedRelayedMessage)
	if err := _OVML1CDM.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMMessageAllowedIterator is returned from FilterMessageAllowed and is used to iterate over the raw logs and unpacked data for MessageAllowed events raised by the OVML1CDM contract.
type OVML1CDMMessageAllowedIterator struct {
	Event *OVML1CDMMessageAllowed // Event containing the contract specifics and raw log

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
func (it *OVML1CDMMessageAllowedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMMessageAllowed)
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
		it.Event = new(OVML1CDMMessageAllowed)
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
func (it *OVML1CDMMessageAllowedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMMessageAllowedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMMessageAllowed represents a MessageAllowed event raised by the OVML1CDM contract.
type OVML1CDMMessageAllowed struct {
	XDomainCalldataHash [32]byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterMessageAllowed is a free log retrieval operation binding the contract event 0x52c8a2680a9f4cc0ad0bf88f32096eadbebf0646ea611d93a0ce6a29a0240405.
//
// Solidity: event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
func (_OVML1CDM *OVML1CDMFilterer) FilterMessageAllowed(opts *bind.FilterOpts, _xDomainCalldataHash [][32]byte) (*OVML1CDMMessageAllowedIterator, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "MessageAllowed", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMMessageAllowedIterator{contract: _OVML1CDM.contract, event: "MessageAllowed", logs: logs, sub: sub}, nil
}

// WatchMessageAllowed is a free log subscription operation binding the contract event 0x52c8a2680a9f4cc0ad0bf88f32096eadbebf0646ea611d93a0ce6a29a0240405.
//
// Solidity: event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
func (_OVML1CDM *OVML1CDMFilterer) WatchMessageAllowed(opts *bind.WatchOpts, sink chan<- *OVML1CDMMessageAllowed, _xDomainCalldataHash [][32]byte) (event.Subscription, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "MessageAllowed", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMMessageAllowed)
				if err := _OVML1CDM.contract.UnpackLog(event, "MessageAllowed", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseMessageAllowed(log types.Log) (*OVML1CDMMessageAllowed, error) {
	event := new(OVML1CDMMessageAllowed)
	if err := _OVML1CDM.contract.UnpackLog(event, "MessageAllowed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMMessageBlockedIterator is returned from FilterMessageBlocked and is used to iterate over the raw logs and unpacked data for MessageBlocked events raised by the OVML1CDM contract.
type OVML1CDMMessageBlockedIterator struct {
	Event *OVML1CDMMessageBlocked // Event containing the contract specifics and raw log

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
func (it *OVML1CDMMessageBlockedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMMessageBlocked)
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
		it.Event = new(OVML1CDMMessageBlocked)
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
func (it *OVML1CDMMessageBlockedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMMessageBlockedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMMessageBlocked represents a MessageBlocked event raised by the OVML1CDM contract.
type OVML1CDMMessageBlocked struct {
	XDomainCalldataHash [32]byte
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterMessageBlocked is a free log retrieval operation binding the contract event 0xf52508d5339edf0d7e5060a416df98db067af561bdc60872d29c0439eaa13a02.
//
// Solidity: event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
func (_OVML1CDM *OVML1CDMFilterer) FilterMessageBlocked(opts *bind.FilterOpts, _xDomainCalldataHash [][32]byte) (*OVML1CDMMessageBlockedIterator, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "MessageBlocked", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMMessageBlockedIterator{contract: _OVML1CDM.contract, event: "MessageBlocked", logs: logs, sub: sub}, nil
}

// WatchMessageBlocked is a free log subscription operation binding the contract event 0xf52508d5339edf0d7e5060a416df98db067af561bdc60872d29c0439eaa13a02.
//
// Solidity: event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
func (_OVML1CDM *OVML1CDMFilterer) WatchMessageBlocked(opts *bind.WatchOpts, sink chan<- *OVML1CDMMessageBlocked, _xDomainCalldataHash [][32]byte) (event.Subscription, error) {

	var _xDomainCalldataHashRule []interface{}
	for _, _xDomainCalldataHashItem := range _xDomainCalldataHash {
		_xDomainCalldataHashRule = append(_xDomainCalldataHashRule, _xDomainCalldataHashItem)
	}

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "MessageBlocked", _xDomainCalldataHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMMessageBlocked)
				if err := _OVML1CDM.contract.UnpackLog(event, "MessageBlocked", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseMessageBlocked(log types.Log) (*OVML1CDMMessageBlocked, error) {
	event := new(OVML1CDMMessageBlocked)
	if err := _OVML1CDM.contract.UnpackLog(event, "MessageBlocked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the OVML1CDM contract.
type OVML1CDMOwnershipTransferredIterator struct {
	Event *OVML1CDMOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *OVML1CDMOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMOwnershipTransferred)
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
		it.Event = new(OVML1CDMOwnershipTransferred)
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
func (it *OVML1CDMOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMOwnershipTransferred represents a OwnershipTransferred event raised by the OVML1CDM contract.
type OVML1CDMOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OVML1CDM *OVML1CDMFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*OVML1CDMOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &OVML1CDMOwnershipTransferredIterator{contract: _OVML1CDM.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OVML1CDM *OVML1CDMFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *OVML1CDMOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMOwnershipTransferred)
				if err := _OVML1CDM.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseOwnershipTransferred(log types.Log) (*OVML1CDMOwnershipTransferred, error) {
	event := new(OVML1CDMOwnershipTransferred)
	if err := _OVML1CDM.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the OVML1CDM contract.
type OVML1CDMPausedIterator struct {
	Event *OVML1CDMPaused // Event containing the contract specifics and raw log

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
func (it *OVML1CDMPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMPaused)
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
		it.Event = new(OVML1CDMPaused)
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
func (it *OVML1CDMPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMPaused represents a Paused event raised by the OVML1CDM contract.
type OVML1CDMPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_OVML1CDM *OVML1CDMFilterer) FilterPaused(opts *bind.FilterOpts) (*OVML1CDMPausedIterator, error) {

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &OVML1CDMPausedIterator{contract: _OVML1CDM.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_OVML1CDM *OVML1CDMFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *OVML1CDMPaused) (event.Subscription, error) {

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMPaused)
				if err := _OVML1CDM.contract.UnpackLog(event, "Paused", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParsePaused(log types.Log) (*OVML1CDMPaused, error) {
	event := new(OVML1CDMPaused)
	if err := _OVML1CDM.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the OVML1CDM contract.
type OVML1CDMRelayedMessageIterator struct {
	Event *OVML1CDMRelayedMessage // Event containing the contract specifics and raw log

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
func (it *OVML1CDMRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMRelayedMessage)
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
		it.Event = new(OVML1CDMRelayedMessage)
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
func (it *OVML1CDMRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMRelayedMessage represents a RelayedMessage event raised by the OVML1CDM contract.
type OVML1CDMRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 msgHash)
func (_OVML1CDM *OVML1CDMFilterer) FilterRelayedMessage(opts *bind.FilterOpts) (*OVML1CDMRelayedMessageIterator, error) {

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "RelayedMessage")
	if err != nil {
		return nil, err
	}
	return &OVML1CDMRelayedMessageIterator{contract: _OVML1CDM.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 msgHash)
func (_OVML1CDM *OVML1CDMFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *OVML1CDMRelayedMessage) (event.Subscription, error) {

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "RelayedMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMRelayedMessage)
				if err := _OVML1CDM.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseRelayedMessage(log types.Log) (*OVML1CDMRelayedMessage, error) {
	event := new(OVML1CDMRelayedMessage)
	if err := _OVML1CDM.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the OVML1CDM contract.
type OVML1CDMSentMessageIterator struct {
	Event *OVML1CDMSentMessage // Event containing the contract specifics and raw log

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
func (it *OVML1CDMSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMSentMessage)
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
		it.Event = new(OVML1CDMSentMessage)
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
func (it *OVML1CDMSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMSentMessage represents a SentMessage event raised by the OVML1CDM contract.
type OVML1CDMSentMessage struct {
	Message []byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326.
//
// Solidity: event SentMessage(bytes message)
func (_OVML1CDM *OVML1CDMFilterer) FilterSentMessage(opts *bind.FilterOpts) (*OVML1CDMSentMessageIterator, error) {

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "SentMessage")
	if err != nil {
		return nil, err
	}
	return &OVML1CDMSentMessageIterator{contract: _OVML1CDM.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0x0ee9ffdb2334d78de97ffb066b23a352a4d35180cefb36589d663fbb1eb6f326.
//
// Solidity: event SentMessage(bytes message)
func (_OVML1CDM *OVML1CDMFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *OVML1CDMSentMessage) (event.Subscription, error) {

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "SentMessage")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMSentMessage)
				if err := _OVML1CDM.contract.UnpackLog(event, "SentMessage", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseSentMessage(log types.Log) (*OVML1CDMSentMessage, error) {
	event := new(OVML1CDMSentMessage)
	if err := _OVML1CDM.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVML1CDMUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the OVML1CDM contract.
type OVML1CDMUnpausedIterator struct {
	Event *OVML1CDMUnpaused // Event containing the contract specifics and raw log

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
func (it *OVML1CDMUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVML1CDMUnpaused)
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
		it.Event = new(OVML1CDMUnpaused)
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
func (it *OVML1CDMUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVML1CDMUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVML1CDMUnpaused represents a Unpaused event raised by the OVML1CDM contract.
type OVML1CDMUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_OVML1CDM *OVML1CDMFilterer) FilterUnpaused(opts *bind.FilterOpts) (*OVML1CDMUnpausedIterator, error) {

	logs, sub, err := _OVML1CDM.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &OVML1CDMUnpausedIterator{contract: _OVML1CDM.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_OVML1CDM *OVML1CDMFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *OVML1CDMUnpaused) (event.Subscription, error) {

	logs, sub, err := _OVML1CDM.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVML1CDMUnpaused)
				if err := _OVML1CDM.contract.UnpackLog(event, "Unpaused", log); err != nil {
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
func (_OVML1CDM *OVML1CDMFilterer) ParseUnpaused(log types.Log) (*OVML1CDMUnpaused, error) {
	event := new(OVML1CDMUnpaused)
	if err := _OVML1CDM.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
