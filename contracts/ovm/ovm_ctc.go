// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package OVM_CTC

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Lib_OVMCodecChainBatchHeader is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecChainBatchHeader struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
}

// Lib_OVMCodecChainInclusionProof is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecChainInclusionProof struct {
	Index    *big.Int
	Siblings [][32]byte
}

// Lib_OVMCodecQueueElement is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecQueueElement struct {
	TransactionHash [32]byte
	Timestamp       *big.Int
	BlockNumber     *big.Int
}

// Lib_OVMCodecTransaction is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecTransaction struct {
	Timestamp     *big.Int
	BlockNumber   *big.Int
	L1QueueOrigin uint8
	L1TxOrigin    common.Address
	Entrypoint    common.Address
	GasLimit      *big.Int
	Data          []byte
}

// Lib_OVMCodecTransactionChainElement is an auto generated low-level Go binding around an user-defined struct.
type Lib_OVMCodecTransactionChainElement struct {
	IsSequenced bool
	QueueIndex  *big.Int
	Timestamp   *big.Int
	BlockNumber *big.Int
	TxData      []byte
}

// OVMCTCABI is the input ABI used to generate the binding from.
const OVMCTCABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_forceInclusionPeriodSeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_forceInclusionPeriodBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxTransactionGasLimit\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"QueueBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_startingQueueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_numQueueElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"name\":\"SequencerBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"TransactionBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_l1TxOrigin\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"}],\"name\":\"TransactionEnqueued\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"L2_GAS_DISCOUNT_DIVISOR\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_ROLLUP_TX_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_ROLLUP_TX_GAS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"appendQueueBatch\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"appendSequencerBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractiOVM_ChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"enqueue\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"forceInclusionPeriodBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"forceInclusionPeriodSeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockNumber\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastTimestamp\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextQueueIndex\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNumPendingQueueElements\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"getQueueElement\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"transactionHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint40\",\"name\":\"timestamp\",\"type\":\"uint40\"},{\"internalType\":\"uint40\",\"name\":\"blockNumber\",\"type\":\"uint40\"}],\"internalType\":\"structLib_OVMCodec.QueueElement\",\"name\":\"_element\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getQueueLength\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxTransactionGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"queue\",\"outputs\":[{\"internalType\":\"contractiOVM_ChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"enumLib_OVMCodec.QueueOrigin\",\"name\":\"l1QueueOrigin\",\"type\":\"uint8\"},{\"internalType\":\"address\",\"name\":\"l1TxOrigin\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"entrypoint\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.Transaction\",\"name\":\"_transaction\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isSequenced\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"txData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.TransactionChainElement\",\"name\":\"_txChainElement\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"_inclusionProof\",\"type\":\"tuple\"}],\"name\":\"verifyTransaction\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// OVMCTC is an auto generated Go binding around an Ethereum contract.
type OVMCTC struct {
	OVMCTCCaller     // Read-only binding to the contract
	OVMCTCTransactor // Write-only binding to the contract
	OVMCTCFilterer   // Log filterer for contract events
}

// OVMCTCCaller is an auto generated read-only Go binding around an Ethereum contract.
type OVMCTCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMCTCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OVMCTCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMCTCFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OVMCTCFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMCTCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OVMCTCSession struct {
	Contract     *OVMCTC           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OVMCTCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OVMCTCCallerSession struct {
	Contract *OVMCTCCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OVMCTCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OVMCTCTransactorSession struct {
	Contract     *OVMCTCTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OVMCTCRaw is an auto generated low-level Go binding around an Ethereum contract.
type OVMCTCRaw struct {
	Contract *OVMCTC // Generic contract binding to access the raw methods on
}

// OVMCTCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OVMCTCCallerRaw struct {
	Contract *OVMCTCCaller // Generic read-only contract binding to access the raw methods on
}

// OVMCTCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OVMCTCTransactorRaw struct {
	Contract *OVMCTCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOVMCTC creates a new instance of OVMCTC, bound to a specific deployed contract.
func NewOVMCTC(address common.Address, backend bind.ContractBackend) (*OVMCTC, error) {
	contract, err := bindOVMCTC(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OVMCTC{OVMCTCCaller: OVMCTCCaller{contract: contract}, OVMCTCTransactor: OVMCTCTransactor{contract: contract}, OVMCTCFilterer: OVMCTCFilterer{contract: contract}}, nil
}

// NewOVMCTCCaller creates a new read-only instance of OVMCTC, bound to a specific deployed contract.
func NewOVMCTCCaller(address common.Address, caller bind.ContractCaller) (*OVMCTCCaller, error) {
	contract, err := bindOVMCTC(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OVMCTCCaller{contract: contract}, nil
}

// NewOVMCTCTransactor creates a new write-only instance of OVMCTC, bound to a specific deployed contract.
func NewOVMCTCTransactor(address common.Address, transactor bind.ContractTransactor) (*OVMCTCTransactor, error) {
	contract, err := bindOVMCTC(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OVMCTCTransactor{contract: contract}, nil
}

// NewOVMCTCFilterer creates a new log filterer instance of OVMCTC, bound to a specific deployed contract.
func NewOVMCTCFilterer(address common.Address, filterer bind.ContractFilterer) (*OVMCTCFilterer, error) {
	contract, err := bindOVMCTC(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OVMCTCFilterer{contract: contract}, nil
}

// bindOVMCTC binds a generic wrapper to an already deployed contract.
func bindOVMCTC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OVMCTCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVMCTC *OVMCTCRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVMCTC.Contract.OVMCTCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVMCTC *OVMCTCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVMCTC.Contract.OVMCTCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVMCTC *OVMCTCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVMCTC.Contract.OVMCTCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVMCTC *OVMCTCCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVMCTC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVMCTC *OVMCTCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVMCTC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVMCTC *OVMCTCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVMCTC.Contract.contract.Transact(opts, method, params...)
}

// L2GASDISCOUNTDIVISOR is a free data retrieval call binding the contract method 0xc2cf696f.
//
// Solidity: function L2_GAS_DISCOUNT_DIVISOR() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) L2GASDISCOUNTDIVISOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "L2_GAS_DISCOUNT_DIVISOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2GASDISCOUNTDIVISOR is a free data retrieval call binding the contract method 0xc2cf696f.
//
// Solidity: function L2_GAS_DISCOUNT_DIVISOR() view returns(uint256)
func (_OVMCTC *OVMCTCSession) L2GASDISCOUNTDIVISOR() (*big.Int, error) {
	return _OVMCTC.Contract.L2GASDISCOUNTDIVISOR(&_OVMCTC.CallOpts)
}

// L2GASDISCOUNTDIVISOR is a free data retrieval call binding the contract method 0xc2cf696f.
//
// Solidity: function L2_GAS_DISCOUNT_DIVISOR() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) L2GASDISCOUNTDIVISOR() (*big.Int, error) {
	return _OVMCTC.Contract.L2GASDISCOUNTDIVISOR(&_OVMCTC.CallOpts)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) MAXROLLUPTXSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "MAX_ROLLUP_TX_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OVMCTC *OVMCTCSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _OVMCTC.Contract.MAXROLLUPTXSIZE(&_OVMCTC.CallOpts)
}

// MAXROLLUPTXSIZE is a free data retrieval call binding the contract method 0x876ed5cb.
//
// Solidity: function MAX_ROLLUP_TX_SIZE() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) MAXROLLUPTXSIZE() (*big.Int, error) {
	return _OVMCTC.Contract.MAXROLLUPTXSIZE(&_OVMCTC.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) MINROLLUPTXGAS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "MIN_ROLLUP_TX_GAS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OVMCTC *OVMCTCSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _OVMCTC.Contract.MINROLLUPTXGAS(&_OVMCTC.CallOpts)
}

// MINROLLUPTXGAS is a free data retrieval call binding the contract method 0x78f4b2f2.
//
// Solidity: function MIN_ROLLUP_TX_GAS() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) MINROLLUPTXGAS() (*big.Int, error) {
	return _OVMCTC.Contract.MINROLLUPTXGAS(&_OVMCTC.CallOpts)
}

// AppendQueueBatch is a free data retrieval call binding the contract method 0xfacdc5da.
//
// Solidity: function appendQueueBatch(uint256 ) pure returns()
func (_OVMCTC *OVMCTCCaller) AppendQueueBatch(opts *bind.CallOpts, arg0 *big.Int) error {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "appendQueueBatch", arg0)

	if err != nil {
		return err
	}

	return err

}

// AppendQueueBatch is a free data retrieval call binding the contract method 0xfacdc5da.
//
// Solidity: function appendQueueBatch(uint256 ) pure returns()
func (_OVMCTC *OVMCTCSession) AppendQueueBatch(arg0 *big.Int) error {
	return _OVMCTC.Contract.AppendQueueBatch(&_OVMCTC.CallOpts, arg0)
}

// AppendQueueBatch is a free data retrieval call binding the contract method 0xfacdc5da.
//
// Solidity: function appendQueueBatch(uint256 ) pure returns()
func (_OVMCTC *OVMCTCCallerSession) AppendQueueBatch(arg0 *big.Int) error {
	return _OVMCTC.Contract.AppendQueueBatch(&_OVMCTC.CallOpts, arg0)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMCTC *OVMCTCCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "batches")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMCTC *OVMCTCSession) Batches() (common.Address, error) {
	return _OVMCTC.Contract.Batches(&_OVMCTC.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMCTC *OVMCTCCallerSession) Batches() (common.Address, error) {
	return _OVMCTC.Contract.Batches(&_OVMCTC.CallOpts)
}

// ForceInclusionPeriodBlocks is a free data retrieval call binding the contract method 0x138387a4.
//
// Solidity: function forceInclusionPeriodBlocks() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) ForceInclusionPeriodBlocks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "forceInclusionPeriodBlocks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ForceInclusionPeriodBlocks is a free data retrieval call binding the contract method 0x138387a4.
//
// Solidity: function forceInclusionPeriodBlocks() view returns(uint256)
func (_OVMCTC *OVMCTCSession) ForceInclusionPeriodBlocks() (*big.Int, error) {
	return _OVMCTC.Contract.ForceInclusionPeriodBlocks(&_OVMCTC.CallOpts)
}

// ForceInclusionPeriodBlocks is a free data retrieval call binding the contract method 0x138387a4.
//
// Solidity: function forceInclusionPeriodBlocks() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) ForceInclusionPeriodBlocks() (*big.Int, error) {
	return _OVMCTC.Contract.ForceInclusionPeriodBlocks(&_OVMCTC.CallOpts)
}

// ForceInclusionPeriodSeconds is a free data retrieval call binding the contract method 0xc139eb15.
//
// Solidity: function forceInclusionPeriodSeconds() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) ForceInclusionPeriodSeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "forceInclusionPeriodSeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ForceInclusionPeriodSeconds is a free data retrieval call binding the contract method 0xc139eb15.
//
// Solidity: function forceInclusionPeriodSeconds() view returns(uint256)
func (_OVMCTC *OVMCTCSession) ForceInclusionPeriodSeconds() (*big.Int, error) {
	return _OVMCTC.Contract.ForceInclusionPeriodSeconds(&_OVMCTC.CallOpts)
}

// ForceInclusionPeriodSeconds is a free data retrieval call binding the contract method 0xc139eb15.
//
// Solidity: function forceInclusionPeriodSeconds() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) ForceInclusionPeriodSeconds() (*big.Int, error) {
	return _OVMCTC.Contract.ForceInclusionPeriodSeconds(&_OVMCTC.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OVMCTC *OVMCTCCaller) GetLastBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getLastBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OVMCTC *OVMCTCSession) GetLastBlockNumber() (*big.Int, error) {
	return _OVMCTC.Contract.GetLastBlockNumber(&_OVMCTC.CallOpts)
}

// GetLastBlockNumber is a free data retrieval call binding the contract method 0x5ae6256d.
//
// Solidity: function getLastBlockNumber() view returns(uint40)
func (_OVMCTC *OVMCTCCallerSession) GetLastBlockNumber() (*big.Int, error) {
	return _OVMCTC.Contract.GetLastBlockNumber(&_OVMCTC.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OVMCTC *OVMCTCCaller) GetLastTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getLastTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OVMCTC *OVMCTCSession) GetLastTimestamp() (*big.Int, error) {
	return _OVMCTC.Contract.GetLastTimestamp(&_OVMCTC.CallOpts)
}

// GetLastTimestamp is a free data retrieval call binding the contract method 0x37899770.
//
// Solidity: function getLastTimestamp() view returns(uint40)
func (_OVMCTC *OVMCTCCallerSession) GetLastTimestamp() (*big.Int, error) {
	return _OVMCTC.Contract.GetLastTimestamp(&_OVMCTC.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OVMCTC *OVMCTCCaller) GetNextQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getNextQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OVMCTC *OVMCTCSession) GetNextQueueIndex() (*big.Int, error) {
	return _OVMCTC.Contract.GetNextQueueIndex(&_OVMCTC.CallOpts)
}

// GetNextQueueIndex is a free data retrieval call binding the contract method 0x7a167a8a.
//
// Solidity: function getNextQueueIndex() view returns(uint40)
func (_OVMCTC *OVMCTCCallerSession) GetNextQueueIndex() (*big.Int, error) {
	return _OVMCTC.Contract.GetNextQueueIndex(&_OVMCTC.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OVMCTC *OVMCTCCaller) GetNumPendingQueueElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getNumPendingQueueElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OVMCTC *OVMCTCSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _OVMCTC.Contract.GetNumPendingQueueElements(&_OVMCTC.CallOpts)
}

// GetNumPendingQueueElements is a free data retrieval call binding the contract method 0xf722b41a.
//
// Solidity: function getNumPendingQueueElements() view returns(uint40)
func (_OVMCTC *OVMCTCCallerSession) GetNumPendingQueueElements() (*big.Int, error) {
	return _OVMCTC.Contract.GetNumPendingQueueElements(&_OVMCTC.CallOpts)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OVMCTC *OVMCTCCaller) GetQueueElement(opts *bind.CallOpts, _index *big.Int) (Lib_OVMCodecQueueElement, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getQueueElement", _index)

	if err != nil {
		return *new(Lib_OVMCodecQueueElement), err
	}

	out0 := *abi.ConvertType(out[0], new(Lib_OVMCodecQueueElement)).(*Lib_OVMCodecQueueElement)

	return out0, err

}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OVMCTC *OVMCTCSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _OVMCTC.Contract.GetQueueElement(&_OVMCTC.CallOpts, _index)
}

// GetQueueElement is a free data retrieval call binding the contract method 0x2a7f18be.
//
// Solidity: function getQueueElement(uint256 _index) view returns((bytes32,uint40,uint40) _element)
func (_OVMCTC *OVMCTCCallerSession) GetQueueElement(_index *big.Int) (Lib_OVMCodecQueueElement, error) {
	return _OVMCTC.Contract.GetQueueElement(&_OVMCTC.CallOpts, _index)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OVMCTC *OVMCTCCaller) GetQueueLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getQueueLength")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OVMCTC *OVMCTCSession) GetQueueLength() (*big.Int, error) {
	return _OVMCTC.Contract.GetQueueLength(&_OVMCTC.CallOpts)
}

// GetQueueLength is a free data retrieval call binding the contract method 0xb8f77005.
//
// Solidity: function getQueueLength() view returns(uint40)
func (_OVMCTC *OVMCTCCallerSession) GetQueueLength() (*big.Int, error) {
	return _OVMCTC.Contract.GetQueueLength(&_OVMCTC.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMCTC *OVMCTCCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getTotalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMCTC *OVMCTCSession) GetTotalBatches() (*big.Int, error) {
	return _OVMCTC.Contract.GetTotalBatches(&_OVMCTC.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMCTC *OVMCTCCallerSession) GetTotalBatches() (*big.Int, error) {
	return _OVMCTC.Contract.GetTotalBatches(&_OVMCTC.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMCTC *OVMCTCCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "getTotalElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMCTC *OVMCTCSession) GetTotalElements() (*big.Int, error) {
	return _OVMCTC.Contract.GetTotalElements(&_OVMCTC.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMCTC *OVMCTCCallerSession) GetTotalElements() (*big.Int, error) {
	return _OVMCTC.Contract.GetTotalElements(&_OVMCTC.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMCTC *OVMCTCCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMCTC *OVMCTCSession) LibAddressManager() (common.Address, error) {
	return _OVMCTC.Contract.LibAddressManager(&_OVMCTC.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMCTC *OVMCTCCallerSession) LibAddressManager() (common.Address, error) {
	return _OVMCTC.Contract.LibAddressManager(&_OVMCTC.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OVMCTC *OVMCTCCaller) MaxTransactionGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "maxTransactionGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OVMCTC *OVMCTCSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _OVMCTC.Contract.MaxTransactionGasLimit(&_OVMCTC.CallOpts)
}

// MaxTransactionGasLimit is a free data retrieval call binding the contract method 0x8d38c6c1.
//
// Solidity: function maxTransactionGasLimit() view returns(uint256)
func (_OVMCTC *OVMCTCCallerSession) MaxTransactionGasLimit() (*big.Int, error) {
	return _OVMCTC.Contract.MaxTransactionGasLimit(&_OVMCTC.CallOpts)
}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_OVMCTC *OVMCTCCaller) Queue(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "queue")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_OVMCTC *OVMCTCSession) Queue() (common.Address, error) {
	return _OVMCTC.Contract.Queue(&_OVMCTC.CallOpts)
}

// Queue is a free data retrieval call binding the contract method 0xe10d29ee.
//
// Solidity: function queue() view returns(address)
func (_OVMCTC *OVMCTCCallerSession) Queue() (common.Address, error) {
	return _OVMCTC.Contract.Queue(&_OVMCTC.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMCTC *OVMCTCCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMCTC *OVMCTCSession) Resolve(_name string) (common.Address, error) {
	return _OVMCTC.Contract.Resolve(&_OVMCTC.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMCTC *OVMCTCCallerSession) Resolve(_name string) (common.Address, error) {
	return _OVMCTC.Contract.Resolve(&_OVMCTC.CallOpts, _name)
}

// VerifyTransaction is a free data retrieval call binding the contract method 0x4de569ce.
//
// Solidity: function verifyTransaction((uint256,uint256,uint8,address,address,uint256,bytes) _transaction, (bool,uint256,uint256,uint256,bytes) _txChainElement, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _inclusionProof) view returns(bool)
func (_OVMCTC *OVMCTCCaller) VerifyTransaction(opts *bind.CallOpts, _transaction Lib_OVMCodecTransaction, _txChainElement Lib_OVMCodecTransactionChainElement, _batchHeader Lib_OVMCodecChainBatchHeader, _inclusionProof Lib_OVMCodecChainInclusionProof) (bool, error) {
	var out []interface{}
	err := _OVMCTC.contract.Call(opts, &out, "verifyTransaction", _transaction, _txChainElement, _batchHeader, _inclusionProof)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifyTransaction is a free data retrieval call binding the contract method 0x4de569ce.
//
// Solidity: function verifyTransaction((uint256,uint256,uint8,address,address,uint256,bytes) _transaction, (bool,uint256,uint256,uint256,bytes) _txChainElement, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _inclusionProof) view returns(bool)
func (_OVMCTC *OVMCTCSession) VerifyTransaction(_transaction Lib_OVMCodecTransaction, _txChainElement Lib_OVMCodecTransactionChainElement, _batchHeader Lib_OVMCodecChainBatchHeader, _inclusionProof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OVMCTC.Contract.VerifyTransaction(&_OVMCTC.CallOpts, _transaction, _txChainElement, _batchHeader, _inclusionProof)
}

// VerifyTransaction is a free data retrieval call binding the contract method 0x4de569ce.
//
// Solidity: function verifyTransaction((uint256,uint256,uint8,address,address,uint256,bytes) _transaction, (bool,uint256,uint256,uint256,bytes) _txChainElement, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _inclusionProof) view returns(bool)
func (_OVMCTC *OVMCTCCallerSession) VerifyTransaction(_transaction Lib_OVMCodecTransaction, _txChainElement Lib_OVMCodecTransactionChainElement, _batchHeader Lib_OVMCodecChainBatchHeader, _inclusionProof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OVMCTC.Contract.VerifyTransaction(&_OVMCTC.CallOpts, _transaction, _txChainElement, _batchHeader, _inclusionProof)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OVMCTC *OVMCTCTransactor) AppendSequencerBatch(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVMCTC.contract.Transact(opts, "appendSequencerBatch")
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OVMCTC *OVMCTCSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _OVMCTC.Contract.AppendSequencerBatch(&_OVMCTC.TransactOpts)
}

// AppendSequencerBatch is a paid mutator transaction binding the contract method 0xd0f89344.
//
// Solidity: function appendSequencerBatch() returns()
func (_OVMCTC *OVMCTCTransactorSession) AppendSequencerBatch() (*types.Transaction, error) {
	return _OVMCTC.Contract.AppendSequencerBatch(&_OVMCTC.TransactOpts)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OVMCTC *OVMCTCTransactor) Enqueue(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OVMCTC.contract.Transact(opts, "enqueue", _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OVMCTC *OVMCTCSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OVMCTC.Contract.Enqueue(&_OVMCTC.TransactOpts, _target, _gasLimit, _data)
}

// Enqueue is a paid mutator transaction binding the contract method 0x6fee07e0.
//
// Solidity: function enqueue(address _target, uint256 _gasLimit, bytes _data) returns()
func (_OVMCTC *OVMCTCTransactorSession) Enqueue(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _OVMCTC.Contract.Enqueue(&_OVMCTC.TransactOpts, _target, _gasLimit, _data)
}

// OVMCTCQueueBatchAppendedIterator is returned from FilterQueueBatchAppended and is used to iterate over the raw logs and unpacked data for QueueBatchAppended events raised by the OVMCTC contract.
type OVMCTCQueueBatchAppendedIterator struct {
	Event *OVMCTCQueueBatchAppended // Event containing the contract specifics and raw log

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
func (it *OVMCTCQueueBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMCTCQueueBatchAppended)
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
		it.Event = new(OVMCTCQueueBatchAppended)
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
func (it *OVMCTCQueueBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMCTCQueueBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMCTCQueueBatchAppended represents a QueueBatchAppended event raised by the OVMCTC contract.
type OVMCTCQueueBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterQueueBatchAppended is a free log retrieval operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OVMCTC *OVMCTCFilterer) FilterQueueBatchAppended(opts *bind.FilterOpts) (*OVMCTCQueueBatchAppendedIterator, error) {

	logs, sub, err := _OVMCTC.contract.FilterLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return &OVMCTCQueueBatchAppendedIterator{contract: _OVMCTC.contract, event: "QueueBatchAppended", logs: logs, sub: sub}, nil
}

// WatchQueueBatchAppended is a free log subscription operation binding the contract event 0x64d7f508348c70dea42d5302a393987e4abc20e45954ab3f9d320207751956f0.
//
// Solidity: event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OVMCTC *OVMCTCFilterer) WatchQueueBatchAppended(opts *bind.WatchOpts, sink chan<- *OVMCTCQueueBatchAppended) (event.Subscription, error) {

	logs, sub, err := _OVMCTC.contract.WatchLogs(opts, "QueueBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMCTCQueueBatchAppended)
				if err := _OVMCTC.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
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
func (_OVMCTC *OVMCTCFilterer) ParseQueueBatchAppended(log types.Log) (*OVMCTCQueueBatchAppended, error) {
	event := new(OVMCTCQueueBatchAppended)
	if err := _OVMCTC.contract.UnpackLog(event, "QueueBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVMCTCSequencerBatchAppendedIterator is returned from FilterSequencerBatchAppended and is used to iterate over the raw logs and unpacked data for SequencerBatchAppended events raised by the OVMCTC contract.
type OVMCTCSequencerBatchAppendedIterator struct {
	Event *OVMCTCSequencerBatchAppended // Event containing the contract specifics and raw log

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
func (it *OVMCTCSequencerBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMCTCSequencerBatchAppended)
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
		it.Event = new(OVMCTCSequencerBatchAppended)
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
func (it *OVMCTCSequencerBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMCTCSequencerBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMCTCSequencerBatchAppended represents a SequencerBatchAppended event raised by the OVMCTC contract.
type OVMCTCSequencerBatchAppended struct {
	StartingQueueIndex *big.Int
	NumQueueElements   *big.Int
	TotalElements      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterSequencerBatchAppended is a free log retrieval operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OVMCTC *OVMCTCFilterer) FilterSequencerBatchAppended(opts *bind.FilterOpts) (*OVMCTCSequencerBatchAppendedIterator, error) {

	logs, sub, err := _OVMCTC.contract.FilterLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return &OVMCTCSequencerBatchAppendedIterator{contract: _OVMCTC.contract, event: "SequencerBatchAppended", logs: logs, sub: sub}, nil
}

// WatchSequencerBatchAppended is a free log subscription operation binding the contract event 0x602f1aeac0ca2e7a13e281a9ef0ad7838542712ce16780fa2ecffd351f05f899.
//
// Solidity: event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
func (_OVMCTC *OVMCTCFilterer) WatchSequencerBatchAppended(opts *bind.WatchOpts, sink chan<- *OVMCTCSequencerBatchAppended) (event.Subscription, error) {

	logs, sub, err := _OVMCTC.contract.WatchLogs(opts, "SequencerBatchAppended")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMCTCSequencerBatchAppended)
				if err := _OVMCTC.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
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
func (_OVMCTC *OVMCTCFilterer) ParseSequencerBatchAppended(log types.Log) (*OVMCTCSequencerBatchAppended, error) {
	event := new(OVMCTCSequencerBatchAppended)
	if err := _OVMCTC.contract.UnpackLog(event, "SequencerBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVMCTCTransactionBatchAppendedIterator is returned from FilterTransactionBatchAppended and is used to iterate over the raw logs and unpacked data for TransactionBatchAppended events raised by the OVMCTC contract.
type OVMCTCTransactionBatchAppendedIterator struct {
	Event *OVMCTCTransactionBatchAppended // Event containing the contract specifics and raw log

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
func (it *OVMCTCTransactionBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMCTCTransactionBatchAppended)
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
		it.Event = new(OVMCTCTransactionBatchAppended)
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
func (it *OVMCTCTransactionBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMCTCTransactionBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMCTCTransactionBatchAppended represents a TransactionBatchAppended event raised by the OVMCTC contract.
type OVMCTCTransactionBatchAppended struct {
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
func (_OVMCTC *OVMCTCFilterer) FilterTransactionBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OVMCTCTransactionBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMCTC.contract.FilterLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OVMCTCTransactionBatchAppendedIterator{contract: _OVMCTC.contract, event: "TransactionBatchAppended", logs: logs, sub: sub}, nil
}

// WatchTransactionBatchAppended is a free log subscription operation binding the contract event 0x127186556e7be68c7e31263195225b4de02820707889540969f62c05cf73525e.
//
// Solidity: event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OVMCTC *OVMCTCFilterer) WatchTransactionBatchAppended(opts *bind.WatchOpts, sink chan<- *OVMCTCTransactionBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMCTC.contract.WatchLogs(opts, "TransactionBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMCTCTransactionBatchAppended)
				if err := _OVMCTC.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
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
func (_OVMCTC *OVMCTCFilterer) ParseTransactionBatchAppended(log types.Log) (*OVMCTCTransactionBatchAppended, error) {
	event := new(OVMCTCTransactionBatchAppended)
	if err := _OVMCTC.contract.UnpackLog(event, "TransactionBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVMCTCTransactionEnqueuedIterator is returned from FilterTransactionEnqueued and is used to iterate over the raw logs and unpacked data for TransactionEnqueued events raised by the OVMCTC contract.
type OVMCTCTransactionEnqueuedIterator struct {
	Event *OVMCTCTransactionEnqueued // Event containing the contract specifics and raw log

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
func (it *OVMCTCTransactionEnqueuedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMCTCTransactionEnqueued)
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
		it.Event = new(OVMCTCTransactionEnqueued)
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
func (it *OVMCTCTransactionEnqueuedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMCTCTransactionEnqueuedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMCTCTransactionEnqueued represents a TransactionEnqueued event raised by the OVMCTC contract.
type OVMCTCTransactionEnqueued struct {
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
// Solidity: event TransactionEnqueued(address _l1TxOrigin, address _target, uint256 _gasLimit, bytes _data, uint256 _queueIndex, uint256 _timestamp)
func (_OVMCTC *OVMCTCFilterer) FilterTransactionEnqueued(opts *bind.FilterOpts) (*OVMCTCTransactionEnqueuedIterator, error) {

	logs, sub, err := _OVMCTC.contract.FilterLogs(opts, "TransactionEnqueued")
	if err != nil {
		return nil, err
	}
	return &OVMCTCTransactionEnqueuedIterator{contract: _OVMCTC.contract, event: "TransactionEnqueued", logs: logs, sub: sub}, nil
}

// WatchTransactionEnqueued is a free log subscription operation binding the contract event 0x4b388aecf9fa6cc92253704e5975a6129a4f735bdbd99567df4ed0094ee4ceb5.
//
// Solidity: event TransactionEnqueued(address _l1TxOrigin, address _target, uint256 _gasLimit, bytes _data, uint256 _queueIndex, uint256 _timestamp)
func (_OVMCTC *OVMCTCFilterer) WatchTransactionEnqueued(opts *bind.WatchOpts, sink chan<- *OVMCTCTransactionEnqueued) (event.Subscription, error) {

	logs, sub, err := _OVMCTC.contract.WatchLogs(opts, "TransactionEnqueued")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMCTCTransactionEnqueued)
				if err := _OVMCTC.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
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
// Solidity: event TransactionEnqueued(address _l1TxOrigin, address _target, uint256 _gasLimit, bytes _data, uint256 _queueIndex, uint256 _timestamp)
func (_OVMCTC *OVMCTCFilterer) ParseTransactionEnqueued(log types.Log) (*OVMCTCTransactionEnqueued, error) {
	event := new(OVMCTCTransactionEnqueued)
	if err := _OVMCTC.contract.UnpackLog(event, "TransactionEnqueued", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
