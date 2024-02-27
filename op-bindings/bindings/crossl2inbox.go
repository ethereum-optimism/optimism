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

// CrossL2InboxIdentifier is an auto generated low-level Go binding around an user-defined struct.
type CrossL2InboxIdentifier struct {
	Origin      common.Address
	Blocknumber *big.Int
	LogIndex    *big.Int
	Timestamp   *big.Int
	ChainId     *big.Int
}

// CrossL2InboxMetaData contains all meta data concerning the CrossL2Inbox contract.
var CrossL2InboxMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_l1Block\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"BLOCKNUMBER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"CHAINID_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L1_BLOCK\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"LOG_INDEX_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ORIGIN_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"TIMESTAMP_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blocknumber\",\"inputs\":[],\"outputs\":[{\"name\":\"_blocknumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"chainId\",\"inputs\":[],\"outputs\":[{\"name\":\"_chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"executeMessage\",\"inputs\":[{\"name\":\"_msg\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_id\",\"type\":\"tuple\",\"internalType\":\"structCrossL2Inbox.Identifier\",\"components\":[{\"name\":\"origin\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"blocknumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"logIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"logIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"_logIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"origin\",\"inputs\":[],\"outputs\":[{\"name\":\"_origin\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"timestamp\",\"inputs\":[],\"outputs\":[{\"name\":\"_timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x60a060405234801561000f575f80fd5b506040516107ec3803806107ec83398101604081905261002e9161003f565b6001600160a01b031660805261006c565b5f6020828403121561004f575f80fd5b81516001600160a01b0381168114610065575f80fd5b9392505050565b60805161076161008b5f395f81816101e601526103e801526107615ff3fe6080604052600436106100c3575f3560e01c806379d6268011610071578063a19f75271161004c578063a19f7527146102c6578063b80777ea146102db578063da99f7291461030e575f80fd5b806379d626801461022d578063938b5f32146102605780639a8a059214610293575f80fd5b8063122f8b66116100a1578063122f8b661461016f578063260e6413146101a257806347718590146101d5575f80fd5b806305062247146100c757806307049933146101095780630f04cf1b1461013c575b5f80fd5b3480156100d2575f80fd5b507f5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc5c5b6040519081526020015b60405180910390f35b348015610114575f80fd5b506100f67f6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d31381565b348015610147575f80fd5b506100f67f5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc81565b34801561017a575f80fd5b506100f67f2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e0281565b3480156101ad575f80fd5b506100f67fd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c81581565b3480156101e0575f80fd5b506102087f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610100565b348015610238575f80fd5b506100f67fab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d81565b34801561026b575f80fd5b507fd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c8155c610208565b34801561029e575f80fd5b507f6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d3135c6100f6565b6102d96102d436600461065e565b610341565b005b3480156102e6575f80fd5b507f2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e025c6100f6565b348015610319575f80fd5b507fab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d5c6100f6565b42826060013511156103b4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601160248201527f496e76616c69642074696d657374616d7000000000000000000000000000000060448201526064015b60405180910390fd5b60808201516040517fe38bbc32000000000000000000000000000000000000000000000000000000008152600481018290527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff169063e38bbc3290602401602060405180830381865afa158015610442573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610466919061072e565b6104cc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600f60248201527f496e76616c696420636861696e4964000000000000000000000000000000000060448201526064016103ab565b333214610535576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600760248201527f4e6f7420454f410000000000000000000000000000000000000000000000000060448201526064016103ab565b82517fd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c8155d60208301517f5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc5d60408301517fab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d5d60608301517f2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e025d807f6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d3135d5f610633835a3489898080601f0160208091040260200160405190810160405280939291908181526020018383808284375f9201919091525061064692505050565b90508061063e575f80fd5b505050505050565b5f805f80845160208601878a8af19695505050505050565b5f805f8084860360e0811215610672575f80fd5b853567ffffffffffffffff80821115610689575f80fd5b818801915088601f83011261069c575f80fd5b8135818111156106aa575f80fd5b8960208285010111156106bb575f80fd5b60208301975080965050505060a07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0820112156106f6575f80fd5b5060208501915060c085013573ffffffffffffffffffffffffffffffffffffffff81168114610723575f80fd5b939692955090935050565b5f6020828403121561073e575f80fd5b8151801515811461074d575f80fd5b939250505056fea164736f6c6343000818000a",
}

// CrossL2InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2InboxMetaData.ABI instead.
var CrossL2InboxABI = CrossL2InboxMetaData.ABI

// CrossL2InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2InboxMetaData.Bin instead.
var CrossL2InboxBin = CrossL2InboxMetaData.Bin

// DeployCrossL2Inbox deploys a new Ethereum contract, binding an instance of CrossL2Inbox to it.
func DeployCrossL2Inbox(auth *bind.TransactOpts, backend bind.ContractBackend, _l1Block common.Address) (common.Address, *types.Transaction, *CrossL2Inbox, error) {
	parsed, err := CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2InboxBin), backend, _l1Block)
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

// BLOCKNUMBERSLOT is a free data retrieval call binding the contract method 0x0f04cf1b.
//
// Solidity: function BLOCKNUMBER_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) BLOCKNUMBERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "BLOCKNUMBER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// BLOCKNUMBERSLOT is a free data retrieval call binding the contract method 0x0f04cf1b.
//
// Solidity: function BLOCKNUMBER_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) BLOCKNUMBERSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.BLOCKNUMBERSLOT(&_CrossL2Inbox.CallOpts)
}

// BLOCKNUMBERSLOT is a free data retrieval call binding the contract method 0x0f04cf1b.
//
// Solidity: function BLOCKNUMBER_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) BLOCKNUMBERSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.BLOCKNUMBERSLOT(&_CrossL2Inbox.CallOpts)
}

// CHAINIDSLOT is a free data retrieval call binding the contract method 0x07049933.
//
// Solidity: function CHAINID_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) CHAINIDSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "CHAINID_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CHAINIDSLOT is a free data retrieval call binding the contract method 0x07049933.
//
// Solidity: function CHAINID_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) CHAINIDSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.CHAINIDSLOT(&_CrossL2Inbox.CallOpts)
}

// CHAINIDSLOT is a free data retrieval call binding the contract method 0x07049933.
//
// Solidity: function CHAINID_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) CHAINIDSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.CHAINIDSLOT(&_CrossL2Inbox.CallOpts)
}

// L1BLOCK is a free data retrieval call binding the contract method 0x47718590.
//
// Solidity: function L1_BLOCK() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) L1BLOCK(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "L1_BLOCK")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BLOCK is a free data retrieval call binding the contract method 0x47718590.
//
// Solidity: function L1_BLOCK() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) L1BLOCK() (common.Address, error) {
	return _CrossL2Inbox.Contract.L1BLOCK(&_CrossL2Inbox.CallOpts)
}

// L1BLOCK is a free data retrieval call binding the contract method 0x47718590.
//
// Solidity: function L1_BLOCK() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) L1BLOCK() (common.Address, error) {
	return _CrossL2Inbox.Contract.L1BLOCK(&_CrossL2Inbox.CallOpts)
}

// LOGINDEXSLOT is a free data retrieval call binding the contract method 0x79d62680.
//
// Solidity: function LOG_INDEX_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) LOGINDEXSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "LOG_INDEX_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// LOGINDEXSLOT is a free data retrieval call binding the contract method 0x79d62680.
//
// Solidity: function LOG_INDEX_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) LOGINDEXSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.LOGINDEXSLOT(&_CrossL2Inbox.CallOpts)
}

// LOGINDEXSLOT is a free data retrieval call binding the contract method 0x79d62680.
//
// Solidity: function LOG_INDEX_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) LOGINDEXSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.LOGINDEXSLOT(&_CrossL2Inbox.CallOpts)
}

// ORIGINSLOT is a free data retrieval call binding the contract method 0x260e6413.
//
// Solidity: function ORIGIN_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) ORIGINSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "ORIGIN_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ORIGINSLOT is a free data retrieval call binding the contract method 0x260e6413.
//
// Solidity: function ORIGIN_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) ORIGINSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.ORIGINSLOT(&_CrossL2Inbox.CallOpts)
}

// ORIGINSLOT is a free data retrieval call binding the contract method 0x260e6413.
//
// Solidity: function ORIGIN_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) ORIGINSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.ORIGINSLOT(&_CrossL2Inbox.CallOpts)
}

// TIMESTAMPSLOT is a free data retrieval call binding the contract method 0x122f8b66.
//
// Solidity: function TIMESTAMP_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) TIMESTAMPSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "TIMESTAMP_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// TIMESTAMPSLOT is a free data retrieval call binding the contract method 0x122f8b66.
//
// Solidity: function TIMESTAMP_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) TIMESTAMPSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.TIMESTAMPSLOT(&_CrossL2Inbox.CallOpts)
}

// TIMESTAMPSLOT is a free data retrieval call binding the contract method 0x122f8b66.
//
// Solidity: function TIMESTAMP_SLOT() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) TIMESTAMPSLOT() ([32]byte, error) {
	return _CrossL2Inbox.Contract.TIMESTAMPSLOT(&_CrossL2Inbox.CallOpts)
}

// Blocknumber is a free data retrieval call binding the contract method 0x05062247.
//
// Solidity: function blocknumber() view returns(uint256 _blocknumber)
func (_CrossL2Inbox *CrossL2InboxCaller) Blocknumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "blocknumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Blocknumber is a free data retrieval call binding the contract method 0x05062247.
//
// Solidity: function blocknumber() view returns(uint256 _blocknumber)
func (_CrossL2Inbox *CrossL2InboxSession) Blocknumber() (*big.Int, error) {
	return _CrossL2Inbox.Contract.Blocknumber(&_CrossL2Inbox.CallOpts)
}

// Blocknumber is a free data retrieval call binding the contract method 0x05062247.
//
// Solidity: function blocknumber() view returns(uint256 _blocknumber)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Blocknumber() (*big.Int, error) {
	return _CrossL2Inbox.Contract.Blocknumber(&_CrossL2Inbox.CallOpts)
}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint256 _chainId)
func (_CrossL2Inbox *CrossL2InboxCaller) ChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "chainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint256 _chainId)
func (_CrossL2Inbox *CrossL2InboxSession) ChainId() (*big.Int, error) {
	return _CrossL2Inbox.Contract.ChainId(&_CrossL2Inbox.CallOpts)
}

// ChainId is a free data retrieval call binding the contract method 0x9a8a0592.
//
// Solidity: function chainId() view returns(uint256 _chainId)
func (_CrossL2Inbox *CrossL2InboxCallerSession) ChainId() (*big.Int, error) {
	return _CrossL2Inbox.Contract.ChainId(&_CrossL2Inbox.CallOpts)
}

// LogIndex is a free data retrieval call binding the contract method 0xda99f729.
//
// Solidity: function logIndex() view returns(uint256 _logIndex)
func (_CrossL2Inbox *CrossL2InboxCaller) LogIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "logIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LogIndex is a free data retrieval call binding the contract method 0xda99f729.
//
// Solidity: function logIndex() view returns(uint256 _logIndex)
func (_CrossL2Inbox *CrossL2InboxSession) LogIndex() (*big.Int, error) {
	return _CrossL2Inbox.Contract.LogIndex(&_CrossL2Inbox.CallOpts)
}

// LogIndex is a free data retrieval call binding the contract method 0xda99f729.
//
// Solidity: function logIndex() view returns(uint256 _logIndex)
func (_CrossL2Inbox *CrossL2InboxCallerSession) LogIndex() (*big.Int, error) {
	return _CrossL2Inbox.Contract.LogIndex(&_CrossL2Inbox.CallOpts)
}

// Origin is a free data retrieval call binding the contract method 0x938b5f32.
//
// Solidity: function origin() view returns(address _origin)
func (_CrossL2Inbox *CrossL2InboxCaller) Origin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "origin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Origin is a free data retrieval call binding the contract method 0x938b5f32.
//
// Solidity: function origin() view returns(address _origin)
func (_CrossL2Inbox *CrossL2InboxSession) Origin() (common.Address, error) {
	return _CrossL2Inbox.Contract.Origin(&_CrossL2Inbox.CallOpts)
}

// Origin is a free data retrieval call binding the contract method 0x938b5f32.
//
// Solidity: function origin() view returns(address _origin)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Origin() (common.Address, error) {
	return _CrossL2Inbox.Contract.Origin(&_CrossL2Inbox.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256 _timestamp)
func (_CrossL2Inbox *CrossL2InboxCaller) Timestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "timestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256 _timestamp)
func (_CrossL2Inbox *CrossL2InboxSession) Timestamp() (*big.Int, error) {
	return _CrossL2Inbox.Contract.Timestamp(&_CrossL2Inbox.CallOpts)
}

// Timestamp is a free data retrieval call binding the contract method 0xb80777ea.
//
// Solidity: function timestamp() view returns(uint256 _timestamp)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Timestamp() (*big.Int, error) {
	return _CrossL2Inbox.Contract.Timestamp(&_CrossL2Inbox.CallOpts)
}

// ExecuteMessage is a paid mutator transaction binding the contract method 0xa19f7527.
//
// Solidity: function executeMessage(bytes _msg, (address,uint256,uint256,uint256,uint256) _id, address _target) payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) ExecuteMessage(opts *bind.TransactOpts, _msg []byte, _id CrossL2InboxIdentifier, _target common.Address) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "executeMessage", _msg, _id, _target)
}

// ExecuteMessage is a paid mutator transaction binding the contract method 0xa19f7527.
//
// Solidity: function executeMessage(bytes _msg, (address,uint256,uint256,uint256,uint256) _id, address _target) payable returns()
func (_CrossL2Inbox *CrossL2InboxSession) ExecuteMessage(_msg []byte, _id CrossL2InboxIdentifier, _target common.Address) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.ExecuteMessage(&_CrossL2Inbox.TransactOpts, _msg, _id, _target)
}

// ExecuteMessage is a paid mutator transaction binding the contract method 0xa19f7527.
//
// Solidity: function executeMessage(bytes _msg, (address,uint256,uint256,uint256,uint256) _id, address _target) payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) ExecuteMessage(_msg []byte, _id CrossL2InboxIdentifier, _target common.Address) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.ExecuteMessage(&_CrossL2Inbox.TransactOpts, _msg, _id, _target)
}
