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

// L2ToL1MessagePasserMetaData contains all meta data concerning the L2ToL1MessagePasser contract.
var L2ToL1MessagePasserMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"WithdrawalInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"WithdrawerBalanceBurnt\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"initiateWithdrawal\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nonce\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"sentMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60e060405234801561001057600080fd5b506000608081905260a052600160c05260805160a05160c0516108e861004f60003960006102d3015260006102aa0152600061028101526108e86000f3fe60806040526004361061005e5760003560e01c806382e3702d1161004357806382e3702d146100c7578063affed0e014610107578063c2b3e5ac1461012b57600080fd5b806344df8e701461008757806354fd4d501461009c57600080fd5b366100825761008033620186a060405180602001604052806000815250610139565b005b600080fd5b34801561009357600080fd5b50610080610242565b3480156100a857600080fd5b506100b161027a565b6040516100be919061055c565b60405180910390f35b3480156100d357600080fd5b506100f76100e2366004610576565b60006020819052908152604090205460ff1681565b60405190151581526020016100be565b34801561011357600080fd5b5061011d60015481565b6040519081526020016100be565b6100806101393660046105be565b600061019e6040518060c0016040528060015481526020013373ffffffffffffffffffffffffffffffffffffffff1681526020018673ffffffffffffffffffffffffffffffffffffffff1681526020013481526020018581526020018481525061031d565b6000818152602081905260409081902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915554905191925073ffffffffffffffffffffffffffffffffffffffff8616913391907f87bf7b546c8de873abb0db5b579ec131f8d0cf5b14f39933551cf9ced23a61369061022c903490899089906106c2565b60405180910390a4505060018054810190555050565b4761024c8161036a565b60405181907f7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f90600090a250565b60606102a57f0000000000000000000000000000000000000000000000000000000000000000610399565b6102ce7f0000000000000000000000000000000000000000000000000000000000000000610399565b6102f77f0000000000000000000000000000000000000000000000000000000000000000610399565b604051602001610309939291906106ea565b604051602081830303815290604052905090565b80516020808301516040808501516060860151608087015160a0880151935160009761034d979096959101610760565b604051602081830303815290604052805190602001209050919050565b80604051610377906104d6565b6040518091039082f0905080158015610394573d6000803e3d6000fd5b505050565b6060816000036103dc57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b811561040657806103f0816107e6565b91506103ff9050600a8361084d565b91506103e0565b60008167ffffffffffffffff8111156104215761042161058f565b6040519080825280601f01601f19166020018201604052801561044b576020820181803683370190505b5090505b84156104ce57610460600183610861565b915061046d600a86610878565b61047890603061088c565b60f81b81838151811061048d5761048d6108a4565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506104c7600a8661084d565b945061044f565b949350505050565b6008806108d483390190565b60005b838110156104fd5781810151838201526020016104e5565b8381111561050c576000848401525b50505050565b6000815180845261052a8160208601602086016104e2565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061056f6020830184610512565b9392505050565b60006020828403121561058857600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000806000606084860312156105d357600080fd5b833573ffffffffffffffffffffffffffffffffffffffff811681146105f757600080fd5b925060208401359150604084013567ffffffffffffffff8082111561061b57600080fd5b818601915086601f83011261062f57600080fd5b8135818111156106415761064161058f565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156106875761068761058f565b816040528281528960208487010111156106a057600080fd5b8260208601602083013760006020848301015280955050505050509250925092565b8381528260208201526060604082015260006106e16060830184610512565b95945050505050565b600084516106fc8184602089016104e2565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610738816001850160208a016104e2565b600192019182015283516107538160028401602088016104e2565b0160020195945050505050565b868152600073ffffffffffffffffffffffffffffffffffffffff808816602084015280871660408401525084606083015283608083015260c060a08301526107ab60c0830184610512565b98975050505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610817576108176107b7565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261085c5761085c61081e565b500490565b600082821015610873576108736107b7565b500390565b6000826108875761088761081e565b500690565b6000821982111561089f5761089f6107b7565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfe608060405230fffea164736f6c634300080f000a",
}

// L2ToL1MessagePasserABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ToL1MessagePasserMetaData.ABI instead.
var L2ToL1MessagePasserABI = L2ToL1MessagePasserMetaData.ABI

// L2ToL1MessagePasserBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2ToL1MessagePasserMetaData.Bin instead.
var L2ToL1MessagePasserBin = L2ToL1MessagePasserMetaData.Bin

// DeployL2ToL1MessagePasser deploys a new Ethereum contract, binding an instance of L2ToL1MessagePasser to it.
func DeployL2ToL1MessagePasser(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L2ToL1MessagePasser, error) {
	parsed, err := L2ToL1MessagePasserMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2ToL1MessagePasserBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2ToL1MessagePasser{L2ToL1MessagePasserCaller: L2ToL1MessagePasserCaller{contract: contract}, L2ToL1MessagePasserTransactor: L2ToL1MessagePasserTransactor{contract: contract}, L2ToL1MessagePasserFilterer: L2ToL1MessagePasserFilterer{contract: contract}}, nil
}

// L2ToL1MessagePasser is an auto generated Go binding around an Ethereum contract.
type L2ToL1MessagePasser struct {
	L2ToL1MessagePasserCaller     // Read-only binding to the contract
	L2ToL1MessagePasserTransactor // Write-only binding to the contract
	L2ToL1MessagePasserFilterer   // Log filterer for contract events
}

// L2ToL1MessagePasserCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ToL1MessagePasserCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL1MessagePasserTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ToL1MessagePasserTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL1MessagePasserFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2ToL1MessagePasserFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL1MessagePasserSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2ToL1MessagePasserSession struct {
	Contract     *L2ToL1MessagePasser // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// L2ToL1MessagePasserCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2ToL1MessagePasserCallerSession struct {
	Contract *L2ToL1MessagePasserCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// L2ToL1MessagePasserTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2ToL1MessagePasserTransactorSession struct {
	Contract     *L2ToL1MessagePasserTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// L2ToL1MessagePasserRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2ToL1MessagePasserRaw struct {
	Contract *L2ToL1MessagePasser // Generic contract binding to access the raw methods on
}

// L2ToL1MessagePasserCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2ToL1MessagePasserCallerRaw struct {
	Contract *L2ToL1MessagePasserCaller // Generic read-only contract binding to access the raw methods on
}

// L2ToL1MessagePasserTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2ToL1MessagePasserTransactorRaw struct {
	Contract *L2ToL1MessagePasserTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2ToL1MessagePasser creates a new instance of L2ToL1MessagePasser, bound to a specific deployed contract.
func NewL2ToL1MessagePasser(address common.Address, backend bind.ContractBackend) (*L2ToL1MessagePasser, error) {
	contract, err := bindL2ToL1MessagePasser(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasser{L2ToL1MessagePasserCaller: L2ToL1MessagePasserCaller{contract: contract}, L2ToL1MessagePasserTransactor: L2ToL1MessagePasserTransactor{contract: contract}, L2ToL1MessagePasserFilterer: L2ToL1MessagePasserFilterer{contract: contract}}, nil
}

// NewL2ToL1MessagePasserCaller creates a new read-only instance of L2ToL1MessagePasser, bound to a specific deployed contract.
func NewL2ToL1MessagePasserCaller(address common.Address, caller bind.ContractCaller) (*L2ToL1MessagePasserCaller, error) {
	contract, err := bindL2ToL1MessagePasser(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasserCaller{contract: contract}, nil
}

// NewL2ToL1MessagePasserTransactor creates a new write-only instance of L2ToL1MessagePasser, bound to a specific deployed contract.
func NewL2ToL1MessagePasserTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ToL1MessagePasserTransactor, error) {
	contract, err := bindL2ToL1MessagePasser(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasserTransactor{contract: contract}, nil
}

// NewL2ToL1MessagePasserFilterer creates a new log filterer instance of L2ToL1MessagePasser, bound to a specific deployed contract.
func NewL2ToL1MessagePasserFilterer(address common.Address, filterer bind.ContractFilterer) (*L2ToL1MessagePasserFilterer, error) {
	contract, err := bindL2ToL1MessagePasser(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasserFilterer{contract: contract}, nil
}

// bindL2ToL1MessagePasser binds a generic wrapper to an already deployed contract.
func bindL2ToL1MessagePasser(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ToL1MessagePasserABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL1MessagePasser.Contract.L2ToL1MessagePasserCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.L2ToL1MessagePasserTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.L2ToL1MessagePasserTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL1MessagePasser.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.contract.Transact(opts, method, params...)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCaller) Nonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL1MessagePasser.contract.Call(opts, &out, "nonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) Nonce() (*big.Int, error) {
	return _L2ToL1MessagePasser.Contract.Nonce(&_L2ToL1MessagePasser.CallOpts)
}

// Nonce is a free data retrieval call binding the contract method 0xaffed0e0.
//
// Solidity: function nonce() view returns(uint256)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCallerSession) Nonce() (*big.Int, error) {
	return _L2ToL1MessagePasser.Contract.Nonce(&_L2ToL1MessagePasser.CallOpts)
}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCaller) SentMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ToL1MessagePasser.contract.Call(opts, &out, "sentMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) SentMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL1MessagePasser.Contract.SentMessages(&_L2ToL1MessagePasser.CallOpts, arg0)
}

// SentMessages is a free data retrieval call binding the contract method 0x82e3702d.
//
// Solidity: function sentMessages(bytes32 ) view returns(bool)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCallerSession) SentMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL1MessagePasser.Contract.SentMessages(&_L2ToL1MessagePasser.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2ToL1MessagePasser.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) Version() (string, error) {
	return _L2ToL1MessagePasser.Contract.Version(&_L2ToL1MessagePasser.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserCallerSession) Version() (string, error) {
	return _L2ToL1MessagePasser.Contract.Version(&_L2ToL1MessagePasser.CallOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactor) Burn(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.contract.Transact(opts, "burn")
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) Burn() (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.Burn(&_L2ToL1MessagePasser.TransactOpts)
}

// Burn is a paid mutator transaction binding the contract method 0x44df8e70.
//
// Solidity: function burn() returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactorSession) Burn() (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.Burn(&_L2ToL1MessagePasser.TransactOpts)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xc2b3e5ac.
//
// Solidity: function initiateWithdrawal(address _target, uint256 _gasLimit, bytes _data) payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactor) InitiateWithdrawal(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.contract.Transact(opts, "initiateWithdrawal", _target, _gasLimit, _data)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xc2b3e5ac.
//
// Solidity: function initiateWithdrawal(address _target, uint256 _gasLimit, bytes _data) payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) InitiateWithdrawal(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.InitiateWithdrawal(&_L2ToL1MessagePasser.TransactOpts, _target, _gasLimit, _data)
}

// InitiateWithdrawal is a paid mutator transaction binding the contract method 0xc2b3e5ac.
//
// Solidity: function initiateWithdrawal(address _target, uint256 _gasLimit, bytes _data) payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactorSession) InitiateWithdrawal(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.InitiateWithdrawal(&_L2ToL1MessagePasser.TransactOpts, _target, _gasLimit, _data)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL1MessagePasser.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserSession) Receive() (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.Receive(&_L2ToL1MessagePasser.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L2ToL1MessagePasser *L2ToL1MessagePasserTransactorSession) Receive() (*types.Transaction, error) {
	return _L2ToL1MessagePasser.Contract.Receive(&_L2ToL1MessagePasser.TransactOpts)
}

// L2ToL1MessagePasserWithdrawalInitiatedIterator is returned from FilterWithdrawalInitiated and is used to iterate over the raw logs and unpacked data for WithdrawalInitiated events raised by the L2ToL1MessagePasser contract.
type L2ToL1MessagePasserWithdrawalInitiatedIterator struct {
	Event *L2ToL1MessagePasserWithdrawalInitiated // Event containing the contract specifics and raw log

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
func (it *L2ToL1MessagePasserWithdrawalInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL1MessagePasserWithdrawalInitiated)
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
		it.Event = new(L2ToL1MessagePasserWithdrawalInitiated)
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
func (it *L2ToL1MessagePasserWithdrawalInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL1MessagePasserWithdrawalInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL1MessagePasserWithdrawalInitiated represents a WithdrawalInitiated event raised by the L2ToL1MessagePasser contract.
type L2ToL1MessagePasserWithdrawalInitiated struct {
	Nonce    *big.Int
	Sender   common.Address
	Target   common.Address
	Value    *big.Int
	GasLimit *big.Int
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalInitiated is a free log retrieval operation binding the contract event 0x87bf7b546c8de873abb0db5b579ec131f8d0cf5b14f39933551cf9ced23a6136.
//
// Solidity: event WithdrawalInitiated(uint256 indexed nonce, address indexed sender, address indexed target, uint256 value, uint256 gasLimit, bytes data)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) FilterWithdrawalInitiated(opts *bind.FilterOpts, nonce []*big.Int, sender []common.Address, target []common.Address) (*L2ToL1MessagePasserWithdrawalInitiatedIterator, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L2ToL1MessagePasser.contract.FilterLogs(opts, "WithdrawalInitiated", nonceRule, senderRule, targetRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasserWithdrawalInitiatedIterator{contract: _L2ToL1MessagePasser.contract, event: "WithdrawalInitiated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalInitiated is a free log subscription operation binding the contract event 0x87bf7b546c8de873abb0db5b579ec131f8d0cf5b14f39933551cf9ced23a6136.
//
// Solidity: event WithdrawalInitiated(uint256 indexed nonce, address indexed sender, address indexed target, uint256 value, uint256 gasLimit, bytes data)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) WatchWithdrawalInitiated(opts *bind.WatchOpts, sink chan<- *L2ToL1MessagePasserWithdrawalInitiated, nonce []*big.Int, sender []common.Address, target []common.Address) (event.Subscription, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L2ToL1MessagePasser.contract.WatchLogs(opts, "WithdrawalInitiated", nonceRule, senderRule, targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL1MessagePasserWithdrawalInitiated)
				if err := _L2ToL1MessagePasser.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
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

// ParseWithdrawalInitiated is a log parse operation binding the contract event 0x87bf7b546c8de873abb0db5b579ec131f8d0cf5b14f39933551cf9ced23a6136.
//
// Solidity: event WithdrawalInitiated(uint256 indexed nonce, address indexed sender, address indexed target, uint256 value, uint256 gasLimit, bytes data)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) ParseWithdrawalInitiated(log types.Log) (*L2ToL1MessagePasserWithdrawalInitiated, error) {
	event := new(L2ToL1MessagePasserWithdrawalInitiated)
	if err := _L2ToL1MessagePasser.contract.UnpackLog(event, "WithdrawalInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2ToL1MessagePasserWithdrawerBalanceBurntIterator is returned from FilterWithdrawerBalanceBurnt and is used to iterate over the raw logs and unpacked data for WithdrawerBalanceBurnt events raised by the L2ToL1MessagePasser contract.
type L2ToL1MessagePasserWithdrawerBalanceBurntIterator struct {
	Event *L2ToL1MessagePasserWithdrawerBalanceBurnt // Event containing the contract specifics and raw log

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
func (it *L2ToL1MessagePasserWithdrawerBalanceBurntIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL1MessagePasserWithdrawerBalanceBurnt)
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
		it.Event = new(L2ToL1MessagePasserWithdrawerBalanceBurnt)
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
func (it *L2ToL1MessagePasserWithdrawerBalanceBurntIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL1MessagePasserWithdrawerBalanceBurntIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL1MessagePasserWithdrawerBalanceBurnt represents a WithdrawerBalanceBurnt event raised by the L2ToL1MessagePasser contract.
type L2ToL1MessagePasserWithdrawerBalanceBurnt struct {
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterWithdrawerBalanceBurnt is a free log retrieval operation binding the contract event 0x7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f.
//
// Solidity: event WithdrawerBalanceBurnt(uint256 indexed amount)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) FilterWithdrawerBalanceBurnt(opts *bind.FilterOpts, amount []*big.Int) (*L2ToL1MessagePasserWithdrawerBalanceBurntIterator, error) {

	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _L2ToL1MessagePasser.contract.FilterLogs(opts, "WithdrawerBalanceBurnt", amountRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL1MessagePasserWithdrawerBalanceBurntIterator{contract: _L2ToL1MessagePasser.contract, event: "WithdrawerBalanceBurnt", logs: logs, sub: sub}, nil
}

// WatchWithdrawerBalanceBurnt is a free log subscription operation binding the contract event 0x7967de617a5ac1cc7eba2d6f37570a0135afa950d8bb77cdd35f0d0b4e85a16f.
//
// Solidity: event WithdrawerBalanceBurnt(uint256 indexed amount)
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) WatchWithdrawerBalanceBurnt(opts *bind.WatchOpts, sink chan<- *L2ToL1MessagePasserWithdrawerBalanceBurnt, amount []*big.Int) (event.Subscription, error) {

	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _L2ToL1MessagePasser.contract.WatchLogs(opts, "WithdrawerBalanceBurnt", amountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL1MessagePasserWithdrawerBalanceBurnt)
				if err := _L2ToL1MessagePasser.contract.UnpackLog(event, "WithdrawerBalanceBurnt", log); err != nil {
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
func (_L2ToL1MessagePasser *L2ToL1MessagePasserFilterer) ParseWithdrawerBalanceBurnt(log types.Log) (*L2ToL1MessagePasserWithdrawerBalanceBurnt, error) {
	event := new(L2ToL1MessagePasserWithdrawerBalanceBurnt)
	if err := _L2ToL1MessagePasser.contract.UnpackLog(event, "WithdrawerBalanceBurnt", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
