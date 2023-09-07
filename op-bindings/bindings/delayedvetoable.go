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

// DelayedVetoableMetaData contains all meta data concerning the DelayedVetoable contract.
var DelayedVetoableMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"vetoer_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"initiator_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target_\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"delay_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ForwardingEarly\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TargetUnitialized\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"expected\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"actual\",\"type\":\"address\"}],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"callHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"Forwarded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"callHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"Initiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"callHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"Vetoed\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"inputs\":[],\"name\":\"delay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initiator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"target\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"vetoer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60e060405234801561001057600080fd5b50604051610a27380380610a2783398101604081905261002f9161009f565b6000608081905260a0819052600160c081905280546001600160a01b03199081166001600160a01b0397881617909155600280548216958716959095179094558054909316919093161790556004556100ea565b80516001600160a01b038116811461009a57600080fd5b919050565b600080600080608085870312156100b557600080fd5b6100be85610083565b93506100cc60208601610083565b92506100da60408601610083565b6060959095015193969295505050565b60805160a05160c05161090e61011960003960006104560152600061042d01526000610404015261090e6000f3fe60806040526004361061005e5760003560e01c80636a42b8f8116100435780636a42b8f8146100da578063d4b83992146100fd578063d8bff440146101125761006d565b806354fd4d50146100755780635c39fcc1146100a05761006d565b3661006d5761006b610127565b005b61006b610127565b34801561008157600080fd5b5061008a6103fd565b6040516100979190610692565b60405180910390f35b3480156100ac57600080fd5b506100b56104a0565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610097565b3480156100e657600080fd5b506100ef6104cf565b604051908152602001610097565b34801561010957600080fd5b506100b56104dd565b34801561011e57600080fd5b506100b5610501565b600080366040516101399291906106e3565b60405190819003902060025490915073ffffffffffffffffffffffffffffffffffffffff16331480156101785750600081815260036020526040902054155b156101cb576000818152600360205260408082204290555182917f87a332a414acbc7da074543639ce7ae02ff1ea72e88379da9f261b080beb5a13916101c0919036906106f3565b60405180910390a250565b60015473ffffffffffffffffffffffffffffffffffffffff16331480156101ff575060008181526003602052604090205415155b80156102265750600454600082815260036020526040902054610222919061076f565b4211155b1561026e576000818152600360205260408082208290555182917fbede6852c1d97d93ff557f676de76670cd0dec861e7fe8beb13aa0ba2b0ab040916101c0919036906106f3565b60008181526003602052604081205490036102db576002546040517f295a81c100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff909116600482015233602482015260440160405180910390fd5b60045460008281526003602052604090205442916102f89161076f565b1015610330576040517f43dc986d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000818152600360205260408082208290555182917f4c109d85bcd0bb5c735b4be850953d652afe4cd9aa2e0b1426a65a4dcb2e122991610373919036906106f3565b60405180910390a26000805460405173ffffffffffffffffffffffffffffffffffffffff909116906103a890839036906106e3565b6000604051808303816000865af19150503d80600081146103e5576040519150601f19603f3d011682016040523d82523d6000602084013e6103ea565b606091505b50509050806103f8573d6000fd5b3d6000f35b60606104287f0000000000000000000000000000000000000000000000000000000000000000610525565b6104517f0000000000000000000000000000000000000000000000000000000000000000610525565b61047a7f0000000000000000000000000000000000000000000000000000000000000000610525565b60405160200161048c93929190610787565b604051602081830303815290604052905090565b6000336104c4575060025473ffffffffffffffffffffffffffffffffffffffff1690565b6104cc610127565b90565b6000336104c4575060045490565b6000336104c4575060005473ffffffffffffffffffffffffffffffffffffffff1690565b6000336104c4575060015473ffffffffffffffffffffffffffffffffffffffff1690565b60608160000361056857505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610592578061057c816107fd565b915061058b9050600a83610864565b915061056c565b60008167ffffffffffffffff8111156105ad576105ad610878565b6040519080825280601f01601f1916602001820160405280156105d7576020820181803683370190505b5090505b841561065a576105ec6001836108a7565b91506105f9600a866108be565b61060490603061076f565b60f81b818381518110610619576106196108d2565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610653600a86610864565b94506105db565b949350505050565b60005b8381101561067d578181015183820152602001610665565b8381111561068c576000848401525b50505050565b60208152600082518060208401526106b1816040850160208701610662565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b8183823760009101908152919050565b60208152816020820152818360408301376000818301604090810191909152601f9092017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0160101919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561078257610782610740565b500190565b60008451610799818460208901610662565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516107d5816001850160208a01610662565b600192019182015283516107f0816002840160208801610662565b0160020195945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361082e5761082e610740565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261087357610873610835565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156108b9576108b9610740565b500390565b6000826108cd576108cd610835565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// DelayedVetoableABI is the input ABI used to generate the binding from.
// Deprecated: Use DelayedVetoableMetaData.ABI instead.
var DelayedVetoableABI = DelayedVetoableMetaData.ABI

// DelayedVetoableBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DelayedVetoableMetaData.Bin instead.
var DelayedVetoableBin = DelayedVetoableMetaData.Bin

// DeployDelayedVetoable deploys a new Ethereum contract, binding an instance of DelayedVetoable to it.
func DeployDelayedVetoable(auth *bind.TransactOpts, backend bind.ContractBackend, vetoer_ common.Address, initiator_ common.Address, target_ common.Address, delay_ *big.Int) (common.Address, *types.Transaction, *DelayedVetoable, error) {
	parsed, err := DelayedVetoableMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DelayedVetoableBin), backend, vetoer_, initiator_, target_, delay_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DelayedVetoable{DelayedVetoableCaller: DelayedVetoableCaller{contract: contract}, DelayedVetoableTransactor: DelayedVetoableTransactor{contract: contract}, DelayedVetoableFilterer: DelayedVetoableFilterer{contract: contract}}, nil
}

// DelayedVetoable is an auto generated Go binding around an Ethereum contract.
type DelayedVetoable struct {
	DelayedVetoableCaller     // Read-only binding to the contract
	DelayedVetoableTransactor // Write-only binding to the contract
	DelayedVetoableFilterer   // Log filterer for contract events
}

// DelayedVetoableCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelayedVetoableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedVetoableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelayedVetoableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedVetoableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelayedVetoableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelayedVetoableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelayedVetoableSession struct {
	Contract     *DelayedVetoable  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DelayedVetoableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelayedVetoableCallerSession struct {
	Contract *DelayedVetoableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// DelayedVetoableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelayedVetoableTransactorSession struct {
	Contract     *DelayedVetoableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DelayedVetoableRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelayedVetoableRaw struct {
	Contract *DelayedVetoable // Generic contract binding to access the raw methods on
}

// DelayedVetoableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelayedVetoableCallerRaw struct {
	Contract *DelayedVetoableCaller // Generic read-only contract binding to access the raw methods on
}

// DelayedVetoableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelayedVetoableTransactorRaw struct {
	Contract *DelayedVetoableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelayedVetoable creates a new instance of DelayedVetoable, bound to a specific deployed contract.
func NewDelayedVetoable(address common.Address, backend bind.ContractBackend) (*DelayedVetoable, error) {
	contract, err := bindDelayedVetoable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoable{DelayedVetoableCaller: DelayedVetoableCaller{contract: contract}, DelayedVetoableTransactor: DelayedVetoableTransactor{contract: contract}, DelayedVetoableFilterer: DelayedVetoableFilterer{contract: contract}}, nil
}

// NewDelayedVetoableCaller creates a new read-only instance of DelayedVetoable, bound to a specific deployed contract.
func NewDelayedVetoableCaller(address common.Address, caller bind.ContractCaller) (*DelayedVetoableCaller, error) {
	contract, err := bindDelayedVetoable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableCaller{contract: contract}, nil
}

// NewDelayedVetoableTransactor creates a new write-only instance of DelayedVetoable, bound to a specific deployed contract.
func NewDelayedVetoableTransactor(address common.Address, transactor bind.ContractTransactor) (*DelayedVetoableTransactor, error) {
	contract, err := bindDelayedVetoable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableTransactor{contract: contract}, nil
}

// NewDelayedVetoableFilterer creates a new log filterer instance of DelayedVetoable, bound to a specific deployed contract.
func NewDelayedVetoableFilterer(address common.Address, filterer bind.ContractFilterer) (*DelayedVetoableFilterer, error) {
	contract, err := bindDelayedVetoable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableFilterer{contract: contract}, nil
}

// bindDelayedVetoable binds a generic wrapper to an already deployed contract.
func bindDelayedVetoable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DelayedVetoableABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelayedVetoable *DelayedVetoableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelayedVetoable.Contract.DelayedVetoableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelayedVetoable *DelayedVetoableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.DelayedVetoableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelayedVetoable *DelayedVetoableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.DelayedVetoableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelayedVetoable *DelayedVetoableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelayedVetoable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelayedVetoable *DelayedVetoableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelayedVetoable *DelayedVetoableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.contract.Transact(opts, method, params...)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedVetoable *DelayedVetoableCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DelayedVetoable.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedVetoable *DelayedVetoableSession) Version() (string, error) {
	return _DelayedVetoable.Contract.Version(&_DelayedVetoable.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DelayedVetoable *DelayedVetoableCallerSession) Version() (string, error) {
	return _DelayedVetoable.Contract.Version(&_DelayedVetoable.CallOpts)
}

// Delay is a paid mutator transaction binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() returns(uint256)
func (_DelayedVetoable *DelayedVetoableTransactor) Delay(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.contract.Transact(opts, "delay")
}

// Delay is a paid mutator transaction binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() returns(uint256)
func (_DelayedVetoable *DelayedVetoableSession) Delay() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Delay(&_DelayedVetoable.TransactOpts)
}

// Delay is a paid mutator transaction binding the contract method 0x6a42b8f8.
//
// Solidity: function delay() returns(uint256)
func (_DelayedVetoable *DelayedVetoableTransactorSession) Delay() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Delay(&_DelayedVetoable.TransactOpts)
}

// Initiator is a paid mutator transaction binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactor) Initiator(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.contract.Transact(opts, "initiator")
}

// Initiator is a paid mutator transaction binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() returns(address)
func (_DelayedVetoable *DelayedVetoableSession) Initiator() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Initiator(&_DelayedVetoable.TransactOpts)
}

// Initiator is a paid mutator transaction binding the contract method 0x5c39fcc1.
//
// Solidity: function initiator() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactorSession) Initiator() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Initiator(&_DelayedVetoable.TransactOpts)
}

// Target is a paid mutator transaction binding the contract method 0xd4b83992.
//
// Solidity: function target() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactor) Target(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.contract.Transact(opts, "target")
}

// Target is a paid mutator transaction binding the contract method 0xd4b83992.
//
// Solidity: function target() returns(address)
func (_DelayedVetoable *DelayedVetoableSession) Target() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Target(&_DelayedVetoable.TransactOpts)
}

// Target is a paid mutator transaction binding the contract method 0xd4b83992.
//
// Solidity: function target() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactorSession) Target() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Target(&_DelayedVetoable.TransactOpts)
}

// Vetoer is a paid mutator transaction binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactor) Vetoer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.contract.Transact(opts, "vetoer")
}

// Vetoer is a paid mutator transaction binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() returns(address)
func (_DelayedVetoable *DelayedVetoableSession) Vetoer() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Vetoer(&_DelayedVetoable.TransactOpts)
}

// Vetoer is a paid mutator transaction binding the contract method 0xd8bff440.
//
// Solidity: function vetoer() returns(address)
func (_DelayedVetoable *DelayedVetoableTransactorSession) Vetoer() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Vetoer(&_DelayedVetoable.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedVetoable *DelayedVetoableTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _DelayedVetoable.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedVetoable *DelayedVetoableSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Fallback(&_DelayedVetoable.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_DelayedVetoable *DelayedVetoableTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Fallback(&_DelayedVetoable.TransactOpts, calldata)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedVetoable *DelayedVetoableTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelayedVetoable.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedVetoable *DelayedVetoableSession) Receive() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Receive(&_DelayedVetoable.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DelayedVetoable *DelayedVetoableTransactorSession) Receive() (*types.Transaction, error) {
	return _DelayedVetoable.Contract.Receive(&_DelayedVetoable.TransactOpts)
}

// DelayedVetoableForwardedIterator is returned from FilterForwarded and is used to iterate over the raw logs and unpacked data for Forwarded events raised by the DelayedVetoable contract.
type DelayedVetoableForwardedIterator struct {
	Event *DelayedVetoableForwarded // Event containing the contract specifics and raw log

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
func (it *DelayedVetoableForwardedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedVetoableForwarded)
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
		it.Event = new(DelayedVetoableForwarded)
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
func (it *DelayedVetoableForwardedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedVetoableForwardedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedVetoableForwarded represents a Forwarded event raised by the DelayedVetoable contract.
type DelayedVetoableForwarded struct {
	CallHash [32]byte
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterForwarded is a free log retrieval operation binding the contract event 0x4c109d85bcd0bb5c735b4be850953d652afe4cd9aa2e0b1426a65a4dcb2e1229.
//
// Solidity: event Forwarded(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) FilterForwarded(opts *bind.FilterOpts, callHash [][32]byte) (*DelayedVetoableForwardedIterator, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.FilterLogs(opts, "Forwarded", callHashRule)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableForwardedIterator{contract: _DelayedVetoable.contract, event: "Forwarded", logs: logs, sub: sub}, nil
}

// WatchForwarded is a free log subscription operation binding the contract event 0x4c109d85bcd0bb5c735b4be850953d652afe4cd9aa2e0b1426a65a4dcb2e1229.
//
// Solidity: event Forwarded(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) WatchForwarded(opts *bind.WatchOpts, sink chan<- *DelayedVetoableForwarded, callHash [][32]byte) (event.Subscription, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.WatchLogs(opts, "Forwarded", callHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedVetoableForwarded)
				if err := _DelayedVetoable.contract.UnpackLog(event, "Forwarded", log); err != nil {
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

// ParseForwarded is a log parse operation binding the contract event 0x4c109d85bcd0bb5c735b4be850953d652afe4cd9aa2e0b1426a65a4dcb2e1229.
//
// Solidity: event Forwarded(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) ParseForwarded(log types.Log) (*DelayedVetoableForwarded, error) {
	event := new(DelayedVetoableForwarded)
	if err := _DelayedVetoable.contract.UnpackLog(event, "Forwarded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedVetoableInitiatedIterator is returned from FilterInitiated and is used to iterate over the raw logs and unpacked data for Initiated events raised by the DelayedVetoable contract.
type DelayedVetoableInitiatedIterator struct {
	Event *DelayedVetoableInitiated // Event containing the contract specifics and raw log

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
func (it *DelayedVetoableInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedVetoableInitiated)
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
		it.Event = new(DelayedVetoableInitiated)
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
func (it *DelayedVetoableInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedVetoableInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedVetoableInitiated represents a Initiated event raised by the DelayedVetoable contract.
type DelayedVetoableInitiated struct {
	CallHash [32]byte
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterInitiated is a free log retrieval operation binding the contract event 0x87a332a414acbc7da074543639ce7ae02ff1ea72e88379da9f261b080beb5a13.
//
// Solidity: event Initiated(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) FilterInitiated(opts *bind.FilterOpts, callHash [][32]byte) (*DelayedVetoableInitiatedIterator, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.FilterLogs(opts, "Initiated", callHashRule)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableInitiatedIterator{contract: _DelayedVetoable.contract, event: "Initiated", logs: logs, sub: sub}, nil
}

// WatchInitiated is a free log subscription operation binding the contract event 0x87a332a414acbc7da074543639ce7ae02ff1ea72e88379da9f261b080beb5a13.
//
// Solidity: event Initiated(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) WatchInitiated(opts *bind.WatchOpts, sink chan<- *DelayedVetoableInitiated, callHash [][32]byte) (event.Subscription, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.WatchLogs(opts, "Initiated", callHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedVetoableInitiated)
				if err := _DelayedVetoable.contract.UnpackLog(event, "Initiated", log); err != nil {
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

// ParseInitiated is a log parse operation binding the contract event 0x87a332a414acbc7da074543639ce7ae02ff1ea72e88379da9f261b080beb5a13.
//
// Solidity: event Initiated(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) ParseInitiated(log types.Log) (*DelayedVetoableInitiated, error) {
	event := new(DelayedVetoableInitiated)
	if err := _DelayedVetoable.contract.UnpackLog(event, "Initiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelayedVetoableVetoedIterator is returned from FilterVetoed and is used to iterate over the raw logs and unpacked data for Vetoed events raised by the DelayedVetoable contract.
type DelayedVetoableVetoedIterator struct {
	Event *DelayedVetoableVetoed // Event containing the contract specifics and raw log

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
func (it *DelayedVetoableVetoedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelayedVetoableVetoed)
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
		it.Event = new(DelayedVetoableVetoed)
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
func (it *DelayedVetoableVetoedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelayedVetoableVetoedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelayedVetoableVetoed represents a Vetoed event raised by the DelayedVetoable contract.
type DelayedVetoableVetoed struct {
	CallHash [32]byte
	Data     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterVetoed is a free log retrieval operation binding the contract event 0xbede6852c1d97d93ff557f676de76670cd0dec861e7fe8beb13aa0ba2b0ab040.
//
// Solidity: event Vetoed(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) FilterVetoed(opts *bind.FilterOpts, callHash [][32]byte) (*DelayedVetoableVetoedIterator, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.FilterLogs(opts, "Vetoed", callHashRule)
	if err != nil {
		return nil, err
	}
	return &DelayedVetoableVetoedIterator{contract: _DelayedVetoable.contract, event: "Vetoed", logs: logs, sub: sub}, nil
}

// WatchVetoed is a free log subscription operation binding the contract event 0xbede6852c1d97d93ff557f676de76670cd0dec861e7fe8beb13aa0ba2b0ab040.
//
// Solidity: event Vetoed(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) WatchVetoed(opts *bind.WatchOpts, sink chan<- *DelayedVetoableVetoed, callHash [][32]byte) (event.Subscription, error) {

	var callHashRule []interface{}
	for _, callHashItem := range callHash {
		callHashRule = append(callHashRule, callHashItem)
	}

	logs, sub, err := _DelayedVetoable.contract.WatchLogs(opts, "Vetoed", callHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelayedVetoableVetoed)
				if err := _DelayedVetoable.contract.UnpackLog(event, "Vetoed", log); err != nil {
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

// ParseVetoed is a log parse operation binding the contract event 0xbede6852c1d97d93ff557f676de76670cd0dec861e7fe8beb13aa0ba2b0ab040.
//
// Solidity: event Vetoed(bytes32 indexed callHash, bytes data)
func (_DelayedVetoable *DelayedVetoableFilterer) ParseVetoed(log types.Log) (*DelayedVetoableVetoed, error) {
	event := new(DelayedVetoableVetoed)
	if err := _DelayedVetoable.contract.UnpackLog(event, "Vetoed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
