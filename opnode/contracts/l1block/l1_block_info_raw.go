// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package l1block

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

// L1blockMetaData contains all meta data concerning the L1block contract.
var L1blockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"OnlyDepositor\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DEPOSITOR_ACCOUNT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"basefee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"hash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"number\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_number\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_basefee\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"}],\"name\":\"setL1BlockValues\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506103a2806100206000396000f3fe608060405234801561001057600080fd5b50600436106100625760003560e01c806309bd5a60146100675780635cf24969146100855780638381f58a146100a3578063b80777ea146100c1578063c03ba43e146100df578063e591b282146100fb575b600080fd5b61006f610119565b60405161007c91906101fd565b60405180910390f35b61008d61011f565b60405161009a9190610231565b60405180910390f35b6100ab610125565b6040516100b89190610231565b60405180910390f35b6100c961012b565b6040516100d69190610231565b60405180910390f35b6100f960048036038101906100f491906102a9565b610131565b005b6101036101cc565b6040516101109190610351565b60405180910390f35b60035481565b60025481565b60005481565b60015481565b73deaddeaddeaddeaddeaddeaddeaddeaddead000173ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101aa576040517fce8c104800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8360008190555082600181905550816002819055508060038190555050505050565b73deaddeaddeaddeaddeaddeaddeaddeaddead000181565b6000819050919050565b6101f7816101e4565b82525050565b600060208201905061021260008301846101ee565b92915050565b6000819050919050565b61022b81610218565b82525050565b60006020820190506102466000830184610222565b92915050565b600080fd5b61025a81610218565b811461026557600080fd5b50565b60008135905061027781610251565b92915050565b610286816101e4565b811461029157600080fd5b50565b6000813590506102a38161027d565b92915050565b600080600080608085870312156102c3576102c261024c565b5b60006102d187828801610268565b94505060206102e287828801610268565b93505060406102f387828801610268565b925050606061030487828801610294565b91505092959194509250565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b600061033b82610310565b9050919050565b61034b81610330565b82525050565b60006020820190506103666000830184610342565b9291505056fea2646970667358221220cdaea4f1ee477c3fe7f1b3caa2283ad58d53d6a652006ec9e6f6bf22354b258264736f6c634300080a0033",
}

// L1blockABI is the input ABI used to generate the binding from.
// Deprecated: Use L1blockMetaData.ABI instead.
var L1blockABI = L1blockMetaData.ABI

// L1blockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1blockMetaData.Bin instead.
var L1blockBin = L1blockMetaData.Bin

// DeployL1block deploys a new Ethereum contract, binding an instance of L1block to it.
func DeployL1block(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1block, error) {
	parsed, err := L1blockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1blockBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1block{L1blockCaller: L1blockCaller{contract: contract}, L1blockTransactor: L1blockTransactor{contract: contract}, L1blockFilterer: L1blockFilterer{contract: contract}}, nil
}

// L1block is an auto generated Go binding around an Ethereum contract.
type L1block struct {
	L1blockCaller     // Read-only binding to the contract
	L1blockTransactor // Write-only binding to the contract
	L1blockFilterer   // Log filterer for contract events
}

// L1blockCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1blockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1blockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1blockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1blockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1blockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1blockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1blockSession struct {
	Contract     *L1block          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1blockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1blockCallerSession struct {
	Contract *L1blockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// L1blockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1blockTransactorSession struct {
	Contract     *L1blockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// L1blockRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1blockRaw struct {
	Contract *L1block // Generic contract binding to access the raw methods on
}

// L1blockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1blockCallerRaw struct {
	Contract *L1blockCaller // Generic read-only contract binding to access the raw methods on
}

// L1blockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1blockTransactorRaw struct {
	Contract *L1blockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1block creates a new instance of L1block, bound to a specific deployed contract.
func NewL1block(address common.Address, backend bind.ContractBackend) (*L1block, error) {
	contract, err := bindL1block(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1block{L1blockCaller: L1blockCaller{contract: contract}, L1blockTransactor: L1blockTransactor{contract: contract}, L1blockFilterer: L1blockFilterer{contract: contract}}, nil
}

// NewL1blockCaller creates a new read-only instance of L1block, bound to a specific deployed contract.
func NewL1blockCaller(address common.Address, caller bind.ContractCaller) (*L1blockCaller, error) {
	contract, err := bindL1block(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1blockCaller{contract: contract}, nil
}

// NewL1blockTransactor creates a new write-only instance of L1block, bound to a specific deployed contract.
func NewL1blockTransactor(address common.Address, transactor bind.ContractTransactor) (*L1blockTransactor, error) {
	contract, err := bindL1block(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1blockTransactor{contract: contract}, nil
}

// NewL1blockFilterer creates a new log filterer instance of L1block, bound to a specific deployed contract.
func NewL1blockFilterer(address common.Address, filterer bind.ContractFilterer) (*L1blockFilterer, error) {
	contract, err := bindL1block(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1blockFilterer{contract: contract}, nil
}

// bindL1block binds a generic wrapper to an already deployed contract.
func bindL1block(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1blockABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1block *L1blockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1block.Contract.L1blockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1block *L1blockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1block.Contract.L1blockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1block *L1blockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1block.Contract.L1blockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1block *L1blockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1block.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1block *L1blockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1block.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1block *L1blockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1block.Contract.contract.Transact(opts, method, params...)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1block *L1blockCaller) DEPOSITORACCOUNT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1block.contract.Call(opts, &out, "DEPOSITOR_ACCOUNT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1block *L1blockSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1block.Contract.DEPOSITORACCOUNT(&_L1block.CallOpts)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1block *L1blockCallerSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1block.Contract.DEPOSITORACCOUNT(&_L1block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1block *L1blockCaller) Basefee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1block.contract.Call(opts, &out, "basefee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1block *L1blockSession) Basefee() (*big.Int, error) {
	return _L1block.Contract.Basefee(&_L1block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1block *L1blockCallerSession) Basefee() (*big.Int, error) {
	return _L1block.Contract.Basefee(&_L1block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1block *L1blockCaller) Hash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1block.contract.Call(opts, &out, "hash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1block *L1blockSession) Hash() ([32]byte, error) {
	return _L1block.Contract.Hash(&_L1block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1block *L1blockCallerSession) Hash() ([32]byte, error) {
	return _L1block.Contract.Hash(&_L1block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_L1block *L1blockCaller) Number(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1block.contract.Call(opts, &out, "number")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_L1block *L1blockSession) Number() (*big.Int, error) {
	return _L1block.Contract.Number(&_L1block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_L1block *L1blockCallerSession) Number() (*big.Int, error) {
	return _L1block.Contract.Number(&_L1block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256)
func (_L1block *L1blockCaller) Timestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1block.contract.Call(opts, &out, "timestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256)
func (_L1block *L1blockSession) Timestamp() (*big.Int, error) {
	return _L1block.Contract.Timestamp(&_L1block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256)
func (_L1block *L1blockCallerSession) Timestamp() (*big.Int, error) {
	return _L1block.Contract.Timestamp(&_L1block.CallOpts)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0xc03ba43e.
//
// Solidity: function setL1BlockValues(uint256 _number, uint256 _timestamp, uint256 _basefee, bytes32 _hash) returns()
func (_L1block *L1blockTransactor) SetL1BlockValues(opts *bind.TransactOpts, _number *big.Int, _timestamp *big.Int, _basefee *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _L1block.contract.Transact(opts, "setL1BlockValues", _number, _timestamp, _basefee, _hash)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0xc03ba43e.
//
// Solidity: function setL1BlockValues(uint256 _number, uint256 _timestamp, uint256 _basefee, bytes32 _hash) returns()
func (_L1block *L1blockSession) SetL1BlockValues(_number *big.Int, _timestamp *big.Int, _basefee *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _L1block.Contract.SetL1BlockValues(&_L1block.TransactOpts, _number, _timestamp, _basefee, _hash)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0xc03ba43e.
//
// Solidity: function setL1BlockValues(uint256 _number, uint256 _timestamp, uint256 _basefee, bytes32 _hash) returns()
func (_L1block *L1blockTransactorSession) SetL1BlockValues(_number *big.Int, _timestamp *big.Int, _basefee *big.Int, _hash [32]byte) (*types.Transaction, error) {
	return _L1block.Contract.SetL1BlockValues(&_L1block.TransactOpts, _number, _timestamp, _basefee, _hash)
}
