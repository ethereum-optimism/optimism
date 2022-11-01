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

// L1BlockMetaData contains all meta data concerning the L1Block contract.
var L1BlockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEPOSITOR_ACCOUNT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"basefee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"hash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"number\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequenceNumber\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_number\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"_timestamp\",\"type\":\"uint64\"},{\"internalType\":\"uint256\",\"name\":\"_basefee\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_sequenceNumber\",\"type\":\"uint64\"}],\"name\":\"setL1BlockValues\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"timestamp\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b506000608081905260a052600160c05260805160a05160c05161074c61004f6000396000610302015260006102d9015260006102b0015261074c6000f3fe608060405234801561001057600080fd5b50600436106100885760003560e01c806364ca23ef1161005b57806364ca23ef146100dc5780638381f58a14610109578063b80777ea1461011d578063e591b2821461013d57600080fd5b8063042c2f571461008d57806309bd5a60146100a257806354fd4d50146100be5780635cf24969146100d3575b600080fd5b6100a061009b3660046104a6565b61017d565b005b6100ab60025481565b6040519081526020015b60405180910390f35b6100c66102a9565b6040516100b5919061052d565b6100ab60015481565b6003546100f09067ffffffffffffffff1681565b60405167ffffffffffffffff90911681526020016100b5565b6000546100f09067ffffffffffffffff1681565b6000546100f09068010000000000000000900467ffffffffffffffff1681565b61015873deaddeaddeaddeaddeaddeaddeaddeaddead000181565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100b5565b3373deaddeaddeaddeaddeaddeaddeaddeaddead000114610224576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f4c31426c6f636b3a206f6e6c7920746865206465706f7369746f72206163636f60448201527f756e742063616e20736574204c3120626c6f636b2076616c7565730000000000606482015260840160405180910390fd5b6000805467ffffffffffffffff9687167fffffffffffffffffffffffffffffffff0000000000000000000000000000000090911617680100000000000000009587169590950294909417909355600191909155600255600380547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001691909216179055565b60606102d47f000000000000000000000000000000000000000000000000000000000000000061034c565b6102fd7f000000000000000000000000000000000000000000000000000000000000000061034c565b6103267f000000000000000000000000000000000000000000000000000000000000000061034c565b6040516020016103389392919061057e565b604051602081830303815290604052905090565b60608160000361038f57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156103b957806103a381610623565b91506103b29050600a8361068a565b9150610393565b60008167ffffffffffffffff8111156103d4576103d461069e565b6040519080825280601f01601f1916602001820160405280156103fe576020820181803683370190505b5090505b8415610481576104136001836106cd565b9150610420600a866106e4565b61042b9060306106f8565b60f81b81838151811061044057610440610710565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535061047a600a8661068a565b9450610402565b949350505050565b803567ffffffffffffffff811681146104a157600080fd5b919050565b600080600080600060a086880312156104be57600080fd5b6104c786610489565b94506104d560208701610489565b935060408601359250606086013591506104f160808701610489565b90509295509295909350565b60005b83811015610518578181015183820152602001610500565b83811115610527576000848401525b50505050565b602081526000825180602084015261054c8160408501602087016104fd565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b600084516105908184602089016104fd565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516105cc816001850160208a016104fd565b600192019182015283516105e78160028401602088016104fd565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610654576106546105f4565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826106995761069961065b565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156106df576106df6105f4565b500390565b6000826106f3576106f361065b565b500690565b6000821982111561070b5761070b6105f4565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// L1BlockABI is the input ABI used to generate the binding from.
// Deprecated: Use L1BlockMetaData.ABI instead.
var L1BlockABI = L1BlockMetaData.ABI

// L1BlockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1BlockMetaData.Bin instead.
var L1BlockBin = L1BlockMetaData.Bin

// DeployL1Block deploys a new Ethereum contract, binding an instance of L1Block to it.
func DeployL1Block(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1Block, error) {
	parsed, err := L1BlockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1BlockBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1Block{L1BlockCaller: L1BlockCaller{contract: contract}, L1BlockTransactor: L1BlockTransactor{contract: contract}, L1BlockFilterer: L1BlockFilterer{contract: contract}}, nil
}

// L1Block is an auto generated Go binding around an Ethereum contract.
type L1Block struct {
	L1BlockCaller     // Read-only binding to the contract
	L1BlockTransactor // Write-only binding to the contract
	L1BlockFilterer   // Log filterer for contract events
}

// L1BlockCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1BlockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1BlockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1BlockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1BlockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1BlockSession struct {
	Contract     *L1Block          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1BlockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1BlockCallerSession struct {
	Contract *L1BlockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// L1BlockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1BlockTransactorSession struct {
	Contract     *L1BlockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// L1BlockRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1BlockRaw struct {
	Contract *L1Block // Generic contract binding to access the raw methods on
}

// L1BlockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1BlockCallerRaw struct {
	Contract *L1BlockCaller // Generic read-only contract binding to access the raw methods on
}

// L1BlockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1BlockTransactorRaw struct {
	Contract *L1BlockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1Block creates a new instance of L1Block, bound to a specific deployed contract.
func NewL1Block(address common.Address, backend bind.ContractBackend) (*L1Block, error) {
	contract, err := bindL1Block(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1Block{L1BlockCaller: L1BlockCaller{contract: contract}, L1BlockTransactor: L1BlockTransactor{contract: contract}, L1BlockFilterer: L1BlockFilterer{contract: contract}}, nil
}

// NewL1BlockCaller creates a new read-only instance of L1Block, bound to a specific deployed contract.
func NewL1BlockCaller(address common.Address, caller bind.ContractCaller) (*L1BlockCaller, error) {
	contract, err := bindL1Block(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockCaller{contract: contract}, nil
}

// NewL1BlockTransactor creates a new write-only instance of L1Block, bound to a specific deployed contract.
func NewL1BlockTransactor(address common.Address, transactor bind.ContractTransactor) (*L1BlockTransactor, error) {
	contract, err := bindL1Block(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1BlockTransactor{contract: contract}, nil
}

// NewL1BlockFilterer creates a new log filterer instance of L1Block, bound to a specific deployed contract.
func NewL1BlockFilterer(address common.Address, filterer bind.ContractFilterer) (*L1BlockFilterer, error) {
	contract, err := bindL1Block(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1BlockFilterer{contract: contract}, nil
}

// bindL1Block binds a generic wrapper to an already deployed contract.
func bindL1Block(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1BlockABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Block *L1BlockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Block.Contract.L1BlockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Block *L1BlockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Block.Contract.L1BlockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Block *L1BlockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Block.Contract.L1BlockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1Block *L1BlockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1Block.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1Block *L1BlockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1Block.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1Block *L1BlockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1Block.Contract.contract.Transact(opts, method, params...)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockCaller) DEPOSITORACCOUNT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "DEPOSITOR_ACCOUNT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1Block.Contract.DEPOSITORACCOUNT(&_L1Block.CallOpts)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_L1Block *L1BlockCallerSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _L1Block.Contract.DEPOSITORACCOUNT(&_L1Block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockCaller) Basefee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "basefee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockSession) Basefee() (*big.Int, error) {
	return _L1Block.Contract.Basefee(&_L1Block.CallOpts)
}

// Basefee is a free data retrieval call binding the contract method 0x5cf24969.
//
// Solidity: function basefee() view returns(uint256)
func (_L1Block *L1BlockCallerSession) Basefee() (*big.Int, error) {
	return _L1Block.Contract.Basefee(&_L1Block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockCaller) Hash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "hash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockSession) Hash() ([32]byte, error) {
	return _L1Block.Contract.Hash(&_L1Block.CallOpts)
}

// Hash is a free data retrieval call binding the contract method 0x09bd5a60.
//
// Solidity: function hash() view returns(bytes32)
func (_L1Block *L1BlockCallerSession) Hash() ([32]byte, error) {
	return _L1Block.Contract.Hash(&_L1Block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockCaller) Number(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "number")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockSession) Number() (uint64, error) {
	return _L1Block.Contract.Number(&_L1Block.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint64)
func (_L1Block *L1BlockCallerSession) Number() (uint64, error) {
	return _L1Block.Contract.Number(&_L1Block.CallOpts)
}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockCaller) SequenceNumber(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "sequenceNumber")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockSession) SequenceNumber() (uint64, error) {
	return _L1Block.Contract.SequenceNumber(&_L1Block.CallOpts)
}

// SequenceNumber is a free data retrieval call binding the contract method 0x64ca23ef.
//
// Solidity: function sequenceNumber() view returns(uint64)
func (_L1Block *L1BlockCallerSession) SequenceNumber() (uint64, error) {
	return _L1Block.Contract.SequenceNumber(&_L1Block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockCaller) Timestamp(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "timestamp")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockSession) Timestamp() (uint64, error) {
	return _L1Block.Contract.Timestamp(&_L1Block.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint64)
func (_L1Block *L1BlockCallerSession) Timestamp() (uint64, error) {
	return _L1Block.Contract.Timestamp(&_L1Block.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1Block.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockSession) Version() (string, error) {
	return _L1Block.Contract.Version(&_L1Block.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1Block *L1BlockCallerSession) Version() (string, error) {
	return _L1Block.Contract.Version(&_L1Block.CallOpts)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x042c2f57.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber) returns()
func (_L1Block *L1BlockTransactor) SetL1BlockValues(opts *bind.TransactOpts, _number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64) (*types.Transaction, error) {
	return _L1Block.contract.Transact(opts, "setL1BlockValues", _number, _timestamp, _basefee, _hash, _sequenceNumber)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x042c2f57.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber) returns()
func (_L1Block *L1BlockSession) SetL1BlockValues(_number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64) (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValues(&_L1Block.TransactOpts, _number, _timestamp, _basefee, _hash, _sequenceNumber)
}

// SetL1BlockValues is a paid mutator transaction binding the contract method 0x042c2f57.
//
// Solidity: function setL1BlockValues(uint64 _number, uint64 _timestamp, uint256 _basefee, bytes32 _hash, uint64 _sequenceNumber) returns()
func (_L1Block *L1BlockTransactorSession) SetL1BlockValues(_number uint64, _timestamp uint64, _basefee *big.Int, _hash [32]byte, _sequenceNumber uint64) (*types.Transaction, error) {
	return _L1Block.Contract.SetL1BlockValues(&_L1Block.TransactOpts, _number, _timestamp, _basefee, _hash, _sequenceNumber)
}
