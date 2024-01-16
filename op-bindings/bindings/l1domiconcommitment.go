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

// L1DomiconCommitmentMetaData contains all meta data concerning the L1DomiconCommitment contract.
var L1DomiconCommitmentMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"FinalizeSubmitCommitment\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"SendDACommitment\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_COMMITMENT\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"SubmitCommitment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_broadcaster\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"finalizeSubmitCommitment\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"indices\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"contractDomiconNode\",\"name\":\"_node\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherCommitment\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"submits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a06040523480156200001157600080fd5b50734200000000000000000000000000000000000022608052620000376000806200003d565b620001ca565b600054600390610100900460ff1615801562000060575060005460ff8083169116105b620000c95760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620000e983836200012f565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b600054610100900460ff166200019c5760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000c0565b600480546001600160a01b039384166001600160a01b03199182161790915560058054929093169116179055565b60805161173d620001fb600039600081816102ca01528181610300015281816104d90152610f0d015261173d6000f3fe6080604052600436106100c75760003560e01c8063777109f811610074578063e4a200c81161004e578063e4a200c81461029b578063e996e9ac146102bb578063fce1c974146102ee57600080fd5b8063777109f81461022a578063927ede2d1461023d578063dcf36d571461026857600080fd5b80635063e207116100a55780635063e2071461016c57806354fd4d50146101a75780635fa4ad36146101fd57600080fd5b80633817ce86146100cc5780633cb747bf1461011d578063485cc9551461014a575b600080fd5b3480156100d857600080fd5b5060055473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561012957600080fd5b506004546100f39073ffffffffffffffffffffffffffffffffffffffff1681565b34801561015657600080fd5b5061016a610165366004611063565b610322565b005b34801561017857600080fd5b5061019961018736600461109c565b60036020526000908152604090205481565b604051908152602001610114565b3480156101b357600080fd5b506101f06040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b604051610114919061112b565b34801561020957600080fd5b506005546100f39073ffffffffffffffffffffffffffffffffffffffff1681565b61016a610238366004611187565b610473565b34801561024957600080fd5b5060045473ffffffffffffffffffffffffffffffffffffffff166100f3565b34801561027457600080fd5b5061028861028336600461123b565b610892565b6040516101149796959493929190611267565b3480156102a757600080fd5b5061016a6102b63660046112d2565b610a03565b3480156102c757600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006100f3565b3480156102fa57600080fd5b506100f37f000000000000000000000000000000000000000000000000000000000000000081565b600054600390610100900460ff16158015610344575060005460ff8083169116105b6103d5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084015b60405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556104108383610dfc565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050565b60045473ffffffffffffffffffffffffffffffffffffffff1633148015610562575060048054604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000811694931692636e296e45928082019260209290918290030181865afa158015610526573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061054a9190611371565b73ffffffffffffffffffffffffffffffffffffffff16145b610614576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20746865206f7468657220636f60648201527f6d6d69746d656e74000000000000000000000000000000000000000000000000608482015260a4016103cc565b6040518060e001604052808a81526020018981526020018881526020018673ffffffffffffffffffffffffffffffffffffffff1681526020013373ffffffffffffffffffffffffffffffffffffffff16815260200185858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f860181900481028201810190925284815291810191908590859081908401838280828437600081840152601f19601f82011690508083019250505050505050815250600260008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008b815260200190815260200160002060008201518160000155602082015181600101556040820151816002015560608201518160030160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060808201518160040160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060a08201518160050190816107fd919061145f565b5060c08201516006820190610812908261145f565b509050508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff167f9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea8b8b8b8989898960405161087f97969594939291906115c2565b60405180910390a3505050505050505050565b60026020818152600093845260408085209091529183529120805460018201549282015460038301546004840154600585018054949695939473ffffffffffffffffffffffffffffffffffffffff9384169492909316926108f2906113bd565b80601f016020809104026020016040519081016040528092919081815260200182805461091e906113bd565b801561096b5780601f106109405761010080835404028352916020019161096b565b820191906000526020600020905b81548152906001019060200180831161094e57829003601f168201915b505050505090806006018054610980906113bd565b80601f01602080910402602001604051908101604052809291908181526020018280546109ac906113bd565b80156109f95780601f106109ce576101008083540402835291602001916109f9565b820191906000526020600020905b8154815290600101906020018083116109dc57829003601f168201915b5050505050905087565b333b15610a92576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20616e20454f4100000000000060648201526084016103cc565b6005546040517fc0f2acea00000000000000000000000000000000000000000000000000000000815233600482015273ffffffffffffffffffffffffffffffffffffffff9091169063c0f2acea906024016020604051808303816000875af1158015610b02573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b2691906115fb565b610bb2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f446f6d69636f6e436f6d6d69746d656e743a2062726f616463617374206e6f6460448201527f652061646472657373206572726f72000000000000000000000000000000000060648201526084016103cc565b6040518060e001604052808981526020018881526020018781526020018673ffffffffffffffffffffffffffffffffffffffff1681526020013373ffffffffffffffffffffffffffffffffffffffff16815260200185858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f860181900481028201810190925284815291810191908590859081908401838280828437600092018290525093909452505073ffffffffffffffffffffffffffffffffffffffff80891682526002602081815260408085208f865282529384902086518155908601516001820155928501519083015560608401516003830180549183167fffffffffffffffffffffffff0000000000000000000000000000000000000000928316179055608085015160048401805491909316911617905560a08301519091506005820190610d1e908261145f565b5060c08201516006820190610d33908261145f565b50505073ffffffffffffffffffffffffffffffffffffffff85166000908152600360205260408120805491610d678361161d565b91905055508473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e8a8a8a89898989604051610dd597969594939291906115c2565b60405180910390a3610df262030d40898989338a8a8a8a8a610ee6565b5050505050505050565b600054610100900460ff16610e93576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016103cc565b6004805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560058054929093169116179055565b60045460405173ffffffffffffffffffffffffffffffffffffffff9091169063b8920c14907f0000000000000000000000000000000000000000000000000000000000000000907f777109f80000000000000000000000000000000000000000000000000000000090610f6d908e908e908e908e908e908e908e908e908e9060240161167c565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b909216825261100092918f906004016116eb565b600060405180830381600087803b15801561101a57600080fd5b505af115801561102e573d6000803e3d6000fd5b5050505050505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461106057600080fd5b50565b6000806040838503121561107657600080fd5b82356110818161103e565b915060208301356110918161103e565b809150509250929050565b6000602082840312156110ae57600080fd5b81356110b98161103e565b9392505050565b6000815180845260005b818110156110e6576020818501810151868301820152016110ca565b818111156110f8576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006110b960208301846110c0565b60008083601f84011261115057600080fd5b50813567ffffffffffffffff81111561116857600080fd5b60208301915083602082850101111561118057600080fd5b9250929050565b600080600080600080600080600060e08a8c0312156111a557600080fd5b8935985060208a0135975060408a0135965060608a01356111c58161103e565b955060808a01356111d58161103e565b945060a08a013567ffffffffffffffff808211156111f257600080fd5b6111fe8d838e0161113e565b909650945060c08c013591508082111561121757600080fd5b506112248c828d0161113e565b915080935050809150509295985092959850929598565b6000806040838503121561124e57600080fd5b82356112598161103e565b946020939093013593505050565b878152866020820152856040820152600073ffffffffffffffffffffffffffffffffffffffff808716606084015280861660808401525060e060a08301526112b260e08301856110c0565b82810360c08401526112c481856110c0565b9a9950505050505050505050565b60008060008060008060008060c0898b0312156112ee57600080fd5b883597506020890135965060408901359550606089013561130e8161103e565b9450608089013567ffffffffffffffff8082111561132b57600080fd5b6113378c838d0161113e565b909650945060a08b013591508082111561135057600080fd5b5061135d8b828c0161113e565b999c989b5096995094979396929594505050565b60006020828403121561138357600080fd5b81516110b98161103e565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c908216806113d157607f821691505b60208210810361140a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561145a57600081815260208120601f850160051c810160208610156114375750805b601f850160051c820191505b8181101561145657828155600101611443565b5050505b505050565b815167ffffffffffffffff8111156114795761147961138e565b61148d8161148784546113bd565b84611410565b602080601f8311600181146114e057600084156114aa5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555611456565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b8281101561152d5788860151825594840194600190910190840161150e565b508582101561156957878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b87815286602082015285604082015260a0606082015260006115e860a083018688611579565b82810360808401526112c4818587611579565b60006020828403121561160d57600080fd5b815180151581146110b957600080fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203611675577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b5060010190565b898152886020820152876040820152600073ffffffffffffffffffffffffffffffffffffffff808916606084015280881660808401525060e060a08301526116c860e083018688611579565b82810360c08401526116db818587611579565b9c9b505050505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff8416815260606020820152600061171a60608301856110c0565b905063ffffffff8316604083015294935050505056fea164736f6c634300080f000a",
}

// L1DomiconCommitmentABI is the input ABI used to generate the binding from.
// Deprecated: Use L1DomiconCommitmentMetaData.ABI instead.
var L1DomiconCommitmentABI = L1DomiconCommitmentMetaData.ABI

// L1DomiconCommitmentBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1DomiconCommitmentMetaData.Bin instead.
var L1DomiconCommitmentBin = L1DomiconCommitmentMetaData.Bin

// DeployL1DomiconCommitment deploys a new Ethereum contract, binding an instance of L1DomiconCommitment to it.
func DeployL1DomiconCommitment(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1DomiconCommitment, error) {
	parsed, err := L1DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1DomiconCommitmentBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1DomiconCommitment{L1DomiconCommitmentCaller: L1DomiconCommitmentCaller{contract: contract}, L1DomiconCommitmentTransactor: L1DomiconCommitmentTransactor{contract: contract}, L1DomiconCommitmentFilterer: L1DomiconCommitmentFilterer{contract: contract}}, nil
}

// L1DomiconCommitment is an auto generated Go binding around an Ethereum contract.
type L1DomiconCommitment struct {
	L1DomiconCommitmentCaller     // Read-only binding to the contract
	L1DomiconCommitmentTransactor // Write-only binding to the contract
	L1DomiconCommitmentFilterer   // Log filterer for contract events
}

// L1DomiconCommitmentCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1DomiconCommitmentCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1DomiconCommitmentTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1DomiconCommitmentFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1DomiconCommitmentSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1DomiconCommitmentSession struct {
	Contract     *L1DomiconCommitment // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// L1DomiconCommitmentCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1DomiconCommitmentCallerSession struct {
	Contract *L1DomiconCommitmentCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// L1DomiconCommitmentTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1DomiconCommitmentTransactorSession struct {
	Contract     *L1DomiconCommitmentTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// L1DomiconCommitmentRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1DomiconCommitmentRaw struct {
	Contract *L1DomiconCommitment // Generic contract binding to access the raw methods on
}

// L1DomiconCommitmentCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1DomiconCommitmentCallerRaw struct {
	Contract *L1DomiconCommitmentCaller // Generic read-only contract binding to access the raw methods on
}

// L1DomiconCommitmentTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1DomiconCommitmentTransactorRaw struct {
	Contract *L1DomiconCommitmentTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1DomiconCommitment creates a new instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitment(address common.Address, backend bind.ContractBackend) (*L1DomiconCommitment, error) {
	contract, err := bindL1DomiconCommitment(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitment{L1DomiconCommitmentCaller: L1DomiconCommitmentCaller{contract: contract}, L1DomiconCommitmentTransactor: L1DomiconCommitmentTransactor{contract: contract}, L1DomiconCommitmentFilterer: L1DomiconCommitmentFilterer{contract: contract}}, nil
}

// NewL1DomiconCommitmentCaller creates a new read-only instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentCaller(address common.Address, caller bind.ContractCaller) (*L1DomiconCommitmentCaller, error) {
	contract, err := bindL1DomiconCommitment(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentCaller{contract: contract}, nil
}

// NewL1DomiconCommitmentTransactor creates a new write-only instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentTransactor(address common.Address, transactor bind.ContractTransactor) (*L1DomiconCommitmentTransactor, error) {
	contract, err := bindL1DomiconCommitment(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentTransactor{contract: contract}, nil
}

// NewL1DomiconCommitmentFilterer creates a new log filterer instance of L1DomiconCommitment, bound to a specific deployed contract.
func NewL1DomiconCommitmentFilterer(address common.Address, filterer bind.ContractFilterer) (*L1DomiconCommitmentFilterer, error) {
	contract, err := bindL1DomiconCommitment(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentFilterer{contract: contract}, nil
}

// bindL1DomiconCommitment binds a generic wrapper to an already deployed contract.
func bindL1DomiconCommitment(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1DomiconCommitmentMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconCommitment *L1DomiconCommitmentRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.L1DomiconCommitmentTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1DomiconCommitment *L1DomiconCommitmentCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1DomiconCommitment.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.contract.Transact(opts, method, params...)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) DOMICONNODE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "DOMICON_NODE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) DOMICONNODE() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOMICONNODE(&_L1DomiconCommitment.CallOpts)
}

// DOMICONNODE is a free data retrieval call binding the contract method 0x3817ce86.
//
// Solidity: function DOMICON_NODE() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) DOMICONNODE() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DOMICONNODE(&_L1DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) MESSENGER() (common.Address, error) {
	return _L1DomiconCommitment.Contract.MESSENGER(&_L1DomiconCommitment.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) MESSENGER() (common.Address, error) {
	return _L1DomiconCommitment.Contract.MESSENGER(&_L1DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) OTHERCOMMITMENT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "OTHER_COMMITMENT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OTHERCOMMITMENT(&_L1DomiconCommitment.CallOpts)
}

// OTHERCOMMITMENT is a free data retrieval call binding the contract method 0xfce1c974.
//
// Solidity: function OTHER_COMMITMENT() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) OTHERCOMMITMENT() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OTHERCOMMITMENT(&_L1DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) DomiconNode(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "domiconNode")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) DomiconNode() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DomiconNode(&_L1DomiconCommitment.CallOpts)
}

// DomiconNode is a free data retrieval call binding the contract method 0x5fa4ad36.
//
// Solidity: function domiconNode() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) DomiconNode() (common.Address, error) {
	return _L1DomiconCommitment.Contract.DomiconNode(&_L1DomiconCommitment.CallOpts)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Indices(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "indices", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L1DomiconCommitment.Contract.Indices(&_L1DomiconCommitment.CallOpts, arg0)
}

// Indices is a free data retrieval call binding the contract method 0x5063e207.
//
// Solidity: function indices(address ) view returns(uint256)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Indices(arg0 common.Address) (*big.Int, error) {
	return _L1DomiconCommitment.Contract.Indices(&_L1DomiconCommitment.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Messenger() (common.Address, error) {
	return _L1DomiconCommitment.Contract.Messenger(&_L1DomiconCommitment.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Messenger() (common.Address, error) {
	return _L1DomiconCommitment.Contract.Messenger(&_L1DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) OtherCommitment(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "otherCommitment")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) OtherCommitment() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OtherCommitment(&_L1DomiconCommitment.CallOpts)
}

// OtherCommitment is a free data retrieval call binding the contract method 0xe996e9ac.
//
// Solidity: function otherCommitment() view returns(address)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) OtherCommitment() (common.Address, error) {
	return _L1DomiconCommitment.Contract.OtherCommitment(&_L1DomiconCommitment.CallOpts)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(uint256 index, uint256 length, uint256 price, address user, address broadcaster, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Submits(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "submits", arg0, arg1)

	outstruct := new(struct {
		Index       *big.Int
		Length      *big.Int
		Price       *big.Int
		User        common.Address
		Broadcaster common.Address
		Sign        []byte
		Commitment  []byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Index = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Length = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Price = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.User = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Broadcaster = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Sign = *abi.ConvertType(out[5], new([]byte)).(*[]byte)
	outstruct.Commitment = *abi.ConvertType(out[6], new([]byte)).(*[]byte)

	return *outstruct, err

}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(uint256 index, uint256 length, uint256 price, address user, address broadcaster, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Submits(arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
	return _L1DomiconCommitment.Contract.Submits(&_L1DomiconCommitment.CallOpts, arg0, arg1)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(uint256 index, uint256 length, uint256 price, address user, address broadcaster, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Submits(arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
	return _L1DomiconCommitment.Contract.Submits(&_L1DomiconCommitment.CallOpts, arg0, arg1)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L1DomiconCommitment.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Version() (string, error) {
	return _L1DomiconCommitment.Contract.Version(&_L1DomiconCommitment.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L1DomiconCommitment *L1DomiconCommitmentCallerSession) Version() (string, error) {
	return _L1DomiconCommitment.Contract.Version(&_L1DomiconCommitment.CallOpts)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) SubmitCommitment(opts *bind.TransactOpts, _index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "SubmitCommitment", _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) SubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) SubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.SubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) FinalizeSubmitCommitment(opts *bind.TransactOpts, _index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "finalizeSubmitCommitment", _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// FinalizeSubmitCommitment is a paid mutator transaction binding the contract method 0x777109f8.
//
// Solidity: function finalizeSubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _broadcaster, address _user, bytes _sign, bytes _commitment) payable returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) FinalizeSubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _broadcaster common.Address, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.FinalizeSubmitCommitment(&_L1DomiconCommitment.TransactOpts, _index, _length, _price, _broadcaster, _user, _sign, _commitment)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.contract.Transact(opts, "initialize", _messenger, _node)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentSession) Initialize(_messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.Initialize(&_L1DomiconCommitment.TransactOpts, _messenger, _node)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _messenger, address _node) returns()
func (_L1DomiconCommitment *L1DomiconCommitmentTransactorSession) Initialize(_messenger common.Address, _node common.Address) (*types.Transaction, error) {
	return _L1DomiconCommitment.Contract.Initialize(&_L1DomiconCommitment.TransactOpts, _messenger, _node)
}

// L1DomiconCommitmentFinalizeSubmitCommitmentIterator is returned from FilterFinalizeSubmitCommitment and is used to iterate over the raw logs and unpacked data for FinalizeSubmitCommitment events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentFinalizeSubmitCommitmentIterator struct {
	Event *L1DomiconCommitmentFinalizeSubmitCommitment // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentFinalizeSubmitCommitment)
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
		it.Event = new(L1DomiconCommitmentFinalizeSubmitCommitment)
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
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentFinalizeSubmitCommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentFinalizeSubmitCommitment represents a FinalizeSubmitCommitment event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentFinalizeSubmitCommitment struct {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterFinalizeSubmitCommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L1DomiconCommitmentFinalizeSubmitCommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentFinalizeSubmitCommitmentIterator{contract: _L1DomiconCommitment.contract, event: "FinalizeSubmitCommitment", logs: logs, sub: sub}, nil
}

// WatchFinalizeSubmitCommitment is a free log subscription operation binding the contract event 0x9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea.
//
// Solidity: event FinalizeSubmitCommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchFinalizeSubmitCommitment(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentFinalizeSubmitCommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "FinalizeSubmitCommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentFinalizeSubmitCommitment)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseFinalizeSubmitCommitment(log types.Log) (*L1DomiconCommitmentFinalizeSubmitCommitment, error) {
	event := new(L1DomiconCommitmentFinalizeSubmitCommitment)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "FinalizeSubmitCommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconCommitmentInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentInitializedIterator struct {
	Event *L1DomiconCommitmentInitialized // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentInitialized)
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
		it.Event = new(L1DomiconCommitmentInitialized)
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
func (it *L1DomiconCommitmentInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentInitialized represents a Initialized event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1DomiconCommitmentInitializedIterator, error) {

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentInitializedIterator{contract: _L1DomiconCommitment.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentInitialized) (event.Subscription, error) {

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentInitialized)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseInitialized(log types.Log) (*L1DomiconCommitmentInitialized, error) {
	event := new(L1DomiconCommitmentInitialized)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1DomiconCommitmentSendDACommitmentIterator is returned from FilterSendDACommitment and is used to iterate over the raw logs and unpacked data for SendDACommitment events raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentSendDACommitmentIterator struct {
	Event *L1DomiconCommitmentSendDACommitment // Event containing the contract specifics and raw log

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
func (it *L1DomiconCommitmentSendDACommitmentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1DomiconCommitmentSendDACommitment)
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
		it.Event = new(L1DomiconCommitmentSendDACommitment)
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
func (it *L1DomiconCommitmentSendDACommitmentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1DomiconCommitmentSendDACommitmentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1DomiconCommitmentSendDACommitment represents a SendDACommitment event raised by the L1DomiconCommitment contract.
type L1DomiconCommitmentSendDACommitment struct {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) FilterSendDACommitment(opts *bind.FilterOpts, broadcaster []common.Address, user []common.Address) (*L1DomiconCommitmentSendDACommitmentIterator, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.FilterLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return &L1DomiconCommitmentSendDACommitmentIterator{contract: _L1DomiconCommitment.contract, event: "SendDACommitment", logs: logs, sub: sub}, nil
}

// WatchSendDACommitment is a free log subscription operation binding the contract event 0xce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e.
//
// Solidity: event SendDACommitment(uint256 index, uint256 length, uint256 price, address indexed broadcaster, address indexed user, bytes sign, bytes commitment)
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) WatchSendDACommitment(opts *bind.WatchOpts, sink chan<- *L1DomiconCommitmentSendDACommitment, broadcaster []common.Address, user []common.Address) (event.Subscription, error) {

	var broadcasterRule []interface{}
	for _, broadcasterItem := range broadcaster {
		broadcasterRule = append(broadcasterRule, broadcasterItem)
	}
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _L1DomiconCommitment.contract.WatchLogs(opts, "SendDACommitment", broadcasterRule, userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1DomiconCommitmentSendDACommitment)
				if err := _L1DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
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
func (_L1DomiconCommitment *L1DomiconCommitmentFilterer) ParseSendDACommitment(log types.Log) (*L1DomiconCommitmentSendDACommitment, error) {
	event := new(L1DomiconCommitmentSendDACommitment)
	if err := _L1DomiconCommitment.contract.UnpackLog(event, "SendDACommitment", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
