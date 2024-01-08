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
	ABI: "[{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"_messenger\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"destinationChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ERC20BridgeInitiated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"destinationChain\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"ETHBridgeInitiated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MESSENGER\",\"outputs\":[{\"internalType\":\"contractInteropL2CrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeERC20To\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"bridgeETHTo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_localToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeERC20\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"finalizeBridgeETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051611b95380380611b9583398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051611acb6100ca60003960008181610110015281816101a2015281816101e30152818161054601528181610633015281816106740152818161092701528181610a300152818161101601526112f90152611acb6000f3fe6080604052600436106100655760003560e01c8063927ede2d11610043578063927ede2d146100fe578063c28af70114610157578063f6e7e0181461017757600080fd5b80630166a07a1461006a5780631635f5fd1461008c57806354fd4d501461009f575b600080fd5b34801561007657600080fd5b5061008a610085366004611604565b61018a565b005b61008a61009a36600461169c565b61061b565b3480156100ab57600080fd5b506100e86040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b6040516100f5919061177a565b60405180910390f35b34801561010a57600080fd5b506101327f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100f5565b34801561016357600080fd5b5061008a6101723660046117a6565b610bac565b61008a610185366004611845565b611148565b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561028857503073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa15801561024c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906102709190611899565b73ffffffffffffffffffffffffffffffffffffffff16145b61033f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a4015b60405180910390fd5b8673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa15801561038a573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103ae9190611899565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff161461048e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605360248201527f496e7465726f704c325374616e646172644272696467653a2077726f6e67207260448201527f656d6f746520746f6b656e20666f72204f7074696d69736d204d696e7461626c60648201527f65204552433230206c6f63616c20746f6b656e00000000000000000000000000608482015260a401610336565b6040517f40c10f1900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8581166004830152602482018590528816906340c10f1990604401600060405180830381600087803b1580156104fe57600080fd5b505af1158015610512573d6000803e3d6000fd5b505050508473ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663817e0f906040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105af573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906105d391906118b6565b7feaae742fc63d30b6192f73412b23b6d13379db7cd6203129d365aad63df89e14898888888860405161060a959493929190611918565b60405180910390a450505050505050565b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561071957503073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa1580156106dd573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107019190611899565b73ffffffffffffffffffffffffffffffffffffffff16145b6107cb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f5374616e646172644272696467653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20746865206f7468657220627269646760648201527f6500000000000000000000000000000000000000000000000000000000000000608482015260a401610336565b823414610880576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f496e7465726f704c325374616e646172644272696467653a20616d6f756e742060448201527f73656e7420646f6573206e6f74206d6174636820616d6f756e7420726571756960648201527f7265640000000000000000000000000000000000000000000000000000000000608482015260a401610336565b3073ffffffffffffffffffffffffffffffffffffffff851603610925576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f496e7465726f704c325374616e646172644272696467653a2063616e6e6f742060448201527f73656e6420746f2073656c6600000000000000000000000000000000000000006064820152608401610336565b7f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610a00576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603160248201527f496e7465726f704c325374616e646172644272696467653a2063616e6e6f742060448201527f73656e6420746f206d657373656e6765720000000000000000000000000000006064820152608401610336565b8373ffffffffffffffffffffffffffffffffffffffff168573ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663817e0f906040518163ffffffff1660e01b8152600401602060405180830381865afa158015610a99573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610abd91906118b6565b7fc213a139c92f889f26ff6474d914ee4285be601b9ab9bf1355ffa333daf09543868686604051610af093929190611958565b60405180910390a46000610b15855a8660405180602001604052806000815250611425565b905080610ba4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f496e7465726f704c325374616e646172644272696467653a204554482074726160448201527f6e73666572206661696c656400000000000000000000000000000000000000006064820152608401610336565b505050505050565b333b15610c3b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610336565b610c65877fec4fc8e30000000000000000000000000000000000000000000000000000000061143f565b610d17576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f79206272696467652074686520494f7074696d69736d4d696e7461626c65455260648201527f43323020696e7465726661636500000000000000000000000000000000000000608482015260a401610336565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8801610dd1576040517f540abf730000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109063540abf7390610d9a908a908a908a908a908a908a908a9060040161197b565b600060405180830381600087803b158015610db457600080fd5b505af1158015610dc8573d6000803e3d6000fd5b5050505061113e565b8673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa158015610e1c573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610e409190611899565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614610f20576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605360248201527f496e7465726f704c325374616e646172644272696467653a2077726f6e67207260448201527f656d6f746520746f6b656e20666f72204f7074696d69736d204d696e7461626c60648201527f65204552433230206c6f63616c20746f6b656e00000000000000000000000000608482015260a401610336565b6040517f9dc29fac0000000000000000000000000000000000000000000000000000000081523360048201526024810185905273ffffffffffffffffffffffffffffffffffffffff881690639dc29fac90604401600060405180830381600087803b158015610f8e57600080fd5b505af1158015610fa2573d6000803e3d6000fd5b505050508573ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff16897f2b35b16e988ee2999b95340a098654d9ac6835a87c611c7b04cd352b123ea9f5338989888860405161100c959493929190611918565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663c5736a9b8930630166a07a60e01b8b8b338c8c8b8b60405160240161107797969594939291906119dc565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e086901b909216825261110b9392918990600401611a2c565b600060405180830381600087803b15801561112557600080fd5b505af1158015611139573d6000803e3d6000fd5b505050505b5050505050505050565b333b156111d7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f496e7465726f704c325374616e646172644272696467653a2063616e206f6e6c60448201527f792062652063616c6c65642066726f6d20616e20454f410000000000000000006064820152608401610336565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff850161128d576040517fe11013dd0000000000000000000000000000000000000000000000000000000081527342000000000000000000000000000000000000109063e11013dd903490611256908890889088908890600401611a78565b6000604051808303818588803b15801561126f57600080fd5b505af1158015611283573d6000803e3d6000fd5b505050505061141e565b8373ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16867fa1b6404e2aac22351000f1efebacce24eb59e9298fa0bede3f302f92975a96363486866040516112ef93929190611958565b60405180910390a47f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663c5736a9b348730631635f5fd60e01b338a348a8a604051602401611357959493929190611918565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009485161790525160e087901b90921682526113eb9392918a90600401611a2c565b6000604051808303818588803b15801561140457600080fd5b505af1158015611418573d6000803e3d6000fd5b50505050505b5050505050565b600080600080845160208601878a8af19695505050505050565b600061144a83611462565b801561145b575061145b83836114c7565b9392505050565b600061148e827f01ffc9a7000000000000000000000000000000000000000000000000000000006114c7565b80156114c157506114bf827fffffffff000000000000000000000000000000000000000000000000000000006114c7565b155b92915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d9150600051905082801561157f575060208210155b801561158b5750600081115b979650505050505050565b73ffffffffffffffffffffffffffffffffffffffff811681146115b857600080fd5b50565b60008083601f8401126115cd57600080fd5b50813567ffffffffffffffff8111156115e557600080fd5b6020830191508360208285010111156115fd57600080fd5b9250929050565b600080600080600080600060c0888a03121561161f57600080fd5b873561162a81611596565b9650602088013561163a81611596565b9550604088013561164a81611596565b9450606088013561165a81611596565b93506080880135925060a088013567ffffffffffffffff81111561167d57600080fd5b6116898a828b016115bb565b989b979a50959850939692959293505050565b6000806000806000608086880312156116b457600080fd5b85356116bf81611596565b945060208601356116cf81611596565b935060408601359250606086013567ffffffffffffffff8111156116f257600080fd5b6116fe888289016115bb565b969995985093965092949392505050565b6000815180845260005b8181101561173557602081850181015186830182015201611719565b81811115611747576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061145b602083018461170f565b803563ffffffff811681146117a157600080fd5b919050565b60008060008060008060008060e0898b0312156117c257600080fd5b8835975060208901356117d481611596565b965060408901356117e481611596565b955060608901356117f481611596565b94506080890135935061180960a08a0161178d565b925060c089013567ffffffffffffffff81111561182557600080fd5b6118318b828c016115bb565b999c989b5096995094979396929594505050565b60008060008060006080868803121561185d57600080fd5b85359450602086013561186f81611596565b935061187d6040870161178d565b9250606086013567ffffffffffffffff8111156116f257600080fd5b6000602082840312156118ab57600080fd5b815161145b81611596565b6000602082840312156118c857600080fd5b5051919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b600073ffffffffffffffffffffffffffffffffffffffff80881683528087166020840152508460408301526080606083015261158b6080830184866118cf565b8381526040602082015260006119726040830184866118cf565b95945050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a168352808916602084015280881660408401525085606083015263ffffffff8516608083015260c060a08301526119cf60c0830184866118cf565b9998505050505050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a08301526119cf60c0830184866118cf565b84815273ffffffffffffffffffffffffffffffffffffffff84166020820152608060408201526000611a61608083018561170f565b905063ffffffff8316606083015295945050505050565b73ffffffffffffffffffffffffffffffffffffffff8516815263ffffffff84166020820152606060408201526000611ab46060830184866118cf565b969550505050505056fea164736f6c634300080f000a",
}

// InteropL2StandardBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use InteropL2StandardBridgeMetaData.ABI instead.
var InteropL2StandardBridgeABI = InteropL2StandardBridgeMetaData.ABI

// InteropL2StandardBridgeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InteropL2StandardBridgeMetaData.Bin instead.
var InteropL2StandardBridgeBin = InteropL2StandardBridgeMetaData.Bin

// DeployInteropL2StandardBridge deploys a new Ethereum contract, binding an instance of InteropL2StandardBridge to it.
func DeployInteropL2StandardBridge(auth *bind.TransactOpts, backend bind.ContractBackend, _messenger common.Address) (common.Address, *types.Transaction, *InteropL2StandardBridge, error) {
	parsed, err := InteropL2StandardBridgeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InteropL2StandardBridgeBin), backend, _messenger)
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

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_InteropL2StandardBridge *InteropL2StandardBridgeCaller) MESSENGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InteropL2StandardBridge.contract.Call(opts, &out, "MESSENGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) MESSENGER() (common.Address, error) {
	return _InteropL2StandardBridge.Contract.MESSENGER(&_InteropL2StandardBridge.CallOpts)
}

// MESSENGER is a free data retrieval call binding the contract method 0x927ede2d.
//
// Solidity: function MESSENGER() view returns(address)
func (_InteropL2StandardBridge *InteropL2StandardBridgeCallerSession) MESSENGER() (common.Address, error) {
	return _InteropL2StandardBridge.Contract.MESSENGER(&_InteropL2StandardBridge.CallOpts)
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

// BridgeERC20To is a paid mutator transaction binding the contract method 0xc28af701.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactor) BridgeERC20To(opts *bind.TransactOpts, targetChain [32]byte, _localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.contract.Transact(opts, "bridgeERC20To", targetChain, _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0xc28af701.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeSession) BridgeERC20To(targetChain [32]byte, _localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeERC20To(&_InteropL2StandardBridge.TransactOpts, targetChain, _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
}

// BridgeERC20To is a paid mutator transaction binding the contract method 0xc28af701.
//
// Solidity: function bridgeERC20To(bytes32 targetChain, address _localToken, address _remoteToken, address _to, uint256 _amount, uint32 _minGasLimit, bytes _extraData) returns()
func (_InteropL2StandardBridge *InteropL2StandardBridgeTransactorSession) BridgeERC20To(targetChain [32]byte, _localToken common.Address, _remoteToken common.Address, _to common.Address, _amount *big.Int, _minGasLimit uint32, _extraData []byte) (*types.Transaction, error) {
	return _InteropL2StandardBridge.Contract.BridgeERC20To(&_InteropL2StandardBridge.TransactOpts, targetChain, _localToken, _remoteToken, _to, _amount, _minGasLimit, _extraData)
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
