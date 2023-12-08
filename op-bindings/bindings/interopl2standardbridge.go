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

// InteropL2StandardBridgeMetaData contains all meta data concerning the InteropL2StandardBridge contract.
var InteropL2StandardBridgeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"destinationChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"destinationChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeInitiated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC20To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeETHTo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeERC20\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405273420000000000000000000000000000000000000760805234801561002857600080fd5b5060805161193561007c6000396000818161013e0152818161017f015281816104bc0152818161092001528181610a6a01528181610aab01528181610d5e01528181610e67015261119401526119356000f3fe60806040526004361061005a5760003560e01c80631635f5fd116100435780631635f5fd146100a157806354fd4d50146100b4578063f6e7e0181461011357600080fd5b80630166a07a1461005f5780630bc848c214610081575b600080fd5b34801561006b57600080fd5b5061007f61007a36600461149f565b610126565b005b34801561008d57600080fd5b5061007f61009c366004611550565b610591565b61007f6100af3660046115be565b610a52565b3480156100c057600080fd5b506100fd6040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b60405161010a919061169c565b60405180910390f35b61007f6101213660046116af565b610fe3565b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561022457503073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101e8573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061020c9190611703565b73ffffffffffffffffffffffffffffffffffffffff16145b6102db576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b8573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa15801561033d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103619190611703565b73ffffffffffffffffffffffffffffffffffffffff1614610404576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e7465726f704c325374616e646172644272696467653a2072656d6f74652060448201527f746f6b656e206d69736d6174636800000000000000000000000000000000000060648201526084016102d2565b6040517f40c10f1900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8581166004830152602482018590528816906340c10f1990604401600060405180830381600087803b15801561047457600080fd5b505af1158015610488573d6000803e3d6000fd5b505050508473ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663817e0f906040518163ffffffff1660e01b8152600401602060405180830381865afa158015610525573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105499190611720565b7feaae742fc63d30b6192f73412b23b6d13379db7cd6203129d365aad63df89e148988888888604051610580959493929190611782565b60405180910390a450505050505050565b333b15610620576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f4100000000000000000060648201526084016102d2565b61064a867fec4fc8e3000000000000000000000000000000000000000000000000000000006112c0565b6106fc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f79206272696467652074686520494f7074696d69736d4d696e7461626c65455260648201527f43323020696e7465726661636500000000000000000000000000000000000000608482015260a4016102d2565b60008673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa158015610749573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061076d9190611703565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff880161082a576040517f540abf730000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109063540abf73906107f2908a9085908b908b908b908b908b906004016117c2565b600060405180830381600087803b15801561080c57600080fd5b505af1158015610820573d6000803e3d6000fd5b5050505050610a49565b6040517f9dc29fac0000000000000000000000000000000000000000000000000000000081523360048201526024810186905273ffffffffffffffffffffffffffffffffffffffff881690639dc29fac90604401600060405180830381600087803b15801561089857600080fd5b505af11580156108ac573d6000803e3d6000fd5b505050508073ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff16897f2b35b16e988ee2999b95340a098654d9ac6835a87c611c7b04cd352b123ea9f5338a8a8989604051610916959493929190611782565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663c5736a9b8930630166a07a60e01b8b86338d8d8c8c6040516024016109819796959493929190611823565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e086901b9092168252610a159392918a90600401611873565b600060405180830381600087803b158015610a2f57600080fd5b505af1158015610a43573d6000803e3d6000fd5b50505050505b50505050505050565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016148015610b5057503073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610b14573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b389190611703565b73ffffffffffffffffffffffffffffffffffffffff16145b610c02576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a4016102d2565b823414610cb7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f496e7465726f704c325374616e646172644272696467653a20616d6f756e742060448201527f73656e7420646f6573206e6f74206d6174636820616d6f756e7420726571756960648201527f7265640000000000000000000000000000000000000000000000000000000000608482015260a4016102d2565b3073ffffffffffffffffffffffffffffffffffffffff851603610d5c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f496e7465726f704c325374616e646172644272696467653a2063616e6e6f742060448201527f73656e6420746f2073656c66000000000000000000000000000000000000000060648201526084016102d2565b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610e37576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603160248201527f496e7465726f704c325374616e646172644272696467653a2063616e6e6f742060448201527f73656e6420746f206d657373656e67657200000000000000000000000000000060648201526084016102d2565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663817e0f906040518163ffffffff1660e01b8152600401602060405180830381865afa158015610ed0573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610ef49190611720565b7fc213a139c92f889f26ff6474d914ee4285be601b9ab9bf1355ffa333daf09543868686604051610f27939291906118bf565b60405180910390a46000610f4c855a86604051806020016040528060008152506112e3565b905080610fdb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f496e7465726f704c325374616e646172644272696467653a204554482074726160448201527f6e73666572206661696c6564000000000000000000000000000000000000000060648201526084016102d2565b505050505050565b333b15611072576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f4100000000000000000060648201526084016102d2565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8501611128576040517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109063e11013dd9034906110f19088908890889088906004016118e2565b6000604051808303818588803b15801561110a57600080fd5b505af115801561111e573d6000803e3d6000fd5b50505050506112b9565b8373ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16867fa1b6404e2aac22351000f1efebacce24eb59e9298fa0bede3f302f92975a963634868660405161118a939291906118bf565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663c5736a9b348730631635f5fd60e01b338a348a8a6040516024016111f2959493929190611782565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e087901b90921682526112869392918a90600401611873565b6000604051808303818588803b15801561129f57600080fd5b505af11580156112b3573d6000803e3d6000fd5b50505050505b5050505050565b60006112cb836112fd565b80156112dc57506112dc8383611362565b9392505050565b600080600080845160208601878a8af19695505050505050565b6000611329827f01ffc9a700000000000000000000000000000000000000000000000000000000611362565b801561135c575061135a827fffffffff00000000000000000000000000000000000000000000000000000000611362565b155b92915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d9150600051905082801561141a575060208210155b80156114265750600081115b979650505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461145357600080fd5b50565b60008083601f84011261146857600080fd5b50813567ffffffffffffffff81111561148057600080fd5b60208301915083602082850101111561149857600080fd5b9250929050565b600080600080600080600060c0888a0312156114ba57600080fd5b87356114c581611431565b965060208801356114d581611431565b955060408801356114e581611431565b945060608801356114f581611431565b93506080880135925060a088013567ffffffffffffffff81111561151857600080fd5b6115248a828b01611456565b989b979a50959850939692959293505050565b803563ffffffff8116811461154b57600080fd5b919050565b600080600080600080600060c0888a03121561156b57600080fd5b87359650602088013561157d81611431565b9550604088013561158d81611431565b9450606088013593506115a260808901611537565b925060a088013567ffffffffffffffff81111561151857600080fd5b6000806000806000608086880312156115d657600080fd5b85356115e181611431565b945060208601356115f181611431565b935060408601359250606086013567ffffffffffffffff81111561161457600080fd5b61162088828901611456565b969995985093965092949392505050565b6000815180845260005b818110156116575760208185018101518683018201520161163b565b81811115611669576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006112dc6020830184611631565b6000806000806000608086880312156116c757600080fd5b8535945060208601356116d981611431565b93506116e760408701611537565b9250606086013567ffffffffffffffff81111561161457600080fd5b60006020828403121561171557600080fd5b81516112dc81611431565b60006020828403121561173257600080fd5b5051919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b600073ffffffffffffffffffffffffffffffffffffffff808816835280871660208401525084604083015260806060830152611426608083018486611739565b600073ffffffffffffffffffffffffffffffffffffffff808a168352808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a083015261181660c083018486611739565b9998505050505050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a083015261181660c083018486611739565b84815273ffffffffffffffffffffffffffffffffffffffff841660208201526080604082015260006118a86080830185611631565b905063ffffffff8316606083015295945050505050565b8381526040602082015260006118d9604083018486611739565b95945050505050565b73ffffffffffffffffffffffffffffffffffffffff8516815263ffffffff8416602082015260606040820152600061191e606083018486611739565b969550505050505056fea164736f6c634300080f000a",
}

// InteropL2StandardBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use InteropL2StandardBridgeMetaData.ABI instead.
var InteropL2StandardBridgeABI = InteropL2StandardBridgeMetaData.ABI

// InteropL2StandardBridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InteropL2StandardBridgeMetaData.Bin instead.
var InteropL2StandardBridgeBin = InteropL2StandardBridgeMetaData.Bin

// DeployInteropL2StandardBridge deploys a new Ethereum contract, binding an instance of InteropL2StandardBridge to it.
func DeployInteropL2StandardBridge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *InteropL2StandardBridge, error) {
	parsed, err := InteropL2StandardBridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InteropL2StandardBridgeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InteropL2StandardBridge{InteropL2StandardBridgeCaller: InteropL2StandardBridgeCaller{contract: contract}, InteropL2StandardBridgeTransactor: InteropL2StandardBridgeTransactor{contract: contract}, InteropL2StandardBridgeFilterer: InteropL2StandardBridgeFilterer{contract: contract}}, nil
}

// InteropL2StandardBridge is an auto generated Go binding around an Ethereum contract.
type InteropL2StandardBridge struct {
	InteropL2StandardBridgeCaller     // Read-only binding to the contract
	InteropL2StandardBridgeTransactor // Write-only binding to the contract
	InteropL2StandardBridgeFilterer   // Log filterer for contract events
}

// InteropL2StandardBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type InteropL2StandardBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2StandardBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InteropL2StandardBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2StandardBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InteropL2StandardBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2StandardBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InteropL2StandardBridgeSession struct {
	Contract     *InteropL2StandardBridge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// InteropL2StandardBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InteropL2StandardBridgeCallerSession struct {
	Contract *InteropL2StandardBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// InteropL2StandardBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InteropL2StandardBridgeTransactorSession struct {
	Contract     *InteropL2StandardBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// InteropL2StandardBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type InteropL2StandardBridgeRaw struct {
	Contract *InteropL2StandardBridge // Generic contract binding to access the raw methods on
}

// InteropL2StandardBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InteropL2StandardBridgeCallerRaw struct {
	Contract *InteropL2StandardBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// InteropL2StandardBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InteropL2StandardBridgeTransactorRaw struct {
	Contract *InteropL2StandardBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInteropL2StandardBridge creates a new instance of InteropL2StandardBridge, bound to a specific deployed contract.
func NewInteropL2StandardBridge(address common.Address, backend bind.ContractBackend) (*InteropL2StandardBridge, error) {
	contract, err := bindInteropL2StandardBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridge{InteropL2StandardBridgeCaller: InteropL2StandardBridgeCaller{contract: contract}, InteropL2StandardBridgeTransactor: InteropL2StandardBridgeTransactor{contract: contract}, InteropL2StandardBridgeFilterer: InteropL2StandardBridgeFilterer{contract: contract}}, nil
}

// NewInteropL2StandardBridgeCaller creates a new read-only instance of InteropL2StandardBridge, bound to a specific deployed contract.
func NewInteropL2StandardBridgeCaller(address common.Address, caller bind.ContractCaller) (*InteropL2StandardBridgeCaller, error) {
	contract, err := bindInteropL2StandardBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeCaller{contract: contract}, nil
}

// NewInteropL2StandardBridgeTransactor creates a new write-only instance of InteropL2StandardBridge, bound to a specific deployed contract.
func NewInteropL2StandardBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*InteropL2StandardBridgeTransactor, error) {
	contract, err := bindInteropL2StandardBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeTransactor{contract: contract}, nil
}

// NewInteropL2StandardBridgeFilterer creates a new log filterer instance of InteropL2StandardBridge, bound to a specific deployed contract.
func NewInteropL2StandardBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*InteropL2StandardBridgeFilterer, error) {
	contract, err := bindInteropL2StandardBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeFilterer{contract: contract}, nil
}

// bindInteropL2StandardBridge binds a generic wrapper to an already deployed contract.
func bindInteropL2StandardBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(InteropL2StandardBridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropL2StandardBridge *InteropL2StandardBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropL2StandardBridge.Contract.InteropL2StandardBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropL2StandardBridge *InteropL2StandardBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.InteropL2StandardBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropL2StandardBridge *InteropL2StandardBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.InteropL2StandardBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropL2StandardBridge *InteropL2StandardBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropL2StandardBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.contract.Transact(opts, method, params...)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2StandardBridge *InteropL2StandardBridgeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InteropL2StandardBridge.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) Version() (string, error) {
	return _InteropL2StandardBridge.Contract.Version(&_InteropL2StandardBridge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2StandardBridge *InteropL2StandardBridgeCallerSession) Version() (string, error) {
	return _InteropL2StandardBridge.Contract.Version(&_InteropL2StandardBridge.CallOpts)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x0bc848c2.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactor) BridgeERC20To(opts *bind.TransactOpts, targetChain [32]byte, _localToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.contract.Transact(opts, "bridgeERC20To", targetChain, _localToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x0bc848c2.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) BridgeERC20To(targetChain [32]byte, _localToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeERC20To(&_InteropL2StandardBridge.TransactOpts, targetChain, _localToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0x0bc848c2.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorSession) BridgeERC20To(targetChain [32]byte, _localToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeERC20To(&_InteropL2StandardBridge.TransactOpts, targetChain, _localToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xf6e7e018.
//
// Solidity: function bridgeETHTo(bytes32 targetChain, address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactor) BridgeETHTo(opts *bind.TransactOpts, targetChain [32]byte, _to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.contract.Transact(opts, "bridgeETHTo", targetChain, _to, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xf6e7e018.
//
// Solidity: function bridgeETHTo(bytes32 targetChain, address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) BridgeETHTo(targetChain [32]byte, _to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeETHTo(&_InteropL2StandardBridge.TransactOpts, targetChain, _to, _minGasLimit, _extraData)
}

// BridgeETHTo is a paid mutator transaction binding the contract method 0xf6e7e018.
//
// Solidity: function bridgeETHTo(bytes32 targetChain, address _to, uint32 _minGasLimit, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorSession) BridgeETHTo(targetChain [32]byte, _to common.Address, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeETHTo(&_InteropL2StandardBridge.TransactOpts, targetChain, _to, _minGasLimit, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactor) FinalizeBridgeERC20(opts *bind.TransactOpts, _localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.contract.Transact(opts, "finalizeBridgeERC20", _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) FinalizeBridgeERC20(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.FinalizeBridgeERC20(&_InteropL2StandardBridge.TransactOpts, _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeERC20 is a paid mutator transaction binding the contract method 0x0166a07a.
//
// Solidity: function finalizeBridgeERC20(address _localToken, address _remoteToken, address _from, address _to, uint256 _amount, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorSession) FinalizeBridgeERC20(_localToken common.Address, _remoteToken common.Address, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.FinalizeBridgeERC20(&_InteropL2StandardBridge.TransactOpts, _localToken, _remoteToken, _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactor) FinalizeBridgeETH(opts *bind.TransactOpts, _from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.contract.Transact(opts, "finalizeBridgeETH", _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) FinalizeBridgeETH(_from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.FinalizeBridgeETH(&_InteropL2StandardBridge.TransactOpts, _from, _to, _amount, _extraData)
}

// FinalizeBridgeETH is a paid mutator transaction binding the contract method 0x1635f5fd.
//
// Solidity: function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes _extraData) payable returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorSession) FinalizeBridgeETH(_from common.Address, _to common.Address, _amount *big.Int, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.FinalizeBridgeETH(&_InteropL2StandardBridge.TransactOpts, _from, _to, _amount, _extraData)
}

// InteropL2StandardBridgeERC20BridgeFinalizedIterator is returned from FilterERC20BridgeFinalized and is used to iterate over the raw logs and unpacked data for ERC20BridgeFinalized events raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeERC20BridgeFinalizedIterator struct {
	Event *InteropL2StandardBridgeERC20BridgeFinalized // Event containing the contract specifics and raw log

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
func (it *InteropL2StandardBridgeERC20BridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2StandardBridgeERC20BridgeFinalized)
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
		it.Event = new(InteropL2StandardBridgeERC20BridgeFinalized)
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
func (it *InteropL2StandardBridgeERC20BridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2StandardBridgeERC20BridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2StandardBridgeERC20BridgeFinalized represents a ERC20BridgeFinalized event raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeERC20BridgeFinalized struct {
	SourceChain [32]byte
	LocalToken  common.Address
	From        common.Address
	RemoteToken common.Address
	To          common.Address
	Amount      *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterERC20BridgeFinalized is a free log retrieval operation binding the contract event 0xeaae742fc63d30b6192f73412b23b6d13379db7cd6203129d365aad63df89e14.
//
// Solidity: event ERC20BridgeFinalized(bytes32 indexed sourceChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) FilterERC20BridgeFinalized(opts *bind.FilterOpts, sourceChain [][32]byte, localToken []common.Address, from []common.Address) (*InteropL2StandardBridgeERC20BridgeFinalizedIterator, error) {

	var sourceChainRule []interface{}
	for _, sourceChainItem := range sourceChain {
		sourceChainRule = append(sourceChainRule, sourceChainItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.FilterLogs(opts, "ERC20BridgeFinalized", sourceChainRule, localTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeERC20BridgeFinalizedIterator{contract: _InteropL2StandardBridge.contract, event: "ERC20BridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchERC20BridgeFinalized is a free log subscription operation binding the contract event 0xeaae742fc63d30b6192f73412b23b6d13379db7cd6203129d365aad63df89e14.
//
// Solidity: event ERC20BridgeFinalized(bytes32 indexed sourceChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) WatchERC20BridgeFinalized(opts *bind.WatchOpts, sink chan<- *InteropL2StandardBridgeERC20BridgeFinalized, sourceChain [][32]byte, localToken []common.Address, from []common.Address) (event.Subscription, error) {

	var sourceChainRule []interface{}
	for _, sourceChainItem := range sourceChain {
		sourceChainRule = append(sourceChainRule, sourceChainItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.WatchLogs(opts, "ERC20BridgeFinalized", sourceChainRule, localTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2StandardBridgeERC20BridgeFinalized)
				if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ERC20BridgeFinalized", log); err != nil {
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

// ParseERC20BridgeFinalized is a log parse operation binding the contract event 0xeaae742fc63d30b6192f73412b23b6d13379db7cd6203129d365aad63df89e14.
//
// Solidity: event ERC20BridgeFinalized(bytes32 indexed sourceChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) ParseERC20BridgeFinalized(log types.Log) (*InteropL2StandardBridgeERC20BridgeFinalized, error) {
	event := new(InteropL2StandardBridgeERC20BridgeFinalized)
	if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ERC20BridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropL2StandardBridgeERC20BridgeInitiatedIterator is returned from FilterERC20BridgeInitiated and is used to iterate over the raw logs and unpacked data for ERC20BridgeInitiated events raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeERC20BridgeInitiatedIterator struct {
	Event *InteropL2StandardBridgeERC20BridgeInitiated // Event containing the contract specifics and raw log

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
func (it *InteropL2StandardBridgeERC20BridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2StandardBridgeERC20BridgeInitiated)
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
		it.Event = new(InteropL2StandardBridgeERC20BridgeInitiated)
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
func (it *InteropL2StandardBridgeERC20BridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2StandardBridgeERC20BridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2StandardBridgeERC20BridgeInitiated represents a ERC20BridgeInitiated event raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeERC20BridgeInitiated struct {
	DestinationChain [32]byte
	LocalToken       common.Address
	From             common.Address
	RemoteToken      common.Address
	To               common.Address
	Amount           *big.Int
	ExtraData        []byte
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterERC20BridgeInitiated is a free log retrieval operation binding the contract event 0x2b35b16e988ee2999b95340a098654d9ac6835a87c611c7b04cd352b123ea9f5.
//
// Solidity: event ERC20BridgeInitiated(bytes32 indexed destinationChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) FilterERC20BridgeInitiated(opts *bind.FilterOpts, destinationChain [][32]byte, localToken []common.Address, from []common.Address) (*InteropL2StandardBridgeERC20BridgeInitiatedIterator, error) {

	var destinationChainRule []interface{}
	for _, destinationChainItem := range destinationChain {
		destinationChainRule = append(destinationChainRule, destinationChainItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.FilterLogs(opts, "ERC20BridgeInitiated", destinationChainRule, localTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeERC20BridgeInitiatedIterator{contract: _InteropL2StandardBridge.contract, event: "ERC20BridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchERC20BridgeInitiated is a free log subscription operation binding the contract event 0x2b35b16e988ee2999b95340a098654d9ac6835a87c611c7b04cd352b123ea9f5.
//
// Solidity: event ERC20BridgeInitiated(bytes32 indexed destinationChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) WatchERC20BridgeInitiated(opts *bind.WatchOpts, sink chan<- *InteropL2StandardBridgeERC20BridgeInitiated, destinationChain [][32]byte, localToken []common.Address, from []common.Address) (event.Subscription, error) {

	var destinationChainRule []interface{}
	for _, destinationChainItem := range destinationChain {
		destinationChainRule = append(destinationChainRule, destinationChainItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.WatchLogs(opts, "ERC20BridgeInitiated", destinationChainRule, localTokenRule, fromRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2StandardBridgeERC20BridgeInitiated)
				if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ERC20BridgeInitiated", log); err != nil {
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

// ParseERC20BridgeInitiated is a log parse operation binding the contract event 0x2b35b16e988ee2999b95340a098654d9ac6835a87c611c7b04cd352b123ea9f5.
//
// Solidity: event ERC20BridgeInitiated(bytes32 indexed destinationChain, address indexed localToken, address indexed from, address remoteToken, address to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) ParseERC20BridgeInitiated(log types.Log) (*InteropL2StandardBridgeERC20BridgeInitiated, error) {
	event := new(InteropL2StandardBridgeERC20BridgeInitiated)
	if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ERC20BridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropL2StandardBridgeETHBridgeFinalizedIterator is returned from FilterETHBridgeFinalized and is used to iterate over the raw logs and unpacked data for ETHBridgeFinalized events raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeETHBridgeFinalizedIterator struct {
	Event *InteropL2StandardBridgeETHBridgeFinalized // Event containing the contract specifics and raw log

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
func (it *InteropL2StandardBridgeETHBridgeFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2StandardBridgeETHBridgeFinalized)
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
		it.Event = new(InteropL2StandardBridgeETHBridgeFinalized)
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
func (it *InteropL2StandardBridgeETHBridgeFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2StandardBridgeETHBridgeFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2StandardBridgeETHBridgeFinalized represents a ETHBridgeFinalized event raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeETHBridgeFinalized struct {
	SourceChain [32]byte
	From        common.Address
	To          common.Address
	Amount      *big.Int
	ExtraData   []byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterETHBridgeFinalized is a free log retrieval operation binding the contract event 0xc213a139c92f889f26ff6474d914ee4285be601b9ab9bf1355ffa333daf09543.
//
// Solidity: event ETHBridgeFinalized(bytes32 indexed sourceChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) FilterETHBridgeFinalized(opts *bind.FilterOpts, sourceChain [][32]byte, from []common.Address, to []common.Address) (*InteropL2StandardBridgeETHBridgeFinalizedIterator, error) {

	var sourceChainRule []interface{}
	for _, sourceChainItem := range sourceChain {
		sourceChainRule = append(sourceChainRule, sourceChainItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.FilterLogs(opts, "ETHBridgeFinalized", sourceChainRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeETHBridgeFinalizedIterator{contract: _InteropL2StandardBridge.contract, event: "ETHBridgeFinalized", logs: logs, sub: sub}, nil
}

// WatchETHBridgeFinalized is a free log subscription operation binding the contract event 0xc213a139c92f889f26ff6474d914ee4285be601b9ab9bf1355ffa333daf09543.
//
// Solidity: event ETHBridgeFinalized(bytes32 indexed sourceChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) WatchETHBridgeFinalized(opts *bind.WatchOpts, sink chan<- *InteropL2StandardBridgeETHBridgeFinalized, sourceChain [][32]byte, from []common.Address, to []common.Address) (event.Subscription, error) {

	var sourceChainRule []interface{}
	for _, sourceChainItem := range sourceChain {
		sourceChainRule = append(sourceChainRule, sourceChainItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.WatchLogs(opts, "ETHBridgeFinalized", sourceChainRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2StandardBridgeETHBridgeFinalized)
				if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ETHBridgeFinalized", log); err != nil {
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

// ParseETHBridgeFinalized is a log parse operation binding the contract event 0xc213a139c92f889f26ff6474d914ee4285be601b9ab9bf1355ffa333daf09543.
//
// Solidity: event ETHBridgeFinalized(bytes32 indexed sourceChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) ParseETHBridgeFinalized(log types.Log) (*InteropL2StandardBridgeETHBridgeFinalized, error) {
	event := new(InteropL2StandardBridgeETHBridgeFinalized)
	if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ETHBridgeFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropL2StandardBridgeETHBridgeInitiatedIterator is returned from FilterETHBridgeInitiated and is used to iterate over the raw logs and unpacked data for ETHBridgeInitiated events raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeETHBridgeInitiatedIterator struct {
	Event *InteropL2StandardBridgeETHBridgeInitiated // Event containing the contract specifics and raw log

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
func (it *InteropL2StandardBridgeETHBridgeInitiatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2StandardBridgeETHBridgeInitiated)
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
		it.Event = new(InteropL2StandardBridgeETHBridgeInitiated)
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
func (it *InteropL2StandardBridgeETHBridgeInitiatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2StandardBridgeETHBridgeInitiatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2StandardBridgeETHBridgeInitiated represents a ETHBridgeInitiated event raised by the InteropL2StandardBridge contract.
type InteropL2StandardBridgeETHBridgeInitiated struct {
	DestinationChain [32]byte
	From             common.Address
	To               common.Address
	Amount           *big.Int
	ExtraData        []byte
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterETHBridgeInitiated is a free log retrieval operation binding the contract event 0xa1b6404e2aac22351000f1efebacce24eb59e9298fa0bede3f302f92975a9636.
//
// Solidity: event ETHBridgeInitiated(bytes32 indexed destinationChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) FilterETHBridgeInitiated(opts *bind.FilterOpts, destinationChain [][32]byte, from []common.Address, to []common.Address) (*InteropL2StandardBridgeETHBridgeInitiatedIterator, error) {

	var destinationChainRule []interface{}
	for _, destinationChainItem := range destinationChain {
		destinationChainRule = append(destinationChainRule, destinationChainItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.FilterLogs(opts, "ETHBridgeInitiated", destinationChainRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2StandardBridgeETHBridgeInitiatedIterator{contract: _InteropL2StandardBridge.contract, event: "ETHBridgeInitiated", logs: logs, sub: sub}, nil
}

// WatchETHBridgeInitiated is a free log subscription operation binding the contract event 0xa1b6404e2aac22351000f1efebacce24eb59e9298fa0bede3f302f92975a9636.
//
// Solidity: event ETHBridgeInitiated(bytes32 indexed destinationChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) WatchETHBridgeInitiated(opts *bind.WatchOpts, sink chan<- *InteropL2StandardBridgeETHBridgeInitiated, destinationChain [][32]byte, from []common.Address, to []common.Address) (event.Subscription, error) {

	var destinationChainRule []interface{}
	for _, destinationChainItem := range destinationChain {
		destinationChainRule = append(destinationChainRule, destinationChainItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InteropL2StandardBridge.contract.WatchLogs(opts, "ETHBridgeInitiated", destinationChainRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2StandardBridgeETHBridgeInitiated)
				if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ETHBridgeInitiated", log); err != nil {
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

// ParseETHBridgeInitiated is a log parse operation binding the contract event 0xa1b6404e2aac22351000f1efebacce24eb59e9298fa0bede3f302f92975a9636.
//
// Solidity: event ETHBridgeInitiated(bytes32 indexed destinationChain, address indexed from, address indexed to, uint256 amount, bytes extraData)
func (_InteropL2StandardBridge *InteropL2StandardBridgeFilterer) ParseETHBridgeInitiated(log types.Log) (*InteropL2StandardBridgeETHBridgeInitiated, error) {
	event := new(InteropL2StandardBridgeETHBridgeInitiated)
	if err := _InteropL2StandardBridge.contract.UnpackLog(event, "ETHBridgeInitiated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
