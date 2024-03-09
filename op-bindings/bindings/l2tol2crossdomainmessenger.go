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

// L2ToL2CrossDomainMessengerMetaData contains all meta data concerning the L2ToL2CrossDomainMessenger contract.
var L2ToL2CrossDomainMessengerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"CROSS_DOMAIN_MESSAGE_SENDER_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"CROSS_DOMAIN_MESSAGE_SOURCE_SLOT\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MESSAGE_VERSION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint16\",\"internalType\":\"uint16\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"crossDomainMessageSender\",\"inputs\":[],\"outputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"crossDomainMessageSource\",\"inputs\":[],\"outputs\":[{\"name\":\"_source\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messageNonce\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"relayMessage\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_source\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"sendMessage\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"successfulMessages\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"FailedRelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RelayedMessage\",\"inputs\":[{\"name\":\"msgHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentMessage\",\"inputs\":[{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":true}]",
	Bin: "0x608060405234801561000f575f80fd5b50610dbf8061001d5f395ff3fe6080604052600436106100ad575f3560e01c80637056f41f11610066578063b1b1b2091161004c578063b1b1b2091461021a578063b1f35f2c14610258578063ecc704281461028b575f80fd5b80637056f41f146101d45780638fe5a280146101e7575f80fd5b806338ffde181161009657806338ffde18146101085780633f827a5a1461015957806354fd4d501461017f575f80fd5b80631ecd26f2146100b157806324794462146100c6575b5f80fd5b6100c46100bf366004610a3a565b6102bf565b005b3480156100d1575f80fd5b507f711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee75c5b6040519081526020015b60405180910390f35b348015610113575f80fd5b5060405173ffffffffffffffffffffffffffffffffffffffff7fb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f35c1681526020016100ff565b348015610164575f80fd5b5061016c5f81565b60405161ffff90911681526020016100ff565b34801561018a575f80fd5b506101c76040518060400160405280600581526020017f312e302e3000000000000000000000000000000000000000000000000000000081525081565b6040516100ff9190610ba3565b6100c46101e2366004610bbc565b6107fe565b3480156101f2575f80fd5b506100f57f711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee781565b348015610225575f80fd5b50610248610234366004610c3e565b5f6020819052908152604090205460ff1681565b60405190151581526020016100ff565b348015610263575f80fd5b506100f57fb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f381565b348015610296575f80fd5b506001547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff166100f5565b3373420000000000000000000000000000000000002214610367576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a2073656e6460448201527f6572206e6f742043726f73734c32496e626f780000000000000000000000000060648201526084015b60405180910390fd5b3073ffffffffffffffffffffffffffffffffffffffff1673420000000000000000000000000000000000002273ffffffffffffffffffffffffffffffffffffffff1663938b5f326040518163ffffffff1660e01b8152600401602060405180830381865afa1580156103db573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103ff9190610c55565b73ffffffffffffffffffffffffffffffffffffffff16146104c8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a2043726f7360448201527f734c32496e626f78206f726967696e206e6f74207468697320636f6e7472616360648201527f7400000000000000000000000000000000000000000000000000000000000000608482015260a40161035e565b468614610557576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a206465737460448201527f696e6174696f6e206e6f74207468697320636861696e00000000000000000000606482015260840161035e565b7fffffffffffffffffffffffffbdffffffffffffffffffffffffffffffffffffde73ffffffffffffffffffffffffffffffffffffffff83160161061c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603b60248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a2043726f7360448201527f734c32496e626f782063616e6e6f742063616c6c20697473656c660000000000606482015260840161035e565b5f86868686868660405160200161063896959493929190610c70565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291815281516020928301205f8181529283905291205490915060ff161561070c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a206d65737360448201527f61676520616c72656164792072656c6179656400000000000000000000000000606482015260840161035e565b5f867f711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee75d847fb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f35d5f8084516020860134885af1905080156107c9575f8281526020819052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a26107f4565b60405182907f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f905f90a25b5050505050505050565b46840361088d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c32546f4c3243726f7373446f6d61696e4d657373656e6765723a2063616e6e60448201527f6f742073656e64206d65737361676520746f2073656c66000000000000000000606482015260840161035e565b5f84466108b96001547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1690565b338787876040516024016108d39796959493929190610cc6565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f1ecd26f20000000000000000000000000000000000000000000000000000000017905251909150610958908290610ba3565b60405180910390a0600180547dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff16905f61098f83610d51565b91906101000a8154817dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff02191690837dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff160217905550505050505050565b73ffffffffffffffffffffffffffffffffffffffff81168114610a0a575f80fd5b50565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b5f805f805f8060c08789031215610a4f575f80fd5b8635955060208701359450604087013593506060870135610a6f816109e9565b92506080870135610a7f816109e9565b915060a087013567ffffffffffffffff80821115610a9b575f80fd5b818901915089601f830112610aae575f80fd5b813581811115610ac057610ac0610a0d565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610b0657610b06610a0d565b816040528281528c6020848701011115610b1e575f80fd5b826020860160208301375f6020848301015280955050505050509295509295509295565b5f81518084525f5b81811015610b6657602081850181015186830182015201610b4a565b505f6020828601015260207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f83011685010191505092915050565b602081525f610bb56020830184610b42565b9392505050565b5f805f8060608587031215610bcf575f80fd5b843593506020850135610be1816109e9565b9250604085013567ffffffffffffffff80821115610bfd575f80fd5b818701915087601f830112610c10575f80fd5b813581811115610c1e575f80fd5b886020828501011115610c2f575f80fd5b95989497505060200194505050565b5f60208284031215610c4e575f80fd5b5035919050565b5f60208284031215610c65575f80fd5b8151610bb5816109e9565b8681528560208201528460408201525f73ffffffffffffffffffffffffffffffffffffffff808616606084015280851660808401525060c060a0830152610cba60c0830184610b42565b98975050505050505050565b8781528660208201528560408201525f73ffffffffffffffffffffffffffffffffffffffff808716606084015280861660808401525060c060a08301528260c0830152828460e08401375f60e0848401015260e07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f850116830101905098975050505050505050565b5f7dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff808316818103610da8577f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b600101939250505056fea164736f6c6343000818000a",
}

// L2ToL2CrossDomainMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L2ToL2CrossDomainMessengerMetaData.ABI instead.
var L2ToL2CrossDomainMessengerABI = L2ToL2CrossDomainMessengerMetaData.ABI

// L2ToL2CrossDomainMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2ToL2CrossDomainMessengerMetaData.Bin instead.
var L2ToL2CrossDomainMessengerBin = L2ToL2CrossDomainMessengerMetaData.Bin

// DeployL2ToL2CrossDomainMessenger deploys a new Ethereum contract, binding an instance of L2ToL2CrossDomainMessenger to it.
func DeployL2ToL2CrossDomainMessenger(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L2ToL2CrossDomainMessenger, error) {
	parsed, err := L2ToL2CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2ToL2CrossDomainMessengerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2ToL2CrossDomainMessenger{L2ToL2CrossDomainMessengerCaller: L2ToL2CrossDomainMessengerCaller{contract: contract}, L2ToL2CrossDomainMessengerTransactor: L2ToL2CrossDomainMessengerTransactor{contract: contract}, L2ToL2CrossDomainMessengerFilterer: L2ToL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// L2ToL2CrossDomainMessenger is an auto generated Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessenger struct {
	L2ToL2CrossDomainMessengerCaller     // Read-only binding to the contract
	L2ToL2CrossDomainMessengerTransactor // Write-only binding to the contract
	L2ToL2CrossDomainMessengerFilterer   // Log filterer for contract events
}

// L2ToL2CrossDomainMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2ToL2CrossDomainMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2ToL2CrossDomainMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2ToL2CrossDomainMessengerSession struct {
	Contract     *L2ToL2CrossDomainMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2ToL2CrossDomainMessengerCallerSession struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// L2ToL2CrossDomainMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2ToL2CrossDomainMessengerTransactorSession struct {
	Contract     *L2ToL2CrossDomainMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// L2ToL2CrossDomainMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerRaw struct {
	Contract *L2ToL2CrossDomainMessenger // Generic contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerCallerRaw struct {
	Contract *L2ToL2CrossDomainMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L2ToL2CrossDomainMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2ToL2CrossDomainMessengerTransactorRaw struct {
	Contract *L2ToL2CrossDomainMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2ToL2CrossDomainMessenger creates a new instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessenger(address common.Address, backend bind.ContractBackend) (*L2ToL2CrossDomainMessenger, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessenger{L2ToL2CrossDomainMessengerCaller: L2ToL2CrossDomainMessengerCaller{contract: contract}, L2ToL2CrossDomainMessengerTransactor: L2ToL2CrossDomainMessengerTransactor{contract: contract}, L2ToL2CrossDomainMessengerFilterer: L2ToL2CrossDomainMessengerFilterer{contract: contract}}, nil
}

// NewL2ToL2CrossDomainMessengerCaller creates a new read-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerCaller(address common.Address, caller bind.ContractCaller) (*L2ToL2CrossDomainMessengerCaller, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerCaller{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerTransactor creates a new write-only instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L2ToL2CrossDomainMessengerTransactor, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerTransactor{contract: contract}, nil
}

// NewL2ToL2CrossDomainMessengerFilterer creates a new log filterer instance of L2ToL2CrossDomainMessenger, bound to a specific deployed contract.
func NewL2ToL2CrossDomainMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L2ToL2CrossDomainMessengerFilterer, error) {
	contract, err := bindL2ToL2CrossDomainMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerFilterer{contract: contract}, nil
}

// bindL2ToL2CrossDomainMessenger binds a generic wrapper to an already deployed contract.
func bindL2ToL2CrossDomainMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2ToL2CrossDomainMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.L2ToL2CrossDomainMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.contract.Transact(opts, method, params...)
}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CROSSDOMAINMESSAGESENDERSLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "CROSS_DOMAIN_MESSAGE_SENDER_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CROSSDOMAINMESSAGESENDERSLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESENDERSLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CROSSDOMAINMESSAGESENDERSLOT is a free data retrieval call binding the contract method 0xb1f35f2c.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SENDER_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CROSSDOMAINMESSAGESENDERSLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESENDERSLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CROSSDOMAINMESSAGESOURCESLOT is a free data retrieval call binding the contract method 0x8fe5a280.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SOURCE_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CROSSDOMAINMESSAGESOURCESLOT(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "CROSS_DOMAIN_MESSAGE_SOURCE_SLOT")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CROSSDOMAINMESSAGESOURCESLOT is a free data retrieval call binding the contract method 0x8fe5a280.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SOURCE_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CROSSDOMAINMESSAGESOURCESLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESOURCESLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CROSSDOMAINMESSAGESOURCESLOT is a free data retrieval call binding the contract method 0x8fe5a280.
//
// Solidity: function CROSS_DOMAIN_MESSAGE_SOURCE_SLOT() view returns(bytes32)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CROSSDOMAINMESSAGESOURCESLOT() ([32]byte, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CROSSDOMAINMESSAGESOURCESLOT(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MESSAGEVERSION(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "MESSAGE_VERSION")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MESSAGEVERSION() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MESSAGEVERSION is a free data retrieval call binding the contract method 0x3f827a5a.
//
// Solidity: function MESSAGE_VERSION() view returns(uint16)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MESSAGEVERSION() (uint16, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MESSAGEVERSION(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address _sender)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address _sender)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossDomainMessageSender() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSender(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSender is a free data retrieval call binding the contract method 0x38ffde18.
//
// Solidity: function crossDomainMessageSender() view returns(address _sender)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossDomainMessageSender() (common.Address, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSender(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 _source)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) CrossDomainMessageSource(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "crossDomainMessageSource")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 _source)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) CrossDomainMessageSource() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSource(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// CrossDomainMessageSource is a free data retrieval call binding the contract method 0x24794462.
//
// Solidity: function crossDomainMessageSource() view returns(uint256 _source)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) CrossDomainMessageSource() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.CrossDomainMessageSource(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) MessageNonce(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "messageNonce")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// MessageNonce is a free data retrieval call binding the contract method 0xecc70428.
//
// Solidity: function messageNonce() view returns(uint256)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) MessageNonce() (*big.Int, error) {
	return _L2ToL2CrossDomainMessenger.Contract.MessageNonce(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) SuccessfulMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "successfulMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// SuccessfulMessages is a free data retrieval call binding the contract method 0xb1b1b209.
//
// Solidity: function successfulMessages(bytes32 ) view returns(bool)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) SuccessfulMessages(arg0 [32]byte) (bool, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SuccessfulMessages(&_L2ToL2CrossDomainMessenger.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2ToL2CrossDomainMessenger.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) Version() (string, error) {
	return _L2ToL2CrossDomainMessenger.Contract.Version(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerCallerSession) Version() (string, error) {
	return _L2ToL2CrossDomainMessenger.Contract.Version(&_L2ToL2CrossDomainMessenger.CallOpts)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x1ecd26f2.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _source, uint256 _nonce, address _sender, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) RelayMessage(opts *bind.TransactOpts, _destination *big.Int, _source *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "relayMessage", _destination, _source, _nonce, _sender, _target, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x1ecd26f2.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _source, uint256 _nonce, address _sender, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) RelayMessage(_destination *big.Int, _source *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _source, _nonce, _sender, _target, _message)
}

// RelayMessage is a paid mutator transaction binding the contract method 0x1ecd26f2.
//
// Solidity: function relayMessage(uint256 _destination, uint256 _source, uint256 _nonce, address _sender, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) RelayMessage(_destination *big.Int, _source *big.Int, _nonce *big.Int, _sender common.Address, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.RelayMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _source, _nonce, _sender, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactor) SendMessage(opts *bind.TransactOpts, _destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.contract.Transact(opts, "sendMessage", _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}

// SendMessage is a paid mutator transaction binding the contract method 0x7056f41f.
//
// Solidity: function sendMessage(uint256 _destination, address _target, bytes _message) payable returns()
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerTransactorSession) SendMessage(_destination *big.Int, _target common.Address, _message []byte) (*types.Transaction, error) {
	return _L2ToL2CrossDomainMessenger.Contract.SendMessage(&_L2ToL2CrossDomainMessenger.TransactOpts, _destination, _target, _message)
}

// L2ToL2CrossDomainMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerFailedRelayedMessageIterator struct {
	Event *L2ToL2CrossDomainMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L2ToL2CrossDomainMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL2CrossDomainMessengerFailedRelayedMessage)
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
		it.Event = new(L2ToL2CrossDomainMessengerFailedRelayedMessage)
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
func (it *L2ToL2CrossDomainMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL2CrossDomainMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL2CrossDomainMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L2ToL2CrossDomainMessengerFailedRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerFailedRelayedMessageIterator{contract: _L2ToL2CrossDomainMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *L2ToL2CrossDomainMessengerFailedRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL2CrossDomainMessengerFailedRelayedMessage)
				if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*L2ToL2CrossDomainMessengerFailedRelayedMessage, error) {
	event := new(L2ToL2CrossDomainMessengerFailedRelayedMessage)
	if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2ToL2CrossDomainMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerRelayedMessageIterator struct {
	Event *L2ToL2CrossDomainMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2ToL2CrossDomainMessengerRelayedMessage)
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
		it.Event = new(L2ToL2CrossDomainMessengerRelayedMessage)
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
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2ToL2CrossDomainMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2ToL2CrossDomainMessengerRelayedMessage represents a RelayedMessage event raised by the L2ToL2CrossDomainMessenger contract.
type L2ToL2CrossDomainMessengerRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L2ToL2CrossDomainMessengerRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.FilterLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L2ToL2CrossDomainMessengerRelayedMessageIterator{contract: _L2ToL2CrossDomainMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *L2ToL2CrossDomainMessengerRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L2ToL2CrossDomainMessenger.contract.WatchLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2ToL2CrossDomainMessengerRelayedMessage)
				if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L2ToL2CrossDomainMessenger *L2ToL2CrossDomainMessengerFilterer) ParseRelayedMessage(log types.Log) (*L2ToL2CrossDomainMessengerRelayedMessage, error) {
	event := new(L2ToL2CrossDomainMessengerRelayedMessage)
	if err := _L2ToL2CrossDomainMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
