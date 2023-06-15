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

// LegacyMessagePasserMetaData contains all meta data concerning the LegacyMessagePasser contract.
var LegacyMessagePasserMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"passMessageToL1\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"sentMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b5060016080819052600060a081905260c081905280610698610048823960006101050152600060dc0152600060b301526106986000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806382e3702d14610064578063cafa81dc14610097575b600080fd5b61004e6100ac565b60405161005b9190610347565b60405180910390f35b610087610072366004610398565b60006020819052908152604090205460ff1681565b604051901515815260200161005b565b6100aa6100a53660046103e0565b61014f565b005b60606100d77f00000000000000000000000000000000000000000000000000000000000000006101da565b6101007f00000000000000000000000000000000000000000000000000000000000000006101da565b6101297f00000000000000000000000000000000000000000000000000000000000000006101da565b60405160200161013b939291906104af565b604051602081830303815290604052905090565b60016000808333604051602001610167929190610525565b604080518083037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe001815291815281516020928301208352908201929092520160002080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001691151591909117905550565b60608160000361021d57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b811561024757806102318161059e565b91506102409050600a83610605565b9150610221565b60008167ffffffffffffffff811115610262576102626103b1565b6040519080825280601f01601f19166020018201604052801561028c576020820181803683370190505b5090505b841561030f576102a1600183610619565b91506102ae600a86610630565b6102b9906030610644565b60f81b8183815181106102ce576102ce61065c565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610308600a86610605565b9450610290565b949350505050565b60005b8381101561033257818101518382015260200161031a565b83811115610341576000848401525b50505050565b6020815260008251806020840152610366816040850160208701610317565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000602082840312156103aa57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000602082840312156103f257600080fd5b813567ffffffffffffffff8082111561040a57600080fd5b818401915084601f83011261041e57600080fd5b813581811115610430576104306103b1565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610476576104766103b1565b8160405282815287602084870101111561048f57600080fd5b826020860160208301376000928101602001929092525095945050505050565b600084516104c1818460208901610317565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516104fd816001850160208a01610317565b60019201918201528351610518816002840160208801610317565b0160020195945050505050565b60008351610537818460208801610317565b60609390931b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169190920190815260140192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036105cf576105cf61056f565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610614576106146105d6565b500490565b60008282101561062b5761062b61056f565b500390565b60008261063f5761063f6105d6565b500690565b600082198211156106575761065761056f565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// LegacyMessagePasserABI is the input ABI used to generate the binding from.
// Deprecated: Use LegacyMessagePasserMetaData.ABI instead.
var LegacyMessagePasserABI = LegacyMessagePasserMetaData.ABI

// LegacyMessagePasserBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LegacyMessagePasserMetaData.Bin instead.
var LegacyMessagePasserBin = LegacyMessagePasserMetaData.Bin

// DeployLegacyMessagePasser deploys a new Ethereum contract, binding an instance of LegacyMessagePasser to it.
func DeployLegacyMessagePasser(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LegacyMessagePasser, error) {
	parsed, err := LegacyMessagePasserMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LegacyMessagePasserBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LegacyMessagePasser{LegacyMessagePasserCaller: LegacyMessagePasserCaller{contract: contract}, LegacyMessagePasserTransactor: LegacyMessagePasserTransactor{contract: contract}, LegacyMessagePasserFilterer: LegacyMessagePasserFilterer{contract: contract}}, nil
}

// LegacyMessagePasser is an auto generated Go binding around an Ethereum contract.
type LegacyMessagePasser struct {
	LegacyMessagePasserCaller     // Read-only binding to the contract
	LegacyMessagePasserTransactor // Write-only binding to the contract
	LegacyMessagePasserFilterer   // Log filterer for contract events
}

// LegacyMessagePasserCaller is an auto generated read-only Go binding around an Ethereum contract.
type LegacyMessagePasserCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyMessagePasserTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LegacyMessagePasserTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyMessagePasserFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LegacyMessagePasserFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LegacyMessagePasserSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LegacyMessagePasserSession struct {
	Contract     *LegacyMessagePasser // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// LegacyMessagePasserCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LegacyMessagePasserCallerSession struct {
	Contract *LegacyMessagePasserCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// LegacyMessagePasserTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LegacyMessagePasserTransactorSession struct {
	Contract     *LegacyMessagePasserTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// LegacyMessagePasserRaw is an auto generated low-level Go binding around an Ethereum contract.
type LegacyMessagePasserRaw struct {
	Contract *LegacyMessagePasser // Generic contract binding to access the raw methods on
}

// LegacyMessagePasserCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LegacyMessagePasserCallerRaw struct {
	Contract *LegacyMessagePasserCaller // Generic read-only contract binding to access the raw methods on
}

// LegacyMessagePasserTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LegacyMessagePasserTransactorRaw struct {
	Contract *LegacyMessagePasserTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLegacyMessagePasser creates a new instance of LegacyMessagePasser, bound to a specific deployed contract.
func NewLegacyMessagePasser(address common.Address, backend bind.ContractBackend) (*LegacyMessagePasser, error) {
	contract, err := bindLegacyMessagePasser(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LegacyMessagePasser{LegacyMessagePasserCaller: LegacyMessagePasserCaller{contract: contract}, LegacyMessagePasserTransactor: LegacyMessagePasserTransactor{contract: contract}, LegacyMessagePasserFilterer: LegacyMessagePasserFilterer{contract: contract}}, nil
}

// NewLegacyMessagePasserCaller creates a new read-only instance of LegacyMessagePasser, bound to a specific deployed contract.
func NewLegacyMessagePasserCaller(address common.Address, caller bind.ContractCaller) (*LegacyMessagePasserCaller, error) {
	contract, err := bindLegacyMessagePasser(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LegacyMessagePasserCaller{contract: contract}, nil
}

// NewLegacyMessagePasserTransactor creates a new write-only instance of LegacyMessagePasser, bound to a specific deployed contract.
func NewLegacyMessagePasserTransactor(address common.Address, transactor bind.ContractTransactor) (*LegacyMessagePasserTransactor, error) {
	contract, err := bindLegacyMessagePasser(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LegacyMessagePasserTransactor{contract: contract}, nil
}

// NewLegacyMessagePasserFilterer creates a new log filterer instance of LegacyMessagePasser, bound to a specific deployed contract.
func NewLegacyMessagePasserFilterer(address common.Address, filterer bind.ContractFilterer) (*LegacyMessagePasserFilterer, error) {
	contract, err := bindLegacyMessagePasser(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LegacyMessagePasserFilterer{contract: contract}, nil
}

// bindLegacyMessagePasser binds a generic wrapper to an already deployed contract.
func bindLegacyMessagePasser(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LegacyMessagePasserMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LegacyMessagePasser *LegacyMessagePasserRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LegacyMessagePasser.Contract.LegacyMessagePasserCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LegacyMessagePasser *LegacyMessagePasserRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.LegacyMessagePasserTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LegacyMessagePasser *LegacyMessagePasserRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.LegacyMessagePasserTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LegacyMessagePasser *LegacyMessagePasserCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LegacyMessagePasser.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LegacyMessagePasser *LegacyMessagePasserTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LegacyMessagePasser *LegacyMessagePasserTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.contract.Transact(opts, method, params...)
}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_LegacyMessagePasser *LegacyMessagePasserCaller) SentMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _LegacyMessagePasser.contract.Call(opts, &out, "sentMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_LegacyMessagePasser *LegacyMessagePasserSession) SentMessages(arg0 [32]byte) (bool, error) {
	return _LegacyMessagePasser.Contract.SentMessages(&_LegacyMessagePasser.CallOpts, arg0)
}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_LegacyMessagePasser *LegacyMessagePasserCallerSession) SentMessages(arg0 [32]byte) (bool, error) {
	return _LegacyMessagePasser.Contract.SentMessages(&_LegacyMessagePasser.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LegacyMessagePasser *LegacyMessagePasserCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _LegacyMessagePasser.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LegacyMessagePasser *LegacyMessagePasserSession) Version() (string, error) {
	return _LegacyMessagePasser.Contract.Version(&_LegacyMessagePasser.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LegacyMessagePasser *LegacyMessagePasserCallerSession) Version() (string, error) {
	return _LegacyMessagePasser.Contract.Version(&_LegacyMessagePasser.CallOpts)
}

// PassMessageToL1 is a paid mutator transaction binding the contract method 0xcafa81dc.
//
// Solidity: function passMessageToL1(bytes _message) returns()
func (_LegacyMessagePasser *LegacyMessagePasserTransactor) PassMessageToL1(opts *bind.TransactOpts, _message []byte) (*types.Transaction, error) {
	return _LegacyMessagePasser.contract.Transact(opts, "passMessageToL1", _message)
}

// PassMessageToL1 is a paid mutator transaction binding the contract method 0xcafa81dc.
//
// Solidity: function passMessageToL1(bytes _message) returns()
func (_LegacyMessagePasser *LegacyMessagePasserSession) PassMessageToL1(_message []byte) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.PassMessageToL1(&_LegacyMessagePasser.TransactOpts, _message)
}

// PassMessageToL1 is a paid mutator transaction binding the contract method 0xcafa81dc.
//
// Solidity: function passMessageToL1(bytes _message) returns()
func (_LegacyMessagePasser *LegacyMessagePasserTransactorSession) PassMessageToL1(_message []byte) (*types.Transaction, error) {
	return _LegacyMessagePasser.Contract.PassMessageToL1(&_LegacyMessagePasser.TransactOpts, _message)
}
