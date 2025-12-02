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

// L1FeeVaultMetaData contains all meta data concerning the L1FeeVault contract.
var L1FeeVaultMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_minWithdrawalAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_withdrawalNetwork\",\"type\":\"uint8\",\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"MIN_WITHDRAWAL_AMOUNT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"RECIPIENT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"WITHDRAWAL_NETWORK\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalProcessed\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"from\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"withdrawalNetwork\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumFeeVault.WithdrawalNetwork\"}],\"anonymous\":false}]",
	Bin: "0x60e060405234801561001057600080fd5b506040516108d13803806108d183398101604081905261002f91610079565b6001600160a01b03831660a0526080829052828282806001811115610056576100566100cc565b60c081600181111561006a5761006a6100cc565b815250505050505050506100e2565b60008060006060848603121561008e57600080fd5b83516001600160a01b03811681146100a557600080fd5b602085015160408601519194509250600281106100c157600080fd5b809150509250925092565b634e487b7160e01b600052602160045260246000fd5b60805160a05160c051610790610141600039600081816101760152818161038901526103c40152600081816087015281816102d801528181610367015281816103fd01526105640152600081816101b701526101db01526107906000f3fe6080604052600436106100695760003560e01c806384411d651161004357806384411d6514610140578063d0e12f9014610164578063d3e5792b146101a557600080fd5b80630d9019e1146100755780633ccfd60b146100d357806354fd4d50146100ea57600080fd5b3661007057005b600080fd5b34801561008157600080fd5b506100a97f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156100df57600080fd5b506100e86101d9565b005b3480156100f657600080fd5b506101336040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b6040516100ca9190610630565b34801561014c57600080fd5b5061015660005481565b6040519081526020016100ca565b34801561017057600080fd5b506101987f000000000000000000000000000000000000000000000000000000000000000081565b6040516100ca91906106b4565b3480156101b157600080fd5b506101567f000000000000000000000000000000000000000000000000000000000000000081565b7f00000000000000000000000000000000000000000000000000000000000000004710156102b4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4665655661756c743a207769746864726177616c20616d6f756e74206d75737460448201527f2062652067726561746572207468616e206d696e696d756d207769746864726160648201527f77616c20616d6f756e7400000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b6000479050806000808282546102ca91906106c8565b9091555050604080518281527f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff166020820152338183015290517fc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba9181900360600190a17f38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee817f0000000000000000000000000000000000000000000000000000000000000000337f00000000000000000000000000000000000000000000000000000000000000006040516103b89493929190610707565b60405180910390a160017f000000000000000000000000000000000000000000000000000000000000000060018111156103f4576103f461064a565b0361050d5760007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d8060008114610473576040519150601f19603f3d011682016040523d82523d6000602084013e610478565b606091505b5050905080610509576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4665655661756c743a206661696c656420746f2073656e642045544820746f2060448201527f4c322066656520726563697069656e740000000000000000000000000000000060648201526084016102ab565b5050565b604080516020810182526000815290517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109163e11013dd918491610590917f0000000000000000000000000000000000000000000000000000000000000000916188b891600401610748565b6000604051808303818588803b1580156105a957600080fd5b505af11580156105bd573d6000803e3d6000fd5b505050505050565b6000815180845260005b818110156105eb576020818501810151868301820152016105cf565b818111156105fd576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061064360208301846105c5565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b600281106106b0577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b9052565b602081016106c28284610679565b92915050565b60008219821115610702577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b500190565b84815273ffffffffffffffffffffffffffffffffffffffff8481166020830152831660408201526080810161073f6060830184610679565b95945050505050565b73ffffffffffffffffffffffffffffffffffffffff8416815263ffffffff8316602082015260606040820152600061073f60608301846105c556fea164736f6c634300080f000a",
}

// L1FeeVaultABI is the input ABI used to generate the binding from.
// Deprecated: Use L1FeeVaultMetaData.ABI instead.
var L1FeeVaultABI = L1FeeVaultMetaData.ABI

// L1FeeVaultBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1FeeVaultMetaData.Bin instead.
var L1FeeVaultBin = L1FeeVaultMetaData.Bin

// DeployL1FeeVault deploys a new Ethereum contract, binding an instance of L1FeeVault to it.
func DeployL1FeeVault(auth *bind.TransactOpts, backend bind.ContractBackend, _recipient common.Address, _minWithdrawalAmount *big.Int, _withdrawalNetwork uint8) (common.Address, *types.Transaction, *L1FeeVault, error) {
	parsed, err := L1FeeVaultMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1FeeVaultBin), backend, _recipient, _minWithdrawalAmount, _withdrawalNetwork)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1FeeVault{L1FeeVaultCaller: L1FeeVaultCaller{contract: contract}, L1FeeVaultTransactor: L1FeeVaultTransactor{contract: contract}, L1FeeVaultFilterer: L1FeeVaultFilterer{contract: contract}}, nil
}

// L1FeeVault is an auto generated Go binding around an Ethereum contract.
type L1FeeVault struct {
	L1FeeVaultCaller     // Read-only binding to the contract
	L1FeeVaultTransactor // Write-only binding to the contract
	L1FeeVaultFilterer   // Log filterer for contract events
}

// L1FeeVaultCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1FeeVaultCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1FeeVaultTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1FeeVaultFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1FeeVaultSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1FeeVaultSession struct {
	Contract     *L1FeeVault       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1FeeVaultCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1FeeVaultCallerSession struct {
	Contract *L1FeeVaultCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// L1FeeVaultTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1FeeVaultTransactorSession struct {
	Contract     *L1FeeVaultTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// L1FeeVaultRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1FeeVaultRaw struct {
	Contract *L1FeeVault // Generic contract binding to access the raw methods on
}

// L1FeeVaultCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1FeeVaultCallerRaw struct {
	Contract *L1FeeVaultCaller // Generic read-only contract binding to access the raw methods on
}

// L1FeeVaultTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1FeeVaultTransactorRaw struct {
	Contract *L1FeeVaultTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1FeeVault creates a new instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVault(address common.Address, backend bind.ContractBackend) (*L1FeeVault, error) {
	contract, err := bindL1FeeVault(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1FeeVault{L1FeeVaultCaller: L1FeeVaultCaller{contract: contract}, L1FeeVaultTransactor: L1FeeVaultTransactor{contract: contract}, L1FeeVaultFilterer: L1FeeVaultFilterer{contract: contract}}, nil
}

// NewL1FeeVaultCaller creates a new read-only instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultCaller(address common.Address, caller bind.ContractCaller) (*L1FeeVaultCaller, error) {
	contract, err := bindL1FeeVault(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultCaller{contract: contract}, nil
}

// NewL1FeeVaultTransactor creates a new write-only instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultTransactor(address common.Address, transactor bind.ContractTransactor) (*L1FeeVaultTransactor, error) {
	contract, err := bindL1FeeVault(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultTransactor{contract: contract}, nil
}

// NewL1FeeVaultFilterer creates a new log filterer instance of L1FeeVault, bound to a specific deployed contract.
func NewL1FeeVaultFilterer(address common.Address, filterer bind.ContractFilterer) (*L1FeeVaultFilterer, error) {
	contract, err := bindL1FeeVault(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultFilterer{contract: contract}, nil
}

// bindL1FeeVault binds a generic wrapper to an already deployed contract.
func bindL1FeeVault(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1FeeVaultABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1FeeVault *L1FeeVaultRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1FeeVault.Contract.L1FeeVaultCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1FeeVault *L1FeeVaultRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.Contract.L1FeeVaultTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1FeeVault *L1FeeVaultRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1FeeVault.Contract.L1FeeVaultTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1FeeVault *L1FeeVaultCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1FeeVault.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1FeeVault *L1FeeVaultTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1FeeVault *L1FeeVaultTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1FeeVault.Contract.contract.Transact(opts, method, params...)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCaller) MINWITHDRAWALAMOUNT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "MIN_WITHDRAWAL_AMOUNT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _L1FeeVault.Contract.MINWITHDRAWALAMOUNT(&_L1FeeVault.CallOpts)
}

// MINWITHDRAWALAMOUNT is a free data retrieval call binding the contract method 0xd3e5792b.
//
// Solidity: function MIN_WITHDRAWAL_AMOUNT() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCallerSession) MINWITHDRAWALAMOUNT() (*big.Int, error) {
	return _L1FeeVault.Contract.MINWITHDRAWALAMOUNT(&_L1FeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_L1FeeVault *L1FeeVaultCaller) RECIPIENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "RECIPIENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_L1FeeVault *L1FeeVaultSession) RECIPIENT() (common.Address, error) {
	return _L1FeeVault.Contract.RECIPIENT(&_L1FeeVault.CallOpts)
}

// RECIPIENT is a free data retrieval call binding the contract method 0x0d9019e1.
//
// Solidity: function RECIPIENT() view returns(address)
func (_L1FeeVault *L1FeeVaultCallerSession) RECIPIENT() (common.Address, error) {
	return _L1FeeVault.Contract.RECIPIENT(&_L1FeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_L1FeeVault *L1FeeVaultCaller) WITHDRAWALNETWORK(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "WITHDRAWAL_NETWORK")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_L1FeeVault *L1FeeVaultSession) WITHDRAWALNETWORK() (uint8, error) {
	return _L1FeeVault.Contract.WITHDRAWALNETWORK(&_L1FeeVault.CallOpts)
}

// WITHDRAWALNETWORK is a free data retrieval call binding the contract method 0xd0e12f90.
//
// Solidity: function WITHDRAWAL_NETWORK() view returns(uint8)
func (_L1FeeVault *L1FeeVaultCallerSession) WITHDRAWALNETWORK() (uint8, error) {
	return _L1FeeVault.Contract.WITHDRAWALNETWORK(&_L1FeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCaller) TotalProcessed(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "totalProcessed")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_L1FeeVault *L1FeeVaultSession) TotalProcessed() (*big.Int, error) {
	return _L1FeeVault.Contract.TotalProcessed(&_L1FeeVault.CallOpts)
}

// TotalProcessed is a free data retrieval call binding the contract method 0x84411d65.
//
// Solidity: function totalProcessed() view returns(uint256)
func (_L1FeeVault *L1FeeVaultCallerSession) TotalProcessed() (*big.Int, error) {
	return _L1FeeVault.Contract.TotalProcessed(&_L1FeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1FeeVault.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultSession) Version() (string, error) {
	return _L1FeeVault.Contract.Version(&_L1FeeVault.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1FeeVault *L1FeeVaultCallerSession) Version() (string, error) {
	return _L1FeeVault.Contract.Version(&_L1FeeVault.CallOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultSession) Withdraw() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Withdraw(&_L1FeeVault.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_L1FeeVault *L1FeeVaultTransactorSession) Withdraw() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Withdraw(&_L1FeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1FeeVault.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultSession) Receive() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Receive(&_L1FeeVault.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1FeeVault *L1FeeVaultTransactorSession) Receive() (*types.Transaction, error) {
	return _L1FeeVault.Contract.Receive(&_L1FeeVault.TransactOpts)
}

// L1FeeVaultWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the L1FeeVault contract.
type L1FeeVaultWithdrawalIterator struct {
	Event *L1FeeVaultWithdrawal // Event containing the contract specifics and raw log

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
func (it *L1FeeVaultWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1FeeVaultWithdrawal)
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
		it.Event = new(L1FeeVaultWithdrawal)
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
func (it *L1FeeVaultWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1FeeVaultWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1FeeVaultWithdrawal represents a Withdrawal event raised by the L1FeeVault contract.
type L1FeeVaultWithdrawal struct {
	Value *big.Int
	To    common.Address
	From  common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_L1FeeVault *L1FeeVaultFilterer) FilterWithdrawal(opts *bind.FilterOpts) (*L1FeeVaultWithdrawalIterator, error) {

	logs, sub, err := _L1FeeVault.contract.FilterLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultWithdrawalIterator{contract: _L1FeeVault.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0xc8a211cc64b6ed1b50595a9fcb1932b6d1e5a6e8ef15b60e5b1f988ea9086bba.
//
// Solidity: event Withdrawal(uint256 value, address to, address from)
func (_L1FeeVault *L1FeeVaultFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *L1FeeVaultWithdrawal) (event.Subscription, error) {

	logs, sub, err := _L1FeeVault.contract.WatchLogs(opts, "Withdrawal")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1FeeVaultWithdrawal)
				if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
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
func (_L1FeeVault *L1FeeVaultFilterer) ParseWithdrawal(log types.Log) (*L1FeeVaultWithdrawal, error) {
	event := new(L1FeeVaultWithdrawal)
	if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1FeeVaultWithdrawal0Iterator is returned from FilterWithdrawal0 and is used to iterate over the raw logs and unpacked data for Withdrawal0 events raised by the L1FeeVault contract.
type L1FeeVaultWithdrawal0Iterator struct {
	Event *L1FeeVaultWithdrawal0 // Event containing the contract specifics and raw log

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
func (it *L1FeeVaultWithdrawal0Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1FeeVaultWithdrawal0)
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
		it.Event = new(L1FeeVaultWithdrawal0)
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
func (it *L1FeeVaultWithdrawal0Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1FeeVaultWithdrawal0Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1FeeVaultWithdrawal0 represents a Withdrawal0 event raised by the L1FeeVault contract.
type L1FeeVaultWithdrawal0 struct {
	Value             *big.Int
	To                common.Address
	From              common.Address
	WithdrawalNetwork uint8
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal0 is a free log retrieval operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_L1FeeVault *L1FeeVaultFilterer) FilterWithdrawal0(opts *bind.FilterOpts) (*L1FeeVaultWithdrawal0Iterator, error) {

	logs, sub, err := _L1FeeVault.contract.FilterLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return &L1FeeVaultWithdrawal0Iterator{contract: _L1FeeVault.contract, event: "Withdrawal0", logs: logs, sub: sub}, nil
}

// WatchWithdrawal0 is a free log subscription operation binding the contract event 0x38e04cbeb8c10f8f568618aa75be0f10b6729b8b4237743b4de20cbcde2839ee.
//
// Solidity: event Withdrawal(uint256 value, address to, address from, uint8 withdrawalNetwork)
func (_L1FeeVault *L1FeeVaultFilterer) WatchWithdrawal0(opts *bind.WatchOpts, sink chan<- *L1FeeVaultWithdrawal0) (event.Subscription, error) {

	logs, sub, err := _L1FeeVault.contract.WatchLogs(opts, "Withdrawal0")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1FeeVaultWithdrawal0)
				if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
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
func (_L1FeeVault *L1FeeVaultFilterer) ParseWithdrawal0(log types.Log) (*L1FeeVaultWithdrawal0, error) {
	event := new(L1FeeVaultWithdrawal0)
	if err := _L1FeeVault.contract.UnpackLog(event, "Withdrawal0", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
