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

// Lib_OVMCodecQueueElement is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecQueueElement struct {
	TransactionHash [32]byte
	Timestamp       *big.Int
	BlockNumber     *big.Int
}

// CanonicalTransactionChainMetaData contains all meta data concerning the CanonicalTransactionChain contract.
var CanonicalTransactionChainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxTransactionGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueGasCost\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"enqueueL2GasPrepaid\",\"type\":\"uint256\"}],\"name\":\"L2GasParamsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"QueueBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"SequencerBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"TransactionBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_l1TxOrigin\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"TransactionEnqueued\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_ROLLUP_TX_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_ROLLUP_TX_GAS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"appendSequencerBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractIChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"enqueue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueGasCost\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"enqueueL2GasPrepaid\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockNumber\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastTimestamp\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextQueueIndex\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumPendingQueueElements\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getQueueElement\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"transactionHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint40\",\"name\":\"timestamp\",\"type\":\"uint40\"},{\"internalType\":\"uint40\",\"name\":\"blockNumber\",\"type\":\"uint40\"}],\"internalType\":\"structLib_OVMCodec.QueueElement\",\"name\":\"_element\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getQueueLength\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2GasDiscountDivisor\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTransactionGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2GasDiscountDivisor\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_enqueueGasCost\",\"type\":\"uint256\"}],\"name\":\"setGasParams\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0xnull",
}

// CanonicalTransactionChainABI is the input ABI used to generate the binding from.
// Deprecated: Use CanonicalTransactionChainMetaData.ABI instead.
var CanonicalTransactionChainABI = CanonicalTransactionChainMetaData.ABI

// CanonicalTransactionChainBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CanonicalTransactionChainMetaData.Bin instead.
var CanonicalTransactionChainBin = CanonicalTransactionChainMetaData.Bin

// DeployCanonicalTransactionChain deploys a new Ethereum contract, binding an instance of CanonicalTransactionChain to it.
func DeployCanonicalTransactionChain(auth *bind.TransactOpts, backend bind.ContractBackend, _libAddressManager common.Address, _maxTransactionGasLimit *big.Int, _l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (common.Address, *types.Transaction, *CanonicalTransactionChain, error) {
	parsed, err := CanonicalTransactionChainMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CanonicalTransactionChainBin), backend, _libAddressManager, _maxTransactionGasLimit, _l2GasDiscountDivisor, _enqueueGasCost)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CanonicalTransactionChain{CanonicalTransactionChainCaller: CanonicalTransactionChainCaller{contract: contract}, CanonicalTransactionChainTransactor: CanonicalTransactionChainTransactor{contract: contract}, CanonicalTransactionChainFilterer: CanonicalTransactionChainFilterer{contract: contract}}, nil
}

// CanonicalTransactionChain is an auto generated Go binding around an Ethereum contract.
type CanonicalTransactionChain struct {
	CanonicalTransactionChainCaller     // Read-only binding to the contract
	CanonicalTransactionChainTransactor // Write-only binding to the contract
	CanonicalTransactionChainFilterer   // Log filterer for contract events
}

// CanonicalTransactionChainCaller is an auto generated read-only Go binding around an Ethereum contract.
type CanonicalTransactionChainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CanonicalTransactionChainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CanonicalTransactionChainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CanonicalTransactionChainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CanonicalTransactionChainSession struct {
	Contract     *CanonicalTransactionChain // Generic contract binding to set the session for
	CallOpts     bind.CallOpts              // Call options to use throughout this session
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// CanonicalTransactionChainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CanonicalTransactionChainCallerSession struct {
	Contract *CanonicalTransactionChainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                    // Call options to use throughout this session
}

// CanonicalTransactionChainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CanonicalTransactionChainTransactorSession struct {
	Contract     *CanonicalTransactionChainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                    // Transaction auth options to use throughout this session
}

// CanonicalTransactionChainRaw is an auto generated low-level Go binding around an Ethereum contract.
type CanonicalTransactionChainRaw struct {
	Contract *CanonicalTransactionChain // Generic contract binding to access the raw methods on
}

// CanonicalTransactionChainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CanonicalTransactionChainCallerRaw struct {
	Contract *CanonicalTransactionChainCaller // Generic read-only contract binding to access the raw methods on
}

// CanonicalTransactionChainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CanonicalTransactionChainTransactorRaw struct {
	Contract *CanonicalTransactionChainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCanonicalTransactionChain creates a new instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChain(address common.Address, backend bind.ContractBackend) (*CanonicalTransactionChain, error) {
	contract, err := bindCanonicalTransactionChain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChain{CanonicalTransactionChainCaller: CanonicalTransactionChainCaller{contract: contract}, CanonicalTransactionChainTransactor: CanonicalTransactionChainTransactor{contract: contract}, CanonicalTransactionChainFilterer: CanonicalTransactionChainFilterer{contract: contract}}, nil
}

// NewCanonicalTransactionChainCaller creates a new read-only instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainCaller(address common.Address, caller bind.ContractCaller) (*CanonicalTransactionChainCaller, error) {
	contract, err := bindCanonicalTransactionChain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainCaller{contract: contract}, nil
}

// NewCanonicalTransactionChainTransactor creates a new write-only instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainTransactor(address common.Address, transactor bind.ContractTransactor) (*CanonicalTransactionChainTransactor, error) {
	contract, err := bindCanonicalTransactionChain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactor{contract: contract}, nil
}

// NewCanonicalTransactionChainFilterer creates a new log filterer instance of CanonicalTransactionChain, bound to a specific deployed contract.
func NewCanonicalTransactionChainFilterer(address common.Address, filterer bind.ContractFilterer) (*CanonicalTransactionChainFilterer, error) {
	contract, err := bindCanonicalTransactionChain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainFilterer{contract: contract}, nil
}

// bindCanonicalTransactionChain binds a generic wrapper to an already deployed contract.
func bindCanonicalTransactionChain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CanonicalTransactionChainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CanonicalTransactionChain *CanonicalTransactionChainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.CanonicalTransactionChainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CanonicalTransactionChain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.contract.Transact(opts, method, params...)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MAXROLLUPTXSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "MAX_ROLLUP_TX_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MAXROLLUPTXSIZE(&_CanonicalTransactionChain.CallOpts)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MAXROLLUPTXSIZE(&_CanonicalTransactionChain.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MINROLLUPTXGAS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "MIN_ROLLUP_TX_GAS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MINROLLUPTXGAS(&_CanonicalTransactionChain.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MINROLLUPTXGAS(&_CanonicalTransactionChain.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "batches")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Batches() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Batches(&_CanonicalTransactionChain.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) Batches() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Batches(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) EnqueueGasCost(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "enqueueGasCost")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) EnqueueGasCost() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueGasCost(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueGasCost is a free data retrieval call binding the contract method 0xe654b1fb.
//
// Solidity: function enqueueGasCost() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) EnqueueGasCost() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueGasCost(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) EnqueueL2GasPrepaid(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "enqueueL2GasPrepaid")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueL2GasPrepaid(&_CanonicalTransactionChain.CallOpts)
}

// EnqueueL2GasPrepaid is a free data retrieval call binding the contract method 0x0b3dfa97.
//
// Solidity: function enqueueL2GasPrepaid() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) EnqueueL2GasPrepaid() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.EnqueueL2GasPrepaid(&_CanonicalTransactionChain.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetLastBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getLastBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetLastBlockNumber() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastBlockNumber(&_CanonicalTransactionChain.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetLastBlockNumber() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastBlockNumber(&_CanonicalTransactionChain.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetLastTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getLastTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetLastTimestamp() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastTimestamp(&_CanonicalTransactionChain.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetLastTimestamp() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetLastTimestamp(&_CanonicalTransactionChain.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetNextQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getNextQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetNextQueueIndex() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNextQueueIndex(&_CanonicalTransactionChain.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetNextQueueIndex() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNextQueueIndex(&_CanonicalTransactionChain.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetNumPendingQueueElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getNumPendingQueueElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNumPendingQueueElements(&_CanonicalTransactionChain.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetNumPendingQueueElements(&_CanonicalTransactionChain.CallOpts)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetQueueElement(opts *bind.CallOpts, _index *big.Int) (Lib_OVMCodecQueueElement, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getQueueElement", _index)

	if err != nil {
		return *new(Lib_OVMCodecQueueElement), err
	}

	out0 := *abi.ConvertType(out[0], new(Lib_OVMCodecQueueElement)).(*Lib_OVMCodecQueueElement)

	return out0, err

}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _CanonicalTransactionChain.Contract.GetQueueElement(&_CanonicalTransactionChain.CallOpts, _index)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _CanonicalTransactionChain.Contract.GetQueueElement(&_CanonicalTransactionChain.CallOpts, _index)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetQueueLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getQueueLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetQueueLength() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetQueueLength(&_CanonicalTransactionChain.CallOpts)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetQueueLength() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetQueueLength(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getTotalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetTotalBatches() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalBatches(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetTotalBatches() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalBatches(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "getTotalElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) GetTotalElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalElements(&_CanonicalTransactionChain.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) GetTotalElements() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.GetTotalElements(&_CanonicalTransactionChain.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) L2GasDiscountDivisor(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "l2GasDiscountDivisor")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.L2GasDiscountDivisor(&_CanonicalTransactionChain.CallOpts)
}

// L2GasDiscountDivisor is a free data retrieval call binding the contract method 0xccf987c8.
//
// Solidity: function l2GasDiscountDivisor() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) L2GasDiscountDivisor() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.L2GasDiscountDivisor(&_CanonicalTransactionChain.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) LibAddressManager() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.LibAddressManager(&_CanonicalTransactionChain.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) LibAddressManager() (common.Address, error) {
	return _CanonicalTransactionChain.Contract.LibAddressManager(&_CanonicalTransactionChain.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) MaxTransactionGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "maxTransactionGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MaxTransactionGasLimit(&_CanonicalTransactionChain.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _CanonicalTransactionChain.Contract.MaxTransactionGasLimit(&_CanonicalTransactionChain.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _CanonicalTransactionChain.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Resolve(_name string) (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Resolve(&_CanonicalTransactionChain.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_CanonicalTransactionChain *CanonicalTransactionChainCallerSession) Resolve(_name string) (common.Address, error) {
	return _CanonicalTransactionChain.Contract.Resolve(&_CanonicalTransactionChain.CallOpts, _name)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) AppendSequencerBatch(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "appendSequencerBatch")
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.AppendSequencerBatch(&_CanonicalTransactionChain.TransactOpts)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.AppendSequencerBatch(&_CanonicalTransactionChain.TransactOpts)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) Enqueue(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "enqueue", _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.Enqueue(&_CanonicalTransactionChain.TransactOpts, _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.Enqueue(&_CanonicalTransactionChain.TransactOpts, _target, _gasLimit, _data)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactor) SetGasParams(opts *bind.TransactOpts, _l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.contract.Transact(opts, "setGasParams", _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.SetGasParams(&_CanonicalTransactionChain.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// SetGasParams is a paid mutator transaction binding the contract method 0xedcc4a45.
//
// Solidity: function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) returns()
func (_CanonicalTransactionChain *CanonicalTransactionChainTransactorSession) SetGasParams(_l2GasDiscountDivisor *big.Int, _enqueueGasCost *big.Int) (*types.Transaction, error) {
	return _CanonicalTransactionChain.Contract.SetGasParams(&_CanonicalTransactionChain.TransactOpts, _l2GasDiscountDivisor, _enqueueGasCost)
}

// CanonicalTransactionChainL2GasParamsUpdatedIterator is returned from FilterL2GasParamsUpdated and is used to iterate over the raw logs and unpacked data for L2GasParamsUpdated events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainL2GasParamsUpdatedIterator struct {
	Event *CanonicalTransactionChainL2GasParamsUpdated // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainL2GasParamsUpdated)
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
		it.Event = new(CanonicalTransactionChainL2GasParamsUpdated)
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
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainL2GasParamsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainL2GasParamsUpdated represents a L2GasParamsUpdated event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainL2GasParamsUpdated struct {
	L2GasDiscountDivisor *big.Int
	EnqueueGasCost       *big.Int
	EnqueueL2GasPrepaid  *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterL2GasParamsUpdated is a free log retrieval operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterL2GasParamsUpdated(opts *bind.FilterOpts) (*CanonicalTransactionChainL2GasParamsUpdatedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainL2GasParamsUpdatedIterator{contract: _CanonicalTransactionChain.contract, event: "L2GasParamsUpdated", logs: logs, sub: sub}, nil
}

// WatchL2GasParamsUpdated is a free log subscription operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchL2GasParamsUpdated(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainL2GasParamsUpdated) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "L2GasParamsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainL2GasParamsUpdated)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
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

// ParseL2GasParamsUpdated is a log parse operation binding the contract event 0xc6ed75e96b8b18b71edc1a6e82a9d677f8268c774a262c624eeb2cf0a8b3e07e.
//
// Solidity: event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseL2GasParamsUpdated(log types.Log) (*CanonicalTransactionChainL2GasParamsUpdated, error) {
	event := new(CanonicalTransactionChainL2GasParamsUpdated)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "L2GasParamsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainQueueBatchAppendedIterator is returned from FilterQueueBatchAppended and is used to iterate over the raw logs and unpacked data for QueueBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainQueueBatchAppendedIterator struct {
	Event *CanonicalTransactionChainQueueBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainQueueBatchAppended)
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
		it.Event = new(CanonicalTransactionChainQueueBatchAppended)
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
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainQueueBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainQueueBatchAppended represents a QueueBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainQueueBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterQueueBatchAppended is a free log retrieval operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterQueueBatchAppended(opts *bind.FilterOpts) (*CanonicalTransactionChainQueueBatchAppendedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainQueueBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "QueueBatchAppended", logs: logs, sub: sub}, nil
}

// WatchQueueBatchAppended is a free log subscription operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchQueueBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainQueueBatchAppended) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainQueueBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
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

// ParseQueueBatchAppended is a log parse operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseQueueBatchAppended(log types.Log) (*CanonicalTransactionChainQueueBatchAppended, error) {
	event := new(CanonicalTransactionChainQueueBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainSequencerBatchAppendedIterator is returned from FilterSequencerBatchAppended and is used to iterate over the raw logs and unpacked data for SequencerBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainSequencerBatchAppendedIterator struct {
	Event *CanonicalTransactionChainSequencerBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainSequencerBatchAppended)
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
		it.Event = new(CanonicalTransactionChainSequencerBatchAppended)
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
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainSequencerBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainSequencerBatchAppended represents a SequencerBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainSequencerBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchAppended is a free log retrieval operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterSequencerBatchAppended(opts *bind.FilterOpts) (*CanonicalTransactionChainSequencerBatchAppendedIterator, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainSequencerBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "SequencerBatchAppended", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchAppended is a free log subscription operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchSequencerBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainSequencerBatchAppended) (event.Subscription, error) {

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainSequencerBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
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

// ParseSequencerBatchAppended is a log parse operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseSequencerBatchAppended(log types.Log) (*CanonicalTransactionChainSequencerBatchAppended, error) {
	event := new(CanonicalTransactionChainSequencerBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainTransactionBatchAppendedIterator is returned from FilterTransactionBatchAppended and is used to iterate over the raw logs and unpacked data for TransactionBatchAppended events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionBatchAppendedIterator struct {
	Event *CanonicalTransactionChainTransactionBatchAppended // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainTransactionBatchAppended)
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
		it.Event = new(CanonicalTransactionChainTransactionBatchAppended)
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
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainTransactionBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainTransactionBatchAppended represents a TransactionBatchAppended event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionBatchAppended struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterTransactionBatchAppended is a free log retrieval operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterTransactionBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*CanonicalTransactionChainTransactionBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactionBatchAppendedIterator{contract: _CanonicalTransactionChain.contract, event: "TransactionBatchAppended", logs: logs, sub: sub}, nil
}

// WatchTransactionBatchAppended is a free log subscription operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchTransactionBatchAppended(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainTransactionBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainTransactionBatchAppended)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
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

// ParseTransactionBatchAppended is a log parse operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseTransactionBatchAppended(log types.Log) (*CanonicalTransactionChainTransactionBatchAppended, error) {
	event := new(CanonicalTransactionChainTransactionBatchAppended)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CanonicalTransactionChainTransactionEnqueuedIterator is returned from FilterTransactionEnqueued and is used to iterate over the raw logs and unpacked data for TransactionEnqueued events raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionEnqueuedIterator struct {
	Event *CanonicalTransactionChainTransactionEnqueued // Event containing the contract specifics and raw log

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
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CanonicalTransactionChainTransactionEnqueued)
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
		it.Event = new(CanonicalTransactionChainTransactionEnqueued)
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
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CanonicalTransactionChainTransactionEnqueuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CanonicalTransactionChainTransactionEnqueued represents a TransactionEnqueued event raised by the CanonicalTransactionChain contract.
type CanonicalTransactionChainTransactionEnqueued struct {
	L1TxOrigin common.Address
	Target     common.Address
	GasLimit   *big.Int
	Data       []byte
	QueueIndex *big.Int
	Timestamp  *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionEnqueued is a free log retrieval operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) FilterTransactionEnqueued(opts *bind.FilterOpts, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (*CanonicalTransactionChainTransactionEnqueuedIterator, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.FilterLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return &CanonicalTransactionChainTransactionEnqueuedIterator{contract: _CanonicalTransactionChain.contract, event: "TransactionEnqueued", logs: logs, sub: sub}, nil
}

// WatchTransactionEnqueued is a free log subscription operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) WatchTransactionEnqueued(opts *bind.WatchOpts, sink chan<- *CanonicalTransactionChainTransactionEnqueued, _l1TxOrigin []common.Address, _target []common.Address, _queueIndex []*big.Int) (event.Subscription, error) {

	var _l1TxOriginRule []interface{}
	for _, _l1TxOriginItem := range _l1TxOrigin {
		_l1TxOriginRule = append(_l1TxOriginRule, _l1TxOriginItem)
	}
	var _targetRule []interface{}
	for _, _targetItem := range _target {
		_targetRule = append(_targetRule, _targetItem)
	}

	var _queueIndexRule []interface{}
	for _, _queueIndexItem := range _queueIndex {
		_queueIndexRule = append(_queueIndexRule, _queueIndexItem)
	}

	logs, sub, err := _CanonicalTransactionChain.contract.WatchLogs(opts, "TransactionEnqueued", _l1TxOriginRule, _targetRule, _queueIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CanonicalTransactionChainTransactionEnqueued)
				if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
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

// ParseTransactionEnqueued is a log parse operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
func (_CanonicalTransactionChain *CanonicalTransactionChainFilterer) ParseTransactionEnqueued(log types.Log) (*CanonicalTransactionChainTransactionEnqueued, error) {
	event := new(CanonicalTransactionChainTransactionEnqueued)
	if err := _CanonicalTransactionChain.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
