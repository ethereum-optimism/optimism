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

// L2OutputOracleOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type L2OutputOracleOutputProposal struct {
	OutputRoot [32]byte
	Timestamp  *big.Int
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sequencer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"}],\"name\":\"l2OutputAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"}],\"name\":\"l2OutputDeleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"HISTORICAL_TOTAL_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_BLOCK_TIME\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_BLOCK_TIMESTAMP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1Blocknumber\",\"type\":\"uint256\"}],\"name\":\"appendL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"}],\"name\":\"computeL2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"_proposal\",\"type\":\"tuple\"}],\"name\":\"deleteL2Output\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2Timestamp\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x61010060405234801561001157600080fd5b50604051610f9e380380610f9e83398101604081905261003091610164565b61003933610114565b61004385876101c8565b156100ba5760405162461bcd60e51b815260206004820152603760248201527f5375626d697373696f6e20496e74657276616c206d7573742062652061206d7560448201527f6c7469706c65206f66204c3220426c6f636b2054696d65000000000000000000606482015260840160405180910390fd5b608086905260a0859052604080518082018252858152426020808301918252600086815260029091529290922090518155905160019182015560c084905282905560e082905261010981610114565b5050505050506101ea565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60008060008060008060c0878903121561017d57600080fd5b86516020880151604089015160608a015160808b015160a08c0151949a50929850909650945092506001600160a01b03811681146101ba57600080fd5b809150509295509295509295565b6000826101e557634e487b7160e01b600052601260045260246000fd5b500690565b60805160a05160c05160e051610d5261024c600039600081816101af0152818161031b01526103f10152600081816102d5015261042201526000818160e801526103d00152600081816101e30152818161062201526109570152610d526000f3fe6080604052600436106100d15760003560e01c806341840fa61161007f5780638da5cb5b116100595780638da5cb5b1461021a578063a25ae5571461024f578063a4771aad146102c3578063f2fde38b146102f757600080fd5b806341840fa61461019d578063529933df146101d1578063715018a61461020557600080fd5b80630c1952d3116100b05780630c1952d31461015f5780632518810414610175578063357e951f1461018857600080fd5b80622134cc146100d657806302e513451461011d578063093b3d901461013d575b600080fd5b3480156100e257600080fd5b5061010a7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b34801561012957600080fd5b5061010a610138366004610bba565b610317565b34801561014957600080fd5b5061015d610158366004610bd3565b610449565b005b34801561016b57600080fd5b5061010a60015481565b61015d610183366004610c49565b61064e565b34801561019457600080fd5b5061010a610953565b3480156101a957600080fd5b5061010a7f000000000000000000000000000000000000000000000000000000000000000081565b3480156101dd57600080fd5b5061010a7f000000000000000000000000000000000000000000000000000000000000000081565b34801561021157600080fd5b5061015d610988565b34801561022657600080fd5b5060005460405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610114565b34801561025b57600080fd5b506102a861026a366004610bba565b604080518082019091526000808252602082015250600090815260026020908152604091829020825180840190935280548352600101549082015290565b60408051825181526020928301519281019290925201610114565b3480156102cf57600080fd5b5061010a7f000000000000000000000000000000000000000000000000000000000000000081565b34801561030357600080fd5b5061015d610312366004610c7b565b610a15565b60007f00000000000000000000000000000000000000000000000000000000000000008210156103ce576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f54696d657374616d70207072696f7220746f207374617274696e67426c6f636b60448201527f54696d657374616d70000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b7f00000000000000000000000000000000000000000000000000000000000000007f000000000000000000000000000000000000000000000000000000000000000083038161041f5761041f610cb8565b047f0000000000000000000000000000000000000000000000000000000000000000019050919050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146104ca576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016103c5565b6001805460009081526002602090815260409182902082518084019093528054808452930154908201528251909114610585576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602760248201527f43616e206f6e6c792064656c65746520746865206d6f737420726563656e742060448201527f6f75747075742e0000000000000000000000000000000000000000000000000060648201526084016103c5565b80602001518260200151146105d0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600060248201526044016103c5565b600154602082015182516040517f6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c700990600090a460018054600090815260026020526040812081815582015554610647907f000000000000000000000000000000000000000000000000000000000000000090610d16565b6001555050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146106cf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016103c5565b42831061075e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f43616e6e6f7420617070656e64204c32206f757470757420696e20667574757260448201527f650000000000000000000000000000000000000000000000000000000000000060648201526084016103c5565b610766610953565b83146107f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f54696d657374616d70206e6f7420657175616c20746f206e657874206578706560448201527f637465642074696d657374616d7000000000000000000000000000000000000060648201526084016103c5565b8361085b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f43616e6e6f74207375626d697420656d707479204c32206f757470757400000060448201526064016103c5565b81156108f157818140146108f1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f426c6f636b6861736820646f6573206e6f74206d61746368207468652068617360448201527f6820617420746865206578706563746564206865696768742e0000000000000060648201526084016103c5565b60408051808201825285815242602080830182815260008881526002909252848220935184555160019384015591869055915185929187917f54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba9190a450505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001546109839190610d2d565b905090565b60005473ffffffffffffffffffffffffffffffffffffffff163314610a09576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016103c5565b610a136000610b45565b565b60005473ffffffffffffffffffffffffffffffffffffffff163314610a96576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016103c5565b73ffffffffffffffffffffffffffffffffffffffff8116610b39576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016103c5565b610b4281610b45565b50565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b600060208284031215610bcc57600080fd5b5035919050565b600060408284031215610be557600080fd5b6040516040810181811067ffffffffffffffff82111715610c2f577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052823581526020928301359281019290925250919050565b60008060008060808587031215610c5f57600080fd5b5050823594602084013594506040840135936060013592509050565b600060208284031215610c8d57600080fd5b813573ffffffffffffffffffffffffffffffffffffffff81168114610cb157600080fd5b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015610d2857610d28610ce7565b500390565b60008219821115610d4057610d40610ce7565b50019056fea164736f6c634300080a000a",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _l2BlockTime *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int, _startingBlockTimestamp *big.Int, sequencer common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _l2BlockTime, _genesisL2Output, _historicalTotalBlocks, _startingBlockTimestamp, sequencer)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// L2OutputOracle is an auto generated Go binding around an Ethereum contract.
type L2OutputOracle struct {
	L2OutputOracleCaller     // Read-only binding to the contract
	L2OutputOracleTransactor // Write-only binding to the contract
	L2OutputOracleFilterer   // Log filterer for contract events
}

// L2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2OutputOracleSession struct {
	Contract     *L2OutputOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2OutputOracleCallerSession struct {
	Contract *L2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2OutputOracleTransactorSession struct {
	Contract     *L2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2OutputOracleRaw struct {
	Contract *L2OutputOracle // Generic contract binding to access the raw methods on
}

// L2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2OutputOracleCallerRaw struct {
	Contract *L2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// L2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactorRaw struct {
	Contract *L2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2OutputOracle creates a new instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracle(address common.Address, backend bind.ContractBackend) (*L2OutputOracle, error) {
	contract, err := bindL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// NewL2OutputOracleCaller creates a new read-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*L2OutputOracleCaller, error) {
	contract, err := bindL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleCaller{contract: contract}, nil
}

// NewL2OutputOracleTransactor creates a new write-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2OutputOracleTransactor, error) {
	contract, err := bindL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleTransactor{contract: contract}, nil
}

// NewL2OutputOracleFilterer creates a new log filterer instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*L2OutputOracleFilterer, error) {
	contract, err := bindL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleFilterer{contract: contract}, nil
}

// bindL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.L2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) HISTORICALTOTALBLOCKS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "HISTORICAL_TOTAL_BLOCKS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) L2BLOCKTIME(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "L2_BLOCK_TIME")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKTIMESTAMP is a free data retrieval call binding the contract method 0x41840fa6.
//
// Solidity: function STARTING_BLOCK_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) STARTINGBLOCKTIMESTAMP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "STARTING_BLOCK_TIMESTAMP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// STARTINGBLOCKTIMESTAMP is a free data retrieval call binding the contract method 0x41840fa6.
//
// Solidity: function STARTING_BLOCK_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) STARTINGBLOCKTIMESTAMP() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKTIMESTAMP(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKTIMESTAMP is a free data retrieval call binding the contract method 0x41840fa6.
//
// Solidity: function STARTING_BLOCK_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) STARTINGBLOCKTIMESTAMP() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKTIMESTAMP(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SUBMISSIONINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "SUBMISSION_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) ComputeL2BlockNumber(opts *bind.CallOpts, _l2timestamp *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "computeL2BlockNumber", _l2timestamp)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) ComputeL2BlockNumber(_l2timestamp *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2BlockNumber(&_L2OutputOracle.CallOpts, _l2timestamp)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) ComputeL2BlockNumber(_l2timestamp *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2BlockNumber(&_L2OutputOracle.CallOpts, _l2timestamp)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2Timestamp *big.Int) (L2OutputOracleOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2Timestamp)

	if err != nil {
		return *new(L2OutputOracleOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(L2OutputOracleOutputProposal)).(*L2OutputOracleOutputProposal)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2Timestamp *big.Int) (L2OutputOracleOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2Timestamp)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2Timestamp *big.Int) (L2OutputOracleOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2Timestamp)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextTimestamp(&_L2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextTimestamp(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) AppendL2Output(opts *bind.TransactOpts, _l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "appendL2Output", _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) AppendL2Output(_l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) AppendL2Output(_l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Output(opts *bind.TransactOpts, _proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Output", _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// L2OutputOracleOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferredIterator struct {
	Event *L2OutputOracleOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOwnershipTransferred)
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
		it.Event = new(L2OutputOracleOwnershipTransferred)
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
func (it *L2OutputOracleOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L2OutputOracleOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOwnershipTransferredIterator{contract: _L2OutputOracle.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOwnershipTransferred)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOwnershipTransferred(log types.Log) (*L2OutputOracleOwnershipTransferred, error) {
	event := new(L2OutputOracleOwnershipTransferred)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputAppendedIterator is returned from FilterL2OutputAppended and is used to iterate over the raw logs and unpacked data for L2OutputAppended events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppendedIterator struct {
	Event *L2OutputOracleL2OutputAppended // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputAppended)
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
		it.Event = new(L2OutputOracleL2OutputAppended)
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
func (it *L2OutputOracleL2OutputAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputAppended represents a L2OutputAppended event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppended struct {
	L2Output    [32]byte
	L1Timestamp *big.Int
	L2timestamp *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterL2OutputAppended is a free log retrieval operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputAppended(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2timestamp []*big.Int) (*L2OutputOracleL2OutputAppendedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "l2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputAppendedIterator{contract: _L2OutputOracle.contract, event: "l2OutputAppended", logs: logs, sub: sub}, nil
}

// WatchL2OutputAppended is a free log subscription operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputAppended(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputAppended, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2timestamp []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "l2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputAppended)
				if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
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

// ParseL2OutputAppended is a log parse operation binding the contract event 0x54fbf9b58db0b6543a3a0cb9fbd4e98a7c7b88878978fa125b84a2c7f51d34ba.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputAppended(log types.Log) (*L2OutputOracleL2OutputAppended, error) {
	event := new(L2OutputOracleL2OutputAppended)
	if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputDeletedIterator is returned from FilterL2OutputDeleted and is used to iterate over the raw logs and unpacked data for L2OutputDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeletedIterator struct {
	Event *L2OutputOracleL2OutputDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputDeleted)
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
		it.Event = new(L2OutputOracleL2OutputDeleted)
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
func (it *L2OutputOracleL2OutputDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputDeleted represents a L2OutputDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeleted struct {
	L2Output    [32]byte
	L1Timestamp *big.Int
	L2timestamp *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterL2OutputDeleted is a free log retrieval operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputDeleted(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2timestamp []*big.Int) (*L2OutputOracleL2OutputDeletedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "l2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputDeletedIterator{contract: _L2OutputOracle.contract, event: "l2OutputDeleted", logs: logs, sub: sub}, nil
}

// WatchL2OutputDeleted is a free log subscription operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputDeleted, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2timestamp []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "l2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputDeleted", log); err != nil {
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

// ParseL2OutputDeleted is a log parse operation binding the contract event 0x6897e92e2fea3b89bf0d45ed867487716da71bfa624c6878569d227d736c7009.
//
// Solidity: event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputDeleted(log types.Log) (*L2OutputOracleL2OutputDeleted, error) {
	event := new(L2OutputOracleL2OutputDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
