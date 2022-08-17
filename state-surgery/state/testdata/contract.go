// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package testdata

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

// TestdataMetaData contains all meta data concerning the Testdata contract.
var TestdataMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"_address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_bool\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_uint256\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"addresses\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"}],\"name\":\"getStorage\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset0\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset1\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset2\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset3\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset4\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"offset5\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"key\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"setStorage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506102a4806100206000396000f3fe608060405234801561001057600080fd5b50600436106100b45760003560e01c8063502a6c5e11610071578063502a6c5e1461019a5780635e006859146101cc5780639b0b0fda146101d55780639e6ba9c2146101e9578063a753307d14610206578063edf26d9b1461023357600080fd5b8063099ea180146100b957806309f395f1146100dd57806318bad217146100ef57806332da25e11461011a578063350e81cc146101535780634ba664e71461017a575b600080fd5b6004546100c69060ff1681565b60405160ff90911681526020015b60405180910390f35b6004546100c690610100900460ff1681565b600054610102906001600160a01b031681565b6040516001600160a01b0390911681526020016100d4565b60045461013a9068010000000000000000900467ffffffffffffffff1681565b60405167ffffffffffffffff90911681526020016100d4565b6004546101679062010000900461ffff1681565b60405161ffff90911681526020016100d4565b61018c61018836600461025c565b5490565b6040519081526020016100d4565b6004546101b490600160801b90046001600160801b031681565b6040516001600160801b0390911681526020016100d4565b61018c60035481565b6101e76101e3366004610275565b9055565b005b6002546101f69060ff1681565b60405190151581526020016100d4565b60045461021e90640100000000900463ffffffff1681565b60405163ffffffff90911681526020016100d4565b61010261024136600461025c565b6001602052600090815260409020546001600160a01b031681565b60006020828403121561026e57600080fd5b5035919050565b6000806040838503121561028857600080fd5b5050803592602090910135915056fea164736f6c634300080f000a",
}

// TestdataABI is the input ABI used to generate the binding from.
// Deprecated: Use TestdataMetaData.ABI instead.
var TestdataABI = TestdataMetaData.ABI

// TestdataBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TestdataMetaData.Bin instead.
var TestdataBin = TestdataMetaData.Bin

// DeployTestdata deploys a new Ethereum contract, binding an instance of Testdata to it.
func DeployTestdata(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Testdata, error) {
	parsed, err := TestdataMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TestdataBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Testdata{TestdataCaller: TestdataCaller{contract: contract}, TestdataTransactor: TestdataTransactor{contract: contract}, TestdataFilterer: TestdataFilterer{contract: contract}}, nil
}

// Testdata is an auto generated Go binding around an Ethereum contract.
type Testdata struct {
	TestdataCaller     // Read-only binding to the contract
	TestdataTransactor // Write-only binding to the contract
	TestdataFilterer   // Log filterer for contract events
}

// TestdataCaller is an auto generated read-only Go binding around an Ethereum contract.
type TestdataCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestdataTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TestdataTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestdataFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TestdataFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TestdataSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TestdataSession struct {
	Contract     *Testdata         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TestdataCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TestdataCallerSession struct {
	Contract *TestdataCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// TestdataTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TestdataTransactorSession struct {
	Contract     *TestdataTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// TestdataRaw is an auto generated low-level Go binding around an Ethereum contract.
type TestdataRaw struct {
	Contract *Testdata // Generic contract binding to access the raw methods on
}

// TestdataCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TestdataCallerRaw struct {
	Contract *TestdataCaller // Generic read-only contract binding to access the raw methods on
}

// TestdataTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TestdataTransactorRaw struct {
	Contract *TestdataTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTestdata creates a new instance of Testdata, bound to a specific deployed contract.
func NewTestdata(address common.Address, backend bind.ContractBackend) (*Testdata, error) {
	contract, err := bindTestdata(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Testdata{TestdataCaller: TestdataCaller{contract: contract}, TestdataTransactor: TestdataTransactor{contract: contract}, TestdataFilterer: TestdataFilterer{contract: contract}}, nil
}

// NewTestdataCaller creates a new read-only instance of Testdata, bound to a specific deployed contract.
func NewTestdataCaller(address common.Address, caller bind.ContractCaller) (*TestdataCaller, error) {
	contract, err := bindTestdata(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TestdataCaller{contract: contract}, nil
}

// NewTestdataTransactor creates a new write-only instance of Testdata, bound to a specific deployed contract.
func NewTestdataTransactor(address common.Address, transactor bind.ContractTransactor) (*TestdataTransactor, error) {
	contract, err := bindTestdata(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TestdataTransactor{contract: contract}, nil
}

// NewTestdataFilterer creates a new log filterer instance of Testdata, bound to a specific deployed contract.
func NewTestdataFilterer(address common.Address, filterer bind.ContractFilterer) (*TestdataFilterer, error) {
	contract, err := bindTestdata(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TestdataFilterer{contract: contract}, nil
}

// bindTestdata binds a generic wrapper to an already deployed contract.
func bindTestdata(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TestdataABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Testdata *TestdataRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Testdata.Contract.TestdataCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Testdata *TestdataRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testdata.Contract.TestdataTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Testdata *TestdataRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Testdata.Contract.TestdataTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Testdata *TestdataCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Testdata.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Testdata *TestdataTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Testdata.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Testdata *TestdataTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Testdata.Contract.contract.Transact(opts, method, params...)
}

// Address is a free data retrieval call binding the contract method 0x18bad217.
//
// Solidity: function _address() view returns(address)
func (_Testdata *TestdataCaller) Address(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "_address")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Address is a free data retrieval call binding the contract method 0x18bad217.
//
// Solidity: function _address() view returns(address)
func (_Testdata *TestdataSession) Address() (common.Address, error) {
	return _Testdata.Contract.Address(&_Testdata.CallOpts)
}

// Address is a free data retrieval call binding the contract method 0x18bad217.
//
// Solidity: function _address() view returns(address)
func (_Testdata *TestdataCallerSession) Address() (common.Address, error) {
	return _Testdata.Contract.Address(&_Testdata.CallOpts)
}

// Bool is a free data retrieval call binding the contract method 0x9e6ba9c2.
//
// Solidity: function _bool() view returns(bool)
func (_Testdata *TestdataCaller) Bool(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "_bool")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Bool is a free data retrieval call binding the contract method 0x9e6ba9c2.
//
// Solidity: function _bool() view returns(bool)
func (_Testdata *TestdataSession) Bool() (bool, error) {
	return _Testdata.Contract.Bool(&_Testdata.CallOpts)
}

// Bool is a free data retrieval call binding the contract method 0x9e6ba9c2.
//
// Solidity: function _bool() view returns(bool)
func (_Testdata *TestdataCallerSession) Bool() (bool, error) {
	return _Testdata.Contract.Bool(&_Testdata.CallOpts)
}

// Uint256 is a free data retrieval call binding the contract method 0x5e006859.
//
// Solidity: function _uint256() view returns(uint256)
func (_Testdata *TestdataCaller) Uint256(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "_uint256")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Uint256 is a free data retrieval call binding the contract method 0x5e006859.
//
// Solidity: function _uint256() view returns(uint256)
func (_Testdata *TestdataSession) Uint256() (*big.Int, error) {
	return _Testdata.Contract.Uint256(&_Testdata.CallOpts)
}

// Uint256 is a free data retrieval call binding the contract method 0x5e006859.
//
// Solidity: function _uint256() view returns(uint256)
func (_Testdata *TestdataCallerSession) Uint256() (*big.Int, error) {
	return _Testdata.Contract.Uint256(&_Testdata.CallOpts)
}

// Addresses is a free data retrieval call binding the contract method 0xedf26d9b.
//
// Solidity: function addresses(uint256 ) view returns(address)
func (_Testdata *TestdataCaller) Addresses(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "addresses", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Addresses is a free data retrieval call binding the contract method 0xedf26d9b.
//
// Solidity: function addresses(uint256 ) view returns(address)
func (_Testdata *TestdataSession) Addresses(arg0 *big.Int) (common.Address, error) {
	return _Testdata.Contract.Addresses(&_Testdata.CallOpts, arg0)
}

// Addresses is a free data retrieval call binding the contract method 0xedf26d9b.
//
// Solidity: function addresses(uint256 ) view returns(address)
func (_Testdata *TestdataCallerSession) Addresses(arg0 *big.Int) (common.Address, error) {
	return _Testdata.Contract.Addresses(&_Testdata.CallOpts, arg0)
}

// GetStorage is a free data retrieval call binding the contract method 0x4ba664e7.
//
// Solidity: function getStorage(bytes32 key) view returns(bytes32)
func (_Testdata *TestdataCaller) GetStorage(opts *bind.CallOpts, key [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "getStorage", key)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetStorage is a free data retrieval call binding the contract method 0x4ba664e7.
//
// Solidity: function getStorage(bytes32 key) view returns(bytes32)
func (_Testdata *TestdataSession) GetStorage(key [32]byte) ([32]byte, error) {
	return _Testdata.Contract.GetStorage(&_Testdata.CallOpts, key)
}

// GetStorage is a free data retrieval call binding the contract method 0x4ba664e7.
//
// Solidity: function getStorage(bytes32 key) view returns(bytes32)
func (_Testdata *TestdataCallerSession) GetStorage(key [32]byte) ([32]byte, error) {
	return _Testdata.Contract.GetStorage(&_Testdata.CallOpts, key)
}

// Offset0 is a free data retrieval call binding the contract method 0x099ea180.
//
// Solidity: function offset0() view returns(uint8)
func (_Testdata *TestdataCaller) Offset0(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset0")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Offset0 is a free data retrieval call binding the contract method 0x099ea180.
//
// Solidity: function offset0() view returns(uint8)
func (_Testdata *TestdataSession) Offset0() (uint8, error) {
	return _Testdata.Contract.Offset0(&_Testdata.CallOpts)
}

// Offset0 is a free data retrieval call binding the contract method 0x099ea180.
//
// Solidity: function offset0() view returns(uint8)
func (_Testdata *TestdataCallerSession) Offset0() (uint8, error) {
	return _Testdata.Contract.Offset0(&_Testdata.CallOpts)
}

// Offset1 is a free data retrieval call binding the contract method 0x09f395f1.
//
// Solidity: function offset1() view returns(uint8)
func (_Testdata *TestdataCaller) Offset1(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset1")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Offset1 is a free data retrieval call binding the contract method 0x09f395f1.
//
// Solidity: function offset1() view returns(uint8)
func (_Testdata *TestdataSession) Offset1() (uint8, error) {
	return _Testdata.Contract.Offset1(&_Testdata.CallOpts)
}

// Offset1 is a free data retrieval call binding the contract method 0x09f395f1.
//
// Solidity: function offset1() view returns(uint8)
func (_Testdata *TestdataCallerSession) Offset1() (uint8, error) {
	return _Testdata.Contract.Offset1(&_Testdata.CallOpts)
}

// Offset2 is a free data retrieval call binding the contract method 0x350e81cc.
//
// Solidity: function offset2() view returns(uint16)
func (_Testdata *TestdataCaller) Offset2(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset2")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// Offset2 is a free data retrieval call binding the contract method 0x350e81cc.
//
// Solidity: function offset2() view returns(uint16)
func (_Testdata *TestdataSession) Offset2() (uint16, error) {
	return _Testdata.Contract.Offset2(&_Testdata.CallOpts)
}

// Offset2 is a free data retrieval call binding the contract method 0x350e81cc.
//
// Solidity: function offset2() view returns(uint16)
func (_Testdata *TestdataCallerSession) Offset2() (uint16, error) {
	return _Testdata.Contract.Offset2(&_Testdata.CallOpts)
}

// Offset3 is a free data retrieval call binding the contract method 0xa753307d.
//
// Solidity: function offset3() view returns(uint32)
func (_Testdata *TestdataCaller) Offset3(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset3")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// Offset3 is a free data retrieval call binding the contract method 0xa753307d.
//
// Solidity: function offset3() view returns(uint32)
func (_Testdata *TestdataSession) Offset3() (uint32, error) {
	return _Testdata.Contract.Offset3(&_Testdata.CallOpts)
}

// Offset3 is a free data retrieval call binding the contract method 0xa753307d.
//
// Solidity: function offset3() view returns(uint32)
func (_Testdata *TestdataCallerSession) Offset3() (uint32, error) {
	return _Testdata.Contract.Offset3(&_Testdata.CallOpts)
}

// Offset4 is a free data retrieval call binding the contract method 0x32da25e1.
//
// Solidity: function offset4() view returns(uint64)
func (_Testdata *TestdataCaller) Offset4(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset4")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// Offset4 is a free data retrieval call binding the contract method 0x32da25e1.
//
// Solidity: function offset4() view returns(uint64)
func (_Testdata *TestdataSession) Offset4() (uint64, error) {
	return _Testdata.Contract.Offset4(&_Testdata.CallOpts)
}

// Offset4 is a free data retrieval call binding the contract method 0x32da25e1.
//
// Solidity: function offset4() view returns(uint64)
func (_Testdata *TestdataCallerSession) Offset4() (uint64, error) {
	return _Testdata.Contract.Offset4(&_Testdata.CallOpts)
}

// Offset5 is a free data retrieval call binding the contract method 0x502a6c5e.
//
// Solidity: function offset5() view returns(uint128)
func (_Testdata *TestdataCaller) Offset5(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Testdata.contract.Call(opts, &out, "offset5")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Offset5 is a free data retrieval call binding the contract method 0x502a6c5e.
//
// Solidity: function offset5() view returns(uint128)
func (_Testdata *TestdataSession) Offset5() (*big.Int, error) {
	return _Testdata.Contract.Offset5(&_Testdata.CallOpts)
}

// Offset5 is a free data retrieval call binding the contract method 0x502a6c5e.
//
// Solidity: function offset5() view returns(uint128)
func (_Testdata *TestdataCallerSession) Offset5() (*big.Int, error) {
	return _Testdata.Contract.Offset5(&_Testdata.CallOpts)
}

// SetStorage is a paid mutator transaction binding the contract method 0x9b0b0fda.
//
// Solidity: function setStorage(bytes32 key, bytes32 value) returns()
func (_Testdata *TestdataTransactor) SetStorage(opts *bind.TransactOpts, key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Testdata.contract.Transact(opts, "setStorage", key, value)
}

// SetStorage is a paid mutator transaction binding the contract method 0x9b0b0fda.
//
// Solidity: function setStorage(bytes32 key, bytes32 value) returns()
func (_Testdata *TestdataSession) SetStorage(key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Testdata.Contract.SetStorage(&_Testdata.TransactOpts, key, value)
}

// SetStorage is a paid mutator transaction binding the contract method 0x9b0b0fda.
//
// Solidity: function setStorage(bytes32 key, bytes32 value) returns()
func (_Testdata *TestdataTransactorSession) SetStorage(key [32]byte, value [32]byte) (*types.Transaction, error) {
	return _Testdata.Contract.SetStorage(&_Testdata.TransactOpts, key, value)
}

