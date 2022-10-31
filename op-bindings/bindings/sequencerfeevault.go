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

// SequencerFeeVaultMetaData contains all meta data concerning the SequencerFeeVault contract.
var SequencerFeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_recipient\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1FeeWallet\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"recipient\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x61010060405234801561001157600080fd5b506040516108bb3803806108bb8339810160408190526100309161006e565b678ac7230489e80000608052600080546001600160a01b0319166001600160a01b039290921691909117815560a081905260c052600160e05261009e565b60006020828403121561008057600080fd5b81516001600160a01b038116811461009757600080fd5b9392505050565b60805160a05160c05160e0516107dd6100de60003960006103b0015260006103870152600061035e015260008181610110015261016d01526107dd6000f3fe60806040526004361061005e5760003560e01c806366d003ac1161004357806366d003ac146100ac578063d3e5792b146100fe578063d4ff92181461014057600080fd5b80633ccfd60b1461006a57806354fd4d501461008157600080fd5b3661006557005b600080fd5b34801561007657600080fd5b5061007f61016b565b005b34801561008d57600080fd5b50610096610357565b6040516100a391906105b1565b60405180910390f35b3480156100b857600080fd5b506000546100d99073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100a3565b34801561010a57600080fd5b506101327f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020016100a3565b34801561014c57600080fd5b5060005473ffffffffffffffffffffffffffffffffffffffff166100d9565b7f0000000000000000000000000000000000000000000000000000000000000000471015610245576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a40160405180910390fd5b6000805460408051602081018252928352517fe11013dd00000000000000000000000000000000000000000000000000000000815247927342000000000000000000000000000000000000109263e11013dd9285926102c29273ffffffffffffffffffffffffffffffffffffffff1691614e2091906004016105cb565b6000604051808303818588803b1580156102db57600080fd5b505af11580156102ef573d6000803e3d6000fd5b50506000546040805186815273ffffffffffffffffffffffffffffffffffffffff909216602083015233908201527fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9350606001915061034c9050565b60405180910390a150565b60606103827f00000000000000000000000000000000000000000000000000000000000000006103fa565b6103ab7f00000000000000000000000000000000000000000000000000000000000000006103fa565b6103d47f00000000000000000000000000000000000000000000000000000000000000006103fa565b6040516020016103e69392919061060f565b604051602081830303815290604052905090565b60608160000361043d57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156104675780610451816106b4565b91506104609050600a8361071b565b9150610441565b60008167ffffffffffffffff8111156104825761048261072f565b6040519080825280601f01601f1916602001820160405280156104ac576020820181803683370190505b5090505b841561052f576104c160018361075e565b91506104ce600a86610775565b6104d9906030610789565b60f81b8183815181106104ee576104ee6107a1565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610528600a8661071b565b94506104b0565b949350505050565b60005b8381101561055257818101518382015260200161053a565b83811115610561576000848401525b50505050565b6000815180845261057f816020860160208601610537565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006105c46020830184610567565b9392505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff831660208201526060604082015260006106066060830184610567565b95945050505050565b60008451610621818460208901610537565b80830190507f2e00000000000000000000000000000000000000000000000000000000000000808252855161065d816001850160208a01610537565b60019201918201528351610678816002840160208801610537565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036106e5576106e5610685565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261072a5761072a6106ec565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008282101561077057610770610685565b500390565b600082610784576107846106ec565b500690565b6000821982111561079c5761079c610685565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// SequencerFeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerFeeVaultMetaData.ABI instead.
var SequencerFeeVaultABI = SequencerFeeVaultMetaData.ABI

// SequencerFeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerFeeVaultMetaData.Bin instead.
var SequencerFeeVaultBin = SequencerFeeVaultMetaData.Bin

// DeploySequencerFeeVault deploys a new Ethereum contract, binding an instance of SequencerFeeVault to it.
func DeploySequencerFeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address) (common.Address, *types.Transaction, *SequencerFeeVault, error) {
	parsed, err := SequencerFeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerFeeVaultBin), backend, _recipient)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SequencerFeeVault{SequencerFeeVaultCaller: SequencerFeeVaultCaller{contract: contract}, SequencerFeeVaultTransactor: SequencerFeeVaultTransactor{contract: contract}, SequencerFeeVaultFilterer: SequencerFeeVaultFilterer{contract: contract}}, nil
}

// SequencerFeeVault is an auto generated Go binding around an Ethereum contract.
type SequencerFeeVault struct {
	SequencerFeeVaultCaller     // Read-only binding to the contract
	SequencerFeeVaultTransactor // Write-only binding to the contract
	SequencerFeeVaultFilterer   // Log filterer for contract events
}

// SequencerFeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerFeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerFeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerFeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerFeeVaultSession struct {
	Contract     *SequencerFeeVault // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// SequencerFeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerFeeVaultCallerSession struct {
	Contract *SequencerFeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// SequencerFeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerFeeVaultTransactorSession struct {
	Contract     *SequencerFeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// SequencerFeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerFeeVaultRaw struct {
	Contract *SequencerFeeVault // Generic contract binding to access the raw methods on
}

// SequencerFeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerFeeVaultCallerRaw struct {
	Contract *SequencerFeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerFeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerFeeVaultTransactorRaw struct {
	Contract *SequencerFeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencerFeeVault creates a new instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVault(address common.Address, backend bind.ContractBackend) (*SequencerFeeVault, error) {
	contract, err := bindSequencerFeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVault{SequencerFeeVaultCaller: SequencerFeeVaultCaller{contract: contract}, SequencerFeeVaultTransactor: SequencerFeeVaultTransactor{contract: contract}, SequencerFeeVaultFilterer: SequencerFeeVaultFilterer{contract: contract}}, nil
}

// NewSequencerFeeVaultCaller creates a new read-only instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultCaller(address common.Address, caller bind.ContractCaller) (*SequencerFeeVaultCaller, error) {
	contract, err := bindSequencerFeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultCaller{contract: contract}, nil
}

// NewSequencerFeeVaultTransactor creates a new write-only instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerFeeVaultTransactor, error) {
	contract, err := bindSequencerFeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultTransactor{contract: contract}, nil
}

// NewSequencerFeeVaultFilterer creates a new log filterer instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerFeeVaultFilterer, error) {
	contract, err := bindSequencerFeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultFilterer{contract: contract}, nil
}

// bindSequencerFeeVault binds a generic wrapper to an already deployed contract.
func bindSequencerFeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SequencerFeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerFeeVault.Contract.SequencerFeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.SequencerFeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.SequencerFeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerFeeVault *SequencerFeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerFeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerFeeVault *SequencerFeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerFeeVault *SequencerFeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _SequencerFeeVault.Contract.MINWITHDRAWALAMOUNT(&_SequencerFeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _SequencerFeeVault.Contract.MINWITHDRAWALAMOUNT(&_SequencerFeeVault.CallOpts)
}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCaller) L1FeeWallet(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "l1FeeWallet")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultSession) L1FeeWallet() (common.Address, error) {
	return _SequencerFeeVault.Contract.L1FeeWallet(&_SequencerFeeVault.CallOpts)
}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) L1FeeWallet() (common.Address, error) {
	return _SequencerFeeVault.Contract.L1FeeWallet(&_SequencerFeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCaller) Recipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "recipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultSession) Recipient() (common.Address, error) {
	return _SequencerFeeVault.Contract.Recipient(&_SequencerFeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) Recipient() (common.Address, error) {
	return _SequencerFeeVault.Contract.Recipient(&_SequencerFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultSession) Version() (string, error) {
	return _SequencerFeeVault.Contract.Version(&_SequencerFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) Version() (string, error) {
	return _SequencerFeeVault.Contract.Version(&_SequencerFeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Withdraw(&_SequencerFeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Withdraw(&_SequencerFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultSession) Receive() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Receive(&_SequencerFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Receive(&_SequencerFeeVault.TransactOpts)
}

// SequencerFeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawalIterator struct {
	Event *SequencerFeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *SequencerFeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerFeeVaultWithdrawal)
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
		it.Event = new(SequencerFeeVaultWithdrawal)
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
func (it *SequencerFeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerFeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerFeeVaultWithdrawal represents a Withdrawal event raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*SequencerFeeVaultWithdrawalIterator, error) {

	logs, sub, err := _SequencerFeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultWithdrawalIterator{contract: _SequencerFeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *SequencerFeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _SequencerFeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerFeeVaultWithdrawal)
				if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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
func (_SequencerFeeVault *SequencerFeeVaultFilterer) ParseWithdrawal(log types.Log) (*SequencerFeeVaultWithdrawal, error) {
	event := new(SequencerFeeVaultWithdrawal)
	if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
