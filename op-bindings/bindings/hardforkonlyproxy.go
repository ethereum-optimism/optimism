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

// HardforkOnlyProxyMetaData contains all meta data concerning the HardforkOnlyProxy contract.
var HardforkOnlyProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"previousAdmin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"AdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"implementation\",\"type\":\"address\"}],\"name\":\"Upgraded\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"admin\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_admin\",\"type\":\"address\"}],\"name\":\"changeAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"implementation\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"}],\"name\":\"upgradeTo\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_implementation\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"upgradeToAndCall\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b5073123400000000000000000000000000000000432161002f81610035565b506100a9565b600061004d6000805160206108bf8339815191525490565b6000805160206108bf833981519152839055604080516001600160a01b038084168252851660208201529192507f7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f910160405180910390a15050565b610807806100b86000396000f3fe60806040526004361061005e5760003560e01c80635c60da1b116100435780635c60da1b146100be5780638f283970146100f8578063f851a440146101185761006d565b80633659cfe6146100755780634f1ef286146100955761006d565b3661006d5761006b61012d565b005b61006b61012d565b34801561008157600080fd5b5061006b6100903660046106d9565b610224565b6100a86100a33660046106f4565b610296565b6040516100b59190610777565b60405180910390f35b3480156100ca57600080fd5b506100d3610419565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100b5565b34801561010457600080fd5b5061006b6101133660046106d9565b6104b0565b34801561012457600080fd5b506100d3610517565b60006101577f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc5490565b905073ffffffffffffffffffffffffffffffffffffffff8116610201576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f50726f78793a20696d706c656d656e746174696f6e206e6f7420696e6974696160448201527f6c697a656400000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b3660008037600080366000845af43d6000803e8061021e573d6000fd5b503d6000f35b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16148061027d575033155b1561028e5761028b816105a3565b50565b61028b61012d565b60606102c07fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614806102f7575033155b1561040a57610305846105a3565b6000808573ffffffffffffffffffffffffffffffffffffffff16858560405161032f9291906107ea565b600060405180830381855af49150503d806000811461036a576040519150601f19603f3d011682016040523d82523d6000602084013e61036f565b606091505b509150915081610401576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f50726f78793a2064656c656761746563616c6c20746f206e657720696d706c6560448201527f6d656e746174696f6e20636f6e7472616374206661696c65640000000000000060648201526084016101f8565b91506104129050565b61041261012d565b9392505050565b60006104437fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16148061047a575033155b156104a557507f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc5490565b6104ad61012d565b90565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480610509575033155b1561028e5761028b8161060b565b60006105417fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161480610578575033155b156104a557507fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b7f360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc81905560405173ffffffffffffffffffffffffffffffffffffffff8216907fbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b90600090a250565b60006106357fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b7fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61038390556040805173ffffffffffffffffffffffffffffffffffffffff8084168252851660208201529192507f7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f910160405180910390a15050565b803573ffffffffffffffffffffffffffffffffffffffff811681146106d457600080fd5b919050565b6000602082840312156106eb57600080fd5b610412826106b0565b60008060006040848603121561070957600080fd5b610712846106b0565b9250602084013567ffffffffffffffff8082111561072f57600080fd5b818601915086601f83011261074357600080fd5b81358181111561075257600080fd5b87602082850101111561076457600080fd5b6020830194508093505050509250925092565b600060208083528351808285015260005b818110156107a457858101830151858201604001528201610788565b818111156107b6576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b818382376000910190815291905056fea164736f6c634300080f000ab53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103",
}

// HardforkOnlyProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use HardforkOnlyProxyMetaData.ABI instead.
var HardforkOnlyProxyABI = HardforkOnlyProxyMetaData.ABI

// HardforkOnlyProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use HardforkOnlyProxyMetaData.Bin instead.
var HardforkOnlyProxyBin = HardforkOnlyProxyMetaData.Bin

// DeployHardforkOnlyProxy deploys a new Ethereum contract, binding an instance of HardforkOnlyProxy to it.
func DeployHardforkOnlyProxy(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *HardforkOnlyProxy, error) {
	parsed, err := HardforkOnlyProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(HardforkOnlyProxyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &HardforkOnlyProxy{HardforkOnlyProxyCaller: HardforkOnlyProxyCaller{contract: contract}, HardforkOnlyProxyTransactor: HardforkOnlyProxyTransactor{contract: contract}, HardforkOnlyProxyFilterer: HardforkOnlyProxyFilterer{contract: contract}}, nil
}

// HardforkOnlyProxy is an auto generated Go binding around an Ethereum contract.
type HardforkOnlyProxy struct {
	HardforkOnlyProxyCaller     // Read-only binding to the contract
	HardforkOnlyProxyTransactor // Write-only binding to the contract
	HardforkOnlyProxyFilterer   // Log filterer for contract events
}

// HardforkOnlyProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type HardforkOnlyProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HardforkOnlyProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HardforkOnlyProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HardforkOnlyProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HardforkOnlyProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HardforkOnlyProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HardforkOnlyProxySession struct {
	Contract     *HardforkOnlyProxy // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// HardforkOnlyProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HardforkOnlyProxyCallerSession struct {
	Contract *HardforkOnlyProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// HardforkOnlyProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HardforkOnlyProxyTransactorSession struct {
	Contract     *HardforkOnlyProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// HardforkOnlyProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type HardforkOnlyProxyRaw struct {
	Contract *HardforkOnlyProxy // Generic contract binding to access the raw methods on
}

// HardforkOnlyProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HardforkOnlyProxyCallerRaw struct {
	Contract *HardforkOnlyProxyCaller // Generic read-only contract binding to access the raw methods on
}

// HardforkOnlyProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HardforkOnlyProxyTransactorRaw struct {
	Contract *HardforkOnlyProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHardforkOnlyProxy creates a new instance of HardforkOnlyProxy, bound to a specific deployed contract.
func NewHardforkOnlyProxy(address common.Address, backend bind.ContractBackend) (*HardforkOnlyProxy, error) {
	contract, err := bindHardforkOnlyProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxy{HardforkOnlyProxyCaller: HardforkOnlyProxyCaller{contract: contract}, HardforkOnlyProxyTransactor: HardforkOnlyProxyTransactor{contract: contract}, HardforkOnlyProxyFilterer: HardforkOnlyProxyFilterer{contract: contract}}, nil
}

// NewHardforkOnlyProxyCaller creates a new read-only instance of HardforkOnlyProxy, bound to a specific deployed contract.
func NewHardforkOnlyProxyCaller(address common.Address, caller bind.ContractCaller) (*HardforkOnlyProxyCaller, error) {
	contract, err := bindHardforkOnlyProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxyCaller{contract: contract}, nil
}

// NewHardforkOnlyProxyTransactor creates a new write-only instance of HardforkOnlyProxy, bound to a specific deployed contract.
func NewHardforkOnlyProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*HardforkOnlyProxyTransactor, error) {
	contract, err := bindHardforkOnlyProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxyTransactor{contract: contract}, nil
}

// NewHardforkOnlyProxyFilterer creates a new log filterer instance of HardforkOnlyProxy, bound to a specific deployed contract.
func NewHardforkOnlyProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*HardforkOnlyProxyFilterer, error) {
	contract, err := bindHardforkOnlyProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxyFilterer{contract: contract}, nil
}

// bindHardforkOnlyProxy binds a generic wrapper to an already deployed contract.
func bindHardforkOnlyProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(HardforkOnlyProxyABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HardforkOnlyProxy *HardforkOnlyProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HardforkOnlyProxy.Contract.HardforkOnlyProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HardforkOnlyProxy *HardforkOnlyProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.HardforkOnlyProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HardforkOnlyProxy *HardforkOnlyProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.HardforkOnlyProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HardforkOnlyProxy *HardforkOnlyProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HardforkOnlyProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.contract.Transact(opts, method, params...)
}

// Admin is a paid mutator transaction binding the contract method 0xf851a440.
//
// Solidity: function admin() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) Admin(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.Transact(opts, "admin")
}

// Admin is a paid mutator transaction binding the contract method 0xf851a440.
//
// Solidity: function admin() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxySession) Admin() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Admin(&_HardforkOnlyProxy.TransactOpts)
}

// Admin is a paid mutator transaction binding the contract method 0xf851a440.
//
// Solidity: function admin() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) Admin() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Admin(&_HardforkOnlyProxy.TransactOpts)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x8f283970.
//
// Solidity: function changeAdmin(address _admin) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) ChangeAdmin(opts *bind.TransactOpts, _admin common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.Transact(opts, "changeAdmin", _admin)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x8f283970.
//
// Solidity: function changeAdmin(address _admin) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxySession) ChangeAdmin(_admin common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.ChangeAdmin(&_HardforkOnlyProxy.TransactOpts, _admin)
}

// ChangeAdmin is a paid mutator transaction binding the contract method 0x8f283970.
//
// Solidity: function changeAdmin(address _admin) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) ChangeAdmin(_admin common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.ChangeAdmin(&_HardforkOnlyProxy.TransactOpts, _admin)
}

// Implementation is a paid mutator transaction binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) Implementation(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.Transact(opts, "implementation")
}

// Implementation is a paid mutator transaction binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxySession) Implementation() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Implementation(&_HardforkOnlyProxy.TransactOpts)
}

// Implementation is a paid mutator transaction binding the contract method 0x5c60da1b.
//
// Solidity: function implementation() returns(address)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) Implementation() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Implementation(&_HardforkOnlyProxy.TransactOpts)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address _implementation) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) UpgradeTo(opts *bind.TransactOpts, _implementation common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.Transact(opts, "upgradeTo", _implementation)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address _implementation) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxySession) UpgradeTo(_implementation common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.UpgradeTo(&_HardforkOnlyProxy.TransactOpts, _implementation)
}

// UpgradeTo is a paid mutator transaction binding the contract method 0x3659cfe6.
//
// Solidity: function upgradeTo(address _implementation) returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) UpgradeTo(_implementation common.Address) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.UpgradeTo(&_HardforkOnlyProxy.TransactOpts, _implementation)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address _implementation, bytes _data) payable returns(bytes)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) UpgradeToAndCall(opts *bind.TransactOpts, _implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.Transact(opts, "upgradeToAndCall", _implementation, _data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address _implementation, bytes _data) payable returns(bytes)
func (_HardforkOnlyProxy *HardforkOnlyProxySession) UpgradeToAndCall(_implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.UpgradeToAndCall(&_HardforkOnlyProxy.TransactOpts, _implementation, _data)
}

// UpgradeToAndCall is a paid mutator transaction binding the contract method 0x4f1ef286.
//
// Solidity: function upgradeToAndCall(address _implementation, bytes _data) payable returns(bytes)
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) UpgradeToAndCall(_implementation common.Address, _data []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.UpgradeToAndCall(&_HardforkOnlyProxy.TransactOpts, _implementation, _data)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Fallback(&_HardforkOnlyProxy.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Fallback(&_HardforkOnlyProxy.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HardforkOnlyProxy.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxySession) Receive() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Receive(&_HardforkOnlyProxy.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_HardforkOnlyProxy *HardforkOnlyProxyTransactorSession) Receive() (*types.Transaction, error) {
	return _HardforkOnlyProxy.Contract.Receive(&_HardforkOnlyProxy.TransactOpts)
}

// HardforkOnlyProxyAdminChangedIterator is returned from FilterAdminChanged and is used to iterate over the raw logs and unpacked data for AdminChanged events raised by the HardforkOnlyProxy contract.
type HardforkOnlyProxyAdminChangedIterator struct {
	Event *HardforkOnlyProxyAdminChanged // Event containing the contract specifics and raw log

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
func (it *HardforkOnlyProxyAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HardforkOnlyProxyAdminChanged)
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
		it.Event = new(HardforkOnlyProxyAdminChanged)
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
func (it *HardforkOnlyProxyAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HardforkOnlyProxyAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HardforkOnlyProxyAdminChanged represents a AdminChanged event raised by the HardforkOnlyProxy contract.
type HardforkOnlyProxyAdminChanged struct {
	PreviousAdmin common.Address
	NewAdmin      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAdminChanged is a free log retrieval operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) FilterAdminChanged(opts *bind.FilterOpts) (*HardforkOnlyProxyAdminChangedIterator, error) {

	logs, sub, err := _HardforkOnlyProxy.contract.FilterLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxyAdminChangedIterator{contract: _HardforkOnlyProxy.contract, event: "AdminChanged", logs: logs, sub: sub}, nil
}

// WatchAdminChanged is a free log subscription operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) WatchAdminChanged(opts *bind.WatchOpts, sink chan<- *HardforkOnlyProxyAdminChanged) (event.Subscription, error) {

	logs, sub, err := _HardforkOnlyProxy.contract.WatchLogs(opts, "AdminChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HardforkOnlyProxyAdminChanged)
				if err := _HardforkOnlyProxy.contract.UnpackLog(event, "AdminChanged", log); err != nil {
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

// ParseAdminChanged is a log parse operation binding the contract event 0x7e644d79422f17c01e4894b5f4f588d331ebfa28653d42ae832dc59e38c9798f.
//
// Solidity: event AdminChanged(address previousAdmin, address newAdmin)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) ParseAdminChanged(log types.Log) (*HardforkOnlyProxyAdminChanged, error) {
	event := new(HardforkOnlyProxyAdminChanged)
	if err := _HardforkOnlyProxy.contract.UnpackLog(event, "AdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HardforkOnlyProxyUpgradedIterator is returned from FilterUpgraded and is used to iterate over the raw logs and unpacked data for Upgraded events raised by the HardforkOnlyProxy contract.
type HardforkOnlyProxyUpgradedIterator struct {
	Event *HardforkOnlyProxyUpgraded // Event containing the contract specifics and raw log

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
func (it *HardforkOnlyProxyUpgradedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HardforkOnlyProxyUpgraded)
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
		it.Event = new(HardforkOnlyProxyUpgraded)
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
func (it *HardforkOnlyProxyUpgradedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HardforkOnlyProxyUpgradedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HardforkOnlyProxyUpgraded represents a Upgraded event raised by the HardforkOnlyProxy contract.
type HardforkOnlyProxyUpgraded struct {
	Implementation common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpgraded is a free log retrieval operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) FilterUpgraded(opts *bind.FilterOpts, implementation []common.Address) (*HardforkOnlyProxyUpgradedIterator, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _HardforkOnlyProxy.contract.FilterLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return &HardforkOnlyProxyUpgradedIterator{contract: _HardforkOnlyProxy.contract, event: "Upgraded", logs: logs, sub: sub}, nil
}

// WatchUpgraded is a free log subscription operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) WatchUpgraded(opts *bind.WatchOpts, sink chan<- *HardforkOnlyProxyUpgraded, implementation []common.Address) (event.Subscription, error) {

	var implementationRule []interface{}
	for _, implementationItem := range implementation {
		implementationRule = append(implementationRule, implementationItem)
	}

	logs, sub, err := _HardforkOnlyProxy.contract.WatchLogs(opts, "Upgraded", implementationRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HardforkOnlyProxyUpgraded)
				if err := _HardforkOnlyProxy.contract.UnpackLog(event, "Upgraded", log); err != nil {
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

// ParseUpgraded is a log parse operation binding the contract event 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b.
//
// Solidity: event Upgraded(address indexed implementation)
func (_HardforkOnlyProxy *HardforkOnlyProxyFilterer) ParseUpgraded(log types.Log) (*HardforkOnlyProxyUpgraded, error) {
	event := new(HardforkOnlyProxyUpgraded)
	if err := _HardforkOnlyProxy.contract.UnpackLog(event, "Upgraded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
