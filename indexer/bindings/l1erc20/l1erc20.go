// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package l1erc20

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

// L1ERC20MetaData contains all meta data concerning the L1ERC20 contract.
var L1ERC20MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_initialAmount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_tokenName\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"_decimalUnits\",\"type\":\"uint8\"},{\"internalType\":\"string\",\"name\":\"_tokenSymbol\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"remaining\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowed\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"destroy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b5060405162000a0c38038062000a0c833981016040819052620000349162000203565b336000908152602081815260409091208590556005859055835162000060916002919086019062000090565b506003805460ff191660ff841617905580516200008590600490602084019062000090565b5050505050620002cf565b8280546200009e9062000292565b90600052602060002090601f016020900481019282620000c257600085556200010d565b82601f10620000dd57805160ff19168380011785556200010d565b828001600101855582156200010d579182015b828111156200010d578251825591602001919060010190620000f0565b506200011b9291506200011f565b5090565b5b808211156200011b576000815560010162000120565b634e487b7160e01b600052604160045260246000fd5b600082601f8301126200015e57600080fd5b81516001600160401b03808211156200017b576200017b62000136565b604051601f8301601f19908116603f01168101908282118183101715620001a657620001a662000136565b81604052838152602092508683858801011115620001c357600080fd5b600091505b83821015620001e75785820183015181830184015290820190620001c8565b83821115620001f95760008385830101525b9695505050505050565b600080600080608085870312156200021a57600080fd5b845160208601519094506001600160401b03808211156200023a57600080fd5b62000248888389016200014c565b94506040870151915060ff821682146200026157600080fd5b6060870151919350808211156200027757600080fd5b5062000286878288016200014c565b91505092959194509250565b600181811c90821680620002a757607f821691505b60208210811415620002c957634e487b7160e01b600052602260045260246000fd5b50919050565b61072d80620002df6000396000f3fe608060405234801561001057600080fd5b50600436106100b45760003560e01c80635c658165116100715780635c6581651461016357806370a082311461018e57806383197ef0146101b757806395d89b41146101bf578063a9059cbb146101c7578063dd62ed3e146101da57600080fd5b806306fdde03146100b9578063095ea7b3146100d757806318160ddd146100fa57806323b872dd1461011157806327e235e314610124578063313ce56714610144575b600080fd5b6100c1610213565b6040516100ce9190610574565b60405180910390f35b6100ea6100e53660046105e5565b6102a1565b60405190151581526020016100ce565b61010360055481565b6040519081526020016100ce565b6100ea61011f36600461060f565b61030d565b61010361013236600461064b565b60006020819052908152604090205481565b6003546101519060ff1681565b60405160ff90911681526020016100ce565b61010361017136600461066d565b600160209081526000928352604080842090915290825290205481565b61010361019c36600461064b565b6001600160a01b031660009081526020819052604090205490565b6101bd33ff5b005b6100c1610483565b6100ea6101d53660046105e5565b610490565b6101036101e836600461066d565b6001600160a01b03918216600090815260016020908152604080832093909416825291909152205490565b60028054610220906106a0565b80601f016020809104026020016040519081016040528092919081815260200182805461024c906106a0565b80156102995780601f1061026e57610100808354040283529160200191610299565b820191906000526020600020905b81548152906001019060200180831161027c57829003601f168201915b505050505081565b3360008181526001602090815260408083206001600160a01b038716808552925280832085905551919290917f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925906102fc9086815260200190565b60405180910390a350600192915050565b6001600160a01b038316600081815260016020908152604080832033845282528083205493835290829052812054909190831180159061034d5750828110155b61038e5760405162461bcd60e51b815260206004820152600d60248201526c62616420616c6c6f77616e636560981b60448201526064015b60405180910390fd5b6001600160a01b038416600090815260208190526040812080548592906103b69084906106f1565b90915550506001600160a01b038516600090815260208190526040812080548592906103e3908490610709565b909155505060001981101561042b576001600160a01b038516600090815260016020908152604080832033845290915281208054859290610425908490610709565b90915550505b836001600160a01b0316856001600160a01b03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8560405161047091815260200190565b60405180910390a3506001949350505050565b60048054610220906106a0565b336000908152602081905260408120548211156104e65760405162461bcd60e51b8152602060048201526014602482015273696e73756666696369656e742062616c616e636560601b6044820152606401610385565b3360009081526020819052604081208054849290610505908490610709565b90915550506001600160a01b038316600090815260208190526040812080548492906105329084906106f1565b90915550506040518281526001600160a01b0384169033907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef906020016102fc565b600060208083528351808285015260005b818110156105a157858101830151858201604001528201610585565b818111156105b3576000604083870101525b50601f01601f1916929092016040019392505050565b80356001600160a01b03811681146105e057600080fd5b919050565b600080604083850312156105f857600080fd5b610601836105c9565b946020939093013593505050565b60008060006060848603121561062457600080fd5b61062d846105c9565b925061063b602085016105c9565b9150604084013590509250925092565b60006020828403121561065d57600080fd5b610666826105c9565b9392505050565b6000806040838503121561068057600080fd5b610689836105c9565b9150610697602084016105c9565b90509250929050565b600181811c908216806106b457607f821691505b602082108114156106d557634e487b7160e01b600052602260045260246000fd5b50919050565b634e487b7160e01b600052601160045260246000fd5b60008219821115610704576107046106db565b500190565b60008282101561071b5761071b6106db565b50039056fea164736f6c6343000809000a",
}

// L1ERC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use L1ERC20MetaData.ABI instead.
var L1ERC20ABI = L1ERC20MetaData.ABI

// L1ERC20Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1ERC20MetaData.Bin instead.
var L1ERC20Bin = L1ERC20MetaData.Bin

// DeployL1ERC20 deploys a new Ethereum contract, binding an instance of L1ERC20 to it.
func DeployL1ERC20(auth *bind.TransactOpts, backend bind.ContractBackend, _initialAmount *big.Int, _tokenName string, _decimalUnits uint8, _tokenSymbol string) (common.Address, *types.Transaction, *L1ERC20, error) {
	parsed, err := L1ERC20MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1ERC20Bin), backend, _initialAmount, _tokenName, _decimalUnits, _tokenSymbol)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1ERC20{L1ERC20Caller: L1ERC20Caller{contract: contract}, L1ERC20Transactor: L1ERC20Transactor{contract: contract}, L1ERC20Filterer: L1ERC20Filterer{contract: contract}}, nil
}

// L1ERC20 is an auto generated Go binding around an Ethereum contract.
type L1ERC20 struct {
	L1ERC20Caller     // Read-only binding to the contract
	L1ERC20Transactor // Write-only binding to the contract
	L1ERC20Filterer   // Log filterer for contract events
}

// L1ERC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type L1ERC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type L1ERC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1ERC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ERC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1ERC20Session struct {
	Contract     *L1ERC20          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1ERC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1ERC20CallerSession struct {
	Contract *L1ERC20Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// L1ERC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1ERC20TransactorSession struct {
	Contract     *L1ERC20Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// L1ERC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type L1ERC20Raw struct {
	Contract *L1ERC20 // Generic contract binding to access the raw methods on
}

// L1ERC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1ERC20CallerRaw struct {
	Contract *L1ERC20Caller // Generic read-only contract binding to access the raw methods on
}

// L1ERC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1ERC20TransactorRaw struct {
	Contract *L1ERC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewL1ERC20 creates a new instance of L1ERC20, bound to a specific deployed contract.
func NewL1ERC20(address common.Address, backend bind.ContractBackend) (*L1ERC20, error) {
	contract, err := bindL1ERC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1ERC20{L1ERC20Caller: L1ERC20Caller{contract: contract}, L1ERC20Transactor: L1ERC20Transactor{contract: contract}, L1ERC20Filterer: L1ERC20Filterer{contract: contract}}, nil
}

// NewL1ERC20Caller creates a new read-only instance of L1ERC20, bound to a specific deployed contract.
func NewL1ERC20Caller(address common.Address, caller bind.ContractCaller) (*L1ERC20Caller, error) {
	contract, err := bindL1ERC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1ERC20Caller{contract: contract}, nil
}

// NewL1ERC20Transactor creates a new write-only instance of L1ERC20, bound to a specific deployed contract.
func NewL1ERC20Transactor(address common.Address, transactor bind.ContractTransactor) (*L1ERC20Transactor, error) {
	contract, err := bindL1ERC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1ERC20Transactor{contract: contract}, nil
}

// NewL1ERC20Filterer creates a new log filterer instance of L1ERC20, bound to a specific deployed contract.
func NewL1ERC20Filterer(address common.Address, filterer bind.ContractFilterer) (*L1ERC20Filterer, error) {
	contract, err := bindL1ERC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1ERC20Filterer{contract: contract}, nil
}

// bindL1ERC20 binds a generic wrapper to an already deployed contract.
func bindL1ERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1ERC20ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ERC20 *L1ERC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ERC20.Contract.L1ERC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ERC20 *L1ERC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ERC20.Contract.L1ERC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ERC20 *L1ERC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ERC20.Contract.L1ERC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ERC20 *L1ERC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ERC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ERC20 *L1ERC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ERC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ERC20 *L1ERC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ERC20.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_L1ERC20 *L1ERC20Caller) Allowance(opts *bind.CallOpts, _owner common.Address, _spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "allowance", _owner, _spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_L1ERC20 *L1ERC20Session) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Allowance(&_L1ERC20.CallOpts, _owner, _spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address _owner, address _spender) view returns(uint256 remaining)
func (_L1ERC20 *L1ERC20CallerSession) Allowance(_owner common.Address, _spender common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Allowance(&_L1ERC20.CallOpts, _owner, _spender)
}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_L1ERC20 *L1ERC20Caller) Allowed(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "allowed", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_L1ERC20 *L1ERC20Session) Allowed(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Allowed(&_L1ERC20.CallOpts, arg0, arg1)
}

// Allowed is a free data retrieval call binding the contract method 0x5c658165.
//
// Solidity: function allowed(address , address ) view returns(uint256)
func (_L1ERC20 *L1ERC20CallerSession) Allowed(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Allowed(&_L1ERC20.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_L1ERC20 *L1ERC20Caller) BalanceOf(opts *bind.CallOpts, _owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "balanceOf", _owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_L1ERC20 *L1ERC20Session) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.BalanceOf(&_L1ERC20.CallOpts, _owner)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _owner) view returns(uint256 balance)
func (_L1ERC20 *L1ERC20CallerSession) BalanceOf(_owner common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.BalanceOf(&_L1ERC20.CallOpts, _owner)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_L1ERC20 *L1ERC20Caller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_L1ERC20 *L1ERC20Session) Balances(arg0 common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Balances(&_L1ERC20.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_L1ERC20 *L1ERC20CallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _L1ERC20.Contract.Balances(&_L1ERC20.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_L1ERC20 *L1ERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_L1ERC20 *L1ERC20Session) Decimals() (uint8, error) {
	return _L1ERC20.Contract.Decimals(&_L1ERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_L1ERC20 *L1ERC20CallerSession) Decimals() (uint8, error) {
	return _L1ERC20.Contract.Decimals(&_L1ERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_L1ERC20 *L1ERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_L1ERC20 *L1ERC20Session) Name() (string, error) {
	return _L1ERC20.Contract.Name(&_L1ERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_L1ERC20 *L1ERC20CallerSession) Name() (string, error) {
	return _L1ERC20.Contract.Name(&_L1ERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_L1ERC20 *L1ERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_L1ERC20 *L1ERC20Session) Symbol() (string, error) {
	return _L1ERC20.Contract.Symbol(&_L1ERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_L1ERC20 *L1ERC20CallerSession) Symbol() (string, error) {
	return _L1ERC20.Contract.Symbol(&_L1ERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_L1ERC20 *L1ERC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1ERC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_L1ERC20 *L1ERC20Session) TotalSupply() (*big.Int, error) {
	return _L1ERC20.Contract.TotalSupply(&_L1ERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_L1ERC20 *L1ERC20CallerSession) TotalSupply() (*big.Int, error) {
	return _L1ERC20.Contract.TotalSupply(&_L1ERC20.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Transactor) Approve(opts *bind.TransactOpts, _spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.contract.Transact(opts, "approve", _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Session) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.Approve(&_L1ERC20.TransactOpts, _spender, _value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address _spender, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20TransactorSession) Approve(_spender common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.Approve(&_L1ERC20.TransactOpts, _spender, _value)
}

// Destroy is a paid mutator transaction binding the contract method 0x83197ef0.
//
// Solidity: function destroy() returns()
func (_L1ERC20 *L1ERC20Transactor) Destroy(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ERC20.contract.Transact(opts, "destroy")
}

// Destroy is a paid mutator transaction binding the contract method 0x83197ef0.
//
// Solidity: function destroy() returns()
func (_L1ERC20 *L1ERC20Session) Destroy() (*types.Transaction, error) {
	return _L1ERC20.Contract.Destroy(&_L1ERC20.TransactOpts)
}

// Destroy is a paid mutator transaction binding the contract method 0x83197ef0.
//
// Solidity: function destroy() returns()
func (_L1ERC20 *L1ERC20TransactorSession) Destroy() (*types.Transaction, error) {
	return _L1ERC20.Contract.Destroy(&_L1ERC20.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Transactor) Transfer(opts *bind.TransactOpts, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.contract.Transact(opts, "transfer", _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Session) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.Transfer(&_L1ERC20.TransactOpts, _to, _value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20TransactorSession) Transfer(_to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.Transfer(&_L1ERC20.TransactOpts, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Transactor) TransferFrom(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.contract.Transact(opts, "transferFrom", _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20Session) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.TransferFrom(&_L1ERC20.TransactOpts, _from, _to, _value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (_L1ERC20 *L1ERC20TransactorSession) TransferFrom(_from common.Address, _to common.Address, _value *big.Int) (*types.Transaction, error) {
	return _L1ERC20.Contract.TransferFrom(&_L1ERC20.TransactOpts, _from, _to, _value)
}

// L1ERC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the L1ERC20 contract.
type L1ERC20ApprovalIterator struct {
	Event *L1ERC20Approval // Event containing the contract specifics and raw log

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
func (it *L1ERC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ERC20Approval)
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
		it.Event = new(L1ERC20Approval)
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
func (it *L1ERC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ERC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ERC20Approval represents a Approval event raised by the L1ERC20 contract.
type L1ERC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) FilterApproval(opts *bind.FilterOpts, _owner []common.Address, _spender []common.Address) (*L1ERC20ApprovalIterator, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _L1ERC20.contract.FilterLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return &L1ERC20ApprovalIterator{contract: _L1ERC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *L1ERC20Approval, _owner []common.Address, _spender []common.Address) (event.Subscription, error) {

	var _ownerRule []interface{}
	for _, _ownerItem := range _owner {
		_ownerRule = append(_ownerRule, _ownerItem)
	}
	var _spenderRule []interface{}
	for _, _spenderItem := range _spender {
		_spenderRule = append(_spenderRule, _spenderItem)
	}

	logs, sub, err := _L1ERC20.contract.WatchLogs(opts, "Approval", _ownerRule, _spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ERC20Approval)
				if err := _L1ERC20.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed _owner, address indexed _spender, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) ParseApproval(log types.Log) (*L1ERC20Approval, error) {
	event := new(L1ERC20Approval)
	if err := _L1ERC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ERC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the L1ERC20 contract.
type L1ERC20TransferIterator struct {
	Event *L1ERC20Transfer // Event containing the contract specifics and raw log

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
func (it *L1ERC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ERC20Transfer)
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
		it.Event = new(L1ERC20Transfer)
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
func (it *L1ERC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ERC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ERC20Transfer represents a Transfer event raised by the L1ERC20 contract.
type L1ERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) FilterTransfer(opts *bind.FilterOpts, _from []common.Address, _to []common.Address) (*L1ERC20TransferIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1ERC20.contract.FilterLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return &L1ERC20TransferIterator{contract: _L1ERC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *L1ERC20Transfer, _from []common.Address, _to []common.Address) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	logs, sub, err := _L1ERC20.contract.WatchLogs(opts, "Transfer", _fromRule, _toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ERC20Transfer)
				if err := _L1ERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed _from, address indexed _to, uint256 _value)
func (_L1ERC20 *L1ERC20Filterer) ParseTransfer(log types.Log) (*L1ERC20Transfer, error) {
	event := new(L1ERC20Transfer)
	if err := _L1ERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
