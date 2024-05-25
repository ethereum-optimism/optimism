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
	ABI: "[{\"type\":\"function\",\"name\":\"checkpoint\",\"inputs\":[],\"outputs\":[{\"name\":\"blockNumber_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"load\",\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"blockInfo_\",\"type\":\"tuple\",\"internalType\":\"structBlockOracle.BlockInfo\",\"components\":[{\"name\":\"hash\",\"type\":\"bytes32\",\"internalType\":\"Hash\"},{\"name\":\"childTimestamp\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Checkpoint\",\"inputs\":[{\"name\":\"blockNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"blockHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"Hash\"},{\"name\":\"childTimestamp\",\"type\":\"uint64\",\"indexed\":true,\"internalType\":\"Timestamp\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BlockHashNotPresent\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b506102e7806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806399d548aa14610098578063c2c4c5c1146100d0575b600080fd5b6100826040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b60405161008f9190610210565b60405180910390f35b6100ab6100a6366004610283565b6100e6565b604080518251815260209283015167ffffffffffffffff16928101929092520161008f565b6100d8610165565b60405190815260200161008f565b604080518082018252600080825260209182018190528381528082528281208351808501909452805480855260019091015467ffffffffffffffff169284019290925203610160576040517f37cf270500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b919050565b600061017260014361029c565b60408051808201825282408082524267ffffffffffffffff81811660208086018281526000898152918290528782209651875551600190960180547fffffffffffffffffffffffffffffffffffffffffffffffff000000000000000016969093169590951790915593519495509093909291849186917fb67ff58b33060fd371a35ae2d9f1c3cdaec9b8197969f6efe2594a1ff4ba68c691a4505090565b600060208083528351808285015260005b8181101561023d57858101830151858201604001528201610221565b8181111561024f576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60006020828403121561029557600080fd5b5035919050565b6000828210156102d5577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a",
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

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BlockOracle *BlockOracleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BlockOracle.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BlockOracle *BlockOracleSession) Version() (string, error) {
	return _BlockOracle.Contract.Version(&_BlockOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BlockOracle *BlockOracleCallerSession) Version() (string, error) {
	return _BlockOracle.Contract.Version(&_BlockOracle.CallOpts)
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

// BlockOracleCheckpointIterator is returned from FilterCheckpoint and is used to iterate over the raw logs and unpacked data for Checkpoint events raised by the BlockOracle contract.
type BlockOracleCheckpointIterator struct {
	Event *BlockOracleCheckpoint // Event containing the contract specifics and raw log

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
func (it *BlockOracleCheckpointIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BlockOracleCheckpoint)
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
		it.Event = new(BlockOracleCheckpoint)
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
func (it *BlockOracleCheckpointIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BlockOracleCheckpointIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BlockOracleCheckpoint represents a Checkpoint event raised by the BlockOracle contract.
type BlockOracleCheckpoint struct {
	BlockNumber    *big.Int
	BlockHash      [32]byte
	ChildTimestamp uint64
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterCheckpoint is a free log retrieval operation binding the contract event 0xb67ff58b33060fd371a35ae2d9f1c3cdaec9b8197969f6efe2594a1ff4ba68c6.
//
// Solidity: event Checkpoint(uint256 indexed blockNumber, bytes32 indexed blockHash, uint64 indexed childTimestamp)
func (_BlockOracle *BlockOracleFilterer) FilterCheckpoint(opts *bind.FilterOpts, blockNumber []*big.Int, blockHash [][32]byte, childTimestamp []uint64) (*BlockOracleCheckpointIterator, error) {

	var blockNumberRule []interface{}
	for _, blockNumberItem := range blockNumber {
		blockNumberRule = append(blockNumberRule, blockNumberItem)
	}
	var blockHashRule []interface{}
	for _, blockHashItem := range blockHash {
		blockHashRule = append(blockHashRule, blockHashItem)
	}
	var childTimestampRule []interface{}
	for _, childTimestampItem := range childTimestamp {
		childTimestampRule = append(childTimestampRule, childTimestampItem)
	}

	logs, sub, err := _BlockOracle.contract.FilterLogs(opts, "Checkpoint", blockNumberRule, blockHashRule, childTimestampRule)
	if err != nil {
		return nil, err
	}
	return &BlockOracleCheckpointIterator{contract: _BlockOracle.contract, event: "Checkpoint", logs: logs, sub: sub}, nil
}

// WatchCheckpoint is a free log subscription operation binding the contract event 0xb67ff58b33060fd371a35ae2d9f1c3cdaec9b8197969f6efe2594a1ff4ba68c6.
//
// Solidity: event Checkpoint(uint256 indexed blockNumber, bytes32 indexed blockHash, uint64 indexed childTimestamp)
func (_BlockOracle *BlockOracleFilterer) WatchCheckpoint(opts *bind.WatchOpts, sink chan<- *BlockOracleCheckpoint, blockNumber []*big.Int, blockHash [][32]byte, childTimestamp []uint64) (event.Subscription, error) {

	var blockNumberRule []interface{}
	for _, blockNumberItem := range blockNumber {
		blockNumberRule = append(blockNumberRule, blockNumberItem)
	}
	var blockHashRule []interface{}
	for _, blockHashItem := range blockHash {
		blockHashRule = append(blockHashRule, blockHashItem)
	}
	var childTimestampRule []interface{}
	for _, childTimestampItem := range childTimestamp {
		childTimestampRule = append(childTimestampRule, childTimestampItem)
	}

	logs, sub, err := _BlockOracle.contract.WatchLogs(opts, "Checkpoint", blockNumberRule, blockHashRule, childTimestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BlockOracleCheckpoint)
				if err := _BlockOracle.contract.UnpackLog(event, "Checkpoint", log); err != nil {
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

// ParseCheckpoint is a log parse operation binding the contract event 0xb67ff58b33060fd371a35ae2d9f1c3cdaec9b8197969f6efe2594a1ff4ba68c6.
//
// Solidity: event Checkpoint(uint256 indexed blockNumber, bytes32 indexed blockHash, uint64 indexed childTimestamp)
func (_BlockOracle *BlockOracleFilterer) ParseCheckpoint(log types.Log) (*BlockOracleCheckpoint, error) {
	event := new(BlockOracleCheckpoint)
	if err := _BlockOracle.contract.UnpackLog(event, "Checkpoint", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
