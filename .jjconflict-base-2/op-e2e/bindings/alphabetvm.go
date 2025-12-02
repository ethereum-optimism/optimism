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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_oracle\",\"type\":\"address\",\"internalType\":\"contractPreimageOracle\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"oracle\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"step\",\"inputs\":[{\"name\":\"_stateData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"postState_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x60a060405234801561001057600080fd5b506040516105d23803806105d283398101604081905261002f91610059565b608091909152600080546001600160a01b0319166001600160a01b03909216919091179055610096565b6000806040838503121561006c57600080fd5b825160208401519092506001600160a01b038116811461008b57600080fd5b809150509250929050565b6080516105226100b0600039600060af01526105226000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80637dc0d1d01461003b578063e14ced3214610085575b600080fd5b60005461005b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b61009861009336600461039c565b6100a6565b60405190815260200161007c565b600080600060087f0000000000000000000000000000000000000000000000000000000000000000901b600889896040516100e2929190610410565b6040518091039020901b036101e0576000805473ffffffffffffffffffffffffffffffffffffffff1663e03110e161011b6004886102a6565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b1681526004810191909152600060248201526044016040805180830381865afa158015610175573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906101999190610420565b50905060006101b3600163ffffffff608085901c16610473565b600481901b9450905060006101ca8a8c018c61048a565b90506101d685826104a3565b935050505061020d565b6101ec878901896104bb565b9092509050816101fb816104dd565b9250508080610209906104dd565b9150505b6040805160208101849052908101829052606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905280516020909101207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001798975050505050505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83161761034c818360408051600093845233602052918152606090922091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b9392505050565b60008083601f84011261036557600080fd5b50813567ffffffffffffffff81111561037d57600080fd5b60208301915083602082850101111561039557600080fd5b9250929050565b6000806000806000606086880312156103b457600080fd5b853567ffffffffffffffff808211156103cc57600080fd5b6103d889838a01610353565b909750955060208801359150808211156103f157600080fd5b506103fe88828901610353565b96999598509660400135949350505050565b8183823760009101908152919050565b6000806040838503121561043357600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008282101561048557610485610444565b500390565b60006020828403121561049c57600080fd5b5035919050565b600082198211156104b6576104b6610444565b500190565b600080604083850312156104ce57600080fd5b50508035926020909101359150565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361050e5761050e610444565b506001019056fea164736f6c634300080f000a",
}

// AlphabetVMABI is the input ABI used to generate the binding from.
// Deprecated: Use AlphabetVMMetaData.ABI instead.
var AlphabetVMABI = AlphabetVMMetaData.ABI

// AlphabetVMBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AlphabetVMMetaData.Bin instead.
var AlphabetVMBin = AlphabetVMMetaData.Bin

// DeployAlphabetVM deploys a new Ethereum contract, binding an instance of AlphabetVM to it.
func DeployAlphabetVM(auth *bind.TransactOpts, backend bind.ContractBackend, _absolutePrestate [32]byte, _oracle common.Address) (common.Address, *types.Transaction, *AlphabetVM, error) {
	parsed, err := AlphabetVMMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AlphabetVMBin), backend, _absolutePrestate, _oracle)
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

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMCaller) Step(opts *bind.CallOpts, _stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _AlphabetVM.contract.Call(opts, &out, "step", _stateData, arg1, _localContext)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMSession) Step(_stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	return _AlphabetVM.Contract.Step(&_AlphabetVM.CallOpts, _stateData, arg1, _localContext)
}

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM *AlphabetVMCallerSession) Step(_stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	return _AlphabetVM.Contract.Step(&_AlphabetVM.CallOpts, _stateData, arg1, _localContext)
}
