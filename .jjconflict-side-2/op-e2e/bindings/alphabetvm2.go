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

// AlphabetVM2MetaData contains all meta data concerning the AlphabetVM2 contract.
var AlphabetVM2MetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"oracle\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"step\",\"inputs\":[{\"name\":\"_stateData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"postState_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051610c23380380610c2383398101604081905261002f91610090565b608081905260405161004090610083565b604051809103906000f08015801561005c573d6000803e3d6000fd5b50600080546001600160a01b0319166001600160a01b0392909216919091179055506100a9565b61065c806105c783390190565b6000602082840312156100a257600080fd5b5051919050565b6080516105046100c3600039600060af01526105046000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80637dc0d1d01461003b578063e14ced3214610085575b600080fd5b60005461005b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b610098610093366004610395565b6100a6565b60405190815260200161007c565b600080600060087f0000000000000000000000000000000000000000000000000000000000000000901b600889896040516100e2929190610409565b6040518091039020901b036101d9576000805473ffffffffffffffffffffffffffffffffffffffff1663e03110e161011b60048861029f565b6040517fffffffff0000000000000000000000000000000000000000000000000000000060e084901b1681526004810191909152600060248201526044016040805180830381865afa158015610175573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906101999190610419565b50640ffffffff0607c82901c169350905063ffffffff608082901c1660006101c38a8c018c61043d565b90506101cf8582610485565b9350505050610206565b6101e58789018961049d565b9092509050816101f4816104bf565b9250508080610202906104bf565b9150505b6040805160208101849052908101829052606001604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905280516020909101207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001798975050505050505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff831617610345818360408051600093845233602052918152606090922091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b9392505050565b60008083601f84011261035e57600080fd5b50813567ffffffffffffffff81111561037657600080fd5b60208301915083602082850101111561038e57600080fd5b9250929050565b6000806000806000606086880312156103ad57600080fd5b853567ffffffffffffffff808211156103c557600080fd5b6103d189838a0161034c565b909750955060208801359150808211156103ea57600080fd5b506103f78882890161034c565b96999598509660400135949350505050565b8183823760009101908152919050565b6000806040838503121561042c57600080fd5b505080516020909101519092909150565b60006020828403121561044f57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561049857610498610456565b500190565b600080604083850312156104b057600080fd5b50508035926020909101359150565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036104f0576104f0610456565b506001019056fea164736f6c634300080f000a608060405234801561001057600080fd5b5061063c806100206000396000f3fe608060405234801561001057600080fd5b50600436106100725760003560e01c8063e03110e111610050578063e03110e114610106578063e15926111461012e578063fef2b4ed1461014357600080fd5b806352f0f3ad1461007757806361238bde1461009d5780638542cf50146100c8575b600080fd5b61008a6100853660046104df565b610163565b6040519081526020015b60405180910390f35b61008a6100ab36600461051a565b600160209081526000928352604080842090915290825290205481565b6100f66100d636600461051a565b600260209081526000928352604080842090915290825290205460ff1681565b6040519015158152602001610094565b61011961011436600461051a565b610238565b60408051928352602083019190915201610094565b61014161013c36600461053c565b610329565b005b61008a6101513660046105b8565b60006020819052908152604090205481565b600061016f8686610432565b905061017c836008610600565b8211806101895750602083115b156101c0576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000602081815260c085901b82526008959095528251828252600286526040808320858452875280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660019081179091558484528752808320948352938652838220558181529384905292205592915050565b6000828152600260209081526040808320848452909152812054819060ff166102c1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f7072652d696d616765206d757374206578697374000000000000000000000000604482015260640160405180910390fd5b50600083815260208181526040909120546102dd816008610600565b6102e8856020610600565b1061030657836102f9826008610600565b6103039190610618565b91505b506000938452600160209081526040808620948652939052919092205492909150565b604435600080600883018611156103485763fe2549876000526004601cfd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001760008181526002602090815260408083208b8452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915584845282528083209a83529981528982209390935590815290819052959095209190915550505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8316176104d8818360408051600093845233602052918152606090922091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b9392505050565b600080600080600060a086880312156104f757600080fd5b505083359560208501359550604085013594606081013594506080013592509050565b6000806040838503121561052d57600080fd5b50508035926020909101359150565b60008060006040848603121561055157600080fd5b83359250602084013567ffffffffffffffff8082111561057057600080fd5b818601915086601f83011261058457600080fd5b81358181111561059357600080fd5b8760208285010111156105a557600080fd5b6020830194508093505050509250925092565b6000602082840312156105ca57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115610613576106136105d1565b500190565b60008282101561062a5761062a6105d1565b50039056fea164736f6c634300080f000a",
}

// AlphabetVM2ABI is the input ABI used to generate the binding from.
// Deprecated: Use AlphabetVM2MetaData.ABI instead.
var AlphabetVM2ABI = AlphabetVM2MetaData.ABI

// AlphabetVM2Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use AlphabetVM2MetaData.Bin instead.
var AlphabetVM2Bin = AlphabetVM2MetaData.Bin

// DeployAlphabetVM2 deploys a new Ethereum contract, binding an instance of AlphabetVM2 to it.
func DeployAlphabetVM2(auth *bind.TransactOpts, backend bind.ContractBackend, _absolutePrestate [32]byte) (common.Address, *types.Transaction, *AlphabetVM2, error) {
	parsed, err := AlphabetVM2MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(AlphabetVM2Bin), backend, _absolutePrestate)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &AlphabetVM2{AlphabetVM2Caller: AlphabetVM2Caller{contract: contract}, AlphabetVM2Transactor: AlphabetVM2Transactor{contract: contract}, AlphabetVM2Filterer: AlphabetVM2Filterer{contract: contract}}, nil
}

// AlphabetVM2 is an auto generated Go binding around an Ethereum contract.
type AlphabetVM2 struct {
	AlphabetVM2Caller     // Read-only binding to the contract
	AlphabetVM2Transactor // Write-only binding to the contract
	AlphabetVM2Filterer   // Log filterer for contract events
}

// AlphabetVM2Caller is an auto generated read-only Go binding around an Ethereum contract.
type AlphabetVM2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVM2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type AlphabetVM2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVM2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AlphabetVM2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AlphabetVM2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AlphabetVM2Session struct {
	Contract     *AlphabetVM2      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AlphabetVM2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AlphabetVM2CallerSession struct {
	Contract *AlphabetVM2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// AlphabetVM2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AlphabetVM2TransactorSession struct {
	Contract     *AlphabetVM2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// AlphabetVM2Raw is an auto generated low-level Go binding around an Ethereum contract.
type AlphabetVM2Raw struct {
	Contract *AlphabetVM2 // Generic contract binding to access the raw methods on
}

// AlphabetVM2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AlphabetVM2CallerRaw struct {
	Contract *AlphabetVM2Caller // Generic read-only contract binding to access the raw methods on
}

// AlphabetVM2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AlphabetVM2TransactorRaw struct {
	Contract *AlphabetVM2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewAlphabetVM2 creates a new instance of AlphabetVM2, bound to a specific deployed contract.
func NewAlphabetVM2(address common.Address, backend bind.ContractBackend) (*AlphabetVM2, error) {
	contract, err := bindAlphabetVM2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AlphabetVM2{AlphabetVM2Caller: AlphabetVM2Caller{contract: contract}, AlphabetVM2Transactor: AlphabetVM2Transactor{contract: contract}, AlphabetVM2Filterer: AlphabetVM2Filterer{contract: contract}}, nil
}

// NewAlphabetVM2Caller creates a new read-only instance of AlphabetVM2, bound to a specific deployed contract.
func NewAlphabetVM2Caller(address common.Address, caller bind.ContractCaller) (*AlphabetVM2Caller, error) {
	contract, err := bindAlphabetVM2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AlphabetVM2Caller{contract: contract}, nil
}

// NewAlphabetVM2Transactor creates a new write-only instance of AlphabetVM2, bound to a specific deployed contract.
func NewAlphabetVM2Transactor(address common.Address, transactor bind.ContractTransactor) (*AlphabetVM2Transactor, error) {
	contract, err := bindAlphabetVM2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AlphabetVM2Transactor{contract: contract}, nil
}

// NewAlphabetVM2Filterer creates a new log filterer instance of AlphabetVM2, bound to a specific deployed contract.
func NewAlphabetVM2Filterer(address common.Address, filterer bind.ContractFilterer) (*AlphabetVM2Filterer, error) {
	contract, err := bindAlphabetVM2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AlphabetVM2Filterer{contract: contract}, nil
}

// bindAlphabetVM2 binds a generic wrapper to an already deployed contract.
func bindAlphabetVM2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AlphabetVM2ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AlphabetVM2 *AlphabetVM2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AlphabetVM2.Contract.AlphabetVM2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AlphabetVM2 *AlphabetVM2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AlphabetVM2.Contract.AlphabetVM2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AlphabetVM2 *AlphabetVM2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AlphabetVM2.Contract.AlphabetVM2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AlphabetVM2 *AlphabetVM2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AlphabetVM2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AlphabetVM2 *AlphabetVM2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AlphabetVM2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AlphabetVM2 *AlphabetVM2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AlphabetVM2.Contract.contract.Transact(opts, method, params...)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM2 *AlphabetVM2Caller) Oracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AlphabetVM2.contract.Call(opts, &out, "oracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM2 *AlphabetVM2Session) Oracle() (common.Address, error) {
	return _AlphabetVM2.Contract.Oracle(&_AlphabetVM2.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
func (_AlphabetVM2 *AlphabetVM2CallerSession) Oracle() (common.Address, error) {
	return _AlphabetVM2.Contract.Oracle(&_AlphabetVM2.CallOpts)
}

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM2 *AlphabetVM2Caller) Step(opts *bind.CallOpts, _stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _AlphabetVM2.contract.Call(opts, &out, "step", _stateData, arg1, _localContext)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM2 *AlphabetVM2Session) Step(_stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	return _AlphabetVM2.Contract.Step(&_AlphabetVM2.CallOpts, _stateData, arg1, _localContext)
}

// Step is a free data retrieval call binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes , bytes32 _localContext) view returns(bytes32 postState_)
func (_AlphabetVM2 *AlphabetVM2CallerSession) Step(_stateData []byte, arg1 []byte, _localContext [32]byte) ([32]byte, error) {
	return _AlphabetVM2.Contract.Step(&_AlphabetVM2.CallOpts, _stateData, arg1, _localContext)
}
