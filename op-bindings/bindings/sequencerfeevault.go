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

// SequencerFeeVaultMetaData contains all meta data concerning the SequencerFeeVault contract.
var SequencerFeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_minWithdrawalAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_withdrawalNetwork\",\"type\":\"uint8\",\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RECIPIENT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WITHDRAWAL_NETWORK\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1FeeWallet\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalProcessed\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"withdrawalNetwork\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"anonymous\":false}]",
	Bin: "0x60e060405234801561001057600080fd5b5060405161091638038061091683398101604081905261002f91610079565b6001600160a01b03831660a0526080829052828282806001811115610056576100566100cc565b60c081600181111561006a5761006a6100cc565b815250505050505050506100e2565b60008060006060848603121561008e57600080fd5b83516001600160a01b03811681146100a557600080fd5b602085015160408601519194509250600281106100c157600080fd5b809150509250925092565b634e487b7160e01b600052602160045260246000fd5b60805160a05160c0516107ce61014860003960008181610181015281816103c701526104020152600081816092015281816101f301528181610316015281816103a50152818161043b01526105a20152600081816101c2015261021901526107ce6000f3fe6080604052600436106100745760003560e01c806384411d651161004e57806384411d651461014b578063d0e12f901461016f578063d3e5792b146101b0578063d4ff9218146101e457600080fd5b80630d9019e1146100805780633ccfd60b146100de57806354fd4d50146100f557600080fd5b3661007b57005b600080fd5b34801561008c57600080fd5b506100b47f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100ea57600080fd5b506100f3610217565b005b34801561010157600080fd5b5061013e6040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b6040516100d5919061066e565b34801561015757600080fd5b5061016160005481565b6040519081526020016100d5565b34801561017b57600080fd5b506101a37f000000000000000000000000000000000000000000000000000000000000000081565b6040516100d591906106f2565b3480156101bc57600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b3480156101f057600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006100b4565b7f00000000000000000000000000000000000000000000000000000000000000004710156102f2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b6000479050806000808282546103089190610706565b9091555050604080518281527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff166020820152338183015290517fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9181900360600190a17f38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee817f0000000000000000000000000000000000000000000000000000000000000000337f00000000000000000000000000000000000000000000000000000000000000006040516103f69493929190610745565b60405180910390a160017f0000000000000000000000000000000000000000000000000000000000000000600181111561043257610432610688565b0361054b5760007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d80600081146104b1576040519150601f19603f3d011682016040523d82523d6000602084013e6104b6565b606091505b5050905080610547576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4665655661756c743a206661696c656420746f2073656e642045544820746f2060448201527f4c322066656520726563697069656e740000000000000000000000000000000060648201526084016102e9565b5050565b604080516020810182526000815290517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109163e11013dd9184916105ce917f0000000000000000000000000000000000000000000000000000000000000000916188b891600401610786565b6000604051808303818588803b1580156105e757600080fd5b505af11580156105fb573d6000803e3d6000fd5b505050505050565b6000815180845260005b818110156106295760208185018101518683018201520161060d565b8181111561063b576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006106816020830184610603565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b600281106106ee577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b9052565b6020810161070082846106b7565b92915050565b60008219821115610740577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b500190565b84815273ffffffffffffffffffffffffffffffffffffffff8481166020830152831660408201526080810161077d60608301846106b7565b95945050505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff8316602082015260606040820152600061077d606083018461060356fea164736f6c634300080f000a",
}

// SequencerFeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use SequencerFeeVaultMetaData.ABI instead.
var SequencerFeeVaultABI = SequencerFeeVaultMetaData.ABI

// SequencerFeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SequencerFeeVaultMetaData.Bin instead.
var SequencerFeeVaultBin = SequencerFeeVaultMetaData.Bin

// DeploySequencerFeeVault deploys a new Ethereum contract, binding an instance of SequencerFeeVault to it.
func DeploySequencerFeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (common.Address, *types.Transaction, *SequencerFeeVault, error) {
	parsed, err := SequencerFeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SequencerFeeVaultBin), backend, _recipient, _minWithdrawalAmount, _withdrawalNetwork)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SequencerFeeVault{SequencerFeeVaultCaller: SequencerFeeVaultCaller{contract: contract}, SequencerFeeVaultTransactor: SequencerFeeVaultTransactor{contract: contract}, SequencerFeeVaultFilterer: SequencerFeeVaultFilterer{contract: contract}}, nil
}

// SequencerFeeVault is an auto generated Go binding around an Ethereum contract.
type SequencerFeeVault struct {
	SequencerFeeVaultCaller     // Read-only binding to the contract
	SequencerFeeVaultTransactor // Write-only binding to the contract
	SequencerFeeVaultFilterer   // Log filterer for contract events
}

// SequencerFeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type SequencerFeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SequencerFeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SequencerFeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SequencerFeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SequencerFeeVaultSession struct {
	Contract     *SequencerFeeVault // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// SequencerFeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SequencerFeeVaultCallerSession struct {
	Contract *SequencerFeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// SequencerFeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SequencerFeeVaultTransactorSession struct {
	Contract     *SequencerFeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// SequencerFeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type SequencerFeeVaultRaw struct {
	Contract *SequencerFeeVault // Generic contract binding to access the raw methods on
}

// SequencerFeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SequencerFeeVaultCallerRaw struct {
	Contract *SequencerFeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// SequencerFeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SequencerFeeVaultTransactorRaw struct {
	Contract *SequencerFeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSequencerFeeVault creates a new instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVault(address common.Address, backend bind.ContractBackend) (*SequencerFeeVault, error) {
	contract, err := bindSequencerFeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVault{SequencerFeeVaultCaller: SequencerFeeVaultCaller{contract: contract}, SequencerFeeVaultTransactor: SequencerFeeVaultTransactor{contract: contract}, SequencerFeeVaultFilterer: SequencerFeeVaultFilterer{contract: contract}}, nil
}

// NewSequencerFeeVaultCaller creates a new read-only instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultCaller(address common.Address, caller bind.ContractCaller) (*SequencerFeeVaultCaller, error) {
	contract, err := bindSequencerFeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultCaller{contract: contract}, nil
}

// NewSequencerFeeVaultTransactor creates a new write-only instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*SequencerFeeVaultTransactor, error) {
	contract, err := bindSequencerFeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultTransactor{contract: contract}, nil
}

// NewSequencerFeeVaultFilterer creates a new log filterer instance of SequencerFeeVault, bound to a specific deployed contract.
func NewSequencerFeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*SequencerFeeVaultFilterer, error) {
	contract, err := bindSequencerFeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultFilterer{contract: contract}, nil
}

// bindSequencerFeeVault binds a generic wrapper to an already deployed contract.
func bindSequencerFeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SequencerFeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerFeeVault.Contract.SequencerFeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.SequencerFeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerFeeVault *SequencerFeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.SequencerFeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SequencerFeeVault *SequencerFeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SequencerFeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SequencerFeeVault *SequencerFeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SequencerFeeVault *SequencerFeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _SequencerFeeVault.Contract.MINWITHDRAWALAMOUNT(&_SequencerFeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _SequencerFeeVault.Contract.MINWITHDRAWALAMOUNT(&_SequencerFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCaller) RECIPIENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "RECIPIENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultSession) RECIPIENT() (common.Address, error) {
	return _SequencerFeeVault.Contract.RECIPIENT(&_SequencerFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) RECIPIENT() (common.Address, error) {
	return _SequencerFeeVault.Contract.RECIPIENT(&_SequencerFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_SequencerFeeVault *SequencerFeeVaultCaller) WITHDRAWALNETWORK(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "WITHDRAWAL_NETWORK")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_SequencerFeeVault *SequencerFeeVaultSession) WITHDRAWALNETWORK() (uint8, error) {
	return _SequencerFeeVault.Contract.WITHDRAWALNETWORK(&_SequencerFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) WITHDRAWALNETWORK() (uint8, error) {
	return _SequencerFeeVault.Contract.WITHDRAWALNETWORK(&_SequencerFeeVault.CallOpts)
}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCaller) L1FeeWallet(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "l1FeeWallet")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultSession) L1FeeWallet() (common.Address, error) {
	return _SequencerFeeVault.Contract.L1FeeWallet(&_SequencerFeeVault.CallOpts)
}

// L1FeeWallet is a free data retrieval call binding the contract method 0xd4ff9218.
//
// Solidity: function l1FeeWallet() view returns(address)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) L1FeeWallet() (common.Address, error) {
	return _SequencerFeeVault.Contract.L1FeeWallet(&_SequencerFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCaller) TotalProcessed(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "totalProcessed")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultSession) TotalProcessed() (*big.Int, error) {
	return _SequencerFeeVault.Contract.TotalProcessed(&_SequencerFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) TotalProcessed() (*big.Int, error) {
	return _SequencerFeeVault.Contract.TotalProcessed(&_SequencerFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SequencerFeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultSession) Version() (string, error) {
	return _SequencerFeeVault.Contract.Version(&_SequencerFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SequencerFeeVault *SequencerFeeVaultCallerSession) Version() (string, error) {
	return _SequencerFeeVault.Contract.Version(&_SequencerFeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Withdraw(&_SequencerFeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Withdraw(&_SequencerFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SequencerFeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultSession) Receive() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Receive(&_SequencerFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_SequencerFeeVault *SequencerFeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _SequencerFeeVault.Contract.Receive(&_SequencerFeeVault.TransactOpts)
}

// SequencerFeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawalIterator struct {
	Event *SequencerFeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *SequencerFeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerFeeVaultWithdrawal)
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
		it.Event = new(SequencerFeeVaultWithdrawal)
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
func (it *SequencerFeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerFeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerFeeVaultWithdrawal represents a Withdrawal event raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*SequencerFeeVaultWithdrawalIterator, error) {

	logs, sub, err := _SequencerFeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultWithdrawalIterator{contract: _SequencerFeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *SequencerFeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _SequencerFeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerFeeVaultWithdrawal)
				if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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

// ParseWithdrawal is a log parse operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) ParseWithdrawal(log types.Log) (*SequencerFeeVaultWithdrawal, error) {
	event := new(SequencerFeeVaultWithdrawal)
	if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SequencerFeeVaultWithdrawal0Iterator is returned from FilterWithdrawal0 and is used to iterate over the raw logs and unpacked data for Withdrawal0 events raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawal0Iterator struct {
	Event *SequencerFeeVaultWithdrawal0 // Event containing the contract specifics and raw log

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
func (it *SequencerFeeVaultWithdrawal0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SequencerFeeVaultWithdrawal0)
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
		it.Event = new(SequencerFeeVaultWithdrawal0)
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
func (it *SequencerFeeVaultWithdrawal0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SequencerFeeVaultWithdrawal0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SequencerFeeVaultWithdrawal0 represents a Withdrawal0 event raised by the SequencerFeeVault contract.
type SequencerFeeVaultWithdrawal0 struct {
	Value             *big.Int
	To                common.Address
	From              common.Address
	WithdrawalNetwork uint8
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal0 is a free log retrieval operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) FilterWithdrawal0(opts *bind.FilterOpts) (*SequencerFeeVaultWithdrawal0Iterator, error) {

	logs, sub, err := _SequencerFeeVault.contract.FilterLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return &SequencerFeeVaultWithdrawal0Iterator{contract: _SequencerFeeVault.contract, event: "Withdrawal0", logs: logs, sub: sub}, nil
}

// WatchWithdrawal0 is a free log subscription operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) WatchWithdrawal0(opts *bind.WatchOpts, sink chan<- *SequencerFeeVaultWithdrawal0) (event.Subscription, error) {

	logs, sub, err := _SequencerFeeVault.contract.WatchLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SequencerFeeVaultWithdrawal0)
				if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
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

// ParseWithdrawal0 is a log parse operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_SequencerFeeVault *SequencerFeeVaultFilterer) ParseWithdrawal0(log types.Log) (*SequencerFeeVaultWithdrawal0, error) {
	event := new(SequencerFeeVaultWithdrawal0)
	if err := _SequencerFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
