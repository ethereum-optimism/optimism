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
	_ = abi.ConvertType
)

// MockAttestationDisputeGameMetaData contains all meta data concerning the MockAttestationDisputeGame contract.
var MockAttestationDisputeGameMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"Claim\",\"name\":\"_rootClaim\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_creator\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"L2_BLOCK_NUMBER\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ROOT_CLAIM\",\"outputs\":[{\"internalType\":\"Claim\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"challenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"challenges\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60c060405234801561001057600080fd5b506040516102b33803806102b383398101604081905261002f9161005f565b60809290925260a0526001600160a01b03166000908152602081905260409020805460ff191660011790556100a5565b60008060006060848603121561007457600080fd5b83516020850151604086015191945092506001600160a01b038116811461009a57600080fd5b809150509250925092565b60805160a0516101eb6100c86000396000608e0152600060c301526101eb6000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c806308b43a1914610051578063326f8195146100895780634a1890f0146100be578063b8b9c188146100e5575b600080fd5b61007461005f366004610113565b60006020819052908152604090205460ff1681565b60405190151581526020015b60405180910390f35b6100b07f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610080565b6100b07f000000000000000000000000000000000000000000000000000000000000000081565b6101116100f3366004610143565b5050336000908152602081905260409020805460ff19166001179055565b005b60006020828403121561012557600080fd5b81356001600160a01b038116811461013c57600080fd5b9392505050565b6000806020838503121561015657600080fd5b823567ffffffffffffffff8082111561016e57600080fd5b818501915085601f83011261018257600080fd5b81358181111561019157600080fd5b8660208285010111156101a357600080fd5b6020929092019691955090935050505056fea2646970667358221220a15f2b475fb3640846a10e20b1d980b5a93201e53321f773b79c483277a32aa364736f6c63430008130033",
}

// MockAttestationDisputeGameABI is the input ABI used to generate the binding from.
// Deprecated: Use MockAttestationDisputeGameMetaData.ABI instead.
var MockAttestationDisputeGameABI = MockAttestationDisputeGameMetaData.ABI

// MockAttestationDisputeGameBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockAttestationDisputeGameMetaData.Bin instead.
var MockAttestationDisputeGameBin = MockAttestationDisputeGameMetaData.Bin

// DeployMockAttestationDisputeGame deploys a new Ethereum contract, binding an instance of MockAttestationDisputeGame to it.
func DeployMockAttestationDisputeGame(auth *bind.TransactOpts, backend bind.ContractBackend, _rootClaim [32]byte, l2BlockNumber *big.Int, _creator common.Address) (common.Address, *types.Transaction, *MockAttestationDisputeGame, error) {
	parsed, err := MockAttestationDisputeGameMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockAttestationDisputeGameBin), backend, _rootClaim, l2BlockNumber, _creator)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockAttestationDisputeGame{MockAttestationDisputeGameCaller: MockAttestationDisputeGameCaller{contract: contract}, MockAttestationDisputeGameTransactor: MockAttestationDisputeGameTransactor{contract: contract}, MockAttestationDisputeGameFilterer: MockAttestationDisputeGameFilterer{contract: contract}}, nil
}

// MockAttestationDisputeGame is an auto generated Go binding around an Ethereum contract.
type MockAttestationDisputeGame struct {
	MockAttestationDisputeGameCaller     // Read-only binding to the contract
	MockAttestationDisputeGameTransactor // Write-only binding to the contract
	MockAttestationDisputeGameFilterer   // Log filterer for contract events
}

// MockAttestationDisputeGameCaller is an auto generated read-only Go binding around an Ethereum contract.
type MockAttestationDisputeGameCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockAttestationDisputeGameTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MockAttestationDisputeGameTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockAttestationDisputeGameFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockAttestationDisputeGameFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockAttestationDisputeGameSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockAttestationDisputeGameSession struct {
	Contract     *MockAttestationDisputeGame // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// MockAttestationDisputeGameCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockAttestationDisputeGameCallerSession struct {
	Contract *MockAttestationDisputeGameCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// MockAttestationDisputeGameTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockAttestationDisputeGameTransactorSession struct {
	Contract     *MockAttestationDisputeGameTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// MockAttestationDisputeGameRaw is an auto generated low-level Go binding around an Ethereum contract.
type MockAttestationDisputeGameRaw struct {
	Contract *MockAttestationDisputeGame // Generic contract binding to access the raw methods on
}

// MockAttestationDisputeGameCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockAttestationDisputeGameCallerRaw struct {
	Contract *MockAttestationDisputeGameCaller // Generic read-only contract binding to access the raw methods on
}

// MockAttestationDisputeGameTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockAttestationDisputeGameTransactorRaw struct {
	Contract *MockAttestationDisputeGameTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMockAttestationDisputeGame creates a new instance of MockAttestationDisputeGame, bound to a specific deployed contract.
func NewMockAttestationDisputeGame(address common.Address, backend bind.ContractBackend) (*MockAttestationDisputeGame, error) {
	contract, err := bindMockAttestationDisputeGame(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockAttestationDisputeGame{MockAttestationDisputeGameCaller: MockAttestationDisputeGameCaller{contract: contract}, MockAttestationDisputeGameTransactor: MockAttestationDisputeGameTransactor{contract: contract}, MockAttestationDisputeGameFilterer: MockAttestationDisputeGameFilterer{contract: contract}}, nil
}

// NewMockAttestationDisputeGameCaller creates a new read-only instance of MockAttestationDisputeGame, bound to a specific deployed contract.
func NewMockAttestationDisputeGameCaller(address common.Address, caller bind.ContractCaller) (*MockAttestationDisputeGameCaller, error) {
	contract, err := bindMockAttestationDisputeGame(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockAttestationDisputeGameCaller{contract: contract}, nil
}

// NewMockAttestationDisputeGameTransactor creates a new write-only instance of MockAttestationDisputeGame, bound to a specific deployed contract.
func NewMockAttestationDisputeGameTransactor(address common.Address, transactor bind.ContractTransactor) (*MockAttestationDisputeGameTransactor, error) {
	contract, err := bindMockAttestationDisputeGame(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockAttestationDisputeGameTransactor{contract: contract}, nil
}

// NewMockAttestationDisputeGameFilterer creates a new log filterer instance of MockAttestationDisputeGame, bound to a specific deployed contract.
func NewMockAttestationDisputeGameFilterer(address common.Address, filterer bind.ContractFilterer) (*MockAttestationDisputeGameFilterer, error) {
	contract, err := bindMockAttestationDisputeGame(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockAttestationDisputeGameFilterer{contract: contract}, nil
}

// bindMockAttestationDisputeGame binds a generic wrapper to an already deployed contract.
func bindMockAttestationDisputeGame(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockAttestationDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockAttestationDisputeGame.Contract.MockAttestationDisputeGameCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.MockAttestationDisputeGameTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.MockAttestationDisputeGameTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockAttestationDisputeGame.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockAttestationDisputeGame *MockAttestationDisputeGameTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.contract.Transact(opts, method, params...)
}

// L2BLOCKNUMBER is a free data retrieval call binding the contract method 0x326f8195.
//
// Solidity: function L2_BLOCK_NUMBER() view returns(uint256)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCaller) L2BLOCKNUMBER(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MockAttestationDisputeGame.contract.Call(opts, &out, "L2_BLOCK_NUMBER")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BLOCKNUMBER is a free data retrieval call binding the contract method 0x326f8195.
//
// Solidity: function L2_BLOCK_NUMBER() view returns(uint256)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameSession) L2BLOCKNUMBER() (*big.Int, error) {
	return _MockAttestationDisputeGame.Contract.L2BLOCKNUMBER(&_MockAttestationDisputeGame.CallOpts)
}

// L2BLOCKNUMBER is a free data retrieval call binding the contract method 0x326f8195.
//
// Solidity: function L2_BLOCK_NUMBER() view returns(uint256)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCallerSession) L2BLOCKNUMBER() (*big.Int, error) {
	return _MockAttestationDisputeGame.Contract.L2BLOCKNUMBER(&_MockAttestationDisputeGame.CallOpts)
}

// ROOTCLAIM is a free data retrieval call binding the contract method 0x4a1890f0.
//
// Solidity: function ROOT_CLAIM() view returns(bytes32)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCaller) ROOTCLAIM(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _MockAttestationDisputeGame.contract.Call(opts, &out, "ROOT_CLAIM")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ROOTCLAIM is a free data retrieval call binding the contract method 0x4a1890f0.
//
// Solidity: function ROOT_CLAIM() view returns(bytes32)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameSession) ROOTCLAIM() ([32]byte, error) {
	return _MockAttestationDisputeGame.Contract.ROOTCLAIM(&_MockAttestationDisputeGame.CallOpts)
}

// ROOTCLAIM is a free data retrieval call binding the contract method 0x4a1890f0.
//
// Solidity: function ROOT_CLAIM() view returns(bytes32)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCallerSession) ROOTCLAIM() ([32]byte, error) {
	return _MockAttestationDisputeGame.Contract.ROOTCLAIM(&_MockAttestationDisputeGame.CallOpts)
}

// Challenges is a free data retrieval call binding the contract method 0x08b43a19.
//
// Solidity: function challenges(address ) view returns(bool)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCaller) Challenges(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _MockAttestationDisputeGame.contract.Call(opts, &out, "challenges", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Challenges is a free data retrieval call binding the contract method 0x08b43a19.
//
// Solidity: function challenges(address ) view returns(bool)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameSession) Challenges(arg0 common.Address) (bool, error) {
	return _MockAttestationDisputeGame.Contract.Challenges(&_MockAttestationDisputeGame.CallOpts, arg0)
}

// Challenges is a free data retrieval call binding the contract method 0x08b43a19.
//
// Solidity: function challenges(address ) view returns(bool)
func (_MockAttestationDisputeGame *MockAttestationDisputeGameCallerSession) Challenges(arg0 common.Address) (bool, error) {
	return _MockAttestationDisputeGame.Contract.Challenges(&_MockAttestationDisputeGame.CallOpts, arg0)
}

// Challenge is a paid mutator transaction binding the contract method 0xb8b9c188.
//
// Solidity: function challenge(bytes _signature) returns()
func (_MockAttestationDisputeGame *MockAttestationDisputeGameTransactor) Challenge(opts *bind.TransactOpts, _signature []byte) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.contract.Transact(opts, "challenge", _signature)
}

// Challenge is a paid mutator transaction binding the contract method 0xb8b9c188.
//
// Solidity: function challenge(bytes _signature) returns()
func (_MockAttestationDisputeGame *MockAttestationDisputeGameSession) Challenge(_signature []byte) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.Challenge(&_MockAttestationDisputeGame.TransactOpts, _signature)
}

// Challenge is a paid mutator transaction binding the contract method 0xb8b9c188.
//
// Solidity: function challenge(bytes _signature) returns()
func (_MockAttestationDisputeGame *MockAttestationDisputeGameTransactorSession) Challenge(_signature []byte) (*types.Transaction, error) {
	return _MockAttestationDisputeGame.Contract.Challenge(&_MockAttestationDisputeGame.TransactOpts, _signature)
}
