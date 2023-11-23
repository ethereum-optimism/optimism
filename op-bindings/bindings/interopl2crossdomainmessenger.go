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

// InteropL2CrossDomainMessengerMetaData contains all meta data concerning the InteropL2CrossDomainMessenger contract.
var InteropL2CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"source\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"source\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"destination\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MESSAGE_VERSION\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_CALLDATA_OVERHEAD\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAY_CALL_OVERHEAD\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAY_CONSTANT_OVERHEAD\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAY_GAS_CHECK_BUFFER\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAY_RESERVED_GAS\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"}],\"name\":\"baseGas\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"failedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_source\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minGasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"}],\"name\":\"relayMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_destination\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint32\",\"name\":\"_minGasLimit\",\"type\":\"uint32\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"successfulMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611503806100206000396000f3fe6080604052600436106100dd5760003560e01c806383a740741161007f578063a4e7f8bd11610059578063a4e7f8bd14610215578063b1b1b20914610255578063b28ade2514610285578063c5736a9b146102a557600080fd5b806383a74074146101e95780638cbeeef2146101675780638f6787511461020057600080fd5b80633f827a5a116100bb5780633f827a5a1461013f5780634c1d6a691461016757806354fd4d501461017d5780635644cfdf146101d357600080fd5b8063028f85f7146100e25780630c568498146101155780632828d7e81461012a575b600080fd5b3480156100ee57600080fd5b506100f7601081565b60405167ffffffffffffffff90911681526020015b60405180910390f35b34801561012157600080fd5b506100f7603f81565b34801561013657600080fd5b506100f7604081565b34801561014b57600080fd5b50610154600281565b60405161ffff909116815260200161010c565b34801561017357600080fd5b506100f7619c4081565b34801561018957600080fd5b506101c66040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b60405161010c9190610fda565b3480156101df57600080fd5b506100f761138881565b3480156101f557600080fd5b506100f762030d4081565b61021361020e366004611066565b6102b8565b005b34801561022157600080fd5b506102456102303660046110f1565b60036020526000908152604090205460ff1681565b604051901515815260200161010c565b34801561026157600080fd5b506102456102703660046110f1565b60026020526000908152604090205460ff1681565b34801561029157600080fd5b506100f76102a036600461111e565b610a2d565b6102136102b3366004611172565b610a9b565b60f088901c60028114610352576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a20696e636f7260448201527f72656374206d6573736167652076657273696f6e00000000000000000000000060648201526084015b60405180910390fd5b600061039a8a8a468b8b8b8b8b8b8080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610e1692505050565b60008181526002602052604090205490915060ff161561043c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a206d6573736160448201527f676520616c72656164792070726f6365737365640000000000000000000000006064820152608401610349565b7fffffffffffffffffffffffffbdffffffffffffffffffffffffffffffffffff1f330161049657853414610472576104726111e1565b60008181526003602052604090205460ff1615610491576104916111e1565b6105c2565b3415610524576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603d60248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a2063616e6e6f60448201527f74207265706c617920776974682061646469746f6e616c2066756e64730000006064820152608401610349565b60008181526003602052604090205460ff166105c2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a206d6573736160448201527f67652063616e6e6f74206265207265706c6179656400000000000000000000006064820152608401610349565b6105cb87610e3d565b1561067e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a2063616e6e6f60448201527f742073656e64206d65737361676520746f20626c6f636b65642073797374656d60648201527f2061646472657373000000000000000000000000000000000000000000000000608482015260a401610349565b61069f85610690611388619c4061123f565b67ffffffffffffffff16610e92565b15806106c5575060015473ffffffffffffffffffffffffffffffffffffffff1661dead14155b156107e25760008181526003602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555182918b918d917fc8ad05025c78e9382b6f4840403c8cce0714b7d76ee21c820260f7484dd4bd0991a47fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff32016107db576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603260248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a206661696c6560448201527f6420746f2072656c6179206d65737361676500000000000000000000000000006064820152608401610349565b5050610a23565b600180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8a16179055600061087388619c405a610836919061126b565b8988888080601f016020809104026020016040519081016040528093929190818152602001838380828437600092019190915250610eb092505050565b600180547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead1790559050801561090e5760008281526002602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183918c918e917f9d060c26cd9dce5a30923192e2f956f4444cf4511f83137058caafb29ea6ef4b91a4610a1f565b60008281526003602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183918c918e917fc8ad05025c78e9382b6f4840403c8cce0714b7d76ee21c820260f7484dd4bd0991a47fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff3201610a1f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603260248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a206661696c6560448201527f6420746f2072656c6179206d65737361676500000000000000000000000000006064820152608401610349565b5050505b5050505050505050565b6000611388619c4080603f610a49604063ffffffff8816611282565b610a5391906112b2565b610a5e601088611282565b610a6b9062030d4061123f565b610a75919061123f565b610a7f919061123f565b610a89919061123f565b610a93919061123f565b949350505050565b60017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8601610b52576040517f3dbb202b00000000000000000000000000000000000000000000000000000000815273420000000000000000000000000000000000000790633dbb202b90610b1a908890889088908890600401611349565b600060405180830381600087803b158015610b3457600080fd5b505af1158015610b48573d6000803e3d6000fd5b5050505050610e0f565b468603610be1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4e65774c3243726f7373446f6d61696e4d657373656e6765723a206d6573736160448201527f67652063616e742062652073656e7420746f2073656c660000000000000000006064820152608401610349565b600080547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167e0200000000000000000000000000000000000000000000000000000000000017905060007f8f678751000000000000000000000000000000000000000000000000000000008246338a34898c8c604051602401610c6a989796959493929190611390565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009093169290921790915290507342000000000000000000000000000000000000e0637c9582f88930610d108a8a8a610a2d565b856040518563ffffffff1660e01b8152600401610d3094939291906113f6565b600060405180830381600087803b158015610d4a57600080fd5b505af1158015610d5e573d6000803e3d6000fd5b505050508673ffffffffffffffffffffffffffffffffffffffff1688837fc69ce72212d62fdf5ffdeebf20d977f6f8bda8f6667237633b751c285fd1f262338a8a8a34604051610db2959493929190611445565b60405180910390a45050600080547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808216600101167fffff000000000000000000000000000000000000000000000000000000000000909116179055505b5050505050565b6000610e288989898989898989610eca565b80519060200120905098975050505050505050565b600073ffffffffffffffffffffffffffffffffffffffff8216301480610e8c575073ffffffffffffffffffffffffffffffffffffffff82167342000000000000000000000000000000000000e0145b92915050565b600080603f83619c4001026040850201603f5a021015949350505050565b600080600080845160208601878a8af19695505050505050565b60608888888888888888604051602401610eeb98979695949392919061148f565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529190526020810180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f49930a7100000000000000000000000000000000000000000000000000000000179052905098975050505050505050565b6000815180845260005b81811015610f9557602081850181015186830182015201610f79565b81811115610fa7576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610fed6020830184610f6f565b9392505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461101857600080fd5b919050565b60008083601f84011261102f57600080fd5b50813567ffffffffffffffff81111561104757600080fd5b60208301915083602082850101111561105f57600080fd5b9250929050565b60008060008060008060008060e0898b03121561108257600080fd5b883597506020890135965061109960408a01610ff4565b95506110a760608a01610ff4565b94506080890135935060a0890135925060c089013567ffffffffffffffff8111156110d157600080fd5b6110dd8b828c0161101d565b999c989b5096995094979396929594505050565b60006020828403121561110357600080fd5b5035919050565b803563ffffffff8116811461101857600080fd5b60008060006040848603121561113357600080fd5b833567ffffffffffffffff81111561114a57600080fd5b6111568682870161101d565b909450925061116990506020850161110a565b90509250925092565b60008060008060006080868803121561118a57600080fd5b8535945061119a60208701610ff4565b9350604086013567ffffffffffffffff8111156111b657600080fd5b6111c28882890161101d565b90945092506111d590506060870161110a565b90509295509295909350565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052600160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600067ffffffffffffffff80831681851680830382111561126257611262611210565b01949350505050565b60008282101561127d5761127d611210565b500390565b600067ffffffffffffffff808316818516818304811182151516156112a9576112a9611210565b02949350505050565b600067ffffffffffffffff808416806112f4577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910492915050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff85168152606060208201526000611379606083018587611300565b905063ffffffff8316604083015295945050505050565b888152876020820152600073ffffffffffffffffffffffffffffffffffffffff808916604084015280881660608401525085608083015263ffffffff851660a083015260e060c08301526113e860e083018486611300565b9a9950505050505050505050565b84815273ffffffffffffffffffffffffffffffffffffffff8416602082015267ffffffffffffffff8316604082015260806060820152600061143b6080830184610f6f565b9695505050505050565b73ffffffffffffffffffffffffffffffffffffffff86168152608060208201526000611475608083018688611300565b63ffffffff94909416604083015250606001529392505050565b60006101008a835289602084015288604084015273ffffffffffffffffffffffffffffffffffffffff80891660608501528088166080850152508560a08401528460c08401528060e08401526114e781840185610f6f565b9b9a505050505050505050505056fea164736f6c634300080f000a",
}

// InteropL2CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use InteropL2CrossDomainMessengerMetaData.ABI instead.
var InteropL2CrossDomainMessengerABI = InteropL2CrossDomainMessengerMetaData.ABI

// InteropL2CrossDomainMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InteropL2CrossDomainMessengerMetaData.Bin instead.
var InteropL2CrossDomainMessengerBin = InteropL2CrossDomainMessengerMetaData.Bin

// DeployInteropL2CrossDomainMessenger deploys a new Ethereum contract, binding an instance of InteropL2CrossDomainMessenger to it.
func DeployInteropL2CrossDomainMessenger(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *InteropL2CrossDomainMessenger, error) {
	parsed, err := InteropL2CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InteropL2CrossDomainMessengerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InteropL2CrossDomainMessenger{InteropL2CrossDomainMessengerCaller: InteropL2CrossDomainMessengerCaller{contract: contract}, InteropL2CrossDomainMessengerTransactor: InteropL2CrossDomainMessengerTransactor{contract: contract}, InteropL2CrossDomainMessengerFilterer: InteropL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// InteropL2CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type InteropL2CrossDomainMessenger struct {
	InteropL2CrossDomainMessengerCaller     // Read-only binding to the contract
	InteropL2CrossDomainMessengerTransactor // Write-only binding to the contract
	InteropL2CrossDomainMessengerFilterer   // Log filterer for contract events
}

// InteropL2CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type InteropL2CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InteropL2CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InteropL2CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InteropL2CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InteropL2CrossDomainMessengerSession struct {
	Contract     *InteropL2CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                  // Call options to use throughout this session
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// InteropL2CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InteropL2CrossDomainMessengerCallerSession struct {
	Contract *InteropL2CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                        // Call options to use throughout this session
}

// InteropL2CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InteropL2CrossDomainMessengerTransactorSession struct {
	Contract     *InteropL2CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                        // Transaction auth options to use throughout this session
}

// InteropL2CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type InteropL2CrossDomainMessengerRaw struct {
	Contract *InteropL2CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// InteropL2CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InteropL2CrossDomainMessengerCallerRaw struct {
	Contract *InteropL2CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// InteropL2CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InteropL2CrossDomainMessengerTransactorRaw struct {
	Contract *InteropL2CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInteropL2CrossDomainMessenger creates a new instance of InteropL2CrossDomainMessenger, bound to a specific deployed contract.
func NewInteropL2CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*InteropL2CrossDomainMessenger, error) {
	contract, err := bindInteropL2CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessenger{InteropL2CrossDomainMessengerCaller: InteropL2CrossDomainMessengerCaller{contract: contract}, InteropL2CrossDomainMessengerTransactor: InteropL2CrossDomainMessengerTransactor{contract: contract}, InteropL2CrossDomainMessengerFilterer: InteropL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewInteropL2CrossDomainMessengerCaller creates a new read-only instance of InteropL2CrossDomainMessenger, bound to a specific deployed contract.
func NewInteropL2CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*InteropL2CrossDomainMessengerCaller, error) {
	contract, err := bindInteropL2CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerCaller{contract: contract}, nil
}

// NewInteropL2CrossDomainMessengerTransactor creates a new write-only instance of InteropL2CrossDomainMessenger, bound to a specific deployed contract.
func NewInteropL2CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*InteropL2CrossDomainMessengerTransactor, error) {
	contract, err := bindInteropL2CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewInteropL2CrossDomainMessengerFilterer creates a new log filterer instance of InteropL2CrossDomainMessenger, bound to a specific deployed contract.
func NewInteropL2CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*InteropL2CrossDomainMessengerFilterer, error) {
	contract, err := bindInteropL2CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindInteropL2CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindInteropL2CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(InteropL2CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropL2CrossDomainMessenger.Contract.InteropL2CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.InteropL2CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.InteropL2CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InteropL2CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _InteropL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _InteropL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) MINGASCALLDATAOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_CALLDATA_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASCALLDATAOVERHEAD is a free data retrieval call binding the contract method 0x028f85f7.
//
// Solidity: function MIN_GAS_CALLDATA_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) MINGASCALLDATAOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASCALLDATAOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADDENOMINATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADDENOMINATOR is a free data retrieval call binding the contract method 0x0c568498.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADDENOMINATOR() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADDENOMINATOR(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) MINGASDYNAMICOVERHEADNUMERATOR(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_InteropL2CrossDomainMessenger.CallOpts)
}

// MINGASDYNAMICOVERHEADNUMERATOR is a free data retrieval call binding the contract method 0x2828d7e8.
//
// Solidity: function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) MINGASDYNAMICOVERHEADNUMERATOR() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.MINGASDYNAMICOVERHEADNUMERATOR(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) RELAYCALLOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CALL_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYCALLOVERHEAD is a free data retrieval call binding the contract method 0x4c1d6a69.
//
// Solidity: function RELAY_CALL_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) RELAYCALLOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYCALLOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) RELAYCONSTANTOVERHEAD(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "RELAY_CONSTANT_OVERHEAD")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYCONSTANTOVERHEAD is a free data retrieval call binding the contract method 0x83a74074.
//
// Solidity: function RELAY_CONSTANT_OVERHEAD() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) RELAYCONSTANTOVERHEAD() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYCONSTANTOVERHEAD(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) RELAYGASCHECKBUFFER(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "RELAY_GAS_CHECK_BUFFER")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYGASCHECKBUFFER is a free data retrieval call binding the contract method 0x5644cfdf.
//
// Solidity: function RELAY_GAS_CHECK_BUFFER() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) RELAYGASCHECKBUFFER() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYGASCHECKBUFFER(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) RELAYRESERVEDGAS(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "RELAY_RESERVED_GAS")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RELAYRESERVEDGAS is a free data retrieval call binding the contract method 0x8cbeeef2.
//
// Solidity: function RELAY_RESERVED_GAS() view returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) RELAYRESERVEDGAS() (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.RELAYRESERVEDGAS(&_InteropL2CrossDomainMessenger.CallOpts)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) BaseGas(opts *bind.CallOpts, _message []byte, _minGasLimit uint32) (uint64, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "baseGas", _message, _minGasLimit)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.BaseGas(&_InteropL2CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// BaseGas is a free data retrieval call binding the contract method 0xb28ade25.
//
// Solidity: function baseGas(bytes _message, uint32 _minGasLimit) pure returns(uint64)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) BaseGas(_message []byte, _minGasLimit uint32) (uint64, error) {
	return _InteropL2CrossDomainMessenger.Contract.BaseGas(&_InteropL2CrossDomainMessenger.CallOpts, _message, _minGasLimit)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) FailedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "failedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _InteropL2CrossDomainMessenger.Contract.FailedMessages(&_InteropL2CrossDomainMessenger.CallOpts, arg0)
}

// FailedMessages is a free data retrieval call binding the contract method 0xa4e7f8bd.
//
// Solidity: function failedMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) FailedMessages(arg0 [32]byte) (bool, error) {
	return _InteropL2CrossDomainMessenger.Contract.FailedMessages(&_InteropL2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _InteropL2CrossDomainMessenger.Contract.SuccessfulMessages(&_InteropL2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _InteropL2CrossDomainMessenger.Contract.SuccessfulMessages(&_InteropL2CrossDomainMessenger.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InteropL2CrossDomainMessenger.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) Version() (string, error) {
	return _InteropL2CrossDomainMessenger.Contract.Version(&_InteropL2CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerCallerSession) Version() (string, error) {
	return _InteropL2CrossDomainMessenger.Contract.Version(&_InteropL2CrossDomainMessenger.CallOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8f678751.
//
// Solidity: function relayMessage(uint256 _nonce, bytes32 _source, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _nonce *big.Int, _source [32]byte, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.contract.Transact(opts, "relayMessage", _nonce, _source, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8f678751.
//
// Solidity: function relayMessage(uint256 _nonce, bytes32 _source, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) RelayMessage(_nonce *big.Int, _source [32]byte, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.RelayMessage(&_InteropL2CrossDomainMessenger.TransactOpts, _nonce, _source, _sender, _target, _value, _minGasLimit, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x8f678751.
//
// Solidity: function relayMessage(uint256 _nonce, bytes32 _source, address _sender, address _target, uint256 _value, uint256 _minGasLimit, bytes _message) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactorSession) RelayMessage(_nonce *big.Int, _source [32]byte, _sender common.Address, _target common.Address, _value *big.Int, _minGasLimit *big.Int, _message []byte) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.RelayMessage(&_InteropL2CrossDomainMessenger.TransactOpts, _nonce, _source, _sender, _target, _value, _minGasLimit, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0xc5736a9b.
//
// Solidity: function sendMessage(bytes32 _destination, address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _destination [32]byte, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.contract.Transact(opts, "sendMessage", _destination, _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0xc5736a9b.
//
// Solidity: function sendMessage(bytes32 _destination, address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerSession) SendMessage(_destination [32]byte, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.SendMessage(&_InteropL2CrossDomainMessenger.TransactOpts, _destination, _target, _message, _minGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0xc5736a9b.
//
// Solidity: function sendMessage(bytes32 _destination, address _target, bytes _message, uint32 _minGasLimit) payable returns()
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerTransactorSession) SendMessage(_destination [32]byte, _target common.Address, _message []byte, _minGasLimit uint32) (*types.Transaction, error) {
	return _InteropL2CrossDomainMessenger.Contract.SendMessage(&_InteropL2CrossDomainMessenger.TransactOpts, _destination, _target, _message, _minGasLimit)
}

// InteropL2CrossDomainMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerFailedRelayedMessageIterator struct {
	Event *InteropL2CrossDomainMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *InteropL2CrossDomainMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2CrossDomainMessengerFailedRelayedMessage)
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
		it.Event = new(InteropL2CrossDomainMessengerFailedRelayedMessage)
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
func (it *InteropL2CrossDomainMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2CrossDomainMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2CrossDomainMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerFailedRelayedMessage struct {
	MessageNonce *big.Int
	Source       [32]byte
	MsgHash      [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0xc8ad05025c78e9382b6f4840403c8cce0714b7d76ee21c820260f7484dd4bd09.
//
// Solidity: event FailedRelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, messageNonce []*big.Int, source [][32]byte, msgHash [][32]byte) (*InteropL2CrossDomainMessengerFailedRelayedMessageIterator, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", messageNonceRule, sourceRule, msgHashRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerFailedRelayedMessageIterator{contract: _InteropL2CrossDomainMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0xc8ad05025c78e9382b6f4840403c8cce0714b7d76ee21c820260f7484dd4bd09.
//
// Solidity: event FailedRelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *InteropL2CrossDomainMessengerFailedRelayedMessage, messageNonce []*big.Int, source [][32]byte, msgHash [][32]byte) (event.Subscription, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", messageNonceRule, sourceRule, msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2CrossDomainMessengerFailedRelayedMessage)
				if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0xc8ad05025c78e9382b6f4840403c8cce0714b7d76ee21c820260f7484dd4bd09.
//
// Solidity: event FailedRelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*InteropL2CrossDomainMessengerFailedRelayedMessage, error) {
	event := new(InteropL2CrossDomainMessengerFailedRelayedMessage)
	if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropL2CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerRelayedMessageIterator struct {
	Event *InteropL2CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *InteropL2CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2CrossDomainMessengerRelayedMessage)
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
		it.Event = new(InteropL2CrossDomainMessengerRelayedMessage)
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
func (it *InteropL2CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerRelayedMessage struct {
	MessageNonce *big.Int
	Source       [32]byte
	MsgHash      [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x9d060c26cd9dce5a30923192e2f956f4444cf4511f83137058caafb29ea6ef4b.
//
// Solidity: event RelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, messageNonce []*big.Int, source [][32]byte, msgHash [][32]byte) (*InteropL2CrossDomainMessengerRelayedMessageIterator, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", messageNonceRule, sourceRule, msgHashRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerRelayedMessageIterator{contract: _InteropL2CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x9d060c26cd9dce5a30923192e2f956f4444cf4511f83137058caafb29ea6ef4b.
//
// Solidity: event RelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *InteropL2CrossDomainMessengerRelayedMessage, messageNonce []*big.Int, source [][32]byte, msgHash [][32]byte) (event.Subscription, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var sourceRule []interface{}
	for _, sourceItem := range source {
		sourceRule = append(sourceRule, sourceItem)
	}
	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", messageNonceRule, sourceRule, msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2CrossDomainMessengerRelayedMessage)
				if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x9d060c26cd9dce5a30923192e2f956f4444cf4511f83137058caafb29ea6ef4b.
//
// Solidity: event RelayedMessage(uint256 indexed messageNonce, bytes32 indexed source, bytes32 indexed msgHash)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*InteropL2CrossDomainMessengerRelayedMessage, error) {
	event := new(InteropL2CrossDomainMessengerRelayedMessage)
	if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InteropL2CrossDomainMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerSentMessageIterator struct {
	Event *InteropL2CrossDomainMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *InteropL2CrossDomainMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InteropL2CrossDomainMessengerSentMessage)
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
		it.Event = new(InteropL2CrossDomainMessengerSentMessage)
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
func (it *InteropL2CrossDomainMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InteropL2CrossDomainMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InteropL2CrossDomainMessengerSentMessage represents a SentMessage event raised by the InteropL2CrossDomainMessenger contract.
type InteropL2CrossDomainMessengerSentMessage struct {
	MessageNonce *big.Int
	Destination  [32]byte
	Target       common.Address
	Sender       common.Address
	Message      []byte
	GasLimit     *big.Int
	Value        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0xc69ce72212d62fdf5ffdeebf20d977f6f8bda8f6667237633b751c285fd1f262.
//
// Solidity: event SentMessage(uint256 indexed messageNonce, bytes32 indexed destination, address indexed target, address sender, bytes message, uint256 gasLimit, uint256 value)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, messageNonce []*big.Int, destination [][32]byte, target []common.Address) (*InteropL2CrossDomainMessengerSentMessageIterator, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.FilterLogs(opts, "SentMessage", messageNonceRule, destinationRule, targetRule)
	if err != nil {
		return nil, err
	}
	return &InteropL2CrossDomainMessengerSentMessageIterator{contract: _InteropL2CrossDomainMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0xc69ce72212d62fdf5ffdeebf20d977f6f8bda8f6667237633b751c285fd1f262.
//
// Solidity: event SentMessage(uint256 indexed messageNonce, bytes32 indexed destination, address indexed target, address sender, bytes message, uint256 gasLimit, uint256 value)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *InteropL2CrossDomainMessengerSentMessage, messageNonce []*big.Int, destination [][32]byte, target []common.Address) (event.Subscription, error) {

	var messageNonceRule []interface{}
	for _, messageNonceItem := range messageNonce {
		messageNonceRule = append(messageNonceRule, messageNonceItem)
	}
	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _InteropL2CrossDomainMessenger.contract.WatchLogs(opts, "SentMessage", messageNonceRule, destinationRule, targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InteropL2CrossDomainMessengerSentMessage)
				if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0xc69ce72212d62fdf5ffdeebf20d977f6f8bda8f6667237633b751c285fd1f262.
//
// Solidity: event SentMessage(uint256 indexed messageNonce, bytes32 indexed destination, address indexed target, address sender, bytes message, uint256 gasLimit, uint256 value)
func (_InteropL2CrossDomainMessenger *InteropL2CrossDomainMessengerFilterer) ParseSentMessage(log types.Log) (*InteropL2CrossDomainMessengerSentMessage, error) {
	event := new(InteropL2CrossDomainMessengerSentMessage)
	if err := _InteropL2CrossDomainMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
