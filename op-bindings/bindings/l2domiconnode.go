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

// DomiconNodeNodeInfo is an auto generated low-level Go binding around an user-defined struct.
type DomiconNodeNodeInfo struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}

// L2DomiconNodeMetaData contains all meta data concerning the L2DomiconNode contract.
var L2DomiconNodeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"_otherNode\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"}],\"name\":\"BroadcastNode\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"indexed\":false,\"internalType\":\"structDomiconNode.NodeInfo\",\"name\":\"nodeInfo\",\"type\":\"tuple\"}],\"name\":\"FinalizeBroadcastNode\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BROADCAST_NODES\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"IsNodeBroadcast\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STORAGE_NODES\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"broadcastNodeList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"broadcastingNodes\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"node\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"internalType\":\"structDomiconNode.NodeInfo\",\"name\":\"nodeInfo\",\"type\":\"tuple\"}],\"name\":\"finalizeBroadcastNode\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherDomiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"storageNodeList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"storageNodes\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405260006001553480156200001657600080fd5b5060405162001682380380620016828339810160408190526200003991620001e9565b6001600160a01b0381166080526200005062000057565b506200021b565b600054600390610100900460ff161580156200007a575060005460ff8083169116105b620000e35760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620001167342000000000000000000000000000000000000076200015a565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b600054610100900460ff16620001c75760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000da565b600880546001600160a01b0319166001600160a01b0392909216919091179055565b600060208284031215620001fc57600080fd5b81516001600160a01b03811681146200021457600080fd5b9392505050565b6080516114366200024c600039600081816102890152818161031f0152818161045e01526109e101526114366000f3fe6080604052600436106100dd5760003560e01c80638129fc1c1161007f578063c0f2acea11610059578063c0f2acea146102ab578063cac5800a146102db578063db2b9423146102fb578063e261fba31461031057600080fd5b80638129fc1c14610237578063927ede2d1461024c5780639b6679931461027757600080fd5b80636a4fe4a8116100bb5780636a4fe4a8146101af5780637667d104146101d15780637cce10a4146101f15780638118eb331461020657600080fd5b80633cb747bf146100e257806354fd4d501461013957806356f7be581461018f575b600080fd5b3480156100ee57600080fd5b5060085461010f9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561014557600080fd5b506101826040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b6040516101309190610d2c565b34801561019b57600080fd5b5061010f6101aa366004610d46565b610343565b3480156101bb57600080fd5b506101c461037a565b6040516101309190610d5f565b3480156101dd57600080fd5b5061010f6101ec366004610d46565b6103e9565b6102046101ff366004610dde565b6103f9565b005b34801561021257600080fd5b50610226610221366004610e35565b6105e5565b604051610130959493929190610e52565b34801561024357600080fd5b5061020461073b565b34801561025857600080fd5b5060085473ffffffffffffffffffffffffffffffffffffffff1661010f565b34801561028357600080fd5b5061010f7f000000000000000000000000000000000000000000000000000000000000000081565b3480156102b757600080fd5b506102cb6102c6366004610e35565b610898565b6040519015158152602001610130565b3480156102e757600080fd5b506102266102f6366004610e35565b6108d6565b34801561030757600080fd5b506101c461090f565b34801561031c57600080fd5b507f000000000000000000000000000000000000000000000000000000000000000061010f565b6005818154811061035357600080fd5b60009182526020909120015473ffffffffffffffffffffffffffffffffffffffff16905081565b606060058054806020026020016040519081016040528092919081815260200182805480156103df57602002820191906000526020600020905b815473ffffffffffffffffffffffffffffffffffffffff1681526001909101906020018083116103b4575b5050505050905090565b6007818154811061035357600080fd5b60085473ffffffffffffffffffffffffffffffffffffffff16331480156104e85750600854604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000008116931691636e296e459160048083019260209291908290030181865afa1580156104ac573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104d09190610ea8565b73ffffffffffffffffffffffffffffffffffffffff16145b6105a0576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526044602482018190527f446f6d69636f6e4e6f64653a2066756e6374696f6e2063616e206f6e6c792062908201527f652063616c6c65642066726f6d20746865206f7468657220646f6d69636f6e2060648201527f6e6f646500000000000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b7f59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71816040516105cf9190610f79565b60405180910390a16105e1828261097c565b5050565b6006602052600090815260409020805460018201805473ffffffffffffffffffffffffffffffffffffffff909216929161061e90611030565b80601f016020809104026020016040519081016040528092919081815260200182805461064a90611030565b80156106975780601f1061066c57610100808354040283529160200191610697565b820191906000526020600020905b81548152906001019060200180831161067a57829003601f168201915b5050505050908060020180546106ac90611030565b80601f01602080910402602001604051908101604052809291908181526020018280546106d890611030565b80156107255780601f106106fa57610100808354040283529160200191610725565b820191906000526020600020905b81548152906001019060200180831161070857829003601f168201915b5050505050908060030154908060040154905085565b600054600390610100900460ff1615801561075d575060005460ff8083169116105b6107e9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610597565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff831617610100179055610837734200000000000000000000000000000000000007610be3565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b73ffffffffffffffffffffffffffffffffffffffff8116600090815260046020526040812060030154156108ce57506001919050565b506000919050565b6004602052600090815260409020805460018201805473ffffffffffffffffffffffffffffffffffffffff909216929161061e90611030565b606060078054806020026020016040519081016040528092919081815260200182805480156103df5760200282019190600052602060002090815473ffffffffffffffffffffffffffffffffffffffff1681526001909101906020018083116103b4575050505050905090565b60085473ffffffffffffffffffffffffffffffffffffffff1633148015610a6b5750600854604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000008116931691636e296e459160048083019260209291908290030181865afa158015610a2f573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610a539190610ea8565b73ffffffffffffffffffffffffffffffffffffffff16145b610b1e576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526044602482018190527f446f6d69636f6e4e6f64653a2066756e6374696f6e2063616e206f6e6c792062908201527f652063616c6c65642066726f6d20746865206f7468657220646f6d69636f6e2060648201527f6e6f646500000000000000000000000000000000000000000000000000000000608482015260a401610597565b7f59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b7181604051610b4d9190610f79565b60405180910390a160058054600181019091557f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db00180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff841690811790915560009081526004602052604090208190610bdd8282611281565b50505050565b600054610100900460ff16610c7a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610597565b600880547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b6000815180845260005b81811015610ce757602081850181015186830182015201610ccb565b81811115610cf9576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610d3f6020830184610cc1565b9392505050565b600060208284031215610d5857600080fd5b5035919050565b6020808252825182820181905260009190848201906040850190845b81811015610dad57835173ffffffffffffffffffffffffffffffffffffffff1683529284019291840191600101610d7b565b50909695505050505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610ddb57600080fd5b50565b60008060408385031215610df157600080fd5b8235610dfc81610db9565b9150602083013567ffffffffffffffff811115610e1857600080fd5b830160a08186031215610e2a57600080fd5b809150509250929050565b600060208284031215610e4757600080fd5b8135610d3f81610db9565b73ffffffffffffffffffffffffffffffffffffffff8616815260a060208201526000610e8160a0830187610cc1565b8281036040840152610e938187610cc1565b60608401959095525050608001529392505050565b600060208284031215610eba57600080fd5b8151610d3f81610db9565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610efa57600080fd5b830160208101925035905067ffffffffffffffff811115610f1a57600080fd5b803603821315610f2957600080fd5b9250929050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b6020815260008235610f8a81610db9565b73ffffffffffffffffffffffffffffffffffffffff8116602084015250610fb46020840184610ec5565b60a06040850152610fc960c085018284610f30565b915050610fd96040850185610ec5565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe085840301606086015261100e838284610f30565b9250505060608401356080840152608084013560a08401528091505092915050565b600181811c9082168061104457607f821691505b60208210810361107d577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126110b857600080fd5b83018035915067ffffffffffffffff8211156110d357600080fd5b602001915036819003821315610f2957600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b601f82111561116157600081815260208120601f850160051c8101602086101561113e5750805b601f850160051c820191505b8181101561115d5782815560010161114a565b5050505b505050565b67ffffffffffffffff83111561117e5761117e6110e8565b6111928361118c8354611030565b83611117565b6000601f8411600181146111e457600085156111ae5750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600387901b1c1916600186901b17835561127a565b6000838152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0861690835b828110156112335786850135825560209485019460019092019101611213565b508682101561126e577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88860031b161c19848701351681555b505060018560011b0183555b5050505050565b813561128c81610db9565b73ffffffffffffffffffffffffffffffffffffffff81167fffffffffffffffffffffffff000000000000000000000000000000000000000083541617825550600180820160206112de81860186611083565b67ffffffffffffffff8111156112f6576112f66110e8565b61130a816113048654611030565b86611117565b6000601f82116001811461135c57600083156113265750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600385901b1c1916600184901b1786556113ed565b6000868152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0841690835b828110156113a8578685013582559387019390890190870161138b565b50848210156113e3577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88660031b161c19848701351681555b50508683881b0186555b505050505050506114016040830183611083565b61140f818360028601611166565b50506060820135600382015560808201356004820155505056fea164736f6c634300080f000a",
}

// L2DomiconNodeABI is the input ABI used to generate the binding from.
// Deprecated: Use L2DomiconNodeMetaData.ABI instead.
var L2DomiconNodeABI = L2DomiconNodeMetaData.ABI

// L2DomiconNodeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2DomiconNodeMetaData.Bin instead.
var L2DomiconNodeBin = L2DomiconNodeMetaData.Bin

// DeployL2DomiconNode deploys a new Ethereum contract, binding an instance of L2DomiconNode to it.
func DeployL2DomiconNode(auth *bind.TransactOpts, backend bind.ContractBackend, _otherNode common.Address) (common.Address, *types.Transaction, *L2DomiconNode, error) {
	parsed, err := L2DomiconNodeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2DomiconNodeBin), backend, _otherNode)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2DomiconNode{L2DomiconNodeCaller: L2DomiconNodeCaller{contract: contract}, L2DomiconNodeTransactor: L2DomiconNodeTransactor{contract: contract}, L2DomiconNodeFilterer: L2DomiconNodeFilterer{contract: contract}}, nil
}

// L2DomiconNode is an auto generated Go binding around an Ethereum contract.
type L2DomiconNode struct {
	L2DomiconNodeCaller     // Read-only binding to the contract
	L2DomiconNodeTransactor // Write-only binding to the contract
	L2DomiconNodeFilterer   // Log filterer for contract events
}

// L2DomiconNodeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2DomiconNodeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconNodeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2DomiconNodeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconNodeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2DomiconNodeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2DomiconNodeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2DomiconNodeSession struct {
	Contract     *L2DomiconNode    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2DomiconNodeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2DomiconNodeCallerSession struct {
	Contract *L2DomiconNodeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// L2DomiconNodeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2DomiconNodeTransactorSession struct {
	Contract     *L2DomiconNodeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// L2DomiconNodeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2DomiconNodeRaw struct {
	Contract *L2DomiconNode // Generic contract binding to access the raw methods on
}

// L2DomiconNodeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2DomiconNodeCallerRaw struct {
	Contract *L2DomiconNodeCaller // Generic read-only contract binding to access the raw methods on
}

// L2DomiconNodeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2DomiconNodeTransactorRaw struct {
	Contract *L2DomiconNodeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2DomiconNode creates a new instance of L2DomiconNode, bound to a specific deployed contract.
func NewL2DomiconNode(address common.Address, backend bind.ContractBackend) (*L2DomiconNode, error) {
	contract, err := bindL2DomiconNode(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2DomiconNode{L2DomiconNodeCaller: L2DomiconNodeCaller{contract: contract}, L2DomiconNodeTransactor: L2DomiconNodeTransactor{contract: contract}, L2DomiconNodeFilterer: L2DomiconNodeFilterer{contract: contract}}, nil
}

// NewL2DomiconNodeCaller creates a new read-only instance of L2DomiconNode, bound to a specific deployed contract.
func NewL2DomiconNodeCaller(address common.Address, caller bind.ContractCaller) (*L2DomiconNodeCaller, error) {
	contract, err := bindL2DomiconNode(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeCaller{contract: contract}, nil
}

// NewL2DomiconNodeTransactor creates a new write-only instance of L2DomiconNode, bound to a specific deployed contract.
func NewL2DomiconNodeTransactor(address common.Address, transactor bind.ContractTransactor) (*L2DomiconNodeTransactor, error) {
	contract, err := bindL2DomiconNode(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeTransactor{contract: contract}, nil
}

// NewL2DomiconNodeFilterer creates a new log filterer instance of L2DomiconNode, bound to a specific deployed contract.
func NewL2DomiconNodeFilterer(address common.Address, filterer bind.ContractFilterer) (*L2DomiconNodeFilterer, error) {
	contract, err := bindL2DomiconNode(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeFilterer{contract: contract}, nil
}

// bindL2DomiconNode binds a generic wrapper to an already deployed contract.
func bindL2DomiconNode(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L2DomiconNodeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2DomiconNode *L2DomiconNodeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2DomiconNode.Contract.L2DomiconNodeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2DomiconNode *L2DomiconNodeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.L2DomiconNodeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2DomiconNode *L2DomiconNodeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.L2DomiconNodeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2DomiconNode *L2DomiconNodeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2DomiconNode.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2DomiconNode *L2DomiconNodeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2DomiconNode *L2DomiconNodeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.contract.Transact(opts, method, params...)
}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeCaller) BROADCASTNODES(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "BROADCAST_NODES")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeSession) BROADCASTNODES() ([]common.Address, error) {
	return _L2DomiconNode.Contract.BROADCASTNODES(&_L2DomiconNode.CallOpts)
}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeCallerSession) BROADCASTNODES() ([]common.Address, error) {
	return _L2DomiconNode.Contract.BROADCASTNODES(&_L2DomiconNode.CallOpts)
}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L2DomiconNode *L2DomiconNodeCaller) IsNodeBroadcast(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "IsNodeBroadcast", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L2DomiconNode *L2DomiconNodeSession) IsNodeBroadcast(addr common.Address) (bool, error) {
	return _L2DomiconNode.Contract.IsNodeBroadcast(&_L2DomiconNode.CallOpts, addr)
}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L2DomiconNode *L2DomiconNodeCallerSession) IsNodeBroadcast(addr common.Address) (bool, error) {
	return _L2DomiconNode.Contract.IsNodeBroadcast(&_L2DomiconNode.CallOpts, addr)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) MESSENGER() (common.Address, error) {
	return _L2DomiconNode.Contract.MESSENGER(&_L2DomiconNode.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) MESSENGER() (common.Address, error) {
	return _L2DomiconNode.Contract.MESSENGER(&_L2DomiconNode.CallOpts)
}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) OTHERDOMICONNODE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "OTHER_DOMICON_NODE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) OTHERDOMICONNODE() (common.Address, error) {
	return _L2DomiconNode.Contract.OTHERDOMICONNODE(&_L2DomiconNode.CallOpts)
}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) OTHERDOMICONNODE() (common.Address, error) {
	return _L2DomiconNode.Contract.OTHERDOMICONNODE(&_L2DomiconNode.CallOpts)
}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeCaller) STORAGENODES(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "STORAGE_NODES")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeSession) STORAGENODES() ([]common.Address, error) {
	return _L2DomiconNode.Contract.STORAGENODES(&_L2DomiconNode.CallOpts)
}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L2DomiconNode *L2DomiconNodeCallerSession) STORAGENODES() ([]common.Address, error) {
	return _L2DomiconNode.Contract.STORAGENODES(&_L2DomiconNode.CallOpts)
}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) BroadcastNodeList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "broadcastNodeList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) BroadcastNodeList(arg0 *big.Int) (common.Address, error) {
	return _L2DomiconNode.Contract.BroadcastNodeList(&_L2DomiconNode.CallOpts, arg0)
}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) BroadcastNodeList(arg0 *big.Int) (common.Address, error) {
	return _L2DomiconNode.Contract.BroadcastNodeList(&_L2DomiconNode.CallOpts, arg0)
}

// BroadcastingNodes is a free data retrieval call binding the contract method 0xcac5800a.
//
// Solidity: function broadcastingNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeCaller) BroadcastingNodes(opts *bind.CallOpts, arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "broadcastingNodes", arg0)

	outstruct := new(struct {
		Add          common.Address
		Rpc          string
		Name         string
		StakedTokens *big.Int
		Index        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Add = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Rpc = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Name = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.StakedTokens = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Index = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// BroadcastingNodes is a free data retrieval call binding the contract method 0xcac5800a.
//
// Solidity: function broadcastingNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeSession) BroadcastingNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L2DomiconNode.Contract.BroadcastingNodes(&_L2DomiconNode.CallOpts, arg0)
}

// BroadcastingNodes is a free data retrieval call binding the contract method 0xcac5800a.
//
// Solidity: function broadcastingNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeCallerSession) BroadcastingNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L2DomiconNode.Contract.BroadcastingNodes(&_L2DomiconNode.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) Messenger() (common.Address, error) {
	return _L2DomiconNode.Contract.Messenger(&_L2DomiconNode.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) Messenger() (common.Address, error) {
	return _L2DomiconNode.Contract.Messenger(&_L2DomiconNode.CallOpts)
}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) OtherDomiconNode(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "otherDomiconNode")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) OtherDomiconNode() (common.Address, error) {
	return _L2DomiconNode.Contract.OtherDomiconNode(&_L2DomiconNode.CallOpts)
}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) OtherDomiconNode() (common.Address, error) {
	return _L2DomiconNode.Contract.OtherDomiconNode(&_L2DomiconNode.CallOpts)
}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeCaller) StorageNodeList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "storageNodeList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeSession) StorageNodeList(arg0 *big.Int) (common.Address, error) {
	return _L2DomiconNode.Contract.StorageNodeList(&_L2DomiconNode.CallOpts, arg0)
}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L2DomiconNode *L2DomiconNodeCallerSession) StorageNodeList(arg0 *big.Int) (common.Address, error) {
	return _L2DomiconNode.Contract.StorageNodeList(&_L2DomiconNode.CallOpts, arg0)
}

// StorageNodes is a free data retrieval call binding the contract method 0x8118eb33.
//
// Solidity: function storageNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeCaller) StorageNodes(opts *bind.CallOpts, arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "storageNodes", arg0)

	outstruct := new(struct {
		Add          common.Address
		Rpc          string
		Name         string
		StakedTokens *big.Int
		Index        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Add = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Rpc = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Name = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.StakedTokens = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Index = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// StorageNodes is a free data retrieval call binding the contract method 0x8118eb33.
//
// Solidity: function storageNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeSession) StorageNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L2DomiconNode.Contract.StorageNodes(&_L2DomiconNode.CallOpts, arg0)
}

// StorageNodes is a free data retrieval call binding the contract method 0x8118eb33.
//
// Solidity: function storageNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L2DomiconNode *L2DomiconNodeCallerSession) StorageNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L2DomiconNode.Contract.StorageNodes(&_L2DomiconNode.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconNode *L2DomiconNodeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2DomiconNode.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconNode *L2DomiconNodeSession) Version() (string, error) {
	return _L2DomiconNode.Contract.Version(&_L2DomiconNode.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2DomiconNode *L2DomiconNodeCallerSession) Version() (string, error) {
	return _L2DomiconNode.Contract.Version(&_L2DomiconNode.CallOpts)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L2DomiconNode *L2DomiconNodeTransactor) FinalizeBroadcastNode(opts *bind.TransactOpts, node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L2DomiconNode.contract.Transact(opts, "finalizeBroadcastNode", node, nodeInfo)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L2DomiconNode *L2DomiconNodeSession) FinalizeBroadcastNode(node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.FinalizeBroadcastNode(&_L2DomiconNode.TransactOpts, node, nodeInfo)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L2DomiconNode *L2DomiconNodeTransactorSession) FinalizeBroadcastNode(node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L2DomiconNode.Contract.FinalizeBroadcastNode(&_L2DomiconNode.TransactOpts, node, nodeInfo)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconNode *L2DomiconNodeTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2DomiconNode.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconNode *L2DomiconNodeSession) Initialize() (*types.Transaction, error) {
	return _L2DomiconNode.Contract.Initialize(&_L2DomiconNode.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_L2DomiconNode *L2DomiconNodeTransactorSession) Initialize() (*types.Transaction, error) {
	return _L2DomiconNode.Contract.Initialize(&_L2DomiconNode.TransactOpts)
}

// L2DomiconNodeBroadcastNodeIterator is returned from FilterBroadcastNode and is used to iterate over the raw logs and unpacked data for BroadcastNode events raised by the L2DomiconNode contract.
type L2DomiconNodeBroadcastNodeIterator struct {
	Event *L2DomiconNodeBroadcastNode // Event containing the contract specifics and raw log

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
func (it *L2DomiconNodeBroadcastNodeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconNodeBroadcastNode)
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
		it.Event = new(L2DomiconNodeBroadcastNode)
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
func (it *L2DomiconNodeBroadcastNodeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconNodeBroadcastNodeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconNodeBroadcastNode represents a BroadcastNode event raised by the L2DomiconNode contract.
type L2DomiconNodeBroadcastNode struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBroadcastNode is a free log retrieval operation binding the contract event 0xf81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf82616815.
//
// Solidity: event BroadcastNode(address indexed add, string rpc, string name, uint256 stakedTokens)
func (_L2DomiconNode *L2DomiconNodeFilterer) FilterBroadcastNode(opts *bind.FilterOpts, add []common.Address) (*L2DomiconNodeBroadcastNodeIterator, error) {

	var addRule []interface{}
	for _, addItem := range add {
		addRule = append(addRule, addItem)
	}

	logs, sub, err := _L2DomiconNode.contract.FilterLogs(opts, "BroadcastNode", addRule)
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeBroadcastNodeIterator{contract: _L2DomiconNode.contract, event: "BroadcastNode", logs: logs, sub: sub}, nil
}

// WatchBroadcastNode is a free log subscription operation binding the contract event 0xf81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf82616815.
//
// Solidity: event BroadcastNode(address indexed add, string rpc, string name, uint256 stakedTokens)
func (_L2DomiconNode *L2DomiconNodeFilterer) WatchBroadcastNode(opts *bind.WatchOpts, sink chan<- *L2DomiconNodeBroadcastNode, add []common.Address) (event.Subscription, error) {

	var addRule []interface{}
	for _, addItem := range add {
		addRule = append(addRule, addItem)
	}

	logs, sub, err := _L2DomiconNode.contract.WatchLogs(opts, "BroadcastNode", addRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconNodeBroadcastNode)
				if err := _L2DomiconNode.contract.UnpackLog(event, "BroadcastNode", log); err != nil {
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

// ParseBroadcastNode is a log parse operation binding the contract event 0xf81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf82616815.
//
// Solidity: event BroadcastNode(address indexed add, string rpc, string name, uint256 stakedTokens)
func (_L2DomiconNode *L2DomiconNodeFilterer) ParseBroadcastNode(log types.Log) (*L2DomiconNodeBroadcastNode, error) {
	event := new(L2DomiconNodeBroadcastNode)
	if err := _L2DomiconNode.contract.UnpackLog(event, "BroadcastNode", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2DomiconNodeFinalizeBroadcastNodeIterator is returned from FilterFinalizeBroadcastNode and is used to iterate over the raw logs and unpacked data for FinalizeBroadcastNode events raised by the L2DomiconNode contract.
type L2DomiconNodeFinalizeBroadcastNodeIterator struct {
	Event *L2DomiconNodeFinalizeBroadcastNode // Event containing the contract specifics and raw log

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
func (it *L2DomiconNodeFinalizeBroadcastNodeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconNodeFinalizeBroadcastNode)
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
		it.Event = new(L2DomiconNodeFinalizeBroadcastNode)
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
func (it *L2DomiconNodeFinalizeBroadcastNodeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconNodeFinalizeBroadcastNodeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconNodeFinalizeBroadcastNode represents a FinalizeBroadcastNode event raised by the L2DomiconNode contract.
type L2DomiconNodeFinalizeBroadcastNode struct {
	NodeInfo DomiconNodeNodeInfo
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFinalizeBroadcastNode is a free log retrieval operation binding the contract event 0x59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71.
//
// Solidity: event FinalizeBroadcastNode((address,string,string,uint256,uint256) nodeInfo)
func (_L2DomiconNode *L2DomiconNodeFilterer) FilterFinalizeBroadcastNode(opts *bind.FilterOpts) (*L2DomiconNodeFinalizeBroadcastNodeIterator, error) {

	logs, sub, err := _L2DomiconNode.contract.FilterLogs(opts, "FinalizeBroadcastNode")
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeFinalizeBroadcastNodeIterator{contract: _L2DomiconNode.contract, event: "FinalizeBroadcastNode", logs: logs, sub: sub}, nil
}

// WatchFinalizeBroadcastNode is a free log subscription operation binding the contract event 0x59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71.
//
// Solidity: event FinalizeBroadcastNode((address,string,string,uint256,uint256) nodeInfo)
func (_L2DomiconNode *L2DomiconNodeFilterer) WatchFinalizeBroadcastNode(opts *bind.WatchOpts, sink chan<- *L2DomiconNodeFinalizeBroadcastNode) (event.Subscription, error) {

	logs, sub, err := _L2DomiconNode.contract.WatchLogs(opts, "FinalizeBroadcastNode")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconNodeFinalizeBroadcastNode)
				if err := _L2DomiconNode.contract.UnpackLog(event, "FinalizeBroadcastNode", log); err != nil {
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

// ParseFinalizeBroadcastNode is a log parse operation binding the contract event 0x59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71.
//
// Solidity: event FinalizeBroadcastNode((address,string,string,uint256,uint256) nodeInfo)
func (_L2DomiconNode *L2DomiconNodeFilterer) ParseFinalizeBroadcastNode(log types.Log) (*L2DomiconNodeFinalizeBroadcastNode, error) {
	event := new(L2DomiconNodeFinalizeBroadcastNode)
	if err := _L2DomiconNode.contract.UnpackLog(event, "FinalizeBroadcastNode", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2DomiconNodeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2DomiconNode contract.
type L2DomiconNodeInitializedIterator struct {
	Event *L2DomiconNodeInitialized // Event containing the contract specifics and raw log

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
func (it *L2DomiconNodeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2DomiconNodeInitialized)
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
		it.Event = new(L2DomiconNodeInitialized)
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
func (it *L2DomiconNodeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2DomiconNodeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2DomiconNodeInitialized represents a Initialized event raised by the L2DomiconNode contract.
type L2DomiconNodeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2DomiconNode *L2DomiconNodeFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2DomiconNodeInitializedIterator, error) {

	logs, sub, err := _L2DomiconNode.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2DomiconNodeInitializedIterator{contract: _L2DomiconNode.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2DomiconNode *L2DomiconNodeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2DomiconNodeInitialized) (event.Subscription, error) {

	logs, sub, err := _L2DomiconNode.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2DomiconNodeInitialized)
				if err := _L2DomiconNode.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L2DomiconNode *L2DomiconNodeFilterer) ParseInitialized(log types.Log) (*L2DomiconNodeInitialized, error) {
	event := new(L2DomiconNodeInitialized)
	if err := _L2DomiconNode.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
