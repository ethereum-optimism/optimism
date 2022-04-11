// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package deposit

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

// WithdrawalVerifierOutputRootProof is an auto generated low-level Go binding around an user-defined struct.
type WithdrawalVerifierOutputRootProof struct {
	Version               [32]byte
	StateRoot             [32]byte
	WithdrawerStorageRoot [32]byte
	LatestBlockhash       [32]byte
}

// AddressAliasHelperMetaData contains all meta data concerning the AddressAliasHelper contract.
var AddressAliasHelperMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220c7fa45f5503f62c2292b9c4a466d09aacd158cdc3685a7753531ac6bf8bda5ef64736f6c634300080a0033",
}

// AddressAliasHelperABI is the input ABI used to generate the binding from.
// Deprecated: Use AddressAliasHelperMetaData.ABI instead.
var AddressAliasHelperABI = AddressAliasHelperMetaData.ABI

// AddressAliasHelperBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AddressAliasHelperMetaData.Bin instead.
var AddressAliasHelperBin = AddressAliasHelperMetaData.Bin

// DeployAddressAliasHelper deploys a new Ethereum contract, binding an instance of AddressAliasHelper to it.
func DeployAddressAliasHelper(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *AddressAliasHelper, error) {
	parsed, err := AddressAliasHelperMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AddressAliasHelperBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AddressAliasHelper{AddressAliasHelperCaller: AddressAliasHelperCaller{contract: contract}, AddressAliasHelperTransactor: AddressAliasHelperTransactor{contract: contract}, AddressAliasHelperFilterer: AddressAliasHelperFilterer{contract: contract}}, nil
}

// AddressAliasHelper is an auto generated Go binding around an Ethereum contract.
type AddressAliasHelper struct {
	AddressAliasHelperCaller     // Read-only binding to the contract
	AddressAliasHelperTransactor // Write-only binding to the contract
	AddressAliasHelperFilterer   // Log filterer for contract events
}

// AddressAliasHelperCaller is an auto generated read-only Go binding around an Ethereum contract.
type AddressAliasHelperCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AddressAliasHelperTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AddressAliasHelperFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressAliasHelperSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AddressAliasHelperSession struct {
	Contract     *AddressAliasHelper // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// AddressAliasHelperCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AddressAliasHelperCallerSession struct {
	Contract *AddressAliasHelperCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// AddressAliasHelperTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AddressAliasHelperTransactorSession struct {
	Contract     *AddressAliasHelperTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// AddressAliasHelperRaw is an auto generated low-level Go binding around an Ethereum contract.
type AddressAliasHelperRaw struct {
	Contract *AddressAliasHelper // Generic contract binding to access the raw methods on
}

// AddressAliasHelperCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AddressAliasHelperCallerRaw struct {
	Contract *AddressAliasHelperCaller // Generic read-only contract binding to access the raw methods on
}

// AddressAliasHelperTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AddressAliasHelperTransactorRaw struct {
	Contract *AddressAliasHelperTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAddressAliasHelper creates a new instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelper(address common.Address, backend bind.ContractBackend) (*AddressAliasHelper, error) {
	contract, err := bindAddressAliasHelper(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelper{AddressAliasHelperCaller: AddressAliasHelperCaller{contract: contract}, AddressAliasHelperTransactor: AddressAliasHelperTransactor{contract: contract}, AddressAliasHelperFilterer: AddressAliasHelperFilterer{contract: contract}}, nil
}

// NewAddressAliasHelperCaller creates a new read-only instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperCaller(address common.Address, caller bind.ContractCaller) (*AddressAliasHelperCaller, error) {
	contract, err := bindAddressAliasHelper(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperCaller{contract: contract}, nil
}

// NewAddressAliasHelperTransactor creates a new write-only instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperTransactor(address common.Address, transactor bind.ContractTransactor) (*AddressAliasHelperTransactor, error) {
	contract, err := bindAddressAliasHelper(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperTransactor{contract: contract}, nil
}

// NewAddressAliasHelperFilterer creates a new log filterer instance of AddressAliasHelper, bound to a specific deployed contract.
func NewAddressAliasHelperFilterer(address common.Address, filterer bind.ContractFilterer) (*AddressAliasHelperFilterer, error) {
	contract, err := bindAddressAliasHelper(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AddressAliasHelperFilterer{contract: contract}, nil
}

// bindAddressAliasHelper binds a generic wrapper to an already deployed contract.
func bindAddressAliasHelper(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AddressAliasHelperABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressAliasHelper *AddressAliasHelperRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressAliasHelper.Contract.AddressAliasHelperCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressAliasHelper *AddressAliasHelperRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.AddressAliasHelperTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressAliasHelper *AddressAliasHelperRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.AddressAliasHelperTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressAliasHelper *AddressAliasHelperCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressAliasHelper.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressAliasHelper *AddressAliasHelperTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressAliasHelper *AddressAliasHelperTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressAliasHelper.Contract.contract.Transact(opts, method, params...)
}

// ContextMetaData contains all meta data concerning the Context contract.
var ContextMetaData = &bind.MetaData{
	ABI: "[]",
}

// ContextABI is the input ABI used to generate the binding from.
// Deprecated: Use ContextMetaData.ABI instead.
var ContextABI = ContextMetaData.ABI

// Context is an auto generated Go binding around an Ethereum contract.
type Context struct {
	ContextCaller     // Read-only binding to the contract
	ContextTransactor // Write-only binding to the contract
	ContextFilterer   // Log filterer for contract events
}

// ContextCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContextCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContextTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContextTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContextFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContextFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContextSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContextSession struct {
	Contract     *Context          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContextCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContextCallerSession struct {
	Contract *ContextCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ContextTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContextTransactorSession struct {
	Contract     *ContextTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ContextRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContextRaw struct {
	Contract *Context // Generic contract binding to access the raw methods on
}

// ContextCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContextCallerRaw struct {
	Contract *ContextCaller // Generic read-only contract binding to access the raw methods on
}

// ContextTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContextTransactorRaw struct {
	Contract *ContextTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContext creates a new instance of Context, bound to a specific deployed contract.
func NewContext(address common.Address, backend bind.ContractBackend) (*Context, error) {
	contract, err := bindContext(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Context{ContextCaller: ContextCaller{contract: contract}, ContextTransactor: ContextTransactor{contract: contract}, ContextFilterer: ContextFilterer{contract: contract}}, nil
}

// NewContextCaller creates a new read-only instance of Context, bound to a specific deployed contract.
func NewContextCaller(address common.Address, caller bind.ContractCaller) (*ContextCaller, error) {
	contract, err := bindContext(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContextCaller{contract: contract}, nil
}

// NewContextTransactor creates a new write-only instance of Context, bound to a specific deployed contract.
func NewContextTransactor(address common.Address, transactor bind.ContractTransactor) (*ContextTransactor, error) {
	contract, err := bindContext(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContextTransactor{contract: contract}, nil
}

// NewContextFilterer creates a new log filterer instance of Context, bound to a specific deployed contract.
func NewContextFilterer(address common.Address, filterer bind.ContractFilterer) (*ContextFilterer, error) {
	contract, err := bindContext(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContextFilterer{contract: contract}, nil
}

// bindContext binds a generic wrapper to an already deployed contract.
func bindContext(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContextABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Context *ContextRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Context.Contract.ContextCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Context *ContextRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Context.Contract.ContextTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Context *ContextRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Context.Contract.ContextTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Context *ContextCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Context.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Context *ContextTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Context.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Context *ContextTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Context.Contract.contract.Transact(opts, method, params...)
}

// DepositFeedMetaData contains all meta data concerning the DepositFeed contract.
var DepositFeedMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"NonZeroCreationTarget\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"mint\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isCreation\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
}

// DepositFeedABI is the input ABI used to generate the binding from.
// Deprecated: Use DepositFeedMetaData.ABI instead.
var DepositFeedABI = DepositFeedMetaData.ABI

// DepositFeed is an auto generated Go binding around an Ethereum contract.
type DepositFeed struct {
	DepositFeedCaller     // Read-only binding to the contract
	DepositFeedTransactor // Write-only binding to the contract
	DepositFeedFilterer   // Log filterer for contract events
}

// DepositFeedCaller is an auto generated read-only Go binding around an Ethereum contract.
type DepositFeedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositFeedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DepositFeedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositFeedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DepositFeedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DepositFeedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DepositFeedSession struct {
	Contract     *DepositFeed      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DepositFeedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DepositFeedCallerSession struct {
	Contract *DepositFeedCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// DepositFeedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DepositFeedTransactorSession struct {
	Contract     *DepositFeedTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// DepositFeedRaw is an auto generated low-level Go binding around an Ethereum contract.
type DepositFeedRaw struct {
	Contract *DepositFeed // Generic contract binding to access the raw methods on
}

// DepositFeedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DepositFeedCallerRaw struct {
	Contract *DepositFeedCaller // Generic read-only contract binding to access the raw methods on
}

// DepositFeedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DepositFeedTransactorRaw struct {
	Contract *DepositFeedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDepositFeed creates a new instance of DepositFeed, bound to a specific deployed contract.
func NewDepositFeed(address common.Address, backend bind.ContractBackend) (*DepositFeed, error) {
	contract, err := bindDepositFeed(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DepositFeed{DepositFeedCaller: DepositFeedCaller{contract: contract}, DepositFeedTransactor: DepositFeedTransactor{contract: contract}, DepositFeedFilterer: DepositFeedFilterer{contract: contract}}, nil
}

// NewDepositFeedCaller creates a new read-only instance of DepositFeed, bound to a specific deployed contract.
func NewDepositFeedCaller(address common.Address, caller bind.ContractCaller) (*DepositFeedCaller, error) {
	contract, err := bindDepositFeed(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DepositFeedCaller{contract: contract}, nil
}

// NewDepositFeedTransactor creates a new write-only instance of DepositFeed, bound to a specific deployed contract.
func NewDepositFeedTransactor(address common.Address, transactor bind.ContractTransactor) (*DepositFeedTransactor, error) {
	contract, err := bindDepositFeed(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DepositFeedTransactor{contract: contract}, nil
}

// NewDepositFeedFilterer creates a new log filterer instance of DepositFeed, bound to a specific deployed contract.
func NewDepositFeedFilterer(address common.Address, filterer bind.ContractFilterer) (*DepositFeedFilterer, error) {
	contract, err := bindDepositFeed(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DepositFeedFilterer{contract: contract}, nil
}

// bindDepositFeed binds a generic wrapper to an already deployed contract.
func bindDepositFeed(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DepositFeedABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DepositFeed *DepositFeedRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DepositFeed.Contract.DepositFeedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DepositFeed *DepositFeedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DepositFeed.Contract.DepositFeedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DepositFeed *DepositFeedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DepositFeed.Contract.DepositFeedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DepositFeed *DepositFeedCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DepositFeed.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DepositFeed *DepositFeedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DepositFeed.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DepositFeed *DepositFeedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DepositFeed.Contract.contract.Transact(opts, method, params...)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_DepositFeed *DepositFeedTransactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _DepositFeed.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_DepositFeed *DepositFeedSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _DepositFeed.Contract.DepositTransaction(&_DepositFeed.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_DepositFeed *DepositFeedTransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _DepositFeed.Contract.DepositTransaction(&_DepositFeed.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositFeedTransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the DepositFeed contract.
type DepositFeedTransactionDepositedIterator struct {
	Event *DepositFeedTransactionDeposited // Event containing the contract specifics and raw log

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
func (it *DepositFeedTransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DepositFeedTransactionDeposited)
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
		it.Event = new(DepositFeedTransactionDeposited)
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
func (it *DepositFeedTransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DepositFeedTransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DepositFeedTransactionDeposited represents a TransactionDeposited event raised by the DepositFeed contract.
type DepositFeedTransactionDeposited struct {
	From       common.Address
	To         common.Address
	Mint       *big.Int
	Value      *big.Int
	GasLimit   *big.Int
	IsCreation bool
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_DepositFeed *DepositFeedFilterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*DepositFeedTransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _DepositFeed.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &DepositFeedTransactionDepositedIterator{contract: _DepositFeed.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_DepositFeed *DepositFeedFilterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *DepositFeedTransactionDeposited, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _DepositFeed.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DepositFeedTransactionDeposited)
				if err := _DepositFeed.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
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

// ParseTransactionDeposited is a log parse operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_DepositFeed *DepositFeedFilterer) ParseTransactionDeposited(log types.Log) (*DepositFeedTransactionDeposited, error) {
	event := new(DepositFeedTransactionDeposited)
	if err := _DepositFeed.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sequencer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"}],\"name\":\"l2OutputAppended\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1Blocknumber\",\"type\":\"uint256\"}],\"name\":\"appendL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2timestamp\",\"type\":\"uint256\"}],\"name\":\"computeL2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2Timestamp\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"historicalTotalBlocks\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startingBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"submissionInterval\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6101006040523480156200001257600080fd5b50604051620012f2380380620012f2833981810160405281019062000038919062000260565b620000586200004c620000b460201b60201c565b620000bc60201b60201c565b85608081815250508460a081815250508360026000848152602001908152602001600020819055508260c08181525050816001819055508160e08181525050620000a881620000bc60201b60201c565b505050505050620002fc565b600033905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600080fd5b6000819050919050565b6200019a8162000185565b8114620001a657600080fd5b50565b600081519050620001ba816200018f565b92915050565b6000819050919050565b620001d581620001c0565b8114620001e157600080fd5b50565b600081519050620001f581620001ca565b92915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006200022882620001fb565b9050919050565b6200023a816200021b565b81146200024657600080fd5b50565b6000815190506200025a816200022f565b92915050565b60008060008060008060c0878903121562000280576200027f62000180565b5b60006200029089828a01620001a9565b9650506020620002a389828a01620001a9565b9550506040620002b689828a01620001e4565b9450506060620002c989828a01620001a9565b9350506080620002dc89828a01620001a9565b92505060a0620002ef89828a0162000249565b9150509295509295509295565b60805160a05160c05160e051610f9962000359600039600081816102b30152818161033701526106b001526000818161036b01526106d4015260008181610316015261066f01526000818161058b01526106f80152610f996000f3fe6080604052600436106100a75760003560e01c806393991af31161006457806393991af31461019d578063a25ae557146101c8578063c5095d6814610205578063c90ec2da14610230578063e1a41bcf1461025b578063f2fde38b14610286576100a7565b806302e51345146100ac5780630c1952d3146100e95780632518810414610114578063357e951f14610130578063715018a61461015b5780638da5cb5b14610172575b600080fd5b3480156100b857600080fd5b506100d360048036038101906100ce9190610919565b6102af565b6040516100e09190610955565b60405180910390f35b3480156100f557600080fd5b506100fe610393565b60405161010b9190610955565b60405180910390f35b61012e600480360381019061012991906109a6565b610399565b005b34801561013c57600080fd5b50610145610587565b6040516101529190610955565b60405180910390f35b34801561016757600080fd5b506101706105bc565b005b34801561017e57600080fd5b50610187610644565b6040516101949190610a4e565b60405180910390f35b3480156101a957600080fd5b506101b261066d565b6040516101bf9190610955565b60405180910390f35b3480156101d457600080fd5b506101ef60048036038101906101ea9190610919565b610691565b6040516101fc9190610a78565b60405180910390f35b34801561021157600080fd5b5061021a6106ae565b6040516102279190610955565b60405180910390f35b34801561023c57600080fd5b506102456106d2565b6040516102529190610955565b60405180910390f35b34801561026757600080fd5b506102706106f6565b60405161027d9190610955565b60405180910390f35b34801561029257600080fd5b506102ad60048036038101906102a89190610abf565b61071a565b005b60007f0000000000000000000000000000000000000000000000000000000000000000821015610314576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161030b90610b6f565b60405180910390fd5b7f00000000000000000000000000000000000000000000000000000000000000007f000000000000000000000000000000000000000000000000000000000000000083038161036657610365610b8f565b5b0460017f000000000000000000000000000000000000000000000000000000000000000001019050919050565b60015481565b6103a1610812565b73ffffffffffffffffffffffffffffffffffffffff166103bf610644565b73ffffffffffffffffffffffffffffffffffffffff1614610415576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161040c90610c0a565b60405180910390fd5b428310610457576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161044e90610c9c565b60405180910390fd5b61045f610587565b83146104a0576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161049790610d2e565b60405180910390fd5b6000801b8414156104e6576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016104dd90610d9a565b60405180910390fd5b6000801b82146105345781814014610533576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161052a90610e2c565b60405180910390fd5b5b8360026000858152602001908152602001600020819055508260018190555082847f92701dc658a5d84c16077ea6de344b9995e21a96a05d45e4cd22f37a3d266f8b60405160405180910390a350505050565b60007f00000000000000000000000000000000000000000000000000000000000000006001546105b79190610e7b565b905090565b6105c4610812565b73ffffffffffffffffffffffffffffffffffffffff166105e2610644565b73ffffffffffffffffffffffffffffffffffffffff1614610638576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161062f90610c0a565b60405180910390fd5b610642600061081a565b565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b7f000000000000000000000000000000000000000000000000000000000000000081565b600060026000838152602001908152602001600020549050919050565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000081565b610722610812565b73ffffffffffffffffffffffffffffffffffffffff16610740610644565b73ffffffffffffffffffffffffffffffffffffffff1614610796576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161078d90610c0a565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610806576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107fd90610f43565b60405180910390fd5b61080f8161081a565b50565b600033905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600080fd5b6000819050919050565b6108f6816108e3565b811461090157600080fd5b50565b600081359050610913816108ed565b92915050565b60006020828403121561092f5761092e6108de565b5b600061093d84828501610904565b91505092915050565b61094f816108e3565b82525050565b600060208201905061096a6000830184610946565b92915050565b6000819050919050565b61098381610970565b811461098e57600080fd5b50565b6000813590506109a08161097a565b92915050565b600080600080608085870312156109c0576109bf6108de565b5b60006109ce87828801610991565b94505060206109df87828801610904565b93505060406109f087828801610991565b9250506060610a0187828801610904565b91505092959194509250565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610a3882610a0d565b9050919050565b610a4881610a2d565b82525050565b6000602082019050610a636000830184610a3f565b92915050565b610a7281610970565b82525050565b6000602082019050610a8d6000830184610a69565b92915050565b610a9c81610a2d565b8114610aa757600080fd5b50565b600081359050610ab981610a93565b92915050565b600060208284031215610ad557610ad46108de565b5b6000610ae384828501610aaa565b91505092915050565b600082825260208201905092915050565b7f54696d657374616d70207072696f7220746f207374617274696e67426c6f636b60008201527f54696d657374616d700000000000000000000000000000000000000000000000602082015250565b6000610b59602983610aec565b9150610b6482610afd565b604082019050919050565b60006020820190508181036000830152610b8881610b4c565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b7f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572600082015250565b6000610bf4602083610aec565b9150610bff82610bbe565b602082019050919050565b60006020820190508181036000830152610c2381610be7565b9050919050565b7f43616e6e6f7420617070656e64204c32206f757470757420696e20667574757260008201527f6500000000000000000000000000000000000000000000000000000000000000602082015250565b6000610c86602183610aec565b9150610c9182610c2a565b604082019050919050565b60006020820190508181036000830152610cb581610c79565b9050919050565b7f54696d657374616d70206e6f7420657175616c20746f206e657874206578706560008201527f637465642074696d657374616d70000000000000000000000000000000000000602082015250565b6000610d18602e83610aec565b9150610d2382610cbc565b604082019050919050565b60006020820190508181036000830152610d4781610d0b565b9050919050565b7f43616e6e6f74207375626d697420656d707479204c32206f7574707574000000600082015250565b6000610d84601d83610aec565b9150610d8f82610d4e565b602082019050919050565b60006020820190508181036000830152610db381610d77565b9050919050565b7f426c6f636b6861736820646f6573206e6f74206d61746368207468652068617360008201527f6820617420746865206578706563746564206865696768742e00000000000000602082015250565b6000610e16603983610aec565b9150610e2182610dba565b604082019050919050565b60006020820190508181036000830152610e4581610e09565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610e86826108e3565b9150610e91836108e3565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff03821115610ec657610ec5610e4c565b5b828201905092915050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160008201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b6000610f2d602683610aec565b9150610f3882610ed1565b604082019050919050565b60006020820190508181036000830152610f5c81610f20565b905091905056fea2646970667358221220980520f45819a707ff13a09b23bdace7cce468e4b8e92e7fd2f480d56c85cb7664736f6c634300080a0033",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _l2BlockTime *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int, _startingBlockTimestamp *big.Int, sequencer common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _l2BlockTime, _genesisL2Output, _historicalTotalBlocks, _startingBlockTimestamp, sequencer)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// L2OutputOracle is an auto generated Go binding around an Ethereum contract.
type L2OutputOracle struct {
	L2OutputOracleCaller     // Read-only binding to the contract
	L2OutputOracleTransactor // Write-only binding to the contract
	L2OutputOracleFilterer   // Log filterer for contract events
}

// L2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2OutputOracleSession struct {
	Contract     *L2OutputOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2OutputOracleCallerSession struct {
	Contract *L2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2OutputOracleTransactorSession struct {
	Contract     *L2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2OutputOracleRaw struct {
	Contract *L2OutputOracle // Generic contract binding to access the raw methods on
}

// L2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2OutputOracleCallerRaw struct {
	Contract *L2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// L2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactorRaw struct {
	Contract *L2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2OutputOracle creates a new instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracle(address common.Address, backend bind.ContractBackend) (*L2OutputOracle, error) {
	contract, err := bindL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// NewL2OutputOracleCaller creates a new read-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*L2OutputOracleCaller, error) {
	contract, err := bindL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleCaller{contract: contract}, nil
}

// NewL2OutputOracleTransactor creates a new write-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2OutputOracleTransactor, error) {
	contract, err := bindL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleTransactor{contract: contract}, nil
}

// NewL2OutputOracleFilterer creates a new log filterer instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*L2OutputOracleFilterer, error) {
	contract, err := bindL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleFilterer{contract: contract}, nil
}

// bindL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.L2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) ComputeL2BlockNumber(opts *bind.CallOpts, _l2timestamp *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "computeL2BlockNumber", _l2timestamp)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) ComputeL2BlockNumber(_l2timestamp *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2BlockNumber(&_L2OutputOracle.CallOpts, _l2timestamp)
}

// ComputeL2BlockNumber is a free data retrieval call binding the contract method 0x02e51345.
//
// Solidity: function computeL2BlockNumber(uint256 _l2timestamp) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) ComputeL2BlockNumber(_l2timestamp *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2BlockNumber(&_L2OutputOracle.CallOpts, _l2timestamp)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns(bytes32)
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2Timestamp *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2Timestamp)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns(bytes32)
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2Timestamp *big.Int) ([32]byte, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2Timestamp)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2Timestamp) view returns(bytes32)
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2Timestamp *big.Int) ([32]byte, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2Timestamp)
}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) HistoricalTotalBlocks(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "historicalTotalBlocks")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) HistoricalTotalBlocks() (*big.Int, error) {
	return _L2OutputOracle.Contract.HistoricalTotalBlocks(&_L2OutputOracle.CallOpts)
}

// HistoricalTotalBlocks is a free data retrieval call binding the contract method 0xc90ec2da.
//
// Solidity: function historicalTotalBlocks() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) HistoricalTotalBlocks() (*big.Int, error) {
	return _L2OutputOracle.Contract.HistoricalTotalBlocks(&_L2OutputOracle.CallOpts)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) L2BlockTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "l2BlockTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) L2BlockTime() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BlockTime(&_L2OutputOracle.CallOpts)
}

// L2BlockTime is a free data retrieval call binding the contract method 0x93991af3.
//
// Solidity: function l2BlockTime() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2BlockTime() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BlockTime(&_L2OutputOracle.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// LatestBlockTimestamp is a free data retrieval call binding the contract method 0x0c1952d3.
//
// Solidity: function latestBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextTimestamp(&_L2OutputOracle.CallOpts)
}

// NextTimestamp is a free data retrieval call binding the contract method 0x357e951f.
//
// Solidity: function nextTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextTimestamp(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) StartingBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "startingBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) StartingBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// StartingBlockTimestamp is a free data retrieval call binding the contract method 0xc5095d68.
//
// Solidity: function startingBlockTimestamp() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) StartingBlockTimestamp() (*big.Int, error) {
	return _L2OutputOracle.Contract.StartingBlockTimestamp(&_L2OutputOracle.CallOpts)
}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SubmissionInterval(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "submissionInterval")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SubmissionInterval() (*big.Int, error) {
	return _L2OutputOracle.Contract.SubmissionInterval(&_L2OutputOracle.CallOpts)
}

// SubmissionInterval is a free data retrieval call binding the contract method 0xe1a41bcf.
//
// Solidity: function submissionInterval() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SubmissionInterval() (*big.Int, error) {
	return _L2OutputOracle.Contract.SubmissionInterval(&_L2OutputOracle.CallOpts)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) AppendL2Output(opts *bind.TransactOpts, _l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "appendL2Output", _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) AppendL2Output(_l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) AppendL2Output(_l2Output [32]byte, _l2timestamp *big.Int, _l1Blockhash [32]byte, _l1Blocknumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2timestamp, _l1Blockhash, _l1Blocknumber)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// L2OutputOracleOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferredIterator struct {
	Event *L2OutputOracleOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOwnershipTransferred)
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
		it.Event = new(L2OutputOracleOwnershipTransferred)
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
func (it *L2OutputOracleOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L2OutputOracleOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOwnershipTransferredIterator{contract: _L2OutputOracle.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOwnershipTransferred)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOwnershipTransferred(log types.Log) (*L2OutputOracleOwnershipTransferred, error) {
	event := new(L2OutputOracleOwnershipTransferred)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputAppendedIterator is returned from FilterL2OutputAppended and is used to iterate over the raw logs and unpacked data for L2OutputAppended events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppendedIterator struct {
	Event *L2OutputOracleL2OutputAppended // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputAppended)
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
		it.Event = new(L2OutputOracleL2OutputAppended)
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
func (it *L2OutputOracleL2OutputAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputAppended represents a L2OutputAppended event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppended struct {
	L2Output    [32]byte
	L2timestamp *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterL2OutputAppended is a free log retrieval operation binding the contract event 0x92701dc658a5d84c16077ea6de344b9995e21a96a05d45e4cd22f37a3d266f8b.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputAppended(opts *bind.FilterOpts, _l2Output [][32]byte, _l2timestamp []*big.Int) (*L2OutputOracleL2OutputAppendedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "l2OutputAppended", _l2OutputRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputAppendedIterator{contract: _L2OutputOracle.contract, event: "l2OutputAppended", logs: logs, sub: sub}, nil
}

// WatchL2OutputAppended is a free log subscription operation binding the contract event 0x92701dc658a5d84c16077ea6de344b9995e21a96a05d45e4cd22f37a3d266f8b.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputAppended(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputAppended, _l2Output [][32]byte, _l2timestamp []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l2timestampRule []interface{}
	for _, _l2timestampItem := range _l2timestamp {
		_l2timestampRule = append(_l2timestampRule, _l2timestampItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "l2OutputAppended", _l2OutputRule, _l2timestampRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputAppended)
				if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
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

// ParseL2OutputAppended is a log parse operation binding the contract event 0x92701dc658a5d84c16077ea6de344b9995e21a96a05d45e4cd22f37a3d266f8b.
//
// Solidity: event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputAppended(log types.Log) (*L2OutputOracleL2OutputAppended, error) {
	event := new(L2OutputOracleL2OutputAppended)
	if err := _L2OutputOracle.contract.UnpackLog(event, "l2OutputAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LibBytesUtilsMetaData contains all meta data concerning the LibBytesUtils contract.
var LibBytesUtilsMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220e958e3c5fd0868451927cf89f50e248831442cc24af29e21bca4bc8a02ffeb9764736f6c634300080a0033",
}

// LibBytesUtilsABI is the input ABI used to generate the binding from.
// Deprecated: Use LibBytesUtilsMetaData.ABI instead.
var LibBytesUtilsABI = LibBytesUtilsMetaData.ABI

// LibBytesUtilsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LibBytesUtilsMetaData.Bin instead.
var LibBytesUtilsBin = LibBytesUtilsMetaData.Bin

// DeployLibBytesUtils deploys a new Ethereum contract, binding an instance of LibBytesUtils to it.
func DeployLibBytesUtils(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LibBytesUtils, error) {
	parsed, err := LibBytesUtilsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LibBytesUtilsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LibBytesUtils{LibBytesUtilsCaller: LibBytesUtilsCaller{contract: contract}, LibBytesUtilsTransactor: LibBytesUtilsTransactor{contract: contract}, LibBytesUtilsFilterer: LibBytesUtilsFilterer{contract: contract}}, nil
}

// LibBytesUtils is an auto generated Go binding around an Ethereum contract.
type LibBytesUtils struct {
	LibBytesUtilsCaller     // Read-only binding to the contract
	LibBytesUtilsTransactor // Write-only binding to the contract
	LibBytesUtilsFilterer   // Log filterer for contract events
}

// LibBytesUtilsCaller is an auto generated read-only Go binding around an Ethereum contract.
type LibBytesUtilsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibBytesUtilsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LibBytesUtilsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibBytesUtilsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LibBytesUtilsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibBytesUtilsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LibBytesUtilsSession struct {
	Contract     *LibBytesUtils    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LibBytesUtilsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LibBytesUtilsCallerSession struct {
	Contract *LibBytesUtilsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// LibBytesUtilsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LibBytesUtilsTransactorSession struct {
	Contract     *LibBytesUtilsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// LibBytesUtilsRaw is an auto generated low-level Go binding around an Ethereum contract.
type LibBytesUtilsRaw struct {
	Contract *LibBytesUtils // Generic contract binding to access the raw methods on
}

// LibBytesUtilsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LibBytesUtilsCallerRaw struct {
	Contract *LibBytesUtilsCaller // Generic read-only contract binding to access the raw methods on
}

// LibBytesUtilsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LibBytesUtilsTransactorRaw struct {
	Contract *LibBytesUtilsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLibBytesUtils creates a new instance of LibBytesUtils, bound to a specific deployed contract.
func NewLibBytesUtils(address common.Address, backend bind.ContractBackend) (*LibBytesUtils, error) {
	contract, err := bindLibBytesUtils(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LibBytesUtils{LibBytesUtilsCaller: LibBytesUtilsCaller{contract: contract}, LibBytesUtilsTransactor: LibBytesUtilsTransactor{contract: contract}, LibBytesUtilsFilterer: LibBytesUtilsFilterer{contract: contract}}, nil
}

// NewLibBytesUtilsCaller creates a new read-only instance of LibBytesUtils, bound to a specific deployed contract.
func NewLibBytesUtilsCaller(address common.Address, caller bind.ContractCaller) (*LibBytesUtilsCaller, error) {
	contract, err := bindLibBytesUtils(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LibBytesUtilsCaller{contract: contract}, nil
}

// NewLibBytesUtilsTransactor creates a new write-only instance of LibBytesUtils, bound to a specific deployed contract.
func NewLibBytesUtilsTransactor(address common.Address, transactor bind.ContractTransactor) (*LibBytesUtilsTransactor, error) {
	contract, err := bindLibBytesUtils(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LibBytesUtilsTransactor{contract: contract}, nil
}

// NewLibBytesUtilsFilterer creates a new log filterer instance of LibBytesUtils, bound to a specific deployed contract.
func NewLibBytesUtilsFilterer(address common.Address, filterer bind.ContractFilterer) (*LibBytesUtilsFilterer, error) {
	contract, err := bindLibBytesUtils(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LibBytesUtilsFilterer{contract: contract}, nil
}

// bindLibBytesUtils binds a generic wrapper to an already deployed contract.
func bindLibBytesUtils(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LibBytesUtilsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibBytesUtils *LibBytesUtilsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibBytesUtils.Contract.LibBytesUtilsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibBytesUtils *LibBytesUtilsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibBytesUtils.Contract.LibBytesUtilsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibBytesUtils *LibBytesUtilsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibBytesUtils.Contract.LibBytesUtilsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibBytesUtils *LibBytesUtilsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibBytesUtils.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibBytesUtils *LibBytesUtilsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibBytesUtils.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibBytesUtils *LibBytesUtilsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibBytesUtils.Contract.contract.Transact(opts, method, params...)
}

// LibMerkleTrieMetaData contains all meta data concerning the LibMerkleTrie contract.
var LibMerkleTrieMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212204e5467228078a9262e6dfded9555ccabe760347ff5c34b33bdd846b1084e54b764736f6c634300080a0033",
}

// LibMerkleTrieABI is the input ABI used to generate the binding from.
// Deprecated: Use LibMerkleTrieMetaData.ABI instead.
var LibMerkleTrieABI = LibMerkleTrieMetaData.ABI

// LibMerkleTrieBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LibMerkleTrieMetaData.Bin instead.
var LibMerkleTrieBin = LibMerkleTrieMetaData.Bin

// DeployLibMerkleTrie deploys a new Ethereum contract, binding an instance of LibMerkleTrie to it.
func DeployLibMerkleTrie(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LibMerkleTrie, error) {
	parsed, err := LibMerkleTrieMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LibMerkleTrieBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LibMerkleTrie{LibMerkleTrieCaller: LibMerkleTrieCaller{contract: contract}, LibMerkleTrieTransactor: LibMerkleTrieTransactor{contract: contract}, LibMerkleTrieFilterer: LibMerkleTrieFilterer{contract: contract}}, nil
}

// LibMerkleTrie is an auto generated Go binding around an Ethereum contract.
type LibMerkleTrie struct {
	LibMerkleTrieCaller     // Read-only binding to the contract
	LibMerkleTrieTransactor // Write-only binding to the contract
	LibMerkleTrieFilterer   // Log filterer for contract events
}

// LibMerkleTrieCaller is an auto generated read-only Go binding around an Ethereum contract.
type LibMerkleTrieCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibMerkleTrieTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LibMerkleTrieTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibMerkleTrieFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LibMerkleTrieFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibMerkleTrieSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LibMerkleTrieSession struct {
	Contract     *LibMerkleTrie    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LibMerkleTrieCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LibMerkleTrieCallerSession struct {
	Contract *LibMerkleTrieCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// LibMerkleTrieTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LibMerkleTrieTransactorSession struct {
	Contract     *LibMerkleTrieTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// LibMerkleTrieRaw is an auto generated low-level Go binding around an Ethereum contract.
type LibMerkleTrieRaw struct {
	Contract *LibMerkleTrie // Generic contract binding to access the raw methods on
}

// LibMerkleTrieCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LibMerkleTrieCallerRaw struct {
	Contract *LibMerkleTrieCaller // Generic read-only contract binding to access the raw methods on
}

// LibMerkleTrieTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LibMerkleTrieTransactorRaw struct {
	Contract *LibMerkleTrieTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLibMerkleTrie creates a new instance of LibMerkleTrie, bound to a specific deployed contract.
func NewLibMerkleTrie(address common.Address, backend bind.ContractBackend) (*LibMerkleTrie, error) {
	contract, err := bindLibMerkleTrie(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LibMerkleTrie{LibMerkleTrieCaller: LibMerkleTrieCaller{contract: contract}, LibMerkleTrieTransactor: LibMerkleTrieTransactor{contract: contract}, LibMerkleTrieFilterer: LibMerkleTrieFilterer{contract: contract}}, nil
}

// NewLibMerkleTrieCaller creates a new read-only instance of LibMerkleTrie, bound to a specific deployed contract.
func NewLibMerkleTrieCaller(address common.Address, caller bind.ContractCaller) (*LibMerkleTrieCaller, error) {
	contract, err := bindLibMerkleTrie(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LibMerkleTrieCaller{contract: contract}, nil
}

// NewLibMerkleTrieTransactor creates a new write-only instance of LibMerkleTrie, bound to a specific deployed contract.
func NewLibMerkleTrieTransactor(address common.Address, transactor bind.ContractTransactor) (*LibMerkleTrieTransactor, error) {
	contract, err := bindLibMerkleTrie(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LibMerkleTrieTransactor{contract: contract}, nil
}

// NewLibMerkleTrieFilterer creates a new log filterer instance of LibMerkleTrie, bound to a specific deployed contract.
func NewLibMerkleTrieFilterer(address common.Address, filterer bind.ContractFilterer) (*LibMerkleTrieFilterer, error) {
	contract, err := bindLibMerkleTrie(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LibMerkleTrieFilterer{contract: contract}, nil
}

// bindLibMerkleTrie binds a generic wrapper to an already deployed contract.
func bindLibMerkleTrie(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LibMerkleTrieABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibMerkleTrie *LibMerkleTrieRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibMerkleTrie.Contract.LibMerkleTrieCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibMerkleTrie *LibMerkleTrieRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibMerkleTrie.Contract.LibMerkleTrieTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibMerkleTrie *LibMerkleTrieRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibMerkleTrie.Contract.LibMerkleTrieTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibMerkleTrie *LibMerkleTrieCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibMerkleTrie.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibMerkleTrie *LibMerkleTrieTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibMerkleTrie.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibMerkleTrie *LibMerkleTrieTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibMerkleTrie.Contract.contract.Transact(opts, method, params...)
}

// LibRLPReaderMetaData contains all meta data concerning the LibRLPReader contract.
var LibRLPReaderMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220e5de027b2466f0cb69615d578b02779f2ad9c727a9dd3b17b8e9279e3334f80864736f6c634300080a0033",
}

// LibRLPReaderABI is the input ABI used to generate the binding from.
// Deprecated: Use LibRLPReaderMetaData.ABI instead.
var LibRLPReaderABI = LibRLPReaderMetaData.ABI

// LibRLPReaderBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LibRLPReaderMetaData.Bin instead.
var LibRLPReaderBin = LibRLPReaderMetaData.Bin

// DeployLibRLPReader deploys a new Ethereum contract, binding an instance of LibRLPReader to it.
func DeployLibRLPReader(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LibRLPReader, error) {
	parsed, err := LibRLPReaderMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LibRLPReaderBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LibRLPReader{LibRLPReaderCaller: LibRLPReaderCaller{contract: contract}, LibRLPReaderTransactor: LibRLPReaderTransactor{contract: contract}, LibRLPReaderFilterer: LibRLPReaderFilterer{contract: contract}}, nil
}

// LibRLPReader is an auto generated Go binding around an Ethereum contract.
type LibRLPReader struct {
	LibRLPReaderCaller     // Read-only binding to the contract
	LibRLPReaderTransactor // Write-only binding to the contract
	LibRLPReaderFilterer   // Log filterer for contract events
}

// LibRLPReaderCaller is an auto generated read-only Go binding around an Ethereum contract.
type LibRLPReaderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPReaderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LibRLPReaderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPReaderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LibRLPReaderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPReaderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LibRLPReaderSession struct {
	Contract     *LibRLPReader     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LibRLPReaderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LibRLPReaderCallerSession struct {
	Contract *LibRLPReaderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// LibRLPReaderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LibRLPReaderTransactorSession struct {
	Contract     *LibRLPReaderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// LibRLPReaderRaw is an auto generated low-level Go binding around an Ethereum contract.
type LibRLPReaderRaw struct {
	Contract *LibRLPReader // Generic contract binding to access the raw methods on
}

// LibRLPReaderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LibRLPReaderCallerRaw struct {
	Contract *LibRLPReaderCaller // Generic read-only contract binding to access the raw methods on
}

// LibRLPReaderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LibRLPReaderTransactorRaw struct {
	Contract *LibRLPReaderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLibRLPReader creates a new instance of LibRLPReader, bound to a specific deployed contract.
func NewLibRLPReader(address common.Address, backend bind.ContractBackend) (*LibRLPReader, error) {
	contract, err := bindLibRLPReader(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LibRLPReader{LibRLPReaderCaller: LibRLPReaderCaller{contract: contract}, LibRLPReaderTransactor: LibRLPReaderTransactor{contract: contract}, LibRLPReaderFilterer: LibRLPReaderFilterer{contract: contract}}, nil
}

// NewLibRLPReaderCaller creates a new read-only instance of LibRLPReader, bound to a specific deployed contract.
func NewLibRLPReaderCaller(address common.Address, caller bind.ContractCaller) (*LibRLPReaderCaller, error) {
	contract, err := bindLibRLPReader(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LibRLPReaderCaller{contract: contract}, nil
}

// NewLibRLPReaderTransactor creates a new write-only instance of LibRLPReader, bound to a specific deployed contract.
func NewLibRLPReaderTransactor(address common.Address, transactor bind.ContractTransactor) (*LibRLPReaderTransactor, error) {
	contract, err := bindLibRLPReader(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LibRLPReaderTransactor{contract: contract}, nil
}

// NewLibRLPReaderFilterer creates a new log filterer instance of LibRLPReader, bound to a specific deployed contract.
func NewLibRLPReaderFilterer(address common.Address, filterer bind.ContractFilterer) (*LibRLPReaderFilterer, error) {
	contract, err := bindLibRLPReader(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LibRLPReaderFilterer{contract: contract}, nil
}

// bindLibRLPReader binds a generic wrapper to an already deployed contract.
func bindLibRLPReader(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LibRLPReaderABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibRLPReader *LibRLPReaderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibRLPReader.Contract.LibRLPReaderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibRLPReader *LibRLPReaderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibRLPReader.Contract.LibRLPReaderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibRLPReader *LibRLPReaderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibRLPReader.Contract.LibRLPReaderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibRLPReader *LibRLPReaderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibRLPReader.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibRLPReader *LibRLPReaderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibRLPReader.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibRLPReader *LibRLPReaderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibRLPReader.Contract.contract.Transact(opts, method, params...)
}

// LibRLPWriterMetaData contains all meta data concerning the LibRLPWriter contract.
var LibRLPWriterMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea2646970667358221220bef8e644bef83a13fe830ce5a62f8dd508a8ed03087ab7d1f194c2164c3d8cfe64736f6c634300080a0033",
}

// LibRLPWriterABI is the input ABI used to generate the binding from.
// Deprecated: Use LibRLPWriterMetaData.ABI instead.
var LibRLPWriterABI = LibRLPWriterMetaData.ABI

// LibRLPWriterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LibRLPWriterMetaData.Bin instead.
var LibRLPWriterBin = LibRLPWriterMetaData.Bin

// DeployLibRLPWriter deploys a new Ethereum contract, binding an instance of LibRLPWriter to it.
func DeployLibRLPWriter(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LibRLPWriter, error) {
	parsed, err := LibRLPWriterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LibRLPWriterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LibRLPWriter{LibRLPWriterCaller: LibRLPWriterCaller{contract: contract}, LibRLPWriterTransactor: LibRLPWriterTransactor{contract: contract}, LibRLPWriterFilterer: LibRLPWriterFilterer{contract: contract}}, nil
}

// LibRLPWriter is an auto generated Go binding around an Ethereum contract.
type LibRLPWriter struct {
	LibRLPWriterCaller     // Read-only binding to the contract
	LibRLPWriterTransactor // Write-only binding to the contract
	LibRLPWriterFilterer   // Log filterer for contract events
}

// LibRLPWriterCaller is an auto generated read-only Go binding around an Ethereum contract.
type LibRLPWriterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPWriterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LibRLPWriterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPWriterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LibRLPWriterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibRLPWriterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LibRLPWriterSession struct {
	Contract     *LibRLPWriter     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LibRLPWriterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LibRLPWriterCallerSession struct {
	Contract *LibRLPWriterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// LibRLPWriterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LibRLPWriterTransactorSession struct {
	Contract     *LibRLPWriterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// LibRLPWriterRaw is an auto generated low-level Go binding around an Ethereum contract.
type LibRLPWriterRaw struct {
	Contract *LibRLPWriter // Generic contract binding to access the raw methods on
}

// LibRLPWriterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LibRLPWriterCallerRaw struct {
	Contract *LibRLPWriterCaller // Generic read-only contract binding to access the raw methods on
}

// LibRLPWriterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LibRLPWriterTransactorRaw struct {
	Contract *LibRLPWriterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLibRLPWriter creates a new instance of LibRLPWriter, bound to a specific deployed contract.
func NewLibRLPWriter(address common.Address, backend bind.ContractBackend) (*LibRLPWriter, error) {
	contract, err := bindLibRLPWriter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LibRLPWriter{LibRLPWriterCaller: LibRLPWriterCaller{contract: contract}, LibRLPWriterTransactor: LibRLPWriterTransactor{contract: contract}, LibRLPWriterFilterer: LibRLPWriterFilterer{contract: contract}}, nil
}

// NewLibRLPWriterCaller creates a new read-only instance of LibRLPWriter, bound to a specific deployed contract.
func NewLibRLPWriterCaller(address common.Address, caller bind.ContractCaller) (*LibRLPWriterCaller, error) {
	contract, err := bindLibRLPWriter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LibRLPWriterCaller{contract: contract}, nil
}

// NewLibRLPWriterTransactor creates a new write-only instance of LibRLPWriter, bound to a specific deployed contract.
func NewLibRLPWriterTransactor(address common.Address, transactor bind.ContractTransactor) (*LibRLPWriterTransactor, error) {
	contract, err := bindLibRLPWriter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LibRLPWriterTransactor{contract: contract}, nil
}

// NewLibRLPWriterFilterer creates a new log filterer instance of LibRLPWriter, bound to a specific deployed contract.
func NewLibRLPWriterFilterer(address common.Address, filterer bind.ContractFilterer) (*LibRLPWriterFilterer, error) {
	contract, err := bindLibRLPWriter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LibRLPWriterFilterer{contract: contract}, nil
}

// bindLibRLPWriter binds a generic wrapper to an already deployed contract.
func bindLibRLPWriter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LibRLPWriterABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibRLPWriter *LibRLPWriterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibRLPWriter.Contract.LibRLPWriterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibRLPWriter *LibRLPWriterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibRLPWriter.Contract.LibRLPWriterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibRLPWriter *LibRLPWriterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibRLPWriter.Contract.LibRLPWriterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibRLPWriter *LibRLPWriterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibRLPWriter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibRLPWriter *LibRLPWriterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibRLPWriter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibRLPWriter *LibRLPWriterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibRLPWriter.Contract.contract.Transact(opts, method, params...)
}

// LibSecureMerkleTrieMetaData contains all meta data concerning the LibSecureMerkleTrie contract.
var LibSecureMerkleTrieMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212202826b55c1fd499dcab01cf9cefa4f87b2c8d925e6014f9f077213f8dbc479ba964736f6c634300080a0033",
}

// LibSecureMerkleTrieABI is the input ABI used to generate the binding from.
// Deprecated: Use LibSecureMerkleTrieMetaData.ABI instead.
var LibSecureMerkleTrieABI = LibSecureMerkleTrieMetaData.ABI

// LibSecureMerkleTrieBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LibSecureMerkleTrieMetaData.Bin instead.
var LibSecureMerkleTrieBin = LibSecureMerkleTrieMetaData.Bin

// DeployLibSecureMerkleTrie deploys a new Ethereum contract, binding an instance of LibSecureMerkleTrie to it.
func DeployLibSecureMerkleTrie(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *LibSecureMerkleTrie, error) {
	parsed, err := LibSecureMerkleTrieMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LibSecureMerkleTrieBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LibSecureMerkleTrie{LibSecureMerkleTrieCaller: LibSecureMerkleTrieCaller{contract: contract}, LibSecureMerkleTrieTransactor: LibSecureMerkleTrieTransactor{contract: contract}, LibSecureMerkleTrieFilterer: LibSecureMerkleTrieFilterer{contract: contract}}, nil
}

// LibSecureMerkleTrie is an auto generated Go binding around an Ethereum contract.
type LibSecureMerkleTrie struct {
	LibSecureMerkleTrieCaller     // Read-only binding to the contract
	LibSecureMerkleTrieTransactor // Write-only binding to the contract
	LibSecureMerkleTrieFilterer   // Log filterer for contract events
}

// LibSecureMerkleTrieCaller is an auto generated read-only Go binding around an Ethereum contract.
type LibSecureMerkleTrieCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibSecureMerkleTrieTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LibSecureMerkleTrieTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibSecureMerkleTrieFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LibSecureMerkleTrieFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LibSecureMerkleTrieSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LibSecureMerkleTrieSession struct {
	Contract     *LibSecureMerkleTrie // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// LibSecureMerkleTrieCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LibSecureMerkleTrieCallerSession struct {
	Contract *LibSecureMerkleTrieCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// LibSecureMerkleTrieTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LibSecureMerkleTrieTransactorSession struct {
	Contract     *LibSecureMerkleTrieTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// LibSecureMerkleTrieRaw is an auto generated low-level Go binding around an Ethereum contract.
type LibSecureMerkleTrieRaw struct {
	Contract *LibSecureMerkleTrie // Generic contract binding to access the raw methods on
}

// LibSecureMerkleTrieCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LibSecureMerkleTrieCallerRaw struct {
	Contract *LibSecureMerkleTrieCaller // Generic read-only contract binding to access the raw methods on
}

// LibSecureMerkleTrieTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LibSecureMerkleTrieTransactorRaw struct {
	Contract *LibSecureMerkleTrieTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLibSecureMerkleTrie creates a new instance of LibSecureMerkleTrie, bound to a specific deployed contract.
func NewLibSecureMerkleTrie(address common.Address, backend bind.ContractBackend) (*LibSecureMerkleTrie, error) {
	contract, err := bindLibSecureMerkleTrie(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LibSecureMerkleTrie{LibSecureMerkleTrieCaller: LibSecureMerkleTrieCaller{contract: contract}, LibSecureMerkleTrieTransactor: LibSecureMerkleTrieTransactor{contract: contract}, LibSecureMerkleTrieFilterer: LibSecureMerkleTrieFilterer{contract: contract}}, nil
}

// NewLibSecureMerkleTrieCaller creates a new read-only instance of LibSecureMerkleTrie, bound to a specific deployed contract.
func NewLibSecureMerkleTrieCaller(address common.Address, caller bind.ContractCaller) (*LibSecureMerkleTrieCaller, error) {
	contract, err := bindLibSecureMerkleTrie(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LibSecureMerkleTrieCaller{contract: contract}, nil
}

// NewLibSecureMerkleTrieTransactor creates a new write-only instance of LibSecureMerkleTrie, bound to a specific deployed contract.
func NewLibSecureMerkleTrieTransactor(address common.Address, transactor bind.ContractTransactor) (*LibSecureMerkleTrieTransactor, error) {
	contract, err := bindLibSecureMerkleTrie(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LibSecureMerkleTrieTransactor{contract: contract}, nil
}

// NewLibSecureMerkleTrieFilterer creates a new log filterer instance of LibSecureMerkleTrie, bound to a specific deployed contract.
func NewLibSecureMerkleTrieFilterer(address common.Address, filterer bind.ContractFilterer) (*LibSecureMerkleTrieFilterer, error) {
	contract, err := bindLibSecureMerkleTrie(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LibSecureMerkleTrieFilterer{contract: contract}, nil
}

// bindLibSecureMerkleTrie binds a generic wrapper to an already deployed contract.
func bindLibSecureMerkleTrie(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(LibSecureMerkleTrieABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibSecureMerkleTrie.Contract.LibSecureMerkleTrieCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibSecureMerkleTrie.Contract.LibSecureMerkleTrieTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibSecureMerkleTrie.Contract.LibSecureMerkleTrieTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LibSecureMerkleTrie.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LibSecureMerkleTrie.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LibSecureMerkleTrie *LibSecureMerkleTrieTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LibSecureMerkleTrie.Contract.contract.Transact(opts, method, params...)
}

// OptimismPortalMetaData contains all meta data concerning the OptimismPortal contract.
var OptimismPortalMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"_l2Oracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_finalizationPeriod\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"InvalidOutputRootProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidWithdrawalInclusionProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonZeroCreationTarget\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotYetFinal\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WithdrawalAlreadyFinalized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"mint\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"isCreation\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"FINALIZATION_PERIOD\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_ORACLE\",\"outputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawerStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structWithdrawalVerifier.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"_withdrawalProof\",\"type\":\"bytes\"}],\"name\":\"finalizeWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"finalizedWithdrawals\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x60c060405261dead6000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055503480156200005357600080fd5b506040516200306b3803806200306b83398181016040528101906200007991906200017a565b81818173ffffffffffffffffffffffffffffffffffffffff1660a08173ffffffffffffffffffffffffffffffffffffffff1681525050806080818152505050505050620001c1565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000620000f382620000c6565b9050919050565b60006200010782620000e6565b9050919050565b6200011981620000fa565b81146200012557600080fd5b50565b60008151905062000139816200010e565b92915050565b6000819050919050565b62000154816200013f565b81146200016057600080fd5b50565b600081519050620001748162000149565b92915050565b60008060408385031215620001945762000193620000c1565b5b6000620001a48582860162000128565b9250506020620001b78582860162000163565b9150509250929050565b60805160a051612e76620001f5600039600081816102b2015261037801526000818161031a01526106720152612e766000f3fe6080604052600436106100585760003560e01c80621c2ff6146100835780639bf62d82146100ae578063a14238e7146100d9578063eecf1c3614610116578063fa92670c1461013f578063ff61cc931461015b5761007e565b3661007e5761007c3334617530600060405180602001604052806000815250610186565b005b600080fd5b34801561008f57600080fd5b506100986102b0565b6040516100a59190611bd9565b60405180910390f35b3480156100ba57600080fd5b506100c36102d4565b6040516100d09190611c15565b60405180910390f35b3480156100e557600080fd5b5061010060048036038101906100fb9190611c7a565b6102f8565b60405161010d9190611cc2565b60405180910390f35b34801561012257600080fd5b5061013d60048036038101906101389190611dc8565b610318565b005b61015960048036038101906101549190612041565b610186565b005b34801561016757600080fd5b50610170610670565b60405161017d91906120e7565b60405180910390f35b8180156101c05750600073ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff1614155b156101f7576040517ff98844ef00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60003390503273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161461023b5761023833610694565b90505b8573ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad34888888886040516102a095949392919061218a565b60405180910390a3505050505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b60008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60016020528060005260406000206000915054906101000a900460ff1681565b7f00000000000000000000000000000000000000000000000000000000000000008401421015610374576040517fe4750a3000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663a25ae557866040518263ffffffff1660e01b81526004016103cf91906120e7565b602060405180830381865afa1580156103ec573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061041091906121f9565b905061041b846106b4565b8114610453576040517f9cc00b5b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006104648d8d8d8d8d8d8d6106fa565b90506000151561047a828760400135878761073c565b151514156104b4576040517feb00eb2200000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600115156001600083815260200190815260200160002060009054906101000a900460ff1615151415610513576040517fae89945400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8b6000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060008b73ffffffffffffffffffffffffffffffffffffffff168b8b908b8b60405161057f929190612256565b600060405180830381858888f193505050503d80600081146105bd576040519150601f19603f3d011682016040523d82523d6000602084013e6105c2565b606091505b5050905061dead6000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550600180600084815260200190815260200160002060006101000a81548160ff021916908315150217905550817f894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f60405160405180910390a25050505050505050505050505050565b7f000000000000000000000000000000000000000000000000000000000000000081565b600073111100000000000000000000000000000000111182019050919050565b600081600001358260200135836040013584606001356040516020016106dd949392919061227e565b604051602081830303815290604052805190602001209050919050565b60008787878787878760405160200161071997969594939291906122f0565b604051602081830303815290604052805190602001209050979650505050505050565b60008085600160405160200161075392919061235a565b60405160208183030381529060405280519060200120905061080f8160405160200161077f91906123a4565b6040516020818303038152906040526040518060400160405280600181526020017f010000000000000000000000000000000000000000000000000000000000000081525086868080601f016020809104026020016040519081016040528093929190818152602001838380828437600081840152601f19601f820116905080830192505050505050508861081a565b915050949350505050565b6000806108268661083f565b90506108348186868661086f565b915050949350505050565b6060818051906020012060405160200161085991906123a4565b6040516020818303038152906040529050919050565b600080600061087f8786866108a2565b915091508180156108965750610895868261097b565b5b92505050949350505050565b6000606060006108b185610996565b905060008060006108c3848a89610a8b565b925092509250600080835114905080806108da5750815b610919576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016109109061241c565b60405180910390fd5b6000816109355760405180602001604052806000815250610965565b61096486600187610946919061246b565b815181106109575761095661249f565b5b6020026020010151610efc565b5b9050818197509750505050505050935093915050565b60008180519060200120838051906020012014905092915050565b606060006109a383610f3d565b90506000815167ffffffffffffffff8111156109c2576109c1611f16565b5b6040519080825280602002602001820160405280156109fb57816020015b6109e8611b26565b8152602001906001900390816109e05790505b50905060005b8251811015610a80576000610a2f848381518110610a2257610a2161249f565b5b6020026020010151610f57565b90506040518060400160405280828152602001610a4b83610f3d565b815250838381518110610a6157610a6061249f565b5b6020026020010181905250508080610a78906124ce565b915050610a01565b508092505050919050565b60006060600080600090506000610aa187610fed565b90506000869050600080610ab3611b26565b60005b8c51811015610eac578c8181518110610ad257610ad161249f565b5b602002602001015191508284610ae89190612517565b9350600187610af79190612517565b96506000841415610b54578482600001518051906020012014610b4f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610b46906125b9565b60405180910390fd5b610c03565b602082600001515110610bb3578482600001518051906020012014610bae576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610ba590612625565b60405180910390fd5b610c02565b84610bc18360000151611192565b14610c01576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610bf890612691565b60405180910390fd5b5b5b60016010610c119190612517565b8260200151511415610c8c578551841415610c2b57610eac565b6000868581518110610c4057610c3f61249f565b5b602001015160f81c60f81b60f81c9050600083602001518260ff1681518110610c6c57610c6b61249f565b5b60200260200101519050610c7f816111cc565b9650600194505050610e99565b60028260200151511415610e5e576000610ca58361120a565b9050600081600081518110610cbd57610cbc61249f565b5b602001015160f81c60f81b60f81c90506000600282610cdc91906126ed565b6002610ce8919061271e565b90506000610cf9848360ff16611243565b90506000610d078b8a611243565b90506000610d158383611284565b9050600260ff168560ff161480610d325750600360ff168560ff16145b15610d8f57808351148015610d475750808251145b15610d5b57808a610d589190612517565b99505b608060f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19169a50505050505050610eac565b600060ff168560ff161480610daa5750600160ff168560ff16145b15610e235782518114610deb57608060f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19169a50505050505050610eac565b610e138860200151600181518110610e0657610e0561249f565b5b60200260200101516111cc565b9a50809850505050505050610e99565b6040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e55906127c4565b60405180910390fd5b6040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e9090612830565b60405180910390fd5b8080610ea4906124ce565b915050610ab6565b506000608060f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168514905086610ee58786611243565b829950995099505050505050505093509350939050565b6060610f3682602001516001846020015151610f18919061246b565b81518110610f2957610f2861249f565b5b6020026020010151610f57565b9050919050565b6060610f50610f4b83611347565b611375565b9050919050565b60606000806000610f678561156c565b92509250925060006001811115610f8157610f80612850565b5b816001811115610f9457610f93612850565b5b14610fd4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610fcb906128cb565b60405180910390fd5b610fe385602001518484611885565b9350505050919050565b6060600060028351610fff91906128eb565b67ffffffffffffffff81111561101857611017611f16565b5b6040519080825280601f01601f19166020018201604052801561104a5781602001600182028036833780820191505090505b50905060005b835181101561118857600484828151811061106e5761106d61249f565b5b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916901c826002836110ab91906128eb565b815181106110bc576110bb61249f565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506010848281518110611100576110ff61249f565b5b602001015160f81c60f81b60f81c61111891906126ed565b60f81b82600160028461112b91906128eb565b6111359190612517565b815181106111465761114561249f565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053508080611180906124ce565b915050611050565b5080915050919050565b60006020825110156111b057600060208301519050809150506111c7565b818060200190518101906111c491906121f9565b90505b919050565b600060606020836000015110156111ed576111e68361198b565b90506111f9565b6111f683610f57565b90505b61120281611192565b915050919050565b606061123c611237836020015160008151811061122a5761122961249f565b5b6020026020010151610f57565b610fed565b9050919050565b6060825182106112645760405180602001604052806000815250905061127e565b61127b8383848651611276919061246b565b61199d565b90505b92915050565b600080600090505b80845111801561129c5750808351115b801561132557508281815181106112b6576112b561249f565b5b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168482815181106112f6576112f561249f565b5b602001015160f81c60f81b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916145b1561133d578080611335906124ce565b91505061128c565b8091505092915050565b61134f611b40565b600060208301905060405180604001604052808451815260200182815250915050919050565b60606000806113838461156c565b925050915060018081111561139b5761139a612850565b5b8160018111156113ae576113ad612850565b5b146113ee576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016113e590612991565b60405180910390fd5b6000602067ffffffffffffffff81111561140b5761140a611f16565b5b60405190808252806020026020018201604052801561144457816020015b611431611b40565b8152602001906001900390816114295790505b5090506000808490505b866000015181101561155c576020821061149d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161149490612a23565b60405180910390fd5b6000806114db6040518060400160405280858c600001516114be919061246b565b8152602001858c602001516114d39190612517565b81525061156c565b5091509150604051806040016040528083836114f79190612517565b8152602001848b6020015161150c9190612517565b8152508585815181106115225761152161249f565b5b602002602001018190525060018461153a9190612517565b935080826115489190612517565b836115539190612517565b9250505061144e565b8183528295505050505050919050565b6000806000808460000151116115b7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016115ae90612a8f565b60405180910390fd5b6000846020015190506000815160001a9050607f81116115e457600060016000945094509450505061187e565b60b781116116565760006080826115fb919061246b565b905080876000015111611643576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161163a90612afb565b60405180910390fd5b600181600095509550955050505061187e565b60bf811161173757600060b78261166d919061246b565b9050808760000151116116b5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016116ac90612b67565b60405180910390fd5b6000816020036101000a600185015104905080826116d39190612517565b886000015111611718576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161170f90612bd3565b60405180910390fd5b8160016117259190612517565b8160009650965096505050505061187e565b60f781116117a957600060c08261174e919061246b565b905080876000015111611796576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161178d90612c3f565b60405180910390fd5b600181600195509550955050505061187e565b600060f7826117b8919061246b565b905080876000015111611800576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016117f790612cab565b60405180910390fd5b6000816020036101000a6001850151049050808261181e9190612517565b886000015111611863576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161185a90612d17565b60405180910390fd5b8160016118709190612517565b816001965096509650505050505b9193909250565b606060008267ffffffffffffffff8111156118a3576118a2611f16565b5b6040519080825280601f01601f1916602001820160405280156118d55781602001600182028036833780820191505090505b5090506000815114156118eb5780915050611984565b600084866118f99190612517565b9050600060208301905060005b6020866119139190612d37565b81101561194f578251825260208361192b9190612517565b925060208261193a9190612517565b91508080611947906124ce565b915050611906565b506000600160208781611965576119646126be565b5b066020036101000a039050808251168119845116178252839450505050505b9392505050565b606061199682611b09565b9050919050565b606081601f836119ad9190612517565b10156119ee576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016119e590612db4565b60405180910390fd5b8282846119fb9190612517565b1015611a3c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401611a3390612db4565b60405180910390fd5b8183611a489190612517565b84511015611a8b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401611a8290612e20565b60405180910390fd5b6060821560008114611aac5760405191506000825260208201604052611afd565b6040519150601f8416801560200281840101858101878315602002848b0101015b81831015611aea5780518352602083019250602081019050611acd565b50868552601f19601f8301166040525050505b50809150509392505050565b6060611b1f826020015160008460000151611885565b9050919050565b604051806040016040528060608152602001606081525090565b604051806040016040528060008152602001600081525090565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b6000611b9f611b9a611b9584611b5a565b611b7a565b611b5a565b9050919050565b6000611bb182611b84565b9050919050565b6000611bc382611ba6565b9050919050565b611bd381611bb8565b82525050565b6000602082019050611bee6000830184611bca565b92915050565b6000611bff82611b5a565b9050919050565b611c0f81611bf4565b82525050565b6000602082019050611c2a6000830184611c06565b92915050565b6000604051905090565b600080fd5b600080fd5b6000819050919050565b611c5781611c44565b8114611c6257600080fd5b50565b600081359050611c7481611c4e565b92915050565b600060208284031215611c9057611c8f611c3a565b5b6000611c9e84828501611c65565b91505092915050565b60008115159050919050565b611cbc81611ca7565b82525050565b6000602082019050611cd76000830184611cb3565b92915050565b6000819050919050565b611cf081611cdd565b8114611cfb57600080fd5b50565b600081359050611d0d81611ce7565b92915050565b611d1c81611bf4565b8114611d2757600080fd5b50565b600081359050611d3981611d13565b92915050565b600080fd5b600080fd5b600080fd5b60008083601f840112611d6457611d63611d3f565b5b8235905067ffffffffffffffff811115611d8157611d80611d44565b5b602083019150836001820283011115611d9d57611d9c611d49565b5b9250929050565b600080fd5b600060808284031215611dbf57611dbe611da4565b5b81905092915050565b60008060008060008060008060008060006101808c8e031215611dee57611ded611c3a565b5b6000611dfc8e828f01611cfe565b9b50506020611e0d8e828f01611d2a565b9a50506040611e1e8e828f01611d2a565b9950506060611e2f8e828f01611cfe565b9850506080611e408e828f01611cfe565b97505060a08c013567ffffffffffffffff811115611e6157611e60611c3f565b5b611e6d8e828f01611d4e565b965096505060c0611e808e828f01611cfe565b94505060e0611e918e828f01611da9565b9350506101608c013567ffffffffffffffff811115611eb357611eb2611c3f565b5b611ebf8e828f01611d4e565b92509250509295989b509295989b9093969950565b611edd81611ca7565b8114611ee857600080fd5b50565b600081359050611efa81611ed4565b92915050565b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b611f4e82611f05565b810181811067ffffffffffffffff82111715611f6d57611f6c611f16565b5b80604052505050565b6000611f80611c30565b9050611f8c8282611f45565b919050565b600067ffffffffffffffff821115611fac57611fab611f16565b5b611fb582611f05565b9050602081019050919050565b82818337600083830152505050565b6000611fe4611fdf84611f91565b611f76565b90508281526020810184848401111561200057611fff611f00565b5b61200b848285611fc2565b509392505050565b600082601f83011261202857612027611d3f565b5b8135612038848260208601611fd1565b91505092915050565b600080600080600060a0868803121561205d5761205c611c3a565b5b600061206b88828901611d2a565b955050602061207c88828901611cfe565b945050604061208d88828901611cfe565b935050606061209e88828901611eeb565b925050608086013567ffffffffffffffff8111156120bf576120be611c3f565b5b6120cb88828901612013565b9150509295509295909350565b6120e181611cdd565b82525050565b60006020820190506120fc60008301846120d8565b92915050565b600081519050919050565b600082825260208201905092915050565b60005b8381101561213c578082015181840152602081019050612121565b8381111561214b576000848401525b50505050565b600061215c82612102565b612166818561210d565b935061217681856020860161211e565b61217f81611f05565b840191505092915050565b600060a08201905061219f60008301886120d8565b6121ac60208301876120d8565b6121b960408301866120d8565b6121c66060830185611cb3565b81810360808301526121d88184612151565b90509695505050505050565b6000815190506121f381611c4e565b92915050565b60006020828403121561220f5761220e611c3a565b5b600061221d848285016121e4565b91505092915050565b600081905092915050565b600061223d8385612226565b935061224a838584611fc2565b82840190509392505050565b6000612263828486612231565b91508190509392505050565b61227881611c44565b82525050565b6000608082019050612293600083018761226f565b6122a0602083018661226f565b6122ad604083018561226f565b6122ba606083018461226f565b95945050505050565b60006122cf838561210d565b93506122dc838584611fc2565b6122e583611f05565b840190509392505050565b600060c082019050612305600083018a6120d8565b6123126020830189611c06565b61231f6040830188611c06565b61232c60608301876120d8565b61233960808301866120d8565b81810360a083015261234c8184866122c3565b905098975050505050505050565b600060408201905061236f600083018561226f565b61237c60208301846120d8565b9392505050565b6000819050919050565b61239e61239982611c44565b612383565b82525050565b60006123b0828461238d565b60208201915081905092915050565b600082825260208201905092915050565b7f50726f76696465642070726f6f6620697320696e76616c69642e000000000000600082015250565b6000612406601a836123bf565b9150612411826123d0565b602082019050919050565b60006020820190508181036000830152612435816123f9565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061247682611cdd565b915061248183611cdd565b9250828210156124945761249361243c565b5b828203905092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60006124d982611cdd565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82141561250c5761250b61243c565b5b600182019050919050565b600061252282611cdd565b915061252d83611cdd565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156125625761256161243c565b5b828201905092915050565b7f496e76616c696420726f6f742068617368000000000000000000000000000000600082015250565b60006125a36011836123bf565b91506125ae8261256d565b602082019050919050565b600060208201905081810360008301526125d281612596565b9050919050565b7f496e76616c6964206c6172676520696e7465726e616c20686173680000000000600082015250565b600061260f601b836123bf565b915061261a826125d9565b602082019050919050565b6000602082019050818103600083015261263e81612602565b9050919050565b7f496e76616c696420696e7465726e616c206e6f64652068617368000000000000600082015250565b600061267b601a836123bf565b915061268682612645565b602082019050919050565b600060208201905081810360008301526126aa8161266e565b9050919050565b600060ff82169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60006126f8826126b1565b9150612703836126b1565b925082612713576127126126be565b5b828206905092915050565b6000612729826126b1565b9150612734836126b1565b9250828210156127475761274661243c565b5b828203905092915050565b7f52656365697665642061206e6f6465207769746820616e20756e6b6e6f776e2060008201527f7072656669780000000000000000000000000000000000000000000000000000602082015250565b60006127ae6026836123bf565b91506127b982612752565b604082019050919050565b600060208201905081810360008301526127dd816127a1565b9050919050565b7f526563656976656420616e20756e706172736561626c65206e6f64652e000000600082015250565b600061281a601d836123bf565b9150612825826127e4565b602082019050919050565b600060208201905081810360008301526128498161280d565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b7f496e76616c696420524c502062797465732076616c75652e0000000000000000600082015250565b60006128b56018836123bf565b91506128c08261287f565b602082019050919050565b600060208201905081810360008301526128e4816128a8565b9050919050565b60006128f682611cdd565b915061290183611cdd565b9250817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff048311821515161561293a5761293961243c565b5b828202905092915050565b7f496e76616c696420524c50206c6973742076616c75652e000000000000000000600082015250565b600061297b6017836123bf565b915061298682612945565b602082019050919050565b600060208201905081810360008301526129aa8161296e565b9050919050565b7f50726f766964656420524c50206c6973742065786365656473206d6178206c6960008201527f7374206c656e6774682e00000000000000000000000000000000000000000000602082015250565b6000612a0d602a836123bf565b9150612a18826129b1565b604082019050919050565b60006020820190508181036000830152612a3c81612a00565b9050919050565b7f524c50206974656d2063616e6e6f74206265206e756c6c2e0000000000000000600082015250565b6000612a796018836123bf565b9150612a8482612a43565b602082019050919050565b60006020820190508181036000830152612aa881612a6c565b9050919050565b7f496e76616c696420524c502073686f727420737472696e672e00000000000000600082015250565b6000612ae56019836123bf565b9150612af082612aaf565b602082019050919050565b60006020820190508181036000830152612b1481612ad8565b9050919050565b7f496e76616c696420524c50206c6f6e6720737472696e67206c656e6774682e00600082015250565b6000612b51601f836123bf565b9150612b5c82612b1b565b602082019050919050565b60006020820190508181036000830152612b8081612b44565b9050919050565b7f496e76616c696420524c50206c6f6e6720737472696e672e0000000000000000600082015250565b6000612bbd6018836123bf565b9150612bc882612b87565b602082019050919050565b60006020820190508181036000830152612bec81612bb0565b9050919050565b7f496e76616c696420524c502073686f7274206c6973742e000000000000000000600082015250565b6000612c296017836123bf565b9150612c3482612bf3565b602082019050919050565b60006020820190508181036000830152612c5881612c1c565b9050919050565b7f496e76616c696420524c50206c6f6e67206c697374206c656e6774682e000000600082015250565b6000612c95601d836123bf565b9150612ca082612c5f565b602082019050919050565b60006020820190508181036000830152612cc481612c88565b9050919050565b7f496e76616c696420524c50206c6f6e67206c6973742e00000000000000000000600082015250565b6000612d016016836123bf565b9150612d0c82612ccb565b602082019050919050565b60006020820190508181036000830152612d3081612cf4565b9050919050565b6000612d4282611cdd565b9150612d4d83611cdd565b925082612d5d57612d5c6126be565b5b828204905092915050565b7f736c6963655f6f766572666c6f77000000000000000000000000000000000000600082015250565b6000612d9e600e836123bf565b9150612da982612d68565b602082019050919050565b60006020820190508181036000830152612dcd81612d91565b9050919050565b7f736c6963655f6f75744f66426f756e6473000000000000000000000000000000600082015250565b6000612e0a6011836123bf565b9150612e1582612dd4565b602082019050919050565b60006020820190508181036000830152612e3981612dfd565b905091905056fea2646970667358221220476f1b0a95f0a876e335daed5cca410381992c0577e2b5bd936c3a5060def44964736f6c634300080a0033",
}

// OptimismPortalABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismPortalMetaData.ABI instead.
var OptimismPortalABI = OptimismPortalMetaData.ABI

// OptimismPortalBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OptimismPortalMetaData.Bin instead.
var OptimismPortalBin = OptimismPortalMetaData.Bin

// DeployOptimismPortal deploys a new Ethereum contract, binding an instance of OptimismPortal to it.
func DeployOptimismPortal(auth *bind.TransactOpts, backend bind.ContractBackend, _l2Oracle common.Address, _finalizationPeriod *big.Int) (common.Address, *types.Transaction, *OptimismPortal, error) {
	parsed, err := OptimismPortalMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OptimismPortalBin), backend, _l2Oracle, _finalizationPeriod)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OptimismPortal{OptimismPortalCaller: OptimismPortalCaller{contract: contract}, OptimismPortalTransactor: OptimismPortalTransactor{contract: contract}, OptimismPortalFilterer: OptimismPortalFilterer{contract: contract}}, nil
}

// OptimismPortal is an auto generated Go binding around an Ethereum contract.
type OptimismPortal struct {
	OptimismPortalCaller     // Read-only binding to the contract
	OptimismPortalTransactor // Write-only binding to the contract
	OptimismPortalFilterer   // Log filterer for contract events
}

// OptimismPortalCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismPortalCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismPortalTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismPortalFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortalSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismPortalSession struct {
	Contract     *OptimismPortal   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismPortalCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismPortalCallerSession struct {
	Contract *OptimismPortalCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// OptimismPortalTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismPortalTransactorSession struct {
	Contract     *OptimismPortalTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// OptimismPortalRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismPortalRaw struct {
	Contract *OptimismPortal // Generic contract binding to access the raw methods on
}

// OptimismPortalCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismPortalCallerRaw struct {
	Contract *OptimismPortalCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismPortalTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismPortalTransactorRaw struct {
	Contract *OptimismPortalTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismPortal creates a new instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortal(address common.Address, backend bind.ContractBackend) (*OptimismPortal, error) {
	contract, err := bindOptimismPortal(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal{OptimismPortalCaller: OptimismPortalCaller{contract: contract}, OptimismPortalTransactor: OptimismPortalTransactor{contract: contract}, OptimismPortalFilterer: OptimismPortalFilterer{contract: contract}}, nil
}

// NewOptimismPortalCaller creates a new read-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalCaller(address common.Address, caller bind.ContractCaller) (*OptimismPortalCaller, error) {
	contract, err := bindOptimismPortal(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalCaller{contract: contract}, nil
}

// NewOptimismPortalTransactor creates a new write-only instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismPortalTransactor, error) {
	contract, err := bindOptimismPortal(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactor{contract: contract}, nil
}

// NewOptimismPortalFilterer creates a new log filterer instance of OptimismPortal, bound to a specific deployed contract.
func NewOptimismPortalFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismPortalFilterer, error) {
	contract, err := bindOptimismPortal(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalFilterer{contract: contract}, nil
}

// bindOptimismPortal binds a generic wrapper to an already deployed contract.
func bindOptimismPortal(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OptimismPortalABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.OptimismPortalCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.OptimismPortalTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal *OptimismPortalCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal *OptimismPortalTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal.Contract.contract.Transact(opts, method, params...)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalCaller) FINALIZATIONPERIOD(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "FINALIZATION_PERIOD")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalSession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _OptimismPortal.Contract.FINALIZATIONPERIOD(&_OptimismPortal.CallOpts)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_OptimismPortal *OptimismPortalCallerSession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _OptimismPortal.Contract.FINALIZATIONPERIOD(&_OptimismPortal.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2ORACLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "L2_ORACLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2ORACLE() (common.Address, error) {
	return _OptimismPortal.Contract.L2ORACLE(&_OptimismPortal.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2ORACLE() (common.Address, error) {
	return _OptimismPortal.Contract.L2ORACLE(&_OptimismPortal.CallOpts)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCaller) FinalizedWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "finalizedWithdrawals", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal *OptimismPortalCallerSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal.Contract.FinalizedWithdrawals(&_OptimismPortal.CallOpts, arg0)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCaller) L2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal.contract.Call(opts, &out, "l2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal *OptimismPortalCallerSession) L2Sender() (common.Address, error) {
	return _OptimismPortal.Contract.L2Sender(&_OptimismPortal.CallOpts)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xfa92670c.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint256 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit *big.Int, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.DepositTransaction(&_OptimismPortal.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactor) FinalizeWithdrawalTransaction(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.contract.Transact(opts, "finalizeWithdrawalTransaction", _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalSession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_OptimismPortal *OptimismPortalTransactorSession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _OptimismPortal.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal *OptimismPortalTransactorSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal.Contract.Receive(&_OptimismPortal.TransactOpts)
}

// OptimismPortalTransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the OptimismPortal contract.
type OptimismPortalTransactionDepositedIterator struct {
	Event *OptimismPortalTransactionDeposited // Event containing the contract specifics and raw log

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
func (it *OptimismPortalTransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalTransactionDeposited)
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
		it.Event = new(OptimismPortalTransactionDeposited)
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
func (it *OptimismPortalTransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalTransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalTransactionDeposited represents a TransactionDeposited event raised by the OptimismPortal contract.
type OptimismPortalTransactionDeposited struct {
	From       common.Address
	To         common.Address
	Mint       *big.Int
	Value      *big.Int
	GasLimit   *big.Int
	IsCreation bool
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*OptimismPortalTransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalTransactionDepositedIterator{contract: _OptimismPortal.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *OptimismPortalTransactionDeposited, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalTransactionDeposited)
				if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
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

// ParseTransactionDeposited is a log parse operation binding the contract event 0x26137a5e34446f63aa9ea28797a0e70c3987720913879898802dd60b944615ad.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 mint, uint256 value, uint256 gasLimit, bool isCreation, bytes data)
func (_OptimismPortal *OptimismPortalFilterer) ParseTransactionDeposited(log types.Log) (*OptimismPortalTransactionDeposited, error) {
	event := new(OptimismPortalTransactionDeposited)
	if err := _OptimismPortal.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortalWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalizedIterator struct {
	Event *OptimismPortalWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *OptimismPortalWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortalWithdrawalFinalized)
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
		it.Event = new(OptimismPortalWithdrawalFinalized)
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
func (it *OptimismPortalWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortalWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortalWithdrawalFinalized represents a WithdrawalFinalized event raised by the OptimismPortal contract.
type OptimismPortalWithdrawalFinalized struct {
	Arg0 [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, arg0 [][32]byte) (*OptimismPortalWithdrawalFinalizedIterator, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _OptimismPortal.contract.FilterLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortalWithdrawalFinalizedIterator{contract: _OptimismPortal.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *OptimismPortalWithdrawalFinalized, arg0 [][32]byte) (event.Subscription, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _OptimismPortal.contract.WatchLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortalWithdrawalFinalized)
				if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_OptimismPortal *OptimismPortalFilterer) ParseWithdrawalFinalized(log types.Log) (*OptimismPortalWithdrawalFinalized, error) {
	event := new(OptimismPortalWithdrawalFinalized)
	if err := _OptimismPortal.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OwnableMetaData contains all meta data concerning the Ownable contract.
var OwnableMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// OwnableABI is the input ABI used to generate the binding from.
// Deprecated: Use OwnableMetaData.ABI instead.
var OwnableABI = OwnableMetaData.ABI

// Ownable is an auto generated Go binding around an Ethereum contract.
type Ownable struct {
	OwnableCaller     // Read-only binding to the contract
	OwnableTransactor // Write-only binding to the contract
	OwnableFilterer   // Log filterer for contract events
}

// OwnableCaller is an auto generated read-only Go binding around an Ethereum contract.
type OwnableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OwnableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OwnableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OwnableSession struct {
	Contract     *Ownable          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OwnableCallerSession struct {
	Contract *OwnableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// OwnableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OwnableTransactorSession struct {
	Contract     *OwnableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// OwnableRaw is an auto generated low-level Go binding around an Ethereum contract.
type OwnableRaw struct {
	Contract *Ownable // Generic contract binding to access the raw methods on
}

// OwnableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OwnableCallerRaw struct {
	Contract *OwnableCaller // Generic read-only contract binding to access the raw methods on
}

// OwnableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OwnableTransactorRaw struct {
	Contract *OwnableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwnable creates a new instance of Ownable, bound to a specific deployed contract.
func NewOwnable(address common.Address, backend bind.ContractBackend) (*Ownable, error) {
	contract, err := bindOwnable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Ownable{OwnableCaller: OwnableCaller{contract: contract}, OwnableTransactor: OwnableTransactor{contract: contract}, OwnableFilterer: OwnableFilterer{contract: contract}}, nil
}

// NewOwnableCaller creates a new read-only instance of Ownable, bound to a specific deployed contract.
func NewOwnableCaller(address common.Address, caller bind.ContractCaller) (*OwnableCaller, error) {
	contract, err := bindOwnable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OwnableCaller{contract: contract}, nil
}

// NewOwnableTransactor creates a new write-only instance of Ownable, bound to a specific deployed contract.
func NewOwnableTransactor(address common.Address, transactor bind.ContractTransactor) (*OwnableTransactor, error) {
	contract, err := bindOwnable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OwnableTransactor{contract: contract}, nil
}

// NewOwnableFilterer creates a new log filterer instance of Ownable, bound to a specific deployed contract.
func NewOwnableFilterer(address common.Address, filterer bind.ContractFilterer) (*OwnableFilterer, error) {
	contract, err := bindOwnable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OwnableFilterer{contract: contract}, nil
}

// bindOwnable binds a generic wrapper to an already deployed contract.
func bindOwnable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnableABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ownable *OwnableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ownable.Contract.OwnableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ownable *OwnableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ownable.Contract.OwnableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ownable *OwnableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ownable.Contract.OwnableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ownable *OwnableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ownable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ownable *OwnableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ownable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ownable *OwnableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ownable.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Ownable.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableSession) Owner() (common.Address, error) {
	return _Ownable.Contract.Owner(&_Ownable.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableCallerSession) Owner() (common.Address, error) {
	return _Ownable.Contract.Owner(&_Ownable.CallOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Ownable *OwnableTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ownable.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Ownable *OwnableSession) RenounceOwnership() (*types.Transaction, error) {
	return _Ownable.Contract.RenounceOwnership(&_Ownable.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Ownable *OwnableTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Ownable.Contract.RenounceOwnership(&_Ownable.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Ownable *OwnableTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Ownable *OwnableSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.Contract.TransferOwnership(&_Ownable.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Ownable *OwnableTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.Contract.TransferOwnership(&_Ownable.TransactOpts, newOwner)
}

// OwnableOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Ownable contract.
type OwnableOwnershipTransferredIterator struct {
	Event *OwnableOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *OwnableOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OwnableOwnershipTransferred)
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
		it.Event = new(OwnableOwnershipTransferred)
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
func (it *OwnableOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OwnableOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OwnableOwnershipTransferred represents a OwnershipTransferred event raised by the Ownable contract.
type OwnableOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Ownable *OwnableFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*OwnableOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Ownable.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &OwnableOwnershipTransferredIterator{contract: _Ownable.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Ownable *OwnableFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *OwnableOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Ownable.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OwnableOwnershipTransferred)
				if err := _Ownable.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Ownable *OwnableFilterer) ParseOwnershipTransferred(log types.Log) (*OwnableOwnershipTransferred, error) {
	event := new(OwnableOwnershipTransferred)
	if err := _Ownable.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// WithdrawalVerifierMetaData contains all meta data concerning the WithdrawalVerifier contract.
var WithdrawalVerifierMetaData = &bind.MetaData{
	ABI: "[]",
	Bin: "0x60566050600b82828239805160001a6073146043577f4e487b7100000000000000000000000000000000000000000000000000000000600052600060045260246000fd5b30600052607381538281f3fe73000000000000000000000000000000000000000030146080604052600080fdfea26469706673582212203fefb10a3ca14596cd54a35ac033725fc08679d4c580652ab8e765249653bb8c64736f6c634300080a0033",
}

// WithdrawalVerifierABI is the input ABI used to generate the binding from.
// Deprecated: Use WithdrawalVerifierMetaData.ABI instead.
var WithdrawalVerifierABI = WithdrawalVerifierMetaData.ABI

// WithdrawalVerifierBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use WithdrawalVerifierMetaData.Bin instead.
var WithdrawalVerifierBin = WithdrawalVerifierMetaData.Bin

// DeployWithdrawalVerifier deploys a new Ethereum contract, binding an instance of WithdrawalVerifier to it.
func DeployWithdrawalVerifier(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *WithdrawalVerifier, error) {
	parsed, err := WithdrawalVerifierMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(WithdrawalVerifierBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &WithdrawalVerifier{WithdrawalVerifierCaller: WithdrawalVerifierCaller{contract: contract}, WithdrawalVerifierTransactor: WithdrawalVerifierTransactor{contract: contract}, WithdrawalVerifierFilterer: WithdrawalVerifierFilterer{contract: contract}}, nil
}

// WithdrawalVerifier is an auto generated Go binding around an Ethereum contract.
type WithdrawalVerifier struct {
	WithdrawalVerifierCaller     // Read-only binding to the contract
	WithdrawalVerifierTransactor // Write-only binding to the contract
	WithdrawalVerifierFilterer   // Log filterer for contract events
}

// WithdrawalVerifierCaller is an auto generated read-only Go binding around an Ethereum contract.
type WithdrawalVerifierCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalVerifierTransactor is an auto generated write-only Go binding around an Ethereum contract.
type WithdrawalVerifierTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalVerifierFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type WithdrawalVerifierFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalVerifierSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type WithdrawalVerifierSession struct {
	Contract     *WithdrawalVerifier // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// WithdrawalVerifierCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type WithdrawalVerifierCallerSession struct {
	Contract *WithdrawalVerifierCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// WithdrawalVerifierTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type WithdrawalVerifierTransactorSession struct {
	Contract     *WithdrawalVerifierTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// WithdrawalVerifierRaw is an auto generated low-level Go binding around an Ethereum contract.
type WithdrawalVerifierRaw struct {
	Contract *WithdrawalVerifier // Generic contract binding to access the raw methods on
}

// WithdrawalVerifierCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type WithdrawalVerifierCallerRaw struct {
	Contract *WithdrawalVerifierCaller // Generic read-only contract binding to access the raw methods on
}

// WithdrawalVerifierTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type WithdrawalVerifierTransactorRaw struct {
	Contract *WithdrawalVerifierTransactor // Generic write-only contract binding to access the raw methods on
}

// NewWithdrawalVerifier creates a new instance of WithdrawalVerifier, bound to a specific deployed contract.
func NewWithdrawalVerifier(address common.Address, backend bind.ContractBackend) (*WithdrawalVerifier, error) {
	contract, err := bindWithdrawalVerifier(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &WithdrawalVerifier{WithdrawalVerifierCaller: WithdrawalVerifierCaller{contract: contract}, WithdrawalVerifierTransactor: WithdrawalVerifierTransactor{contract: contract}, WithdrawalVerifierFilterer: WithdrawalVerifierFilterer{contract: contract}}, nil
}

// NewWithdrawalVerifierCaller creates a new read-only instance of WithdrawalVerifier, bound to a specific deployed contract.
func NewWithdrawalVerifierCaller(address common.Address, caller bind.ContractCaller) (*WithdrawalVerifierCaller, error) {
	contract, err := bindWithdrawalVerifier(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &WithdrawalVerifierCaller{contract: contract}, nil
}

// NewWithdrawalVerifierTransactor creates a new write-only instance of WithdrawalVerifier, bound to a specific deployed contract.
func NewWithdrawalVerifierTransactor(address common.Address, transactor bind.ContractTransactor) (*WithdrawalVerifierTransactor, error) {
	contract, err := bindWithdrawalVerifier(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &WithdrawalVerifierTransactor{contract: contract}, nil
}

// NewWithdrawalVerifierFilterer creates a new log filterer instance of WithdrawalVerifier, bound to a specific deployed contract.
func NewWithdrawalVerifierFilterer(address common.Address, filterer bind.ContractFilterer) (*WithdrawalVerifierFilterer, error) {
	contract, err := bindWithdrawalVerifier(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalVerifierFilterer{contract: contract}, nil
}

// bindWithdrawalVerifier binds a generic wrapper to an already deployed contract.
func bindWithdrawalVerifier(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(WithdrawalVerifierABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WithdrawalVerifier *WithdrawalVerifierRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WithdrawalVerifier.Contract.WithdrawalVerifierCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WithdrawalVerifier *WithdrawalVerifierRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WithdrawalVerifier.Contract.WithdrawalVerifierTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WithdrawalVerifier *WithdrawalVerifierRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WithdrawalVerifier.Contract.WithdrawalVerifierTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WithdrawalVerifier *WithdrawalVerifierCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WithdrawalVerifier.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WithdrawalVerifier *WithdrawalVerifierTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WithdrawalVerifier.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WithdrawalVerifier *WithdrawalVerifierTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WithdrawalVerifier.Contract.contract.Transact(opts, method, params...)
}

// WithdrawalsRelayMetaData contains all meta data concerning the WithdrawalsRelay contract.
var WithdrawalsRelayMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"InvalidOutputRootProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidWithdrawalInclusionProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotYetFinal\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"WithdrawalAlreadyFinalized\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"FINALIZATION_PERIOD\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_ORACLE\",\"outputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_timestamp\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"withdrawerStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structWithdrawalVerifier.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"_withdrawalProof\",\"type\":\"bytes\"}],\"name\":\"finalizeWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"finalizedWithdrawals\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// WithdrawalsRelayABI is the input ABI used to generate the binding from.
// Deprecated: Use WithdrawalsRelayMetaData.ABI instead.
var WithdrawalsRelayABI = WithdrawalsRelayMetaData.ABI

// WithdrawalsRelay is an auto generated Go binding around an Ethereum contract.
type WithdrawalsRelay struct {
	WithdrawalsRelayCaller     // Read-only binding to the contract
	WithdrawalsRelayTransactor // Write-only binding to the contract
	WithdrawalsRelayFilterer   // Log filterer for contract events
}

// WithdrawalsRelayCaller is an auto generated read-only Go binding around an Ethereum contract.
type WithdrawalsRelayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalsRelayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type WithdrawalsRelayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalsRelayFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type WithdrawalsRelayFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// WithdrawalsRelaySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type WithdrawalsRelaySession struct {
	Contract     *WithdrawalsRelay // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// WithdrawalsRelayCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type WithdrawalsRelayCallerSession struct {
	Contract *WithdrawalsRelayCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// WithdrawalsRelayTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type WithdrawalsRelayTransactorSession struct {
	Contract     *WithdrawalsRelayTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// WithdrawalsRelayRaw is an auto generated low-level Go binding around an Ethereum contract.
type WithdrawalsRelayRaw struct {
	Contract *WithdrawalsRelay // Generic contract binding to access the raw methods on
}

// WithdrawalsRelayCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type WithdrawalsRelayCallerRaw struct {
	Contract *WithdrawalsRelayCaller // Generic read-only contract binding to access the raw methods on
}

// WithdrawalsRelayTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type WithdrawalsRelayTransactorRaw struct {
	Contract *WithdrawalsRelayTransactor // Generic write-only contract binding to access the raw methods on
}

// NewWithdrawalsRelay creates a new instance of WithdrawalsRelay, bound to a specific deployed contract.
func NewWithdrawalsRelay(address common.Address, backend bind.ContractBackend) (*WithdrawalsRelay, error) {
	contract, err := bindWithdrawalsRelay(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &WithdrawalsRelay{WithdrawalsRelayCaller: WithdrawalsRelayCaller{contract: contract}, WithdrawalsRelayTransactor: WithdrawalsRelayTransactor{contract: contract}, WithdrawalsRelayFilterer: WithdrawalsRelayFilterer{contract: contract}}, nil
}

// NewWithdrawalsRelayCaller creates a new read-only instance of WithdrawalsRelay, bound to a specific deployed contract.
func NewWithdrawalsRelayCaller(address common.Address, caller bind.ContractCaller) (*WithdrawalsRelayCaller, error) {
	contract, err := bindWithdrawalsRelay(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &WithdrawalsRelayCaller{contract: contract}, nil
}

// NewWithdrawalsRelayTransactor creates a new write-only instance of WithdrawalsRelay, bound to a specific deployed contract.
func NewWithdrawalsRelayTransactor(address common.Address, transactor bind.ContractTransactor) (*WithdrawalsRelayTransactor, error) {
	contract, err := bindWithdrawalsRelay(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &WithdrawalsRelayTransactor{contract: contract}, nil
}

// NewWithdrawalsRelayFilterer creates a new log filterer instance of WithdrawalsRelay, bound to a specific deployed contract.
func NewWithdrawalsRelayFilterer(address common.Address, filterer bind.ContractFilterer) (*WithdrawalsRelayFilterer, error) {
	contract, err := bindWithdrawalsRelay(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &WithdrawalsRelayFilterer{contract: contract}, nil
}

// bindWithdrawalsRelay binds a generic wrapper to an already deployed contract.
func bindWithdrawalsRelay(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(WithdrawalsRelayABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WithdrawalsRelay *WithdrawalsRelayRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WithdrawalsRelay.Contract.WithdrawalsRelayCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WithdrawalsRelay *WithdrawalsRelayRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.WithdrawalsRelayTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WithdrawalsRelay *WithdrawalsRelayRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.WithdrawalsRelayTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_WithdrawalsRelay *WithdrawalsRelayCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _WithdrawalsRelay.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_WithdrawalsRelay *WithdrawalsRelayTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_WithdrawalsRelay *WithdrawalsRelayTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.contract.Transact(opts, method, params...)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_WithdrawalsRelay *WithdrawalsRelayCaller) FINALIZATIONPERIOD(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _WithdrawalsRelay.contract.Call(opts, &out, "FINALIZATION_PERIOD")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_WithdrawalsRelay *WithdrawalsRelaySession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _WithdrawalsRelay.Contract.FINALIZATIONPERIOD(&_WithdrawalsRelay.CallOpts)
}

// FINALIZATIONPERIOD is a free data retrieval call binding the contract method 0xff61cc93.
//
// Solidity: function FINALIZATION_PERIOD() view returns(uint256)
func (_WithdrawalsRelay *WithdrawalsRelayCallerSession) FINALIZATIONPERIOD() (*big.Int, error) {
	return _WithdrawalsRelay.Contract.FINALIZATIONPERIOD(&_WithdrawalsRelay.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelayCaller) L2ORACLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _WithdrawalsRelay.contract.Call(opts, &out, "L2_ORACLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelaySession) L2ORACLE() (common.Address, error) {
	return _WithdrawalsRelay.Contract.L2ORACLE(&_WithdrawalsRelay.CallOpts)
}

// L2ORACLE is a free data retrieval call binding the contract method 0x001c2ff6.
//
// Solidity: function L2_ORACLE() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelayCallerSession) L2ORACLE() (common.Address, error) {
	return _WithdrawalsRelay.Contract.L2ORACLE(&_WithdrawalsRelay.CallOpts)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_WithdrawalsRelay *WithdrawalsRelayCaller) FinalizedWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _WithdrawalsRelay.contract.Call(opts, &out, "finalizedWithdrawals", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_WithdrawalsRelay *WithdrawalsRelaySession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _WithdrawalsRelay.Contract.FinalizedWithdrawals(&_WithdrawalsRelay.CallOpts, arg0)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_WithdrawalsRelay *WithdrawalsRelayCallerSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _WithdrawalsRelay.Contract.FinalizedWithdrawals(&_WithdrawalsRelay.CallOpts, arg0)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelayCaller) L2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _WithdrawalsRelay.contract.Call(opts, &out, "l2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelaySession) L2Sender() (common.Address, error) {
	return _WithdrawalsRelay.Contract.L2Sender(&_WithdrawalsRelay.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_WithdrawalsRelay *WithdrawalsRelayCallerSession) L2Sender() (common.Address, error) {
	return _WithdrawalsRelay.Contract.L2Sender(&_WithdrawalsRelay.CallOpts)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_WithdrawalsRelay *WithdrawalsRelayTransactor) FinalizeWithdrawalTransaction(opts *bind.TransactOpts, _nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _WithdrawalsRelay.contract.Transact(opts, "finalizeWithdrawalTransaction", _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_WithdrawalsRelay *WithdrawalsRelaySession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.FinalizeWithdrawalTransaction(&_WithdrawalsRelay.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0xeecf1c36.
//
// Solidity: function finalizeWithdrawalTransaction(uint256 _nonce, address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data, uint256 _timestamp, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes _withdrawalProof) returns()
func (_WithdrawalsRelay *WithdrawalsRelayTransactorSession) FinalizeWithdrawalTransaction(_nonce *big.Int, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte, _timestamp *big.Int, _outputRootProof WithdrawalVerifierOutputRootProof, _withdrawalProof []byte) (*types.Transaction, error) {
	return _WithdrawalsRelay.Contract.FinalizeWithdrawalTransaction(&_WithdrawalsRelay.TransactOpts, _nonce, _sender, _target, _value, _gasLimit, _data, _timestamp, _outputRootProof, _withdrawalProof)
}

// WithdrawalsRelayWithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the WithdrawalsRelay contract.
type WithdrawalsRelayWithdrawalFinalizedIterator struct {
	Event *WithdrawalsRelayWithdrawalFinalized // Event containing the contract specifics and raw log

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
func (it *WithdrawalsRelayWithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(WithdrawalsRelayWithdrawalFinalized)
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
		it.Event = new(WithdrawalsRelayWithdrawalFinalized)
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
func (it *WithdrawalsRelayWithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *WithdrawalsRelayWithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// WithdrawalsRelayWithdrawalFinalized represents a WithdrawalFinalized event raised by the WithdrawalsRelay contract.
type WithdrawalsRelayWithdrawalFinalized struct {
	Arg0 [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_WithdrawalsRelay *WithdrawalsRelayFilterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, arg0 [][32]byte) (*WithdrawalsRelayWithdrawalFinalizedIterator, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _WithdrawalsRelay.contract.FilterLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return &WithdrawalsRelayWithdrawalFinalizedIterator{contract: _WithdrawalsRelay.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_WithdrawalsRelay *WithdrawalsRelayFilterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *WithdrawalsRelayWithdrawalFinalized, arg0 [][32]byte) (event.Subscription, error) {

	var arg0Rule []interface{}
	for _, arg0Item := range arg0 {
		arg0Rule = append(arg0Rule, arg0Item)
	}

	logs, sub, err := _WithdrawalsRelay.contract.WatchLogs(opts, "WithdrawalFinalized", arg0Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(WithdrawalsRelayWithdrawalFinalized)
				if err := _WithdrawalsRelay.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
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

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0x894485e328061b8d209b7dd043d2f613fc2892260497cadefac9a183962a990f.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed arg0)
func (_WithdrawalsRelay *WithdrawalsRelayFilterer) ParseWithdrawalFinalized(log types.Log) (*WithdrawalsRelayWithdrawalFinalized, error) {
	event := new(WithdrawalsRelayWithdrawalFinalized)
	if err := _WithdrawalsRelay.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
