// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package invoker

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

// InvokerMetaData contains all meta data concerning the Invoker contract.
var InvokerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"invokePrecompile\",\"inputs\":[{\"name\":\"_precompile\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_input\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"PrecompileInvoked\",\"inputs\":[{\"name\":\"precompile\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"result\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"delegateCallResult\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"PrecompileCallFailed\",\"inputs\":[]}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103bf8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c8063051f3bdf1461002d575b5f5ffd5b61004061003b366004610208565b610042565b005b5f5f8373ffffffffffffffffffffffffffffffffffffffff16836040516100699190610323565b5f604051808303815f865af19150503d805f81146100a2576040519150601f19603f3d011682016040523d82523d5f602084013e6100a7565b606091505b5091509150816100e3576040517ffd23ff6400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60608473ffffffffffffffffffffffffffffffffffffffff168460405161010a9190610323565b5f60405180830381855af49150503d805f8114610142576040519150601f19603f3d011682016040523d82523d5f602084013e610147565b606091505b50909350905082610184576040517ffd23ff6400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8473ffffffffffffffffffffffffffffffffffffffff167fc331673664ab9732fd2c0b1a4aa0cd948da43af82aca20906b4c306c7228e07983836040516101cc929190610385565b60405180910390a25050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b5f5f60408385031215610219575f5ffd5b823573ffffffffffffffffffffffffffffffffffffffff8116811461023c575f5ffd5b9150602083013567ffffffffffffffff811115610257575f5ffd5b8301601f81018513610267575f5ffd5b803567ffffffffffffffff811115610281576102816101db565b6040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0603f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8501160116810181811067ffffffffffffffff821117156102ed576102ed6101db565b604052818152828201602001871015610304575f5ffd5b816020840160208301375f602083830101528093505050509250929050565b5f82518060208501845e5f920191825250919050565b5f81518084528060208401602086015e5f6020828601015260207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f83011685010191505092915050565b604081525f6103976040830185610339565b82810360208401526103a98185610339565b9594505050505056fea164736f6c634300081c000a",
}

// InvokerABI is the input ABI used to generate the binding from.
// Deprecated: Use InvokerMetaData.ABI instead.
var InvokerABI = InvokerMetaData.ABI

// InvokerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InvokerMetaData.Bin instead.
var InvokerBin = InvokerMetaData.Bin

// DeployInvoker deploys a new Ethereum contract, binding an instance of Invoker to it.
func DeployInvoker(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Invoker, error) {
	parsed, err := InvokerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InvokerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Invoker{InvokerCaller: InvokerCaller{contract: contract}, InvokerTransactor: InvokerTransactor{contract: contract}, InvokerFilterer: InvokerFilterer{contract: contract}}, nil
}

// Invoker is an auto generated Go binding around an Ethereum contract.
type Invoker struct {
	InvokerCaller     // Read-only binding to the contract
	InvokerTransactor // Write-only binding to the contract
	InvokerFilterer   // Log filterer for contract events
}

// InvokerCaller is an auto generated read-only Go binding around an Ethereum contract.
type InvokerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvokerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InvokerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvokerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InvokerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InvokerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InvokerSession struct {
	Contract     *Invoker          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InvokerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InvokerCallerSession struct {
	Contract *InvokerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// InvokerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InvokerTransactorSession struct {
	Contract     *InvokerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// InvokerRaw is an auto generated low-level Go binding around an Ethereum contract.
type InvokerRaw struct {
	Contract *Invoker // Generic contract binding to access the raw methods on
}

// InvokerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InvokerCallerRaw struct {
	Contract *InvokerCaller // Generic read-only contract binding to access the raw methods on
}

// InvokerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InvokerTransactorRaw struct {
	Contract *InvokerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInvoker creates a new instance of Invoker, bound to a specific deployed contract.
func NewInvoker(address common.Address, backend bind.ContractBackend) (*Invoker, error) {
	contract, err := bindInvoker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Invoker{InvokerCaller: InvokerCaller{contract: contract}, InvokerTransactor: InvokerTransactor{contract: contract}, InvokerFilterer: InvokerFilterer{contract: contract}}, nil
}

// NewInvokerCaller creates a new read-only instance of Invoker, bound to a specific deployed contract.
func NewInvokerCaller(address common.Address, caller bind.ContractCaller) (*InvokerCaller, error) {
	contract, err := bindInvoker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InvokerCaller{contract: contract}, nil
}

// NewInvokerTransactor creates a new write-only instance of Invoker, bound to a specific deployed contract.
func NewInvokerTransactor(address common.Address, transactor bind.ContractTransactor) (*InvokerTransactor, error) {
	contract, err := bindInvoker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InvokerTransactor{contract: contract}, nil
}

// NewInvokerFilterer creates a new log filterer instance of Invoker, bound to a specific deployed contract.
func NewInvokerFilterer(address common.Address, filterer bind.ContractFilterer) (*InvokerFilterer, error) {
	contract, err := bindInvoker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InvokerFilterer{contract: contract}, nil
}

// bindInvoker binds a generic wrapper to an already deployed contract.
func bindInvoker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InvokerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Invoker *InvokerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Invoker.Contract.InvokerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Invoker *InvokerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invoker.Contract.InvokerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Invoker *InvokerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Invoker.Contract.InvokerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Invoker *InvokerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Invoker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Invoker *InvokerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Invoker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Invoker *InvokerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Invoker.Contract.contract.Transact(opts, method, params...)
}

// InvokePrecompile is a paid mutator transaction binding the contract method 0x051f3bdf.
//
// Solidity: function invokePrecompile(address _precompile, bytes _input) returns()
func (_Invoker *InvokerTransactor) InvokePrecompile(opts *bind.TransactOpts, _precompile common.Address, _input []byte) (*types.Transaction, error) {
	return _Invoker.contract.Transact(opts, "invokePrecompile", _precompile, _input)
}

// InvokePrecompile is a paid mutator transaction binding the contract method 0x051f3bdf.
//
// Solidity: function invokePrecompile(address _precompile, bytes _input) returns()
func (_Invoker *InvokerSession) InvokePrecompile(_precompile common.Address, _input []byte) (*types.Transaction, error) {
	return _Invoker.Contract.InvokePrecompile(&_Invoker.TransactOpts, _precompile, _input)
}

// InvokePrecompile is a paid mutator transaction binding the contract method 0x051f3bdf.
//
// Solidity: function invokePrecompile(address _precompile, bytes _input) returns()
func (_Invoker *InvokerTransactorSession) InvokePrecompile(_precompile common.Address, _input []byte) (*types.Transaction, error) {
	return _Invoker.Contract.InvokePrecompile(&_Invoker.TransactOpts, _precompile, _input)
}

// InvokerPrecompileInvokedIterator is returned from FilterPrecompileInvoked and is used to iterate over the raw logs and unpacked data for PrecompileInvoked events raised by the Invoker contract.
type InvokerPrecompileInvokedIterator struct {
	Event *InvokerPrecompileInvoked // Event containing the contract specifics and raw log

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
func (it *InvokerPrecompileInvokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InvokerPrecompileInvoked)
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
		it.Event = new(InvokerPrecompileInvoked)
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
func (it *InvokerPrecompileInvokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InvokerPrecompileInvokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InvokerPrecompileInvoked represents a PrecompileInvoked event raised by the Invoker contract.
type InvokerPrecompileInvoked struct {
	Precompile         common.Address
	Result             []byte
	DelegateCallResult []byte
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterPrecompileInvoked is a free log retrieval operation binding the contract event 0xc331673664ab9732fd2c0b1a4aa0cd948da43af82aca20906b4c306c7228e079.
//
// Solidity: event PrecompileInvoked(address indexed precompile, bytes result, bytes delegateCallResult)
func (_Invoker *InvokerFilterer) FilterPrecompileInvoked(opts *bind.FilterOpts, precompile []common.Address) (*InvokerPrecompileInvokedIterator, error) {

	var precompileRule []interface{}
	for _, precompileItem := range precompile {
		precompileRule = append(precompileRule, precompileItem)
	}

	logs, sub, err := _Invoker.contract.FilterLogs(opts, "PrecompileInvoked", precompileRule)
	if err != nil {
		return nil, err
	}
	return &InvokerPrecompileInvokedIterator{contract: _Invoker.contract, event: "PrecompileInvoked", logs: logs, sub: sub}, nil
}

// WatchPrecompileInvoked is a free log subscription operation binding the contract event 0xc331673664ab9732fd2c0b1a4aa0cd948da43af82aca20906b4c306c7228e079.
//
// Solidity: event PrecompileInvoked(address indexed precompile, bytes result, bytes delegateCallResult)
func (_Invoker *InvokerFilterer) WatchPrecompileInvoked(opts *bind.WatchOpts, sink chan<- *InvokerPrecompileInvoked, precompile []common.Address) (event.Subscription, error) {

	var precompileRule []interface{}
	for _, precompileItem := range precompile {
		precompileRule = append(precompileRule, precompileItem)
	}

	logs, sub, err := _Invoker.contract.WatchLogs(opts, "PrecompileInvoked", precompileRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InvokerPrecompileInvoked)
				if err := _Invoker.contract.UnpackLog(event, "PrecompileInvoked", log); err != nil {
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

// ParsePrecompileInvoked is a log parse operation binding the contract event 0xc331673664ab9732fd2c0b1a4aa0cd948da43af82aca20906b4c306c7228e079.
//
// Solidity: event PrecompileInvoked(address indexed precompile, bytes result, bytes delegateCallResult)
func (_Invoker *InvokerFilterer) ParsePrecompileInvoked(log types.Log) (*InvokerPrecompileInvoked, error) {
	event := new(InvokerPrecompileInvoked)
	if err := _Invoker.contract.UnpackLog(event, "PrecompileInvoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
