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

// AlphabetVMMetaData contains all meta data concerning the AlphabetVM contract.
var AlphabetVMMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_absolutePrestate\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"postState_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b5060405161030438038061030483398101604081905261002f91610037565b608052610050565b60006020828403121561004957600080fd5b5051919050565b60805161029a61006a6000396000605c015261029a6000f3fe608060405234801561001057600080fd5b506004361061002b5760003560e01c8063f8e0cb9614610030575b600080fd5b61004361003e366004610157565b610055565b60405190815260200160405180910390f35b60008060007f0000000000000000000000000000000000000000000000000000000000000000878760405161008b9291906101c3565b6040518091039020036100af57600091506100a8868801886101d3565b90506100ce565b6100bb868801886101ec565b9092509050816100ca8161023d565b9250505b816100da826001610275565b6040805160208101939093528201526060016040516020818303038152906040528051906020012092505050949350505050565b60008083601f84011261012057600080fd5b50813567ffffffffffffffff81111561013857600080fd5b60208301915083602082850101111561015057600080fd5b9250929050565b6000806000806040858703121561016d57600080fd5b843567ffffffffffffffff8082111561018557600080fd5b6101918883890161010e565b909650945060208701359150808211156101aa57600080fd5b506101b78782880161010e565b95989497509550505050565b8183823760009101908152919050565b6000602082840312156101e557600080fd5b5035919050565b600080604083850312156101ff57600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361026e5761026e61020e565b5060010190565b600082198211156102885761028861020e565b50019056fea164736f6c634300080f000a",
}

// AlphabetVMABI is the input ABI used to generate the binding from.
// Deprecated: Use AlphabetVMMetaData.ABI instead.
var AlphabetVMABI = AlphabetVMMetaData.ABI

// AlphabetVMBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AlphabetVMMetaData.Bin instead.
var AlphabetVMBin = AlphabetVMMetaData.Bin

// DeployAlphabetVM deploys a new Ethereum contract, binding an instance of AlphabetVM to it.
func DeployAlphabetVM(auth *bind.TransactOpts, backend bind.ContractBackend, _absolutePrestate [32]byte) (common.Address, *types.Transaction, *AlphabetVM, error) {
	parsed, err := AlphabetVMMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AlphabetVMBin), backend, _absolutePrestate)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AlphabetVM{AlphabetVMCaller: AlphabetVMCaller{contract: contract}, AlphabetVMTransactor: AlphabetVMTransactor{contract: contract}, AlphabetVMFilterer: AlphabetVMFilterer{contract: contract}}, nil
}

// AlphabetVM is an auto generated Go binding around an Ethereum contract.
type AlphabetVM struct {
	AlphabetVMCaller     // Read-only binding to the contract
	AlphabetVMTransactor // Write-only binding to the contract
	AlphabetVMFilterer   // Log filterer for contract events
}

// AlphabetVMCaller is an auto generated read-only Go binding around an Ethereum contract.
type AlphabetVMCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVMTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AlphabetVMTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVMFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AlphabetVMFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVMSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AlphabetVMSession struct {
	Contract     *AlphabetVM       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AlphabetVMCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AlphabetVMCallerSession struct {
	Contract *AlphabetVMCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AlphabetVMTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AlphabetVMTransactorSession struct {
	Contract     *AlphabetVMTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AlphabetVMRaw is an auto generated low-level Go binding around an Ethereum contract.
type AlphabetVMRaw struct {
	Contract *AlphabetVM // Generic contract binding to access the raw methods on
}

// AlphabetVMCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AlphabetVMCallerRaw struct {
	Contract *AlphabetVMCaller // Generic read-only contract binding to access the raw methods on
}

// AlphabetVMTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AlphabetVMTransactorRaw struct {
	Contract *AlphabetVMTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAlphabetVM creates a new instance of AlphabetVM, bound to a specific deployed contract.
func NewAlphabetVM(address common.Address, backend bind.ContractBackend) (*AlphabetVM, error) {
	contract, err := bindAlphabetVM(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AlphabetVM{AlphabetVMCaller: AlphabetVMCaller{contract: contract}, AlphabetVMTransactor: AlphabetVMTransactor{contract: contract}, AlphabetVMFilterer: AlphabetVMFilterer{contract: contract}}, nil
}

// NewAlphabetVMCaller creates a new read-only instance of AlphabetVM, bound to a specific deployed contract.
func NewAlphabetVMCaller(address common.Address, caller bind.ContractCaller) (*AlphabetVMCaller, error) {
	contract, err := bindAlphabetVM(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AlphabetVMCaller{contract: contract}, nil
}

// NewAlphabetVMTransactor creates a new write-only instance of AlphabetVM, bound to a specific deployed contract.
func NewAlphabetVMTransactor(address common.Address, transactor bind.ContractTransactor) (*AlphabetVMTransactor, error) {
	contract, err := bindAlphabetVM(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AlphabetVMTransactor{contract: contract}, nil
}

// NewAlphabetVMFilterer creates a new log filterer instance of AlphabetVM, bound to a specific deployed contract.
func NewAlphabetVMFilterer(address common.Address, filterer bind.ContractFilterer) (*AlphabetVMFilterer, error) {
	contract, err := bindAlphabetVM(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AlphabetVMFilterer{contract: contract}, nil
}

// bindAlphabetVM binds a generic wrapper to an already deployed contract.
func bindAlphabetVM(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AlphabetVMABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AlphabetVM *AlphabetVMRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AlphabetVM.Contract.AlphabetVMCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AlphabetVM *AlphabetVMRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AlphabetVM.Contract.AlphabetVMTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AlphabetVM *AlphabetVMRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AlphabetVM.Contract.AlphabetVMTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AlphabetVM *AlphabetVMCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AlphabetVM.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AlphabetVM *AlphabetVMTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AlphabetVM.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AlphabetVM *AlphabetVMTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AlphabetVM.Contract.contract.Transact(opts, method, params...)
}

// Step is a free data retrieval call binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes _stateData, bytes ) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMCaller) Step(opts *bind.CallOpts, _stateData []byte, arg1 []byte) ([32]byte, error) {
	var out []interface{}
	err := _AlphabetVM.contract.Call(opts, &out, "step", _stateData, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Step is a free data retrieval call binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes _stateData, bytes ) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMSession) Step(_stateData []byte, arg1 []byte) ([32]byte, error) {
	return _AlphabetVM.Contract.Step(&_AlphabetVM.CallOpts, _stateData, arg1)
}

// Step is a free data retrieval call binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes _stateData, bytes ) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMCallerSession) Step(_stateData []byte, arg1 []byte) ([32]byte, error) {
	return _AlphabetVM.Contract.Step(&_AlphabetVM.CallOpts, _stateData, arg1)
}
