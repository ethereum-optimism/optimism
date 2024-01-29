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

// L1DomiconNodeMetaData contains all meta data concerning the L1DomiconNode contract.
var L1DomiconNodeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"}],\"name\":\"BroadcastNode\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"indexed\":false,\"internalType\":\"structDomiconNode.NodeInfo\",\"name\":\"nodeInfo\",\"type\":\"tuple\"}],\"name\":\"FinalizeBroadcastNode\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BROADCAST_NODES\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"IsNodeBroadcast\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_address\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"_stakedTokens\",\"type\":\"uint256\"}],\"name\":\"RegisterBroadcastNode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STORAGE_NODES\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"broadcastNodeList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"broadcastingNodes\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"node\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"internalType\":\"structDomiconNode.NodeInfo\",\"name\":\"nodeInfo\",\"type\":\"tuple\"}],\"name\":\"finalizeBroadcastNode\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherDomiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"storageNodeList\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"storageNodes\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"add\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"rpc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"stakedTokens\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405260006001553480156200001657600080fd5b507342000000000000000000000000000000000000236080526200003b600062000041565b620001c0565b600054600390610100900460ff1615801562000064575060005460ff8083169116105b620000cd5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620000ec8262000131565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b600054610100900460ff166200019e5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000c4565b600880546001600160a01b0319166001600160a01b0392909216919091179055565b6080516118bd620001f16000396000818161029f01528181610355015281816105fd0152610c4701526118bd6000f3fe6080604052600436106100e85760003560e01c80638118eb331161008a578063c4d66de811610059578063c4d66de8146102f1578063cac5800a14610311578063db2b942314610331578063e261fba31461034657600080fd5b80638118eb3314610231578063927ede2d146102625780639b6679931461028d578063c0f2acea146102c157600080fd5b806356f7be58116100c657806356f7be58146101bc5780636a4fe4a8146101dc5780637667d104146101fe5780637cce10a41461021e57600080fd5b80631d4c517f146100ed5780633cb747bf1461010f57806354fd4d5014610166575b600080fd5b3480156100f957600080fd5b5061010d610108366004610eb1565b610379565b005b34801561011b57600080fd5b5060085461013c9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561017257600080fd5b506101af6040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b60405161015d9190610fa7565b3480156101c857600080fd5b5061013c6101d7366004610fc1565b6104e2565b3480156101e857600080fd5b506101f1610519565b60405161015d9190610fda565b34801561020a57600080fd5b5061013c610219366004610fc1565b610588565b61010d61022c366004611034565b610598565b34801561023d57600080fd5b5061025161024c36600461108b565b610804565b60405161015d9594939291906110a8565b34801561026e57600080fd5b5060085473ffffffffffffffffffffffffffffffffffffffff1661013c565b34801561029957600080fd5b5061013c7f000000000000000000000000000000000000000000000000000000000000000081565b3480156102cd57600080fd5b506102e16102dc36600461108b565b61095a565b604051901515815260200161015d565b3480156102fd57600080fd5b5061010d61030c36600461108b565b610998565b34801561031d57600080fd5b5061025161032c36600461108b565b610ae2565b34801561033d57600080fd5b506101f1610b1b565b34801561035257600080fd5b507f000000000000000000000000000000000000000000000000000000000000000061013c565b8573ffffffffffffffffffffffffffffffffffffffff167ff81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf8261681586868686866040516103c7959493929190611147565b60405180910390a26005805460018101825560009182527f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db00180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff89169081179091556040805160a08101825291825280516020601f8901819004810282018101909252878152818301918990899081908401838280828437600092019190915250505090825250604080516020601f88018190048102820181019092528681529181019190879087908190840183828082843760009201829052509385525050506020820185905260409091015290506104d98782610b88565b50505050505050565b600581815481106104f257600080fd5b60009182526020909120015473ffffffffffffffffffffffffffffffffffffffff16905081565b6060600580548060200260200160405190810160405280929190818152602001828054801561057e57602002820191906000526020600020905b815473ffffffffffffffffffffffffffffffffffffffff168152600190910190602001808311610553575b5050505050905090565b600781815481106104f257600080fd5b60085473ffffffffffffffffffffffffffffffffffffffff16331480156106875750600854604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000008116931691636e296e459160048083019260209291908290030181865afa15801561064b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061066f9190611181565b73ffffffffffffffffffffffffffffffffffffffff16145b61073f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526044602482018190527f446f6d69636f6e4e6f64653a2066756e6374696f6e2063616e206f6e6c792062908201527f652063616c6c65642066726f6d20746865206f7468657220646f6d69636f6e2060648201527f6e6f646500000000000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b7f59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b718160405161076e9190611202565b60405180910390a160058054600181019091557f036b6384b5eca791c62761152d0c79bb0604c104a5fb6f4eb0703f3154bb3db00180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8416908117909155600090815260046020526040902081906107fe8282611506565b50505050565b6006602052600090815260409020805460018201805473ffffffffffffffffffffffffffffffffffffffff909216929161083d9061134d565b80601f01602080910402602001604051908101604052809291908181526020018280546108699061134d565b80156108b65780601f1061088b576101008083540402835291602001916108b6565b820191906000526020600020905b81548152906001019060200180831161089957829003601f168201915b5050505050908060020180546108cb9061134d565b80601f01602080910402602001604051908101604052809291908181526020018280546108f79061134d565b80156109445780601f1061091957610100808354040283529160200191610944565b820191906000526020600020905b81548152906001019060200180831161092757829003601f168201915b5050505050908060030154908060040154905085565b73ffffffffffffffffffffffffffffffffffffffff81166000908152600460205260408120600301541561099057506001919050565b506000919050565b600054600390610100900460ff161580156109ba575060005460ff8083169116105b610a46576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610736565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff831617610100179055610a8082610d65565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6004602052600090815260409020805460018201805473ffffffffffffffffffffffffffffffffffffffff909216929161083d9061134d565b6060600780548060200260200160405190810160405280929190818152602001828054801561057e5760200282019190600052602060002090815473ffffffffffffffffffffffffffffffffffffffff168152600190910190602001808311610553575050505050905090565b73ffffffffffffffffffffffffffffffffffffffff8281166000908152600460209081526040909120835181547fffffffffffffffffffffffff00000000000000000000000000000000000000001693169290921782558201518291906001820190610bf490826116ae565b5060408201516002820190610c0990826116ae565b506060820151600382015560809091015160049091015560085460405173ffffffffffffffffffffffffffffffffffffffff90911690633dbb202b907f0000000000000000000000000000000000000000000000000000000000000000907f7cce10a40000000000000000000000000000000000000000000000000000000090610c9990879087906024016117c8565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b9092168252610d2f929162030d409060040161186b565b600060405180830381600087803b158015610d4957600080fd5b505af1158015610d5d573d6000803e3d6000fd5b505050505050565b600054610100900460ff16610dfc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610736565b600880547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b73ffffffffffffffffffffffffffffffffffffffff81168114610e6557600080fd5b50565b60008083601f840112610e7a57600080fd5b50813567ffffffffffffffff811115610e9257600080fd5b602083019150836020828501011115610eaa57600080fd5b9250929050565b60008060008060008060808789031215610eca57600080fd5b8635610ed581610e43565b9550602087013567ffffffffffffffff80821115610ef257600080fd5b610efe8a838b01610e68565b90975095506040890135915080821115610f1757600080fd5b50610f2489828a01610e68565b979a9699509497949695606090950135949350505050565b6000815180845260005b81811015610f6257602081850181015186830182015201610f46565b81811115610f74576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610fba6020830184610f3c565b9392505050565b600060208284031215610fd357600080fd5b5035919050565b6020808252825182820181905260009190848201906040850190845b8181101561102857835173ffffffffffffffffffffffffffffffffffffffff1683529284019291840191600101610ff6565b50909695505050505050565b6000806040838503121561104757600080fd5b823561105281610e43565b9150602083013567ffffffffffffffff81111561106e57600080fd5b830160a0818603121561108057600080fd5b809150509250929050565b60006020828403121561109d57600080fd5b8135610fba81610e43565b73ffffffffffffffffffffffffffffffffffffffff8616815260a0602082015260006110d760a0830187610f3c565b82810360408401526110e98187610f3c565b60608401959095525050608001529392505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b60608152600061115b6060830187896110fe565b828103602084015261116e8186886110fe565b9150508260408301529695505050505050565b60006020828403121561119357600080fd5b8151610fba81610e43565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126111d357600080fd5b830160208101925035905067ffffffffffffffff8111156111f357600080fd5b803603821315610eaa57600080fd5b602081526000823561121381610e43565b73ffffffffffffffffffffffffffffffffffffffff811660208401525061123d602084018461119e565b60a0604085015261125260c0850182846110fe565b915050611262604085018561119e565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08584030160608601526112978382846110fe565b9250505060608401356080840152608084013560a08401528091505092915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126112ee57600080fd5b83018035915067ffffffffffffffff82111561130957600080fd5b602001915036819003821315610eaa57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c9082168061136157607f821691505b60208210810361139a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f8211156113e657600081815260208120601f850160051c810160208610156113c75750805b601f850160051c820191505b81811015610d5d578281556001016113d3565b505050565b67ffffffffffffffff8311156114035761140361131e565b61141783611411835461134d565b836113a0565b6000601f84116001811461146957600085156114335750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600387901b1c1916600186901b1783556114ff565b6000838152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0861690835b828110156114b85786850135825560209485019460019092019101611498565b50868210156114f3577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88860031b161c19848701351681555b505060018560011b0183555b5050505050565b813561151181610e43565b73ffffffffffffffffffffffffffffffffffffffff81167fffffffffffffffffffffffff00000000000000000000000000000000000000008354161782555060018082016020611563818601866112b9565b67ffffffffffffffff81111561157b5761157b61131e565b61158f81611589865461134d565b866113a0565b6000601f8211600181146115e157600083156115ab5750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600385901b1c1916600184901b178655611672565b6000868152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0841690835b8281101561162d5786850135825593870193908901908701611610565b5084821015611668577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88660031b161c19848701351681555b50508683881b0186555b5050505050505061168660408301836112b9565b6116948183600286016113eb565b505060608201356003820155608082013560048201555050565b815167ffffffffffffffff8111156116c8576116c861131e565b6116dc816116d6845461134d565b846113a0565b602080601f83116001811461172f57600084156116f95750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555610d5d565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561177c5788860151825594840194600190910190840161175d565b50858210156117b857878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b600073ffffffffffffffffffffffffffffffffffffffff80851683526040602084015280845116604084015250602083015160a0606084015261180e60e0840182610f3c565b905060408401517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc08483030160808501526118498282610f3c565b915050606084015160a0840152608084015160c0840152809150509392505050565b73ffffffffffffffffffffffffffffffffffffffff8416815260606020820152600061189a6060830185610f3c565b905063ffffffff8316604083015294935050505056fea164736f6c634300080f000a",
}

// L1DomiconNodeABI is the input ABI used to generate the binding from.
// Deprecated: Use L1DomiconNodeMetaData.ABI instead.
var L1DomiconNodeABI = L1DomiconNodeMetaData.ABI

// L1DomiconNodeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1DomiconNodeMetaData.Bin instead.
var L1DomiconNodeBin = L1DomiconNodeMetaData.Bin

// DeployL1DomiconNode deploys a new Ethereum contract, binding an instance of L1DomiconNode to it.
func DeployL1DomiconNode(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1DomiconNode, error) {
	parsed, err := L1DomiconNodeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1DomiconNodeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1DomiconNode{L1DomiconNodeCaller: L1DomiconNodeCaller{contract: contract}, L1DomiconNodeTransactor: L1DomiconNodeTransactor{contract: contract}, L1DomiconNodeFilterer: L1DomiconNodeFilterer{contract: contract}}, nil
}

// L1DomiconNode is an auto generated Go binding around an Ethereum contract.
type L1DomiconNode struct {
	L1DomiconNodeCaller     // Read-only binding to the contract
	L1DomiconNodeTransactor // Write-only binding to the contract
	L1DomiconNodeFilterer   // Log filterer for contract events
}

// L1DomiconNodeCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1DomiconNodeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconNodeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1DomiconNodeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconNodeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1DomiconNodeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconNodeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1DomiconNodeSession struct {
	Contract     *L1DomiconNode    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1DomiconNodeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1DomiconNodeCallerSession struct {
	Contract *L1DomiconNodeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// L1DomiconNodeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1DomiconNodeTransactorSession struct {
	Contract     *L1DomiconNodeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// L1DomiconNodeRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1DomiconNodeRaw struct {
	Contract *L1DomiconNode // Generic contract binding to access the raw methods on
}

// L1DomiconNodeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1DomiconNodeCallerRaw struct {
	Contract *L1DomiconNodeCaller // Generic read-only contract binding to access the raw methods on
}

// L1DomiconNodeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1DomiconNodeTransactorRaw struct {
	Contract *L1DomiconNodeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1DomiconNode creates a new instance of L1DomiconNode, bound to a specific deployed contract.
func NewL1DomiconNode(address common.Address, backend bind.ContractBackend) (*L1DomiconNode, error) {
	contract, err := bindL1DomiconNode(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1DomiconNode{L1DomiconNodeCaller: L1DomiconNodeCaller{contract: contract}, L1DomiconNodeTransactor: L1DomiconNodeTransactor{contract: contract}, L1DomiconNodeFilterer: L1DomiconNodeFilterer{contract: contract}}, nil
}

// NewL1DomiconNodeCaller creates a new read-only instance of L1DomiconNode, bound to a specific deployed contract.
func NewL1DomiconNodeCaller(address common.Address, caller bind.ContractCaller) (*L1DomiconNodeCaller, error) {
	contract, err := bindL1DomiconNode(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeCaller{contract: contract}, nil
}

// NewL1DomiconNodeTransactor creates a new write-only instance of L1DomiconNode, bound to a specific deployed contract.
func NewL1DomiconNodeTransactor(address common.Address, transactor bind.ContractTransactor) (*L1DomiconNodeTransactor, error) {
	contract, err := bindL1DomiconNode(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeTransactor{contract: contract}, nil
}

// NewL1DomiconNodeFilterer creates a new log filterer instance of L1DomiconNode, bound to a specific deployed contract.
func NewL1DomiconNodeFilterer(address common.Address, filterer bind.ContractFilterer) (*L1DomiconNodeFilterer, error) {
	contract, err := bindL1DomiconNode(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeFilterer{contract: contract}, nil
}

// bindL1DomiconNode binds a generic wrapper to an already deployed contract.
func bindL1DomiconNode(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1DomiconNodeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconNode *L1DomiconNodeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconNode.Contract.L1DomiconNodeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconNode *L1DomiconNodeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.L1DomiconNodeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconNode *L1DomiconNodeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.L1DomiconNodeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconNode *L1DomiconNodeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconNode.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconNode *L1DomiconNodeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconNode *L1DomiconNodeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.contract.Transact(opts, method, params...)
}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeCaller) BROADCASTNODES(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "BROADCAST_NODES")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeSession) BROADCASTNODES() ([]common.Address, error) {
	return _L1DomiconNode.Contract.BROADCASTNODES(&_L1DomiconNode.CallOpts)
}

// BROADCASTNODES is a free data retrieval call binding the contract method 0x6a4fe4a8.
//
// Solidity: function BROADCAST_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeCallerSession) BROADCASTNODES() ([]common.Address, error) {
	return _L1DomiconNode.Contract.BROADCASTNODES(&_L1DomiconNode.CallOpts)
}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L1DomiconNode *L1DomiconNodeCaller) IsNodeBroadcast(opts *bind.CallOpts, addr common.Address) (bool, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "IsNodeBroadcast", addr)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L1DomiconNode *L1DomiconNodeSession) IsNodeBroadcast(addr common.Address) (bool, error) {
	return _L1DomiconNode.Contract.IsNodeBroadcast(&_L1DomiconNode.CallOpts, addr)
}

// IsNodeBroadcast is a free data retrieval call binding the contract method 0xc0f2acea.
//
// Solidity: function IsNodeBroadcast(address addr) view returns(bool)
func (_L1DomiconNode *L1DomiconNodeCallerSession) IsNodeBroadcast(addr common.Address) (bool, error) {
	return _L1DomiconNode.Contract.IsNodeBroadcast(&_L1DomiconNode.CallOpts, addr)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) MESSENGER() (common.Address, error) {
	return _L1DomiconNode.Contract.MESSENGER(&_L1DomiconNode.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) MESSENGER() (common.Address, error) {
	return _L1DomiconNode.Contract.MESSENGER(&_L1DomiconNode.CallOpts)
}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) OTHERDOMICONNODE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "OTHER_DOMICON_NODE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) OTHERDOMICONNODE() (common.Address, error) {
	return _L1DomiconNode.Contract.OTHERDOMICONNODE(&_L1DomiconNode.CallOpts)
}

// OTHERDOMICONNODE is a free data retrieval call binding the contract method 0x9b667993.
//
// Solidity: function OTHER_DOMICON_NODE() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) OTHERDOMICONNODE() (common.Address, error) {
	return _L1DomiconNode.Contract.OTHERDOMICONNODE(&_L1DomiconNode.CallOpts)
}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeCaller) STORAGENODES(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "STORAGE_NODES")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeSession) STORAGENODES() ([]common.Address, error) {
	return _L1DomiconNode.Contract.STORAGENODES(&_L1DomiconNode.CallOpts)
}

// STORAGENODES is a free data retrieval call binding the contract method 0xdb2b9423.
//
// Solidity: function STORAGE_NODES() view returns(address[])
func (_L1DomiconNode *L1DomiconNodeCallerSession) STORAGENODES() ([]common.Address, error) {
	return _L1DomiconNode.Contract.STORAGENODES(&_L1DomiconNode.CallOpts)
}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) BroadcastNodeList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "broadcastNodeList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) BroadcastNodeList(arg0 *big.Int) (common.Address, error) {
	return _L1DomiconNode.Contract.BroadcastNodeList(&_L1DomiconNode.CallOpts, arg0)
}

// BroadcastNodeList is a free data retrieval call binding the contract method 0x56f7be58.
//
// Solidity: function broadcastNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) BroadcastNodeList(arg0 *big.Int) (common.Address, error) {
	return _L1DomiconNode.Contract.BroadcastNodeList(&_L1DomiconNode.CallOpts, arg0)
}

// BroadcastingNodes is a free data retrieval call binding the contract method 0xcac5800a.
//
// Solidity: function broadcastingNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L1DomiconNode *L1DomiconNodeCaller) BroadcastingNodes(opts *bind.CallOpts, arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "broadcastingNodes", arg0)

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
func (_L1DomiconNode *L1DomiconNodeSession) BroadcastingNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L1DomiconNode.Contract.BroadcastingNodes(&_L1DomiconNode.CallOpts, arg0)
}

// BroadcastingNodes is a free data retrieval call binding the contract method 0xcac5800a.
//
// Solidity: function broadcastingNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L1DomiconNode *L1DomiconNodeCallerSession) BroadcastingNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L1DomiconNode.Contract.BroadcastingNodes(&_L1DomiconNode.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) Messenger() (common.Address, error) {
	return _L1DomiconNode.Contract.Messenger(&_L1DomiconNode.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) Messenger() (common.Address, error) {
	return _L1DomiconNode.Contract.Messenger(&_L1DomiconNode.CallOpts)
}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) OtherDomiconNode(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "otherDomiconNode")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) OtherDomiconNode() (common.Address, error) {
	return _L1DomiconNode.Contract.OtherDomiconNode(&_L1DomiconNode.CallOpts)
}

// OtherDomiconNode is a free data retrieval call binding the contract method 0xe261fba3.
//
// Solidity: function otherDomiconNode() view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) OtherDomiconNode() (common.Address, error) {
	return _L1DomiconNode.Contract.OtherDomiconNode(&_L1DomiconNode.CallOpts)
}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeCaller) StorageNodeList(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "storageNodeList", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeSession) StorageNodeList(arg0 *big.Int) (common.Address, error) {
	return _L1DomiconNode.Contract.StorageNodeList(&_L1DomiconNode.CallOpts, arg0)
}

// StorageNodeList is a free data retrieval call binding the contract method 0x7667d104.
//
// Solidity: function storageNodeList(uint256 ) view returns(address)
func (_L1DomiconNode *L1DomiconNodeCallerSession) StorageNodeList(arg0 *big.Int) (common.Address, error) {
	return _L1DomiconNode.Contract.StorageNodeList(&_L1DomiconNode.CallOpts, arg0)
}

// StorageNodes is a free data retrieval call binding the contract method 0x8118eb33.
//
// Solidity: function storageNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L1DomiconNode *L1DomiconNodeCaller) StorageNodes(opts *bind.CallOpts, arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "storageNodes", arg0)

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
func (_L1DomiconNode *L1DomiconNodeSession) StorageNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L1DomiconNode.Contract.StorageNodes(&_L1DomiconNode.CallOpts, arg0)
}

// StorageNodes is a free data retrieval call binding the contract method 0x8118eb33.
//
// Solidity: function storageNodes(address ) view returns(address add, string rpc, string name, uint256 stakedTokens, uint256 index)
func (_L1DomiconNode *L1DomiconNodeCallerSession) StorageNodes(arg0 common.Address) (struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Index        *big.Int
}, error) {
	return _L1DomiconNode.Contract.StorageNodes(&_L1DomiconNode.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconNode *L1DomiconNodeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1DomiconNode.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconNode *L1DomiconNodeSession) Version() (string, error) {
	return _L1DomiconNode.Contract.Version(&_L1DomiconNode.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconNode *L1DomiconNodeCallerSession) Version() (string, error) {
	return _L1DomiconNode.Contract.Version(&_L1DomiconNode.CallOpts)
}

// RegisterBroadcastNode is a paid mutator transaction binding the contract method 0x1d4c517f.
//
// Solidity: function RegisterBroadcastNode(address _address, string _rpc, string _name, uint256 _stakedTokens) returns()
func (_L1DomiconNode *L1DomiconNodeTransactor) RegisterBroadcastNode(opts *bind.TransactOpts, _address common.Address, _rpc string, _name string, _stakedTokens *big.Int) (*types.Transaction, error) {
	return _L1DomiconNode.contract.Transact(opts, "RegisterBroadcastNode", _address, _rpc, _name, _stakedTokens)
}

// RegisterBroadcastNode is a paid mutator transaction binding the contract method 0x1d4c517f.
//
// Solidity: function RegisterBroadcastNode(address _address, string _rpc, string _name, uint256 _stakedTokens) returns()
func (_L1DomiconNode *L1DomiconNodeSession) RegisterBroadcastNode(_address common.Address, _rpc string, _name string, _stakedTokens *big.Int) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.RegisterBroadcastNode(&_L1DomiconNode.TransactOpts, _address, _rpc, _name, _stakedTokens)
}

// RegisterBroadcastNode is a paid mutator transaction binding the contract method 0x1d4c517f.
//
// Solidity: function RegisterBroadcastNode(address _address, string _rpc, string _name, uint256 _stakedTokens) returns()
func (_L1DomiconNode *L1DomiconNodeTransactorSession) RegisterBroadcastNode(_address common.Address, _rpc string, _name string, _stakedTokens *big.Int) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.RegisterBroadcastNode(&_L1DomiconNode.TransactOpts, _address, _rpc, _name, _stakedTokens)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L1DomiconNode *L1DomiconNodeTransactor) FinalizeBroadcastNode(opts *bind.TransactOpts, node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L1DomiconNode.contract.Transact(opts, "finalizeBroadcastNode", node, nodeInfo)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L1DomiconNode *L1DomiconNodeSession) FinalizeBroadcastNode(node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.FinalizeBroadcastNode(&_L1DomiconNode.TransactOpts, node, nodeInfo)
}

// FinalizeBroadcastNode is a paid mutator transaction binding the contract method 0x7cce10a4.
//
// Solidity: function finalizeBroadcastNode(address node, (address,string,string,uint256,uint256) nodeInfo) payable returns()
func (_L1DomiconNode *L1DomiconNodeTransactorSession) FinalizeBroadcastNode(node common.Address, nodeInfo DomiconNodeNodeInfo) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.FinalizeBroadcastNode(&_L1DomiconNode.TransactOpts, node, nodeInfo)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _messenger) returns()
func (_L1DomiconNode *L1DomiconNodeTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address) (*types.Transaction, error) {
	return _L1DomiconNode.contract.Transact(opts, "initialize", _messenger)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _messenger) returns()
func (_L1DomiconNode *L1DomiconNodeSession) Initialize(_messenger common.Address) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.Initialize(&_L1DomiconNode.TransactOpts, _messenger)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _messenger) returns()
func (_L1DomiconNode *L1DomiconNodeTransactorSession) Initialize(_messenger common.Address) (*types.Transaction, error) {
	return _L1DomiconNode.Contract.Initialize(&_L1DomiconNode.TransactOpts, _messenger)
}

// L1DomiconNodeBroadcastNodeIterator is returned from FilterBroadcastNode and is used to iterate over the raw logs and unpacked data for BroadcastNode events raised by the L1DomiconNode contract.
type L1DomiconNodeBroadcastNodeIterator struct {
	Event *L1DomiconNodeBroadcastNode // Event containing the contract specifics and raw log

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
func (it *L1DomiconNodeBroadcastNodeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconNodeBroadcastNode)
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
		it.Event = new(L1DomiconNodeBroadcastNode)
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
func (it *L1DomiconNodeBroadcastNodeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconNodeBroadcastNodeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconNodeBroadcastNode represents a BroadcastNode event raised by the L1DomiconNode contract.
type L1DomiconNodeBroadcastNode struct {
	Add          common.Address
	Rpc          string
	Name         string
	StakedTokens *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBroadcastNode is a free log retrieval operation binding the contract event 0xf81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf82616815.
//
// Solidity: event BroadcastNode(address indexed add, string rpc, string name, uint256 stakedTokens)
func (_L1DomiconNode *L1DomiconNodeFilterer) FilterBroadcastNode(opts *bind.FilterOpts, add []common.Address) (*L1DomiconNodeBroadcastNodeIterator, error) {

	var addRule []interface{}
	for _, addItem := range add {
		addRule = append(addRule, addItem)
	}

	logs, sub, err := _L1DomiconNode.contract.FilterLogs(opts, "BroadcastNode", addRule)
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeBroadcastNodeIterator{contract: _L1DomiconNode.contract, event: "BroadcastNode", logs: logs, sub: sub}, nil
}

// WatchBroadcastNode is a free log subscription operation binding the contract event 0xf81ce16a7ccf3a5a010dfa9ea629627f1144fc81731e9d33059eb7bf82616815.
//
// Solidity: event BroadcastNode(address indexed add, string rpc, string name, uint256 stakedTokens)
func (_L1DomiconNode *L1DomiconNodeFilterer) WatchBroadcastNode(opts *bind.WatchOpts, sink chan<- *L1DomiconNodeBroadcastNode, add []common.Address) (event.Subscription, error) {

	var addRule []interface{}
	for _, addItem := range add {
		addRule = append(addRule, addItem)
	}

	logs, sub, err := _L1DomiconNode.contract.WatchLogs(opts, "BroadcastNode", addRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconNodeBroadcastNode)
				if err := _L1DomiconNode.contract.UnpackLog(event, "BroadcastNode", log); err != nil {
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
func (_L1DomiconNode *L1DomiconNodeFilterer) ParseBroadcastNode(log types.Log) (*L1DomiconNodeBroadcastNode, error) {
	event := new(L1DomiconNodeBroadcastNode)
	if err := _L1DomiconNode.contract.UnpackLog(event, "BroadcastNode", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconNodeFinalizeBroadcastNodeIterator is returned from FilterFinalizeBroadcastNode and is used to iterate over the raw logs and unpacked data for FinalizeBroadcastNode events raised by the L1DomiconNode contract.
type L1DomiconNodeFinalizeBroadcastNodeIterator struct {
	Event *L1DomiconNodeFinalizeBroadcastNode // Event containing the contract specifics and raw log

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
func (it *L1DomiconNodeFinalizeBroadcastNodeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconNodeFinalizeBroadcastNode)
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
		it.Event = new(L1DomiconNodeFinalizeBroadcastNode)
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
func (it *L1DomiconNodeFinalizeBroadcastNodeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconNodeFinalizeBroadcastNodeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconNodeFinalizeBroadcastNode represents a FinalizeBroadcastNode event raised by the L1DomiconNode contract.
type L1DomiconNodeFinalizeBroadcastNode struct {
	NodeInfo DomiconNodeNodeInfo
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFinalizeBroadcastNode is a free log retrieval operation binding the contract event 0x59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71.
//
// Solidity: event FinalizeBroadcastNode((address,string,string,uint256,uint256) nodeInfo)
func (_L1DomiconNode *L1DomiconNodeFilterer) FilterFinalizeBroadcastNode(opts *bind.FilterOpts) (*L1DomiconNodeFinalizeBroadcastNodeIterator, error) {

	logs, sub, err := _L1DomiconNode.contract.FilterLogs(opts, "FinalizeBroadcastNode")
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeFinalizeBroadcastNodeIterator{contract: _L1DomiconNode.contract, event: "FinalizeBroadcastNode", logs: logs, sub: sub}, nil
}

// WatchFinalizeBroadcastNode is a free log subscription operation binding the contract event 0x59a5ba12d05c423999d0a159ea082d9a302e5372e8d770bccdcd3af869a89b71.
//
// Solidity: event FinalizeBroadcastNode((address,string,string,uint256,uint256) nodeInfo)
func (_L1DomiconNode *L1DomiconNodeFilterer) WatchFinalizeBroadcastNode(opts *bind.WatchOpts, sink chan<- *L1DomiconNodeFinalizeBroadcastNode) (event.Subscription, error) {

	logs, sub, err := _L1DomiconNode.contract.WatchLogs(opts, "FinalizeBroadcastNode")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconNodeFinalizeBroadcastNode)
				if err := _L1DomiconNode.contract.UnpackLog(event, "FinalizeBroadcastNode", log); err != nil {
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
func (_L1DomiconNode *L1DomiconNodeFilterer) ParseFinalizeBroadcastNode(log types.Log) (*L1DomiconNodeFinalizeBroadcastNode, error) {
	event := new(L1DomiconNodeFinalizeBroadcastNode)
	if err := _L1DomiconNode.contract.UnpackLog(event, "FinalizeBroadcastNode", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconNodeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1DomiconNode contract.
type L1DomiconNodeInitializedIterator struct {
	Event *L1DomiconNodeInitialized // Event containing the contract specifics and raw log

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
func (it *L1DomiconNodeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconNodeInitialized)
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
		it.Event = new(L1DomiconNodeInitialized)
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
func (it *L1DomiconNodeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconNodeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconNodeInitialized represents a Initialized event raised by the L1DomiconNode contract.
type L1DomiconNodeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconNode *L1DomiconNodeFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1DomiconNodeInitializedIterator, error) {

	logs, sub, err := _L1DomiconNode.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1DomiconNodeInitializedIterator{contract: _L1DomiconNode.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconNode *L1DomiconNodeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1DomiconNodeInitialized) (event.Subscription, error) {

	logs, sub, err := _L1DomiconNode.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconNodeInitialized)
				if err := _L1DomiconNode.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L1DomiconNode *L1DomiconNodeFilterer) ParseInitialized(log types.Log) (*L1DomiconNodeInitialized, error) {
	event := new(L1DomiconNodeInitialized)
	if err := _L1DomiconNode.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
