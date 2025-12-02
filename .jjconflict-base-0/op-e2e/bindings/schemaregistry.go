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

// SchemaRecord is an auto generated low-level Go binding around an user-defined struct.
type SchemaRecord struct {
	Uid       [32]byte
	Resolver  common.Address
	Revocable bool
	Schema    string
}

// SchemaRegistryMetaData contains all meta data concerning the SchemaRegistry contract.
var SchemaRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"getSchema\",\"inputs\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structSchemaRecord\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"resolver\",\"type\":\"address\",\"internalType\":\"contractISchemaResolver\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"schema\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"register\",\"inputs\":[{\"name\":\"schema\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"resolver\",\"type\":\"address\",\"internalType\":\"contractISchemaResolver\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Registered\",\"inputs\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"registerer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"schema\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structSchemaRecord\",\"components\":[{\"name\":\"uid\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"resolver\",\"type\":\"address\",\"internalType\":\"contractISchemaResolver\"},{\"name\":\"revocable\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"schema\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"AlreadyExists\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b506107fe806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806360d7a27814610098578063a2ea7c6e146100b9575b600080fd5b6100826040518060400160405280600581526020017f312e332e3000000000000000000000000000000000000000000000000000000081525081565b60405161008f9190610473565b60405180910390f35b6100ab6100a636600461048d565b6100d9565b60405190815260200161008f565b6100cc6100c736600461053f565b61029d565b60405161008f9190610558565b60008060405180608001604052806000801b81526020018573ffffffffffffffffffffffffffffffffffffffff168152602001841515815260200187878080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201829052509390945250929350915061015b9050826103c5565b600081815260208190526040902054909150156101a4576040517f23369fa600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b80825260008181526020818152604091829020845181559084015160018201805493860151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00000000000000000000000000000000000000000090941673ffffffffffffffffffffffffffffffffffffffff9092169190911792909217909155606083015183919060028201906102409082610682565b509050503373ffffffffffffffffffffffffffffffffffffffff16817fd0b86852e21f9e5fa4bc3b0cff9757ffe243d50c4b43968a42202153d651ea5e8460405161028b9190610558565b60405180910390a39695505050505050565b604080516080810182526000808252602082018190529181019190915260608082015260008281526020818152604091829020825160808101845281548152600182015473ffffffffffffffffffffffffffffffffffffffff8116938201939093527401000000000000000000000000000000000000000090920460ff1615159282019290925260028201805491929160608401919061033c906105e0565b80601f0160208091040260200160405190810160405280929190818152602001828054610368906105e0565b80156103b55780601f1061038a576101008083540402835291602001916103b5565b820191906000526020600020905b81548152906001019060200180831161039857829003601f168201915b5050505050815250509050919050565b60008160600151826020015183604001516040516020016103e89392919061079c565b604051602081830303815290604052805190602001209050919050565b60005b83811015610420578181015183820152602001610408565b50506000910152565b60008151808452610441816020860160208601610405565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006104866020830184610429565b9392505050565b600080600080606085870312156104a357600080fd5b843567ffffffffffffffff808211156104bb57600080fd5b818701915087601f8301126104cf57600080fd5b8135818111156104de57600080fd5b8860208285010111156104f057600080fd5b6020928301965094505085013573ffffffffffffffffffffffffffffffffffffffff8116811461051f57600080fd5b91506040850135801515811461053457600080fd5b939692955090935050565b60006020828403121561055157600080fd5b5035919050565b602081528151602082015273ffffffffffffffffffffffffffffffffffffffff6020830151166040820152604082015115156060820152600060608301516080808401526105a960a0840182610429565b949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c908216806105f457607f821691505b60208210810361062d577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561067d57600081815260208120601f850160051c8101602086101561065a5750805b601f850160051c820191505b8181101561067957828155600101610666565b5050505b505050565b815167ffffffffffffffff81111561069c5761069c6105b1565b6106b0816106aa84546105e0565b84610633565b602080601f83116001811461070357600084156106cd5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555610679565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561075057888601518255948401946001909101908401610731565b508582101561078c57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b600084516107ae818460208901610405565b60609490941b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169190930190815290151560f81b60148201526015019291505056fea164736f6c6343000813000a",
}

// SchemaRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use SchemaRegistryMetaData.ABI instead.
var SchemaRegistryABI = SchemaRegistryMetaData.ABI

// SchemaRegistryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use SchemaRegistryMetaData.Bin instead.
var SchemaRegistryBin = SchemaRegistryMetaData.Bin

// DeploySchemaRegistry deploys a new Ethereum contract, binding an instance of SchemaRegistry to it.
func DeploySchemaRegistry(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *SchemaRegistry, error) {
	parsed, err := SchemaRegistryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(SchemaRegistryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SchemaRegistry{SchemaRegistryCaller: SchemaRegistryCaller{contract: contract}, SchemaRegistryTransactor: SchemaRegistryTransactor{contract: contract}, SchemaRegistryFilterer: SchemaRegistryFilterer{contract: contract}}, nil
}

// SchemaRegistry is an auto generated Go binding around an Ethereum contract.
type SchemaRegistry struct {
	SchemaRegistryCaller     // Read-only binding to the contract
	SchemaRegistryTransactor // Write-only binding to the contract
	SchemaRegistryFilterer   // Log filterer for contract events
}

// SchemaRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type SchemaRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SchemaRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SchemaRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SchemaRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SchemaRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SchemaRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SchemaRegistrySession struct {
	Contract     *SchemaRegistry   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SchemaRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SchemaRegistryCallerSession struct {
	Contract *SchemaRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// SchemaRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SchemaRegistryTransactorSession struct {
	Contract     *SchemaRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// SchemaRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type SchemaRegistryRaw struct {
	Contract *SchemaRegistry // Generic contract binding to access the raw methods on
}

// SchemaRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SchemaRegistryCallerRaw struct {
	Contract *SchemaRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// SchemaRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SchemaRegistryTransactorRaw struct {
	Contract *SchemaRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSchemaRegistry creates a new instance of SchemaRegistry, bound to a specific deployed contract.
func NewSchemaRegistry(address common.Address, backend bind.ContractBackend) (*SchemaRegistry, error) {
	contract, err := bindSchemaRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistry{SchemaRegistryCaller: SchemaRegistryCaller{contract: contract}, SchemaRegistryTransactor: SchemaRegistryTransactor{contract: contract}, SchemaRegistryFilterer: SchemaRegistryFilterer{contract: contract}}, nil
}

// NewSchemaRegistryCaller creates a new read-only instance of SchemaRegistry, bound to a specific deployed contract.
func NewSchemaRegistryCaller(address common.Address, caller bind.ContractCaller) (*SchemaRegistryCaller, error) {
	contract, err := bindSchemaRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistryCaller{contract: contract}, nil
}

// NewSchemaRegistryTransactor creates a new write-only instance of SchemaRegistry, bound to a specific deployed contract.
func NewSchemaRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*SchemaRegistryTransactor, error) {
	contract, err := bindSchemaRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistryTransactor{contract: contract}, nil
}

// NewSchemaRegistryFilterer creates a new log filterer instance of SchemaRegistry, bound to a specific deployed contract.
func NewSchemaRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*SchemaRegistryFilterer, error) {
	contract, err := bindSchemaRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistryFilterer{contract: contract}, nil
}

// bindSchemaRegistry binds a generic wrapper to an already deployed contract.
func bindSchemaRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SchemaRegistryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SchemaRegistry *SchemaRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SchemaRegistry.Contract.SchemaRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SchemaRegistry *SchemaRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.SchemaRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SchemaRegistry *SchemaRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.SchemaRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SchemaRegistry *SchemaRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _SchemaRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SchemaRegistry *SchemaRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SchemaRegistry *SchemaRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.contract.Transact(opts, method, params...)
}

// GetSchema is a free data retrieval call binding the contract method 0xa2ea7c6e.
//
// Solidity: function getSchema(bytes32 uid) view returns((bytes32,address,bool,string))
func (_SchemaRegistry *SchemaRegistryCaller) GetSchema(opts *bind.CallOpts, uid [32]byte) (SchemaRecord, error) {
	var out []interface{}
	err := _SchemaRegistry.contract.Call(opts, &out, "getSchema", uid)

	if err != nil {
		return *new(SchemaRecord), err
	}

	out0 := *abi.ConvertType(out[0], new(SchemaRecord)).(*SchemaRecord)

	return out0, err

}

// GetSchema is a free data retrieval call binding the contract method 0xa2ea7c6e.
//
// Solidity: function getSchema(bytes32 uid) view returns((bytes32,address,bool,string))
func (_SchemaRegistry *SchemaRegistrySession) GetSchema(uid [32]byte) (SchemaRecord, error) {
	return _SchemaRegistry.Contract.GetSchema(&_SchemaRegistry.CallOpts, uid)
}

// GetSchema is a free data retrieval call binding the contract method 0xa2ea7c6e.
//
// Solidity: function getSchema(bytes32 uid) view returns((bytes32,address,bool,string))
func (_SchemaRegistry *SchemaRegistryCallerSession) GetSchema(uid [32]byte) (SchemaRecord, error) {
	return _SchemaRegistry.Contract.GetSchema(&_SchemaRegistry.CallOpts, uid)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SchemaRegistry *SchemaRegistryCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _SchemaRegistry.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SchemaRegistry *SchemaRegistrySession) Version() (string, error) {
	return _SchemaRegistry.Contract.Version(&_SchemaRegistry.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_SchemaRegistry *SchemaRegistryCallerSession) Version() (string, error) {
	return _SchemaRegistry.Contract.Version(&_SchemaRegistry.CallOpts)
}

// Register is a paid mutator transaction binding the contract method 0x60d7a278.
//
// Solidity: function register(string schema, address resolver, bool revocable) returns(bytes32)
func (_SchemaRegistry *SchemaRegistryTransactor) Register(opts *bind.TransactOpts, schema string, resolver common.Address, revocable bool) (*types.Transaction, error) {
	return _SchemaRegistry.contract.Transact(opts, "register", schema, resolver, revocable)
}

// Register is a paid mutator transaction binding the contract method 0x60d7a278.
//
// Solidity: function register(string schema, address resolver, bool revocable) returns(bytes32)
func (_SchemaRegistry *SchemaRegistrySession) Register(schema string, resolver common.Address, revocable bool) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.Register(&_SchemaRegistry.TransactOpts, schema, resolver, revocable)
}

// Register is a paid mutator transaction binding the contract method 0x60d7a278.
//
// Solidity: function register(string schema, address resolver, bool revocable) returns(bytes32)
func (_SchemaRegistry *SchemaRegistryTransactorSession) Register(schema string, resolver common.Address, revocable bool) (*types.Transaction, error) {
	return _SchemaRegistry.Contract.Register(&_SchemaRegistry.TransactOpts, schema, resolver, revocable)
}

// SchemaRegistryRegisteredIterator is returned from FilterRegistered and is used to iterate over the raw logs and unpacked data for Registered events raised by the SchemaRegistry contract.
type SchemaRegistryRegisteredIterator struct {
	Event *SchemaRegistryRegistered // Event containing the contract specifics and raw log

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
func (it *SchemaRegistryRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SchemaRegistryRegistered)
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
		it.Event = new(SchemaRegistryRegistered)
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
func (it *SchemaRegistryRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SchemaRegistryRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SchemaRegistryRegistered represents a Registered event raised by the SchemaRegistry contract.
type SchemaRegistryRegistered struct {
	Uid        [32]byte
	Registerer common.Address
	Schema     SchemaRecord
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterRegistered is a free log retrieval operation binding the contract event 0xd0b86852e21f9e5fa4bc3b0cff9757ffe243d50c4b43968a42202153d651ea5e.
//
// Solidity: event Registered(bytes32 indexed uid, address indexed registerer, (bytes32,address,bool,string) schema)
func (_SchemaRegistry *SchemaRegistryFilterer) FilterRegistered(opts *bind.FilterOpts, uid [][32]byte, registerer []common.Address) (*SchemaRegistryRegisteredIterator, error) {

	var uidRule []interface{}
	for _, uidItem := range uid {
		uidRule = append(uidRule, uidItem)
	}
	var registererRule []interface{}
	for _, registererItem := range registerer {
		registererRule = append(registererRule, registererItem)
	}

	logs, sub, err := _SchemaRegistry.contract.FilterLogs(opts, "Registered", uidRule, registererRule)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistryRegisteredIterator{contract: _SchemaRegistry.contract, event: "Registered", logs: logs, sub: sub}, nil
}

// WatchRegistered is a free log subscription operation binding the contract event 0xd0b86852e21f9e5fa4bc3b0cff9757ffe243d50c4b43968a42202153d651ea5e.
//
// Solidity: event Registered(bytes32 indexed uid, address indexed registerer, (bytes32,address,bool,string) schema)
func (_SchemaRegistry *SchemaRegistryFilterer) WatchRegistered(opts *bind.WatchOpts, sink chan<- *SchemaRegistryRegistered, uid [][32]byte, registerer []common.Address) (event.Subscription, error) {

	var uidRule []interface{}
	for _, uidItem := range uid {
		uidRule = append(uidRule, uidItem)
	}
	var registererRule []interface{}
	for _, registererItem := range registerer {
		registererRule = append(registererRule, registererItem)
	}

	logs, sub, err := _SchemaRegistry.contract.WatchLogs(opts, "Registered", uidRule, registererRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SchemaRegistryRegistered)
				if err := _SchemaRegistry.contract.UnpackLog(event, "Registered", log); err != nil {
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

// ParseRegistered is a log parse operation binding the contract event 0xd0b86852e21f9e5fa4bc3b0cff9757ffe243d50c4b43968a42202153d651ea5e.
//
// Solidity: event Registered(bytes32 indexed uid, address indexed registerer, (bytes32,address,bool,string) schema)
func (_SchemaRegistry *SchemaRegistryFilterer) ParseRegistered(log types.Log) (*SchemaRegistryRegistered, error) {
	event := new(SchemaRegistryRegistered)
	if err := _SchemaRegistry.contract.UnpackLog(event, "Registered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
