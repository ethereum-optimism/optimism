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
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AlreadyExists\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"registerer\",\"type\":\"address\"}],\"name\":\"Registered\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"}],\"name\":\"getSchema\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"uid\",\"type\":\"bytes32\"},{\"internalType\":\"contractISchemaResolver\",\"name\":\"resolver\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"schema\",\"type\":\"string\"}],\"internalType\":\"structSchemaRecord\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"schema\",\"type\":\"string\"},{\"internalType\":\"contractISchemaResolver\",\"name\":\"resolver\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"revocable\",\"type\":\"bool\"}],\"name\":\"register\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b5060016080819052600060a081905260c082905281610b166100478339600060fe0152600060d50152600060ac0152610b166000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c806354fd4d501461004657806360d7a27814610064578063a2ea7c6e14610085575b600080fd5b61004e6100a5565b60405161005b9190610604565b60405180910390f35b61007761007236600461061e565b610148565b60405190815260200161005b565b6100986100933660046106d0565b6102f1565b60405161005b91906106e9565b60606100d07f0000000000000000000000000000000000000000000000000000000000000000610419565b6100f97f0000000000000000000000000000000000000000000000000000000000000000610419565b6101227f0000000000000000000000000000000000000000000000000000000000000000610419565b6040516020016101349392919061073a565b604051602081830303815290604052905090565b60008060405180608001604052806000801b81526020018573ffffffffffffffffffffffffffffffffffffffff168152602001841515815260200187878080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920182905250939094525092935091506101ca905082610556565b60008181526020819052604090205490915015610213576040517f23369fa600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b80825260008181526020818152604091829020845181559084015160018201805493860151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff00000000000000000000000000000000000000000090941673ffffffffffffffffffffffffffffffffffffffff9092169190911792909217909155606083015183919060028201906102af9082610881565b50506040513381528291507f7d917fcbc9a29a9705ff9936ffa599500e4fd902e4486bae317414fe967b307c9060200160405180910390a29695505050505050565b604080516080810182526000808252602082018190529181019190915260608082015260008281526020818152604091829020825160808101845281548152600182015473ffffffffffffffffffffffffffffffffffffffff8116938201939093527401000000000000000000000000000000000000000090920460ff16151592820192909252600282018054919291606084019190610390906107df565b80601f01602080910402602001604051908101604052809291908181526020018280546103bc906107df565b80156104095780601f106103de57610100808354040283529160200191610409565b820191906000526020600020905b8154815290600101906020018083116103ec57829003601f168201915b5050505050815250509050919050565b60608160000361045c57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156104865780610470816109ca565b915061047f9050600a83610a31565b9150610460565b60008167ffffffffffffffff8111156104a1576104a16107b0565b6040519080825280601f01601f1916602001820160405280156104cb576020820181803683370190505b5090505b841561054e576104e0600183610a45565b91506104ed600a86610a5e565b6104f8906030610a72565b60f81b81838151811061050d5761050d610a85565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610547600a86610a31565b94506104cf565b949350505050565b600081606001518260200151836040015160405160200161057993929190610ab4565b604051602081830303815290604052805190602001209050919050565b60005b838110156105b1578181015183820152602001610599565b50506000910152565b600081518084526105d2816020860160208601610596565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061061760208301846105ba565b9392505050565b6000806000806060858703121561063457600080fd5b843567ffffffffffffffff8082111561064c57600080fd5b818701915087601f83011261066057600080fd5b81358181111561066f57600080fd5b88602082850101111561068157600080fd5b6020928301965094505085013573ffffffffffffffffffffffffffffffffffffffff811681146106b057600080fd5b9150604085013580151581146106c557600080fd5b939692955090935050565b6000602082840312156106e257600080fd5b5035919050565b602081528151602082015273ffffffffffffffffffffffffffffffffffffffff60208301511660408201526040820151151560608201526000606083015160808084015261054e60a08401826105ba565b6000845161074c818460208901610596565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610788816001850160208a01610596565b600192019182015283516107a3816002840160208801610596565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c908216806107f357607f821691505b60208210810361082c577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561087c57600081815260208120601f850160051c810160208610156108595750805b601f850160051c820191505b8181101561087857828155600101610865565b5050505b505050565b815167ffffffffffffffff81111561089b5761089b6107b0565b6108af816108a984546107df565b84610832565b602080601f83116001811461090257600084156108cc5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555610878565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561094f57888601518255948401946001909101908401610930565b508582101561098b57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036109fb576109fb61099b565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610a4057610a40610a02565b500490565b81810381811115610a5857610a5861099b565b92915050565b600082610a6d57610a6d610a02565b500690565b80820180821115610a5857610a5861099b565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008451610ac6818460208901610596565b60609490941b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169190930190815290151560f81b60148201526015019291505056fea164736f6c6343000813000a",
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
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterRegistered is a free log retrieval operation binding the contract event 0x7d917fcbc9a29a9705ff9936ffa599500e4fd902e4486bae317414fe967b307c.
//
// Solidity: event Registered(bytes32 indexed uid, address registerer)
func (_SchemaRegistry *SchemaRegistryFilterer) FilterRegistered(opts *bind.FilterOpts, uid [][32]byte) (*SchemaRegistryRegisteredIterator, error) {

	var uidRule []interface{}
	for _, uidItem := range uid {
		uidRule = append(uidRule, uidItem)
	}

	logs, sub, err := _SchemaRegistry.contract.FilterLogs(opts, "Registered", uidRule)
	if err != nil {
		return nil, err
	}
	return &SchemaRegistryRegisteredIterator{contract: _SchemaRegistry.contract, event: "Registered", logs: logs, sub: sub}, nil
}

// WatchRegistered is a free log subscription operation binding the contract event 0x7d917fcbc9a29a9705ff9936ffa599500e4fd902e4486bae317414fe967b307c.
//
// Solidity: event Registered(bytes32 indexed uid, address registerer)
func (_SchemaRegistry *SchemaRegistryFilterer) WatchRegistered(opts *bind.WatchOpts, sink chan<- *SchemaRegistryRegistered, uid [][32]byte) (event.Subscription, error) {

	var uidRule []interface{}
	for _, uidItem := range uid {
		uidRule = append(uidRule, uidItem)
	}

	logs, sub, err := _SchemaRegistry.contract.WatchLogs(opts, "Registered", uidRule)
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

// ParseRegistered is a log parse operation binding the contract event 0x7d917fcbc9a29a9705ff9936ffa599500e4fd902e4486bae317414fe967b307c.
//
// Solidity: event Registered(bytes32 indexed uid, address registerer)
func (_SchemaRegistry *SchemaRegistryFilterer) ParseRegistered(log types.Log) (*SchemaRegistryRegistered, error) {
	event := new(SchemaRegistryRegistered)
	if err := _SchemaRegistry.contract.UnpackLog(event, "Registered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
