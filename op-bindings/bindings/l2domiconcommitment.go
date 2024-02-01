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

// L2DomiconCommitmentMetaData contains all meta data concerning the L2DomiconCommitment contract.
var L2DomiconCommitmentMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"_otherCommitment\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"FinalizeSubmitCommitment\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"SendDACommitment\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOM\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_COMMITMENT\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"SetDom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_broadcaster\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"finalizeSubmitCommitment\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"indices\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherCommitment\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"submits\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a0604052600280546001600160a01b03191673779877a7b0d9e8603169ddbd7836e478b462478917905534801561003657600080fd5b50604051610fe9380380610fe98339810160408190526100559161021d565b6001600160a01b03811660805261006a610070565b5061024d565b600054600390610100900460ff16158015610092575060005460ff8083169116105b6100fa5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055610140734200000000000000000000000000000000000007734200000000000000000000000000000000000023610184565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b600054610100900460ff166101ef5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b60648201526084016100f1565b600580546001600160a01b039384166001600160a01b03199182161790915560068054929093169116179055565b60006020828403121561022f57600080fd5b81516001600160a01b038116811461024657600080fd5b9392505050565b608051610d73610276600039600081816103260152818161035c01526103e30152610d736000f3fe6080604052600436106100d25760003560e01c80636d8819891161007f578063927ede2d11610059578063927ede2d146102cc578063dcf36d57146102f7578063e996e9ac14610317578063fce1c9741461034a57600080fd5b80636d88198914610240578063777109f8146102a45780638129fc1c146102b757600080fd5b806354fd4d50116100b057806354fd4d50146101905780635fa4ad36146101e65780636a57f6b11461021357600080fd5b80633817ce86146100d75780633cb747bf146101285780635063e20714610155575b600080fd5b3480156100e357600080fd5b5060065473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561013457600080fd5b506005546100fe9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561016157600080fd5b50610182610170366004610902565b60046020526000908152604090205481565b60405190815260200161011f565b34801561019c57600080fd5b506101d96040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b60405161011f9190610991565b3480156101f257600080fd5b506006546100fe9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561021f57600080fd5b506002546100fe9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561024c57600080fd5b506102a261025b366004610902565b600280547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b005b6102a26102b23660046109ed565b61037e565b3480156102c357600080fd5b506102a26105dc565b3480156102d857600080fd5b5060055473ffffffffffffffffffffffffffffffffffffffff166100fe565b34801561030357600080fd5b506101d9610312366004610aa1565b61074e565b34801561032357600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006100fe565b34801561035657600080fd5b506100fe7f000000000000000000000000000000000000000000000000000000000000000081565b60055473ffffffffffffffffffffffffffffffffffffffff163314801561046d5750600554604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000008116931691636e296e459160048083019260209291908290030181865afa158015610431573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104559190610acd565b73ffffffffffffffffffffffffffffffffffffffff16145b610524576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20746865206f7468657220636f60648201527f6d6d69746d656e74000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff851660009081526003602090815260408083208c8452909152902061055f828483610bbb565b508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff167f9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea8b8b8b898989896040516105c99796959493929190610d1f565b60405180910390a3505050505050505050565b600054600390610100900460ff161580156105fe575060005460ff8083169116105b61068a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161051b565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556106ed7342000000000000000000000000000000000000077342000000000000000000000000000000000000236107f3565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b60036020908152600092835260408084209091529082529020805461077290610b19565b80601f016020809104026020016040519081016040528092919081815260200182805461079e90610b19565b80156107eb5780601f106107c0576101008083540402835291602001916107eb565b820191906000526020600020905b8154815290600101906020018083116107ce57829003601f168201915b505050505081565b600054610100900460ff1661088a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161051b565b6005805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560068054929093169116179055565b73ffffffffffffffffffffffffffffffffffffffff811681146108ff57600080fd5b50565b60006020828403121561091457600080fd5b813561091f816108dd565b9392505050565b6000815180845260005b8181101561094c57602081850181015186830182015201610930565b8181111561095e576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061091f6020830184610926565b60008083601f8401126109b657600080fd5b50813567ffffffffffffffff8111156109ce57600080fd5b6020830191508360208285010111156109e657600080fd5b9250929050565b600080600080600080600080600060e08a8c031215610a0b57600080fd5b8935985060208a0135975060408a0135965060608a0135610a2b816108dd565b955060808a0135610a3b816108dd565b945060a08a013567ffffffffffffffff80821115610a5857600080fd5b610a648d838e016109a4565b909650945060c08c0135915080821115610a7d57600080fd5b50610a8a8c828d016109a4565b915080935050809150509295985092959850929598565b60008060408385031215610ab457600080fd5b8235610abf816108dd565b946020939093013593505050565b600060208284031215610adf57600080fd5b815161091f816108dd565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c90821680610b2d57607f821691505b602082108103610b66577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f821115610bb657600081815260208120601f850160051c81016020861015610b935750805b601f850160051c820191505b81811015610bb257828155600101610b9f565b5050505b505050565b67ffffffffffffffff831115610bd357610bd3610aea565b610be783610be18354610b19565b83610b6c565b6000601f841160018114610c395760008515610c035750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600387901b1c1916600186901b178355610ccf565b6000838152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0861690835b82811015610c885786850135825560209485019460019092019101610c68565b5086821015610cc3577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88860031b161c19848701351681555b505060018560011b0183555b5050505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b87815286602082015285604082015260a060608201526000610d4560a083018688610cd6565b8281036080840152610d58818587610cd6565b9a995050505050505050505056fea164736f6c634300080f000a",
}

// L2DomiconCommitmentABI is the input ABI used to generate the binding from.
// Deprecated: Use L2DomiconCommitmentMetaData.ABI instead.
var L2DomiconCommitmentABI = L2DomiconCommitmentMetaData.ABI

// L2DomiconCommitmentBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2DomiconCommitmentMetaData.Bin instead.
var L2DomiconCommitmentBin = L2DomiconCommitmentMetaData.Bin

// DeployL2DomiconCommitment deploys a new Ethereum contract, binding an instance of L2DomiconCommitment to it.
func DeployL2DomiconCommitment(auth *bind.TransactOpts, backend bind.ContractBackend, _otherCommitment common.Address) (common.Address, *types.Transaction, *L2DomiconCommitment, error) {
	parsed, err := L2DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2DomiconCommitmentBin), backend, _otherCommitment)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2DomiconCommitment{L2DomiconCommitmentCaller: L2DomiconCommitmentCaller{contract: contract}, L2DomiconCommitmentTransactor: L2DomiconCommitmentTransactor{contract: contract}, L2DomiconCommitmentFilterer: L2DomiconCommitmentFilterer{contract: contract}}, nil
}

// L2DomiconCommitment is an auto generated Go binding around an Ethereum contract.
type L2DomiconCommitment struct {
	L2DomiconCommitmentCaller     // Read-only binding to the contract
	L2DomiconCommitmentTransactor // Write-only binding to the contract
	L2DomiconCommitmentFilterer   // Log filterer for contract events
}

// L2DomiconCommitmentCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2DomiconCommitmentCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconCommitmentTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2DomiconCommitmentTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconCommitmentFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2DomiconCommitmentFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconCommitmentSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2DomiconCommitmentSession struct {
	Contract     *L2DomiconCommitment // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// L2DomiconCommitmentCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2DomiconCommitmentCallerSession struct {
	Contract *L2DomiconCommitmentCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// L2DomiconCommitmentTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2DomiconCommitmentTransactorSession struct {
	Contract     *L2DomiconCommitmentTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// L2DomiconCommitmentRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2DomiconCommitmentRaw struct {
	Contract *L2DomiconCommitment // Generic contract binding to access the raw methods on
}

// L2DomiconCommitmentCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2DomiconCommitmentCallerRaw struct {
	Contract *L2DomiconCommitmentCaller // Generic read-only contract binding to access the raw methods on
}

// L2DomiconCommitmentTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2DomiconCommitmentTransactorRaw struct {
	Contract *L2DomiconCommitmentTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2DomiconCommitment creates a new instance of L2DomiconCommitment, bound to a specific deployed contract.
func NewL2DomiconCommitment(address common.Address, backend bind.ContractBackend) (*L2DomiconCommitment, error) {
	contract, err := bindL2DomiconCommitment(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitment{L2DomiconCommitmentCaller: L2DomiconCommitmentCaller{contract: contract}, L2DomiconCommitmentTransactor: L2DomiconCommitmentTransactor{contract: contract}, L2DomiconCommitmentFilterer: L2DomiconCommitmentFilterer{contract: contract}}, nil
}

// NewL2DomiconCommitmentCaller creates a new read-only instance of L2DomiconCommitment, bound to a specific deployed contract.
func NewL2DomiconCommitmentCaller(address common.Address, caller bind.ContractCaller) (*L2DomiconCommitmentCaller, error) {
	contract, err := bindL2DomiconCommitment(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentCaller{contract: contract}, nil
}

// NewL2DomiconCommitmentTransactor creates a new write-only instance of L2DomiconCommitment, bound to a specific deployed contract.
func NewL2DomiconCommitmentTransactor(address common.Address, transactor bind.ContractTransactor) (*L2DomiconCommitmentTransactor, error) {
	contract, err := bindL2DomiconCommitment(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentTransactor{contract: contract}, nil
}

// NewL2DomiconCommitmentFilterer creates a new log filterer instance of L2DomiconCommitment, bound to a specific deployed contract.
func NewL2DomiconCommitmentFilterer(address common.Address, filterer bind.ContractFilterer) (*L2DomiconCommitmentFilterer, error) {
	contract, err := bindL2DomiconCommitment(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentFilterer{contract: contract}, nil
}

// bindL2DomiconCommitment binds a generic wrapper to an already deployed contract.
func bindL2DomiconCommitment(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L2DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2DomiconCommitment *L2DomiconCommitmentRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2DomiconCommitment.Contract.L2DomiconCommitmentCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2DomiconCommitment *L2DomiconCommitmentRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.L2DomiconCommitmentTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2DomiconCommitment *L2DomiconCommitmentRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.L2DomiconCommitmentTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2DomiconCommitment *L2DomiconCommitmentCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2DomiconCommitment.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.contract.Transact(opts, method, params...)
}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) DOM(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "DOM")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) DOM() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DOM(&_L2DomiconCommitment.CallOpts)
}

// DOM is a free data retrieval call binding the contract method 0x6a57f6b1.
//
// Solidity: function DOM() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) DOM() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DOM(&_L2DomiconCommitment.CallOpts)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) DOMICONNODE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "DOMICON_NODE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) DOMICONNODE() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DOMICONNODE(&_L2DomiconCommitment.CallOpts)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) DOMICONNODE() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DOMICONNODE(&_L2DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) MESSENGER() (common.Address, error) {
	return _L2DomiconCommitment.Contract.MESSENGER(&_L2DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) MESSENGER() (common.Address, error) {
	return _L2DomiconCommitment.Contract.MESSENGER(&_L2DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) OTHERCOMMITMENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "OTHER_COMMITMENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L2DomiconCommitment.Contract.OTHERCOMMITMENT(&_L2DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L2DomiconCommitment.Contract.OTHERCOMMITMENT(&_L2DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) DomiconNode(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "domiconNode")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) DomiconNode() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DomiconNode(&_L2DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) DomiconNode() (common.Address, error) {
	return _L2DomiconCommitment.Contract.DomiconNode(&_L2DomiconCommitment.CallOpts)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) Indices(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "indices", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L2DomiconCommitment.Contract.Indices(&_L2DomiconCommitment.CallOpts, arg0)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L2DomiconCommitment.Contract.Indices(&_L2DomiconCommitment.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Messenger() (common.Address, error) {
	return _L2DomiconCommitment.Contract.Messenger(&_L2DomiconCommitment.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) Messenger() (common.Address, error) {
	return _L2DomiconCommitment.Contract.Messenger(&_L2DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) OtherCommitment(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "otherCommitment")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) OtherCommitment() (common.Address, error) {
	return _L2DomiconCommitment.Contract.OtherCommitment(&_L2DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) OtherCommitment() (common.Address, error) {
	return _L2DomiconCommitment.Contract.OtherCommitment(&_L2DomiconCommitment.CallOpts)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) Submits(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "submits", arg0, arg1)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Submits(arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	return _L2DomiconCommitment.Contract.Submits(&_L2DomiconCommitment.CallOpts, arg0, arg1)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(bytes)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) Submits(arg0 common.Address, arg1 *big.Int) ([]byte, error) {
	return _L2DomiconCommitment.Contract.Submits(&_L2DomiconCommitment.CallOpts, arg0, arg1)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Version() (string, error) {
	return _L2DomiconCommitment.Contract.Version(&_L2DomiconCommitment.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) Version() (string, error) {
	return _L2DomiconCommitment.Contract.Version(&_L2DomiconCommitment.CallOpts)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactor) SetDom(opts *bind.TransactOpts, addr common.Address) (*types.Transaction, error) {
	return _L2DomiconCommitment.contract.Transact(opts, "SetDom", addr)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentSession) SetDom(addr common.Address) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.SetDom(&_L2DomiconCommitment.TransactOpts, addr)
}

// SetDom is a paid mutator transaction binding the contract method 0x6d881989.
//
// Solidity: function SetDom(address addr) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorSession) SetDom(addr common.Address) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.SetDom(&_L2DomiconCommitment.TransactOpts, addr)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactor) FinalizeSubmitCommitment(opts *bind.TransactOpts, _index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.contract.Transact(opts, "finalizeSubmitCommitment", _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L2DomiconCommitment *L2DomiconCommitmentSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L2DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L2DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconCommitment.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Initialize() (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.Initialize(&_L2DomiconCommitment.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorSession) Initialize() (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.Initialize(&_L2DomiconCommitment.TransactOpts)
}

// L2DomiconCommitmentFinalizeSubmitCommitmentIterator is returned from FilterFinalizeSubmitCommitment and is used to iterate over the raw logs and unpacked data for FinalizeSubmitCommitment events raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentFinalizeSubmitCommitmentIterator struct {
	Event *L2DomiconCommitmentFinalizeSubmitCommitment // Event containing the contract specifics and raw log

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
func (it *L2DomiconCommitmentFinalizeSubmitCommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconCommitmentFinalizeSubmitCommitment)
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
		it.Event = new(L2DomiconCommitmentFinalizeSubmitCommitment)
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
func (it *L2DomiconCommitmentFinalizeSubmitCommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconCommitmentFinalizeSubmitCommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconCommitmentFinalizeSubmitCommitment represents a FinalizeSubmitCommitment event raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentFinalizeSubmitCommitment struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	Broadcaster common.Address
	User        common.Address
	Sign        []byte
	Commitment  []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterFinalizeSubmitCommitment is a free log retrieval operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) FilterFinalizeSubmitCommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L2DomiconCommitmentFinalizeSubmitCommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L2DomiconCommitment.contract.FilterLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentFinalizeSubmitCommitmentIterator{contract: _L2DomiconCommitment.contract, event: "FinalizeSubmitCommitment", logs: logs, sub: sub}, nil
}

// WatchFinalizeSubmitCommitment is a free log subscription operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) WatchFinalizeSubmitCommitment(opts *bind.WatchOpts, sink chan<- *L2DomiconCommitmentFinalizeSubmitCommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L2DomiconCommitment.contract.WatchLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconCommitmentFinalizeSubmitCommitment)
				if err := _L2DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
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

// ParseFinalizeSubmitCommitment is a log parse operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) ParseFinalizeSubmitCommitment(log types.Log) (*L2DomiconCommitmentFinalizeSubmitCommitment, error) {
	event := new(L2DomiconCommitmentFinalizeSubmitCommitment)
	if err := _L2DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2DomiconCommitmentInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentInitializedIterator struct {
	Event *L2DomiconCommitmentInitialized // Event containing the contract specifics and raw log

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
func (it *L2DomiconCommitmentInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconCommitmentInitialized)
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
		it.Event = new(L2DomiconCommitmentInitialized)
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
func (it *L2DomiconCommitmentInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconCommitmentInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconCommitmentInitialized represents a Initialized event raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2DomiconCommitmentInitializedIterator, error) {

	logs, sub, err := _L2DomiconCommitment.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentInitializedIterator{contract: _L2DomiconCommitment.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2DomiconCommitmentInitialized) (event.Subscription, error) {

	logs, sub, err := _L2DomiconCommitment.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconCommitmentInitialized)
				if err := _L2DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) ParseInitialized(log types.Log) (*L2DomiconCommitmentInitialized, error) {
	event := new(L2DomiconCommitmentInitialized)
	if err := _L2DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2DomiconCommitmentSendDACommitmentIterator is returned from FilterSendDACommitment and is used to iterate over the raw logs and unpacked data for SendDACommitment events raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentSendDACommitmentIterator struct {
	Event *L2DomiconCommitmentSendDACommitment // Event containing the contract specifics and raw log

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
func (it *L2DomiconCommitmentSendDACommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconCommitmentSendDACommitment)
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
		it.Event = new(L2DomiconCommitmentSendDACommitment)
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
func (it *L2DomiconCommitmentSendDACommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconCommitmentSendDACommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconCommitmentSendDACommitment represents a SendDACommitment event raised by the L2DomiconCommitment contract.
type L2DomiconCommitmentSendDACommitment struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	Broadcaster common.Address
	User        common.Address
	Sign        []byte
	Commitment  []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSendDACommitment is a free log retrieval operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) FilterSendDACommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L2DomiconCommitmentSendDACommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L2DomiconCommitment.contract.FilterLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L2DomiconCommitmentSendDACommitmentIterator{contract: _L2DomiconCommitment.contract, event: "SendDACommitment", logs: logs, sub: sub}, nil
}

// WatchSendDACommitment is a free log subscription operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) WatchSendDACommitment(opts *bind.WatchOpts, sink chan<- *L2DomiconCommitmentSendDACommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L2DomiconCommitment.contract.WatchLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconCommitmentSendDACommitment)
				if err := _L2DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
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

// ParseSendDACommitment is a log parse operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentFilterer) ParseSendDACommitment(log types.Log) (*L2DomiconCommitmentSendDACommitment, error) {
	event := new(L2DomiconCommitmentSendDACommitment)
	if err := _L2DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
