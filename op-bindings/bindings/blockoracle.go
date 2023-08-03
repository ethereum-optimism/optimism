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

// BlockOracleBlockInfo is an auto generated low-level Go binding around an user-defined struct.
type BlockOracleBlockInfo struct {
	Hash           [32]byte
	ChildTimestamp uint64
}

// BlockOracleMetaData contains all meta data concerning the BlockOracle contract.
var BlockOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"BlockHashNotPresent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"checkpoint\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"load\",\"outputs\":[{\"components\":[{\"internalType\":\"Hash\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"Timestamp\",\"name\":\"childTimestamp\",\"type\":\"uint64\"}],\"internalType\":\"structBlockOracle.BlockInfo\",\"name\":\"blockInfo_\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506101ef806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806399d548aa1461003b578063c2c4c5c114610078575b600080fd5b61004e61004936600461018b565b61008e565b604080518251815260209283015167ffffffffffffffff1692810192909252015b60405180910390f35b61008061010d565b60405190815260200161006f565b604080518082018252600080825260209182018190528381528082528281208351808501909452805480855260019091015467ffffffffffffffff169284019290925203610108576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b6000604051806040016040528060014361012791906101a4565b804082524267ffffffffffffffff908116602093840152600082815280845260409020845181559390920151600190930180547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001693909216929092179055919050565b60006020828403121561019d57600080fd5b5035919050565b6000828210156101dd577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a",
}

// BlockOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use BlockOracleMetaData.ABI instead.
var BlockOracleABI = BlockOracleMetaData.ABI

// BlockOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BlockOracleMetaData.Bin instead.
var BlockOracleBin = BlockOracleMetaData.Bin

// DeployBlockOracle deploys a new Ethereum contract, binding an instance of BlockOracle to it.
func DeployBlockOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BlockOracle, error) {
	parsed, err := BlockOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BlockOracleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BlockOracle{BlockOracleCaller: BlockOracleCaller{contract: contract}, BlockOracleTransactor: BlockOracleTransactor{contract: contract}, BlockOracleFilterer: BlockOracleFilterer{contract: contract}}, nil
}

// BlockOracle is an auto generated Go binding around an Ethereum contract.
type BlockOracle struct {
	BlockOracleCaller     // Read-only binding to the contract
	BlockOracleTransactor // Write-only binding to the contract
	BlockOracleFilterer   // Log filterer for contract events
}

// BlockOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type BlockOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BlockOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BlockOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BlockOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BlockOracleSession struct {
	Contract     *BlockOracle      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BlockOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BlockOracleCallerSession struct {
	Contract *BlockOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// BlockOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BlockOracleTransactorSession struct {
	Contract     *BlockOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// BlockOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type BlockOracleRaw struct {
	Contract *BlockOracle // Generic contract binding to access the raw methods on
}

// BlockOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BlockOracleCallerRaw struct {
	Contract *BlockOracleCaller // Generic read-only contract binding to access the raw methods on
}

// BlockOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BlockOracleTransactorRaw struct {
	Contract *BlockOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBlockOracle creates a new instance of BlockOracle, bound to a specific deployed contract.
func NewBlockOracle(address common.Address, backend bind.ContractBackend) (*BlockOracle, error) {
	contract, err := bindBlockOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BlockOracle{BlockOracleCaller: BlockOracleCaller{contract: contract}, BlockOracleTransactor: BlockOracleTransactor{contract: contract}, BlockOracleFilterer: BlockOracleFilterer{contract: contract}}, nil
}

// NewBlockOracleCaller creates a new read-only instance of BlockOracle, bound to a specific deployed contract.
func NewBlockOracleCaller(address common.Address, caller bind.ContractCaller) (*BlockOracleCaller, error) {
	contract, err := bindBlockOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BlockOracleCaller{contract: contract}, nil
}

// NewBlockOracleTransactor creates a new write-only instance of BlockOracle, bound to a specific deployed contract.
func NewBlockOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*BlockOracleTransactor, error) {
	contract, err := bindBlockOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BlockOracleTransactor{contract: contract}, nil
}

// NewBlockOracleFilterer creates a new log filterer instance of BlockOracle, bound to a specific deployed contract.
func NewBlockOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*BlockOracleFilterer, error) {
	contract, err := bindBlockOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BlockOracleFilterer{contract: contract}, nil
}

// bindBlockOracle binds a generic wrapper to an already deployed contract.
func bindBlockOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BlockOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockOracle *BlockOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockOracle.Contract.BlockOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockOracle *BlockOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOracle.Contract.BlockOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockOracle *BlockOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockOracle.Contract.BlockOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BlockOracle *BlockOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BlockOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BlockOracle *BlockOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BlockOracle *BlockOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BlockOracle.Contract.contract.Transact(opts, method, params...)
}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns((bytes32,uint64) blockInfo_)
func (_BlockOracle *BlockOracleCaller) Load(opts *bind.CallOpts, _blockNumber *big.Int) (BlockOracleBlockInfo, error) {
	var out []interface{}
	err := _BlockOracle.contract.Call(opts, &out, "load", _blockNumber)

	if err != nil {
		return *new(BlockOracleBlockInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(BlockOracleBlockInfo)).(*BlockOracleBlockInfo)

	return out0, err

}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns((bytes32,uint64) blockInfo_)
func (_BlockOracle *BlockOracleSession) Load(_blockNumber *big.Int) (BlockOracleBlockInfo, error) {
	return _BlockOracle.Contract.Load(&_BlockOracle.CallOpts, _blockNumber)
}

// Load is a free data retrieval call binding the contract method 0x99d548aa.
//
// Solidity: function load(uint256 _blockNumber) view returns((bytes32,uint64) blockInfo_)
func (_BlockOracle *BlockOracleCallerSession) Load(_blockNumber *big.Int) (BlockOracleBlockInfo, error) {
	return _BlockOracle.Contract.Load(&_BlockOracle.CallOpts, _blockNumber)
}

// Checkpoint is a paid mutator transaction binding the contract method 0xc2c4c5c1.
//
// Solidity: function checkpoint() returns(uint256 blockNumber_)
func (_BlockOracle *BlockOracleTransactor) Checkpoint(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BlockOracle.contract.Transact(opts, "checkpoint")
}

// Checkpoint is a paid mutator transaction binding the contract method 0xc2c4c5c1.
//
// Solidity: function checkpoint() returns(uint256 blockNumber_)
func (_BlockOracle *BlockOracleSession) Checkpoint() (*types.Transaction, error) {
	return _BlockOracle.Contract.Checkpoint(&_BlockOracle.TransactOpts)
}

// Checkpoint is a paid mutator transaction binding the contract method 0xc2c4c5c1.
//
// Solidity: function checkpoint() returns(uint256 blockNumber_)
func (_BlockOracle *BlockOracleTransactorSession) Checkpoint() (*types.Transaction, error) {
	return _BlockOracle.Contract.Checkpoint(&_BlockOracle.TransactOpts)
}
