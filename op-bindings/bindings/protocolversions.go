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

// ProtocolVersionsMetaData contains all meta data concerning the ProtocolVersions contract.
var ProtocolVersionsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"enumProtocolVersions.UpdateType\",\"name\":\"updateType\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"ConfigUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"RECOMMENDED_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"REQUIRED_SLOT\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VERSION\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"ProtocolVersion\",\"name\":\"_required\",\"type\":\"uint256\"},{\"internalType\":\"ProtocolVersion\",\"name\":\"_recommended\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"recommended\",\"outputs\":[{\"internalType\":\"ProtocolVersion\",\"name\":\"out_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"required\",\"outputs\":[{\"internalType\":\"ProtocolVersion\",\"name\":\"out_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"ProtocolVersion\",\"name\":\"_recommended\",\"type\":\"uint256\"}],\"name\":\"setRecommended\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"ProtocolVersion\",\"name\":\"_required\",\"type\":\"uint256\"}],\"name\":\"setRequired\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b506200002261dead60008062000028565b620004c9565b600054600390610100900460ff161580156200004b575060005460ff8083169116105b620000b45760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620000d26200013a565b620000dd84620001a2565b620000e88362000221565b620000f382620002d1565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150505050565b600054610100900460ff16620001965760405162461bcd60e51b815260206004820152602b602482015260008051602062000e9683398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000ab565b620001a062000332565b565b620001ac62000399565b6001600160a01b038116620002135760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b6064820152608401620000ab565b6200021e81620003f5565b50565b620002676200025260017f4aaefe95bd84fd3f32700cf3b7566bc944b73138e41958b5785826df2aecace16200044b565b60001b826200044760201b620004f51760201c565b6000816040516020016200027d91815260200190565b60408051601f19818403018152919052905060005b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be83604051620002c5919062000471565b60405180910390a35050565b620003026200025260017fe314dfc40f0025322aacc0ba8ef420b62fb3b702cf01e0cdf3d829117ac2ff1b6200044b565b6000816040516020016200031891815260200190565b60408051601f198184030181529190529050600162000292565b600054610100900460ff166200038e5760405162461bcd60e51b815260206004820152602b602482015260008051602062000e9683398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000ab565b620001a033620003f5565b6033546001600160a01b03163314620001a05760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401620000ab565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b9055565b6000828210156200046c57634e487b7160e01b600052601160045260246000fd5b500390565b600060208083528351808285015260005b81811015620004a05785810183015185820160400152820162000482565b81811115620004b3576000604083870101525b50601f01601f1916929092016040019392505050565b6109bd80620004d96000396000f3fe608060405234801561001057600080fd5b50600436106100d45760003560e01c80638da5cb5b11610081578063f2fde38b1161005b578063f2fde38b146101b8578063f7d12760146101cb578063ffa1ad74146101d357600080fd5b80638da5cb5b14610180578063d798b1ac146101a8578063dc8452cd146101b057600080fd5b80635fd579af116100b25780635fd579af14610152578063715018a6146101655780637a1ac61e1461016d57600080fd5b80630457d6f2146100d9578063206a8300146100ee57806354fd4d5014610109575b600080fd5b6100ec6100e736600461085d565b6101db565b005b6100f66101ef565b6040519081526020015b60405180910390f35b6101456040518060400160405280600581526020017f312e302e3000000000000000000000000000000000000000000000000000000081525081565b60405161010091906108e1565b6100ec61016036600461085d565b61021d565b6100ec61022e565b6100ec61017b366004610924565b610242565b60335460405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610100565b6100f66103ad565b6100f66103e6565b6100ec6101c6366004610957565b610416565b6100f66104ca565b6100f6600081565b6101e36104f9565b6101ec8161057a565b50565b61021a60017f4aaefe95bd84fd3f32700cf3b7566bc944b73138e41958b5785826df2aecace1610972565b81565b6102256104f9565b6101ec81610632565b6102366104f9565b61024060006106ac565b565b600054600390610100900460ff16158015610264575060005460ff8083169116105b6102f5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff83161761010017905561032e610723565b61033784610416565b6103408361057a565b61034982610632565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150505050565b60006103e16103dd60017fe314dfc40f0025322aacc0ba8ef420b62fb3b702cf01e0cdf3d829117ac2ff1b610972565b5490565b905090565b60006103e16103dd60017f4aaefe95bd84fd3f32700cf3b7566bc944b73138e41958b5785826df2aecace1610972565b61041e6104f9565b73ffffffffffffffffffffffffffffffffffffffff81166104c1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016102ec565b6101ec816106ac565b61021a60017fe314dfc40f0025322aacc0ba8ef420b62fb3b702cf01e0cdf3d829117ac2ff1b610972565b9055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610240576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016102ec565b6105ad6105a860017f4aaefe95bd84fd3f32700cf3b7566bc944b73138e41958b5785826df2aecace1610972565b829055565b6000816040516020016105c291815260200190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152919052905060005b60007f1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be8360405161062691906108e1565b60405180910390a35050565b6106606105a860017fe314dfc40f0025322aacc0ba8ef420b62fb3b702cf01e0cdf3d829117ac2ff1b610972565b60008160405160200161067591815260200190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152919052905060016105f5565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff166107ba576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016102ec565b610240600054610100900460ff16610854576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016102ec565b610240336106ac565b60006020828403121561086f57600080fd5b5035919050565b6000815180845260005b8181101561089c57602081850181015186830182015201610880565b818111156108ae576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006108f46020830184610876565b9392505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461091f57600080fd5b919050565b60008060006060848603121561093957600080fd5b610942846108fb565b95602085013595506040909401359392505050565b60006020828403121561096957600080fd5b6108f4826108fb565b6000828210156109ab577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b50039056fea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// ProtocolVersionsABI is the input ABI used to generate the binding from.
// Deprecated: Use ProtocolVersionsMetaData.ABI instead.
var ProtocolVersionsABI = ProtocolVersionsMetaData.ABI

// ProtocolVersionsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProtocolVersionsMetaData.Bin instead.
var ProtocolVersionsBin = ProtocolVersionsMetaData.Bin

// DeployProtocolVersions deploys a new Ethereum contract, binding an instance of ProtocolVersions to it.
func DeployProtocolVersions(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ProtocolVersions, error) {
	parsed, err := ProtocolVersionsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProtocolVersionsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProtocolVersions{ProtocolVersionsCaller: ProtocolVersionsCaller{contract: contract}, ProtocolVersionsTransactor: ProtocolVersionsTransactor{contract: contract}, ProtocolVersionsFilterer: ProtocolVersionsFilterer{contract: contract}}, nil
}

// ProtocolVersions is an auto generated Go binding around an Ethereum contract.
type ProtocolVersions struct {
	ProtocolVersionsCaller     // Read-only binding to the contract
	ProtocolVersionsTransactor // Write-only binding to the contract
	ProtocolVersionsFilterer   // Log filterer for contract events
}

// ProtocolVersionsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProtocolVersionsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolVersionsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProtocolVersionsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolVersionsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProtocolVersionsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolVersionsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProtocolVersionsSession struct {
	Contract     *ProtocolVersions // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ProtocolVersionsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProtocolVersionsCallerSession struct {
	Contract *ProtocolVersionsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// ProtocolVersionsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProtocolVersionsTransactorSession struct {
	Contract     *ProtocolVersionsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// ProtocolVersionsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProtocolVersionsRaw struct {
	Contract *ProtocolVersions // Generic contract binding to access the raw methods on
}

// ProtocolVersionsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProtocolVersionsCallerRaw struct {
	Contract *ProtocolVersionsCaller // Generic read-only contract binding to access the raw methods on
}

// ProtocolVersionsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProtocolVersionsTransactorRaw struct {
	Contract *ProtocolVersionsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProtocolVersions creates a new instance of ProtocolVersions, bound to a specific deployed contract.
func NewProtocolVersions(address common.Address, backend bind.ContractBackend) (*ProtocolVersions, error) {
	contract, err := bindProtocolVersions(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersions{ProtocolVersionsCaller: ProtocolVersionsCaller{contract: contract}, ProtocolVersionsTransactor: ProtocolVersionsTransactor{contract: contract}, ProtocolVersionsFilterer: ProtocolVersionsFilterer{contract: contract}}, nil
}

// NewProtocolVersionsCaller creates a new read-only instance of ProtocolVersions, bound to a specific deployed contract.
func NewProtocolVersionsCaller(address common.Address, caller bind.ContractCaller) (*ProtocolVersionsCaller, error) {
	contract, err := bindProtocolVersions(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsCaller{contract: contract}, nil
}

// NewProtocolVersionsTransactor creates a new write-only instance of ProtocolVersions, bound to a specific deployed contract.
func NewProtocolVersionsTransactor(address common.Address, transactor bind.ContractTransactor) (*ProtocolVersionsTransactor, error) {
	contract, err := bindProtocolVersions(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsTransactor{contract: contract}, nil
}

// NewProtocolVersionsFilterer creates a new log filterer instance of ProtocolVersions, bound to a specific deployed contract.
func NewProtocolVersionsFilterer(address common.Address, filterer bind.ContractFilterer) (*ProtocolVersionsFilterer, error) {
	contract, err := bindProtocolVersions(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsFilterer{contract: contract}, nil
}

// bindProtocolVersions binds a generic wrapper to an already deployed contract.
func bindProtocolVersions(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ProtocolVersionsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProtocolVersions *ProtocolVersionsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProtocolVersions.Contract.ProtocolVersionsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProtocolVersions *ProtocolVersionsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.ProtocolVersionsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProtocolVersions *ProtocolVersionsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.ProtocolVersionsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProtocolVersions *ProtocolVersionsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProtocolVersions.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProtocolVersions *ProtocolVersionsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProtocolVersions *ProtocolVersionsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.contract.Transact(opts, method, params...)
}

// RECOMMENDEDSLOT is a free data retrieval call binding the contract method 0xf7d12760.
//
// Solidity: function RECOMMENDED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsCaller) RECOMMENDEDSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "RECOMMENDED_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RECOMMENDEDSLOT is a free data retrieval call binding the contract method 0xf7d12760.
//
// Solidity: function RECOMMENDED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsSession) RECOMMENDEDSLOT() ([32]byte, error) {
	return _ProtocolVersions.Contract.RECOMMENDEDSLOT(&_ProtocolVersions.CallOpts)
}

// RECOMMENDEDSLOT is a free data retrieval call binding the contract method 0xf7d12760.
//
// Solidity: function RECOMMENDED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsCallerSession) RECOMMENDEDSLOT() ([32]byte, error) {
	return _ProtocolVersions.Contract.RECOMMENDEDSLOT(&_ProtocolVersions.CallOpts)
}

// REQUIREDSLOT is a free data retrieval call binding the contract method 0x206a8300.
//
// Solidity: function REQUIRED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsCaller) REQUIREDSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "REQUIRED_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// REQUIREDSLOT is a free data retrieval call binding the contract method 0x206a8300.
//
// Solidity: function REQUIRED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsSession) REQUIREDSLOT() ([32]byte, error) {
	return _ProtocolVersions.Contract.REQUIREDSLOT(&_ProtocolVersions.CallOpts)
}

// REQUIREDSLOT is a free data retrieval call binding the contract method 0x206a8300.
//
// Solidity: function REQUIRED_SLOT() view returns(bytes32)
func (_ProtocolVersions *ProtocolVersionsCallerSession) REQUIREDSLOT() ([32]byte, error) {
	return _ProtocolVersions.Contract.REQUIREDSLOT(&_ProtocolVersions.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_ProtocolVersions *ProtocolVersionsCaller) VERSION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "VERSION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_ProtocolVersions *ProtocolVersionsSession) VERSION() (*big.Int, error) {
	return _ProtocolVersions.Contract.VERSION(&_ProtocolVersions.CallOpts)
}

// VERSION is a free data retrieval call binding the contract method 0xffa1ad74.
//
// Solidity: function VERSION() view returns(uint256)
func (_ProtocolVersions *ProtocolVersionsCallerSession) VERSION() (*big.Int, error) {
	return _ProtocolVersions.Contract.VERSION(&_ProtocolVersions.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProtocolVersions *ProtocolVersionsCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProtocolVersions *ProtocolVersionsSession) Owner() (common.Address, error) {
	return _ProtocolVersions.Contract.Owner(&_ProtocolVersions.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ProtocolVersions *ProtocolVersionsCallerSession) Owner() (common.Address, error) {
	return _ProtocolVersions.Contract.Owner(&_ProtocolVersions.CallOpts)
}

// Recommended is a free data retrieval call binding the contract method 0xd798b1ac.
//
// Solidity: function recommended() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsCaller) Recommended(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "recommended")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Recommended is a free data retrieval call binding the contract method 0xd798b1ac.
//
// Solidity: function recommended() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsSession) Recommended() (*big.Int, error) {
	return _ProtocolVersions.Contract.Recommended(&_ProtocolVersions.CallOpts)
}

// Recommended is a free data retrieval call binding the contract method 0xd798b1ac.
//
// Solidity: function recommended() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsCallerSession) Recommended() (*big.Int, error) {
	return _ProtocolVersions.Contract.Recommended(&_ProtocolVersions.CallOpts)
}

// Required is a free data retrieval call binding the contract method 0xdc8452cd.
//
// Solidity: function required() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsCaller) Required(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "required")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Required is a free data retrieval call binding the contract method 0xdc8452cd.
//
// Solidity: function required() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsSession) Required() (*big.Int, error) {
	return _ProtocolVersions.Contract.Required(&_ProtocolVersions.CallOpts)
}

// Required is a free data retrieval call binding the contract method 0xdc8452cd.
//
// Solidity: function required() view returns(uint256 out_)
func (_ProtocolVersions *ProtocolVersionsCallerSession) Required() (*big.Int, error) {
	return _ProtocolVersions.Contract.Required(&_ProtocolVersions.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ProtocolVersions *ProtocolVersionsCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ProtocolVersions.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ProtocolVersions *ProtocolVersionsSession) Version() (string, error) {
	return _ProtocolVersions.Contract.Version(&_ProtocolVersions.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_ProtocolVersions *ProtocolVersionsCallerSession) Version() (string, error) {
	return _ProtocolVersions.Contract.Version(&_ProtocolVersions.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x7a1ac61e.
//
// Solidity: function initialize(address _owner, uint256 _required, uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _required *big.Int, _recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.contract.Transact(opts, "initialize", _owner, _required, _recommended)
}

// Initialize is a paid mutator transaction binding the contract method 0x7a1ac61e.
//
// Solidity: function initialize(address _owner, uint256 _required, uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsSession) Initialize(_owner common.Address, _required *big.Int, _recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.Initialize(&_ProtocolVersions.TransactOpts, _owner, _required, _recommended)
}

// Initialize is a paid mutator transaction binding the contract method 0x7a1ac61e.
//
// Solidity: function initialize(address _owner, uint256 _required, uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsTransactorSession) Initialize(_owner common.Address, _required *big.Int, _recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.Initialize(&_ProtocolVersions.TransactOpts, _owner, _required, _recommended)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProtocolVersions *ProtocolVersionsTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProtocolVersions.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProtocolVersions *ProtocolVersionsSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProtocolVersions.Contract.RenounceOwnership(&_ProtocolVersions.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ProtocolVersions *ProtocolVersionsTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ProtocolVersions.Contract.RenounceOwnership(&_ProtocolVersions.TransactOpts)
}

// SetRecommended is a paid mutator transaction binding the contract method 0x5fd579af.
//
// Solidity: function setRecommended(uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsTransactor) SetRecommended(opts *bind.TransactOpts, _recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.contract.Transact(opts, "setRecommended", _recommended)
}

// SetRecommended is a paid mutator transaction binding the contract method 0x5fd579af.
//
// Solidity: function setRecommended(uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsSession) SetRecommended(_recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.SetRecommended(&_ProtocolVersions.TransactOpts, _recommended)
}

// SetRecommended is a paid mutator transaction binding the contract method 0x5fd579af.
//
// Solidity: function setRecommended(uint256 _recommended) returns()
func (_ProtocolVersions *ProtocolVersionsTransactorSession) SetRecommended(_recommended *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.SetRecommended(&_ProtocolVersions.TransactOpts, _recommended)
}

// SetRequired is a paid mutator transaction binding the contract method 0x0457d6f2.
//
// Solidity: function setRequired(uint256 _required) returns()
func (_ProtocolVersions *ProtocolVersionsTransactor) SetRequired(opts *bind.TransactOpts, _required *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.contract.Transact(opts, "setRequired", _required)
}

// SetRequired is a paid mutator transaction binding the contract method 0x0457d6f2.
//
// Solidity: function setRequired(uint256 _required) returns()
func (_ProtocolVersions *ProtocolVersionsSession) SetRequired(_required *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.SetRequired(&_ProtocolVersions.TransactOpts, _required)
}

// SetRequired is a paid mutator transaction binding the contract method 0x0457d6f2.
//
// Solidity: function setRequired(uint256 _required) returns()
func (_ProtocolVersions *ProtocolVersionsTransactorSession) SetRequired(_required *big.Int) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.SetRequired(&_ProtocolVersions.TransactOpts, _required)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProtocolVersions *ProtocolVersionsTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ProtocolVersions.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProtocolVersions *ProtocolVersionsSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.TransferOwnership(&_ProtocolVersions.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ProtocolVersions *ProtocolVersionsTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ProtocolVersions.Contract.TransferOwnership(&_ProtocolVersions.TransactOpts, newOwner)
}

// ProtocolVersionsConfigUpdateIterator is returned from FilterConfigUpdate and is used to iterate over the raw logs and unpacked data for ConfigUpdate events raised by the ProtocolVersions contract.
type ProtocolVersionsConfigUpdateIterator struct {
	Event *ProtocolVersionsConfigUpdate // Event containing the contract specifics and raw log

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
func (it *ProtocolVersionsConfigUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProtocolVersionsConfigUpdate)
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
		it.Event = new(ProtocolVersionsConfigUpdate)
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
func (it *ProtocolVersionsConfigUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProtocolVersionsConfigUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProtocolVersionsConfigUpdate represents a ConfigUpdate event raised by the ProtocolVersions contract.
type ProtocolVersionsConfigUpdate struct {
	Version    *big.Int
	UpdateType uint8
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterConfigUpdate is a free log retrieval operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_ProtocolVersions *ProtocolVersionsFilterer) FilterConfigUpdate(opts *bind.FilterOpts, version []*big.Int, updateType []uint8) (*ProtocolVersionsConfigUpdateIterator, error) {

	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}
	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _ProtocolVersions.contract.FilterLogs(opts, "ConfigUpdate", versionRule, updateTypeRule)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsConfigUpdateIterator{contract: _ProtocolVersions.contract, event: "ConfigUpdate", logs: logs, sub: sub}, nil
}

// WatchConfigUpdate is a free log subscription operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_ProtocolVersions *ProtocolVersionsFilterer) WatchConfigUpdate(opts *bind.WatchOpts, sink chan<- *ProtocolVersionsConfigUpdate, version []*big.Int, updateType []uint8) (event.Subscription, error) {

	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}
	var updateTypeRule []interface{}
	for _, updateTypeItem := range updateType {
		updateTypeRule = append(updateTypeRule, updateTypeItem)
	}

	logs, sub, err := _ProtocolVersions.contract.WatchLogs(opts, "ConfigUpdate", versionRule, updateTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProtocolVersionsConfigUpdate)
				if err := _ProtocolVersions.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
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

// ParseConfigUpdate is a log parse operation binding the contract event 0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be.
//
// Solidity: event ConfigUpdate(uint256 indexed version, uint8 indexed updateType, bytes data)
func (_ProtocolVersions *ProtocolVersionsFilterer) ParseConfigUpdate(log types.Log) (*ProtocolVersionsConfigUpdate, error) {
	event := new(ProtocolVersionsConfigUpdate)
	if err := _ProtocolVersions.contract.UnpackLog(event, "ConfigUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ProtocolVersionsInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the ProtocolVersions contract.
type ProtocolVersionsInitializedIterator struct {
	Event *ProtocolVersionsInitialized // Event containing the contract specifics and raw log

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
func (it *ProtocolVersionsInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProtocolVersionsInitialized)
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
		it.Event = new(ProtocolVersionsInitialized)
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
func (it *ProtocolVersionsInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProtocolVersionsInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProtocolVersionsInitialized represents a Initialized event raised by the ProtocolVersions contract.
type ProtocolVersionsInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_ProtocolVersions *ProtocolVersionsFilterer) FilterInitialized(opts *bind.FilterOpts) (*ProtocolVersionsInitializedIterator, error) {

	logs, sub, err := _ProtocolVersions.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsInitializedIterator{contract: _ProtocolVersions.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_ProtocolVersions *ProtocolVersionsFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ProtocolVersionsInitialized) (event.Subscription, error) {

	logs, sub, err := _ProtocolVersions.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProtocolVersionsInitialized)
				if err := _ProtocolVersions.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_ProtocolVersions *ProtocolVersionsFilterer) ParseInitialized(log types.Log) (*ProtocolVersionsInitialized, error) {
	event := new(ProtocolVersionsInitialized)
	if err := _ProtocolVersions.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ProtocolVersionsOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ProtocolVersions contract.
type ProtocolVersionsOwnershipTransferredIterator struct {
	Event *ProtocolVersionsOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ProtocolVersionsOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProtocolVersionsOwnershipTransferred)
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
		it.Event = new(ProtocolVersionsOwnershipTransferred)
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
func (it *ProtocolVersionsOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProtocolVersionsOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProtocolVersionsOwnershipTransferred represents a OwnershipTransferred event raised by the ProtocolVersions contract.
type ProtocolVersionsOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProtocolVersions *ProtocolVersionsFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ProtocolVersionsOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProtocolVersions.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ProtocolVersionsOwnershipTransferredIterator{contract: _ProtocolVersions.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ProtocolVersions *ProtocolVersionsFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ProtocolVersionsOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ProtocolVersions.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProtocolVersionsOwnershipTransferred)
				if err := _ProtocolVersions.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_ProtocolVersions *ProtocolVersionsFilterer) ParseOwnershipTransferred(log types.Log) (*ProtocolVersionsOwnershipTransferred, error) {
	event := new(ProtocolVersionsOwnershipTransferred)
	if err := _ProtocolVersions.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
