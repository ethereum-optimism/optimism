// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package deposit

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

// DepositMetaData contains all meta data concerning the Deposit contract.
var DepositMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"NonZeroCreationTarget\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"mint\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isCreation\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061056a806100206000396000f3fe60806040526004361061001e5760003560e01c8063fa92670c14610023575b600080fd5b61003d6004803603810190610038919061039d565b61003f565b005b8180156100795750600073ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff1614155b156100b0576040517ff98844ef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60003390503273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461010257731111000000000000000000000000000000001111330190505b8573ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad34888888886040516101679594939291906104da565b60405180910390a3505050505050565b6000604051905090565b600080fd5b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006101b68261018b565b9050919050565b6101c6816101ab565b81146101d157600080fd5b50565b6000813590506101e3816101bd565b92915050565b6000819050919050565b6101fc816101e9565b811461020757600080fd5b50565b600081359050610219816101f3565b92915050565b60008115159050919050565b6102348161021f565b811461023f57600080fd5b50565b6000813590506102518161022b565b92915050565b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6102aa82610261565b810181811067ffffffffffffffff821117156102c9576102c8610272565b5b80604052505050565b60006102dc610177565b90506102e882826102a1565b919050565b600067ffffffffffffffff82111561030857610307610272565b5b61031182610261565b9050602081019050919050565b82818337600083830152505050565b600061034061033b846102ed565b6102d2565b90508281526020810184848401111561035c5761035b61025c565b5b61036784828561031e565b509392505050565b600082601f83011261038457610383610257565b5b813561039484826020860161032d565b91505092915050565b600080600080600060a086880312156103b9576103b8610181565b5b60006103c7888289016101d4565b95505060206103d88882890161020a565b94505060406103e98882890161020a565b93505060606103fa88828901610242565b925050608086013567ffffffffffffffff81111561041b5761041a610186565b5b6104278882890161036f565b9150509295509295909350565b61043d816101e9565b82525050565b61044c8161021f565b82525050565b600081519050919050565b600082825260208201905092915050565b60005b8381101561048c578082015181840152602081019050610471565b8381111561049b576000848401525b50505050565b60006104ac82610452565b6104b6818561045d565b93506104c681856020860161046e565b6104cf81610261565b840191505092915050565b600060a0820190506104ef6000830188610434565b6104fc6020830187610434565b6105096040830186610434565b6105166060830185610443565b818103608083015261052881846104a1565b9050969550505050505056fea264697066735822122025140b7451be29e927d33c9ad0e2dd3744f824f592e16cb35f009b57442e2c9364736f6c634300080a0033",
}

// DepositABI is the input ABI used to generate the binding from.
// Deprecated: Use DepositMetaData.ABI instead.
var DepositABI = DepositMetaData.ABI

// DepositBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DepositMetaData.Bin instead.
var DepositBin = DepositMetaData.Bin

// DeployDeposit deploys a new Ethereum contract, binding an instance of Deposit to it.
func DeployDeposit(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Deposit, error) {
	parsed, err := DepositMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DepositBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Deposit{DepositCaller: DepositCaller{contract: contract}, DepositTransactor: DepositTransactor{contract: contract}, DepositFilterer: DepositFilterer{contract: contract}}, nil
}

// Deposit is an auto generated Go binding around an Ethereum contract.
type Deposit struct {
	DepositCaller     // Read-only binding to the contract
	DepositTransactor // Write-only binding to the contract
	DepositFilterer   // Log filterer for contract events
}

// DepositCaller is an auto generated read-only Go binding around an Ethereum contract.
type DepositCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DepositTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DepositFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DepositSession struct {
	Contract     *Deposit          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DepositCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DepositCallerSession struct {
	Contract *DepositCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// DepositTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DepositTransactorSession struct {
	Contract     *DepositTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// DepositRaw is an auto generated low-level Go binding around an Ethereum contract.
type DepositRaw struct {
	Contract *Deposit // Generic contract binding to access the raw methods on
}

// DepositCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DepositCallerRaw struct {
	Contract *DepositCaller // Generic read-only contract binding to access the raw methods on
}

// DepositTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DepositTransactorRaw struct {
	Contract *DepositTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDeposit creates a new instance of Deposit, bound to a specific deployed contract.
func NewDeposit(address common.Address, backend bind.ContractBackend) (*Deposit, error) {
	contract, err := bindDeposit(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Deposit{DepositCaller: DepositCaller{contract: contract}, DepositTransactor: DepositTransactor{contract: contract}, DepositFilterer: DepositFilterer{contract: contract}}, nil
}

// NewDepositCaller creates a new read-only instance of Deposit, bound to a specific deployed contract.
func NewDepositCaller(address common.Address, caller bind.ContractCaller) (*DepositCaller, error) {
	contract, err := bindDeposit(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DepositCaller{contract: contract}, nil
}

// NewDepositTransactor creates a new write-only instance of Deposit, bound to a specific deployed contract.
func NewDepositTransactor(address common.Address, transactor bind.ContractTransactor) (*DepositTransactor, error) {
	contract, err := bindDeposit(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DepositTransactor{contract: contract}, nil
}

// NewDepositFilterer creates a new log filterer instance of Deposit, bound to a specific deployed contract.
func NewDepositFilterer(address common.Address, filterer bind.ContractFilterer) (*DepositFilterer, error) {
	contract, err := bindDeposit(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DepositFilterer{contract: contract}, nil
}

// bindDeposit binds a generic wrapper to an already deployed contract.
func bindDeposit(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Deposit *DepositRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Deposit.Contract.DepositCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Deposit *DepositRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Deposit.Contract.DepositTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Deposit *DepositRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Deposit.Contract.DepositTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Deposit *DepositCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Deposit.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Deposit *DepositTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Deposit.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Deposit *DepositTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Deposit.Contract.contract.Transact(opts, method, params...)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_Deposit *DepositTransactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _Deposit.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_Deposit *DepositSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _Deposit.Contract.DepositTransaction(&_Deposit.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_Deposit *DepositTransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _Deposit.Contract.DepositTransaction(&_Deposit.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the Deposit contract.
type DepositTransactionDepositedIterator struct {
	Event *DepositTransactionDeposited // Event containing the contract specifics and raw log

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
func (it *DepositTransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositTransactionDeposited)
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
		it.Event = new(DepositTransactionDeposited)
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
func (it *DepositTransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositTransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositTransactionDeposited represents a TransactionDeposited event raised by the Deposit contract.
type DepositTransactionDeposited struct {
	From       common.Address
	To         common.Address
	Mint       *big.Int
	Value      *big.Int
	GasLimit   *big.Int
	IsCreation bool
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_Deposit *DepositFilterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*DepositTransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Deposit.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &DepositTransactionDepositedIterator{contract: _Deposit.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_Deposit *DepositFilterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *DepositTransactionDeposited, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Deposit.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositTransactionDeposited)
				if err := _Deposit.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
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

// ParseTransactionDeposited is a log parse operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_Deposit *DepositFilterer) ParseTransactionDeposited(log types.Log) (*DepositTransactionDeposited, error) {
	event := new(DepositTransactionDeposited)
	if err := _Deposit.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
