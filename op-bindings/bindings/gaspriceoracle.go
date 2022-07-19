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

// GasPriceOracleMetaData contains all meta data concerning the GasPriceOracle contract.
var GasPriceOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"decimals\",\"type\":\"uint256\"}],\"name\":\"DecimalsUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"overhead\",\"type\":\"uint256\"}],\"name\":\"OverheadUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"scalar\",\"type\":\"uint256\"}],\"name\":\"ScalarUpdated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"baseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasPrice\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1Fee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"getL1GasUsed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BaseFee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"overhead\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"scalar\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_decimals\",\"type\":\"uint256\"}],\"name\":\"setDecimals\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_overhead\",\"type\":\"uint256\"}],\"name\":\"setOverhead\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_scalar\",\"type\":\"uint256\"}],\"name\":\"setScalar\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561001057600080fd5b5060405161109338038061109383398101604081905261002f91610178565b600080600161003d33610058565b60809290925260a05260c052610052816100a8565b506101a8565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b6000546001600160a01b031633146101075760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b6001600160a01b03811661016c5760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b60648201526084016100fe565b61017581610058565b50565b60006020828403121561018a57600080fd5b81516001600160a01b03811681146101a157600080fd5b9392505050565b60805160a05160c051610ebc6101d760003960006103f4015260006103cb015260006103a20152610ebc6000f3fe608060405234801561001057600080fd5b50600436106100f55760003560e01c80637046559711610097578063de26c4a111610066578063de26c4a1146101c0578063f2fde38b146101d3578063f45e65d8146101e6578063fe173b971461016457600080fd5b8063704655971461016a578063715018a61461017d5780638c8885c8146101855780638da5cb5b1461019857600080fd5b806349948e0e116100d357806349948e0e14610134578063519b4bd31461014757806354fd4d501461014f5780636ef25c3a1461016457600080fd5b80630c18c162146100fa578063313ce567146101165780633577afc51461011f575b600080fd5b61010360035481565b6040519081526020015b60405180910390f35b61010360055481565b61013261012d3660046109c4565b6101ef565b005b610103610142366004610a0c565b6102b1565b610103610311565b61015761039b565b60405161010d9190610b0b565b48610103565b6101326101783660046109c4565b61043e565b6101326104f4565b6101326101933660046109c4565b610581565b60005460405173ffffffffffffffffffffffffffffffffffffffff909116815260200161010d565b6101036101ce366004610a0c565b610637565b6101326101e1366004610b5c565b6106e2565b61010360045481565b60005473ffffffffffffffffffffffffffffffffffffffff163314610275576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b60038190556040518181527f32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4906020015b60405180910390a150565b6000806102bd83610637565b905060006102c9610311565b6102d39083610bc8565b90506000600554600a6102e69190610d27565b90506000600454836102f89190610bc8565b905060006103068383610d62565b979650505050505050565b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16635cf249696040518163ffffffff1660e01b8152600401602060405180830381865afa158015610372573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103969190610d76565b905090565b60606103c67f0000000000000000000000000000000000000000000000000000000000000000610812565b6103ef7f0000000000000000000000000000000000000000000000000000000000000000610812565b6104187f0000000000000000000000000000000000000000000000000000000000000000610812565b60405160200161042a93929190610d8f565b604051602081830303815290604052905090565b60005473ffffffffffffffffffffffffffffffffffffffff1633146104bf576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161026c565b60048190556040518181527f3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a906020016102a6565b60005473ffffffffffffffffffffffffffffffffffffffff163314610575576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161026c565b61057f600061094f565b565b60005473ffffffffffffffffffffffffffffffffffffffff163314610602576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161026c565b60058190556040518181527fd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1906020016102a6565b80516000908190815b818110156106ba5784818151811061065a5761065a610e05565b01602001517fff000000000000000000000000000000000000000000000000000000000000001660000361069a57610693600484610e34565b92506106a8565b6106a5601084610e34565b92505b806106b281610e4c565b915050610640565b506000600354836106cb9190610e34565b90506106d981610440610e34565b95945050505050565b60005473ffffffffffffffffffffffffffffffffffffffff163314610763576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161026c565b73ffffffffffffffffffffffffffffffffffffffff8116610806576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161026c565b61080f8161094f565b50565b60608160000361085557505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b811561087f578061086981610e4c565b91506108789050600a83610d62565b9150610859565b60008167ffffffffffffffff81111561089a5761089a6109dd565b6040519080825280601f01601f1916602001820160405280156108c4576020820181803683370190505b5090505b8415610947576108d9600183610e84565b91506108e6600a86610e9b565b6108f1906030610e34565b60f81b81838151811061090657610906610e05565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610940600a86610d62565b94506108c8565b949350505050565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b6000602082840312156109d657600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600060208284031215610a1e57600080fd5b813567ffffffffffffffff80821115610a3657600080fd5b818401915084601f830112610a4a57600080fd5b813581811115610a5c57610a5c6109dd565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610aa257610aa26109dd565b81604052828152876020848701011115610abb57600080fd5b826020860160208301376000928101602001929092525095945050505050565b60005b83811015610af6578181015183820152602001610ade565b83811115610b05576000848401525b50505050565b6020815260008251806020840152610b2a816040850160208701610adb565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b600060208284031215610b6e57600080fd5b813573ffffffffffffffffffffffffffffffffffffffff81168114610b9257600080fd5b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615610c0057610c00610b99565b500290565b600181815b80851115610c5e57817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff04821115610c4457610c44610b99565b80851615610c5157918102915b93841c9390800290610c0a565b509250929050565b600082610c7557506001610d21565b81610c8257506000610d21565b8160018114610c985760028114610ca257610cbe565b6001915050610d21565b60ff841115610cb357610cb3610b99565b50506001821b610d21565b5060208310610133831016604e8410600b8410161715610ce1575081810a610d21565b610ceb8383610c05565b807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff04821115610d1d57610d1d610b99565b0290505b92915050565b6000610b928383610c66565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082610d7157610d71610d33565b500490565b600060208284031215610d8857600080fd5b5051919050565b60008451610da1818460208901610adb565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551610ddd816001850160208a01610adb565b60019201918201528351610df8816002840160208801610adb565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008219821115610e4757610e47610b99565b500190565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610e7d57610e7d610b99565b5060010190565b600082821015610e9657610e96610b99565b500390565b600082610eaa57610eaa610d33565b50069056fea164736f6c634300080f000a",
}

// GasPriceOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use GasPriceOracleMetaData.ABI instead.
var GasPriceOracleABI = GasPriceOracleMetaData.ABI

// GasPriceOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GasPriceOracleMetaData.Bin instead.
var GasPriceOracleBin = GasPriceOracleMetaData.Bin

// DeployGasPriceOracle deploys a new Ethereum contract, binding an instance of GasPriceOracle to it.
func DeployGasPriceOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _owner common.Address) (common.Address, *types.Transaction, *GasPriceOracle, error) {
	parsed, err := GasPriceOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GasPriceOracleBin), backend, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &GasPriceOracle{GasPriceOracleCaller: GasPriceOracleCaller{contract: contract}, GasPriceOracleTransactor: GasPriceOracleTransactor{contract: contract}, GasPriceOracleFilterer: GasPriceOracleFilterer{contract: contract}}, nil
}

// GasPriceOracle is an auto generated Go binding around an Ethereum contract.
type GasPriceOracle struct {
	GasPriceOracleCaller     // Read-only binding to the contract
	GasPriceOracleTransactor // Write-only binding to the contract
	GasPriceOracleFilterer   // Log filterer for contract events
}

// GasPriceOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type GasPriceOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GasPriceOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GasPriceOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GasPriceOracleSession struct {
	Contract     *GasPriceOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GasPriceOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GasPriceOracleCallerSession struct {
	Contract *GasPriceOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// GasPriceOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GasPriceOracleTransactorSession struct {
	Contract     *GasPriceOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// GasPriceOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type GasPriceOracleRaw struct {
	Contract *GasPriceOracle // Generic contract binding to access the raw methods on
}

// GasPriceOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GasPriceOracleCallerRaw struct {
	Contract *GasPriceOracleCaller // Generic read-only contract binding to access the raw methods on
}

// GasPriceOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GasPriceOracleTransactorRaw struct {
	Contract *GasPriceOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGasPriceOracle creates a new instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracle(address common.Address, backend bind.ContractBackend) (*GasPriceOracle, error) {
	contract, err := bindGasPriceOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracle{GasPriceOracleCaller: GasPriceOracleCaller{contract: contract}, GasPriceOracleTransactor: GasPriceOracleTransactor{contract: contract}, GasPriceOracleFilterer: GasPriceOracleFilterer{contract: contract}}, nil
}

// NewGasPriceOracleCaller creates a new read-only instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleCaller(address common.Address, caller bind.ContractCaller) (*GasPriceOracleCaller, error) {
	contract, err := bindGasPriceOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleCaller{contract: contract}, nil
}

// NewGasPriceOracleTransactor creates a new write-only instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*GasPriceOracleTransactor, error) {
	contract, err := bindGasPriceOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleTransactor{contract: contract}, nil
}

// NewGasPriceOracleFilterer creates a new log filterer instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*GasPriceOracleFilterer, error) {
	contract, err := bindGasPriceOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleFilterer{contract: contract}, nil
}

// bindGasPriceOracle binds a generic wrapper to an already deployed contract.
func bindGasPriceOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(GasPriceOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasPriceOracle *GasPriceOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasPriceOracle.Contract.GasPriceOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasPriceOracle *GasPriceOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.GasPriceOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasPriceOracle *GasPriceOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.GasPriceOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasPriceOracle *GasPriceOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasPriceOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasPriceOracle *GasPriceOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasPriceOracle *GasPriceOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.contract.Transact(opts, method, params...)
}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "baseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BaseFee(&_GasPriceOracle.CallOpts)
}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BaseFee(&_GasPriceOracle.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Decimals(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Decimals() (*big.Int, error) {
	return _GasPriceOracle.Contract.Decimals(&_GasPriceOracle.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Decimals() (*big.Int, error) {
	return _GasPriceOracle.Contract.Decimals(&_GasPriceOracle.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GasPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "gasPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GasPrice() (*big.Int, error) {
	return _GasPriceOracle.Contract.GasPrice(&_GasPriceOracle.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GasPrice() (*big.Int, error) {
	return _GasPriceOracle.Contract.GasPrice(&_GasPriceOracle.CallOpts)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GetL1Fee(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "getL1Fee", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1Fee(&_GasPriceOracle.CallOpts, _data)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1Fee(&_GasPriceOracle.CallOpts, _data)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GetL1GasUsed(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "getL1GasUsed", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1GasUsed(&_GasPriceOracle.CallOpts, _data)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1GasUsed(&_GasPriceOracle.CallOpts, _data)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) L1BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "l1BaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) L1BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.L1BaseFee(&_GasPriceOracle.CallOpts)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) L1BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.L1BaseFee(&_GasPriceOracle.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Overhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "overhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Overhead() (*big.Int, error) {
	return _GasPriceOracle.Contract.Overhead(&_GasPriceOracle.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Overhead() (*big.Int, error) {
	return _GasPriceOracle.Contract.Overhead(&_GasPriceOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GasPriceOracle *GasPriceOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GasPriceOracle *GasPriceOracleSession) Owner() (common.Address, error) {
	return _GasPriceOracle.Contract.Owner(&_GasPriceOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GasPriceOracle *GasPriceOracleCallerSession) Owner() (common.Address, error) {
	return _GasPriceOracle.Contract.Owner(&_GasPriceOracle.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Scalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "scalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Scalar() (*big.Int, error) {
	return _GasPriceOracle.Contract.Scalar(&_GasPriceOracle.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Scalar() (*big.Int, error) {
	return _GasPriceOracle.Contract.Scalar(&_GasPriceOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleSession) Version() (string, error) {
	return _GasPriceOracle.Contract.Version(&_GasPriceOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleCallerSession) Version() (string, error) {
	return _GasPriceOracle.Contract.Version(&_GasPriceOracle.CallOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GasPriceOracle *GasPriceOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GasPriceOracle *GasPriceOracleSession) RenounceOwnership() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.RenounceOwnership(&_GasPriceOracle.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.RenounceOwnership(&_GasPriceOracle.TransactOpts)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_GasPriceOracle *GasPriceOracleTransactor) SetDecimals(opts *bind.TransactOpts, _decimals *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "setDecimals", _decimals)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_GasPriceOracle *GasPriceOracleSession) SetDecimals(_decimals *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetDecimals(&_GasPriceOracle.TransactOpts, _decimals)
}

// SetDecimals is a paid mutator transaction binding the contract method 0x8c8885c8.
//
// Solidity: function setDecimals(uint256 _decimals) returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) SetDecimals(_decimals *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetDecimals(&_GasPriceOracle.TransactOpts, _decimals)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_GasPriceOracle *GasPriceOracleTransactor) SetOverhead(opts *bind.TransactOpts, _overhead *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "setOverhead", _overhead)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_GasPriceOracle *GasPriceOracleSession) SetOverhead(_overhead *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetOverhead(&_GasPriceOracle.TransactOpts, _overhead)
}

// SetOverhead is a paid mutator transaction binding the contract method 0x3577afc5.
//
// Solidity: function setOverhead(uint256 _overhead) returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) SetOverhead(_overhead *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetOverhead(&_GasPriceOracle.TransactOpts, _overhead)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_GasPriceOracle *GasPriceOracleTransactor) SetScalar(opts *bind.TransactOpts, _scalar *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "setScalar", _scalar)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_GasPriceOracle *GasPriceOracleSession) SetScalar(_scalar *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetScalar(&_GasPriceOracle.TransactOpts, _scalar)
}

// SetScalar is a paid mutator transaction binding the contract method 0x70465597.
//
// Solidity: function setScalar(uint256 _scalar) returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) SetScalar(_scalar *big.Int) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetScalar(&_GasPriceOracle.TransactOpts, _scalar)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GasPriceOracle *GasPriceOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GasPriceOracle *GasPriceOracleSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.TransferOwnership(&_GasPriceOracle.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.TransferOwnership(&_GasPriceOracle.TransactOpts, newOwner)
}

// GasPriceOracleDecimalsUpdatedIterator is returned from FilterDecimalsUpdated and is used to iterate over the raw logs and unpacked data for DecimalsUpdated events raised by the GasPriceOracle contract.
type GasPriceOracleDecimalsUpdatedIterator struct {
	Event *GasPriceOracleDecimalsUpdated // Event containing the contract specifics and raw log

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
func (it *GasPriceOracleDecimalsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GasPriceOracleDecimalsUpdated)
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
		it.Event = new(GasPriceOracleDecimalsUpdated)
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
func (it *GasPriceOracleDecimalsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GasPriceOracleDecimalsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GasPriceOracleDecimalsUpdated represents a DecimalsUpdated event raised by the GasPriceOracle contract.
type GasPriceOracleDecimalsUpdated struct {
	Decimals *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterDecimalsUpdated is a free log retrieval operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 decimals)
func (_GasPriceOracle *GasPriceOracleFilterer) FilterDecimalsUpdated(opts *bind.FilterOpts) (*GasPriceOracleDecimalsUpdatedIterator, error) {

	logs, sub, err := _GasPriceOracle.contract.FilterLogs(opts, "DecimalsUpdated")
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleDecimalsUpdatedIterator{contract: _GasPriceOracle.contract, event: "DecimalsUpdated", logs: logs, sub: sub}, nil
}

// WatchDecimalsUpdated is a free log subscription operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 decimals)
func (_GasPriceOracle *GasPriceOracleFilterer) WatchDecimalsUpdated(opts *bind.WatchOpts, sink chan<- *GasPriceOracleDecimalsUpdated) (event.Subscription, error) {

	logs, sub, err := _GasPriceOracle.contract.WatchLogs(opts, "DecimalsUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GasPriceOracleDecimalsUpdated)
				if err := _GasPriceOracle.contract.UnpackLog(event, "DecimalsUpdated", log); err != nil {
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

// ParseDecimalsUpdated is a log parse operation binding the contract event 0xd68112a8707e326d08be3656b528c1bcc5bbbfc47f4177e2179b14d8640838c1.
//
// Solidity: event DecimalsUpdated(uint256 decimals)
func (_GasPriceOracle *GasPriceOracleFilterer) ParseDecimalsUpdated(log types.Log) (*GasPriceOracleDecimalsUpdated, error) {
	event := new(GasPriceOracleDecimalsUpdated)
	if err := _GasPriceOracle.contract.UnpackLog(event, "DecimalsUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GasPriceOracleOverheadUpdatedIterator is returned from FilterOverheadUpdated and is used to iterate over the raw logs and unpacked data for OverheadUpdated events raised by the GasPriceOracle contract.
type GasPriceOracleOverheadUpdatedIterator struct {
	Event *GasPriceOracleOverheadUpdated // Event containing the contract specifics and raw log

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
func (it *GasPriceOracleOverheadUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GasPriceOracleOverheadUpdated)
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
		it.Event = new(GasPriceOracleOverheadUpdated)
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
func (it *GasPriceOracleOverheadUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GasPriceOracleOverheadUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GasPriceOracleOverheadUpdated represents a OverheadUpdated event raised by the GasPriceOracle contract.
type GasPriceOracleOverheadUpdated struct {
	Overhead *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOverheadUpdated is a free log retrieval operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 overhead)
func (_GasPriceOracle *GasPriceOracleFilterer) FilterOverheadUpdated(opts *bind.FilterOpts) (*GasPriceOracleOverheadUpdatedIterator, error) {

	logs, sub, err := _GasPriceOracle.contract.FilterLogs(opts, "OverheadUpdated")
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleOverheadUpdatedIterator{contract: _GasPriceOracle.contract, event: "OverheadUpdated", logs: logs, sub: sub}, nil
}

// WatchOverheadUpdated is a free log subscription operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 overhead)
func (_GasPriceOracle *GasPriceOracleFilterer) WatchOverheadUpdated(opts *bind.WatchOpts, sink chan<- *GasPriceOracleOverheadUpdated) (event.Subscription, error) {

	logs, sub, err := _GasPriceOracle.contract.WatchLogs(opts, "OverheadUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GasPriceOracleOverheadUpdated)
				if err := _GasPriceOracle.contract.UnpackLog(event, "OverheadUpdated", log); err != nil {
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

// ParseOverheadUpdated is a log parse operation binding the contract event 0x32740b35c0ea213650f60d44366b4fb211c9033b50714e4a1d34e65d5beb9bb4.
//
// Solidity: event OverheadUpdated(uint256 overhead)
func (_GasPriceOracle *GasPriceOracleFilterer) ParseOverheadUpdated(log types.Log) (*GasPriceOracleOverheadUpdated, error) {
	event := new(GasPriceOracleOverheadUpdated)
	if err := _GasPriceOracle.contract.UnpackLog(event, "OverheadUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GasPriceOracleOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the GasPriceOracle contract.
type GasPriceOracleOwnershipTransferredIterator struct {
	Event *GasPriceOracleOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *GasPriceOracleOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GasPriceOracleOwnershipTransferred)
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
		it.Event = new(GasPriceOracleOwnershipTransferred)
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
func (it *GasPriceOracleOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GasPriceOracleOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GasPriceOracleOwnershipTransferred represents a OwnershipTransferred event raised by the GasPriceOracle contract.
type GasPriceOracleOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GasPriceOracle *GasPriceOracleFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*GasPriceOracleOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GasPriceOracle.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleOwnershipTransferredIterator{contract: _GasPriceOracle.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GasPriceOracle *GasPriceOracleFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *GasPriceOracleOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GasPriceOracle.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GasPriceOracleOwnershipTransferred)
				if err := _GasPriceOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_GasPriceOracle *GasPriceOracleFilterer) ParseOwnershipTransferred(log types.Log) (*GasPriceOracleOwnershipTransferred, error) {
	event := new(GasPriceOracleOwnershipTransferred)
	if err := _GasPriceOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GasPriceOracleScalarUpdatedIterator is returned from FilterScalarUpdated and is used to iterate over the raw logs and unpacked data for ScalarUpdated events raised by the GasPriceOracle contract.
type GasPriceOracleScalarUpdatedIterator struct {
	Event *GasPriceOracleScalarUpdated // Event containing the contract specifics and raw log

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
func (it *GasPriceOracleScalarUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GasPriceOracleScalarUpdated)
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
		it.Event = new(GasPriceOracleScalarUpdated)
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
func (it *GasPriceOracleScalarUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GasPriceOracleScalarUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GasPriceOracleScalarUpdated represents a ScalarUpdated event raised by the GasPriceOracle contract.
type GasPriceOracleScalarUpdated struct {
	Scalar *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterScalarUpdated is a free log retrieval operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 scalar)
func (_GasPriceOracle *GasPriceOracleFilterer) FilterScalarUpdated(opts *bind.FilterOpts) (*GasPriceOracleScalarUpdatedIterator, error) {

	logs, sub, err := _GasPriceOracle.contract.FilterLogs(opts, "ScalarUpdated")
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleScalarUpdatedIterator{contract: _GasPriceOracle.contract, event: "ScalarUpdated", logs: logs, sub: sub}, nil
}

// WatchScalarUpdated is a free log subscription operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 scalar)
func (_GasPriceOracle *GasPriceOracleFilterer) WatchScalarUpdated(opts *bind.WatchOpts, sink chan<- *GasPriceOracleScalarUpdated) (event.Subscription, error) {

	logs, sub, err := _GasPriceOracle.contract.WatchLogs(opts, "ScalarUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GasPriceOracleScalarUpdated)
				if err := _GasPriceOracle.contract.UnpackLog(event, "ScalarUpdated", log); err != nil {
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

// ParseScalarUpdated is a log parse operation binding the contract event 0x3336cd9708eaf2769a0f0dc0679f30e80f15dcd88d1921b5a16858e8b85c591a.
//
// Solidity: event ScalarUpdated(uint256 scalar)
func (_GasPriceOracle *GasPriceOracleFilterer) ParseScalarUpdated(log types.Log) (*GasPriceOracleScalarUpdated, error) {
	event := new(GasPriceOracleScalarUpdated)
	if err := _GasPriceOracle.contract.UnpackLog(event, "ScalarUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
