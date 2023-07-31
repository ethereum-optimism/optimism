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

// BlockHashOracleMetaData contains all meta data concerning the BlockHashOracle contract.
var BlockHashOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"BlockHashNotPresent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BlockNumberOOB\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"load\",\"outputs\":[{\"internalType\":\"Hash\",\"name\":\"blockHash_\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"store\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610138806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80636057361d1461003b57806399d548aa14610050575b600080fd5b61004e610049366004610112565b610075565b005b61006361005e366004610112565b6100c4565b60405190815260200160405180910390f35b804060008190036100b2576040517fd82756d800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60009182526020829052604090912055565b6000818152602081905260408120549081900361010d576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b60006020828403121561012457600080fd5b503591905056fea164736f6c634300080f000a",
}

// BlockHashOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use BlockHashOracleMetaData.ABI instead.
var BlockHashOracleABI = BlockHashOracleMetaData.ABI

// BlockHashOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlockHashOracleMetaData.Bin instead.
var BlockHashOracleBin = BlockHashOracleMetaData.Bin

// DeployBlockHashOracle deploys a new Ethereum contract, binding an instance of BlockHashOracle to it.
func DeployBlockHashOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockHashOracle, error) {
	parsed, err := BlockHashOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlockHashOracleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockHashOracle{BlockHashOracleCaller: BlockHashOracleCaller{contract: contract}, BlockHashOracleTransactor: BlockHashOracleTransactor{contract: contract}, BlockHashOracleFilterer: BlockHashOracleFilterer{contract: contract}}, nil
}

// BlockHashOracle is an auto generated Go binding around an Ethereum contract.
type BlockHashOracle struct {
	BlockHashOracleCaller     // Read-only binding to the contract
	BlockHashOracleTransactor // Write-only binding to the contract
	BlockHashOracleFilterer   // Log filterer for contract events
}

// BlockHashOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockHashOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockHashOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockHashOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockHashOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockHashOracleSession struct {
	Contract     *BlockHashOracle  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockHashOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockHashOracleCallerSession struct {
	Contract *BlockHashOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// BlockHashOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockHashOracleTransactorSession struct {
	Contract     *BlockHashOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// BlockHashOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockHashOracleRaw struct {
	Contract *BlockHashOracle // Generic contract binding to access the raw methods on
}

// BlockHashOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockHashOracleCallerRaw struct {
	Contract *BlockHashOracleCaller // Generic read-only contract binding to access the raw methods on
}

// BlockHashOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockHashOracleTransactorRaw struct {
	Contract *BlockHashOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockHashOracle creates a new instance of BlockHashOracle, bound to a specific deployed contract.
func NewBlockHashOracle(address common.Address, backend bind.ContractBackend) (*BlockHashOracle, error) {
	contract, err := bindBlockHashOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockHashOracle{BlockHashOracleCaller: BlockHashOracleCaller{contract: contract}, BlockHashOracleTransactor: BlockHashOracleTransactor{contract: contract}, BlockHashOracleFilterer: BlockHashOracleFilterer{contract: contract}}, nil
}

// NewBlockHashOracleCaller creates a new read-only instance of BlockHashOracle, bound to a specific deployed contract.
func NewBlockHashOracleCaller(address common.Address, caller bind.ContractCaller) (*BlockHashOracleCaller, error) {
	contract, err := bindBlockHashOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockHashOracleCaller{contract: contract}, nil
}

// NewBlockHashOracleTransactor creates a new write-only instance of BlockHashOracle, bound to a specific deployed contract.
func NewBlockHashOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockHashOracleTransactor, error) {
	contract, err := bindBlockHashOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockHashOracleTransactor{contract: contract}, nil
}

// NewBlockHashOracleFilterer creates a new log filterer instance of BlockHashOracle, bound to a specific deployed contract.
func NewBlockHashOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockHashOracleFilterer, error) {
	contract, err := bindBlockHashOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockHashOracleFilterer{contract: contract}, nil
}

// bindBlockHashOracle binds a generic wrapper to an already deployed contract.
func bindBlockHashOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BlockHashOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockHashOracle *BlockHashOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockHashOracle.Contract.BlockHashOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockHashOracle *BlockHashOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.BlockHashOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockHashOracle *BlockHashOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.BlockHashOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockHashOracle *BlockHashOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockHashOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockHashOracle *BlockHashOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockHashOracle *BlockHashOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.contract.Transact(opts, method, params...)
}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns(bytes32 blockHash_)
func (_BlockHashOracle *BlockHashOracleCaller) Load(opts *bind.CallOpts, _blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _BlockHashOracle.contract.Call(opts, &out, "load", _blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns(bytes32 blockHash_)
func (_BlockHashOracle *BlockHashOracleSession) Load(_blockNumber *big.Int) ([32]byte, error) {
	return _BlockHashOracle.Contract.Load(&_BlockHashOracle.CallOpts, _blockNumber)
}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns(bytes32 blockHash_)
func (_BlockHashOracle *BlockHashOracleCallerSession) Load(_blockNumber *big.Int) ([32]byte, error) {
	return _BlockHashOracle.Contract.Load(&_BlockHashOracle.CallOpts, _blockNumber)
}

// Store is a paid mutator transaction binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 _blockNumber) returns()
func (_BlockHashOracle *BlockHashOracleTransactor) Store(opts *bind.TransactOpts, _blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockHashOracle.contract.Transact(opts, "store", _blockNumber)
}

// Store is a paid mutator transaction binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 _blockNumber) returns()
func (_BlockHashOracle *BlockHashOracleSession) Store(_blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.Store(&_BlockHashOracle.TransactOpts, _blockNumber)
}

// Store is a paid mutator transaction binding the contract method 0x6057361d.
//
// Solidity: function store(uint256 _blockNumber) returns()
func (_BlockHashOracle *BlockHashOracleTransactorSession) Store(_blockNumber *big.Int) (*types.Transaction, error) {
	return _BlockHashOracle.Contract.Store(&_BlockHashOracle.TransactOpts, _blockNumber)
}
