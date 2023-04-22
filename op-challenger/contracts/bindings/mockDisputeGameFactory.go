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

// MockDisputeGameFactoryMetaData contains all meta data concerning the MockDisputeGameFactory contract.
var MockDisputeGameFactoryMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"disputeProxy\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"enumGameType\",\"name\":\"gameType\",\"type\":\"uint8\"},{\"indexed\":true,\"internalType\":\"Claim\",\"name\":\"rootClaim\",\"type\":\"bytes32\"}],\"name\":\"DisputeGameCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"enumGameType\",\"name\":\"gameType\",\"type\":\"uint8\"},{\"internalType\":\"Claim\",\"name\":\"rootClaim\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"create\",\"outputs\":[{\"internalType\":\"contractMockAttestationDisputeGame\",\"name\":\"mock\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"enumGameType\",\"name\":\"gameType\",\"type\":\"uint8\"},{\"internalType\":\"Claim\",\"name\":\"rootClaim\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"games\",\"outputs\":[{\"internalType\":\"contractIDisputeGame\",\"name\":\"_proxy\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506105d6806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80633142e55e1461003b578063c49d52711461006a575b600080fd5b61004e6100493660046101ec565b61007d565b6040516001600160a01b03909116815260200160405180910390f35b61004e6100783660046101ec565b61018b565b60008061008c83850185610279565b905084813360405161009d906101df565b92835260208301919091526001600160a01b03166040820152606001604051809103906000f0801580156100d5573d6000803e3d6000fd5b509150848660028111156100eb576100eb610292565b6040516001600160a01b038516907ffad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec90600090a4816000808888888860405160200161013a94939291906102a8565b60405160208183030381529060405280519060200120815260200190815260200160002060006101000a8154816001600160a01b0302191690836001600160a01b0316021790555050949350505050565b6000806000868686866040516020016101a794939291906102a8565b60408051601f19818403018152918152815160209283012083529082019290925201600020546001600160a01b031695945050505050565b6102b3806102ee83390190565b6000806000806060858703121561020257600080fd5b84356003811061021157600080fd5b935060208501359250604085013567ffffffffffffffff8082111561023557600080fd5b818701915087601f83011261024957600080fd5b81358181111561025857600080fd5b88602082850101111561026a57600080fd5b95989497505060200194505050565b60006020828403121561028b57600080fd5b5035919050565b634e487b7160e01b600052602160045260246000fd5b6000600386106102c857634e487b7160e01b600052602160045260246000fd5b8560f81b8252846001830152828460218401375060009101602101908152939250505056fe60c060405234801561001057600080fd5b506040516102b33803806102b383398101604081905261002f9161005f565b60809290925260a0526001600160a01b03166000908152602081905260409020805460ff191660011790556100a5565b60008060006060848603121561007457600080fd5b83516020850151604086015191945092506001600160a01b038116811461009a57600080fd5b809150509250925092565b60805160a0516101eb6100c86000396000608e0152600060c301526101eb6000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c806308b43a1914610051578063326f8195146100895780634a1890f0146100be578063b8b9c188146100e5575b600080fd5b61007461005f366004610113565b60006020819052908152604090205460ff1681565b60405190151581526020015b60405180910390f35b6100b07f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610080565b6100b07f000000000000000000000000000000000000000000000000000000000000000081565b6101116100f3366004610143565b5050336000908152602081905260409020805460ff19166001179055565b005b60006020828403121561012557600080fd5b81356001600160a01b038116811461013c57600080fd5b9392505050565b6000806020838503121561015657600080fd5b823567ffffffffffffffff8082111561016e57600080fd5b818501915085601f83011261018257600080fd5b81358181111561019157600080fd5b8660208285010111156101a357600080fd5b6020929092019691955090935050505056fea2646970667358221220a15f2b475fb3640846a10e20b1d980b5a93201e53321f773b79c483277a32aa364736f6c63430008130033a2646970667358221220b4c9410279382b1338bd3a1d99779f93e552ef22ee4c93df63943561dea0f18764736f6c63430008130033",
}

// MockDisputeGameFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use MockDisputeGameFactoryMetaData.ABI instead.
var MockDisputeGameFactoryABI = MockDisputeGameFactoryMetaData.ABI

// MockDisputeGameFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MockDisputeGameFactoryMetaData.Bin instead.
var MockDisputeGameFactoryBin = MockDisputeGameFactoryMetaData.Bin

// DeployMockDisputeGameFactory deploys a new Ethereum contract, binding an instance of MockDisputeGameFactory to it.
func DeployMockDisputeGameFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MockDisputeGameFactory, error) {
	parsed, err := MockDisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MockDisputeGameFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MockDisputeGameFactory{MockDisputeGameFactoryCaller: MockDisputeGameFactoryCaller{contract: contract}, MockDisputeGameFactoryTransactor: MockDisputeGameFactoryTransactor{contract: contract}, MockDisputeGameFactoryFilterer: MockDisputeGameFactoryFilterer{contract: contract}}, nil
}

// MockDisputeGameFactory is an auto generated Go binding around an Ethereum contract.
type MockDisputeGameFactory struct {
	MockDisputeGameFactoryCaller     // Read-only binding to the contract
	MockDisputeGameFactoryTransactor // Write-only binding to the contract
	MockDisputeGameFactoryFilterer   // Log filterer for contract events
}

// MockDisputeGameFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type MockDisputeGameFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockDisputeGameFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MockDisputeGameFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockDisputeGameFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MockDisputeGameFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MockDisputeGameFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MockDisputeGameFactorySession struct {
	Contract     *MockDisputeGameFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// MockDisputeGameFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MockDisputeGameFactoryCallerSession struct {
	Contract *MockDisputeGameFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// MockDisputeGameFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MockDisputeGameFactoryTransactorSession struct {
	Contract     *MockDisputeGameFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// MockDisputeGameFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type MockDisputeGameFactoryRaw struct {
	Contract *MockDisputeGameFactory // Generic contract binding to access the raw methods on
}

// MockDisputeGameFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MockDisputeGameFactoryCallerRaw struct {
	Contract *MockDisputeGameFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// MockDisputeGameFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MockDisputeGameFactoryTransactorRaw struct {
	Contract *MockDisputeGameFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMockDisputeGameFactory creates a new instance of MockDisputeGameFactory, bound to a specific deployed contract.
func NewMockDisputeGameFactory(address common.Address, backend bind.ContractBackend) (*MockDisputeGameFactory, error) {
	contract, err := bindMockDisputeGameFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MockDisputeGameFactory{MockDisputeGameFactoryCaller: MockDisputeGameFactoryCaller{contract: contract}, MockDisputeGameFactoryTransactor: MockDisputeGameFactoryTransactor{contract: contract}, MockDisputeGameFactoryFilterer: MockDisputeGameFactoryFilterer{contract: contract}}, nil
}

// NewMockDisputeGameFactoryCaller creates a new read-only instance of MockDisputeGameFactory, bound to a specific deployed contract.
func NewMockDisputeGameFactoryCaller(address common.Address, caller bind.ContractCaller) (*MockDisputeGameFactoryCaller, error) {
	contract, err := bindMockDisputeGameFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MockDisputeGameFactoryCaller{contract: contract}, nil
}

// NewMockDisputeGameFactoryTransactor creates a new write-only instance of MockDisputeGameFactory, bound to a specific deployed contract.
func NewMockDisputeGameFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*MockDisputeGameFactoryTransactor, error) {
	contract, err := bindMockDisputeGameFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MockDisputeGameFactoryTransactor{contract: contract}, nil
}

// NewMockDisputeGameFactoryFilterer creates a new log filterer instance of MockDisputeGameFactory, bound to a specific deployed contract.
func NewMockDisputeGameFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*MockDisputeGameFactoryFilterer, error) {
	contract, err := bindMockDisputeGameFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MockDisputeGameFactoryFilterer{contract: contract}, nil
}

// bindMockDisputeGameFactory binds a generic wrapper to an already deployed contract.
func bindMockDisputeGameFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MockDisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockDisputeGameFactory *MockDisputeGameFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockDisputeGameFactory.Contract.MockDisputeGameFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockDisputeGameFactory *MockDisputeGameFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.MockDisputeGameFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockDisputeGameFactory *MockDisputeGameFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.MockDisputeGameFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MockDisputeGameFactory *MockDisputeGameFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MockDisputeGameFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MockDisputeGameFactory *MockDisputeGameFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MockDisputeGameFactory *MockDisputeGameFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.contract.Transact(opts, method, params...)
}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 gameType, bytes32 rootClaim, bytes extraData) view returns(address _proxy)
func (_MockDisputeGameFactory *MockDisputeGameFactoryCaller) Games(opts *bind.CallOpts, gameType uint8, rootClaim [32]byte, extraData []byte) (common.Address, error) {
	var out []interface{}
	err := _MockDisputeGameFactory.contract.Call(opts, &out, "games", gameType, rootClaim, extraData)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 gameType, bytes32 rootClaim, bytes extraData) view returns(address _proxy)
func (_MockDisputeGameFactory *MockDisputeGameFactorySession) Games(gameType uint8, rootClaim [32]byte, extraData []byte) (common.Address, error) {
	return _MockDisputeGameFactory.Contract.Games(&_MockDisputeGameFactory.CallOpts, gameType, rootClaim, extraData)
}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 gameType, bytes32 rootClaim, bytes extraData) view returns(address _proxy)
func (_MockDisputeGameFactory *MockDisputeGameFactoryCallerSession) Games(gameType uint8, rootClaim [32]byte, extraData []byte) (common.Address, error) {
	return _MockDisputeGameFactory.Contract.Games(&_MockDisputeGameFactory.CallOpts, gameType, rootClaim, extraData)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 gameType, bytes32 rootClaim, bytes extraData) returns(address mock)
func (_MockDisputeGameFactory *MockDisputeGameFactoryTransactor) Create(opts *bind.TransactOpts, gameType uint8, rootClaim [32]byte, extraData []byte) (*types.Transaction, error) {
	return _MockDisputeGameFactory.contract.Transact(opts, "create", gameType, rootClaim, extraData)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 gameType, bytes32 rootClaim, bytes extraData) returns(address mock)
func (_MockDisputeGameFactory *MockDisputeGameFactorySession) Create(gameType uint8, rootClaim [32]byte, extraData []byte) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.Create(&_MockDisputeGameFactory.TransactOpts, gameType, rootClaim, extraData)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 gameType, bytes32 rootClaim, bytes extraData) returns(address mock)
func (_MockDisputeGameFactory *MockDisputeGameFactoryTransactorSession) Create(gameType uint8, rootClaim [32]byte, extraData []byte) (*types.Transaction, error) {
	return _MockDisputeGameFactory.Contract.Create(&_MockDisputeGameFactory.TransactOpts, gameType, rootClaim, extraData)
}

// MockDisputeGameFactoryDisputeGameCreatedIterator is returned from FilterDisputeGameCreated and is used to iterate over the raw logs and unpacked data for DisputeGameCreated events raised by the MockDisputeGameFactory contract.
type MockDisputeGameFactoryDisputeGameCreatedIterator struct {
	Event *MockDisputeGameFactoryDisputeGameCreated // Event containing the contract specifics and raw log

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
func (it *MockDisputeGameFactoryDisputeGameCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MockDisputeGameFactoryDisputeGameCreated)
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
		it.Event = new(MockDisputeGameFactoryDisputeGameCreated)
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
func (it *MockDisputeGameFactoryDisputeGameCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MockDisputeGameFactoryDisputeGameCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MockDisputeGameFactoryDisputeGameCreated represents a DisputeGameCreated event raised by the MockDisputeGameFactory contract.
type MockDisputeGameFactoryDisputeGameCreated struct {
	DisputeProxy common.Address
	GameType     uint8
	RootClaim    [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDisputeGameCreated is a free log retrieval operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_MockDisputeGameFactory *MockDisputeGameFactoryFilterer) FilterDisputeGameCreated(opts *bind.FilterOpts, disputeProxy []common.Address, gameType []uint8, rootClaim [][32]byte) (*MockDisputeGameFactoryDisputeGameCreatedIterator, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _MockDisputeGameFactory.contract.FilterLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return &MockDisputeGameFactoryDisputeGameCreatedIterator{contract: _MockDisputeGameFactory.contract, event: "DisputeGameCreated", logs: logs, sub: sub}, nil
}

// WatchDisputeGameCreated is a free log subscription operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_MockDisputeGameFactory *MockDisputeGameFactoryFilterer) WatchDisputeGameCreated(opts *bind.WatchOpts, sink chan<- *MockDisputeGameFactoryDisputeGameCreated, disputeProxy []common.Address, gameType []uint8, rootClaim [][32]byte) (event.Subscription, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _MockDisputeGameFactory.contract.WatchLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MockDisputeGameFactoryDisputeGameCreated)
				if err := _MockDisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
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

// ParseDisputeGameCreated is a log parse operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_MockDisputeGameFactory *MockDisputeGameFactoryFilterer) ParseDisputeGameCreated(log types.Log) (*MockDisputeGameFactoryDisputeGameCreated, error) {
	event := new(MockDisputeGameFactoryDisputeGameCreated)
	if err := _MockDisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
