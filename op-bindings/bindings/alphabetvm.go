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
	ABI: "[{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_absolutePrestate\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"postState_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051610a73380380610a7383398101604081905261002f91610090565b608081905260405161004090610083565b604051809103906000f08015801561005c573d6000803e3d6000fd5b50600080546001600160a01b0319166001600160a01b0392909216919091179055506100a9565b6106c5806103ae83390190565b6000602082840312156100a257600080fd5b5051919050565b6080516102eb6100c3600039600060ad01526102eb6000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80637dc0d1d01461003b578063f8e0cb9614610085575b600080fd5b60005461005b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b6100986100933660046101a8565b6100a6565b60405190815260200161007c565b60008060007f000000000000000000000000000000000000000000000000000000000000000087876040516100dc929190610214565b60405180910390200361010057600091506100f986880188610224565b905061011f565b61010c8688018861023d565b90925090508161011b8161028e565b9250505b8161012b8260016102c6565b6040805160208101939093528201526060016040516020818303038152906040528051906020012092505050949350505050565b60008083601f84011261017157600080fd5b50813567ffffffffffffffff81111561018957600080fd5b6020830191508360208285010111156101a157600080fd5b9250929050565b600080600080604085870312156101be57600080fd5b843567ffffffffffffffff808211156101d657600080fd5b6101e28883890161015f565b909650945060208701359150808211156101fb57600080fd5b506102088782880161015f565b95989497509550505050565b8183823760009101908152919050565b60006020828403121561023657600080fd5b5035919050565b6000806040838503121561025057600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036102bf576102bf61025f565b5060010190565b600082198211156102d9576102d961025f565b50019056fea164736f6c634300080f000a608060405234801561001057600080fd5b506106a5806100206000396000f3fe608060405234801561001057600080fd5b506004361061007d5760003560e01c8063e03110e11161005b578063e03110e114610111578063e159261114610139578063fe4ac08e1461014e578063fef2b4ed146101c357600080fd5b806361238bde146100825780638542cf50146100c05780639a1f5e7f146100fe575b600080fd5b6100ad610090366004610551565b600160209081526000928352604080842090915290825290205481565b6040519081526020015b60405180910390f35b6100ee6100ce366004610551565b600260209081526000928352604080842090915290825290205460ff1681565b60405190151581526020016100b7565b6100ad61010c366004610573565b6101e3565b61012461011f366004610551565b6102b6565b604080519283526020830191909152016100b7565b61014c6101473660046105a5565b6103a7565b005b61014c61015c366004610573565b6000838152600260209081526040808320878452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660019081179091558684528252808320968352958152858220939093559283529082905291902055565b6100ad6101d1366004610621565b60006020819052908152604090205481565b60006101ee856104b0565b90506101fb836008610669565b8211806102085750602083115b1561023f576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000602081815260c085901b82526008959095528251828252600286526040808320858452875280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915584845287528083209483529386528382205581815293849052922055919050565b6000828152600260209081526040808320848452909152812054819060ff1661033f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f7072652d696d616765206d757374206578697374000000000000000000000000604482015260640160405180910390fd5b506000838152602081815260409091205461035b816008610669565b610366856020610669565b106103845783610377826008610669565b6103819190610681565b91505b506000938452600160209081526040808620948652939052919092205492909150565b604435600080600883018611156103c65763fe2549876000526004601cfd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001760008181526002602090815260408083208b8452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915584845282528083209a83529981528982209390935590815290819052959095209190915550505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82161761054b81600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b92915050565b6000806040838503121561056457600080fd5b50508035926020909101359150565b6000806000806080858703121561058957600080fd5b5050823594602084013594506040840135936060013592509050565b6000806000604084860312156105ba57600080fd5b83359250602084013567ffffffffffffffff808211156105d957600080fd5b818601915086601f8301126105ed57600080fd5b8135818111156105fc57600080fd5b87602082850101111561060e57600080fd5b6020830194508093505050509250925092565b60006020828403121561063357600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561067c5761067c61063a565b500190565b6000828210156106935761069361063a565b50039056fea164736f6c634300080f000a",
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

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM *AlphabetVMCaller) Oracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AlphabetVM.contract.Call(opts, &out, "oracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM *AlphabetVMSession) Oracle() (common.Address, error) {
	return _AlphabetVM.Contract.Oracle(&_AlphabetVM.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM *AlphabetVMCallerSession) Oracle() (common.Address, error) {
	return _AlphabetVM.Contract.Oracle(&_AlphabetVM.CallOpts)
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
