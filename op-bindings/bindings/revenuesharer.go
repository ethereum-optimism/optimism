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

// RevenueSharerMetaData contains all meta data concerning the RevenueSharer contract.
var RevenueSharerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_optimismWallet\",\"type\":\"address\",\"internalType\":\"addresspayable\"},{\"name\":\"_l1Wallet\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_feeDisbursementInterval\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"BASIS_POINT_SCALE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"FEE_DISBURSEMENT_INTERVAL\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L1_WALLET\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"OPTIMISM_WALLET\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"addresspayable\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WITHDRAWAL_MIN_GAS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"disburseFees\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"lastDisbursementTime\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"netFeeRevenue\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FeesDisbursed\",\"inputs\":[{\"name\":\"_disbursementTime\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"_paidToOptimism\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"_totalFeesDisbursed\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FeesReceived\",\"inputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"NoFeesCollected\",\"inputs\":[],\"anonymous\":false}]",
	Bin: "0x60e060405234801561001057600080fd5b50604051610e80380380610e8083398101604081905261002f916101c5565b6001600160a01b0383166100a45760405162461bcd60e51b815260206004820152603160248201527f4665654469736275727365723a204f7074696d69736d57616c6c65742063616e6044820152706e6f74206265206164647265737328302960781b60648201526084015b60405180910390fd5b6001600160a01b03821661010e5760405162461bcd60e51b815260206004820152602b60248201527f4665654469736275727365723a204c3157616c6c65742063616e6e6f7420626560448201526a206164647265737328302960a81b606482015260840161009b565b620151808110156101925760405162461bcd60e51b815260206004820152604260248201527f4665654469736275727365723a2046656544697362757273656d656e74496e7460448201527f657276616c2063616e6e6f74206265206c657373207468616e20323420686f75606482015261727360f01b608482015260a40161009b565b6001600160a01b03928316608052911660a05260c052610208565b6001600160a01b03811681146101c257600080fd5b50565b6000806000606084860312156101da57600080fd5b83516101e5816101ad565b60208501519093506101f6816101ad565b80925050604084015190509250925092565b60805160a05160c051610c3461024c600039600081816102e8015261037a0152600081816102880152610626015260008181610207015261051d0152610c346000f3fe6080604052600436106100b55760003560e01c806354664de51161006957806393819a3f1161004e57806393819a3f14610335578063ad41d09c1461034b578063b87ea8d41461036157600080fd5b806354664de5146102d65780635b201d831461030a57600080fd5b806336f1a6e51161009a57806336f1a6e514610276578063394d2731146102aa578063447eb5ac146102c057600080fd5b80630c8cd070146101f5578063235d506d1461025357600080fd5b366101f0573373420000000000000000000000000000000000001114806100ef575033734200000000000000000000000000000000000019145b156101115734600160008282546101069190610a5a565b909155506101b99050565b3373420000000000000000000000000000000000001a146101b9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f4665654469736275727365723a204f6e6c79204665655661756c74732063616e60448201527f2073656e642045544820746f204665654469736275727365720000000000000060648201526084015b60405180910390fd5b60405134815233907f2ccfc58c2cef4ee590b5f16be0548cc54afc12e1c66a67b362b7d640fd16bb2d9060200160405180910390a2005b600080fd5b34801561020157600080fd5b506102297f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561025f57600080fd5b5061026860fa81565b60405190815260200161024a565b34801561028257600080fd5b506102297f000000000000000000000000000000000000000000000000000000000000000081565b3480156102b657600080fd5b5061026860005481565b3480156102cc57600080fd5b5061026860015481565b3480156102e257600080fd5b506102687f000000000000000000000000000000000000000000000000000000000000000081565b34801561031657600080fd5b5061032061271081565b60405163ffffffff909116815260200161024a565b34801561034157600080fd5b506102686105dc81565b34801561035757600080fd5b506103206188b881565b34801561036d57600080fd5b50610376610378565b005b7f00000000000000000000000000000000000000000000000000000000000000006000546103a69190610a5a565b421015610435576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f4665654469736275727365723a2044697362757273656d656e7420696e74657260448201527f76616c206e6f742072656163686564000000000000000000000000000000000060648201526084016101b0565b6104527342000000000000000000000000000000000000116106d4565b61046f7342000000000000000000000000000000000000196106d4565b61048c73420000000000000000000000000000000000001a6106d4565b4760008190036104c2576040517f8c887b1215d5e6b119c1c1008fe1d0919b4c438301d5a0357362a13fb56f6a4090600090a150565b426000908155600154612710906104dc906105dc90610a72565b6104e69190610aaf565b600060018190559091506127106104fe60fa85610a72565b6105089190610aaf565b9050600061051683836109fc565b90506105437f00000000000000000000000000000000000000000000000000000000000000005a83610a15565b6105cf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f4665654469736275727365723a204661696c656420746f2073656e642066756e60448201527f647320746f204f7074696d69736d00000000000000000000000000000000000060648201526084016101b0565b604080516020810182526000815290517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109163e11013dd914791610652917f0000000000000000000000000000000000000000000000000000000000000000916188b891600401610aea565b6000604051808303818588803b15801561066b57600080fd5b505af115801561067f573d6000803e3d6000fd5b5050600054604080519182526020820186905281018890527fe155e054cfe69655d6d2f8bbfb856aa8cdf49ecbea6557901533364539caad94935060600191506106c69050565b60405180910390a150505050565b60018173ffffffffffffffffffffffffffffffffffffffff1663d0e12f906040518163ffffffff1660e01b8152600401602060405180830381865afa158015610721573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107459190610bb7565b600181111561075657610756610b88565b146107e3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f4665654469736275727365723a204665655661756c74206d757374207769746860448201527f6472617720746f204c320000000000000000000000000000000000000000000060648201526084016101b0565b3073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16630d9019e16040518163ffffffff1660e01b8152600401602060405180830381865afa158015610845573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906108699190610bd8565b73ffffffffffffffffffffffffffffffffffffffff161461090c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603d60248201527f4665654469736275727365723a204665655661756c74206d757374207769746860448201527f6472617720746f2046656544697362757273657220636f6e747261637400000060648201526084016101b0565b8073ffffffffffffffffffffffffffffffffffffffff1663d3e5792b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610957573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061097b9190610c0e565b8173ffffffffffffffffffffffffffffffffffffffff1631106109f9578073ffffffffffffffffffffffffffffffffffffffff16633ccfd60b6040518163ffffffff1660e01b8152600401600060405180830381600087803b1580156109e057600080fd5b505af11580156109f4573d6000803e3d6000fd5b505050505b50565b600081831015610a0c5781610a0e565b825b9392505050565b600080600080600080868989f195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115610a6d57610a6d610a2b565b500190565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615610aaa57610aaa610a2b565b500290565b600082610ae5577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500490565b73ffffffffffffffffffffffffffffffffffffffff841681526000602063ffffffff85168184015260606040840152835180606085015260005b81811015610b4057858101830151858201608001528201610b24565b81811115610b52576000608083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160800195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b600060208284031215610bc957600080fd5b815160028110610a0e57600080fd5b600060208284031215610bea57600080fd5b815173ffffffffffffffffffffffffffffffffffffffff81168114610a0e57600080fd5b600060208284031215610c2057600080fd5b505191905056fea164736f6c634300080f000a",
}

// RevenueSharerABI is the input ABI used to generate the binding from.
// Deprecated: Use RevenueSharerMetaData.ABI instead.
var RevenueSharerABI = RevenueSharerMetaData.ABI

// RevenueSharerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RevenueSharerMetaData.Bin instead.
var RevenueSharerBin = RevenueSharerMetaData.Bin

// DeployRevenueSharer deploys a new Ethereum contract, binding an instance of RevenueSharer to it.
func DeployRevenueSharer(auth *bind.TransactOpts, backend bind.ContractBackend, _optimismWallet common.Address, _l1Wallet common.Address, _feeDisbursementInterval *big.Int) (common.Address, *types.Transaction, *RevenueSharer, error) {
	parsed, err := RevenueSharerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RevenueSharerBin), backend, _optimismWallet, _l1Wallet, _feeDisbursementInterval)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RevenueSharer{RevenueSharerCaller: RevenueSharerCaller{contract: contract}, RevenueSharerTransactor: RevenueSharerTransactor{contract: contract}, RevenueSharerFilterer: RevenueSharerFilterer{contract: contract}}, nil
}

// RevenueSharer is an auto generated Go binding around an Ethereum contract.
type RevenueSharer struct {
	RevenueSharerCaller     // Read-only binding to the contract
	RevenueSharerTransactor // Write-only binding to the contract
	RevenueSharerFilterer   // Log filterer for contract events
}

// RevenueSharerCaller is an auto generated read-only Go binding around an Ethereum contract.
type RevenueSharerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevenueSharerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RevenueSharerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevenueSharerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RevenueSharerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RevenueSharerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RevenueSharerSession struct {
	Contract     *RevenueSharer    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RevenueSharerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RevenueSharerCallerSession struct {
	Contract *RevenueSharerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// RevenueSharerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RevenueSharerTransactorSession struct {
	Contract     *RevenueSharerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// RevenueSharerRaw is an auto generated low-level Go binding around an Ethereum contract.
type RevenueSharerRaw struct {
	Contract *RevenueSharer // Generic contract binding to access the raw methods on
}

// RevenueSharerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RevenueSharerCallerRaw struct {
	Contract *RevenueSharerCaller // Generic read-only contract binding to access the raw methods on
}

// RevenueSharerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RevenueSharerTransactorRaw struct {
	Contract *RevenueSharerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRevenueSharer creates a new instance of RevenueSharer, bound to a specific deployed contract.
func NewRevenueSharer(address common.Address, backend bind.ContractBackend) (*RevenueSharer, error) {
	contract, err := bindRevenueSharer(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RevenueSharer{RevenueSharerCaller: RevenueSharerCaller{contract: contract}, RevenueSharerTransactor: RevenueSharerTransactor{contract: contract}, RevenueSharerFilterer: RevenueSharerFilterer{contract: contract}}, nil
}

// NewRevenueSharerCaller creates a new read-only instance of RevenueSharer, bound to a specific deployed contract.
func NewRevenueSharerCaller(address common.Address, caller bind.ContractCaller) (*RevenueSharerCaller, error) {
	contract, err := bindRevenueSharer(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RevenueSharerCaller{contract: contract}, nil
}

// NewRevenueSharerTransactor creates a new write-only instance of RevenueSharer, bound to a specific deployed contract.
func NewRevenueSharerTransactor(address common.Address, transactor bind.ContractTransactor) (*RevenueSharerTransactor, error) {
	contract, err := bindRevenueSharer(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RevenueSharerTransactor{contract: contract}, nil
}

// NewRevenueSharerFilterer creates a new log filterer instance of RevenueSharer, bound to a specific deployed contract.
func NewRevenueSharerFilterer(address common.Address, filterer bind.ContractFilterer) (*RevenueSharerFilterer, error) {
	contract, err := bindRevenueSharer(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RevenueSharerFilterer{contract: contract}, nil
}

// bindRevenueSharer binds a generic wrapper to an already deployed contract.
func bindRevenueSharer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RevenueSharerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RevenueSharer *RevenueSharerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RevenueSharer.Contract.RevenueSharerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RevenueSharer *RevenueSharerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RevenueSharer.Contract.RevenueSharerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RevenueSharer *RevenueSharerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RevenueSharer.Contract.RevenueSharerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RevenueSharer *RevenueSharerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RevenueSharer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RevenueSharer *RevenueSharerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RevenueSharer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RevenueSharer *RevenueSharerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RevenueSharer.Contract.contract.Transact(opts, method, params...)
}

// BASISPOINTSCALE is a free data retrieval call binding the contract method 0x5b201d83.
//
// Solidity: function BASIS_POINT_SCALE() view returns(uint32)
func (_RevenueSharer *RevenueSharerCaller) BASISPOINTSCALE(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "BASIS_POINT_SCALE")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BASISPOINTSCALE is a free data retrieval call binding the contract method 0x5b201d83.
//
// Solidity: function BASIS_POINT_SCALE() view returns(uint32)
func (_RevenueSharer *RevenueSharerSession) BASISPOINTSCALE() (uint32, error) {
	return _RevenueSharer.Contract.BASISPOINTSCALE(&_RevenueSharer.CallOpts)
}

// BASISPOINTSCALE is a free data retrieval call binding the contract method 0x5b201d83.
//
// Solidity: function BASIS_POINT_SCALE() view returns(uint32)
func (_RevenueSharer *RevenueSharerCallerSession) BASISPOINTSCALE() (uint32, error) {
	return _RevenueSharer.Contract.BASISPOINTSCALE(&_RevenueSharer.CallOpts)
}

// FEEDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x54664de5.
//
// Solidity: function FEE_DISBURSEMENT_INTERVAL() view returns(uint256)
func (_RevenueSharer *RevenueSharerCaller) FEEDISBURSEMENTINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "FEE_DISBURSEMENT_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FEEDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x54664de5.
//
// Solidity: function FEE_DISBURSEMENT_INTERVAL() view returns(uint256)
func (_RevenueSharer *RevenueSharerSession) FEEDISBURSEMENTINTERVAL() (*big.Int, error) {
	return _RevenueSharer.Contract.FEEDISBURSEMENTINTERVAL(&_RevenueSharer.CallOpts)
}

// FEEDISBURSEMENTINTERVAL is a free data retrieval call binding the contract method 0x54664de5.
//
// Solidity: function FEE_DISBURSEMENT_INTERVAL() view returns(uint256)
func (_RevenueSharer *RevenueSharerCallerSession) FEEDISBURSEMENTINTERVAL() (*big.Int, error) {
	return _RevenueSharer.Contract.FEEDISBURSEMENTINTERVAL(&_RevenueSharer.CallOpts)
}

// L1WALLET is a free data retrieval call binding the contract method 0x36f1a6e5.
//
// Solidity: function L1_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerCaller) L1WALLET(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "L1_WALLET")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1WALLET is a free data retrieval call binding the contract method 0x36f1a6e5.
//
// Solidity: function L1_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerSession) L1WALLET() (common.Address, error) {
	return _RevenueSharer.Contract.L1WALLET(&_RevenueSharer.CallOpts)
}

// L1WALLET is a free data retrieval call binding the contract method 0x36f1a6e5.
//
// Solidity: function L1_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerCallerSession) L1WALLET() (common.Address, error) {
	return _RevenueSharer.Contract.L1WALLET(&_RevenueSharer.CallOpts)
}

// OPTIMISMGROSSREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x235d506d.
//
// Solidity: function OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerCaller) OPTIMISMGROSSREVENUESHAREBASISPOINTS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OPTIMISMGROSSREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x235d506d.
//
// Solidity: function OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerSession) OPTIMISMGROSSREVENUESHAREBASISPOINTS() (*big.Int, error) {
	return _RevenueSharer.Contract.OPTIMISMGROSSREVENUESHAREBASISPOINTS(&_RevenueSharer.CallOpts)
}

// OPTIMISMGROSSREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x235d506d.
//
// Solidity: function OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerCallerSession) OPTIMISMGROSSREVENUESHAREBASISPOINTS() (*big.Int, error) {
	return _RevenueSharer.Contract.OPTIMISMGROSSREVENUESHAREBASISPOINTS(&_RevenueSharer.CallOpts)
}

// OPTIMISMNETREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x93819a3f.
//
// Solidity: function OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerCaller) OPTIMISMNETREVENUESHAREBASISPOINTS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OPTIMISMNETREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x93819a3f.
//
// Solidity: function OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerSession) OPTIMISMNETREVENUESHAREBASISPOINTS() (*big.Int, error) {
	return _RevenueSharer.Contract.OPTIMISMNETREVENUESHAREBASISPOINTS(&_RevenueSharer.CallOpts)
}

// OPTIMISMNETREVENUESHAREBASISPOINTS is a free data retrieval call binding the contract method 0x93819a3f.
//
// Solidity: function OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS() view returns(uint256)
func (_RevenueSharer *RevenueSharerCallerSession) OPTIMISMNETREVENUESHAREBASISPOINTS() (*big.Int, error) {
	return _RevenueSharer.Contract.OPTIMISMNETREVENUESHAREBASISPOINTS(&_RevenueSharer.CallOpts)
}

// OPTIMISMWALLET is a free data retrieval call binding the contract method 0x0c8cd070.
//
// Solidity: function OPTIMISM_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerCaller) OPTIMISMWALLET(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "OPTIMISM_WALLET")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OPTIMISMWALLET is a free data retrieval call binding the contract method 0x0c8cd070.
//
// Solidity: function OPTIMISM_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerSession) OPTIMISMWALLET() (common.Address, error) {
	return _RevenueSharer.Contract.OPTIMISMWALLET(&_RevenueSharer.CallOpts)
}

// OPTIMISMWALLET is a free data retrieval call binding the contract method 0x0c8cd070.
//
// Solidity: function OPTIMISM_WALLET() view returns(address)
func (_RevenueSharer *RevenueSharerCallerSession) OPTIMISMWALLET() (common.Address, error) {
	return _RevenueSharer.Contract.OPTIMISMWALLET(&_RevenueSharer.CallOpts)
}

// WITHDRAWALMINGAS is a free data retrieval call binding the contract method 0xad41d09c.
//
// Solidity: function WITHDRAWAL_MIN_GAS() view returns(uint32)
func (_RevenueSharer *RevenueSharerCaller) WITHDRAWALMINGAS(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "WITHDRAWAL_MIN_GAS")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// WITHDRAWALMINGAS is a free data retrieval call binding the contract method 0xad41d09c.
//
// Solidity: function WITHDRAWAL_MIN_GAS() view returns(uint32)
func (_RevenueSharer *RevenueSharerSession) WITHDRAWALMINGAS() (uint32, error) {
	return _RevenueSharer.Contract.WITHDRAWALMINGAS(&_RevenueSharer.CallOpts)
}

// WITHDRAWALMINGAS is a free data retrieval call binding the contract method 0xad41d09c.
//
// Solidity: function WITHDRAWAL_MIN_GAS() view returns(uint32)
func (_RevenueSharer *RevenueSharerCallerSession) WITHDRAWALMINGAS() (uint32, error) {
	return _RevenueSharer.Contract.WITHDRAWALMINGAS(&_RevenueSharer.CallOpts)
}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint256)
func (_RevenueSharer *RevenueSharerCaller) LastDisbursementTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "lastDisbursementTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint256)
func (_RevenueSharer *RevenueSharerSession) LastDisbursementTime() (*big.Int, error) {
	return _RevenueSharer.Contract.LastDisbursementTime(&_RevenueSharer.CallOpts)
}

// LastDisbursementTime is a free data retrieval call binding the contract method 0x394d2731.
//
// Solidity: function lastDisbursementTime() view returns(uint256)
func (_RevenueSharer *RevenueSharerCallerSession) LastDisbursementTime() (*big.Int, error) {
	return _RevenueSharer.Contract.LastDisbursementTime(&_RevenueSharer.CallOpts)
}

// NetFeeRevenue is a free data retrieval call binding the contract method 0x447eb5ac.
//
// Solidity: function netFeeRevenue() view returns(uint256)
func (_RevenueSharer *RevenueSharerCaller) NetFeeRevenue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RevenueSharer.contract.Call(opts, &out, "netFeeRevenue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NetFeeRevenue is a free data retrieval call binding the contract method 0x447eb5ac.
//
// Solidity: function netFeeRevenue() view returns(uint256)
func (_RevenueSharer *RevenueSharerSession) NetFeeRevenue() (*big.Int, error) {
	return _RevenueSharer.Contract.NetFeeRevenue(&_RevenueSharer.CallOpts)
}

// NetFeeRevenue is a free data retrieval call binding the contract method 0x447eb5ac.
//
// Solidity: function netFeeRevenue() view returns(uint256)
func (_RevenueSharer *RevenueSharerCallerSession) NetFeeRevenue() (*big.Int, error) {
	return _RevenueSharer.Contract.NetFeeRevenue(&_RevenueSharer.CallOpts)
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_RevenueSharer *RevenueSharerTransactor) DisburseFees(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RevenueSharer.contract.Transact(opts, "disburseFees")
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_RevenueSharer *RevenueSharerSession) DisburseFees() (*types.Transaction, error) {
	return _RevenueSharer.Contract.DisburseFees(&_RevenueSharer.TransactOpts)
}

// DisburseFees is a paid mutator transaction binding the contract method 0xb87ea8d4.
//
// Solidity: function disburseFees() returns()
func (_RevenueSharer *RevenueSharerTransactorSession) DisburseFees() (*types.Transaction, error) {
	return _RevenueSharer.Contract.DisburseFees(&_RevenueSharer.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_RevenueSharer *RevenueSharerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RevenueSharer.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_RevenueSharer *RevenueSharerSession) Receive() (*types.Transaction, error) {
	return _RevenueSharer.Contract.Receive(&_RevenueSharer.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_RevenueSharer *RevenueSharerTransactorSession) Receive() (*types.Transaction, error) {
	return _RevenueSharer.Contract.Receive(&_RevenueSharer.TransactOpts)
}

// RevenueSharerFeesDisbursedIterator is returned from FilterFeesDisbursed and is used to iterate over the raw logs and unpacked data for FeesDisbursed events raised by the RevenueSharer contract.
type RevenueSharerFeesDisbursedIterator struct {
	Event *RevenueSharerFeesDisbursed // Event containing the contract specifics and raw log

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
func (it *RevenueSharerFeesDisbursedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RevenueSharerFeesDisbursed)
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
		it.Event = new(RevenueSharerFeesDisbursed)
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
func (it *RevenueSharerFeesDisbursedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RevenueSharerFeesDisbursedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RevenueSharerFeesDisbursed represents a FeesDisbursed event raised by the RevenueSharer contract.
type RevenueSharerFeesDisbursed struct {
	DisbursementTime   *big.Int
	PaidToOptimism     *big.Int
	TotalFeesDisbursed *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterFeesDisbursed is a free log retrieval operation binding the contract event 0xe155e054cfe69655d6d2f8bbfb856aa8cdf49ecbea6557901533364539caad94.
//
// Solidity: event FeesDisbursed(uint256 _disbursementTime, uint256 _paidToOptimism, uint256 _totalFeesDisbursed)
func (_RevenueSharer *RevenueSharerFilterer) FilterFeesDisbursed(opts *bind.FilterOpts) (*RevenueSharerFeesDisbursedIterator, error) {

	logs, sub, err := _RevenueSharer.contract.FilterLogs(opts, "FeesDisbursed")
	if err != nil {
		return nil, err
	}
	return &RevenueSharerFeesDisbursedIterator{contract: _RevenueSharer.contract, event: "FeesDisbursed", logs: logs, sub: sub}, nil
}

// WatchFeesDisbursed is a free log subscription operation binding the contract event 0xe155e054cfe69655d6d2f8bbfb856aa8cdf49ecbea6557901533364539caad94.
//
// Solidity: event FeesDisbursed(uint256 _disbursementTime, uint256 _paidToOptimism, uint256 _totalFeesDisbursed)
func (_RevenueSharer *RevenueSharerFilterer) WatchFeesDisbursed(opts *bind.WatchOpts, sink chan<- *RevenueSharerFeesDisbursed) (event.Subscription, error) {

	logs, sub, err := _RevenueSharer.contract.WatchLogs(opts, "FeesDisbursed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RevenueSharerFeesDisbursed)
				if err := _RevenueSharer.contract.UnpackLog(event, "FeesDisbursed", log); err != nil {
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

// ParseFeesDisbursed is a log parse operation binding the contract event 0xe155e054cfe69655d6d2f8bbfb856aa8cdf49ecbea6557901533364539caad94.
//
// Solidity: event FeesDisbursed(uint256 _disbursementTime, uint256 _paidToOptimism, uint256 _totalFeesDisbursed)
func (_RevenueSharer *RevenueSharerFilterer) ParseFeesDisbursed(log types.Log) (*RevenueSharerFeesDisbursed, error) {
	event := new(RevenueSharerFeesDisbursed)
	if err := _RevenueSharer.contract.UnpackLog(event, "FeesDisbursed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RevenueSharerFeesReceivedIterator is returned from FilterFeesReceived and is used to iterate over the raw logs and unpacked data for FeesReceived events raised by the RevenueSharer contract.
type RevenueSharerFeesReceivedIterator struct {
	Event *RevenueSharerFeesReceived // Event containing the contract specifics and raw log

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
func (it *RevenueSharerFeesReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RevenueSharerFeesReceived)
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
		it.Event = new(RevenueSharerFeesReceived)
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
func (it *RevenueSharerFeesReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RevenueSharerFeesReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RevenueSharerFeesReceived represents a FeesReceived event raised by the RevenueSharer contract.
type RevenueSharerFeesReceived struct {
	Sender common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterFeesReceived is a free log retrieval operation binding the contract event 0x2ccfc58c2cef4ee590b5f16be0548cc54afc12e1c66a67b362b7d640fd16bb2d.
//
// Solidity: event FeesReceived(address indexed _sender, uint256 _amount)
func (_RevenueSharer *RevenueSharerFilterer) FilterFeesReceived(opts *bind.FilterOpts, _sender []common.Address) (*RevenueSharerFeesReceivedIterator, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}

	logs, sub, err := _RevenueSharer.contract.FilterLogs(opts, "FeesReceived", _senderRule)
	if err != nil {
		return nil, err
	}
	return &RevenueSharerFeesReceivedIterator{contract: _RevenueSharer.contract, event: "FeesReceived", logs: logs, sub: sub}, nil
}

// WatchFeesReceived is a free log subscription operation binding the contract event 0x2ccfc58c2cef4ee590b5f16be0548cc54afc12e1c66a67b362b7d640fd16bb2d.
//
// Solidity: event FeesReceived(address indexed _sender, uint256 _amount)
func (_RevenueSharer *RevenueSharerFilterer) WatchFeesReceived(opts *bind.WatchOpts, sink chan<- *RevenueSharerFeesReceived, _sender []common.Address) (event.Subscription, error) {

	var _senderRule []interface{}
	for _, _senderItem := range _sender {
		_senderRule = append(_senderRule, _senderItem)
	}

	logs, sub, err := _RevenueSharer.contract.WatchLogs(opts, "FeesReceived", _senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RevenueSharerFeesReceived)
				if err := _RevenueSharer.contract.UnpackLog(event, "FeesReceived", log); err != nil {
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

// ParseFeesReceived is a log parse operation binding the contract event 0x2ccfc58c2cef4ee590b5f16be0548cc54afc12e1c66a67b362b7d640fd16bb2d.
//
// Solidity: event FeesReceived(address indexed _sender, uint256 _amount)
func (_RevenueSharer *RevenueSharerFilterer) ParseFeesReceived(log types.Log) (*RevenueSharerFeesReceived, error) {
	event := new(RevenueSharerFeesReceived)
	if err := _RevenueSharer.contract.UnpackLog(event, "FeesReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RevenueSharerNoFeesCollectedIterator is returned from FilterNoFeesCollected and is used to iterate over the raw logs and unpacked data for NoFeesCollected events raised by the RevenueSharer contract.
type RevenueSharerNoFeesCollectedIterator struct {
	Event *RevenueSharerNoFeesCollected // Event containing the contract specifics and raw log

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
func (it *RevenueSharerNoFeesCollectedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RevenueSharerNoFeesCollected)
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
		it.Event = new(RevenueSharerNoFeesCollected)
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
func (it *RevenueSharerNoFeesCollectedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RevenueSharerNoFeesCollectedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RevenueSharerNoFeesCollected represents a NoFeesCollected event raised by the RevenueSharer contract.
type RevenueSharerNoFeesCollected struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterNoFeesCollected is a free log retrieval operation binding the contract event 0x8c887b1215d5e6b119c1c1008fe1d0919b4c438301d5a0357362a13fb56f6a40.
//
// Solidity: event NoFeesCollected()
func (_RevenueSharer *RevenueSharerFilterer) FilterNoFeesCollected(opts *bind.FilterOpts) (*RevenueSharerNoFeesCollectedIterator, error) {

	logs, sub, err := _RevenueSharer.contract.FilterLogs(opts, "NoFeesCollected")
	if err != nil {
		return nil, err
	}
	return &RevenueSharerNoFeesCollectedIterator{contract: _RevenueSharer.contract, event: "NoFeesCollected", logs: logs, sub: sub}, nil
}

// WatchNoFeesCollected is a free log subscription operation binding the contract event 0x8c887b1215d5e6b119c1c1008fe1d0919b4c438301d5a0357362a13fb56f6a40.
//
// Solidity: event NoFeesCollected()
func (_RevenueSharer *RevenueSharerFilterer) WatchNoFeesCollected(opts *bind.WatchOpts, sink chan<- *RevenueSharerNoFeesCollected) (event.Subscription, error) {

	logs, sub, err := _RevenueSharer.contract.WatchLogs(opts, "NoFeesCollected")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RevenueSharerNoFeesCollected)
				if err := _RevenueSharer.contract.UnpackLog(event, "NoFeesCollected", log); err != nil {
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

// ParseNoFeesCollected is a log parse operation binding the contract event 0x8c887b1215d5e6b119c1c1008fe1d0919b4c438301d5a0357362a13fb56f6a40.
//
// Solidity: event NoFeesCollected()
func (_RevenueSharer *RevenueSharerFilterer) ParseNoFeesCollected(log types.Log) (*RevenueSharerNoFeesCollected, error) {
	event := new(RevenueSharerNoFeesCollected)
	if err := _RevenueSharer.contract.UnpackLog(event, "NoFeesCollected", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
