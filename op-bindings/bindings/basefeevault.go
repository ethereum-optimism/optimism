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

// BaseFeeVaultMetaData contains all meta data concerning the BaseFeeVault contract.
var BaseFeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_recipient\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RECIPIENT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x61012060405234801561001157600080fd5b506040516108993803806108998339810160408190526100309161005d565b678ac7230489e800006080526001600160a01b031660a052600060c081905260e05260016101005261008d565b60006020828403121561006f57600080fd5b81516001600160a01b038116811461008657600080fd5b9392505050565b60805160a05160c05160e051610100516107b56100e460003960006103880152600061035f0152600061033601526000818160610152818161020c01526102ce01526000818160f8015261012a01526107b56000f3fe6080604052600436106100435760003560e01c80630d9019e11461004f5780633ccfd60b146100ad57806354fd4d50146100c4578063d3e5792b146100e657600080fd5b3661004a57005b600080fd5b34801561005b57600080fd5b506100837f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100b957600080fd5b506100c2610128565b005b3480156100d057600080fd5b506100d961032f565b6040516100a49190610589565b3480156100f257600080fd5b5061011a7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020016100a4565b7f0000000000000000000000000000000000000000000000000000000000000000471015610202576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a40160405180910390fd5b60408051478082527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff166020830152338284015291517fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9181900360600190a1604080516020810182526000815290517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109163e11013dd9184916102fa917f000000000000000000000000000000000000000000000000000000000000000091614e20916004016105a3565b6000604051808303818588803b15801561031357600080fd5b505af1158015610327573d6000803e3d6000fd5b505050505050565b606061035a7f00000000000000000000000000000000000000000000000000000000000000006103d2565b6103837f00000000000000000000000000000000000000000000000000000000000000006103d2565b6103ac7f00000000000000000000000000000000000000000000000000000000000000006103d2565b6040516020016103be939291906105e7565b604051602081830303815290604052905090565b60608160000361041557505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b811561043f57806104298161068c565b91506104389050600a836106f3565b9150610419565b60008167ffffffffffffffff81111561045a5761045a610707565b6040519080825280601f01601f191660200182016040528015610484576020820181803683370190505b5090505b841561050757610499600183610736565b91506104a6600a8661074d565b6104b1906030610761565b60f81b8183815181106104c6576104c6610779565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610500600a866106f3565b9450610488565b949350505050565b60005b8381101561052a578181015183820152602001610512565b83811115610539576000848401525b50505050565b6000815180845261055781602086016020860161050f565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061059c602083018461053f565b9392505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff831660208201526060604082015260006105de606083018461053f565b95945050505050565b600084516105f981846020890161050f565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610635816001850160208a0161050f565b6001920191820152835161065081600284016020880161050f565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036106bd576106bd61065d565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610702576107026106c4565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156107485761074861065d565b500390565b60008261075c5761075c6106c4565b500690565b600082198211156107745761077461065d565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// BaseFeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use BaseFeeVaultMetaData.ABI instead.
var BaseFeeVaultABI = BaseFeeVaultMetaData.ABI

// BaseFeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BaseFeeVaultMetaData.Bin instead.
var BaseFeeVaultBin = BaseFeeVaultMetaData.Bin

// DeployBaseFeeVault deploys a new Ethereum contract, binding an instance of BaseFeeVault to it.
func DeployBaseFeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address) (common.Address, *types.Transaction, *BaseFeeVault, error) {
	parsed, err := BaseFeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BaseFeeVaultBin), backend, _recipient)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BaseFeeVault{BaseFeeVaultCaller: BaseFeeVaultCaller{contract: contract}, BaseFeeVaultTransactor: BaseFeeVaultTransactor{contract: contract}, BaseFeeVaultFilterer: BaseFeeVaultFilterer{contract: contract}}, nil
}

// BaseFeeVault is an auto generated Go binding around an Ethereum contract.
type BaseFeeVault struct {
	BaseFeeVaultCaller     // Read-only binding to the contract
	BaseFeeVaultTransactor // Write-only binding to the contract
	BaseFeeVaultFilterer   // Log filterer for contract events
}

// BaseFeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type BaseFeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BaseFeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BaseFeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BaseFeeVaultSession struct {
	Contract     *BaseFeeVault     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BaseFeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BaseFeeVaultCallerSession struct {
	Contract *BaseFeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// BaseFeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BaseFeeVaultTransactorSession struct {
	Contract     *BaseFeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// BaseFeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type BaseFeeVaultRaw struct {
	Contract *BaseFeeVault // Generic contract binding to access the raw methods on
}

// BaseFeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BaseFeeVaultCallerRaw struct {
	Contract *BaseFeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// BaseFeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BaseFeeVaultTransactorRaw struct {
	Contract *BaseFeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBaseFeeVault creates a new instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVault(address common.Address, backend bind.ContractBackend) (*BaseFeeVault, error) {
	contract, err := bindBaseFeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVault{BaseFeeVaultCaller: BaseFeeVaultCaller{contract: contract}, BaseFeeVaultTransactor: BaseFeeVaultTransactor{contract: contract}, BaseFeeVaultFilterer: BaseFeeVaultFilterer{contract: contract}}, nil
}

// NewBaseFeeVaultCaller creates a new read-only instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultCaller(address common.Address, caller bind.ContractCaller) (*BaseFeeVaultCaller, error) {
	contract, err := bindBaseFeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultCaller{contract: contract}, nil
}

// NewBaseFeeVaultTransactor creates a new write-only instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*BaseFeeVaultTransactor, error) {
	contract, err := bindBaseFeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultTransactor{contract: contract}, nil
}

// NewBaseFeeVaultFilterer creates a new log filterer instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*BaseFeeVaultFilterer, error) {
	contract, err := bindBaseFeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultFilterer{contract: contract}, nil
}

// bindBaseFeeVault binds a generic wrapper to an already deployed contract.
func bindBaseFeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BaseFeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseFeeVault *BaseFeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseFeeVault.Contract.BaseFeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseFeeVault *BaseFeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.BaseFeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseFeeVault *BaseFeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.BaseFeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseFeeVault *BaseFeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseFeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseFeeVault *BaseFeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseFeeVault *BaseFeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BaseFeeVault.Contract.MINWITHDRAWALAMOUNT(&_BaseFeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BaseFeeVault.Contract.MINWITHDRAWALAMOUNT(&_BaseFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultCaller) RECIPIENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "RECIPIENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultSession) RECIPIENT() (common.Address, error) {
	return _BaseFeeVault.Contract.RECIPIENT(&_BaseFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultCallerSession) RECIPIENT() (common.Address, error) {
	return _BaseFeeVault.Contract.RECIPIENT(&_BaseFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultSession) Version() (string, error) {
	return _BaseFeeVault.Contract.Version(&_BaseFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultCallerSession) Version() (string, error) {
	return _BaseFeeVault.Contract.Version(&_BaseFeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Withdraw(&_BaseFeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Withdraw(&_BaseFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultSession) Receive() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Receive(&_BaseFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Receive(&_BaseFeeVault.TransactOpts)
}

// BaseFeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawalIterator struct {
	Event *BaseFeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *BaseFeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BaseFeeVaultWithdrawal)
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
		it.Event = new(BaseFeeVaultWithdrawal)
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
func (it *BaseFeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BaseFeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BaseFeeVaultWithdrawal represents a Withdrawal event raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_BaseFeeVault *BaseFeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*BaseFeeVaultWithdrawalIterator, error) {

	logs, sub, err := _BaseFeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultWithdrawalIterator{contract: _BaseFeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_BaseFeeVault *BaseFeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *BaseFeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _BaseFeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BaseFeeVaultWithdrawal)
				if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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
func (_BaseFeeVault *BaseFeeVaultFilterer) ParseWithdrawal(log types.Log) (*BaseFeeVaultWithdrawal, error) {
	event := new(BaseFeeVaultWithdrawal)
	if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
