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

// TeleportrDepositMetaData contains all meta data concerning the TeleportrDeposit contract.
var TeleportrDepositMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minDepositAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxDepositAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxBalance\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"name\":\"BalanceWithdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"depositId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"emitter\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"EtherReceived\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousBalance\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"MaxBalanceSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newAmount\",\"type\":\"uint256\"}],\"name\":\"MaxDepositAmountSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newAmount\",\"type\":\"uint256\"}],\"name\":\"MinDepositAmountSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"maxBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxDepositAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minDepositAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxDepositAmount\",\"type\":\"uint256\"}],\"name\":\"setMaxAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxBalance\",\"type\":\"uint256\"}],\"name\":\"setMaxBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minDepositAmount\",\"type\":\"uint256\"}],\"name\":\"setMinAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalDeposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdrawBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b50604051610b4a380380610b4a83398101604081905261002f91610153565b61003833610103565b6001839055600282905560038190556000600481905560408051918252602082018590527f65779d3ca560e9bdec52d08ed75431a84df87cb7796f0e51965f6efc0f556c0f910160405180910390a16040805160008152602081018490527fb1e6cc560df1786578fd4d1fe6e046f089a0c3be401e999b51a5112437911797910160405180910390a16040805160008152602081018390527f185c6391e7218e85de8a9346fc72024a0f88e1f04c186e6351230b93976ad50b910160405180910390a1505050610181565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60008060006060848603121561016857600080fd5b8351925060208401519150604084015190509250925092565b6109ba806101906000396000f3fe6080604052600436106100c05760003560e01c80637d882097116100745780638ed832711161004e5780638ed83271146103445780639d51d9b71461035a578063f2fde38b1461037a57600080fd5b80637d882097146102d9578063897b0637146102ef5780638da5cb5b1461030f57600080fd5b8063645006ca116100a5578063645006ca14610285578063715018a6146102ae57806373ad468a146102c357600080fd5b80634fe47f701461024e5780635fd8c7101461027057600080fd5b3661024957600154341015610136576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601b60248201527f4465706f73697420616d6f756e7420697320746f6f20736d616c6c000000000060448201526064015b60405180910390fd5b6002543411156101a2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f4465706f73697420616d6f756e7420697320746f6f2062696700000000000000604482015260640161012d565b60035447111561020e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f436f6e7472616374206d61782062616c616e6365206578636565646564000000604482015260640161012d565b600454604051349133917f2d27851832fcac28a0d4af1344f01fed7ffcfd15171c14c564a0c42aa57ae5c090600090a4600480546001019055005b600080fd5b34801561025a57600080fd5b5061026e61026936600461092e565b61039a565b005b34801561027c57600080fd5b5061026e61045c565b34801561029157600080fd5b5061029b60015481565b6040519081526020015b60405180910390f35b3480156102ba57600080fd5b5061026e610578565b3480156102cf57600080fd5b5061029b60035481565b3480156102e557600080fd5b5061029b60045481565b3480156102fb57600080fd5b5061026e61030a36600461092e565b610605565b34801561031b57600080fd5b5060005460405173ffffffffffffffffffffffffffffffffffffffff90911681526020016102a5565b34801561035057600080fd5b5061029b60025481565b34801561036657600080fd5b5061026e61037536600461092e565b6106c7565b34801561038657600080fd5b5061026e610395366004610947565b610789565b60005473ffffffffffffffffffffffffffffffffffffffff16331461041b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b60025460408051918252602082018390527fb1e6cc560df1786578fd4d1fe6e046f089a0c3be401e999b51a5112437911797910160405180910390a1600255565b60005473ffffffffffffffffffffffffffffffffffffffff1633146104dd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b6000546040805147808252915173ffffffffffffffffffffffffffffffffffffffff9093169283917fddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1919081900360200190a260405173ffffffffffffffffffffffffffffffffffffffff83169082156108fc029083906000818181858888f19350505050158015610573573d6000803e3d6000fd5b505050565b60005473ffffffffffffffffffffffffffffffffffffffff1633146105f9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b61060360006108b9565b565b60005473ffffffffffffffffffffffffffffffffffffffff163314610686576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b60015460408051918252602082018390527f65779d3ca560e9bdec52d08ed75431a84df87cb7796f0e51965f6efc0f556c0f910160405180910390a1600155565b60005473ffffffffffffffffffffffffffffffffffffffff163314610748576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b60035460408051918252602082018390527f185c6391e7218e85de8a9346fc72024a0f88e1f04c186e6351230b93976ad50b910160405180910390a1600355565b60005473ffffffffffffffffffffffffffffffffffffffff16331461080a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161012d565b73ffffffffffffffffffffffffffffffffffffffff81166108ad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161012d565b6108b6816108b9565b50565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60006020828403121561094057600080fd5b5035919050565b60006020828403121561095957600080fd5b813573ffffffffffffffffffffffffffffffffffffffff8116811461097d57600080fd5b939250505056fea2646970667358221220b610d8106720e33f96db31f8c5c4f9cecb50144a2cf1e4049ac0216cd47d0eff64736f6c63430008090033",
}

// TeleportrDepositABI is the input ABI used to generate the binding from.
// Deprecated: Use TeleportrDepositMetaData.ABI instead.
var TeleportrDepositABI = TeleportrDepositMetaData.ABI

// TeleportrDepositBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TeleportrDepositMetaData.Bin instead.
var TeleportrDepositBin = TeleportrDepositMetaData.Bin

// DeployTeleportrDeposit deploys a new Ethereum contract, binding an instance of TeleportrDeposit to it.
func DeployTeleportrDeposit(auth *bind.TransactOpts, backend bind.ContractBackend, _minDepositAmount *big.Int, _maxDepositAmount *big.Int, _maxBalance *big.Int) (common.Address, *types.Transaction, *TeleportrDeposit, error) {
	parsed, err := TeleportrDepositMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TeleportrDepositBin), backend, _minDepositAmount, _maxDepositAmount, _maxBalance)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TeleportrDeposit{TeleportrDepositCaller: TeleportrDepositCaller{contract: contract}, TeleportrDepositTransactor: TeleportrDepositTransactor{contract: contract}, TeleportrDepositFilterer: TeleportrDepositFilterer{contract: contract}}, nil
}

// TeleportrDeposit is an auto generated Go binding around an Ethereum contract.
type TeleportrDeposit struct {
	TeleportrDepositCaller     // Read-only binding to the contract
	TeleportrDepositTransactor // Write-only binding to the contract
	TeleportrDepositFilterer   // Log filterer for contract events
}

// TeleportrDepositCaller is an auto generated read-only Go binding around an Ethereum contract.
type TeleportrDepositCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDepositTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TeleportrDepositTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDepositFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TeleportrDepositFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TeleportrDepositSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TeleportrDepositSession struct {
	Contract     *TeleportrDeposit // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TeleportrDepositCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TeleportrDepositCallerSession struct {
	Contract *TeleportrDepositCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// TeleportrDepositTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TeleportrDepositTransactorSession struct {
	Contract     *TeleportrDepositTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// TeleportrDepositRaw is an auto generated low-level Go binding around an Ethereum contract.
type TeleportrDepositRaw struct {
	Contract *TeleportrDeposit // Generic contract binding to access the raw methods on
}

// TeleportrDepositCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TeleportrDepositCallerRaw struct {
	Contract *TeleportrDepositCaller // Generic read-only contract binding to access the raw methods on
}

// TeleportrDepositTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TeleportrDepositTransactorRaw struct {
	Contract *TeleportrDepositTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTeleportrDeposit creates a new instance of TeleportrDeposit, bound to a specific deployed contract.
func NewTeleportrDeposit(address common.Address, backend bind.ContractBackend) (*TeleportrDeposit, error) {
	contract, err := bindTeleportrDeposit(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TeleportrDeposit{TeleportrDepositCaller: TeleportrDepositCaller{contract: contract}, TeleportrDepositTransactor: TeleportrDepositTransactor{contract: contract}, TeleportrDepositFilterer: TeleportrDepositFilterer{contract: contract}}, nil
}

// NewTeleportrDepositCaller creates a new read-only instance of TeleportrDeposit, bound to a specific deployed contract.
func NewTeleportrDepositCaller(address common.Address, caller bind.ContractCaller) (*TeleportrDepositCaller, error) {
	contract, err := bindTeleportrDeposit(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositCaller{contract: contract}, nil
}

// NewTeleportrDepositTransactor creates a new write-only instance of TeleportrDeposit, bound to a specific deployed contract.
func NewTeleportrDepositTransactor(address common.Address, transactor bind.ContractTransactor) (*TeleportrDepositTransactor, error) {
	contract, err := bindTeleportrDeposit(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositTransactor{contract: contract}, nil
}

// NewTeleportrDepositFilterer creates a new log filterer instance of TeleportrDeposit, bound to a specific deployed contract.
func NewTeleportrDepositFilterer(address common.Address, filterer bind.ContractFilterer) (*TeleportrDepositFilterer, error) {
	contract, err := bindTeleportrDeposit(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositFilterer{contract: contract}, nil
}

// bindTeleportrDeposit binds a generic wrapper to an already deployed contract.
func bindTeleportrDeposit(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TeleportrDepositABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TeleportrDeposit *TeleportrDepositRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TeleportrDeposit.Contract.TeleportrDepositCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TeleportrDeposit *TeleportrDepositRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.TeleportrDepositTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TeleportrDeposit *TeleportrDepositRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.TeleportrDepositTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TeleportrDeposit *TeleportrDepositCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TeleportrDeposit.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TeleportrDeposit *TeleportrDepositTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TeleportrDeposit *TeleportrDepositTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.contract.Transact(opts, method, params...)
}

// MaxBalance is a free data retrieval call binding the contract method 0x73ad468a.
//
// Solidity: function maxBalance() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCaller) MaxBalance(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TeleportrDeposit.contract.Call(opts, &out, "maxBalance")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxBalance is a free data retrieval call binding the contract method 0x73ad468a.
//
// Solidity: function maxBalance() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositSession) MaxBalance() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MaxBalance(&_TeleportrDeposit.CallOpts)
}

// MaxBalance is a free data retrieval call binding the contract method 0x73ad468a.
//
// Solidity: function maxBalance() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCallerSession) MaxBalance() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MaxBalance(&_TeleportrDeposit.CallOpts)
}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCaller) MaxDepositAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TeleportrDeposit.contract.Call(opts, &out, "maxDepositAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositSession) MaxDepositAmount() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MaxDepositAmount(&_TeleportrDeposit.CallOpts)
}

// MaxDepositAmount is a free data retrieval call binding the contract method 0x8ed83271.
//
// Solidity: function maxDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCallerSession) MaxDepositAmount() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MaxDepositAmount(&_TeleportrDeposit.CallOpts)
}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCaller) MinDepositAmount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TeleportrDeposit.contract.Call(opts, &out, "minDepositAmount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositSession) MinDepositAmount() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MinDepositAmount(&_TeleportrDeposit.CallOpts)
}

// MinDepositAmount is a free data retrieval call binding the contract method 0x645006ca.
//
// Solidity: function minDepositAmount() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCallerSession) MinDepositAmount() (*big.Int, error) {
	return _TeleportrDeposit.Contract.MinDepositAmount(&_TeleportrDeposit.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDeposit *TeleportrDepositCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TeleportrDeposit.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDeposit *TeleportrDepositSession) Owner() (common.Address, error) {
	return _TeleportrDeposit.Contract.Owner(&_TeleportrDeposit.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TeleportrDeposit *TeleportrDepositCallerSession) Owner() (common.Address, error) {
	return _TeleportrDeposit.Contract.Owner(&_TeleportrDeposit.CallOpts)
}

// TotalDeposits is a free data retrieval call binding the contract method 0x7d882097.
//
// Solidity: function totalDeposits() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCaller) TotalDeposits(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TeleportrDeposit.contract.Call(opts, &out, "totalDeposits")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDeposits is a free data retrieval call binding the contract method 0x7d882097.
//
// Solidity: function totalDeposits() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositSession) TotalDeposits() (*big.Int, error) {
	return _TeleportrDeposit.Contract.TotalDeposits(&_TeleportrDeposit.CallOpts)
}

// TotalDeposits is a free data retrieval call binding the contract method 0x7d882097.
//
// Solidity: function totalDeposits() view returns(uint256)
func (_TeleportrDeposit *TeleportrDepositCallerSession) TotalDeposits() (*big.Int, error) {
	return _TeleportrDeposit.Contract.TotalDeposits(&_TeleportrDeposit.CallOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDeposit *TeleportrDepositSession) RenounceOwnership() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.RenounceOwnership(&_TeleportrDeposit.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.RenounceOwnership(&_TeleportrDeposit.TransactOpts)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) SetMaxAmount(opts *bind.TransactOpts, _maxDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "setMaxAmount", _maxDepositAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositSession) SetMaxAmount(_maxDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMaxAmount(&_TeleportrDeposit.TransactOpts, _maxDepositAmount)
}

// SetMaxAmount is a paid mutator transaction binding the contract method 0x4fe47f70.
//
// Solidity: function setMaxAmount(uint256 _maxDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) SetMaxAmount(_maxDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMaxAmount(&_TeleportrDeposit.TransactOpts, _maxDepositAmount)
}

// SetMaxBalance is a paid mutator transaction binding the contract method 0x9d51d9b7.
//
// Solidity: function setMaxBalance(uint256 _maxBalance) returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) SetMaxBalance(opts *bind.TransactOpts, _maxBalance *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "setMaxBalance", _maxBalance)
}

// SetMaxBalance is a paid mutator transaction binding the contract method 0x9d51d9b7.
//
// Solidity: function setMaxBalance(uint256 _maxBalance) returns()
func (_TeleportrDeposit *TeleportrDepositSession) SetMaxBalance(_maxBalance *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMaxBalance(&_TeleportrDeposit.TransactOpts, _maxBalance)
}

// SetMaxBalance is a paid mutator transaction binding the contract method 0x9d51d9b7.
//
// Solidity: function setMaxBalance(uint256 _maxBalance) returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) SetMaxBalance(_maxBalance *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMaxBalance(&_TeleportrDeposit.TransactOpts, _maxBalance)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) SetMinAmount(opts *bind.TransactOpts, _minDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "setMinAmount", _minDepositAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositSession) SetMinAmount(_minDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMinAmount(&_TeleportrDeposit.TransactOpts, _minDepositAmount)
}

// SetMinAmount is a paid mutator transaction binding the contract method 0x897b0637.
//
// Solidity: function setMinAmount(uint256 _minDepositAmount) returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) SetMinAmount(_minDepositAmount *big.Int) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.SetMinAmount(&_TeleportrDeposit.TransactOpts, _minDepositAmount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDeposit *TeleportrDepositSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.TransferOwnership(&_TeleportrDeposit.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.TransferOwnership(&_TeleportrDeposit.TransactOpts, newOwner)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) WithdrawBalance(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.Transact(opts, "withdrawBalance")
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDeposit *TeleportrDepositSession) WithdrawBalance() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.WithdrawBalance(&_TeleportrDeposit.TransactOpts)
}

// WithdrawBalance is a paid mutator transaction binding the contract method 0x5fd8c710.
//
// Solidity: function withdrawBalance() returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) WithdrawBalance() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.WithdrawBalance(&_TeleportrDeposit.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_TeleportrDeposit *TeleportrDepositTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TeleportrDeposit.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_TeleportrDeposit *TeleportrDepositSession) Receive() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.Receive(&_TeleportrDeposit.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_TeleportrDeposit *TeleportrDepositTransactorSession) Receive() (*types.Transaction, error) {
	return _TeleportrDeposit.Contract.Receive(&_TeleportrDeposit.TransactOpts)
}

// TeleportrDepositBalanceWithdrawnIterator is returned from FilterBalanceWithdrawn and is used to iterate over the raw logs and unpacked data for BalanceWithdrawn events raised by the TeleportrDeposit contract.
type TeleportrDepositBalanceWithdrawnIterator struct {
	Event *TeleportrDepositBalanceWithdrawn // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositBalanceWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositBalanceWithdrawn)
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
		it.Event = new(TeleportrDepositBalanceWithdrawn)
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
func (it *TeleportrDepositBalanceWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositBalanceWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositBalanceWithdrawn represents a BalanceWithdrawn event raised by the TeleportrDeposit contract.
type TeleportrDepositBalanceWithdrawn struct {
	Owner   common.Address
	Balance *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBalanceWithdrawn is a free log retrieval operation binding the contract event 0xddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1.
//
// Solidity: event BalanceWithdrawn(address indexed owner, uint256 balance)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterBalanceWithdrawn(opts *bind.FilterOpts, owner []common.Address) (*TeleportrDepositBalanceWithdrawnIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "BalanceWithdrawn", ownerRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositBalanceWithdrawnIterator{contract: _TeleportrDeposit.contract, event: "BalanceWithdrawn", logs: logs, sub: sub}, nil
}

// WatchBalanceWithdrawn is a free log subscription operation binding the contract event 0xddc398b321237a8d40ac914388309c2f52a08c134e4dc4ce61e32f57cb7d80f1.
//
// Solidity: event BalanceWithdrawn(address indexed owner, uint256 balance)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchBalanceWithdrawn(opts *bind.WatchOpts, sink chan<- *TeleportrDepositBalanceWithdrawn, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "BalanceWithdrawn", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositBalanceWithdrawn)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "BalanceWithdrawn", log); err != nil {
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
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseBalanceWithdrawn(log types.Log) (*TeleportrDepositBalanceWithdrawn, error) {
	event := new(TeleportrDepositBalanceWithdrawn)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "BalanceWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDepositEtherReceivedIterator is returned from FilterEtherReceived and is used to iterate over the raw logs and unpacked data for EtherReceived events raised by the TeleportrDeposit contract.
type TeleportrDepositEtherReceivedIterator struct {
	Event *TeleportrDepositEtherReceived // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositEtherReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositEtherReceived)
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
		it.Event = new(TeleportrDepositEtherReceived)
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
func (it *TeleportrDepositEtherReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositEtherReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositEtherReceived represents a EtherReceived event raised by the TeleportrDeposit contract.
type TeleportrDepositEtherReceived struct {
	DepositId *big.Int
	Emitter   common.Address
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterEtherReceived is a free log retrieval operation binding the contract event 0x2d27851832fcac28a0d4af1344f01fed7ffcfd15171c14c564a0c42aa57ae5c0.
//
// Solidity: event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterEtherReceived(opts *bind.FilterOpts, depositId []*big.Int, emitter []common.Address, amount []*big.Int) (*TeleportrDepositEtherReceivedIterator, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var emitterRule []interface{}
	for _, emitterItem := range emitter {
		emitterRule = append(emitterRule, emitterItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "EtherReceived", depositIdRule, emitterRule, amountRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositEtherReceivedIterator{contract: _TeleportrDeposit.contract, event: "EtherReceived", logs: logs, sub: sub}, nil
}

// WatchEtherReceived is a free log subscription operation binding the contract event 0x2d27851832fcac28a0d4af1344f01fed7ffcfd15171c14c564a0c42aa57ae5c0.
//
// Solidity: event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchEtherReceived(opts *bind.WatchOpts, sink chan<- *TeleportrDepositEtherReceived, depositId []*big.Int, emitter []common.Address, amount []*big.Int) (event.Subscription, error) {

	var depositIdRule []interface{}
	for _, depositIdItem := range depositId {
		depositIdRule = append(depositIdRule, depositIdItem)
	}
	var emitterRule []interface{}
	for _, emitterItem := range emitter {
		emitterRule = append(emitterRule, emitterItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "EtherReceived", depositIdRule, emitterRule, amountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositEtherReceived)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "EtherReceived", log); err != nil {
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

// ParseEtherReceived is a log parse operation binding the contract event 0x2d27851832fcac28a0d4af1344f01fed7ffcfd15171c14c564a0c42aa57ae5c0.
//
// Solidity: event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount)
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseEtherReceived(log types.Log) (*TeleportrDepositEtherReceived, error) {
	event := new(TeleportrDepositEtherReceived)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "EtherReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDepositMaxBalanceSetIterator is returned from FilterMaxBalanceSet and is used to iterate over the raw logs and unpacked data for MaxBalanceSet events raised by the TeleportrDeposit contract.
type TeleportrDepositMaxBalanceSetIterator struct {
	Event *TeleportrDepositMaxBalanceSet // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositMaxBalanceSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositMaxBalanceSet)
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
		it.Event = new(TeleportrDepositMaxBalanceSet)
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
func (it *TeleportrDepositMaxBalanceSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositMaxBalanceSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositMaxBalanceSet represents a MaxBalanceSet event raised by the TeleportrDeposit contract.
type TeleportrDepositMaxBalanceSet struct {
	PreviousBalance *big.Int
	NewBalance      *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterMaxBalanceSet is a free log retrieval operation binding the contract event 0x185c6391e7218e85de8a9346fc72024a0f88e1f04c186e6351230b93976ad50b.
//
// Solidity: event MaxBalanceSet(uint256 previousBalance, uint256 newBalance)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterMaxBalanceSet(opts *bind.FilterOpts) (*TeleportrDepositMaxBalanceSetIterator, error) {

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "MaxBalanceSet")
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositMaxBalanceSetIterator{contract: _TeleportrDeposit.contract, event: "MaxBalanceSet", logs: logs, sub: sub}, nil
}

// WatchMaxBalanceSet is a free log subscription operation binding the contract event 0x185c6391e7218e85de8a9346fc72024a0f88e1f04c186e6351230b93976ad50b.
//
// Solidity: event MaxBalanceSet(uint256 previousBalance, uint256 newBalance)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchMaxBalanceSet(opts *bind.WatchOpts, sink chan<- *TeleportrDepositMaxBalanceSet) (event.Subscription, error) {

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "MaxBalanceSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositMaxBalanceSet)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "MaxBalanceSet", log); err != nil {
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

// ParseMaxBalanceSet is a log parse operation binding the contract event 0x185c6391e7218e85de8a9346fc72024a0f88e1f04c186e6351230b93976ad50b.
//
// Solidity: event MaxBalanceSet(uint256 previousBalance, uint256 newBalance)
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseMaxBalanceSet(log types.Log) (*TeleportrDepositMaxBalanceSet, error) {
	event := new(TeleportrDepositMaxBalanceSet)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "MaxBalanceSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDepositMaxDepositAmountSetIterator is returned from FilterMaxDepositAmountSet and is used to iterate over the raw logs and unpacked data for MaxDepositAmountSet events raised by the TeleportrDeposit contract.
type TeleportrDepositMaxDepositAmountSetIterator struct {
	Event *TeleportrDepositMaxDepositAmountSet // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositMaxDepositAmountSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositMaxDepositAmountSet)
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
		it.Event = new(TeleportrDepositMaxDepositAmountSet)
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
func (it *TeleportrDepositMaxDepositAmountSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositMaxDepositAmountSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositMaxDepositAmountSet represents a MaxDepositAmountSet event raised by the TeleportrDeposit contract.
type TeleportrDepositMaxDepositAmountSet struct {
	PreviousAmount *big.Int
	NewAmount      *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterMaxDepositAmountSet is a free log retrieval operation binding the contract event 0xb1e6cc560df1786578fd4d1fe6e046f089a0c3be401e999b51a5112437911797.
//
// Solidity: event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterMaxDepositAmountSet(opts *bind.FilterOpts) (*TeleportrDepositMaxDepositAmountSetIterator, error) {

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "MaxDepositAmountSet")
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositMaxDepositAmountSetIterator{contract: _TeleportrDeposit.contract, event: "MaxDepositAmountSet", logs: logs, sub: sub}, nil
}

// WatchMaxDepositAmountSet is a free log subscription operation binding the contract event 0xb1e6cc560df1786578fd4d1fe6e046f089a0c3be401e999b51a5112437911797.
//
// Solidity: event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchMaxDepositAmountSet(opts *bind.WatchOpts, sink chan<- *TeleportrDepositMaxDepositAmountSet) (event.Subscription, error) {

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "MaxDepositAmountSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositMaxDepositAmountSet)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "MaxDepositAmountSet", log); err != nil {
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

// ParseMaxDepositAmountSet is a log parse operation binding the contract event 0xb1e6cc560df1786578fd4d1fe6e046f089a0c3be401e999b51a5112437911797.
//
// Solidity: event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseMaxDepositAmountSet(log types.Log) (*TeleportrDepositMaxDepositAmountSet, error) {
	event := new(TeleportrDepositMaxDepositAmountSet)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "MaxDepositAmountSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDepositMinDepositAmountSetIterator is returned from FilterMinDepositAmountSet and is used to iterate over the raw logs and unpacked data for MinDepositAmountSet events raised by the TeleportrDeposit contract.
type TeleportrDepositMinDepositAmountSetIterator struct {
	Event *TeleportrDepositMinDepositAmountSet // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositMinDepositAmountSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositMinDepositAmountSet)
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
		it.Event = new(TeleportrDepositMinDepositAmountSet)
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
func (it *TeleportrDepositMinDepositAmountSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositMinDepositAmountSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositMinDepositAmountSet represents a MinDepositAmountSet event raised by the TeleportrDeposit contract.
type TeleportrDepositMinDepositAmountSet struct {
	PreviousAmount *big.Int
	NewAmount      *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterMinDepositAmountSet is a free log retrieval operation binding the contract event 0x65779d3ca560e9bdec52d08ed75431a84df87cb7796f0e51965f6efc0f556c0f.
//
// Solidity: event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterMinDepositAmountSet(opts *bind.FilterOpts) (*TeleportrDepositMinDepositAmountSetIterator, error) {

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "MinDepositAmountSet")
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositMinDepositAmountSetIterator{contract: _TeleportrDeposit.contract, event: "MinDepositAmountSet", logs: logs, sub: sub}, nil
}

// WatchMinDepositAmountSet is a free log subscription operation binding the contract event 0x65779d3ca560e9bdec52d08ed75431a84df87cb7796f0e51965f6efc0f556c0f.
//
// Solidity: event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchMinDepositAmountSet(opts *bind.WatchOpts, sink chan<- *TeleportrDepositMinDepositAmountSet) (event.Subscription, error) {

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "MinDepositAmountSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositMinDepositAmountSet)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "MinDepositAmountSet", log); err != nil {
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

// ParseMinDepositAmountSet is a log parse operation binding the contract event 0x65779d3ca560e9bdec52d08ed75431a84df87cb7796f0e51965f6efc0f556c0f.
//
// Solidity: event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount)
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseMinDepositAmountSet(log types.Log) (*TeleportrDepositMinDepositAmountSet, error) {
	event := new(TeleportrDepositMinDepositAmountSet)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "MinDepositAmountSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TeleportrDepositOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the TeleportrDeposit contract.
type TeleportrDepositOwnershipTransferredIterator struct {
	Event *TeleportrDepositOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *TeleportrDepositOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TeleportrDepositOwnershipTransferred)
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
		it.Event = new(TeleportrDepositOwnershipTransferred)
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
func (it *TeleportrDepositOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TeleportrDepositOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TeleportrDepositOwnershipTransferred represents a OwnershipTransferred event raised by the TeleportrDeposit contract.
type TeleportrDepositOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TeleportrDeposit *TeleportrDepositFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*TeleportrDepositOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &TeleportrDepositOwnershipTransferredIterator{contract: _TeleportrDeposit.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TeleportrDeposit *TeleportrDepositFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *TeleportrDepositOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TeleportrDeposit.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TeleportrDepositOwnershipTransferred)
				if err := _TeleportrDeposit.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_TeleportrDeposit *TeleportrDepositFilterer) ParseOwnershipTransferred(log types.Log) (*TeleportrDepositOwnershipTransferred, error) {
	event := new(TeleportrDepositOwnershipTransferred)
	if err := _TeleportrDeposit.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
