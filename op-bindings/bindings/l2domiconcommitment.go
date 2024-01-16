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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"_otherCommitment\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"FinalizeSubmitCommitment\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"name\":\"SendDACommitment\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMICON_NODE\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OTHER_COMMITMENT\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"SubmitCommitment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domiconNode\",\"outputs\":[{\"internalType\":\"contractDomiconNode\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_broadcaster\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_user\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_commitment\",\"type\":\"bytes\"}],\"name\":\"finalizeSubmitCommitment\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"indices\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractCrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"otherCommitment\",\"outputs\":[{\"internalType\":\"contractDomiconCommitment\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"submits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"broadcaster\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"sign\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a06040523480156200001157600080fd5b506040516200184538038062001845833981016040819052620000349162000205565b6001600160a01b0381166080526200004b62000052565b5062000237565b600054600390610100900460ff1615801562000075575060005460ff8083169116105b620000de5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805461ffff191660ff831617610100179055620001267342000000000000000000000000000000000000077342000000000000000000000000000000000000236200016a565b6000805461ff001916905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b600054610100900460ff16620001d75760405162461bcd60e51b815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201526a6e697469616c697a696e6760a81b6064820152608401620000d5565b600480546001600160a01b039384166001600160a01b03199182161790915560058054929093169116179055565b6000602082840312156200021857600080fd5b81516001600160a01b03811681146200023057600080fd5b9392505050565b6080516115dd62000268600039600081816102bf015281816102f50152818161037d0152610e0801526115dd6000f3fe6080604052600436106100c75760003560e01c80638129fc1c11610074578063e4a200c81161004e578063e4a200c814610290578063e996e9ac146102b0578063fce1c974146102e357600080fd5b80638129fc1c1461021d578063927ede2d14610232578063dcf36d571461025d57600080fd5b806354fd4d50116100a557806354fd4d50146101855780635fa4ad36146101db578063777109f81461020857600080fd5b80633817ce86146100cc5780633cb747bf1461011d5780635063e2071461014a575b600080fd5b3480156100d857600080fd5b5060055473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561012957600080fd5b506004546100f39073ffffffffffffffffffffffffffffffffffffffff1681565b34801561015657600080fd5b50610177610165366004610f5e565b60036020526000908152604090205481565b604051908152602001610114565b34801561019157600080fd5b506101ce6040518060400160405280600581526020017f312e342e3100000000000000000000000000000000000000000000000000000081525081565b6040516101149190610fed565b3480156101e757600080fd5b506005546100f39073ffffffffffffffffffffffffffffffffffffffff1681565b61021b610216366004611049565b610317565b005b34801561022957600080fd5b5061021b61073b565b34801561023e57600080fd5b5060045473ffffffffffffffffffffffffffffffffffffffff166100f3565b34801561026957600080fd5b5061027d6102783660046110fd565b6108ad565b6040516101149796959493929190611129565b34801561029c57600080fd5b5061021b6102ab366004611194565b610a1e565b3480156102bc57600080fd5b507f00000000000000000000000000000000000000000000000000000000000000006100f3565b3480156102ef57600080fd5b506100f37f000000000000000000000000000000000000000000000000000000000000000081565b60045473ffffffffffffffffffffffffffffffffffffffff1633148015610406575060048054604080517f6e296e45000000000000000000000000000000000000000000000000000000008152905173ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000811694931692636e296e45928082019260209290918290030181865afa1580156103ca573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103ee9190611233565b73ffffffffffffffffffffffffffffffffffffffff16145b6104bd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20746865206f7468657220636f60648201527f6d6d69746d656e74000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b6040518060e001604052808a81526020018981526020018881526020018673ffffffffffffffffffffffffffffffffffffffff1681526020013373ffffffffffffffffffffffffffffffffffffffff16815260200185858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f860181900481028201810190925284815291810191908590859081908401838280828437600081840152601f19601f82011690508083019250505050505050815250600260008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008b815260200190815260200160002060008201518160000155602082015181600101556040820151816002015560608201518160030160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060808201518160040160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060a08201518160050190816106a69190611321565b5060c082015160068201906106bb9082611321565b509050508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff167f9abb68e4de67438897a668216c43446bb0f2cf6d2cb96c207701ff4fa54f3bea8b8b8b898989896040516107289796959493929190611484565b60405180910390a3505050505050505050565b600054600390610100900460ff1615801561075d575060005460ff8083169116105b6107e9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016104b4565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff83161761010017905561084c734200000000000000000000000000000000000007734200000000000000000000000000000000000023610cf7565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a150565b60026020818152600093845260408085209091529183529120805460018201549282015460038301546004840154600585018054949695939473ffffffffffffffffffffffffffffffffffffffff93841694929093169261090d9061127f565b80601f01602080910402602001604051908101604052809291908181526020018280546109399061127f565b80156109865780601f1061095b57610100808354040283529160200191610986565b820191906000526020600020905b81548152906001019060200180831161096957829003601f168201915b50505050509080600601805461099b9061127f565b80601f01602080910402602001604051908101604052809291908181526020018280546109c79061127f565b8015610a145780601f106109e957610100808354040283529160200191610a14565b820191906000526020600020905b8154815290600101906020018083116109f757829003601f168201915b5050505050905087565b333b15610aad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f446f6d69636f6e436f6d6d69746d656e743a2066756e6374696f6e2063616e2060448201527f6f6e6c792062652063616c6c65642066726f6d20616e20454f4100000000000060648201526084016104b4565b6040518060e001604052808981526020018881526020018781526020018673ffffffffffffffffffffffffffffffffffffffff1681526020013373ffffffffffffffffffffffffffffffffffffffff16815260200185858080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250505090825250604080516020601f860181900481028201810190925284815291810191908590859081908401838280828437600092018290525093909452505073ffffffffffffffffffffffffffffffffffffffff80891682526002602081815260408085208f865282529384902086518155908601516001820155928501519083015560608401516003830180549183167fffffffffffffffffffffffff0000000000000000000000000000000000000000928316179055608085015160048401805491909316911617905560a08301519091506005820190610c199082611321565b5060c08201516006820190610c2e9082611321565b50505073ffffffffffffffffffffffffffffffffffffffff85166000908152600360205260408120805491610c62836114bd565b91905055508473ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fce7c513598cb8cde5f5798356c301e6ed9a2889048d0cb2e36504b5f9c85d90e8a8a8a89898989604051610cd09796959493929190611484565b60405180910390a3610ced62030d40898989338a8a8a8a8a610de1565b5050505050505050565b600054610100900460ff16610d8e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016104b4565b6004805473ffffffffffffffffffffffffffffffffffffffff9384167fffffffffffffffffffffffff00000000000000000000000000000000000000009182161790915560058054929093169116179055565b60045460405173ffffffffffffffffffffffffffffffffffffffff9091169063b8920c14907f0000000000000000000000000000000000000000000000000000000000000000907f777109f80000000000000000000000000000000000000000000000000000000090610e68908e908e908e908e908e908e908e908e908e9060240161151c565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e085901b9092168252610efb92918f9060040161158b565b600060405180830381600087803b158015610f1557600080fd5b505af1158015610f29573d6000803e3d6000fd5b5050505050505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610f5b57600080fd5b50565b600060208284031215610f7057600080fd5b8135610f7b81610f39565b9392505050565b6000815180845260005b81811015610fa857602081850181015186830182015201610f8c565b81811115610fba576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610f7b6020830184610f82565b60008083601f84011261101257600080fd5b50813567ffffffffffffffff81111561102a57600080fd5b60208301915083602082850101111561104257600080fd5b9250929050565b600080600080600080600080600060e08a8c03121561106757600080fd5b8935985060208a0135975060408a0135965060608a013561108781610f39565b955060808a013561109781610f39565b945060a08a013567ffffffffffffffff808211156110b457600080fd5b6110c08d838e01611000565b909650945060c08c01359150808211156110d957600080fd5b506110e68c828d01611000565b915080935050809150509295985092959850929598565b6000806040838503121561111057600080fd5b823561111b81610f39565b946020939093013593505050565b878152866020820152856040820152600073ffffffffffffffffffffffffffffffffffffffff808716606084015280861660808401525060e060a083015261117460e0830185610f82565b82810360c08401526111868185610f82565b9a9950505050505050505050565b60008060008060008060008060c0898b0312156111b057600080fd5b88359750602089013596506040890135955060608901356111d081610f39565b9450608089013567ffffffffffffffff808211156111ed57600080fd5b6111f98c838d01611000565b909650945060a08b013591508082111561121257600080fd5b5061121f8b828c01611000565b999c989b5096995094979396929594505050565b60006020828403121561124557600080fd5b8151610f7b81610f39565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600181811c9082168061129357607f821691505b6020821081036112cc577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b601f82111561131c57600081815260208120601f850160051c810160208610156112f95750805b601f850160051c820191505b8181101561131857828155600101611305565b5050505b505050565b815167ffffffffffffffff81111561133b5761133b611250565b61134f81611349845461127f565b846112d2565b602080601f8311600181146113a2576000841561136c5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b178555611318565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b828110156113ef578886015182559484019460019091019084016113d0565b508582101561142b57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b87815286602082015285604082015260a0606082015260006114aa60a08301868861143b565b828103608084015261118681858761143b565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203611515577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b5060010190565b898152886020820152876040820152600073ffffffffffffffffffffffffffffffffffffffff808916606084015280881660808401525060e060a083015261156860e08301868861143b565b82810360c084015261157b81858761143b565b9c9b505050505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff841681526060602082015260006115ba6060830185610f82565b905063ffffffff8316604083015294935050505056fea164736f6c634300080f000a",
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
// Solidity: function submits(address , uint256 ) view returns(uint256 index, uint256 length, uint256 price, address user, address broadcaster, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentCaller) Submits(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
	var out []interface{}
	err := _L2DomiconCommitment.contract.Call(opts, &out, "submits", arg0, arg1)

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
func (_L2DomiconCommitment *L2DomiconCommitmentSession) Submits(arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
	return _L2DomiconCommitment.Contract.Submits(&_L2DomiconCommitment.CallOpts, arg0, arg1)
}

// Submits is a free data retrieval call binding the contract method 0xdcf36d57.
//
// Solidity: function submits(address , uint256 ) view returns(uint256 index, uint256 length, uint256 price, address user, address broadcaster, bytes sign, bytes commitment)
func (_L2DomiconCommitment *L2DomiconCommitmentCallerSession) Submits(arg0 common.Address, arg1 *big.Int) (struct {
	Index       *big.Int
	Length      *big.Int
	Price       *big.Int
	User        common.Address
	Broadcaster common.Address
	Sign        []byte
	Commitment  []byte
}, error) {
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

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactor) SubmitCommitment(opts *bind.TransactOpts, _index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.contract.Transact(opts, "SubmitCommitment", _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentSession) SubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.SubmitCommitment(&_L2DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
}

// SubmitCommitment is a paid mutator transaction binding the contract method 0xe4a200c8.
//
// Solidity: function SubmitCommitment(uint256 _index, uint256 _length, uint256 _price, address _user, bytes _sign, bytes _commitment) returns()
func (_L2DomiconCommitment *L2DomiconCommitmentTransactorSession) SubmitCommitment(_index *big.Int, _length *big.Int, _price *big.Int, _user common.Address, _sign []byte, _commitment []byte) (*types.Transaction, error) {
	return _L2DomiconCommitment.Contract.SubmitCommitment(&_L2DomiconCommitment.TransactOpts, _index, _length, _price, _user, _sign, _commitment)
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
