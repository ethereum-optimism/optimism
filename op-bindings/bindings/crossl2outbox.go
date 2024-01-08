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

// CrossL2OutboxMetaData contains all meta data concerning the CrossL2Outbox contract.
var CrossL2OutboxMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"messageRoot\",\"type\":\"bytes32\"}],\"name\":\"MessagePassed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"WithdrawerBalanceBurnt\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"initiateMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061062c806100206000396000f3fe6080604052600436106100345760003560e01c806344df8e701461003957806354fd4d50146100505780637c9582f8146100af575b600080fd5b34801561004557600080fd5b5061004e6100c2565b005b34801561005c57600080fd5b506100996040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b6040516100a69190610489565b60405180910390f35b61004e6100bd3660046104d2565b6100fa565b476100cc8161024f565b60405181907f7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f90600090a250565b6040805161010081018252600080547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff16825246602083015291810186905233606082015273ffffffffffffffffffffffffffffffffffffffff851660808201523460a082015260c0810184905260e081018390526101769061027e565b60005460405191925073ffffffffffffffffffffffffffffffffffffffff86169133917dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff16907fffc1d53e4f99588c5f34fd266ca3b55eaa206b5e91235bc4e0c5247486f90c31906101ef908a9034908a908a908a906105e0565b60405180910390a45050600080547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff000000000000000000000000000000000000000000000000000000000000909116179055505050565b8060405161025c90610412565b6040518091039082f0905080158015610279573d6000803e3d6000fd5b505050565b60e0810151805160209182018190206040805193840182905283019190915260009182906060016040516020818303038152906040528051906020012090506000846060015185608001518660a001518760c00151604051602001610317949392919073ffffffffffffffffffffffffffffffffffffffff94851681529290931660208301526040820152606081019190915260800190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152828252805160209182012081840152828201949094528051808303820181526060830182528051908501208785015197820151608084019890985260a0808401989098528151808403909801885260c08301825287519785019790972060e0830152610100808301979097528051808303909701875261012090910190525083519301929092207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001792915050565b60088061061883390190565b6000815180845260005b8181101561044457602081850181015186830182015201610428565b81811115610456576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061049c602083018461041e565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080600080608085870312156104e857600080fd5b84359350602085013573ffffffffffffffffffffffffffffffffffffffff8116811461051357600080fd5b925060408501359150606085013567ffffffffffffffff8082111561053757600080fd5b818701915087601f83011261054b57600080fd5b81358181111561055d5761055d6104a3565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156105a3576105a36104a3565b816040528281528a60208487010111156105bc57600080fd5b82602086016020830137600060208483010152809550505050505092959194509250565b85815284602082015283604082015260a06060820152600061060560a083018561041e565b9050826080830152969550505050505056fe608060405230fffea164736f6c634300080f000a",
}

// CrossL2OutboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2OutboxMetaData.ABI instead.
var CrossL2OutboxABI = CrossL2OutboxMetaData.ABI

// CrossL2OutboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2OutboxMetaData.Bin instead.
var CrossL2OutboxBin = CrossL2OutboxMetaData.Bin

// DeployCrossL2Outbox deploys a new Ethereum contract, binding an instance of CrossL2Outbox to it.
func DeployCrossL2Outbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CrossL2Outbox, error) {
	parsed, err := CrossL2OutboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2OutboxBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CrossL2Outbox{CrossL2OutboxCaller: CrossL2OutboxCaller{contract: contract}, CrossL2OutboxTransactor: CrossL2OutboxTransactor{contract: contract}, CrossL2OutboxFilterer: CrossL2OutboxFilterer{contract: contract}}, nil
}

// CrossL2Outbox is an auto generated Go binding around an Ethereum contract.
type CrossL2Outbox struct {
	CrossL2OutboxCaller     // Read-only binding to the contract
	CrossL2OutboxTransactor // Write-only binding to the contract
	CrossL2OutboxFilterer   // Log filterer for contract events
}

// CrossL2OutboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrossL2OutboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2OutboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrossL2OutboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2OutboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrossL2OutboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2OutboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrossL2OutboxSession struct {
	Contract     *CrossL2Outbox    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CrossL2OutboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrossL2OutboxCallerSession struct {
	Contract *CrossL2OutboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// CrossL2OutboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrossL2OutboxTransactorSession struct {
	Contract     *CrossL2OutboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// CrossL2OutboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrossL2OutboxRaw struct {
	Contract *CrossL2Outbox // Generic contract binding to access the raw methods on
}

// CrossL2OutboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrossL2OutboxCallerRaw struct {
	Contract *CrossL2OutboxCaller // Generic read-only contract binding to access the raw methods on
}

// CrossL2OutboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrossL2OutboxTransactorRaw struct {
	Contract *CrossL2OutboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrossL2Outbox creates a new instance of CrossL2Outbox, bound to a specific deployed contract.
func NewCrossL2Outbox(address common.Address, backend bind.ContractBackend) (*CrossL2Outbox, error) {
	contract, err := bindCrossL2Outbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CrossL2Outbox{CrossL2OutboxCaller: CrossL2OutboxCaller{contract: contract}, CrossL2OutboxTransactor: CrossL2OutboxTransactor{contract: contract}, CrossL2OutboxFilterer: CrossL2OutboxFilterer{contract: contract}}, nil
}

// NewCrossL2OutboxCaller creates a new read-only instance of CrossL2Outbox, bound to a specific deployed contract.
func NewCrossL2OutboxCaller(address common.Address, caller bind.ContractCaller) (*CrossL2OutboxCaller, error) {
	contract, err := bindCrossL2Outbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2OutboxCaller{contract: contract}, nil
}

// NewCrossL2OutboxTransactor creates a new write-only instance of CrossL2Outbox, bound to a specific deployed contract.
func NewCrossL2OutboxTransactor(address common.Address, transactor bind.ContractTransactor) (*CrossL2OutboxTransactor, error) {
	contract, err := bindCrossL2Outbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2OutboxTransactor{contract: contract}, nil
}

// NewCrossL2OutboxFilterer creates a new log filterer instance of CrossL2Outbox, bound to a specific deployed contract.
func NewCrossL2OutboxFilterer(address common.Address, filterer bind.ContractFilterer) (*CrossL2OutboxFilterer, error) {
	contract, err := bindCrossL2Outbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrossL2OutboxFilterer{contract: contract}, nil
}

// bindCrossL2Outbox binds a generic wrapper to an already deployed contract.
func bindCrossL2Outbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrossL2OutboxABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Outbox *CrossL2OutboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Outbox.Contract.CrossL2OutboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Outbox *CrossL2OutboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.CrossL2OutboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Outbox *CrossL2OutboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.CrossL2OutboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Outbox *CrossL2OutboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Outbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Outbox *CrossL2OutboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Outbox *CrossL2OutboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.contract.Transact(opts, method, params...)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Outbox *CrossL2OutboxCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CrossL2Outbox.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Outbox *CrossL2OutboxSession) Version() (string, error) {
	return _CrossL2Outbox.Contract.Version(&_CrossL2Outbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Outbox *CrossL2OutboxCallerSession) Version() (string, error) {
	return _CrossL2Outbox.Contract.Version(&_CrossL2Outbox.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_CrossL2Outbox *CrossL2OutboxTransactor) Burn(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Outbox.contract.Transact(opts, "burn")
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_CrossL2Outbox *CrossL2OutboxSession) Burn() (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.Burn(&_CrossL2Outbox.TransactOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_CrossL2Outbox *CrossL2OutboxTransactorSession) Burn() (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.Burn(&_CrossL2Outbox.TransactOpts)
}

// InitiateMessage is a paid mutator transaction binding the contract method 0x7c9582f8.
//
// Solidity: function initiateMessage(bytes32 _targetChain, address _to, uint256 _gasLimit, bytes _data) payable returns()
func (_CrossL2Outbox *CrossL2OutboxTransactor) InitiateMessage(opts *bind.TransactOpts, _targetChain [32]byte, _to common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CrossL2Outbox.contract.Transact(opts, "initiateMessage", _targetChain, _to, _gasLimit, _data)
}

// InitiateMessage is a paid mutator transaction binding the contract method 0x7c9582f8.
//
// Solidity: function initiateMessage(bytes32 _targetChain, address _to, uint256 _gasLimit, bytes _data) payable returns()
func (_CrossL2Outbox *CrossL2OutboxSession) InitiateMessage(_targetChain [32]byte, _to common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.InitiateMessage(&_CrossL2Outbox.TransactOpts, _targetChain, _to, _gasLimit, _data)
}

// InitiateMessage is a paid mutator transaction binding the contract method 0x7c9582f8.
//
// Solidity: function initiateMessage(bytes32 _targetChain, address _to, uint256 _gasLimit, bytes _data) payable returns()
func (_CrossL2Outbox *CrossL2OutboxTransactorSession) InitiateMessage(_targetChain [32]byte, _to common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CrossL2Outbox.Contract.InitiateMessage(&_CrossL2Outbox.TransactOpts, _targetChain, _to, _gasLimit, _data)
}

// CrossL2OutboxMessagePassedIterator is returned from FilterMessagePassed and is used to iterate over the raw logs and unpacked data for MessagePassed events raised by the CrossL2Outbox contract.
type CrossL2OutboxMessagePassedIterator struct {
	Event *CrossL2OutboxMessagePassed // Event containing the contract specifics and raw log

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
func (it *CrossL2OutboxMessagePassedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossL2OutboxMessagePassed)
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
		it.Event = new(CrossL2OutboxMessagePassed)
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
func (it *CrossL2OutboxMessagePassedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossL2OutboxMessagePassedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossL2OutboxMessagePassed represents a MessagePassed event raised by the CrossL2Outbox contract.
type CrossL2OutboxMessagePassed struct {
	Nonce       *big.Int
	From        common.Address
	To          common.Address
	TargetChain [32]byte
	Value       *big.Int
	GasLimit    *big.Int
	Data        []byte
	MessageRoot [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMessagePassed is a free log retrieval operation binding the contract event 0xffc1d53e4f99588c5f34fd266ca3b55eaa206b5e91235bc4e0c5247486f90c31.
//
// Solidity: event MessagePassed(uint256 indexed nonce, address indexed from, address indexed to, bytes32 targetChain, uint256 value, uint256 gasLimit, bytes data, bytes32 messageRoot)
func (_CrossL2Outbox *CrossL2OutboxFilterer) FilterMessagePassed(opts *bind.FilterOpts, nonce []*big.Int, from []common.Address, to []common.Address) (*CrossL2OutboxMessagePassedIterator, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CrossL2Outbox.contract.FilterLogs(opts, "MessagePassed", nonceRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &CrossL2OutboxMessagePassedIterator{contract: _CrossL2Outbox.contract, event: "MessagePassed", logs: logs, sub: sub}, nil
}

// WatchMessagePassed is a free log subscription operation binding the contract event 0xffc1d53e4f99588c5f34fd266ca3b55eaa206b5e91235bc4e0c5247486f90c31.
//
// Solidity: event MessagePassed(uint256 indexed nonce, address indexed from, address indexed to, bytes32 targetChain, uint256 value, uint256 gasLimit, bytes data, bytes32 messageRoot)
func (_CrossL2Outbox *CrossL2OutboxFilterer) WatchMessagePassed(opts *bind.WatchOpts, sink chan<- *CrossL2OutboxMessagePassed, nonce []*big.Int, from []common.Address, to []common.Address) (event.Subscription, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _CrossL2Outbox.contract.WatchLogs(opts, "MessagePassed", nonceRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossL2OutboxMessagePassed)
				if err := _CrossL2Outbox.contract.UnpackLog(event, "MessagePassed", log); err != nil {
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

// ParseMessagePassed is a log parse operation binding the contract event 0xffc1d53e4f99588c5f34fd266ca3b55eaa206b5e91235bc4e0c5247486f90c31.
//
// Solidity: event MessagePassed(uint256 indexed nonce, address indexed from, address indexed to, bytes32 targetChain, uint256 value, uint256 gasLimit, bytes data, bytes32 messageRoot)
func (_CrossL2Outbox *CrossL2OutboxFilterer) ParseMessagePassed(log types.Log) (*CrossL2OutboxMessagePassed, error) {
	event := new(CrossL2OutboxMessagePassed)
	if err := _CrossL2Outbox.contract.UnpackLog(event, "MessagePassed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CrossL2OutboxWithdrawerBalanceBurntIterator is returned from FilterWithdrawerBalanceBurnt and is used to iterate over the raw logs and unpacked data for WithdrawerBalanceBurnt events raised by the CrossL2Outbox contract.
type CrossL2OutboxWithdrawerBalanceBurntIterator struct {
	Event *CrossL2OutboxWithdrawerBalanceBurnt // Event containing the contract specifics and raw log

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
func (it *CrossL2OutboxWithdrawerBalanceBurntIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossL2OutboxWithdrawerBalanceBurnt)
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
		it.Event = new(CrossL2OutboxWithdrawerBalanceBurnt)
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
func (it *CrossL2OutboxWithdrawerBalanceBurntIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossL2OutboxWithdrawerBalanceBurntIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossL2OutboxWithdrawerBalanceBurnt represents a WithdrawerBalanceBurnt event raised by the CrossL2Outbox contract.
type CrossL2OutboxWithdrawerBalanceBurnt struct {
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWithdrawerBalanceBurnt is a free log retrieval operation binding the contract event 0x7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f.
//
// Solidity: event WithdrawerBalanceBurnt(uint256 indexed amount)
func (_CrossL2Outbox *CrossL2OutboxFilterer) FilterWithdrawerBalanceBurnt(opts *bind.FilterOpts, amount []*big.Int) (*CrossL2OutboxWithdrawerBalanceBurntIterator, error) {

	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _CrossL2Outbox.contract.FilterLogs(opts, "WithdrawerBalanceBurnt", amountRule)
	if err != nil {
		return nil, err
	}
	return &CrossL2OutboxWithdrawerBalanceBurntIterator{contract: _CrossL2Outbox.contract, event: "WithdrawerBalanceBurnt", logs: logs, sub: sub}, nil
}

// WatchWithdrawerBalanceBurnt is a free log subscription operation binding the contract event 0x7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f.
//
// Solidity: event WithdrawerBalanceBurnt(uint256 indexed amount)
func (_CrossL2Outbox *CrossL2OutboxFilterer) WatchWithdrawerBalanceBurnt(opts *bind.WatchOpts, sink chan<- *CrossL2OutboxWithdrawerBalanceBurnt, amount []*big.Int) (event.Subscription, error) {

	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _CrossL2Outbox.contract.WatchLogs(opts, "WithdrawerBalanceBurnt", amountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossL2OutboxWithdrawerBalanceBurnt)
				if err := _CrossL2Outbox.contract.UnpackLog(event, "WithdrawerBalanceBurnt", log); err != nil {
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

// ParseWithdrawerBalanceBurnt is a log parse operation binding the contract event 0x7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f.
//
// Solidity: event WithdrawerBalanceBurnt(uint256 indexed amount)
func (_CrossL2Outbox *CrossL2OutboxFilterer) ParseWithdrawerBalanceBurnt(log types.Log) (*CrossL2OutboxWithdrawerBalanceBurnt, error) {
	event := new(CrossL2OutboxWithdrawerBalanceBurnt)
	if err := _CrossL2Outbox.contract.UnpackLog(event, "WithdrawerBalanceBurnt", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
