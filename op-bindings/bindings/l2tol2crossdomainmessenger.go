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

// L2ToL2CrossDomainMessengerMetaData contains all meta data concerning the L2ToL2CrossDomainMessenger contract.
var L2ToL2CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"CROSS_DOMAIN_MESSAGE_SENDER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"INITIAL_BALANCE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint248\",\"internalType\":\"uint248\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MESSAGE_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"crossL2Inbox\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messageNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"relayMessage\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"sendMessage\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"successfulMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"SentMessage\",\"inputs\":[{\"name\":\"destination\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"target\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"message\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":true}]",
	Bin: "0x608060405234801561000f575f80fd5b50610c2d8061001d5f395ff3fe608060405260043610610079575f3560e01c8063b1f35f2c1161004c578063b1f35f2c1461015d578063c155fa651461019e578063ecc70428146101bd578063fd2c723e146101f1575f80fd5b806314525bce1461007d5780633f827a5a146100e45780637056f41f1461010a578063b1b1b2091461011f575b5f80fd5b348015610088575f80fd5b506100af7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81565b6040517effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100ef575f80fd5b506100f75f81565b60405161ffff90911681526020016100db565b61011d610118366004610832565b610241565b005b34801561012a575f80fd5b5061014d6101393660046108b4565b60016020525f908152604090205460ff1681565b60405190151581526020016100db565b348015610168575f80fd5b506101907fb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f381565b6040519081526020016100db565b3480156101a9575f80fd5b5061011d6101b83660046108f8565b6103b0565b3480156101c8575f80fd5b506002547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff16610190565b3480156101fc575f80fd5b505f5461021c9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100db565b46840361024c575f80fd5b5f846102776002547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1690565b33863487876040516024016102929796959493929190610a47565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fc155fa65000000000000000000000000000000000000000000000000000000001790525190915061031f9086908690869086908690610b00565b60405180910390a0600280547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff16905f61035683610b53565b91906101000a8154817dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff02191690837dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff160217905550505050505050565b5f5473ffffffffffffffffffffffffffffffffffffffff163314610435576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f4e6f742063726f737320646f6d61696e206d657373656e67657200000000000060448201526064015b60405180910390fd5b5f54604080517f938b5f320000000000000000000000000000000000000000000000000000000081529051309273ffffffffffffffffffffffffffffffffffffffff169163938b5f329160048083019260209291908290030181865afa1580156104a1573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104c59190610bb4565b73ffffffffffffffffffffffffffffffffffffffff1614610568576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f4e6f742066726f6d2074686973204c32546f4c3243726f7373446f6d61696e4d60448201527f657373656e676572000000000000000000000000000000000000000000000000606482015260840161042c565b4686146105d1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f4e6f7420666f72207468697320636861696e0000000000000000000000000000604482015260640161042c565b3073ffffffffffffffffffffffffffffffffffffffff841603610650576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600d60248201527f556e736166652074617267657400000000000000000000000000000000000000604482015260640161042c565b5f86868686868660405160200161066c96959493929190610bd6565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291815281516020928301205f818152600190935291205490915060ff161561071b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f4d65737361676520616c72656164792072656c61796564000000000000000000604482015260640161042c565b847fb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f35d5f61074b855a86866107f6565b9050806107b4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600b60248201527f43616c6c206661696c6564000000000000000000000000000000000000000000604482015260640161042c565b505f90815260016020819052604090912080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169091179055505050505050565b5f805f80845160208601878a8af19695505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461082f575f80fd5b50565b5f805f8060608587031215610845575f80fd5b8435935060208501356108578161080e565b9250604085013567ffffffffffffffff80821115610873575f80fd5b818701915087601f830112610886575f80fd5b813581811115610894575f80fd5b8860208285010111156108a5575f80fd5b95989497505060200194505050565b5f602082840312156108c4575f80fd5b5035919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b5f805f805f8060c0878903121561090d575f80fd5b863595506020870135945060408701356109268161080e565b935060608701356109368161080e565b92506080870135915060a087013567ffffffffffffffff80821115610959575f80fd5b818901915089601f83011261096c575f80fd5b81358181111561097e5761097e6108cb565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156109c4576109c46108cb565b816040528281528c60208487010111156109dc575f80fd5b826020860160208301375f6020848301015280955050505050509295509295509295565b81835281816020850137505f602082840101525f60207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b8781528660208201525f73ffffffffffffffffffffffffffffffffffffffff808816604084015280871660608401525084608083015260c060a0830152610a9260c083018486610a00565b9998505050505050505050565b5f81518084525f5b81811015610ac357602081850181015186830182015201610aa7565b505f6020828601015260207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f83011685010191505092915050565b85815273ffffffffffffffffffffffffffffffffffffffff85166020820152608060408201525f610b35608083018587610a00565b8281036060840152610b478185610a9f565b98975050505050505050565b5f7dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808316818103610baa577f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b6001019392505050565b5f60208284031215610bc4575f80fd5b8151610bcf8161080e565b9392505050565b8681528560208201525f73ffffffffffffffffffffffffffffffffffffffff808716604084015280861660608401525083608083015260c060a0830152610b4760c0830184610a9f56fea164736f6c6343000818000a",
}

// L2ToL2CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ToL2CrossDomainMessengerMetaData.ABI instead.
var L2ToL2CrossDomainMessengerABI = L2ToL2CrossDomainMessengerMetaData.ABI

// L2ToL2CrossDomainMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2ToL2CrossDomainMessengerMetaData.Bin instead.
var L2ToL2CrossDomainMessengerBin = L2ToL2CrossDomainMessengerMetaData.Bin

// DeployL2ToL2CrossDomainMessenger deploys a new Ethereum contract, binding an instance of L2ToL2CrossDomainMessenger to it.
func DeployL2ToL2CrossDomainMessenger(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L2ToL2CrossDomainMessenger, error) {
	parsed, err := L2ToL2CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2ToL2CrossDomainMessengerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2ToL2CrossDomainMessenger{L2ToL2CrossDomainMessengerCaller: L2ToL2CrossDomainMessengerCaller{contract: contract}, L2ToL2CrossDomainMessengerTransactor: L2ToL2CrossDomainMessengerTransactor{contract: contract}, L2ToL2CrossDomainMessengerFilterer: L2ToL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// L2ToL2CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessenger struct {
	L2ToL2CrossDomainMessengerCaller     // Read-only binding to the contract
	L2ToL2CrossDomainMessengerTransactor // Write-only binding to the contract
	L2ToL2CrossDomainMessengerFilterer   // Log filterer for contract events
}

// L2ToL2CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2ToL2CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2ToL2CrossDomainMessengerSession struct {
	Contract     *L2ToL2CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2ToL2CrossDomainMessengerCallerSession struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// L2ToL2CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2ToL2CrossDomainMessengerTransactorSession struct {
	Contract     *L2ToL2CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerRaw struct {
	Contract *L2ToL2CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCallerRaw struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactorRaw struct {
	Contract *L2ToL2CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2ToL2CrossDomainMessenger creates a new instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*L2ToL2CrossDomainMessenger, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessenger{L2ToL2CrossDomainMessengerCaller: L2ToL2CrossDomainMessengerCaller{contract: contract}, L2ToL2CrossDomainMessengerTransactor: L2ToL2CrossDomainMessengerTransactor{contract: contract}, L2ToL2CrossDomainMessengerFilterer: L2ToL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewL2ToL2CrossDomainMessengerCaller creates a new read-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*L2ToL2CrossDomainMessengerCaller, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerCaller{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerTransactor creates a new write-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ToL2CrossDomainMessengerTransactor, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerFilterer creates a new log filterer instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L2ToL2CrossDomainMessengerFilterer, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindL2ToL2CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindL2ToL2CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ToL2CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CROSSDOMAINMESSAGESENDERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "CROSS_DOMAIN_MESSAGE_SENDER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CROSSDOMAINMESSAGESENDERSLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESENDERSLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CROSSDOMAINMESSAGESENDERSLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESENDERSLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// INITIALBALANCE is a free data retrieval call binding the contract method 0x14525bce.
//
// Solidity: function INITIAL_BALANCE() view returns(uint248)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) INITIALBALANCE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "INITIAL_BALANCE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// INITIALBALANCE is a free data retrieval call binding the contract method 0x14525bce.
//
// Solidity: function INITIAL_BALANCE() view returns(uint248)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) INITIALBALANCE() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.INITIALBALANCE(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// INITIALBALANCE is a free data retrieval call binding the contract method 0x14525bce.
//
// Solidity: function INITIAL_BALANCE() view returns(uint248)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) INITIALBALANCE() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.INITIALBALANCE(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossL2Inbox is a free data retrieval call binding the contract method 0xfd2c723e.
//
// Solidity: function crossL2Inbox() view returns(address)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossL2Inbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossL2Inbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossL2Inbox is a free data retrieval call binding the contract method 0xfd2c723e.
//
// Solidity: function crossL2Inbox() view returns(address)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossL2Inbox() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossL2Inbox(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossL2Inbox is a free data retrieval call binding the contract method 0xfd2c723e.
//
// Solidity: function crossL2Inbox() view returns(address)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossL2Inbox() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossL2Inbox(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xc155fa65.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _nonce, address _sender, address _target, uint256 _value, bytes _message) returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _destination *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "relayMessage", _destination, _nonce, _sender, _target, _value, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xc155fa65.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _nonce, address _sender, address _target, uint256 _value, bytes _message) returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) RelayMessage(_destination *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _nonce, _sender, _target, _value, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0xc155fa65.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _nonce, address _sender, address _target, uint256 _value, bytes _message) returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) RelayMessage(_destination *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _nonce, _sender, _target, _value, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "sendMessage", _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}
