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

// L1FeeVaultMetaData contains all meta data concerning the L1FeeVault contract.
var L1FeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_recipient\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"recipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x61012060405234801561001157600080fd5b506040516108ae3803806108ae83398101604081905261003091610061565b5060a08051678ac7230489e800006080526001600160a01b03169052600060c081905260e052600161010052610091565b60006020828403121561007357600080fd5b81516001600160a01b038116811461008a57600080fd5b9392505050565b60805160a05160c05160e051610100516107c66100e86000396000610399015260006103700152600061034701526000818160a30152818161025f01526102d901526000818160fc015261012e01526107c66000f3fe6080604052600436106100435760003560e01c80633ccfd60b1461004f57806354fd4d501461006657806366d003ac14610091578063d3e5792b146100ea57600080fd5b3661004a57005b600080fd5b34801561005b57600080fd5b5061006461012c565b005b34801561007257600080fd5b5061007b610340565b604051610088919061059a565b60405180910390f35b34801561009d57600080fd5b506100c57f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610088565b3480156100f657600080fd5b5061011e7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610088565b7f0000000000000000000000000000000000000000000000000000000000000000471015610206576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a40160405180910390fd5b604080516020810182526000815290517fe11013dd00000000000000000000000000000000000000000000000000000000815247917342000000000000000000000000000000000000109163e11013dd91849161028c917f000000000000000000000000000000000000000000000000000000000000000091614e2091906004016105b4565b6000604051808303818588803b1580156102a557600080fd5b505af11580156102b9573d6000803e3d6000fd5b50506040805185815273ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016602082015233918101919091527fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba935060600191506103359050565b60405180910390a150565b606061036b7f00000000000000000000000000000000000000000000000000000000000000006103e3565b6103947f00000000000000000000000000000000000000000000000000000000000000006103e3565b6103bd7f00000000000000000000000000000000000000000000000000000000000000006103e3565b6040516020016103cf939291906105f8565b604051602081830303815290604052905090565b60608160000361042657505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610450578061043a8161069d565b91506104499050600a83610704565b915061042a565b60008167ffffffffffffffff81111561046b5761046b610718565b6040519080825280601f01601f191660200182016040528015610495576020820181803683370190505b5090505b8415610518576104aa600183610747565b91506104b7600a8661075e565b6104c2906030610772565b60f81b8183815181106104d7576104d761078a565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610511600a86610704565b9450610499565b949350505050565b60005b8381101561053b578181015183820152602001610523565b8381111561054a576000848401525b50505050565b60008151808452610568816020860160208601610520565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006105ad6020830184610550565b9392505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff831660208201526060604082015260006105ef6060830184610550565b95945050505050565b6000845161060a818460208901610520565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610646816001850160208a01610520565b60019201918201528351610661816002840160208801610520565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036106ce576106ce61066e565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610713576107136106d5565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156107595761075961066e565b500390565b60008261076d5761076d6106d5565b500690565b600082198211156107855761078561066e565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// L1FeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use L1FeeVaultMetaData.ABI instead.
var L1FeeVaultABI = L1FeeVaultMetaData.ABI

// L1FeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1FeeVaultMetaData.Bin instead.
var L1FeeVaultBin = L1FeeVaultMetaData.Bin

// DeployL1FeeVault deploys a new Ethereum contract, binding an instance of L1FeeVault to it.
func DeployL1FeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address) (common.Address, *types.Transaction, *L1FeeVault, error) {
	parsed, err := L1FeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1FeeVaultBin), backend, _recipient)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1FeeVault{L1FeeVaultCaller: L1FeeVaultCaller{contract: contract}, L1FeeVaultTransactor: L1FeeVaultTransactor{contract: contract}, L1FeeVaultFilterer: L1FeeVaultFilterer{contract: contract}}, nil
}

// L1FeeVault is an auto generated Go binding around an Ethereum contract.
type L1FeeVault struct {
	L1FeeVaultCaller     // Read-only binding to the contract
	L1FeeVaultTransactor // Write-only binding to the contract
	L1FeeVaultFilterer   // Log filterer for contract events
}

// L1FeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1FeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1FeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1FeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1FeeVaultSession struct {
	Contract     *L1FeeVault       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1FeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1FeeVaultCallerSession struct {
	Contract *L1FeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// L1FeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1FeeVaultTransactorSession struct {
	Contract     *L1FeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// L1FeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1FeeVaultRaw struct {
	Contract *L1FeeVault // Generic contract binding to access the raw methods on
}

// L1FeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1FeeVaultCallerRaw struct {
	Contract *L1FeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// L1FeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1FeeVaultTransactorRaw struct {
	Contract *L1FeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1FeeVault creates a new instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVault(address common.Address, backend bind.ContractBackend) (*L1FeeVault, error) {
	contract, err := bindL1FeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1FeeVault{L1FeeVaultCaller: L1FeeVaultCaller{contract: contract}, L1FeeVaultTransactor: L1FeeVaultTransactor{contract: contract}, L1FeeVaultFilterer: L1FeeVaultFilterer{contract: contract}}, nil
}

// NewL1FeeVaultCaller creates a new read-only instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultCaller(address common.Address, caller bind.ContractCaller) (*L1FeeVaultCaller, error) {
	contract, err := bindL1FeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultCaller{contract: contract}, nil
}

// NewL1FeeVaultTransactor creates a new write-only instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*L1FeeVaultTransactor, error) {
	contract, err := bindL1FeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultTransactor{contract: contract}, nil
}

// NewL1FeeVaultFilterer creates a new log filterer instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*L1FeeVaultFilterer, error) {
	contract, err := bindL1FeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultFilterer{contract: contract}, nil
}

// bindL1FeeVault binds a generic wrapper to an already deployed contract.
func bindL1FeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1FeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1FeeVault *L1FeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1FeeVault.Contract.L1FeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1FeeVault *L1FeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.Contract.L1FeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1FeeVault *L1FeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1FeeVault.Contract.L1FeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1FeeVault *L1FeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1FeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1FeeVault *L1FeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1FeeVault *L1FeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1FeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _L1FeeVault.Contract.MINWITHDRAWALAMOUNT(&_L1FeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _L1FeeVault.Contract.MINWITHDRAWALAMOUNT(&_L1FeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_L1FeeVault *L1FeeVaultCaller) Recipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "recipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_L1FeeVault *L1FeeVaultSession) Recipient() (common.Address, error) {
	return _L1FeeVault.Contract.Recipient(&_L1FeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_L1FeeVault *L1FeeVaultCallerSession) Recipient() (common.Address, error) {
	return _L1FeeVault.Contract.Recipient(&_L1FeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultSession) Version() (string, error) {
	return _L1FeeVault.Contract.Version(&_L1FeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultCallerSession) Version() (string, error) {
	return _L1FeeVault.Contract.Version(&_L1FeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Withdraw(&_L1FeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Withdraw(&_L1FeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultSession) Receive() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Receive(&_L1FeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Receive(&_L1FeeVault.TransactOpts)
}

// L1FeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the L1FeeVault contract.
type L1FeeVaultWithdrawalIterator struct {
	Event *L1FeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *L1FeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1FeeVaultWithdrawal)
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
		it.Event = new(L1FeeVaultWithdrawal)
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
func (it *L1FeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1FeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1FeeVaultWithdrawal represents a Withdrawal event raised by the L1FeeVault contract.
type L1FeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_L1FeeVault *L1FeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*L1FeeVaultWithdrawalIterator, error) {

	logs, sub, err := _L1FeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultWithdrawalIterator{contract: _L1FeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_L1FeeVault *L1FeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *L1FeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _L1FeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1FeeVaultWithdrawal)
				if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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

// ParseWithdrawal is a log parse operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_L1FeeVault *L1FeeVaultFilterer) ParseWithdrawal(log types.Log) (*L1FeeVaultWithdrawal, error) {
	event := new(L1FeeVaultWithdrawal)
	if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
