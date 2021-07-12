// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ovm_scc

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

// OvmSccABI is the input ABI used to generate the binding from.
const OvmSccABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_libAddressManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fraudProofWindow\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_sequencerPublishWindow\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_batchSize\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_prevTotalElements\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"StateBatchAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_batchIndex\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"_batchRoot\",\"type\":\"bytes32\"}],\"name\":\"StateBatchDeleted\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"FRAUD_PROOF_WINDOW\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SEQUENCER_PUBLISH_WINDOW\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"_batch\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"_shouldStartAtElement\",\"type\":\"uint256\"}],\"name\":\"appendStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"batches\",\"outputs\":[{\"internalType\":\"contractiOVM_ChainStorageContainer\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"deleteStateBatch\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastSequencerTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_lastSequencerTimestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalBatches\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalBatches\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getTotalElements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"_totalElements\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"}],\"name\":\"insideFraudProofWindow\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"_inside\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"libAddressManager\",\"outputs\":[{\"internalType\":\"contractLib_AddressManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_element\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"batchRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"batchSize\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"prevTotalElements\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"internalType\":\"structLib_OVMCodec.ChainBatchHeader\",\"name\":\"_batchHeader\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"}],\"internalType\":\"structLib_OVMCodec.ChainInclusionProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"verifyStateCommitment\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// OvmScc is an auto generated Go binding around an Ethereum contract.
type OvmScc struct {
	OvmSccCaller     // Read-only binding to the contract
	OvmSccTransactor // Write-only binding to the contract
	OvmSccFilterer   // Log filterer for contract events
}

// OvmSccCaller is an auto generated read-only Go binding around an Ethereum contract.
type OvmSccCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmSccTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OvmSccTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmSccFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OvmSccFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OvmSccSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OvmSccSession struct {
	Contract     *OvmScc           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OvmSccCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OvmSccCallerSession struct {
	Contract *OvmSccCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// OvmSccTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OvmSccTransactorSession struct {
	Contract     *OvmSccTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OvmSccRaw is an auto generated low-level Go binding around an Ethereum contract.
type OvmSccRaw struct {
	Contract *OvmScc // Generic contract binding to access the raw methods on
}

// OvmSccCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OvmSccCallerRaw struct {
	Contract *OvmSccCaller // Generic read-only contract binding to access the raw methods on
}

// OvmSccTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OvmSccTransactorRaw struct {
	Contract *OvmSccTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOvmScc creates a new instance of OvmScc, bound to a specific deployed contract.
func NewOvmScc(address common.Address, backend bind.ContractBackend) (*OvmScc, error) {
	contract, err := bindOvmScc(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OvmScc{OvmSccCaller: OvmSccCaller{contract: contract}, OvmSccTransactor: OvmSccTransactor{contract: contract}, OvmSccFilterer: OvmSccFilterer{contract: contract}}, nil
}

// NewOvmSccCaller creates a new read-only instance of OvmScc, bound to a specific deployed contract.
func NewOvmSccCaller(address common.Address, caller bind.ContractCaller) (*OvmSccCaller, error) {
	contract, err := bindOvmScc(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OvmSccCaller{contract: contract}, nil
}

// NewOvmSccTransactor creates a new write-only instance of OvmScc, bound to a specific deployed contract.
func NewOvmSccTransactor(address common.Address, transactor bind.ContractTransactor) (*OvmSccTransactor, error) {
	contract, err := bindOvmScc(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OvmSccTransactor{contract: contract}, nil
}

// NewOvmSccFilterer creates a new log filterer instance of OvmScc, bound to a specific deployed contract.
func NewOvmSccFilterer(address common.Address, filterer bind.ContractFilterer) (*OvmSccFilterer, error) {
	contract, err := bindOvmScc(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OvmSccFilterer{contract: contract}, nil
}

// bindOvmScc binds a generic wrapper to an already deployed contract.
func bindOvmScc(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OvmSccABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OvmScc *OvmSccRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OvmScc.Contract.OvmSccCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OvmScc *OvmSccRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmScc.Contract.OvmSccTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OvmScc *OvmSccRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OvmScc.Contract.OvmSccTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OvmScc *OvmSccCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _OvmScc.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OvmScc *OvmSccTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OvmScc.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OvmScc *OvmSccTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OvmScc.Contract.contract.Transact(opts, method, params...)
}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccCaller) FRAUDPROOFWINDOW(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "FRAUD_PROOF_WINDOW")
	return *ret0, err
}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccSession) FRAUDPROOFWINDOW() (*big.Int, error) {
	return _OvmScc.Contract.FRAUDPROOFWINDOW(&_OvmScc.CallOpts)
}

// FRAUDPROOFWINDOW is a free data retrieval call binding the contract method 0xc17b291b.
//
// Solidity: function FRAUD_PROOF_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccCallerSession) FRAUDPROOFWINDOW() (*big.Int, error) {
	return _OvmScc.Contract.FRAUDPROOFWINDOW(&_OvmScc.CallOpts)
}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccCaller) SEQUENCERPUBLISHWINDOW(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "SEQUENCER_PUBLISH_WINDOW")
	return *ret0, err
}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccSession) SEQUENCERPUBLISHWINDOW() (*big.Int, error) {
	return _OvmScc.Contract.SEQUENCERPUBLISHWINDOW(&_OvmScc.CallOpts)
}

// SEQUENCERPUBLISHWINDOW is a free data retrieval call binding the contract method 0x81eb62ef.
//
// Solidity: function SEQUENCER_PUBLISH_WINDOW() view returns(uint256)
func (_OvmScc *OvmSccCallerSession) SEQUENCERPUBLISHWINDOW() (*big.Int, error) {
	return _OvmScc.Contract.SEQUENCERPUBLISHWINDOW(&_OvmScc.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OvmScc *OvmSccCaller) Batches(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "batches")
	return *ret0, err
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OvmScc *OvmSccSession) Batches() (common.Address, error) {
	return _OvmScc.Contract.Batches(&_OvmScc.CallOpts)
}

// Batches is a free data retrieval call binding the contract method 0xcfdf677e.
//
// Solidity: function batches() view returns(address)
func (_OvmScc *OvmSccCallerSession) Batches() (common.Address, error) {
	return _OvmScc.Contract.Batches(&_OvmScc.CallOpts)
}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OvmScc *OvmSccCaller) GetLastSequencerTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "getLastSequencerTimestamp")
	return *ret0, err
}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OvmScc *OvmSccSession) GetLastSequencerTimestamp() (*big.Int, error) {
	return _OvmScc.Contract.GetLastSequencerTimestamp(&_OvmScc.CallOpts)
}

// GetLastSequencerTimestamp is a free data retrieval call binding the contract method 0x7ad168a0.
//
// Solidity: function getLastSequencerTimestamp() view returns(uint256 _lastSequencerTimestamp)
func (_OvmScc *OvmSccCallerSession) GetLastSequencerTimestamp() (*big.Int, error) {
	return _OvmScc.Contract.GetLastSequencerTimestamp(&_OvmScc.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OvmScc *OvmSccCaller) GetTotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "getTotalBatches")
	return *ret0, err
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OvmScc *OvmSccSession) GetTotalBatches() (*big.Int, error) {
	return _OvmScc.Contract.GetTotalBatches(&_OvmScc.CallOpts)
}

// GetTotalBatches is a free data retrieval call binding the contract method 0xe561dddc.
//
// Solidity: function getTotalBatches() view returns(uint256 _totalBatches)
func (_OvmScc *OvmSccCallerSession) GetTotalBatches() (*big.Int, error) {
	return _OvmScc.Contract.GetTotalBatches(&_OvmScc.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OvmScc *OvmSccCaller) GetTotalElements(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "getTotalElements")
	return *ret0, err
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OvmScc *OvmSccSession) GetTotalElements() (*big.Int, error) {
	return _OvmScc.Contract.GetTotalElements(&_OvmScc.CallOpts)
}

// GetTotalElements is a free data retrieval call binding the contract method 0x7aa63a86.
//
// Solidity: function getTotalElements() view returns(uint256 _totalElements)
func (_OvmScc *OvmSccCallerSession) GetTotalElements() (*big.Int, error) {
	return _OvmScc.Contract.GetTotalElements(&_OvmScc.CallOpts)
}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OvmScc *OvmSccCaller) InsideFraudProofWindow(opts *bind.CallOpts, _batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "insideFraudProofWindow", _batchHeader)
	return *ret0, err
}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OvmScc *OvmSccSession) InsideFraudProofWindow(_batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	return _OvmScc.Contract.InsideFraudProofWindow(&_OvmScc.CallOpts, _batchHeader)
}

// InsideFraudProofWindow is a free data retrieval call binding the contract method 0x9418bddd.
//
// Solidity: function insideFraudProofWindow((uint256,bytes32,uint256,uint256,bytes) _batchHeader) view returns(bool _inside)
func (_OvmScc *OvmSccCallerSession) InsideFraudProofWindow(_batchHeader Lib_OVMCodecChainBatchHeader) (bool, error) {
	return _OvmScc.Contract.InsideFraudProofWindow(&_OvmScc.CallOpts, _batchHeader)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmScc *OvmSccCaller) LibAddressManager(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "libAddressManager")
	return *ret0, err
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmScc *OvmSccSession) LibAddressManager() (common.Address, error) {
	return _OvmScc.Contract.LibAddressManager(&_OvmScc.CallOpts)
}

// LibAddressManager is a free data retrieval call binding the contract method 0x299ca478.
//
// Solidity: function libAddressManager() view returns(address)
func (_OvmScc *OvmSccCallerSession) LibAddressManager() (common.Address, error) {
	return _OvmScc.Contract.LibAddressManager(&_OvmScc.CallOpts)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmScc *OvmSccCaller) Resolve(opts *bind.CallOpts, _name string) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "resolve", _name)
	return *ret0, err
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmScc *OvmSccSession) Resolve(_name string) (common.Address, error) {
	return _OvmScc.Contract.Resolve(&_OvmScc.CallOpts, _name)
}

// Resolve is a free data retrieval call binding the contract method 0x461a4478.
//
// Solidity: function resolve(string _name) view returns(address)
func (_OvmScc *OvmSccCallerSession) Resolve(_name string) (common.Address, error) {
	return _OvmScc.Contract.Resolve(&_OvmScc.CallOpts, _name)
}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OvmScc *OvmSccCaller) VerifyStateCommitment(opts *bind.CallOpts, _element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _OvmScc.contract.Call(opts, out, "verifyStateCommitment", _element, _batchHeader, _proof)
	return *ret0, err
}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OvmScc *OvmSccSession) VerifyStateCommitment(_element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OvmScc.Contract.VerifyStateCommitment(&_OvmScc.CallOpts, _element, _batchHeader, _proof)
}

// VerifyStateCommitment is a free data retrieval call binding the contract method 0x4d69ee57.
//
// Solidity: function verifyStateCommitment(bytes32 _element, (uint256,bytes32,uint256,uint256,bytes) _batchHeader, (uint256,bytes32[]) _proof) view returns(bool)
func (_OvmScc *OvmSccCallerSession) VerifyStateCommitment(_element [32]byte, _batchHeader Lib_OVMCodecChainBatchHeader, _proof Lib_OVMCodecChainInclusionProof) (bool, error) {
	return _OvmScc.Contract.VerifyStateCommitment(&_OvmScc.CallOpts, _element, _batchHeader, _proof)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OvmScc *OvmSccTransactor) AppendStateBatch(opts *bind.TransactOpts, _batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OvmScc.contract.Transact(opts, "appendStateBatch", _batch, _shouldStartAtElement)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OvmScc *OvmSccSession) AppendStateBatch(_batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OvmScc.Contract.AppendStateBatch(&_OvmScc.TransactOpts, _batch, _shouldStartAtElement)
}

// AppendStateBatch is a paid mutator transaction binding the contract method 0x8ca5cbb9.
//
// Solidity: function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) returns()
func (_OvmScc *OvmSccTransactorSession) AppendStateBatch(_batch [][32]byte, _shouldStartAtElement *big.Int) (*types.Transaction, error) {
	return _OvmScc.Contract.AppendStateBatch(&_OvmScc.TransactOpts, _batch, _shouldStartAtElement)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OvmScc *OvmSccTransactor) DeleteStateBatch(opts *bind.TransactOpts, _batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OvmScc.contract.Transact(opts, "deleteStateBatch", _batchHeader)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OvmScc *OvmSccSession) DeleteStateBatch(_batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OvmScc.Contract.DeleteStateBatch(&_OvmScc.TransactOpts, _batchHeader)
}

// DeleteStateBatch is a paid mutator transaction binding the contract method 0xb8e189ac.
//
// Solidity: function deleteStateBatch((uint256,bytes32,uint256,uint256,bytes) _batchHeader) returns()
func (_OvmScc *OvmSccTransactorSession) DeleteStateBatch(_batchHeader Lib_OVMCodecChainBatchHeader) (*types.Transaction, error) {
	return _OvmScc.Contract.DeleteStateBatch(&_OvmScc.TransactOpts, _batchHeader)
}

// OvmSccStateBatchAppendedIterator is returned from FilterStateBatchAppended and is used to iterate over the raw logs and unpacked data for StateBatchAppended events raised by the OvmScc contract.
type OvmSccStateBatchAppendedIterator struct {
	Event *OvmSccStateBatchAppended // Event containing the contract specifics and raw log

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
func (it *OvmSccStateBatchAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmSccStateBatchAppended)
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
		it.Event = new(OvmSccStateBatchAppended)
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
func (it *OvmSccStateBatchAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmSccStateBatchAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmSccStateBatchAppended represents a StateBatchAppended event raised by the OvmScc contract.
type OvmSccStateBatchAppended struct {
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
func (_OvmScc *OvmSccFilterer) FilterStateBatchAppended(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OvmSccStateBatchAppendedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OvmScc.contract.FilterLogs(opts, "StateBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OvmSccStateBatchAppendedIterator{contract: _OvmScc.contract, event: "StateBatchAppended", logs: logs, sub: sub}, nil
}

// WatchStateBatchAppended is a free log subscription operation binding the contract event 0x16be4c5129a4e03cf3350262e181dc02ddfb4a6008d925368c0899fcd97ca9c5.
//
// Solidity: event StateBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
func (_OvmScc *OvmSccFilterer) WatchStateBatchAppended(opts *bind.WatchOpts, sink chan<- *OvmSccStateBatchAppended, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OvmScc.contract.WatchLogs(opts, "StateBatchAppended", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmSccStateBatchAppended)
				if err := _OvmScc.contract.UnpackLog(event, "StateBatchAppended", log); err != nil {
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
func (_OvmScc *OvmSccFilterer) ParseStateBatchAppended(log types.Log) (*OvmSccStateBatchAppended, error) {
	event := new(OvmSccStateBatchAppended)
	if err := _OvmScc.contract.UnpackLog(event, "StateBatchAppended", log); err != nil {
		return nil, err
	}
	return event, nil
}

// OvmSccStateBatchDeletedIterator is returned from FilterStateBatchDeleted and is used to iterate over the raw logs and unpacked data for StateBatchDeleted events raised by the OvmScc contract.
type OvmSccStateBatchDeletedIterator struct {
	Event *OvmSccStateBatchDeleted // Event containing the contract specifics and raw log

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
func (it *OvmSccStateBatchDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OvmSccStateBatchDeleted)
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
		it.Event = new(OvmSccStateBatchDeleted)
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
func (it *OvmSccStateBatchDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OvmSccStateBatchDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OvmSccStateBatchDeleted represents a StateBatchDeleted event raised by the OvmScc contract.
type OvmSccStateBatchDeleted struct {
	BatchIndex *big.Int
	BatchRoot  [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterStateBatchDeleted is a free log retrieval operation binding the contract event 0x8747b69ce8fdb31c3b9b0a67bd8049ad8c1a69ea417b69b12174068abd9cbd64.
//
// Solidity: event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
func (_OvmScc *OvmSccFilterer) FilterStateBatchDeleted(opts *bind.FilterOpts, _batchIndex []*big.Int) (*OvmSccStateBatchDeletedIterator, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OvmScc.contract.FilterLogs(opts, "StateBatchDeleted", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return &OvmSccStateBatchDeletedIterator{contract: _OvmScc.contract, event: "StateBatchDeleted", logs: logs, sub: sub}, nil
}

// WatchStateBatchDeleted is a free log subscription operation binding the contract event 0x8747b69ce8fdb31c3b9b0a67bd8049ad8c1a69ea417b69b12174068abd9cbd64.
//
// Solidity: event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
func (_OvmScc *OvmSccFilterer) WatchStateBatchDeleted(opts *bind.WatchOpts, sink chan<- *OvmSccStateBatchDeleted, _batchIndex []*big.Int) (event.Subscription, error) {

	var _batchIndexRule []interface{}
	for _, _batchIndexItem := range _batchIndex {
		_batchIndexRule = append(_batchIndexRule, _batchIndexItem)
	}

	logs, sub, err := _OvmScc.contract.WatchLogs(opts, "StateBatchDeleted", _batchIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OvmSccStateBatchDeleted)
				if err := _OvmScc.contract.UnpackLog(event, "StateBatchDeleted", log); err != nil {
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
func (_OvmScc *OvmSccFilterer) ParseStateBatchDeleted(log types.Log) (*OvmSccStateBatchDeleted, error) {
	event := new(OvmSccStateBatchDeleted)
	if err := _OvmScc.contract.UnpackLog(event, "StateBatchDeleted", log); err != nil {
		return nil, err
	}
	return event, nil
}
