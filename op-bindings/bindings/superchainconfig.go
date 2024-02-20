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

// SuperchainConfigMetaData contains all meta data concerning the SuperchainConfig contract.
var SuperchainConfigMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"GUARDIAN_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"PAUSED_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"fdgSafetyMode\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"guardian\",\"inputs\":[],\"outputs\":[{\"name\":\"guardian_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_guardian\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_paused\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"pause\",\"inputs\":[{\"name\":\"_identifier\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"paused_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setFDGSafetyMode\",\"inputs\":[{\"name\":\"_isEnabled\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unpause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ConfigUpdate\",\"inputs\":[{\"name\":\"updateType\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"enumSuperchainConfig.UpdateType\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Paused\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Unpaused\",\"inputs\":[],\"anonymous\":false}]",
	Bin: "0x60806040523480156200001157600080fd5b506200001f60008062000025565b62000361565b600054610100900460ff1615808015620000465750600054600160ff909116105b8062000076575062000063306200019460201b620007c11760201c565b15801562000076575060005460ff166001145b620000de5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805460ff19166001179055801562000102576000805461ff0019166101001790555b6200010d83620001a3565b81156200014857604080518082019091526012815271125b9a5d1a585b1a5e995c881c185d5cd95960721b6020820152620001489062000248565b80156200018f576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050565b6001600160a01b03163b151590565b620001e9620001d460017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69620002cf565b60001b82620002cb60201b620007dd1760201c565b6000604080516001600160a01b03841660208201527f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb910160408051601f19818403018152908290526200023d9162000345565b60405180910390a250565b6200028f6200027960017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7620002cf565b60001b6001620002cb60201b620007dd1760201c565b7fc32e6d5d6d1de257f64eac19ddb1f700ba13527983849c9486b1ab007ea2838181604051620002c0919062000345565b60405180910390a150565b9055565b600082821015620002f057634e487b7160e01b600052601160045260246000fd5b500390565b6000815180845260005b818110156200031d57602081850181015186830182015201620002ff565b8181111562000330576000602083870101525b50601f01601f19169290920160200192915050565b6020815260006200035a6020830184620002f5565b9392505050565b610b1880620003716000396000f3fe608060405234801561001057600080fd5b50600436106100be5760003560e01c80636da6635511610076578063bc1f825f1161005b578063bc1f825f1461019c578063c23a451a146101af578063c895d100146101b757600080fd5b80636da66355146101735780637fbf7b6a1461018657600080fd5b8063452a9320116100a7578063452a9320146100e057806354fd4d50146101125780635c975abb1461015b57600080fd5b80633f4ba83a146100c3578063400ada75146100cd575b600080fd5b6100cb6101ca565b005b6100cb6100db3660046108e1565b6102f0565b6100e86104c9565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b61014e6040518060400160405280600581526020017f312e322e3000000000000000000000000000000000000000000000000000000081525081565b604051610109919061099a565b610163610502565b6040519015158152602001610109565b6100cb6101813660046109e3565b610532565b61018e610600565b604051908152602001610109565b6100cb6101aa366004610ab2565b61062e565b61018e610796565b6000546101639062010000900460ff1681565b6101d26104c9565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610291576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e20756e706175736500000000000000000000000000000000000000000060648201526084015b60405180910390fd5b6102c56102bf60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7610acd565b60009055565b6040517fa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d1693390600090a1565b600054610100900460ff16158080156103105750600054600160ff909116105b8061032a5750303b15801561032a575060005460ff166001145b6103b6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610288565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055801561041457600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b61041d836107e1565b8115610461576104616040518060400160405280601281526020017f496e697469616c697a657220706175736564000000000000000000000000000081525061085e565b80156104c457600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050565b60006104fd6104f960017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69610acd565b5490565b905090565b60006104fd6104f960017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7610acd565b61053a6104c9565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146105f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e20706175736500000000000000000000000000000000000000000000006064820152608401610288565b6105fd8161085e565b50565b61062b60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7610acd565b81565b6106366104c9565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146106f0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f5375706572636861696e436f6e6669673a206f6e6c7920677561726469616e2060448201527f63616e20656e61626c6520736166657479206d6f6465000000000000000000006064820152608401610288565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffff16620100008315150217905560016040805183151560208201527f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb91015b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529082905261078b9161099a565b60405180910390a250565b61062b60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69610acd565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b9055565b61081461080f60017fd30e835d3f35624761057ff5b27d558f97bd5be034621e62240e5c0b784abe69610acd565b829055565b60006040805173ffffffffffffffffffffffffffffffffffffffff841660208201527f7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb9101610753565b61089261088c60017f54176ff9944c4784e5857ec4e5ef560a462c483bf534eda43f91bb01a470b1b7610acd565b60019055565b7fc32e6d5d6d1de257f64eac19ddb1f700ba13527983849c9486b1ab007ea28381816040516108c1919061099a565b60405180910390a150565b803580151581146108dc57600080fd5b919050565b600080604083850312156108f457600080fd5b823573ffffffffffffffffffffffffffffffffffffffff8116811461091857600080fd5b9150610926602084016108cc565b90509250929050565b6000815180845260005b8181101561095557602081850181015186830182015201610939565b81811115610967576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006109ad602083018461092f565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000602082840312156109f557600080fd5b813567ffffffffffffffff80821115610a0d57600080fd5b818401915084601f830112610a2157600080fd5b813581811115610a3357610a336109b4565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610a7957610a796109b4565b81604052828152876020848701011115610a9257600080fd5b826020860160208301376000928101602001929092525095945050505050565b600060208284031215610ac457600080fd5b6109ad826108cc565b600082821015610b06577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a",
}

// SuperchainConfigABI is the input ABI used to generate the binding from.
// Deprecated: Use SuperchainConfigMetaData.ABI instead.
var SuperchainConfigABI = SuperchainConfigMetaData.ABI

// SuperchainConfigBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SuperchainConfigMetaData.Bin instead.
var SuperchainConfigBin = SuperchainConfigMetaData.Bin

// DeploySuperchainConfig deploys a new Ethereum contract, binding an instance of SuperchainConfig to it.
func DeploySuperchainConfig(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SuperchainConfig, error) {
	parsed, err := SuperchainConfigMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SuperchainConfigBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SuperchainConfig{SuperchainConfigCaller: SuperchainConfigCaller{contract: contract}, SuperchainConfigTransactor: SuperchainConfigTransactor{contract: contract}, SuperchainConfigFilterer: SuperchainConfigFilterer{contract: contract}}, nil
}

// SuperchainConfig is an auto generated Go binding around an Ethereum contract.
type SuperchainConfig struct {
	SuperchainConfigCaller     // Read-only binding to the contract
	SuperchainConfigTransactor // Write-only binding to the contract
	SuperchainConfigFilterer   // Log filterer for contract events
}

// SuperchainConfigCaller is an auto generated read-only Go binding around an Ethereum contract.
type SuperchainConfigCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SuperchainConfigTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SuperchainConfigFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SuperchainConfigSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SuperchainConfigSession struct {
	Contract     *SuperchainConfig // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SuperchainConfigCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SuperchainConfigCallerSession struct {
	Contract *SuperchainConfigCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// SuperchainConfigTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SuperchainConfigTransactorSession struct {
	Contract     *SuperchainConfigTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// SuperchainConfigRaw is an auto generated low-level Go binding around an Ethereum contract.
type SuperchainConfigRaw struct {
	Contract *SuperchainConfig // Generic contract binding to access the raw methods on
}

// SuperchainConfigCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SuperchainConfigCallerRaw struct {
	Contract *SuperchainConfigCaller // Generic read-only contract binding to access the raw methods on
}

// SuperchainConfigTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SuperchainConfigTransactorRaw struct {
	Contract *SuperchainConfigTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSuperchainConfig creates a new instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfig(address common.Address, backend bind.ContractBackend) (*SuperchainConfig, error) {
	contract, err := bindSuperchainConfig(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfig{SuperchainConfigCaller: SuperchainConfigCaller{contract: contract}, SuperchainConfigTransactor: SuperchainConfigTransactor{contract: contract}, SuperchainConfigFilterer: SuperchainConfigFilterer{contract: contract}}, nil
}

// NewSuperchainConfigCaller creates a new read-only instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigCaller(address common.Address, caller bind.ContractCaller) (*SuperchainConfigCaller, error) {
	contract, err := bindSuperchainConfig(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigCaller{contract: contract}, nil
}

// NewSuperchainConfigTransactor creates a new write-only instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigTransactor(address common.Address, transactor bind.ContractTransactor) (*SuperchainConfigTransactor, error) {
	contract, err := bindSuperchainConfig(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigTransactor{contract: contract}, nil
}

// NewSuperchainConfigFilterer creates a new log filterer instance of SuperchainConfig, bound to a specific deployed contract.
func NewSuperchainConfigFilterer(address common.Address, filterer bind.ContractFilterer) (*SuperchainConfigFilterer, error) {
	contract, err := bindSuperchainConfig(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigFilterer{contract: contract}, nil
}

// bindSuperchainConfig binds a generic wrapper to an already deployed contract.
func bindSuperchainConfig(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SuperchainConfigABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SuperchainConfig *SuperchainConfigRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SuperchainConfig.Contract.SuperchainConfigCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SuperchainConfig *SuperchainConfigRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SuperchainConfigTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SuperchainConfig *SuperchainConfigRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SuperchainConfigTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SuperchainConfig *SuperchainConfigCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SuperchainConfig.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SuperchainConfig *SuperchainConfigTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SuperchainConfig *SuperchainConfigTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.contract.Transact(opts, method, params...)
}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) GUARDIANSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "GUARDIAN_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) GUARDIANSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.GUARDIANSLOT(&_SuperchainConfig.CallOpts)
}

// GUARDIANSLOT is a free data retrieval call binding the contract method 0xc23a451a.
//
// Solidity: function GUARDIAN_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) GUARDIANSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.GUARDIANSLOT(&_SuperchainConfig.CallOpts)
}

// PAUSEDSLOT is a free data retrieval call binding the contract method 0x7fbf7b6a.
//
// Solidity: function PAUSED_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCaller) PAUSEDSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "PAUSED_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PAUSEDSLOT is a free data retrieval call binding the contract method 0x7fbf7b6a.
//
// Solidity: function PAUSED_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigSession) PAUSEDSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.PAUSEDSLOT(&_SuperchainConfig.CallOpts)
}

// PAUSEDSLOT is a free data retrieval call binding the contract method 0x7fbf7b6a.
//
// Solidity: function PAUSED_SLOT() view returns(bytes32)
func (_SuperchainConfig *SuperchainConfigCallerSession) PAUSEDSLOT() ([32]byte, error) {
	return _SuperchainConfig.Contract.PAUSEDSLOT(&_SuperchainConfig.CallOpts)
}

// FdgSafetyMode is a free data retrieval call binding the contract method 0xc895d100.
//
// Solidity: function fdgSafetyMode() view returns(bool)
func (_SuperchainConfig *SuperchainConfigCaller) FdgSafetyMode(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "fdgSafetyMode")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FdgSafetyMode is a free data retrieval call binding the contract method 0xc895d100.
//
// Solidity: function fdgSafetyMode() view returns(bool)
func (_SuperchainConfig *SuperchainConfigSession) FdgSafetyMode() (bool, error) {
	return _SuperchainConfig.Contract.FdgSafetyMode(&_SuperchainConfig.CallOpts)
}

// FdgSafetyMode is a free data retrieval call binding the contract method 0xc895d100.
//
// Solidity: function fdgSafetyMode() view returns(bool)
func (_SuperchainConfig *SuperchainConfigCallerSession) FdgSafetyMode() (bool, error) {
	return _SuperchainConfig.Contract.FdgSafetyMode(&_SuperchainConfig.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigCaller) Guardian(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "guardian")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigSession) Guardian() (common.Address, error) {
	return _SuperchainConfig.Contract.Guardian(&_SuperchainConfig.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address guardian_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Guardian() (common.Address, error) {
	return _SuperchainConfig.Contract.Guardian(&_SuperchainConfig.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigSession) Paused() (bool, error) {
	return _SuperchainConfig.Contract.Paused(&_SuperchainConfig.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool paused_)
func (_SuperchainConfig *SuperchainConfigCallerSession) Paused() (bool, error) {
	return _SuperchainConfig.Contract.Paused(&_SuperchainConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SuperchainConfig.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigSession) Version() (string, error) {
	return _SuperchainConfig.Contract.Version(&_SuperchainConfig.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SuperchainConfig *SuperchainConfigCallerSession) Version() (string, error) {
	return _SuperchainConfig.Contract.Version(&_SuperchainConfig.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x400ada75.
//
// Solidity: function initialize(address _guardian, bool _paused) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Initialize(opts *bind.TransactOpts, _guardian common.Address, _paused bool) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "initialize", _guardian, _paused)
}

// Initialize is a paid mutator transaction binding the contract method 0x400ada75.
//
// Solidity: function initialize(address _guardian, bool _paused) returns()
func (_SuperchainConfig *SuperchainConfigSession) Initialize(_guardian common.Address, _paused bool) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _guardian, _paused)
}

// Initialize is a paid mutator transaction binding the contract method 0x400ada75.
//
// Solidity: function initialize(address _guardian, bool _paused) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Initialize(_guardian common.Address, _paused bool) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Initialize(&_SuperchainConfig.TransactOpts, _guardian, _paused)
}

// Pause is a paid mutator transaction binding the contract method 0x6da66355.
//
// Solidity: function pause(string _identifier) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Pause(opts *bind.TransactOpts, _identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "pause", _identifier)
}

// Pause is a paid mutator transaction binding the contract method 0x6da66355.
//
// Solidity: function pause(string _identifier) returns()
func (_SuperchainConfig *SuperchainConfigSession) Pause(_identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, _identifier)
}

// Pause is a paid mutator transaction binding the contract method 0x6da66355.
//
// Solidity: function pause(string _identifier) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Pause(_identifier string) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Pause(&_SuperchainConfig.TransactOpts, _identifier)
}

// SetFDGSafetyMode is a paid mutator transaction binding the contract method 0xbc1f825f.
//
// Solidity: function setFDGSafetyMode(bool _isEnabled) returns()
func (_SuperchainConfig *SuperchainConfigTransactor) SetFDGSafetyMode(opts *bind.TransactOpts, _isEnabled bool) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "setFDGSafetyMode", _isEnabled)
}

// SetFDGSafetyMode is a paid mutator transaction binding the contract method 0xbc1f825f.
//
// Solidity: function setFDGSafetyMode(bool _isEnabled) returns()
func (_SuperchainConfig *SuperchainConfigSession) SetFDGSafetyMode(_isEnabled bool) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SetFDGSafetyMode(&_SuperchainConfig.TransactOpts, _isEnabled)
}

// SetFDGSafetyMode is a paid mutator transaction binding the contract method 0xbc1f825f.
//
// Solidity: function setFDGSafetyMode(bool _isEnabled) returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) SetFDGSafetyMode(_isEnabled bool) (*types.Transaction, error) {
	return _SuperchainConfig.Contract.SetFDGSafetyMode(&_SuperchainConfig.TransactOpts, _isEnabled)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SuperchainConfig.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigSession) Unpause() (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Unpause(&_SuperchainConfig.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SuperchainConfig *SuperchainConfigTransactorSession) Unpause() (*types.Transaction, error) {
	return _SuperchainConfig.Contract.Unpause(&_SuperchainConfig.TransactOpts)
}

// SuperchainConfigConfigUpdateIterator is returned from FilterConfigUpdate and is used to iterate over the raw logs and unpacked data for ConfigUpdate events raised by the SuperchainConfig contract.
type SuperchainConfigConfigUpdateIterator struct {
	Event *SuperchainConfigConfigUpdate // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigConfigUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigConfigUpdate)
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
		it.Event = new(SuperchainConfigConfigUpdate)
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
func (it *SuperchainConfigConfigUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigConfigUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigConfigUpdate represents a ConfigUpdate event raised by the SuperchainConfig contract.
type SuperchainConfigConfigUpdate struct {
	UpdateType uint8
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterConfigUpdate is a free log retrieval operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterConfigUpdate(opts *bind.FilterOpts, updateType []uint8) (*SuperchainConfigConfigUpdateIterator, error) {

	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "ConfigUpdate", updateTypeRule)
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigConfigUpdateIterator{contract: _SuperchainConfig.contract, event: "ConfigUpdate", logs: logs, sub: sub}, nil
}

// WatchConfigUpdate is a free log subscription operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchConfigUpdate(opts *bind.WatchOpts, sink chan<- *SuperchainConfigConfigUpdate, updateType []uint8) (event.Subscription, error) {

	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "ConfigUpdate", updateTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigConfigUpdate)
				if err := _SuperchainConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
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

// ParseConfigUpdate is a log parse operation binding the contract event 0x7b743789cff01dafdeae47739925425aab5dfd02d0c8229e4a508bcd2b9f42bb.
//
// Solidity: event ConfigUpdate(uint8 indexed updateType, bytes data)
func (_SuperchainConfig *SuperchainConfigFilterer) ParseConfigUpdate(log types.Log) (*SuperchainConfigConfigUpdate, error) {
	event := new(SuperchainConfigConfigUpdate)
	if err := _SuperchainConfig.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the SuperchainConfig contract.
type SuperchainConfigInitializedIterator struct {
	Event *SuperchainConfigInitialized // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigInitialized)
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
		it.Event = new(SuperchainConfigInitialized)
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
func (it *SuperchainConfigInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigInitialized represents a Initialized event raised by the SuperchainConfig contract.
type SuperchainConfigInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterInitialized(opts *bind.FilterOpts) (*SuperchainConfigInitializedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigInitializedIterator{contract: _SuperchainConfig.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *SuperchainConfigInitialized) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigInitialized)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_SuperchainConfig *SuperchainConfigFilterer) ParseInitialized(log types.Log) (*SuperchainConfigInitialized, error) {
	event := new(SuperchainConfigInitialized)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the SuperchainConfig contract.
type SuperchainConfigPausedIterator struct {
	Event *SuperchainConfigPaused // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigPaused)
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
		it.Event = new(SuperchainConfigPaused)
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
func (it *SuperchainConfigPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigPaused represents a Paused event raised by the SuperchainConfig contract.
type SuperchainConfigPaused struct {
	Identifier string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0xc32e6d5d6d1de257f64eac19ddb1f700ba13527983849c9486b1ab007ea28381.
//
// Solidity: event Paused(string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) FilterPaused(opts *bind.FilterOpts) (*SuperchainConfigPausedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigPausedIterator{contract: _SuperchainConfig.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0xc32e6d5d6d1de257f64eac19ddb1f700ba13527983849c9486b1ab007ea28381.
//
// Solidity: event Paused(string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *SuperchainConfigPaused) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigPaused)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0xc32e6d5d6d1de257f64eac19ddb1f700ba13527983849c9486b1ab007ea28381.
//
// Solidity: event Paused(string identifier)
func (_SuperchainConfig *SuperchainConfigFilterer) ParsePaused(log types.Log) (*SuperchainConfigPaused, error) {
	event := new(SuperchainConfigPaused)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SuperchainConfigUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the SuperchainConfig contract.
type SuperchainConfigUnpausedIterator struct {
	Event *SuperchainConfigUnpaused // Event containing the contract specifics and raw log

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
func (it *SuperchainConfigUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SuperchainConfigUnpaused)
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
		it.Event = new(SuperchainConfigUnpaused)
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
func (it *SuperchainConfigUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SuperchainConfigUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SuperchainConfigUnpaused represents a Unpaused event raised by the SuperchainConfig contract.
type SuperchainConfigUnpaused struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) FilterUnpaused(opts *bind.FilterOpts) (*SuperchainConfigUnpausedIterator, error) {

	logs, sub, err := _SuperchainConfig.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &SuperchainConfigUnpausedIterator{contract: _SuperchainConfig.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *SuperchainConfigUnpaused) (event.Subscription, error) {

	logs, sub, err := _SuperchainConfig.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SuperchainConfigUnpaused)
				if err := _SuperchainConfig.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0xa45f47fdea8a1efdd9029a5691c7f759c32b7c698632b563573e155625d16933.
//
// Solidity: event Unpaused()
func (_SuperchainConfig *SuperchainConfigFilterer) ParseUnpaused(log types.Log) (*SuperchainConfigUnpaused, error) {
	event := new(SuperchainConfigUnpaused)
	if err := _SuperchainConfig.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
