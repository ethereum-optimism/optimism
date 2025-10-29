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

// OperatorFeeVaultMetaData contains all meta data concerning the OperatorFeeVault contract.
var OperatorFeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RECIPIENT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WITHDRAWAL_NETWORK\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_minWithdrawalAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_withdrawalNetwork\",\"type\":\"uint8\",\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"minWithdrawalAmount\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"recipient\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setMinWithdrawalAmount\",\"inputs\":[{\"name\":\"_newMinWithdrawalAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setRecipient\",\"inputs\":[{\"name\":\"_newRecipient\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setWithdrawalNetwork\",\"inputs\":[{\"name\":\"_newWithdrawalNetwork\",\"type\":\"uint8\",\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"totalProcessed\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[],\"outputs\":[{\"name\":\"value_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawalNetwork\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"MinWithdrawalAmountUpdated\",\"inputs\":[{\"name\":\"oldWithdrawalAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newWithdrawalAmount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RecipientUpdated\",\"inputs\":[{\"name\":\"oldRecipient\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"newRecipient\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"withdrawalNetwork\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WithdrawalNetworkUpdated\",\"inputs\":[{\"name\":\"oldWithdrawalNetwork\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumTypes.WithdrawalNetwork\"},{\"name\":\"newWithdrawalNetwork\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumTypes.WithdrawalNetwork\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"FeeVault_OnlyProxyAdminOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]}]",
	Bin: "0x6080604052348015600e575f80fd5b5060156019565b60c9565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00805468010000000000000000900460ff161560685760405163f92ee8a960e01b815260040160405180910390fd5b80546001600160401b039081161460c65780546001600160401b0319166001600160401b0390811782556040519081527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b50565b610eb9806100d65f395ff3fe6080604052600436106100d1575f3560e01c806382356d8a1161007c57806385b5b14d1161005757806385b5b14d14610276578063b49dc74114610295578063d0e12f90146102b4578063d3e5792b146102e3575f80fd5b806382356d8a1461020f5780638312f1491461024d57806384411d6514610262575f80fd5b80633ccfd60b116100ac5780633ccfd60b1461016c57806354fd4d501461018e57806366d003ac146101e3575f80fd5b80630d9019e1146100dc578063307f29621461012c5780633bbed4a01461014d575f80fd5b366100d857005b5f80fd5b3480156100e7575f80fd5b5060025473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b348015610137575f80fd5b5061014b610146366004610c86565b6102f7565b005b348015610158575f80fd5b5061014b610167366004610cc3565b61047a565b348015610177575f80fd5b506101806105de565b604051908152602001610123565b348015610199575f80fd5b506101d66040518060400160405280600581526020017f312e312e3000000000000000000000000000000000000000000000000000000081525081565b6040516101239190610cde565b3480156101ee575f80fd5b506002546101029073ffffffffffffffffffffffffffffffffffffffff1681565b34801561021a575f80fd5b506002546102409074010000000000000000000000000000000000000000900460ff1681565b6040516101239190610d97565b348015610258575f80fd5b5061018060015481565b34801561026d575f80fd5b506101805f5481565b348015610281575f80fd5b5061014b610290366004610dab565b610918565b3480156102a0575f80fd5b5061014b6102af366004610dc2565b610a3b565b3480156102bf575f80fd5b5060025474010000000000000000000000000000000000000000900460ff16610240565b3480156102ee575f80fd5b50600154610180565b73420000000000000000000000000000000000001873ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610354573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103789190610dfd565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146103dc576040517f7cd7e09f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600280547401000000000000000000000000000000000000000080820460ff1692849290917fffffffffffffffffffffff00ffffffffffffffffffffffffffffffffffffffff9091169083600181111561043857610438610d31565b02179055507ff2ec44eb1c3b3acd547b76333eb2c4b27eee311860c57a9fdb04c95f62398fc8818360405161046e929190610e18565b60405180910390a15050565b73420000000000000000000000000000000000001873ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104d7573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104fb9190610dfd565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461055f576040517f7cd7e09f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff000000000000000000000000000000000000000083168117909355604080519190921680825260208201939093527f62e69886a5df0ba8ffcacbfc1388754e7abd9bde24b036354c561f1acd4e4593910161046e565b5f60015447101561069c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b479050805f808282546106af9190610e33565b90915550506002546040805183815273ffffffffffffffffffffffffffffffffffffffff90921660208301523382820152517fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9181900360600190a16002546040517f38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee9161077491849173ffffffffffffffffffffffffffffffffffffffff811691339174010000000000000000000000000000000000000000900460ff1690610e6b565b60405180910390a1600160025474010000000000000000000000000000000000000000900460ff1660018111156107ad576107ad610d31565b0361086a576002545f906107d79073ffffffffffffffffffffffffffffffffffffffff1683610c4f565b905080610866576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4665655661756c743a206661696c656420746f2073656e642045544820746f2060448201527f4c322066656520726563697069656e74000000000000000000000000000000006064820152608401610693565b5090565b6002546040517fc2b3e5ac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff909116600482015262061a806024820152606060448201525f60648201527342000000000000000000000000000000000000169063c2b3e5ac9083906084015f604051808303818588803b1580156108fe575f80fd5b505af1158015610910573d5f803e3d5ffd5b505050505090565b73420000000000000000000000000000000000001873ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610975573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906109999190610dfd565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146109fd576040517f7cd7e09f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600180549082905560408051828152602081018490527f895a067c78583e800418fabf3da26a9496aab2ff3429cebdf7fefa642b2e4203910161046e565b7ff0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00805468010000000000000000810460ff16159067ffffffffffffffff165f81158015610a855750825b90505f8267ffffffffffffffff166001148015610aa15750303b155b905081158015610aaf575080155b15610ae6576040517ff92ee8a900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b84547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001660011785558315610b475784547fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff16680100000000000000001785555b6002805473ffffffffffffffffffffffffffffffffffffffff8a167fffffffffffffffffffffffff000000000000000000000000000000000000000082168117835560018a81558993927fffffffffffffffffffffff000000000000000000000000000000000000000000169091179074010000000000000000000000000000000000000000908490811115610bdf57610bdf610d31565b02179055508315610c455784547fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff168555604051600181527fc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d29060200160405180910390a15b5050505050505050565b5f610c5b835a84610c62565b9392505050565b5f805f805f858888f1949350505050565b803560028110610c81575f80fd5b919050565b5f60208284031215610c96575f80fd5b610c5b82610c73565b73ffffffffffffffffffffffffffffffffffffffff81168114610cc0575f80fd5b50565b5f60208284031215610cd3575f80fd5b8135610c5b81610c9f565b602081525f82518060208401528060208501604085015e5f6040828501015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f83011684010191505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b60028110610d93577f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b9052565b60208101610da58284610d5e565b92915050565b5f60208284031215610dbb575f80fd5b5035919050565b5f805f60608486031215610dd4575f80fd5b8335610ddf81610c9f565b925060208401359150610df460408501610c73565b90509250925092565b5f60208284031215610e0d575f80fd5b8151610c5b81610c9f565b60408101610e268285610d5e565b610c5b6020830184610d5e565b80820180821115610da5577f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b84815273ffffffffffffffffffffffffffffffffffffffff84811660208301528316604082015260808101610ea36060830184610d5e565b9594505050505056fea164736f6c6343000819000a",
}

// OperatorFeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use OperatorFeeVaultMetaData.ABI instead.
var OperatorFeeVaultABI = OperatorFeeVaultMetaData.ABI

// OperatorFeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OperatorFeeVaultMetaData.Bin instead.
var OperatorFeeVaultBin = OperatorFeeVaultMetaData.Bin

// DeployOperatorFeeVault deploys a new Ethereum contract, binding an instance of OperatorFeeVault to it.
func DeployOperatorFeeVault(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OperatorFeeVault, error) {
	parsed, err := OperatorFeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OperatorFeeVaultBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OperatorFeeVault{OperatorFeeVaultCaller: OperatorFeeVaultCaller{contract: contract}, OperatorFeeVaultTransactor: OperatorFeeVaultTransactor{contract: contract}, OperatorFeeVaultFilterer: OperatorFeeVaultFilterer{contract: contract}}, nil
}

// OperatorFeeVault is an auto generated Go binding around an Ethereum contract.
type OperatorFeeVault struct {
	OperatorFeeVaultCaller     // Read-only binding to the contract
	OperatorFeeVaultTransactor // Write-only binding to the contract
	OperatorFeeVaultFilterer   // Log filterer for contract events
}

// OperatorFeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type OperatorFeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OperatorFeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OperatorFeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OperatorFeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OperatorFeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OperatorFeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OperatorFeeVaultSession struct {
	Contract     *OperatorFeeVault // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OperatorFeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OperatorFeeVaultCallerSession struct {
	Contract *OperatorFeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// OperatorFeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OperatorFeeVaultTransactorSession struct {
	Contract     *OperatorFeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// OperatorFeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type OperatorFeeVaultRaw struct {
	Contract *OperatorFeeVault // Generic contract binding to access the raw methods on
}

// OperatorFeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OperatorFeeVaultCallerRaw struct {
	Contract *OperatorFeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// OperatorFeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OperatorFeeVaultTransactorRaw struct {
	Contract *OperatorFeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOperatorFeeVault creates a new instance of OperatorFeeVault, bound to a specific deployed contract.
func NewOperatorFeeVault(address common.Address, backend bind.ContractBackend) (*OperatorFeeVault, error) {
	contract, err := bindOperatorFeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVault{OperatorFeeVaultCaller: OperatorFeeVaultCaller{contract: contract}, OperatorFeeVaultTransactor: OperatorFeeVaultTransactor{contract: contract}, OperatorFeeVaultFilterer: OperatorFeeVaultFilterer{contract: contract}}, nil
}

// NewOperatorFeeVaultCaller creates a new read-only instance of OperatorFeeVault, bound to a specific deployed contract.
func NewOperatorFeeVaultCaller(address common.Address, caller bind.ContractCaller) (*OperatorFeeVaultCaller, error) {
	contract, err := bindOperatorFeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultCaller{contract: contract}, nil
}

// NewOperatorFeeVaultTransactor creates a new write-only instance of OperatorFeeVault, bound to a specific deployed contract.
func NewOperatorFeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*OperatorFeeVaultTransactor, error) {
	contract, err := bindOperatorFeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultTransactor{contract: contract}, nil
}

// NewOperatorFeeVaultFilterer creates a new log filterer instance of OperatorFeeVault, bound to a specific deployed contract.
func NewOperatorFeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*OperatorFeeVaultFilterer, error) {
	contract, err := bindOperatorFeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultFilterer{contract: contract}, nil
}

// bindOperatorFeeVault binds a generic wrapper to an already deployed contract.
func bindOperatorFeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OperatorFeeVaultMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OperatorFeeVault *OperatorFeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OperatorFeeVault.Contract.OperatorFeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OperatorFeeVault *OperatorFeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.OperatorFeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OperatorFeeVault *OperatorFeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.OperatorFeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OperatorFeeVault *OperatorFeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OperatorFeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OperatorFeeVault *OperatorFeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OperatorFeeVault *OperatorFeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _OperatorFeeVault.Contract.MINWITHDRAWALAMOUNT(&_OperatorFeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _OperatorFeeVault.Contract.MINWITHDRAWALAMOUNT(&_OperatorFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultCaller) RECIPIENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "RECIPIENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultSession) RECIPIENT() (common.Address, error) {
	return _OperatorFeeVault.Contract.RECIPIENT(&_OperatorFeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) RECIPIENT() (common.Address, error) {
	return _OperatorFeeVault.Contract.RECIPIENT(&_OperatorFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultCaller) WITHDRAWALNETWORK(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "WITHDRAWAL_NETWORK")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultSession) WITHDRAWALNETWORK() (uint8, error) {
	return _OperatorFeeVault.Contract.WITHDRAWALNETWORK(&_OperatorFeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) WITHDRAWALNETWORK() (uint8, error) {
	return _OperatorFeeVault.Contract.WITHDRAWALNETWORK(&_OperatorFeeVault.CallOpts)
}

// MinWithdrawalAmount is a free data retrieval call binding the contract method 0x8312f149.
//
// Solidity: function minWithdrawalAmount() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCaller) MinWithdrawalAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "minWithdrawalAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinWithdrawalAmount is a free data retrieval call binding the contract method 0x8312f149.
//
// Solidity: function minWithdrawalAmount() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultSession) MinWithdrawalAmount() (*big.Int, error) {
	return _OperatorFeeVault.Contract.MinWithdrawalAmount(&_OperatorFeeVault.CallOpts)
}

// MinWithdrawalAmount is a free data retrieval call binding the contract method 0x8312f149.
//
// Solidity: function minWithdrawalAmount() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) MinWithdrawalAmount() (*big.Int, error) {
	return _OperatorFeeVault.Contract.MinWithdrawalAmount(&_OperatorFeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultCaller) Recipient(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "recipient")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultSession) Recipient() (common.Address, error) {
	return _OperatorFeeVault.Contract.Recipient(&_OperatorFeeVault.CallOpts)
}

// Recipient is a free data retrieval call binding the contract method 0x66d003ac.
//
// Solidity: function recipient() view returns(address)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) Recipient() (common.Address, error) {
	return _OperatorFeeVault.Contract.Recipient(&_OperatorFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCaller) TotalProcessed(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "totalProcessed")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultSession) TotalProcessed() (*big.Int, error) {
	return _OperatorFeeVault.Contract.TotalProcessed(&_OperatorFeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) TotalProcessed() (*big.Int, error) {
	return _OperatorFeeVault.Contract.TotalProcessed(&_OperatorFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OperatorFeeVault *OperatorFeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OperatorFeeVault *OperatorFeeVaultSession) Version() (string, error) {
	return _OperatorFeeVault.Contract.Version(&_OperatorFeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) Version() (string, error) {
	return _OperatorFeeVault.Contract.Version(&_OperatorFeeVault.CallOpts)
}

// WithdrawalNetwork is a free data retrieval call binding the contract method 0x82356d8a.
//
// Solidity: function withdrawalNetwork() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultCaller) WithdrawalNetwork(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _OperatorFeeVault.contract.Call(opts, &out, "withdrawalNetwork")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WithdrawalNetwork is a free data retrieval call binding the contract method 0x82356d8a.
//
// Solidity: function withdrawalNetwork() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultSession) WithdrawalNetwork() (uint8, error) {
	return _OperatorFeeVault.Contract.WithdrawalNetwork(&_OperatorFeeVault.CallOpts)
}

// WithdrawalNetwork is a free data retrieval call binding the contract method 0x82356d8a.
//
// Solidity: function withdrawalNetwork() view returns(uint8)
func (_OperatorFeeVault *OperatorFeeVaultCallerSession) WithdrawalNetwork() (uint8, error) {
	return _OperatorFeeVault.Contract.WithdrawalNetwork(&_OperatorFeeVault.CallOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xb49dc741.
//
// Solidity: function initialize(address _recipient, uint256 _minWithdrawalAmount, uint8 _withdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactor) Initialize(opts *bind.TransactOpts, _recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.Transact(opts, "initialize", _recipient, _minWithdrawalAmount, _withdrawalNetwork)
}

// Initialize is a paid mutator transaction binding the contract method 0xb49dc741.
//
// Solidity: function initialize(address _recipient, uint256 _minWithdrawalAmount, uint8 _withdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultSession) Initialize(_recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Initialize(&_OperatorFeeVault.TransactOpts, _recipient, _minWithdrawalAmount, _withdrawalNetwork)
}

// Initialize is a paid mutator transaction binding the contract method 0xb49dc741.
//
// Solidity: function initialize(address _recipient, uint256 _minWithdrawalAmount, uint8 _withdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) Initialize(_recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Initialize(&_OperatorFeeVault.TransactOpts, _recipient, _minWithdrawalAmount, _withdrawalNetwork)
}

// SetMinWithdrawalAmount is a paid mutator transaction binding the contract method 0x85b5b14d.
//
// Solidity: function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactor) SetMinWithdrawalAmount(opts *bind.TransactOpts, _newMinWithdrawalAmount *big.Int) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.Transact(opts, "setMinWithdrawalAmount", _newMinWithdrawalAmount)
}

// SetMinWithdrawalAmount is a paid mutator transaction binding the contract method 0x85b5b14d.
//
// Solidity: function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) returns()
func (_OperatorFeeVault *OperatorFeeVaultSession) SetMinWithdrawalAmount(_newMinWithdrawalAmount *big.Int) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetMinWithdrawalAmount(&_OperatorFeeVault.TransactOpts, _newMinWithdrawalAmount)
}

// SetMinWithdrawalAmount is a paid mutator transaction binding the contract method 0x85b5b14d.
//
// Solidity: function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) SetMinWithdrawalAmount(_newMinWithdrawalAmount *big.Int) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetMinWithdrawalAmount(&_OperatorFeeVault.TransactOpts, _newMinWithdrawalAmount)
}

// SetRecipient is a paid mutator transaction binding the contract method 0x3bbed4a0.
//
// Solidity: function setRecipient(address _newRecipient) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactor) SetRecipient(opts *bind.TransactOpts, _newRecipient common.Address) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.Transact(opts, "setRecipient", _newRecipient)
}

// SetRecipient is a paid mutator transaction binding the contract method 0x3bbed4a0.
//
// Solidity: function setRecipient(address _newRecipient) returns()
func (_OperatorFeeVault *OperatorFeeVaultSession) SetRecipient(_newRecipient common.Address) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetRecipient(&_OperatorFeeVault.TransactOpts, _newRecipient)
}

// SetRecipient is a paid mutator transaction binding the contract method 0x3bbed4a0.
//
// Solidity: function setRecipient(address _newRecipient) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) SetRecipient(_newRecipient common.Address) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetRecipient(&_OperatorFeeVault.TransactOpts, _newRecipient)
}

// SetWithdrawalNetwork is a paid mutator transaction binding the contract method 0x307f2962.
//
// Solidity: function setWithdrawalNetwork(uint8 _newWithdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactor) SetWithdrawalNetwork(opts *bind.TransactOpts, _newWithdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.Transact(opts, "setWithdrawalNetwork", _newWithdrawalNetwork)
}

// SetWithdrawalNetwork is a paid mutator transaction binding the contract method 0x307f2962.
//
// Solidity: function setWithdrawalNetwork(uint8 _newWithdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultSession) SetWithdrawalNetwork(_newWithdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetWithdrawalNetwork(&_OperatorFeeVault.TransactOpts, _newWithdrawalNetwork)
}

// SetWithdrawalNetwork is a paid mutator transaction binding the contract method 0x307f2962.
//
// Solidity: function setWithdrawalNetwork(uint8 _newWithdrawalNetwork) returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) SetWithdrawalNetwork(_newWithdrawalNetwork uint8) (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.SetWithdrawalNetwork(&_OperatorFeeVault.TransactOpts, _newWithdrawalNetwork)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns(uint256 value_)
func (_OperatorFeeVault *OperatorFeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns(uint256 value_)
func (_OperatorFeeVault *OperatorFeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Withdraw(&_OperatorFeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns(uint256 value_)
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Withdraw(&_OperatorFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OperatorFeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OperatorFeeVault *OperatorFeeVaultSession) Receive() (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Receive(&_OperatorFeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OperatorFeeVault *OperatorFeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _OperatorFeeVault.Contract.Receive(&_OperatorFeeVault.TransactOpts)
}

// OperatorFeeVaultInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OperatorFeeVault contract.
type OperatorFeeVaultInitializedIterator struct {
	Event *OperatorFeeVaultInitialized // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultInitialized)
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
		it.Event = new(OperatorFeeVaultInitialized)
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
func (it *OperatorFeeVaultInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultInitialized represents a Initialized event raised by the OperatorFeeVault contract.
type OperatorFeeVaultInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterInitialized(opts *bind.FilterOpts) (*OperatorFeeVaultInitializedIterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultInitializedIterator{contract: _OperatorFeeVault.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultInitialized) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultInitialized)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseInitialized(log types.Log) (*OperatorFeeVaultInitialized, error) {
	event := new(OperatorFeeVaultInitialized)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OperatorFeeVaultMinWithdrawalAmountUpdatedIterator is returned from FilterMinWithdrawalAmountUpdated and is used to iterate over the raw logs and unpacked data for MinWithdrawalAmountUpdated events raised by the OperatorFeeVault contract.
type OperatorFeeVaultMinWithdrawalAmountUpdatedIterator struct {
	Event *OperatorFeeVaultMinWithdrawalAmountUpdated // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultMinWithdrawalAmountUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultMinWithdrawalAmountUpdated)
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
		it.Event = new(OperatorFeeVaultMinWithdrawalAmountUpdated)
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
func (it *OperatorFeeVaultMinWithdrawalAmountUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultMinWithdrawalAmountUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultMinWithdrawalAmountUpdated represents a MinWithdrawalAmountUpdated event raised by the OperatorFeeVault contract.
type OperatorFeeVaultMinWithdrawalAmountUpdated struct {
	OldWithdrawalAmount *big.Int
	NewWithdrawalAmount *big.Int
	Raw                 types.Log // Blockchain specific contextual infos
}

// FilterMinWithdrawalAmountUpdated is a free log retrieval operation binding the contract event 0x895a067c78583e800418fabf3da26a9496aab2ff3429cebdf7fefa642b2e4203.
//
// Solidity: event MinWithdrawalAmountUpdated(uint256 oldWithdrawalAmount, uint256 newWithdrawalAmount)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterMinWithdrawalAmountUpdated(opts *bind.FilterOpts) (*OperatorFeeVaultMinWithdrawalAmountUpdatedIterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "MinWithdrawalAmountUpdated")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultMinWithdrawalAmountUpdatedIterator{contract: _OperatorFeeVault.contract, event: "MinWithdrawalAmountUpdated", logs: logs, sub: sub}, nil
}

// WatchMinWithdrawalAmountUpdated is a free log subscription operation binding the contract event 0x895a067c78583e800418fabf3da26a9496aab2ff3429cebdf7fefa642b2e4203.
//
// Solidity: event MinWithdrawalAmountUpdated(uint256 oldWithdrawalAmount, uint256 newWithdrawalAmount)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchMinWithdrawalAmountUpdated(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultMinWithdrawalAmountUpdated) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "MinWithdrawalAmountUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultMinWithdrawalAmountUpdated)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "MinWithdrawalAmountUpdated", log); err != nil {
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

// ParseMinWithdrawalAmountUpdated is a log parse operation binding the contract event 0x895a067c78583e800418fabf3da26a9496aab2ff3429cebdf7fefa642b2e4203.
//
// Solidity: event MinWithdrawalAmountUpdated(uint256 oldWithdrawalAmount, uint256 newWithdrawalAmount)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseMinWithdrawalAmountUpdated(log types.Log) (*OperatorFeeVaultMinWithdrawalAmountUpdated, error) {
	event := new(OperatorFeeVaultMinWithdrawalAmountUpdated)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "MinWithdrawalAmountUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OperatorFeeVaultRecipientUpdatedIterator is returned from FilterRecipientUpdated and is used to iterate over the raw logs and unpacked data for RecipientUpdated events raised by the OperatorFeeVault contract.
type OperatorFeeVaultRecipientUpdatedIterator struct {
	Event *OperatorFeeVaultRecipientUpdated // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultRecipientUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultRecipientUpdated)
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
		it.Event = new(OperatorFeeVaultRecipientUpdated)
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
func (it *OperatorFeeVaultRecipientUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultRecipientUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultRecipientUpdated represents a RecipientUpdated event raised by the OperatorFeeVault contract.
type OperatorFeeVaultRecipientUpdated struct {
	OldRecipient common.Address
	NewRecipient common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRecipientUpdated is a free log retrieval operation binding the contract event 0x62e69886a5df0ba8ffcacbfc1388754e7abd9bde24b036354c561f1acd4e4593.
//
// Solidity: event RecipientUpdated(address oldRecipient, address newRecipient)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterRecipientUpdated(opts *bind.FilterOpts) (*OperatorFeeVaultRecipientUpdatedIterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "RecipientUpdated")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultRecipientUpdatedIterator{contract: _OperatorFeeVault.contract, event: "RecipientUpdated", logs: logs, sub: sub}, nil
}

// WatchRecipientUpdated is a free log subscription operation binding the contract event 0x62e69886a5df0ba8ffcacbfc1388754e7abd9bde24b036354c561f1acd4e4593.
//
// Solidity: event RecipientUpdated(address oldRecipient, address newRecipient)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchRecipientUpdated(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultRecipientUpdated) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "RecipientUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultRecipientUpdated)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "RecipientUpdated", log); err != nil {
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

// ParseRecipientUpdated is a log parse operation binding the contract event 0x62e69886a5df0ba8ffcacbfc1388754e7abd9bde24b036354c561f1acd4e4593.
//
// Solidity: event RecipientUpdated(address oldRecipient, address newRecipient)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseRecipientUpdated(log types.Log) (*OperatorFeeVaultRecipientUpdated, error) {
	event := new(OperatorFeeVaultRecipientUpdated)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "RecipientUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OperatorFeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawalIterator struct {
	Event *OperatorFeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultWithdrawal)
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
		it.Event = new(OperatorFeeVaultWithdrawal)
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
func (it *OperatorFeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultWithdrawal represents a Withdrawal event raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*OperatorFeeVaultWithdrawalIterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultWithdrawalIterator{contract: _OperatorFeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultWithdrawal)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseWithdrawal(log types.Log) (*OperatorFeeVaultWithdrawal, error) {
	event := new(OperatorFeeVaultWithdrawal)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OperatorFeeVaultWithdrawal0Iterator is returned from FilterWithdrawal0 and is used to iterate over the raw logs and unpacked data for Withdrawal0 events raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawal0Iterator struct {
	Event *OperatorFeeVaultWithdrawal0 // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultWithdrawal0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultWithdrawal0)
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
		it.Event = new(OperatorFeeVaultWithdrawal0)
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
func (it *OperatorFeeVaultWithdrawal0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultWithdrawal0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultWithdrawal0 represents a Withdrawal0 event raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawal0 struct {
	Value             *big.Int
	To                common.Address
	From              common.Address
	WithdrawalNetwork uint8
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal0 is a free log retrieval operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterWithdrawal0(opts *bind.FilterOpts) (*OperatorFeeVaultWithdrawal0Iterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultWithdrawal0Iterator{contract: _OperatorFeeVault.contract, event: "Withdrawal0", logs: logs, sub: sub}, nil
}

// WatchWithdrawal0 is a free log subscription operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchWithdrawal0(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultWithdrawal0) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultWithdrawal0)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
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
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseWithdrawal0(log types.Log) (*OperatorFeeVaultWithdrawal0, error) {
	event := new(OperatorFeeVaultWithdrawal0)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OperatorFeeVaultWithdrawalNetworkUpdatedIterator is returned from FilterWithdrawalNetworkUpdated and is used to iterate over the raw logs and unpacked data for WithdrawalNetworkUpdated events raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawalNetworkUpdatedIterator struct {
	Event *OperatorFeeVaultWithdrawalNetworkUpdated // Event containing the contract specifics and raw log

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
func (it *OperatorFeeVaultWithdrawalNetworkUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OperatorFeeVaultWithdrawalNetworkUpdated)
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
		it.Event = new(OperatorFeeVaultWithdrawalNetworkUpdated)
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
func (it *OperatorFeeVaultWithdrawalNetworkUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OperatorFeeVaultWithdrawalNetworkUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OperatorFeeVaultWithdrawalNetworkUpdated represents a WithdrawalNetworkUpdated event raised by the OperatorFeeVault contract.
type OperatorFeeVaultWithdrawalNetworkUpdated struct {
	OldWithdrawalNetwork uint8
	NewWithdrawalNetwork uint8
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalNetworkUpdated is a free log retrieval operation binding the contract event 0xf2ec44eb1c3b3acd547b76333eb2c4b27eee311860c57a9fdb04c95f62398fc8.
//
// Solidity: event WithdrawalNetworkUpdated(uint8 oldWithdrawalNetwork, uint8 newWithdrawalNetwork)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) FilterWithdrawalNetworkUpdated(opts *bind.FilterOpts) (*OperatorFeeVaultWithdrawalNetworkUpdatedIterator, error) {

	logs, sub, err := _OperatorFeeVault.contract.FilterLogs(opts, "WithdrawalNetworkUpdated")
	if err != nil {
		return nil, err
	}
	return &OperatorFeeVaultWithdrawalNetworkUpdatedIterator{contract: _OperatorFeeVault.contract, event: "WithdrawalNetworkUpdated", logs: logs, sub: sub}, nil
}

// WatchWithdrawalNetworkUpdated is a free log subscription operation binding the contract event 0xf2ec44eb1c3b3acd547b76333eb2c4b27eee311860c57a9fdb04c95f62398fc8.
//
// Solidity: event WithdrawalNetworkUpdated(uint8 oldWithdrawalNetwork, uint8 newWithdrawalNetwork)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) WatchWithdrawalNetworkUpdated(opts *bind.WatchOpts, sink chan<- *OperatorFeeVaultWithdrawalNetworkUpdated) (event.Subscription, error) {

	logs, sub, err := _OperatorFeeVault.contract.WatchLogs(opts, "WithdrawalNetworkUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OperatorFeeVaultWithdrawalNetworkUpdated)
				if err := _OperatorFeeVault.contract.UnpackLog(event, "WithdrawalNetworkUpdated", log); err != nil {
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

// ParseWithdrawalNetworkUpdated is a log parse operation binding the contract event 0xf2ec44eb1c3b3acd547b76333eb2c4b27eee311860c57a9fdb04c95f62398fc8.
//
// Solidity: event WithdrawalNetworkUpdated(uint8 oldWithdrawalNetwork, uint8 newWithdrawalNetwork)
func (_OperatorFeeVault *OperatorFeeVaultFilterer) ParseWithdrawalNetworkUpdated(log types.Log) (*OperatorFeeVaultWithdrawalNetworkUpdated, error) {
	event := new(OperatorFeeVaultWithdrawalNetworkUpdated)
	if err := _OperatorFeeVault.contract.UnpackLog(event, "WithdrawalNetworkUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
