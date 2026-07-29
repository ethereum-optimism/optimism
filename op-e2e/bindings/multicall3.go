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

// Multicall3Call is an auto generated low-level Go binding around an user-defined struct.
type Multicall3Call struct {
	Target   common.Address
	CallData []byte
}

// Multicall3Call3 is an auto generated low-level Go binding around an user-defined struct.
type Multicall3Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

// Multicall3Call3Value is an auto generated low-level Go binding around an user-defined struct.
type Multicall3Call3Value struct {
	Target       common.Address
	AllowFailure bool
	Value        *big.Int
	CallData     []byte
}

// Multicall3Result is an auto generated low-level Go binding around an user-defined struct.
type Multicall3Result struct {
	Success    bool
	ReturnData []byte
}

// MultiCall3MetaData contains all meta data concerning the MultiCall3 contract.
var MultiCall3MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes[]\",\"name\":\"returnData\",\"type\":\"bytes[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowFailure\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call3[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate3\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowFailure\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call3Value[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate3Value\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"blockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBasefee\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"basefee\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainid\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockCoinbase\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockDifficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"difficulty\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gaslimit\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"getEthBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryAggregate\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryBlockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall3.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610ee0806100206000396000f3fe6080604052600436106100f35760003560e01c80634d2301cc1161008a578063a8b0574e11610059578063a8b0574e1461025a578063bce38bd714610275578063c3077fa914610288578063ee82ac5e1461029b57600080fd5b80634d2301cc146101ec57806372425d9d1461022157806382ad56cb1461023457806386d516e81461024757600080fd5b80633408e470116100c65780633408e47014610191578063399542e9146101a45780633e64a696146101c657806342cbb15c146101d957600080fd5b80630f28c97d146100f8578063174dea711461011a578063252dba421461013a57806327e86d6e1461015b575b600080fd5b34801561010457600080fd5b50425b6040519081526020015b60405180910390f35b61012d610128366004610a85565b6102ba565b6040516101119190610bbe565b61014d610148366004610a85565b6104ef565b604051610111929190610bd8565b34801561016757600080fd5b50437fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0140610107565b34801561019d57600080fd5b5046610107565b6101b76101b2366004610c60565b610690565b60405161011193929190610cba565b3480156101d257600080fd5b5048610107565b3480156101e557600080fd5b5043610107565b3480156101f857600080fd5b50610107610207366004610ce2565b73ffffffffffffffffffffffffffffffffffffffff163190565b34801561022d57600080fd5b5044610107565b61012d610242366004610a85565b6106ab565b34801561025357600080fd5b5045610107565b34801561026657600080fd5b50604051418152602001610111565b61012d610283366004610c60565b61085a565b6101b7610296366004610a85565b610a1a565b3480156102a757600080fd5b506101076102b6366004610d18565b4090565b60606000828067ffffffffffffffff8111156102d8576102d8610d31565b60405190808252806020026020018201604052801561031e57816020015b6040805180820190915260008152606060208201528152602001906001900390816102f65790505b5092503660005b8281101561047757600085828151811061034157610341610d60565b6020026020010151905087878381811061035d5761035d610d60565b905060200281019061036f9190610d8f565b6040810135958601959093506103886020850185610ce2565b73ffffffffffffffffffffffffffffffffffffffff16816103ac6060870187610dcd565b6040516103ba929190610e32565b60006040518083038185875af1925050503d80600081146103f7576040519150601f19603f3d011682016040523d82523d6000602084013e6103fc565b606091505b50602080850191909152901515808452908501351761046d577f08c379a000000000000000000000000000000000000000000000000000000000600052602060045260176024527f4d756c746963616c6c333a2063616c6c206661696c656400000000000000000060445260846000fd5b5050600101610325565b508234146104e6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601a60248201527f4d756c746963616c6c333a2076616c7565206d69736d6174636800000000000060448201526064015b60405180910390fd5b50505092915050565b436060828067ffffffffffffffff81111561050c5761050c610d31565b60405190808252806020026020018201604052801561053f57816020015b606081526020019060019003908161052a5790505b5091503660005b8281101561068657600087878381811061056257610562610d60565b90506020028101906105749190610e42565b92506105836020840184610ce2565b73ffffffffffffffffffffffffffffffffffffffff166105a66020850185610dcd565b6040516105b4929190610e32565b6000604051808303816000865af19150503d80600081146105f1576040519150601f19603f3d011682016040523d82523d6000602084013e6105f6565b606091505b5086848151811061060957610609610d60565b602090810291909101015290508061067d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f4d756c746963616c6c333a2063616c6c206661696c656400000000000000000060448201526064016104dd565b50600101610546565b5050509250929050565b43804060606106a086868661085a565b905093509350939050565b6060818067ffffffffffffffff8111156106c7576106c7610d31565b60405190808252806020026020018201604052801561070d57816020015b6040805180820190915260008152606060208201528152602001906001900390816106e55790505b5091503660005b828110156104e657600084828151811061073057610730610d60565b6020026020010151905086868381811061074c5761074c610d60565b905060200281019061075e9190610e76565b925061076d6020840184610ce2565b73ffffffffffffffffffffffffffffffffffffffff166107906040850185610dcd565b60405161079e929190610e32565b6000604051808303816000865af19150503d80600081146107db576040519150601f19603f3d011682016040523d82523d6000602084013e6107e0565b606091505b506020808401919091529015158083529084013517610851577f08c379a000000000000000000000000000000000000000000000000000000000600052602060045260176024527f4d756c746963616c6c333a2063616c6c206661696c656400000000000000000060445260646000fd5b50600101610714565b6060818067ffffffffffffffff81111561087657610876610d31565b6040519080825280602002602001820160405280156108bc57816020015b6040805180820190915260008152606060208201528152602001906001900390816108945790505b5091503660005b82811015610a105760008482815181106108df576108df610d60565b602002602001015190508686838181106108fb576108fb610d60565b905060200281019061090d9190610e42565b925061091c6020840184610ce2565b73ffffffffffffffffffffffffffffffffffffffff1661093f6020850185610dcd565b60405161094d929190610e32565b6000604051808303816000865af19150503d806000811461098a576040519150601f19603f3d011682016040523d82523d6000602084013e61098f565b606091505b506020830152151581528715610a07578051610a07576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601760248201527f4d756c746963616c6c333a2063616c6c206661696c656400000000000000000060448201526064016104dd565b506001016108c3565b5050509392505050565b6000806060610a2b60018686610690565b919790965090945092505050565b60008083601f840112610a4b57600080fd5b50813567ffffffffffffffff811115610a6357600080fd5b6020830191508360208260051b8501011115610a7e57600080fd5b9250929050565b60008060208385031215610a9857600080fd5b823567ffffffffffffffff811115610aaf57600080fd5b610abb85828601610a39565b90969095509350505050565b6000815180845260005b81811015610aed57602081850181015186830182015201610ad1565b81811115610aff576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b600082825180855260208086019550808260051b84010181860160005b84811015610bb1578583037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe001895281518051151584528401516040858501819052610b9d81860183610ac7565b9a86019a9450505090830190600101610b4f565b5090979650505050505050565b602081526000610bd16020830184610b32565b9392505050565b600060408201848352602060408185015281855180845260608601915060608160051b870101935082870160005b82811015610c52577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0888703018452610c40868351610ac7565b95509284019290840190600101610c06565b509398975050505050505050565b600080600060408486031215610c7557600080fd5b83358015158114610c8557600080fd5b9250602084013567ffffffffffffffff811115610ca157600080fd5b610cad86828701610a39565b9497909650939450505050565b838152826020820152606060408201526000610cd96060830184610b32565b95945050505050565b600060208284031215610cf457600080fd5b813573ffffffffffffffffffffffffffffffffffffffff81168114610bd157600080fd5b600060208284031215610d2a57600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81833603018112610dc357600080fd5b9190910192915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610e0257600080fd5b83018035915067ffffffffffffffff821115610e1d57600080fd5b602001915036819003821315610a7e57600080fd5b8183823760009101908152919050565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc1833603018112610dc357600080fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa1833603018112610dc357600080fdfea2646970667358221220bb2b5c71a328032f97c676ae39a1ec2148d3e5d6f73d95e9b17910152d61f16264736f6c634300080c0033",
}

// MultiCall3ABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiCall3MetaData.ABI instead.
var MultiCall3ABI = MultiCall3MetaData.ABI

// MultiCall3Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MultiCall3MetaData.Bin instead.
var MultiCall3Bin = MultiCall3MetaData.Bin

// DeployMultiCall3 deploys a new Ethereum contract, binding an instance of MultiCall3 to it.
func DeployMultiCall3(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MultiCall3, error) {
	parsed, err := MultiCall3MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MultiCall3Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiCall3{MultiCall3Caller: MultiCall3Caller{contract: contract}, MultiCall3Transactor: MultiCall3Transactor{contract: contract}, MultiCall3Filterer: MultiCall3Filterer{contract: contract}}, nil
}

// MultiCall3 is an auto generated Go binding around an Ethereum contract.
type MultiCall3 struct {
	MultiCall3Caller     // Read-only binding to the contract
	MultiCall3Transactor // Write-only binding to the contract
	MultiCall3Filterer   // Log filterer for contract events
}

// MultiCall3Caller is an auto generated read-only Go binding around an Ethereum contract.
type MultiCall3Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiCall3Transactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiCall3Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiCall3Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiCall3Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiCall3Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiCall3Session struct {
	Contract     *MultiCall3       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultiCall3CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiCall3CallerSession struct {
	Contract *MultiCall3Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// MultiCall3TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiCall3TransactorSession struct {
	Contract     *MultiCall3Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// MultiCall3Raw is an auto generated low-level Go binding around an Ethereum contract.
type MultiCall3Raw struct {
	Contract *MultiCall3 // Generic contract binding to access the raw methods on
}

// MultiCall3CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiCall3CallerRaw struct {
	Contract *MultiCall3Caller // Generic read-only contract binding to access the raw methods on
}

// MultiCall3TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiCall3TransactorRaw struct {
	Contract *MultiCall3Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiCall3 creates a new instance of MultiCall3, bound to a specific deployed contract.
func NewMultiCall3(address common.Address, backend bind.ContractBackend) (*MultiCall3, error) {
	contract, err := bindMultiCall3(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiCall3{MultiCall3Caller: MultiCall3Caller{contract: contract}, MultiCall3Transactor: MultiCall3Transactor{contract: contract}, MultiCall3Filterer: MultiCall3Filterer{contract: contract}}, nil
}

// NewMultiCall3Caller creates a new read-only instance of MultiCall3, bound to a specific deployed contract.
func NewMultiCall3Caller(address common.Address, caller bind.ContractCaller) (*MultiCall3Caller, error) {
	contract, err := bindMultiCall3(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiCall3Caller{contract: contract}, nil
}

// NewMultiCall3Transactor creates a new write-only instance of MultiCall3, bound to a specific deployed contract.
func NewMultiCall3Transactor(address common.Address, transactor bind.ContractTransactor) (*MultiCall3Transactor, error) {
	contract, err := bindMultiCall3(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiCall3Transactor{contract: contract}, nil
}

// NewMultiCall3Filterer creates a new log filterer instance of MultiCall3, bound to a specific deployed contract.
func NewMultiCall3Filterer(address common.Address, filterer bind.ContractFilterer) (*MultiCall3Filterer, error) {
	contract, err := bindMultiCall3(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiCall3Filterer{contract: contract}, nil
}

// bindMultiCall3 binds a generic wrapper to an already deployed contract.
func bindMultiCall3(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiCall3ABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiCall3 *MultiCall3Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiCall3.Contract.MultiCall3Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiCall3 *MultiCall3Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiCall3.Contract.MultiCall3Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiCall3 *MultiCall3Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiCall3.Contract.MultiCall3Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiCall3 *MultiCall3CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiCall3.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiCall3 *MultiCall3TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiCall3.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiCall3 *MultiCall3TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiCall3.Contract.contract.Transact(opts, method, params...)
}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_MultiCall3 *MultiCall3Caller) GetBasefee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getBasefee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_MultiCall3 *MultiCall3Session) GetBasefee() (*big.Int, error) {
	return _MultiCall3.Contract.GetBasefee(&_MultiCall3.CallOpts)
}

// GetBasefee is a free data retrieval call binding the contract method 0x3e64a696.
//
// Solidity: function getBasefee() view returns(uint256 basefee)
func (_MultiCall3 *MultiCall3CallerSession) GetBasefee() (*big.Int, error) {
	return _MultiCall3.Contract.GetBasefee(&_MultiCall3.CallOpts)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3Caller) GetBlockHash(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getBlockHash", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3Session) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _MultiCall3.Contract.GetBlockHash(&_MultiCall3.CallOpts, blockNumber)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3CallerSession) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _MultiCall3.Contract.GetBlockHash(&_MultiCall3.CallOpts, blockNumber)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_MultiCall3 *MultiCall3Caller) GetBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_MultiCall3 *MultiCall3Session) GetBlockNumber() (*big.Int, error) {
	return _MultiCall3.Contract.GetBlockNumber(&_MultiCall3.CallOpts)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_MultiCall3 *MultiCall3CallerSession) GetBlockNumber() (*big.Int, error) {
	return _MultiCall3.Contract.GetBlockNumber(&_MultiCall3.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_MultiCall3 *MultiCall3Caller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_MultiCall3 *MultiCall3Session) GetChainId() (*big.Int, error) {
	return _MultiCall3.Contract.GetChainId(&_MultiCall3.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainid)
func (_MultiCall3 *MultiCall3CallerSession) GetChainId() (*big.Int, error) {
	return _MultiCall3.Contract.GetChainId(&_MultiCall3.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_MultiCall3 *MultiCall3Caller) GetCurrentBlockCoinbase(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getCurrentBlockCoinbase")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_MultiCall3 *MultiCall3Session) GetCurrentBlockCoinbase() (common.Address, error) {
	return _MultiCall3.Contract.GetCurrentBlockCoinbase(&_MultiCall3.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_MultiCall3 *MultiCall3CallerSession) GetCurrentBlockCoinbase() (common.Address, error) {
	return _MultiCall3.Contract.GetCurrentBlockCoinbase(&_MultiCall3.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_MultiCall3 *MultiCall3Caller) GetCurrentBlockDifficulty(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getCurrentBlockDifficulty")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_MultiCall3 *MultiCall3Session) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockDifficulty(&_MultiCall3.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_MultiCall3 *MultiCall3CallerSession) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockDifficulty(&_MultiCall3.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_MultiCall3 *MultiCall3Caller) GetCurrentBlockGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getCurrentBlockGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_MultiCall3 *MultiCall3Session) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockGasLimit(&_MultiCall3.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_MultiCall3 *MultiCall3CallerSession) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockGasLimit(&_MultiCall3.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_MultiCall3 *MultiCall3Caller) GetCurrentBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getCurrentBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_MultiCall3 *MultiCall3Session) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockTimestamp(&_MultiCall3.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_MultiCall3 *MultiCall3CallerSession) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _MultiCall3.Contract.GetCurrentBlockTimestamp(&_MultiCall3.CallOpts)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_MultiCall3 *MultiCall3Caller) GetEthBalance(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getEthBalance", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_MultiCall3 *MultiCall3Session) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _MultiCall3.Contract.GetEthBalance(&_MultiCall3.CallOpts, addr)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_MultiCall3 *MultiCall3CallerSession) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _MultiCall3.Contract.GetEthBalance(&_MultiCall3.CallOpts, addr)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3Caller) GetLastBlockHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _MultiCall3.contract.Call(opts, &out, "getLastBlockHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3Session) GetLastBlockHash() ([32]byte, error) {
	return _MultiCall3.Contract.GetLastBlockHash(&_MultiCall3.CallOpts)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_MultiCall3 *MultiCall3CallerSession) GetLastBlockHash() ([32]byte, error) {
	return _MultiCall3.Contract.GetLastBlockHash(&_MultiCall3.CallOpts)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_MultiCall3 *MultiCall3Transactor) Aggregate(opts *bind.TransactOpts, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "aggregate", calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_MultiCall3 *MultiCall3Session) Aggregate(calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate(&_MultiCall3.TransactOpts, calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) Aggregate(calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate(&_MultiCall3.TransactOpts, calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Transactor) Aggregate3(opts *bind.TransactOpts, calls []Multicall3Call3) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "aggregate3", calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Session) Aggregate3(calls []Multicall3Call3) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate3(&_MultiCall3.TransactOpts, calls)
}

// Aggregate3 is a paid mutator transaction binding the contract method 0x82ad56cb.
//
// Solidity: function aggregate3((address,bool,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) Aggregate3(calls []Multicall3Call3) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate3(&_MultiCall3.TransactOpts, calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Transactor) Aggregate3Value(opts *bind.TransactOpts, calls []Multicall3Call3Value) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "aggregate3Value", calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Session) Aggregate3Value(calls []Multicall3Call3Value) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate3Value(&_MultiCall3.TransactOpts, calls)
}

// Aggregate3Value is a paid mutator transaction binding the contract method 0x174dea71.
//
// Solidity: function aggregate3Value((address,bool,uint256,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) Aggregate3Value(calls []Multicall3Call3Value) (*types.Transaction, error) {
	return _MultiCall3.Contract.Aggregate3Value(&_MultiCall3.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Transactor) BlockAndAggregate(opts *bind.TransactOpts, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "blockAndAggregate", calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Session) BlockAndAggregate(calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.BlockAndAggregate(&_MultiCall3.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) BlockAndAggregate(calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.BlockAndAggregate(&_MultiCall3.TransactOpts, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Transactor) TryAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "tryAggregate", requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Session) TryAggregate(requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.TryAggregate(&_MultiCall3.TransactOpts, requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns((bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) TryAggregate(requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.TryAggregate(&_MultiCall3.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Transactor) TryBlockAndAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.contract.Transact(opts, "tryBlockAndAggregate", requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3Session) TryBlockAndAggregate(requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.TryBlockAndAggregate(&_MultiCall3.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) payable returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_MultiCall3 *MultiCall3TransactorSession) TryBlockAndAggregate(requireSuccess bool, calls []Multicall3Call) (*types.Transaction, error) {
	return _MultiCall3.Contract.TryBlockAndAggregate(&_MultiCall3.TransactOpts, requireSuccess, calls)
}
