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

// InboxEntry is an auto generated low-level Go binding around an user-defined struct.
type InboxEntry struct {
	Chain  [32]byte
	Output [32]byte
}

// TypesSuperchainMessage is an auto generated low-level Go binding around an user-defined struct.
type TypesSuperchainMessage struct {
	Nonce       *big.Int
	SourceChain [32]byte
	TargetChain [32]byte
	From        common.Address
	To          common.Address
	Value       *big.Int
	GasLimit    *big.Int
	Data        []byte
}

// CrossL2InboxMetaData contains all meta data concerning the CrossL2Inbox contract.
var CrossL2InboxMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_superchainPostie\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"consumedMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"crossL2Sender\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deliverMail\",\"inputs\":[{\"name\":\"mail\",\"type\":\"tuple[]\",\"internalType\":\"structInboxEntry[]\",\"components\":[{\"name\":\"chain\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"output\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"messageSourceChain\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"roots\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"runCrossL2Message\",\"inputs\":[{\"name\":\"_msg\",\"type\":\"tuple\",\"internalType\":\"structTypes.SuperchainMessage\",\"components\":[{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sourceChain\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"targetChain\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"_l2OutputRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_inclusionProof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"superchainPostie\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"CrossL2MessageRelayed\",\"inputs\":[{\"name\":\"messageRoot\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"success\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false}]",
	Bin: "0x60a0604052600180546001600160a01b03191661dead17905534801561002457600080fd5b50604051610f6d380380610f6d83398101604081905261004391610054565b6001600160a01b0316608052610084565b60006020828403121561006657600080fd5b81516001600160a01b038116811461007d57600080fd5b9392505050565b608051610ec76100a6600039600081816101cd015261074e0152610ec76000f3fe60806040526004361061007f5760003560e01c806360a687aa1161004e57806360a687aa1461019a578063c00157da146101be578063db10b9a9146101f1578063f2c5dcb81461022957600080fd5b806339bc3c811461008b5780633d6d0dd4146100d057806354fd4d50146101225780635d9eb6d01461017857600080fd5b3661008657005b600080fd5b34801561009757600080fd5b506100bb6100a6366004610aa0565b60036020526000908152604090205460ff1681565b60405190151581526020015b60405180910390f35b3480156100dc57600080fd5b506001546100fd9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100c7565b34801561012e57600080fd5b5061016b6040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b6040516100c79190610ab9565b34801561018457600080fd5b50610198610193366004610ca2565b610249565b005b3480156101a657600080fd5b506101b060025481565b6040519081526020016100c7565b3480156101ca57600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006100fd565b3480156101fd57600080fd5b506100bb61020c366004610d95565b600060208181529281526040808220909352908152205460ff1681565b34801561023557600080fd5b50610198610244366004610db7565b610736565b604084015146146102e1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f43726f73734c32496e626f783a205f6d73672e746172676574436861696e206460448201527f6f65736e2774206d6174636820626c6f636b2e636861696e696400000000000060648201526084015b60405180910390fd5b60208085015160009081528082526040808220868352909252205460ff166103b1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604c60248201527f43726f73734c32496e626f783a206d7573742070726f6f6620616761696e737460448201527f206b6e6f776e206f757470757420726f6f742066726f6d206d6573736167652060648201527f736f7572636520636861696e0000000000000000000000000000000000000000608482015260a4016102d8565b60006103bc85610890565b905080600052600060205260406000208060005250602060002060405180600182536001828101889052602060218401526041830184905260618301819052608183015260a190910190848683379084018181039250906103e88284836021620186a0fa92508261042d576103e882fd5b600052505060015473ffffffffffffffffffffffffffffffffffffffff1661dead146104db576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603c60248201527f43726f73734c32496e626f783a2063616e206f6e6c792074726967676572206f60448201527f6e652063616c6c207065722063726f7373204c32206d6573736167650000000060648201526084016102d8565b60008181526003602052604090205460ff161561057a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f43726f73734c32496e626f783a206d6573736167652068617320616c7265616460448201527f79206265656e20636f6e73756d6564000000000000000000000000000000000060648201526084016102d8565b60008181526003602090815260408220805460017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff009091168117909155606088015181547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff909116179055860151600255608086015160c087015160a088015160e089015161062193929190610a24565b600180547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055600060025560405190915082907f608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d9061068b90841515815260200190565b60405180910390a2801580156106a15750326001145b1561072e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f43726f73734c32496e626f783a2063726f7373204c32206d657373616765206360448201527f616c6c20657865637574696f6e206661696c656400000000000000000000000060648201526084016102d8565b505050505050565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016146107fb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f43726f73734c32496e626f783a206f6e6c7920706f737469652063616e20646560448201527f6c69766572206d61696c0000000000000000000000000000000000000000000060648201526084016102d8565b60005b8181101561088b57600160008085858581811061081d5761081d610e2c565b905060400201600001358152602001908152602001600020600085858581811061084957610849610e2c565b90506040020160200135815260200190815260200160002060006101000a81548160ff021916908315150217905550808061088390610e5b565b9150506107fe565b505050565b60e0810151805160209182018190206040805193840182905283019190915260009182906060016040516020818303038152906040528051906020012090506000846060015185608001518660a001518760c00151604051602001610929949392919073ffffffffffffffffffffffffffffffffffffffff94851681529290931660208301526040820152606081019190915260800190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152828252805160209182012081840152828201949094528051808303820181526060830182528051908501208785015197820151608084019890985260a0808401989098528151808403909801885260c08301825287519785019790972060e0830152610100808301979097528051808303909701875261012090910190525083519301929092207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001792915050565b6000806000610a34866000610a82565b905080610a6a576308c379a06000526020805278185361666543616c6c3a204e6f7420656e6f756768206761736058526064601cfd5b600080855160208701888b5af1979650505050505050565b600080603f83619c4001026040850201603f5a021015949350505050565b600060208284031215610ab257600080fd5b5035919050565b600060208083528351808285015260005b81811015610ae657858101830151858201604001528201610aca565b81811115610af8576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051610100810167ffffffffffffffff81118282101715610b7f57610b7f610b2c565b60405290565b803573ffffffffffffffffffffffffffffffffffffffff81168114610ba957600080fd5b919050565b600082601f830112610bbf57600080fd5b813567ffffffffffffffff80821115610bda57610bda610b2c565b604051601f83017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908282118183101715610c2057610c20610b2c565b81604052838152866020858801011115610c3957600080fd5b836020870160208301376000602085830101528094505050505092915050565b60008083601f840112610c6b57600080fd5b50813567ffffffffffffffff811115610c8357600080fd5b602083019150836020828501011115610c9b57600080fd5b9250929050565b60008060008060608587031215610cb857600080fd5b843567ffffffffffffffff80821115610cd057600080fd5b908601906101008289031215610ce557600080fd5b610ced610b5b565b823581526020830135602082015260408301356040820152610d1160608401610b85565b6060820152610d2260808401610b85565b608082015260a083013560a082015260c083013560c082015260e083013582811115610d4d57600080fd5b610d598a828601610bae565b60e0830152509550602087013594506040870135915080821115610d7c57600080fd5b50610d8987828801610c59565b95989497509550505050565b60008060408385031215610da857600080fd5b50508035926020909101359150565b60008060208385031215610dca57600080fd5b823567ffffffffffffffff80821115610de257600080fd5b818501915085601f830112610df657600080fd5b813581811115610e0557600080fd5b8660208260061b8501011115610e1a57600080fd5b60209290920196919550909350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610eb3577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b506001019056fea164736f6c634300080f000a",
}

// CrossL2InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2InboxMetaData.ABI instead.
var CrossL2InboxABI = CrossL2InboxMetaData.ABI

// CrossL2InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2InboxMetaData.Bin instead.
var CrossL2InboxBin = CrossL2InboxMetaData.Bin

// DeployCrossL2Inbox deploys a new Ethereum contract, binding an instance of CrossL2Inbox to it.
func DeployCrossL2Inbox(auth *bind.TransactOpts, backend bind.ContractBackend, _superchainPostie common.Address) (common.Address, *types.Transaction, *CrossL2Inbox, error) {
	parsed, err := CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2InboxBin), backend, _superchainPostie)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// CrossL2Inbox is an auto generated Go binding around an Ethereum contract.
type CrossL2Inbox struct {
	CrossL2InboxCaller     // Read-only binding to the contract
	CrossL2InboxTransactor // Write-only binding to the contract
	CrossL2InboxFilterer   // Log filterer for contract events
}

// CrossL2InboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrossL2InboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrossL2InboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrossL2InboxSession struct {
	Contract     *CrossL2Inbox     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CrossL2InboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrossL2InboxCallerSession struct {
	Contract *CrossL2InboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// CrossL2InboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrossL2InboxTransactorSession struct {
	Contract     *CrossL2InboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CrossL2InboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrossL2InboxRaw struct {
	Contract *CrossL2Inbox // Generic contract binding to access the raw methods on
}

// CrossL2InboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrossL2InboxCallerRaw struct {
	Contract *CrossL2InboxCaller // Generic read-only contract binding to access the raw methods on
}

// CrossL2InboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactorRaw struct {
	Contract *CrossL2InboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrossL2Inbox creates a new instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2Inbox(address common.Address, backend bind.ContractBackend) (*CrossL2Inbox, error) {
	contract, err := bindCrossL2Inbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// NewCrossL2InboxCaller creates a new read-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxCaller(address common.Address, caller bind.ContractCaller) (*CrossL2InboxCaller, error) {
	contract, err := bindCrossL2Inbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxCaller{contract: contract}, nil
}

// NewCrossL2InboxTransactor creates a new write-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxTransactor(address common.Address, transactor bind.ContractTransactor) (*CrossL2InboxTransactor, error) {
	contract, err := bindCrossL2Inbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxTransactor{contract: contract}, nil
}

// NewCrossL2InboxFilterer creates a new log filterer instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxFilterer(address common.Address, filterer bind.ContractFilterer) (*CrossL2InboxFilterer, error) {
	contract, err := bindCrossL2Inbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxFilterer{contract: contract}, nil
}

// bindCrossL2Inbox binds a generic wrapper to an already deployed contract.
func bindCrossL2Inbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrossL2InboxABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.CrossL2InboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transact(opts, method, params...)
}

// ConsumedMessages is a free data retrieval call binding the contract method 0x39bc3c81.
//
// Solidity: function consumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCaller) ConsumedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "consumedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ConsumedMessages is a free data retrieval call binding the contract method 0x39bc3c81.
//
// Solidity: function consumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxSession) ConsumedMessages(arg0 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.ConsumedMessages(&_CrossL2Inbox.CallOpts, arg0)
}

// ConsumedMessages is a free data retrieval call binding the contract method 0x39bc3c81.
//
// Solidity: function consumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCallerSession) ConsumedMessages(arg0 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.ConsumedMessages(&_CrossL2Inbox.CallOpts, arg0)
}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) CrossL2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "crossL2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) CrossL2Sender() (common.Address, error) {
	return _CrossL2Inbox.Contract.CrossL2Sender(&_CrossL2Inbox.CallOpts)
}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) CrossL2Sender() (common.Address, error) {
	return _CrossL2Inbox.Contract.CrossL2Sender(&_CrossL2Inbox.CallOpts)
}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) MessageSourceChain(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "messageSourceChain")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) MessageSourceChain() ([32]byte, error) {
	return _CrossL2Inbox.Contract.MessageSourceChain(&_CrossL2Inbox.CallOpts)
}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) MessageSourceChain() ([32]byte, error) {
	return _CrossL2Inbox.Contract.MessageSourceChain(&_CrossL2Inbox.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCaller) Roots(opts *bind.CallOpts, arg0 [32]byte, arg1 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "roots", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) SuperchainPostie(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "superchainPostie")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) SuperchainPostie() (common.Address, error) {
	return _CrossL2Inbox.Contract.SuperchainPostie(&_CrossL2Inbox.CallOpts)
}

// SuperchainPostie is a free data retrieval call binding the contract method 0xc00157da.
//
// Solidity: function superchainPostie() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) SuperchainPostie() (common.Address, error) {
	return _CrossL2Inbox.Contract.SuperchainPostie(&_CrossL2Inbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) DeliverMail(opts *bind.TransactOpts, mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "deliverMail", mail)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxSession) DeliverMail(mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMail(&_CrossL2Inbox.TransactOpts, mail)
}

// DeliverMail is a paid mutator transaction binding the contract method 0xf2c5dcb8.
//
// Solidity: function deliverMail((bytes32,bytes32)[] mail) returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) DeliverMail(mail []InboxEntry) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMail(&_CrossL2Inbox.TransactOpts, mail)
}

// RunCrossL2Message is a paid mutator transaction binding the contract method 0x5d9eb6d0.
//
// Solidity: function runCrossL2Message((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg, bytes32 _l2OutputRoot, bytes _inclusionProof) returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) RunCrossL2Message(opts *bind.TransactOpts, _msg TypesSuperchainMessage, _l2OutputRoot [32]byte, _inclusionProof []byte) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "runCrossL2Message", _msg, _l2OutputRoot, _inclusionProof)
}

// RunCrossL2Message is a paid mutator transaction binding the contract method 0x5d9eb6d0.
//
// Solidity: function runCrossL2Message((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg, bytes32 _l2OutputRoot, bytes _inclusionProof) returns()
func (_CrossL2Inbox *CrossL2InboxSession) RunCrossL2Message(_msg TypesSuperchainMessage, _l2OutputRoot [32]byte, _inclusionProof []byte) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.RunCrossL2Message(&_CrossL2Inbox.TransactOpts, _msg, _l2OutputRoot, _inclusionProof)
}

// RunCrossL2Message is a paid mutator transaction binding the contract method 0x5d9eb6d0.
//
// Solidity: function runCrossL2Message((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg, bytes32 _l2OutputRoot, bytes _inclusionProof) returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) RunCrossL2Message(_msg TypesSuperchainMessage, _l2OutputRoot [32]byte, _inclusionProof []byte) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.RunCrossL2Message(&_CrossL2Inbox.TransactOpts, _msg, _l2OutputRoot, _inclusionProof)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_CrossL2Inbox *CrossL2InboxSession) Receive() (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.Receive(&_CrossL2Inbox.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) Receive() (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.Receive(&_CrossL2Inbox.TransactOpts)
}

// CrossL2InboxCrossL2MessageRelayedIterator is returned from FilterCrossL2MessageRelayed and is used to iterate over the raw logs and unpacked data for CrossL2MessageRelayed events raised by the CrossL2Inbox contract.
type CrossL2InboxCrossL2MessageRelayedIterator struct {
	Event *CrossL2InboxCrossL2MessageRelayed // Event containing the contract specifics and raw log

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
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossL2InboxCrossL2MessageRelayed)
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
		it.Event = new(CrossL2InboxCrossL2MessageRelayed)
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
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossL2InboxCrossL2MessageRelayed represents a CrossL2MessageRelayed event raised by the CrossL2Inbox contract.
type CrossL2InboxCrossL2MessageRelayed struct {
	MessageRoot [32]byte
	Success     bool
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterCrossL2MessageRelayed is a free log retrieval operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) FilterCrossL2MessageRelayed(opts *bind.FilterOpts, messageRoot [][32]byte) (*CrossL2InboxCrossL2MessageRelayedIterator, error) {

	var messageRootRule []interface{}
	for _, messageRootItem := range messageRoot {
		messageRootRule = append(messageRootRule, messageRootItem)
	}

	logs, sub, err := _CrossL2Inbox.contract.FilterLogs(opts, "CrossL2MessageRelayed", messageRootRule)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxCrossL2MessageRelayedIterator{contract: _CrossL2Inbox.contract, event: "CrossL2MessageRelayed", logs: logs, sub: sub}, nil
}

// WatchCrossL2MessageRelayed is a free log subscription operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) WatchCrossL2MessageRelayed(opts *bind.WatchOpts, sink chan<- *CrossL2InboxCrossL2MessageRelayed, messageRoot [][32]byte) (event.Subscription, error) {

	var messageRootRule []interface{}
	for _, messageRootItem := range messageRoot {
		messageRootRule = append(messageRootRule, messageRootItem)
	}

	logs, sub, err := _CrossL2Inbox.contract.WatchLogs(opts, "CrossL2MessageRelayed", messageRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossL2InboxCrossL2MessageRelayed)
				if err := _CrossL2Inbox.contract.UnpackLog(event, "CrossL2MessageRelayed", log); err != nil {
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

// ParseCrossL2MessageRelayed is a log parse operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) ParseCrossL2MessageRelayed(log types.Log) (*CrossL2InboxCrossL2MessageRelayed, error) {
	event := new(CrossL2InboxCrossL2MessageRelayed)
	if err := _CrossL2Inbox.contract.UnpackLog(event, "CrossL2MessageRelayed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
