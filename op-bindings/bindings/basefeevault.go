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

// BaseFeeVaultMetaData contains all meta data concerning the BaseFeeVault contract.
var BaseFeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_minWithdrawalAmount\",\"type\":\"uint256\"},{\"internalType\":\"enumFeeVault.WithdrawalNetwork\",\"name\":\"_withdrawalNetwork\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"enumFeeVault.WithdrawalNetwork\",\"name\":\"withdrawalNetwork\",\"type\":\"uint8\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RECIPIENT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"WITHDRAWAL_NETWORK\",\"outputs\":[{\"internalType\":\"enumFeeVault.WithdrawalNetwork\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalProcessed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x61014060405234801561001157600080fd5b50604051610c44380380610c448339810160408190526100309161008f565b6001600160a01b03831660a05260808290526001600381858585808481111561005b5761005b6100e2565b60c081600181111561006f5761006f6100e2565b905250505060e0939093526101009190915261012052506100f892505050565b6000806000606084860312156100a457600080fd5b83516001600160a01b03811681146100bb57600080fd5b602085015160408601519194509250600281106100d757600080fd5b809150509250925092565b634e487b7160e01b600052602160045260246000fd5b60805160a05160c05160e0516101005161012051610acd61017760003960006105ea015260006105c1015260006105980152600081816101420152818161035501526103900152600081816087015281816102a401528181610333015281816103c9015261053001526000818161018301526101a70152610acd6000f3fe6080604052600436106100695760003560e01c806384411d651161004357806384411d651461010c578063d0e12f9014610130578063d3e5792b1461017157600080fd5b80630d9019e1146100755780633ccfd60b146100d357806354fd4d50146100ea57600080fd5b3661007057005b600080fd5b34801561008157600080fd5b506100a97f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100df57600080fd5b506100e86101a5565b005b3480156100f657600080fd5b506100ff610591565b6040516100ca91906107eb565b34801561011857600080fd5b5061012260005481565b6040519081526020016100ca565b34801561013c57600080fd5b506101647f000000000000000000000000000000000000000000000000000000000000000081565b6040516100ca919061086f565b34801561017d57600080fd5b506101227f000000000000000000000000000000000000000000000000000000000000000081565b7f0000000000000000000000000000000000000000000000000000000000000000471015610280576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b60004790508060008082825461029691906108b2565b9091555050604080518281527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff166020820152338183015290517fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9181900360600190a17f38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee817f0000000000000000000000000000000000000000000000000000000000000000337f000000000000000000000000000000000000000000000000000000000000000060405161038494939291906108ca565b60405180910390a160017f000000000000000000000000000000000000000000000000000000000000000060018111156103c0576103c0610805565b036104d95760007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d806000811461043f576040519150601f19603f3d011682016040523d82523d6000602084013e610444565b606091505b50509050806104d5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4665655661756c743a206661696c656420746f2073656e642045544820746f2060448201527f4c322066656520726563697069656e74000000000000000000000000000000006064820152608401610277565b5050565b604080516020810182526000815290517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109163e11013dd91849161055c917f0000000000000000000000000000000000000000000000000000000000000000916188b89160040161090b565b6000604051808303818588803b15801561057557600080fd5b505af1158015610589573d6000803e3d6000fd5b505050505050565b60606105bc7f0000000000000000000000000000000000000000000000000000000000000000610634565b6105e57f0000000000000000000000000000000000000000000000000000000000000000610634565b61060e7f0000000000000000000000000000000000000000000000000000000000000000610634565b60405160200161062093929190610946565b604051602081830303815290604052905090565b60608160000361067757505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156106a1578061068b816109bc565b915061069a9050600a83610a23565b915061067b565b60008167ffffffffffffffff8111156106bc576106bc610a37565b6040519080825280601f01601f1916602001820160405280156106e6576020820181803683370190505b5090505b8415610769576106fb600183610a66565b9150610708600a86610a7d565b6107139060306108b2565b60f81b81838151811061072857610728610a91565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610762600a86610a23565b94506106ea565b949350505050565b60005b8381101561078c578181015183820152602001610774565b8381111561079b576000848401525b50505050565b600081518084526107b9816020860160208601610771565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006107fe60208301846107a1565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b6002811061086b577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b9052565b6020810161087d8284610834565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156108c5576108c5610883565b500190565b84815273ffffffffffffffffffffffffffffffffffffffff848116602083015283166040820152608081016109026060830184610834565b95945050505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff8316602082015260606040820152600061090260608301846107a1565b60008451610958818460208901610771565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610994816001850160208a01610771565b600192019182015283516109af816002840160208801610771565b0160020195945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036109ed576109ed610883565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610a3257610a326109f4565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082821015610a7857610a78610883565b500390565b600082610a8c57610a8c6109f4565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a",
}

// BaseFeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use BaseFeeVaultMetaData.ABI instead.
var BaseFeeVaultABI = BaseFeeVaultMetaData.ABI

// BaseFeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use BaseFeeVaultMetaData.Bin instead.
var BaseFeeVaultBin = BaseFeeVaultMetaData.Bin

// DeployBaseFeeVault deploys a new Ethereum contract, binding an instance of BaseFeeVault to it.
func DeployBaseFeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (common.Address, *types.Transaction, *BaseFeeVault, error) {
	parsed, err := BaseFeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BaseFeeVaultBin), backend, _recipient, _minWithdrawalAmount, _withdrawalNetwork)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BaseFeeVault{BaseFeeVaultCaller: BaseFeeVaultCaller{contract: contract}, BaseFeeVaultTransactor: BaseFeeVaultTransactor{contract: contract}, BaseFeeVaultFilterer: BaseFeeVaultFilterer{contract: contract}}, nil
}

// BaseFeeVault is an auto generated Go binding around an Ethereum contract.
type BaseFeeVault struct {
	BaseFeeVaultCaller     // Read-only binding to the contract
	BaseFeeVaultTransactor // Write-only binding to the contract
	BaseFeeVaultFilterer   // Log filterer for contract events
}

// BaseFeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type BaseFeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BaseFeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BaseFeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BaseFeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BaseFeeVaultSession struct {
	Contract     *BaseFeeVault     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BaseFeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BaseFeeVaultCallerSession struct {
	Contract *BaseFeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// BaseFeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BaseFeeVaultTransactorSession struct {
	Contract     *BaseFeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// BaseFeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type BaseFeeVaultRaw struct {
	Contract *BaseFeeVault // Generic contract binding to access the raw methods on
}

// BaseFeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BaseFeeVaultCallerRaw struct {
	Contract *BaseFeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// BaseFeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BaseFeeVaultTransactorRaw struct {
	Contract *BaseFeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBaseFeeVault creates a new instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVault(address common.Address, backend bind.ContractBackend) (*BaseFeeVault, error) {
	contract, err := bindBaseFeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVault{BaseFeeVaultCaller: BaseFeeVaultCaller{contract: contract}, BaseFeeVaultTransactor: BaseFeeVaultTransactor{contract: contract}, BaseFeeVaultFilterer: BaseFeeVaultFilterer{contract: contract}}, nil
}

// NewBaseFeeVaultCaller creates a new read-only instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultCaller(address common.Address, caller bind.ContractCaller) (*BaseFeeVaultCaller, error) {
	contract, err := bindBaseFeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultCaller{contract: contract}, nil
}

// NewBaseFeeVaultTransactor creates a new write-only instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*BaseFeeVaultTransactor, error) {
	contract, err := bindBaseFeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultTransactor{contract: contract}, nil
}

// NewBaseFeeVaultFilterer creates a new log filterer instance of BaseFeeVault, bound to a specific deployed contract.
func NewBaseFeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*BaseFeeVaultFilterer, error) {
	contract, err := bindBaseFeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultFilterer{contract: contract}, nil
}

// bindBaseFeeVault binds a generic wrapper to an already deployed contract.
func bindBaseFeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BaseFeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseFeeVault *BaseFeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseFeeVault.Contract.BaseFeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseFeeVault *BaseFeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.BaseFeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseFeeVault *BaseFeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.BaseFeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BaseFeeVault *BaseFeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BaseFeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BaseFeeVault *BaseFeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BaseFeeVault *BaseFeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BaseFeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BaseFeeVault.Contract.MINWITHDRAWALAMOUNT(&_BaseFeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _BaseFeeVault.Contract.MINWITHDRAWALAMOUNT(&_BaseFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultCaller) RECIPIENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "RECIPIENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultSession) RECIPIENT() (common.Address, error) {
	return _BaseFeeVault.Contract.RECIPIENT(&_BaseFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_BaseFeeVault *BaseFeeVaultCallerSession) RECIPIENT() (common.Address, error) {
	return _BaseFeeVault.Contract.RECIPIENT(&_BaseFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_BaseFeeVault *BaseFeeVaultCaller) WITHDRAWALNETWORK(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "WITHDRAWAL_NETWORK")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_BaseFeeVault *BaseFeeVaultSession) WITHDRAWALNETWORK() (uint8, error) {
	return _BaseFeeVault.Contract.WITHDRAWALNETWORK(&_BaseFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_BaseFeeVault *BaseFeeVaultCallerSession) WITHDRAWALNETWORK() (uint8, error) {
	return _BaseFeeVault.Contract.WITHDRAWALNETWORK(&_BaseFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCaller) TotalProcessed(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "totalProcessed")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultSession) TotalProcessed() (*big.Int, error) {
	return _BaseFeeVault.Contract.TotalProcessed(&_BaseFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_BaseFeeVault *BaseFeeVaultCallerSession) TotalProcessed() (*big.Int, error) {
	return _BaseFeeVault.Contract.TotalProcessed(&_BaseFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BaseFeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultSession) Version() (string, error) {
	return _BaseFeeVault.Contract.Version(&_BaseFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_BaseFeeVault *BaseFeeVaultCallerSession) Version() (string, error) {
	return _BaseFeeVault.Contract.Version(&_BaseFeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Withdraw(&_BaseFeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_BaseFeeVault *BaseFeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Withdraw(&_BaseFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BaseFeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultSession) Receive() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Receive(&_BaseFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_BaseFeeVault *BaseFeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _BaseFeeVault.Contract.Receive(&_BaseFeeVault.TransactOpts)
}

// BaseFeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawalIterator struct {
	Event *BaseFeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *BaseFeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BaseFeeVaultWithdrawal)
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
		it.Event = new(BaseFeeVaultWithdrawal)
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
func (it *BaseFeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BaseFeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BaseFeeVaultWithdrawal represents a Withdrawal event raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_BaseFeeVault *BaseFeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*BaseFeeVaultWithdrawalIterator, error) {

	logs, sub, err := _BaseFeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultWithdrawalIterator{contract: _BaseFeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_BaseFeeVault *BaseFeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *BaseFeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _BaseFeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BaseFeeVaultWithdrawal)
				if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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
func (_BaseFeeVault *BaseFeeVaultFilterer) ParseWithdrawal(log types.Log) (*BaseFeeVaultWithdrawal, error) {
	event := new(BaseFeeVaultWithdrawal)
	if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BaseFeeVaultWithdrawal0Iterator is returned from FilterWithdrawal0 and is used to iterate over the raw logs and unpacked data for Withdrawal0 events raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawal0Iterator struct {
	Event *BaseFeeVaultWithdrawal0 // Event containing the contract specifics and raw log

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
func (it *BaseFeeVaultWithdrawal0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BaseFeeVaultWithdrawal0)
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
		it.Event = new(BaseFeeVaultWithdrawal0)
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
func (it *BaseFeeVaultWithdrawal0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BaseFeeVaultWithdrawal0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BaseFeeVaultWithdrawal0 represents a Withdrawal0 event raised by the BaseFeeVault contract.
type BaseFeeVaultWithdrawal0 struct {
	Value             *big.Int
	To                common.Address
	From              common.Address
	WithdrawalNetwork uint8
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal0 is a free log retrieval operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_BaseFeeVault *BaseFeeVaultFilterer) FilterWithdrawal0(opts *bind.FilterOpts) (*BaseFeeVaultWithdrawal0Iterator, error) {

	logs, sub, err := _BaseFeeVault.contract.FilterLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return &BaseFeeVaultWithdrawal0Iterator{contract: _BaseFeeVault.contract, event: "Withdrawal0", logs: logs, sub: sub}, nil
}

// WatchWithdrawal0 is a free log subscription operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_BaseFeeVault *BaseFeeVaultFilterer) WatchWithdrawal0(opts *bind.WatchOpts, sink chan<- *BaseFeeVaultWithdrawal0) (event.Subscription, error) {

	logs, sub, err := _BaseFeeVault.contract.WatchLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BaseFeeVaultWithdrawal0)
				if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
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
func (_BaseFeeVault *BaseFeeVaultFilterer) ParseWithdrawal0(log types.Log) (*BaseFeeVaultWithdrawal0, error) {
	event := new(BaseFeeVaultWithdrawal0)
	if err := _BaseFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
