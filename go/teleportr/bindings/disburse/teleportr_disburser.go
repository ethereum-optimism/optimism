// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package disburse

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

// TeleportrDisburserDisbursement is an auto generated low-level Go binding around an user-defined struct.
type TeleportrDisburserDisbursement struct {
	Amount *big.Int
	Addr   common.Address
}

// TeleportrDisburserMetaData contains all meta data concerning the TeleportrDisburser contract.
var TeleportrDisburserMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"BalanceWithdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"depositId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"DisbursementFailed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"depositId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"DisbursementSuccess\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nextDepositId\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"internalType\":\"structTeleportrDisburser.Disbursement[]\",\"name\":\"_disbursements\",\"type\":\"tuple[]\"}],\"name\":\"disburse\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDisbursements\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061001a33610024565b6000600155610074565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b6109e1806100836000396000f3fe6080604052600436106100655760003560e01c80638da5cb5b116100435780638da5cb5b146100bf578063ad48144d146100f4578063f2fde38b1461010757600080fd5b806325999e7f1461006a5780635fd8c71014610093578063715018a6146100aa575b600080fd5b34801561007657600080fd5b5061008060015481565b6040519081526020015b60405180910390f35b34801561009f57600080fd5b506100a8610127565b005b3480156100b657600080fd5b506100a8610248565b3480156100cb57600080fd5b5060005460405173ffffffffffffffffffffffffffffffffffffffff909116815260200161008a565b6100a8610102366004610840565b6102d5565b34801561011357600080fd5b506100a86101223660046108bf565b61069b565b60005473ffffffffffffffffffffffffffffffffffffffff1633146101ad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b6000546040805147808252915173ffffffffffffffffffffffffffffffffffffffff9093169283917fddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1919081900360200190a260405173ffffffffffffffffffffffffffffffffffffffff83169082156108fc029083906000818181858888f19350505050158015610243573d6000803e3d6000fd5b505050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146102c9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016101a4565b6102d360006107cb565b565b60005473ffffffffffffffffffffffffffffffffffffffff163314610356576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016101a4565b80806103be576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601060248201527f4e6f2064697362757273656d656e74730000000000000000000000000000000060448201526064016101a4565b60015484811461042a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f556e6578706563746564206e657874206465706f73697420696400000000000060448201526064016101a4565b60018054830190556000805b8381101561047757858582818110610450576104506108fc565b610463926040909102013590508361095a565b91508061046f81610972565b915050610436565b50348114610507576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f44697362757273656d656e7420746f74616c20213d20616d6f756e742073656e60448201527f740000000000000000000000000000000000000000000000000000000000000060648201526084016101a4565b60005b83811015610692576000868683818110610526576105266108fc565b9050604002016000013590506000878784818110610546576105466108fc565b905060400201602001602081019061055e91906108bf565b905060008173ffffffffffffffffffffffffffffffffffffffff16836108fc90604051600060405180830381858888f193505050503d80600081146105bf576040519150601f19603f3d011682016040523d82523d6000602084013e6105c4565b606091505b505090508015610624578173ffffffffffffffffffffffffffffffffffffffff16867feaa22fd2d7b875476355b32cf719794faf9d91b66e73bc6375a053cace9caaee8560405161061791815260200190565b60405180910390a3610676565b8173ffffffffffffffffffffffffffffffffffffffff16867f9b478c095979d3d3a7d602ffd9ee1f0843204d853558ae0882c8fcc0a5bc78cf8560405161066d91815260200190565b60405180910390a35b600186019550505050808061068a90610972565b91505061050a565b50505050505050565b60005473ffffffffffffffffffffffffffffffffffffffff16331461071c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016101a4565b73ffffffffffffffffffffffffffffffffffffffff81166107bf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016101a4565b6107c8816107cb565b50565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60008060006040848603121561085557600080fd5b83359250602084013567ffffffffffffffff8082111561087457600080fd5b818601915086601f83011261088857600080fd5b81358181111561089757600080fd5b8760208260061b85010111156108ac57600080fd5b6020830194508093505050509250925092565b6000602082840312156108d157600080fd5b813573ffffffffffffffffffffffffffffffffffffffff811681146108f557600080fd5b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561096d5761096d61092b565b500190565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8214156109a4576109a461092b565b506001019056fea26469706673582212209c98cb9cc04bc2bf5e8c0baf23ef87d045122afe6cfbb0e3c31bd854dd7fad7d64736f6c63430008090033",
}

// TeleportrDisburserABI is the input ABI used to generate the binding from.
// Deprecated: Use TeleportrDisburserMetaData.ABI instead.
var TeleportrDisburserABI = TeleportrDisburserMetaData.ABI

// TeleportrDisburserBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TeleportrDisburserMetaData.Bin instead.
var TeleportrDisburserBin = TeleportrDisburserMetaData.Bin

// DeployTeleportrDisburser deploys a new Ethereum contract, binding an instance of TeleportrDisburser to it.
func DeployTeleportrDisburser(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TeleportrDisburser, error) {
	parsed, err := TeleportrDisburserMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TeleportrDisburserBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TeleportrDisburser{TeleportrDisburserCaller: TeleportrDisburserCaller{contract: contract}, TeleportrDisburserTransactor: TeleportrDisburserTransactor{contract: contract}, TeleportrDisburserFilterer: TeleportrDisburserFilterer{contract: contract}}, nil
}

// TeleportrDisburser is an auto generated Go binding around an Ethereum contract.
type TeleportrDisburser struct {
	TeleportrDisburserCaller     // Read-only binding to the contract
	TeleportrDisburserTransactor // Write-only binding to the contract
	TeleportrDisburserFilterer   // Log filterer for contract events
}

// TeleportrDisburserCaller is an auto generated read-only Go binding around an Ethereum contract.
type TeleportrDisburserCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDisburserTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TeleportrDisburserTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDisburserFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TeleportrDisburserFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDisburserSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TeleportrDisburserSession struct {
	Contract     *TeleportrDisburser // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// TeleportrDisburserCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TeleportrDisburserCallerSession struct {
	Contract *TeleportrDisburserCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// TeleportrDisburserTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TeleportrDisburserTransactorSession struct {
	Contract     *TeleportrDisburserTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// TeleportrDisburserRaw is an auto generated low-level Go binding around an Ethereum contract.
type TeleportrDisburserRaw struct {
	Contract *TeleportrDisburser // Generic contract binding to access the raw methods on
}

// TeleportrDisburserCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TeleportrDisburserCallerRaw struct {
	Contract *TeleportrDisburserCaller // Generic read-only contract binding to access the raw methods on
}

// TeleportrDisburserTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TeleportrDisburserTransactorRaw struct {
	Contract *TeleportrDisburserTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTeleportrDisburser creates a new instance of TeleportrDisburser, bound to a specific deployed contract.
func NewTeleportrDisburser(address common.Address, backend bind.ContractBackend) (*TeleportrDisburser, error) {
	contract, err := bindTeleportrDisburser(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburser{TeleportrDisburserCaller: TeleportrDisburserCaller{contract: contract}, TeleportrDisburserTransactor: TeleportrDisburserTransactor{contract: contract}, TeleportrDisburserFilterer: TeleportrDisburserFilterer{contract: contract}}, nil
}

// NewTeleportrDisburserCaller creates a new read-only instance of TeleportrDisburser, bound to a specific deployed contract.
func NewTeleportrDisburserCaller(address common.Address, caller bind.ContractCaller) (*TeleportrDisburserCaller, error) {
	contract, err := bindTeleportrDisburser(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserCaller{contract: contract}, nil
}

// NewTeleportrDisburserTransactor creates a new write-only instance of TeleportrDisburser, bound to a specific deployed contract.
func NewTeleportrDisburserTransactor(address common.Address, transactor bind.ContractTransactor) (*TeleportrDisburserTransactor, error) {
	contract, err := bindTeleportrDisburser(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserTransactor{contract: contract}, nil
}

// NewTeleportrDisburserFilterer creates a new log filterer instance of TeleportrDisburser, bound to a specific deployed contract.
func NewTeleportrDisburserFilterer(address common.Address, filterer bind.ContractFilterer) (*TeleportrDisburserFilterer, error) {
	contract, err := bindTeleportrDisburser(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserFilterer{contract: contract}, nil
}

// bindTeleportrDisburser binds a generic wrapper to an already deployed contract.
func bindTeleportrDisburser(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TeleportrDisburserABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TeleportrDisburser *TeleportrDisburserRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TeleportrDisburser.Contract.TeleportrDisburserCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TeleportrDisburser *TeleportrDisburserRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.TeleportrDisburserTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TeleportrDisburser *TeleportrDisburserRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.TeleportrDisburserTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TeleportrDisburser *TeleportrDisburserCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TeleportrDisburser.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TeleportrDisburser *TeleportrDisburserTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TeleportrDisburser *TeleportrDisburserTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.contract.Transact(opts, method, params...)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDisburser *TeleportrDisburserCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TeleportrDisburser.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDisburser *TeleportrDisburserSession) Owner() (common.Address, error) {
	return _TeleportrDisburser.Contract.Owner(&_TeleportrDisburser.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDisburser *TeleportrDisburserCallerSession) Owner() (common.Address, error) {
	return _TeleportrDisburser.Contract.Owner(&_TeleportrDisburser.CallOpts)
}

// TotalDisbursements is a free data retrieval call binding the contract method 0x25999e7f.
//
// Solidity: function totalDisbursements() view returns(uint256)
func (_TeleportrDisburser *TeleportrDisburserCaller) TotalDisbursements(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TeleportrDisburser.contract.Call(opts, &out, "totalDisbursements")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDisbursements is a free data retrieval call binding the contract method 0x25999e7f.
//
// Solidity: function totalDisbursements() view returns(uint256)
func (_TeleportrDisburser *TeleportrDisburserSession) TotalDisbursements() (*big.Int, error) {
	return _TeleportrDisburser.Contract.TotalDisbursements(&_TeleportrDisburser.CallOpts)
}

// TotalDisbursements is a free data retrieval call binding the contract method 0x25999e7f.
//
// Solidity: function totalDisbursements() view returns(uint256)
func (_TeleportrDisburser *TeleportrDisburserCallerSession) TotalDisbursements() (*big.Int, error) {
	return _TeleportrDisburser.Contract.TotalDisbursements(&_TeleportrDisburser.CallOpts)
}

// Disburse is a paid mutator transaction binding the contract method 0xad48144d.
//
// Solidity: function disburse(uint256 _nextDepositId, (uint256,address)[] _disbursements) payable returns()
func (_TeleportrDisburser *TeleportrDisburserTransactor) Disburse(opts *bind.TransactOpts, _nextDepositId *big.Int, _disbursements []TeleportrDisburserDisbursement) (*types.Transaction, error) {
	return _TeleportrDisburser.contract.Transact(opts, "disburse", _nextDepositId, _disbursements)
}

// Disburse is a paid mutator transaction binding the contract method 0xad48144d.
//
// Solidity: function disburse(uint256 _nextDepositId, (uint256,address)[] _disbursements) payable returns()
func (_TeleportrDisburser *TeleportrDisburserSession) Disburse(_nextDepositId *big.Int, _disbursements []TeleportrDisburserDisbursement) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.Disburse(&_TeleportrDisburser.TransactOpts, _nextDepositId, _disbursements)
}

// Disburse is a paid mutator transaction binding the contract method 0xad48144d.
//
// Solidity: function disburse(uint256 _nextDepositId, (uint256,address)[] _disbursements) payable returns()
func (_TeleportrDisburser *TeleportrDisburserTransactorSession) Disburse(_nextDepositId *big.Int, _disbursements []TeleportrDisburserDisbursement) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.Disburse(&_TeleportrDisburser.TransactOpts, _nextDepositId, _disbursements)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDisburser *TeleportrDisburserTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDisburser.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDisburser *TeleportrDisburserSession) RenounceOwnership() (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.RenounceOwnership(&_TeleportrDisburser.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDisburser *TeleportrDisburserTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.RenounceOwnership(&_TeleportrDisburser.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDisburser *TeleportrDisburserTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDisburser.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDisburser *TeleportrDisburserSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.TransferOwnership(&_TeleportrDisburser.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDisburser *TeleportrDisburserTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.TransferOwnership(&_TeleportrDisburser.TransactOpts, newOwner)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDisburser *TeleportrDisburserTransactor) WithdrawBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDisburser.contract.Transact(opts, "withdrawBalance")
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDisburser *TeleportrDisburserSession) WithdrawBalance() (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.WithdrawBalance(&_TeleportrDisburser.TransactOpts)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDisburser *TeleportrDisburserTransactorSession) WithdrawBalance() (*types.Transaction, error) {
	return _TeleportrDisburser.Contract.WithdrawBalance(&_TeleportrDisburser.TransactOpts)
}

// TeleportrDisburserBalanceWithdrawnIterator is returned from FilterBalanceWithdrawn and is used to iterate over the raw logs and unpacked data for BalanceWithdrawn events raised by the TeleportrDisburser contract.
type TeleportrDisburserBalanceWithdrawnIterator struct {
	Event *TeleportrDisburserBalanceWithdrawn // Event containing the contract specifics and raw log

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
func (it *TeleportrDisburserBalanceWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDisburserBalanceWithdrawn)
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
		it.Event = new(TeleportrDisburserBalanceWithdrawn)
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
func (it *TeleportrDisburserBalanceWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDisburserBalanceWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDisburserBalanceWithdrawn represents a BalanceWithdrawn event raised by the TeleportrDisburser contract.
type TeleportrDisburserBalanceWithdrawn struct {
	Owner   common.Address
	Balance *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBalanceWithdrawn is a free log retrieval operation binding the contract event 0xddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1.
//
// Solidity: event BalanceWithdrawn(address indexed owner, uint256 balance)
func (_TeleportrDisburser *TeleportrDisburserFilterer) FilterBalanceWithdrawn(opts *bind.FilterOpts, owner []common.Address) (*TeleportrDisburserBalanceWithdrawnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.FilterLogs(opts, "BalanceWithdrawn", ownerRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserBalanceWithdrawnIterator{contract: _TeleportrDisburser.contract, event: "BalanceWithdrawn", logs: logs, sub: sub}, nil
}

// WatchBalanceWithdrawn is a free log subscription operation binding the contract event 0xddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1.
//
// Solidity: event BalanceWithdrawn(address indexed owner, uint256 balance)
func (_TeleportrDisburser *TeleportrDisburserFilterer) WatchBalanceWithdrawn(opts *bind.WatchOpts, sink chan<- *TeleportrDisburserBalanceWithdrawn, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.WatchLogs(opts, "BalanceWithdrawn", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDisburserBalanceWithdrawn)
				if err := _TeleportrDisburser.contract.UnpackLog(event, "BalanceWithdrawn", log); err != nil {
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

// ParseBalanceWithdrawn is a log parse operation binding the contract event 0xddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1.
//
// Solidity: event BalanceWithdrawn(address indexed owner, uint256 balance)
func (_TeleportrDisburser *TeleportrDisburserFilterer) ParseBalanceWithdrawn(log types.Log) (*TeleportrDisburserBalanceWithdrawn, error) {
	event := new(TeleportrDisburserBalanceWithdrawn)
	if err := _TeleportrDisburser.contract.UnpackLog(event, "BalanceWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDisburserDisbursementFailedIterator is returned from FilterDisbursementFailed and is used to iterate over the raw logs and unpacked data for DisbursementFailed events raised by the TeleportrDisburser contract.
type TeleportrDisburserDisbursementFailedIterator struct {
	Event *TeleportrDisburserDisbursementFailed // Event containing the contract specifics and raw log

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
func (it *TeleportrDisburserDisbursementFailedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDisburserDisbursementFailed)
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
		it.Event = new(TeleportrDisburserDisbursementFailed)
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
func (it *TeleportrDisburserDisbursementFailedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDisburserDisbursementFailedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDisburserDisbursementFailed represents a DisbursementFailed event raised by the TeleportrDisburser contract.
type TeleportrDisburserDisbursementFailed struct {
	DepositId *big.Int
	To        common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDisbursementFailed is a free log retrieval operation binding the contract event 0x9b478c095979d3d3a7d602ffd9ee1f0843204d853558ae0882c8fcc0a5bc78cf.
//
// Solidity: event DisbursementFailed(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) FilterDisbursementFailed(opts *bind.FilterOpts, depositId []*big.Int, to []common.Address) (*TeleportrDisburserDisbursementFailedIterator, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.FilterLogs(opts, "DisbursementFailed", depositIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserDisbursementFailedIterator{contract: _TeleportrDisburser.contract, event: "DisbursementFailed", logs: logs, sub: sub}, nil
}

// WatchDisbursementFailed is a free log subscription operation binding the contract event 0x9b478c095979d3d3a7d602ffd9ee1f0843204d853558ae0882c8fcc0a5bc78cf.
//
// Solidity: event DisbursementFailed(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) WatchDisbursementFailed(opts *bind.WatchOpts, sink chan<- *TeleportrDisburserDisbursementFailed, depositId []*big.Int, to []common.Address) (event.Subscription, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.WatchLogs(opts, "DisbursementFailed", depositIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDisburserDisbursementFailed)
				if err := _TeleportrDisburser.contract.UnpackLog(event, "DisbursementFailed", log); err != nil {
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

// ParseDisbursementFailed is a log parse operation binding the contract event 0x9b478c095979d3d3a7d602ffd9ee1f0843204d853558ae0882c8fcc0a5bc78cf.
//
// Solidity: event DisbursementFailed(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) ParseDisbursementFailed(log types.Log) (*TeleportrDisburserDisbursementFailed, error) {
	event := new(TeleportrDisburserDisbursementFailed)
	if err := _TeleportrDisburser.contract.UnpackLog(event, "DisbursementFailed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDisburserDisbursementSuccessIterator is returned from FilterDisbursementSuccess and is used to iterate over the raw logs and unpacked data for DisbursementSuccess events raised by the TeleportrDisburser contract.
type TeleportrDisburserDisbursementSuccessIterator struct {
	Event *TeleportrDisburserDisbursementSuccess // Event containing the contract specifics and raw log

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
func (it *TeleportrDisburserDisbursementSuccessIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDisburserDisbursementSuccess)
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
		it.Event = new(TeleportrDisburserDisbursementSuccess)
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
func (it *TeleportrDisburserDisbursementSuccessIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDisburserDisbursementSuccessIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDisburserDisbursementSuccess represents a DisbursementSuccess event raised by the TeleportrDisburser contract.
type TeleportrDisburserDisbursementSuccess struct {
	DepositId *big.Int
	To        common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDisbursementSuccess is a free log retrieval operation binding the contract event 0xeaa22fd2d7b875476355b32cf719794faf9d91b66e73bc6375a053cace9caaee.
//
// Solidity: event DisbursementSuccess(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) FilterDisbursementSuccess(opts *bind.FilterOpts, depositId []*big.Int, to []common.Address) (*TeleportrDisburserDisbursementSuccessIterator, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.FilterLogs(opts, "DisbursementSuccess", depositIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserDisbursementSuccessIterator{contract: _TeleportrDisburser.contract, event: "DisbursementSuccess", logs: logs, sub: sub}, nil
}

// WatchDisbursementSuccess is a free log subscription operation binding the contract event 0xeaa22fd2d7b875476355b32cf719794faf9d91b66e73bc6375a053cace9caaee.
//
// Solidity: event DisbursementSuccess(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) WatchDisbursementSuccess(opts *bind.WatchOpts, sink chan<- *TeleportrDisburserDisbursementSuccess, depositId []*big.Int, to []common.Address) (event.Subscription, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.WatchLogs(opts, "DisbursementSuccess", depositIdRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDisburserDisbursementSuccess)
				if err := _TeleportrDisburser.contract.UnpackLog(event, "DisbursementSuccess", log); err != nil {
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

// ParseDisbursementSuccess is a log parse operation binding the contract event 0xeaa22fd2d7b875476355b32cf719794faf9d91b66e73bc6375a053cace9caaee.
//
// Solidity: event DisbursementSuccess(uint256 indexed depositId, address indexed to, uint256 amount)
func (_TeleportrDisburser *TeleportrDisburserFilterer) ParseDisbursementSuccess(log types.Log) (*TeleportrDisburserDisbursementSuccess, error) {
	event := new(TeleportrDisburserDisbursementSuccess)
	if err := _TeleportrDisburser.contract.UnpackLog(event, "DisbursementSuccess", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDisburserOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the TeleportrDisburser contract.
type TeleportrDisburserOwnershipTransferredIterator struct {
	Event *TeleportrDisburserOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *TeleportrDisburserOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDisburserOwnershipTransferred)
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
		it.Event = new(TeleportrDisburserOwnershipTransferred)
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
func (it *TeleportrDisburserOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDisburserOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDisburserOwnershipTransferred represents a OwnershipTransferred event raised by the TeleportrDisburser contract.
type TeleportrDisburserOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TeleportrDisburser *TeleportrDisburserFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*TeleportrDisburserOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDisburserOwnershipTransferredIterator{contract: _TeleportrDisburser.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TeleportrDisburser *TeleportrDisburserFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *TeleportrDisburserOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TeleportrDisburser.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDisburserOwnershipTransferred)
				if err := _TeleportrDisburser.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_TeleportrDisburser *TeleportrDisburserFilterer) ParseOwnershipTransferred(log types.Log) (*TeleportrDisburserOwnershipTransferred, error) {
	event := new(TeleportrDisburserOwnershipTransferred)
	if err := _TeleportrDisburser.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
