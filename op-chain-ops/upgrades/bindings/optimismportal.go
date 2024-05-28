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

// TypesOutputRootProof is an auto generated low-level Go binding around an user-defined struct.
type TypesOutputRootProof struct {
	Version                  [32]byte
	StateRoot                [32]byte
	MessagePasserStorageRoot [32]byte
	LatestBlockhash          [32]byte
}

// TypesWithdrawalTransaction is an auto generated low-level Go binding around an user-defined struct.
type TypesWithdrawalTransaction struct {
	Nonce    *big.Int
	Sender   common.Address
	Target   common.Address
	Value    *big.Int
	GasLimit *big.Int
	Data     []byte
}

// OptimismPortalMetaData contains all meta data concerning the OptimismPortal contract.
var OptimismPortalMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"},{\"inputs\":[],\"name\":\"balance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_mint\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositERC20Transaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"donateETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.WithdrawalTransaction\",\"name\":\"_tx\",\"type\":\"tuple\"}],\"name\":\"finalizeWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"finalizedWithdrawals\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"guardian\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"_l2Oracle\",\"type\":\"address\"},{\"internalType\":\"contractSystemConfig\",\"name\":\"_systemConfig\",\"type\":\"address\"},{\"internalType\":\"contractSuperchainConfig\",\"name\":\"_superchainConfig\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2OutputIndex\",\"type\":\"uint256\"}],\"name\":\"isOutputFinalized\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Oracle\",\"outputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_byteCount\",\"type\":\"uint64\"}],\"name\":\"minimumGasLimit\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"params\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"prevBaseFee\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"prevBoughtGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevBlockNum\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"paused_\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.WithdrawalTransaction\",\"name\":\"_tx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_l2OutputIndex\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"messagePasserStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structTypes.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes[]\",\"name\":\"_withdrawalProof\",\"type\":\"bytes[]\"}],\"name\":\"proveWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"provenWithdrawals\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"timestamp\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"l2OutputIndex\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"_name\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_symbol\",\"type\":\"bytes32\"}],\"name\":\"setGasPayingToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainConfig\",\"outputs\":[{\"internalType\":\"contractSuperchainConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemConfig\",\"outputs\":[{\"internalType\":\"contractSystemConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"opaqueData\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"withdrawalHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"withdrawalHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"WithdrawalProven\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BadTarget\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CallPaused\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GasEstimation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LargeCalldata\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonReentrant\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyCustomGasToken\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OutOfGas\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SmallGasLimit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TransferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"}]",
}

// OptimismPortalABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismPortalMetaData.ABI instead.
var OptimismPortalABI = OptimismPortalMetaData.ABI

// OptimismPortal is an auto generated Go binding around an Ethereum contract.
type OptimismPortal struct {
	OptimismPortalCaller     // Read-only binding to the contract
	OptimismPortalTransactor // Write-only binding to the contract
	OptimismPortalFilterer   // Log filterer for contract events
}

// OptimismPortalCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismPortalCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismPortalTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismPortalFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismPortalSession struct {
	Contract     *OptimismPortal   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismPortalCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismPortalCallerSession struct {
	Contract *OptimismPortalCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OptimismPortalTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismPortalTransactorSession struct {
	Contract     *OptimismPortalTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OptimismPortalRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismPortalRaw struct {
	Contract *OptimismPortal // Generic contract binding to access the raw methods on
}

// OptimismPortalCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismPortalCallerRaw struct {
	Contract *OptimismPortalCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismPortalTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismPortalTransactorRaw struct {
	Contract *OptimismPortalTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismPortal creates a new instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortal(address common.Address, backend bind.ContractBackend) (*OptimismPortal, error) {
	contract, err := bindOptimismPortal(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal{OptimismPortalCaller: OptimismPortalCaller{contract: contract}, OptimismPortalTransactor: OptimismPortalTransactor{contract: contract}, OptimismPortalFilterer: OptimismPortalFilterer{contract: contract}}, nil
}

// NewOptimismPortalCaller creates a new read-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalCaller(address common.Address, caller bind.ContractCaller) (*OptimismPortalCaller, error) {
	contract, err := bindOptimismPortal(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalCaller{contract: contract}, nil
}

// NewOptimismPortalTransactor creates a new write-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismPortalTransactor, error) {
	contract, err := bindOptimismPortal(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactor{contract: contract}, nil
}

// NewOptimismPortalFilterer creates a new log filterer instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismPortalFilterer, error) {
	contract, err := bindOptimismPortal(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalFilterer{contract: contract}, nil
}

// bindOptimismPortal binds a generic wrapper to an already deployed contract.
func bindOptimismPortal(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismPortalABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.OptimismPortalCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transact(opts, method, params...)
}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal *OptimismPortalCaller) Balance(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "balance")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal *OptimismPortalSession) Balance() (*big.Int, error) {
	return _OptimismPortal.Contract.Balance(&_OptimismPortal.CallOpts)
}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal *OptimismPortalCallerSession) Balance() (*big.Int, error) {
	return _OptimismPortal.Contract.Balance(&_OptimismPortal.CallOpts)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCaller) FinalizedWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "finalizedWithdrawals", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCallerSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) Guardian(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "guardian")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal *OptimismPortalSession) Guardian() (common.Address, error) {
	return _OptimismPortal.Contract.Guardian(&_OptimismPortal.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) Guardian() (common.Address, error) {
	return _OptimismPortal.Contract.Guardian(&_OptimismPortal.CallOpts)
}

// IsOutputFinalized is a free data retrieval call binding the contract method 0x6dbffb78.
//
// Solidity: function isOutputFinalized(uint256 _l2OutputIndex) view returns(bool)
func (_OptimismPortal *OptimismPortalCaller) IsOutputFinalized(opts *bind.CallOpts, _l2OutputIndex *big.Int) (bool, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "isOutputFinalized", _l2OutputIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOutputFinalized is a free data retrieval call binding the contract method 0x6dbffb78.
//
// Solidity: function isOutputFinalized(uint256 _l2OutputIndex) view returns(bool)
func (_OptimismPortal *OptimismPortalSession) IsOutputFinalized(_l2OutputIndex *big.Int) (bool, error) {
	return _OptimismPortal.Contract.IsOutputFinalized(&_OptimismPortal.CallOpts, _l2OutputIndex)
}

// IsOutputFinalized is a free data retrieval call binding the contract method 0x6dbffb78.
//
// Solidity: function isOutputFinalized(uint256 _l2OutputIndex) view returns(bool)
func (_OptimismPortal *OptimismPortalCallerSession) IsOutputFinalized(_l2OutputIndex *big.Int) (bool, error) {
	return _OptimismPortal.Contract.IsOutputFinalized(&_OptimismPortal.CallOpts, _l2OutputIndex)
}

// L2Oracle is a free data retrieval call binding the contract method 0x9b5f694a.
//
// Solidity: function l2Oracle() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2Oracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "l2Oracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Oracle is a free data retrieval call binding the contract method 0x9b5f694a.
//
// Solidity: function l2Oracle() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2Oracle() (common.Address, error) {
	return _OptimismPortal.Contract.L2Oracle(&_OptimismPortal.CallOpts)
}

// L2Oracle is a free data retrieval call binding the contract method 0x9b5f694a.
//
// Solidity: function l2Oracle() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2Oracle() (common.Address, error) {
	return _OptimismPortal.Contract.L2Oracle(&_OptimismPortal.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "l2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal *OptimismPortalCaller) MinimumGasLimit(opts *bind.CallOpts, _byteCount uint64) (uint64, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "minimumGasLimit", _byteCount)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal *OptimismPortalSession) MinimumGasLimit(_byteCount uint64) (uint64, error) {
	return _OptimismPortal.Contract.MinimumGasLimit(&_OptimismPortal.CallOpts, _byteCount)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal *OptimismPortalCallerSession) MinimumGasLimit(_byteCount uint64) (uint64, error) {
	return _OptimismPortal.Contract.MinimumGasLimit(&_OptimismPortal.CallOpts, _byteCount)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal *OptimismPortalCaller) Params(opts *bind.CallOpts) (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "params")

	outstruct := new(struct {
		PrevBaseFee   *big.Int
		PrevBoughtGas uint64
		PrevBlockNum  uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PrevBaseFee = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.PrevBoughtGas = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.PrevBlockNum = *abi.ConvertType(out[2], new(uint64)).(*uint64)

	return *outstruct, err

}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal *OptimismPortalSession) Params() (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	return _OptimismPortal.Contract.Params(&_OptimismPortal.CallOpts)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal *OptimismPortalCallerSession) Params() (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	return _OptimismPortal.Contract.Params(&_OptimismPortal.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_OptimismPortal *OptimismPortalCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_OptimismPortal *OptimismPortalSession) Paused() (bool, error) {
	return _OptimismPortal.Contract.Paused(&_OptimismPortal.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_OptimismPortal *OptimismPortalCallerSession) Paused() (bool, error) {
	return _OptimismPortal.Contract.Paused(&_OptimismPortal.CallOpts)
}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xe965084c.
//
// Solidity: function provenWithdrawals(bytes32 ) view returns(bytes32 outputRoot, uint128 timestamp, uint128 l2OutputIndex)
func (_OptimismPortal *OptimismPortalCaller) ProvenWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2OutputIndex *big.Int
}, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "provenWithdrawals", arg0)

	outstruct := new(struct {
		OutputRoot    [32]byte
		Timestamp     *big.Int
		L2OutputIndex *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.OutputRoot = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.L2OutputIndex = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xe965084c.
//
// Solidity: function provenWithdrawals(bytes32 ) view returns(bytes32 outputRoot, uint128 timestamp, uint128 l2OutputIndex)
func (_OptimismPortal *OptimismPortalSession) ProvenWithdrawals(arg0 [32]byte) (struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2OutputIndex *big.Int
}, error) {
	return _OptimismPortal.Contract.ProvenWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xe965084c.
//
// Solidity: function provenWithdrawals(bytes32 ) view returns(bytes32 outputRoot, uint128 timestamp, uint128 l2OutputIndex)
func (_OptimismPortal *OptimismPortalCallerSession) ProvenWithdrawals(arg0 [32]byte) (struct {
	OutputRoot    [32]byte
	Timestamp     *big.Int
	L2OutputIndex *big.Int
}, error) {
	return _OptimismPortal.Contract.ProvenWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal *OptimismPortalSession) SuperchainConfig() (common.Address, error) {
	return _OptimismPortal.Contract.SuperchainConfig(&_OptimismPortal.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) SuperchainConfig() (common.Address, error) {
	return _OptimismPortal.Contract.SuperchainConfig(&_OptimismPortal.CallOpts)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) SystemConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "systemConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal *OptimismPortalSession) SystemConfig() (common.Address, error) {
	return _OptimismPortal.Contract.SystemConfig(&_OptimismPortal.CallOpts)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) SystemConfig() (common.Address, error) {
	return _OptimismPortal.Contract.SystemConfig(&_OptimismPortal.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismPortal *OptimismPortalCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismPortal *OptimismPortalSession) Version() (string, error) {
	return _OptimismPortal.Contract.Version(&_OptimismPortal.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismPortal *OptimismPortalCallerSession) Version() (string, error) {
	return _OptimismPortal.Contract.Version(&_OptimismPortal.CallOpts)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal *OptimismPortalTransactor) DepositERC20Transaction(opts *bind.TransactOpts, _to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "depositERC20Transaction", _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal *OptimismPortalSession) DepositERC20Transaction(_to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositERC20Transaction(&_OptimismPortal.TransactOpts, _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) DepositERC20Transaction(_to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositERC20Transaction(&_OptimismPortal.TransactOpts, _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal *OptimismPortalTransactor) DonateETH(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "donateETH")
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal *OptimismPortalSession) DonateETH() (*types.Transaction, error) {
	return _OptimismPortal.Contract.DonateETH(&_OptimismPortal.TransactOpts)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) DonateETH() (*types.Transaction, error) {
	return _OptimismPortal.Contract.DonateETH(&_OptimismPortal.TransactOpts)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal *OptimismPortalTransactor) FinalizeWithdrawalTransaction(opts *bind.TransactOpts, _tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "finalizeWithdrawalTransaction", _tx)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal *OptimismPortalSession) FinalizeWithdrawalTransaction(_tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _tx)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) FinalizeWithdrawalTransaction(_tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _tx)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _l2Oracle, address _systemConfig, address _superchainConfig) returns()
func (_OptimismPortal *OptimismPortalTransactor) Initialize(opts *bind.TransactOpts, _l2Oracle common.Address, _systemConfig common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "initialize", _l2Oracle, _systemConfig, _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _l2Oracle, address _systemConfig, address _superchainConfig) returns()
func (_OptimismPortal *OptimismPortalSession) Initialize(_l2Oracle common.Address, _systemConfig common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _OptimismPortal.Contract.Initialize(&_OptimismPortal.TransactOpts, _l2Oracle, _systemConfig, _superchainConfig)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _l2Oracle, address _systemConfig, address _superchainConfig) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) Initialize(_l2Oracle common.Address, _systemConfig common.Address, _superchainConfig common.Address) (*types.Transaction, error) {
	return _OptimismPortal.Contract.Initialize(&_OptimismPortal.TransactOpts, _l2Oracle, _systemConfig, _superchainConfig)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _l2OutputIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactor) ProveWithdrawalTransaction(opts *bind.TransactOpts, _tx TypesWithdrawalTransaction, _l2OutputIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "proveWithdrawalTransaction", _tx, _l2OutputIndex, _outputRootProof, _withdrawalProof)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _l2OutputIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalSession) ProveWithdrawalTransaction(_tx TypesWithdrawalTransaction, _l2OutputIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.ProveWithdrawalTransaction(&_OptimismPortal.TransactOpts, _tx, _l2OutputIndex, _outputRootProof, _withdrawalProof)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _l2OutputIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) ProveWithdrawalTransaction(_tx TypesWithdrawalTransaction, _l2OutputIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.ProveWithdrawalTransaction(&_OptimismPortal.TransactOpts, _tx, _l2OutputIndex, _outputRootProof, _withdrawalProof)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal *OptimismPortalTransactor) SetGasPayingToken(opts *bind.TransactOpts, _token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "setGasPayingToken", _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal *OptimismPortalSession) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.SetGasPayingToken(&_OptimismPortal.TransactOpts, _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.SetGasPayingToken(&_OptimismPortal.TransactOpts, _token, _decimals, _name, _symbol)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// OptimismPortalInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OptimismPortal contract.
type OptimismPortalInitializedIterator struct {
	Event *OptimismPortalInitialized // Event containing the contract specifics and raw log

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
func (it *OptimismPortalInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalInitialized)
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
		it.Event = new(OptimismPortalInitialized)
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
func (it *OptimismPortalInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalInitialized represents a Initialized event raised by the OptimismPortal contract.
type OptimismPortalInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal *OptimismPortalFilterer) FilterInitialized(opts *bind.FilterOpts) (*OptimismPortalInitializedIterator, error) {

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OptimismPortalInitializedIterator{contract: _OptimismPortal.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal *OptimismPortalFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OptimismPortalInitialized) (event.Subscription, error) {

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalInitialized)
				if err := _OptimismPortal.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal *OptimismPortalFilterer) ParseInitialized(log types.Log) (*OptimismPortalInitialized, error) {
	event := new(OptimismPortalInitialized)
	if err := _OptimismPortal.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortalTransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the OptimismPortal contract.
type OptimismPortalTransactionDepositedIterator struct {
	Event *OptimismPortalTransactionDeposited // Event containing the contract specifics and raw log

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
func (it *OptimismPortalTransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalTransactionDeposited)
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
		it.Event = new(OptimismPortalTransactionDeposited)
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
func (it *OptimismPortalTransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalTransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalTransactionDeposited represents a TransactionDeposited event raised by the OptimismPortal contract.
type OptimismPortalTransactionDeposited struct {
	From       common.Address
	To         common.Address
	Version    *big.Int
	OpaqueData []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal *OptimismPortalFilterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address, version []*big.Int) (*OptimismPortalTransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule, versionRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactionDepositedIterator{contract: _OptimismPortal.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal *OptimismPortalFilterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *OptimismPortalTransactionDeposited, from []common.Address, to []common.Address, version []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule, versionRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalTransactionDeposited)
				if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
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

// ParseTransactionDeposited is a log parse operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal *OptimismPortalFilterer) ParseTransactionDeposited(log types.Log) (*OptimismPortalTransactionDeposited, error) {
	event := new(OptimismPortalTransactionDeposited)
	if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortalWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalizedIterator struct {
	Event *OptimismPortalWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *OptimismPortalWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalWithdrawalFinalized)
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
		it.Event = new(OptimismPortalWithdrawalFinalized)
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
func (it *OptimismPortalWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalWithdrawalFinalized represents a WithdrawalFinalized event raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalized struct {
	WithdrawalHash [32]byte
	Success        bool
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal *OptimismPortalFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, withdrawalHash [][32]byte) (*OptimismPortalWithdrawalFinalizedIterator, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "WithdrawalFinalized", withdrawalHashRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalWithdrawalFinalizedIterator{contract: _OptimismPortal.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal *OptimismPortalFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *OptimismPortalWithdrawalFinalized, withdrawalHash [][32]byte) (event.Subscription, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "WithdrawalFinalized", withdrawalHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalWithdrawalFinalized)
				if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal *OptimismPortalFilterer) ParseWithdrawalFinalized(log types.Log) (*OptimismPortalWithdrawalFinalized, error) {
	event := new(OptimismPortalWithdrawalFinalized)
	if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortalWithdrawalProvenIterator is returned from FilterWithdrawalProven and is used to iterate over the raw logs and unpacked data for WithdrawalProven events raised by the OptimismPortal contract.
type OptimismPortalWithdrawalProvenIterator struct {
	Event *OptimismPortalWithdrawalProven // Event containing the contract specifics and raw log

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
func (it *OptimismPortalWithdrawalProvenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalWithdrawalProven)
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
		it.Event = new(OptimismPortalWithdrawalProven)
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
func (it *OptimismPortalWithdrawalProvenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalWithdrawalProvenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalWithdrawalProven represents a WithdrawalProven event raised by the OptimismPortal contract.
type OptimismPortalWithdrawalProven struct {
	WithdrawalHash [32]byte
	From           common.Address
	To             common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalProven is a free log retrieval operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal *OptimismPortalFilterer) FilterWithdrawalProven(opts *bind.FilterOpts, withdrawalHash [][32]byte, from []common.Address, to []common.Address) (*OptimismPortalWithdrawalProvenIterator, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "WithdrawalProven", withdrawalHashRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalWithdrawalProvenIterator{contract: _OptimismPortal.contract, event: "WithdrawalProven", logs: logs, sub: sub}, nil
}

// WatchWithdrawalProven is a free log subscription operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal *OptimismPortalFilterer) WatchWithdrawalProven(opts *bind.WatchOpts, sink chan<- *OptimismPortalWithdrawalProven, withdrawalHash [][32]byte, from []common.Address, to []common.Address) (event.Subscription, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "WithdrawalProven", withdrawalHashRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalWithdrawalProven)
				if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalProven", log); err != nil {
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

// ParseWithdrawalProven is a log parse operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal *OptimismPortalFilterer) ParseWithdrawalProven(log types.Log) (*OptimismPortalWithdrawalProven, error) {
	event := new(OptimismPortalWithdrawalProven)
	if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalProven", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
