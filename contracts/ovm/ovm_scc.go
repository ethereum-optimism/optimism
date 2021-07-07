// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package OVM_SCC

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

// OVMSCCABI is the input ABI used to generate the binding from.
const OVMSCCABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fraudProofWindow\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_sequencerPublishWindow\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"StateBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"}],\"name\":\"StateBatchDeleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"FRAUD_PROOF_WINDOW\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SEQUENCER_PUBLISH_WINDOW\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"_batch\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"_shouldStartAtElement\",\"type\":\"uint256\"}],\"name\":\"appendStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractiOVM_ChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"deleteStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastSequencerTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_lastSequencerTimestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"insideFraudProofWindow\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"_inside\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_element\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"verifyStateCommitment\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// OVMSCC is an auto generated Go binding around an Ethereum contract.
type OVMSCC struct {
	OVMSCCCaller     // Read-only binding to the contract
	OVMSCCTransactor // Write-only binding to the contract
	OVMSCCFilterer   // Log filterer for contract events
}

// OVMSCCCaller is an auto generated read-only Go binding around an Ethereum contract.
type OVMSCCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMSCCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OVMSCCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMSCCFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OVMSCCFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OVMSCCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OVMSCCSession struct {
	Contract     *OVMSCC           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OVMSCCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OVMSCCCallerSession struct {
	Contract *OVMSCCCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OVMSCCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OVMSCCTransactorSession struct {
	Contract     *OVMSCCTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OVMSCCRaw is an auto generated low-level Go binding around an Ethereum contract.
type OVMSCCRaw struct {
	Contract *OVMSCC // Generic contract binding to access the raw methods on
}

// OVMSCCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OVMSCCCallerRaw struct {
	Contract *OVMSCCCaller // Generic read-only contract binding to access the raw methods on
}

// OVMSCCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OVMSCCTransactorRaw struct {
	Contract *OVMSCCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOVMSCC creates a new instance of OVMSCC, bound to a specific deployed contract.
func NewOVMSCC(address common.Address, backend bind.ContractBackend) (*OVMSCC, error) {
	contract, err := bindOVMSCC(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OVMSCC{OVMSCCCaller: OVMSCCCaller{contract: contract}, OVMSCCTransactor: OVMSCCTransactor{contract: contract}, OVMSCCFilterer: OVMSCCFilterer{contract: contract}}, nil
}

// NewOVMSCCCaller creates a new read-only instance of OVMSCC, bound to a specific deployed contract.
func NewOVMSCCCaller(address common.Address, caller bind.ContractCaller) (*OVMSCCCaller, error) {
	contract, err := bindOVMSCC(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OVMSCCCaller{contract: contract}, nil
}

// NewOVMSCCTransactor creates a new write-only instance of OVMSCC, bound to a specific deployed contract.
func NewOVMSCCTransactor(address common.Address, transactor bind.ContractTransactor) (*OVMSCCTransactor, error) {
	contract, err := bindOVMSCC(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OVMSCCTransactor{contract: contract}, nil
}

// NewOVMSCCFilterer creates a new log filterer instance of OVMSCC, bound to a specific deployed contract.
func NewOVMSCCFilterer(address common.Address, filterer bind.ContractFilterer) (*OVMSCCFilterer, error) {
	contract, err := bindOVMSCC(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OVMSCCFilterer{contract: contract}, nil
}

// bindOVMSCC binds a generic wrapper to an already deployed contract.
func bindOVMSCC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OVMSCCABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVMSCC *OVMSCCRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVMSCC.Contract.OVMSCCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVMSCC *OVMSCCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVMSCC.Contract.OVMSCCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVMSCC *OVMSCCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVMSCC.Contract.OVMSCCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OVMSCC *OVMSCCCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OVMSCC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OVMSCC *OVMSCCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OVMSCC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OVMSCC *OVMSCCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OVMSCC.Contract.contract.Transact(opts, method, params...)
}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCCaller) FRAUDPROOFWINDOW(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "FRAUD_PROOF_WINDOW")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCSession) FRAUDPROOFWINDOW() (*big.Int, error) {
	return _OVMSCC.Contract.FRAUDPROOFWINDOW(&_OVMSCC.CallOpts)
}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCCallerSession) FRAUDPROOFWINDOW() (*big.Int, error) {
	return _OVMSCC.Contract.FRAUDPROOFWINDOW(&_OVMSCC.CallOpts)
}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCCaller) SEQUENCERPUBLISHWINDOW(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "SEQUENCER_PUBLISH_WINDOW")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCSession) SEQUENCERPUBLISHWINDOW() (*big.Int, error) {
	return _OVMSCC.Contract.SEQUENCERPUBLISHWINDOW(&_OVMSCC.CallOpts)
}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OVMSCC *OVMSCCCallerSession) SEQUENCERPUBLISHWINDOW() (*big.Int, error) {
	return _OVMSCC.Contract.SEQUENCERPUBLISHWINDOW(&_OVMSCC.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMSCC *OVMSCCCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "batches")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMSCC *OVMSCCSession) Batches() (common.Address, error) {
	return _OVMSCC.Contract.Batches(&_OVMSCC.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OVMSCC *OVMSCCCallerSession) Batches() (common.Address, error) {
	return _OVMSCC.Contract.Batches(&_OVMSCC.CallOpts)
}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OVMSCC *OVMSCCCaller) GetLastSequencerTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "getLastSequencerTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OVMSCC *OVMSCCSession) GetLastSequencerTimestamp() (*big.Int, error) {
	return _OVMSCC.Contract.GetLastSequencerTimestamp(&_OVMSCC.CallOpts)
}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OVMSCC *OVMSCCCallerSession) GetLastSequencerTimestamp() (*big.Int, error) {
	return _OVMSCC.Contract.GetLastSequencerTimestamp(&_OVMSCC.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMSCC *OVMSCCCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "getTotalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMSCC *OVMSCCSession) GetTotalBatches() (*big.Int, error) {
	return _OVMSCC.Contract.GetTotalBatches(&_OVMSCC.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OVMSCC *OVMSCCCallerSession) GetTotalBatches() (*big.Int, error) {
	return _OVMSCC.Contract.GetTotalBatches(&_OVMSCC.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMSCC *OVMSCCCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "getTotalElements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMSCC *OVMSCCSession) GetTotalElements() (*big.Int, error) {
	return _OVMSCC.Contract.GetTotalElements(&_OVMSCC.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OVMSCC *OVMSCCCallerSession) GetTotalElements() (*big.Int, error) {
	return _OVMSCC.Contract.GetTotalElements(&_OVMSCC.CallOpts)
}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OVMSCC *OVMSCCCaller) InsideFraudProofWindow(opts *bind.CallOpts, _batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "insideFraudProofWindow", _batchHeader)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OVMSCC *OVMSCCSession) InsideFraudProofWindow(_batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	return _OVMSCC.Contract.InsideFraudProofWindow(&_OVMSCC.CallOpts, _batchHeader)
}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OVMSCC *OVMSCCCallerSession) InsideFraudProofWindow(_batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	return _OVMSCC.Contract.InsideFraudProofWindow(&_OVMSCC.CallOpts, _batchHeader)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMSCC *OVMSCCCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "libAddressManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMSCC *OVMSCCSession) LibAddressManager() (common.Address, error) {
	return _OVMSCC.Contract.LibAddressManager(&_OVMSCC.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OVMSCC *OVMSCCCallerSession) LibAddressManager() (common.Address, error) {
	return _OVMSCC.Contract.LibAddressManager(&_OVMSCC.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMSCC *OVMSCCCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "resolve", _name)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMSCC *OVMSCCSession) Resolve(_name string) (common.Address, error) {
	return _OVMSCC.Contract.Resolve(&_OVMSCC.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OVMSCC *OVMSCCCallerSession) Resolve(_name string) (common.Address, error) {
	return _OVMSCC.Contract.Resolve(&_OVMSCC.CallOpts, _name)
}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OVMSCC *OVMSCCCaller) VerifyStateCommitment(opts *bind.CallOpts, _element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	var out []interface{}
	err := _OVMSCC.contract.Call(opts, &out, "verifyStateCommitment", _element, _batchHeader, _proof)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OVMSCC *OVMSCCSession) VerifyStateCommitment(_element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OVMSCC.Contract.VerifyStateCommitment(&_OVMSCC.CallOpts, _element, _batchHeader, _proof)
}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OVMSCC *OVMSCCCallerSession) VerifyStateCommitment(_element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OVMSCC.Contract.VerifyStateCommitment(&_OVMSCC.CallOpts, _element, _batchHeader, _proof)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OVMSCC *OVMSCCTransactor) AppendStateBatch(opts *bind.TransactOpts, _batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OVMSCC.contract.Transact(opts, "appendStateBatch", _batch, _shouldStartAtElement)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OVMSCC *OVMSCCSession) AppendStateBatch(_batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OVMSCC.Contract.AppendStateBatch(&_OVMSCC.TransactOpts, _batch, _shouldStartAtElement)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OVMSCC *OVMSCCTransactorSession) AppendStateBatch(_batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OVMSCC.Contract.AppendStateBatch(&_OVMSCC.TransactOpts, _batch, _shouldStartAtElement)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OVMSCC *OVMSCCTransactor) DeleteStateBatch(opts *bind.TransactOpts, _batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OVMSCC.contract.Transact(opts, "deleteStateBatch", _batchHeader)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OVMSCC *OVMSCCSession) DeleteStateBatch(_batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OVMSCC.Contract.DeleteStateBatch(&_OVMSCC.TransactOpts, _batchHeader)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OVMSCC *OVMSCCTransactorSession) DeleteStateBatch(_batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OVMSCC.Contract.DeleteStateBatch(&_OVMSCC.TransactOpts, _batchHeader)
}

// OVMSCCStateBatchAppendedIterator is returned from FilterStateBatchAppended and is used to iterate over the raw logs and unpacked data for StateBatchAppended events raised by the OVMSCC contract.
type OVMSCCStateBatchAppendedIterator struct {
	Event *OVMSCCStateBatchAppended // Event containing the contract specifics and raw log

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
func (it *OVMSCCStateBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMSCCStateBatchAppended)
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
		it.Event = new(OVMSCCStateBatchAppended)
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
func (it *OVMSCCStateBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMSCCStateBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMSCCStateBatchAppended represents a StateBatchAppended event raised by the OVMSCC contract.
type OVMSCCStateBatchAppended struct {
	BatchIndex        *big.Int
	BatchRoot         [32]byte
	BatchSize         *big.Int
	PrevTotalElements *big.Int
	ExtraData         []byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterStateBatchAppended is a free log retrieval operation binding the contract event 0x16be4c5129a4e03cf3350262e181dc02ddfb4a6008d925368c0899fcd97ca9c5.
//
// Solidity: event StateBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OVMSCC *OVMSCCFilterer) FilterStateBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OVMSCCStateBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMSCC.contract.FilterLogs(opts, "StateBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OVMSCCStateBatchAppendedIterator{contract: _OVMSCC.contract, event: "StateBatchAppended", logs: logs, sub: sub}, nil
}

// WatchStateBatchAppended is a free log subscription operation binding the contract event 0x16be4c5129a4e03cf3350262e181dc02ddfb4a6008d925368c0899fcd97ca9c5.
//
// Solidity: event StateBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OVMSCC *OVMSCCFilterer) WatchStateBatchAppended(opts *bind.WatchOpts, sink chan<- *OVMSCCStateBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMSCC.contract.WatchLogs(opts, "StateBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMSCCStateBatchAppended)
				if err := _OVMSCC.contract.UnpackLog(event, "StateBatchAppended", log); err != nil {
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

// ParseStateBatchAppended is a log parse operation binding the contract event 0x16be4c5129a4e03cf3350262e181dc02ddfb4a6008d925368c0899fcd97ca9c5.
//
// Solidity: event StateBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OVMSCC *OVMSCCFilterer) ParseStateBatchAppended(log types.Log) (*OVMSCCStateBatchAppended, error) {
	event := new(OVMSCCStateBatchAppended)
	if err := _OVMSCC.contract.UnpackLog(event, "StateBatchAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OVMSCCStateBatchDeletedIterator is returned from FilterStateBatchDeleted and is used to iterate over the raw logs and unpacked data for StateBatchDeleted events raised by the OVMSCC contract.
type OVMSCCStateBatchDeletedIterator struct {
	Event *OVMSCCStateBatchDeleted // Event containing the contract specifics and raw log

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
func (it *OVMSCCStateBatchDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OVMSCCStateBatchDeleted)
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
		it.Event = new(OVMSCCStateBatchDeleted)
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
func (it *OVMSCCStateBatchDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OVMSCCStateBatchDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OVMSCCStateBatchDeleted represents a StateBatchDeleted event raised by the OVMSCC contract.
type OVMSCCStateBatchDeleted struct {
	BatchIndex *big.Int
	BatchRoot  [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterStateBatchDeleted is a free log retrieval operation binding the contract event 0x8747b69ce8fdb31c3b9b0a67bd8049ad8c1a69ea417b69b12174068abd9cbd64.
//
// Solidity: event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
func (_OVMSCC *OVMSCCFilterer) FilterStateBatchDeleted(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OVMSCCStateBatchDeletedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMSCC.contract.FilterLogs(opts, "StateBatchDeleted", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OVMSCCStateBatchDeletedIterator{contract: _OVMSCC.contract, event: "StateBatchDeleted", logs: logs, sub: sub}, nil
}

// WatchStateBatchDeleted is a free log subscription operation binding the contract event 0x8747b69ce8fdb31c3b9b0a67bd8049ad8c1a69ea417b69b12174068abd9cbd64.
//
// Solidity: event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
func (_OVMSCC *OVMSCCFilterer) WatchStateBatchDeleted(opts *bind.WatchOpts, sink chan<- *OVMSCCStateBatchDeleted, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OVMSCC.contract.WatchLogs(opts, "StateBatchDeleted", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OVMSCCStateBatchDeleted)
				if err := _OVMSCC.contract.UnpackLog(event, "StateBatchDeleted", log); err != nil {
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

// ParseStateBatchDeleted is a log parse operation binding the contract event 0x8747b69ce8fdb31c3b9b0a67bd8049ad8c1a69ea417b69b12174068abd9cbd64.
//
// Solidity: event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
func (_OVMSCC *OVMSCCFilterer) ParseStateBatchDeleted(log types.Log) (*OVMSCCStateBatchDeleted, error) {
	event := new(OVMSCCStateBatchDeleted)
	if err := _OVMSCC.contract.UnpackLog(event, "StateBatchDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
