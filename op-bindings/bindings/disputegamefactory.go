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

// DisputeGameFactoryMetaData contains all meta data concerning the DisputeGameFactory contract.
var DisputeGameFactoryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"create\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"gameAtIndex\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"gameType_\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"timestamp_\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"},{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameCount\",\"inputs\":[],\"outputs\":[{\"name\":\"gameCount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameImpls\",\"inputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"GameType\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"games\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"},{\"name\":\"timestamp_\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getGameUUID\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"uuid_\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setImplementation\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"_impl\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"DisputeGameCreated\",\"inputs\":[{\"name\":\"disputeProxy\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"gameType\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"GameType\"},{\"name\":\"rootClaim\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"Claim\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ImplementationSet\",\"inputs\":[{\"name\":\"impl\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"gameType\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"GameType\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"GameAlreadyExists\",\"inputs\":[{\"name\":\"uuid\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}]},{\"type\":\"error\",\"name\":\"NoImplementation\",\"inputs\":[{\"name\":\"gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"}]}]",
	Bin: "0x60806040523480156200001157600080fd5b506200001e600062000024565b62000292565b600054610100900460ff1615808015620000455750600054600160ff909116105b8062000075575062000062306200016260201b6200097e1760201c565b15801562000075575060005460ff166001145b620000de5760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff19166001179055801562000102576000805461ff0019166101001790555b6200010c62000171565b6200011782620001d9565b80156200015e576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6001600160a01b03163b151590565b600054610100900460ff16620001cd5760405162461bcd60e51b815260206004820152602b6024820152600080516020620012d183398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000d5565b620001d76200022b565b565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16620002875760405162461bcd60e51b815260206004820152602b6024820152600080516020620012d183398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000d5565b620001d733620001d9565b61102f80620002a26000396000f3fe608060405234801561001057600080fd5b50600436106100d45760003560e01c80638da5cb5b11610081578063c4d66de81161005b578063c4d66de8146102e8578063dfa162d3146102fb578063f2fde38b1461033157600080fd5b80638da5cb5b14610231578063bb8aa1fc1461024f578063c49d5271146102a057600080fd5b80634d1975b4116100b25780634d1975b4146101d857806354fd4d50146101e0578063715018a61461022957600080fd5b806326daafbe146100d95780633142e55e1461018b57806345583b7a146101c3575b600080fd5b6101786100e7366004610d45565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0810180517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc0830180517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08086018051988652968352606087529451609f0190941683209190925291905291905290565b6040519081526020015b60405180910390f35b61019e610199366004610e2e565b610344565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610182565b6101d66101d1366004610ed7565b6105a6565b005b606754610178565b61021c6040518060400160405280600581526020017f302e302e3600000000000000000000000000000000000000000000000000000081525081565b6040516101829190610f0e565b6101d661062d565b60335473ffffffffffffffffffffffffffffffffffffffff1661019e565b61026261025d366004610f81565b610641565b6040805160ff909416845267ffffffffffffffff909216602084015273ffffffffffffffffffffffffffffffffffffffff1690820152606001610182565b6102b36102ae366004610e2e565b6106a3565b6040805173ffffffffffffffffffffffffffffffffffffffff909316835267ffffffffffffffff909116602083015201610182565b6101d66102f6366004610f9a565b61072b565b61019e610309366004610fbe565b60656020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b6101d661033f366004610f9a565b6108c7565b60ff841660009081526065602052604081205473ffffffffffffffffffffffffffffffffffffffff16806103ae576040517f44265d6f00000000000000000000000000000000000000000000000000000000815260ff871660048201526024015b60405180910390fd5b6104118585856040516020016103c693929190610fd9565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905273ffffffffffffffffffffffffffffffffffffffff83169061099a565b91508173ffffffffffffffffffffffffffffffffffffffff16638129fc1c6040518163ffffffff1660e01b8152600401600060405180830381600087803b15801561045b57600080fd5b505af115801561046f573d6000803e3d6000fd5b5050505060006104b6878787878080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506100e792505050565b60008181526066602052604090205490915015610502576040517f014f6fe5000000000000000000000000000000000000000000000000000000008152600481018290526024016103a5565b60004260b81b60f889901b178417600083815260666020526040808220839055606780546001810182559083527f9787eeb91fe3101235e4a76063c7023ecb40f923f97916639c598592fa30d6ae0183905551919250889160ff8b169173ffffffffffffffffffffffffffffffffffffffff8816917ffad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec9190a4505050949350505050565b6105ae610ace565b60ff821660008181526065602052604080822080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8616908117909155905190917f623713f72f6e427a8044bb8b3bd6834357cf285decbaa21bcc73c1d0632c4d8491a35050565b610635610ace565b61063f6000610b4f565b565b60008060006106966067858154811061065c5761065c610ff3565b906000526020600020015460f881901c9167ffffffffffffffff60b883901c169173ffffffffffffffffffffffffffffffffffffffff1690565b9196909550909350915050565b60008060006106e9878787878080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152506100e792505050565b60009081526066602052604090205473ffffffffffffffffffffffffffffffffffffffff81169860b89190911c67ffffffffffffffff16975095505050505050565b600054610100900460ff161580801561074b5750600054600160ff909116105b806107655750303b158015610765575060005460ff166001145b6107f1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016103a5565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055801561084f57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610857610bc6565b61086082610b4f565b80156108c357600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050565b6108cf610ace565b73ffffffffffffffffffffffffffffffffffffffff8116610972576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016103a5565b61097b81610b4f565b50565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b60006002825101603f8101600a81036040518360581b8260e81b177f6100003d81600a3d39f3363d3d373d3d3d3d610000806035363936013d7300001781528660601b601e8201527f5af43d3d93803e603357fd5bf300000000000000000000000000000000000000603282015285519150603f8101602087015b60208410610a5257805182527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe09093019260209182019101610a15565b517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff602085900360031b1b16815260f085901b9083015282816000f0945084610abf577febfef1880000000000000000000000000000000000000000000000000000000060005260206000fd5b90910160405250909392505050565b60335473ffffffffffffffffffffffffffffffffffffffff16331461063f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016103a5565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff16610c5d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016103a5565b61063f600054610100900460ff16610cf7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016103a5565b61063f33610b4f565b803560ff81168114610d1157600080fd5b919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080600060608486031215610d5a57600080fd5b610d6384610d00565b925060208401359150604084013567ffffffffffffffff80821115610d8757600080fd5b818601915086601f830112610d9b57600080fd5b813581811115610dad57610dad610d16565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715610df357610df3610d16565b81604052828152896020848701011115610e0c57600080fd5b8260208601602083013760006020848301015280955050505050509250925092565b60008060008060608587031215610e4457600080fd5b610e4d85610d00565b935060208501359250604085013567ffffffffffffffff80821115610e7157600080fd5b818701915087601f830112610e8557600080fd5b813581811115610e9457600080fd5b886020828501011115610ea657600080fd5b95989497505060200194505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461097b57600080fd5b60008060408385031215610eea57600080fd5b610ef383610d00565b91506020830135610f0381610eb5565b809150509250929050565b600060208083528351808285015260005b81811015610f3b57858101830151858201604001528201610f1f565b81811115610f4d576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b600060208284031215610f9357600080fd5b5035919050565b600060208284031215610fac57600080fd5b8135610fb781610eb5565b9392505050565b600060208284031215610fd057600080fd5b610fb782610d00565b838152818360208301376000910160200190815292915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// DisputeGameFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DisputeGameFactoryMetaData.ABI instead.
var DisputeGameFactoryABI = DisputeGameFactoryMetaData.ABI

// DisputeGameFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DisputeGameFactoryMetaData.Bin instead.
var DisputeGameFactoryBin = DisputeGameFactoryMetaData.Bin

// DeployDisputeGameFactory deploys a new Ethereum contract, binding an instance of DisputeGameFactory to it.
func DeployDisputeGameFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DisputeGameFactory, error) {
	parsed, err := DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DisputeGameFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DisputeGameFactory{DisputeGameFactoryCaller: DisputeGameFactoryCaller{contract: contract}, DisputeGameFactoryTransactor: DisputeGameFactoryTransactor{contract: contract}, DisputeGameFactoryFilterer: DisputeGameFactoryFilterer{contract: contract}}, nil
}

// DisputeGameFactory is an auto generated Go binding around an Ethereum contract.
type DisputeGameFactory struct {
	DisputeGameFactoryCaller     // Read-only binding to the contract
	DisputeGameFactoryTransactor // Write-only binding to the contract
	DisputeGameFactoryFilterer   // Log filterer for contract events
}

// DisputeGameFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DisputeGameFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DisputeGameFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DisputeGameFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DisputeGameFactorySession struct {
	Contract     *DisputeGameFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// DisputeGameFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DisputeGameFactoryCallerSession struct {
	Contract *DisputeGameFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// DisputeGameFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DisputeGameFactoryTransactorSession struct {
	Contract     *DisputeGameFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// DisputeGameFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DisputeGameFactoryRaw struct {
	Contract *DisputeGameFactory // Generic contract binding to access the raw methods on
}

// DisputeGameFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DisputeGameFactoryCallerRaw struct {
	Contract *DisputeGameFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DisputeGameFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DisputeGameFactoryTransactorRaw struct {
	Contract *DisputeGameFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDisputeGameFactory creates a new instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactory(address common.Address, backend bind.ContractBackend) (*DisputeGameFactory, error) {
	contract, err := bindDisputeGameFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactory{DisputeGameFactoryCaller: DisputeGameFactoryCaller{contract: contract}, DisputeGameFactoryTransactor: DisputeGameFactoryTransactor{contract: contract}, DisputeGameFactoryFilterer: DisputeGameFactoryFilterer{contract: contract}}, nil
}

// NewDisputeGameFactoryCaller creates a new read-only instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryCaller(address common.Address, caller bind.ContractCaller) (*DisputeGameFactoryCaller, error) {
	contract, err := bindDisputeGameFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryCaller{contract: contract}, nil
}

// NewDisputeGameFactoryTransactor creates a new write-only instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DisputeGameFactoryTransactor, error) {
	contract, err := bindDisputeGameFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryTransactor{contract: contract}, nil
}

// NewDisputeGameFactoryFilterer creates a new log filterer instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DisputeGameFactoryFilterer, error) {
	contract, err := bindDisputeGameFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryFilterer{contract: contract}, nil
}

// bindDisputeGameFactory binds a generic wrapper to an already deployed contract.
func bindDisputeGameFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DisputeGameFactoryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DisputeGameFactory.Contract.DisputeGameFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.DisputeGameFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.DisputeGameFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DisputeGameFactory *DisputeGameFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DisputeGameFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DisputeGameFactory *DisputeGameFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DisputeGameFactory *DisputeGameFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.contract.Transact(opts, method, params...)
}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint8 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameAtIndex(opts *bind.CallOpts, _index *big.Int) (struct {
	GameType  uint8
	Timestamp uint64
	Proxy     common.Address
}, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameAtIndex", _index)

	outstruct := new(struct {
		GameType  uint8
		Timestamp uint64
		Proxy     common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.GameType = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Proxy = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint8 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactorySession) GameAtIndex(_index *big.Int) (struct {
	GameType  uint8
	Timestamp uint64
	Proxy     common.Address
}, error) {
	return _DisputeGameFactory.Contract.GameAtIndex(&_DisputeGameFactory.CallOpts, _index)
}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint8 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameAtIndex(_index *big.Int) (struct {
	GameType  uint8
	Timestamp uint64
	Proxy     common.Address
}, error) {
	return _DisputeGameFactory.Contract.GameAtIndex(&_DisputeGameFactory.CallOpts, _index)
}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactorySession) GameCount() (*big.Int, error) {
	return _DisputeGameFactory.Contract.GameCount(&_DisputeGameFactory.CallOpts)
}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameCount() (*big.Int, error) {
	return _DisputeGameFactory.Contract.GameCount(&_DisputeGameFactory.CallOpts)
}

// GameImpls is a free data retrieval call binding the contract method 0xdfa162d3.
//
// Solidity: function gameImpls(uint8 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameImpls(opts *bind.CallOpts, arg0 uint8) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameImpls", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GameImpls is a free data retrieval call binding the contract method 0xdfa162d3.
//
// Solidity: function gameImpls(uint8 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) GameImpls(arg0 uint8) (common.Address, error) {
	return _DisputeGameFactory.Contract.GameImpls(&_DisputeGameFactory.CallOpts, arg0)
}

// GameImpls is a free data retrieval call binding the contract method 0xdfa162d3.
//
// Solidity: function gameImpls(uint8 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameImpls(arg0 uint8) (common.Address, error) {
	return _DisputeGameFactory.Contract.GameImpls(&_DisputeGameFactory.CallOpts, arg0)
}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Games(opts *bind.CallOpts, _gameType uint8, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "games", _gameType, _rootClaim, _extraData)

	outstruct := new(struct {
		Proxy     common.Address
		Timestamp uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Proxy = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactorySession) Games(_gameType uint8, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	return _DisputeGameFactory.Contract.Games(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// Games is a free data retrieval call binding the contract method 0xc49d5271.
//
// Solidity: function games(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Games(_gameType uint8, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	return _DisputeGameFactory.Contract.Games(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// GetGameUUID is a free data retrieval call binding the contract method 0x26daafbe.
//
// Solidity: function getGameUUID(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GetGameUUID(opts *bind.CallOpts, _gameType uint8, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "getGameUUID", _gameType, _rootClaim, _extraData)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetGameUUID is a free data retrieval call binding the contract method 0x26daafbe.
//
// Solidity: function getGameUUID(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactorySession) GetGameUUID(_gameType uint8, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	return _DisputeGameFactory.Contract.GetGameUUID(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// GetGameUUID is a free data retrieval call binding the contract method 0x26daafbe.
//
// Solidity: function getGameUUID(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GetGameUUID(_gameType uint8, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	return _DisputeGameFactory.Contract.GetGameUUID(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) Owner() (common.Address, error) {
	return _DisputeGameFactory.Contract.Owner(&_DisputeGameFactory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Owner() (common.Address, error) {
	return _DisputeGameFactory.Contract.Owner(&_DisputeGameFactory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactorySession) Version() (string, error) {
	return _DisputeGameFactory.Contract.Version(&_DisputeGameFactory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Version() (string, error) {
	return _DisputeGameFactory.Contract.Version(&_DisputeGameFactory.CallOpts)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryTransactor) Create(opts *bind.TransactOpts, _gameType uint8, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "create", _gameType, _rootClaim, _extraData)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactorySession) Create(_gameType uint8, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Create(&_DisputeGameFactory.TransactOpts, _gameType, _rootClaim, _extraData)
}

// Create is a paid mutator transaction binding the contract method 0x3142e55e.
//
// Solidity: function create(uint8 _gameType, bytes32 _rootClaim, bytes _extraData) returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) Create(_gameType uint8, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Create(&_DisputeGameFactory.TransactOpts, _gameType, _rootClaim, _extraData)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "initialize", _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) Initialize(_owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Initialize(&_DisputeGameFactory.TransactOpts, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) Initialize(_owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Initialize(&_DisputeGameFactory.TransactOpts, _owner)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactorySession) RenounceOwnership() (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.RenounceOwnership(&_DisputeGameFactory.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.RenounceOwnership(&_DisputeGameFactory.TransactOpts)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x45583b7a.
//
// Solidity: function setImplementation(uint8 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) SetImplementation(opts *bind.TransactOpts, _gameType uint8, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "setImplementation", _gameType, _impl)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x45583b7a.
//
// Solidity: function setImplementation(uint8 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) SetImplementation(_gameType uint8, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation(&_DisputeGameFactory.TransactOpts, _gameType, _impl)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x45583b7a.
//
// Solidity: function setImplementation(uint8 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) SetImplementation(_gameType uint8, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation(&_DisputeGameFactory.TransactOpts, _gameType, _impl)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.TransferOwnership(&_DisputeGameFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.TransferOwnership(&_DisputeGameFactory.TransactOpts, newOwner)
}

// DisputeGameFactoryDisputeGameCreatedIterator is returned from FilterDisputeGameCreated and is used to iterate over the raw logs and unpacked data for DisputeGameCreated events raised by the DisputeGameFactory contract.
type DisputeGameFactoryDisputeGameCreatedIterator struct {
	Event *DisputeGameFactoryDisputeGameCreated // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryDisputeGameCreated)
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
		it.Event = new(DisputeGameFactoryDisputeGameCreated)
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
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryDisputeGameCreated represents a DisputeGameCreated event raised by the DisputeGameFactory contract.
type DisputeGameFactoryDisputeGameCreated struct {
	DisputeProxy common.Address
	GameType     uint8
	RootClaim    [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDisputeGameCreated is a free log retrieval operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterDisputeGameCreated(opts *bind.FilterOpts, disputeProxy []common.Address, gameType []uint8, rootClaim [][32]byte) (*DisputeGameFactoryDisputeGameCreatedIterator, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryDisputeGameCreatedIterator{contract: _DisputeGameFactory.contract, event: "DisputeGameCreated", logs: logs, sub: sub}, nil
}

// WatchDisputeGameCreated is a free log subscription operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchDisputeGameCreated(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryDisputeGameCreated, disputeProxy []common.Address, gameType []uint8, rootClaim [][32]byte) (event.Subscription, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryDisputeGameCreated)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
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

// ParseDisputeGameCreated is a log parse operation binding the contract event 0xfad0599ff449d8d9685eadecca8cb9e00924c5fd8367c1c09469824939e1ffec.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint8 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseDisputeGameCreated(log types.Log) (*DisputeGameFactoryDisputeGameCreated, error) {
	event := new(DisputeGameFactoryDisputeGameCreated)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryImplementationSetIterator is returned from FilterImplementationSet and is used to iterate over the raw logs and unpacked data for ImplementationSet events raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationSetIterator struct {
	Event *DisputeGameFactoryImplementationSet // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryImplementationSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryImplementationSet)
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
		it.Event = new(DisputeGameFactoryImplementationSet)
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
func (it *DisputeGameFactoryImplementationSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryImplementationSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryImplementationSet represents a ImplementationSet event raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationSet struct {
	Impl     common.Address
	GameType uint8
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterImplementationSet is a free log retrieval operation binding the contract event 0x623713f72f6e427a8044bb8b3bd6834357cf285decbaa21bcc73c1d0632c4d84.
//
// Solidity: event ImplementationSet(address indexed impl, uint8 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterImplementationSet(opts *bind.FilterOpts, impl []common.Address, gameType []uint8) (*DisputeGameFactoryImplementationSetIterator, error) {

	var implRule []interface{}
	for _, implItem := range impl {
		implRule = append(implRule, implItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "ImplementationSet", implRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryImplementationSetIterator{contract: _DisputeGameFactory.contract, event: "ImplementationSet", logs: logs, sub: sub}, nil
}

// WatchImplementationSet is a free log subscription operation binding the contract event 0x623713f72f6e427a8044bb8b3bd6834357cf285decbaa21bcc73c1d0632c4d84.
//
// Solidity: event ImplementationSet(address indexed impl, uint8 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchImplementationSet(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryImplementationSet, impl []common.Address, gameType []uint8) (event.Subscription, error) {

	var implRule []interface{}
	for _, implItem := range impl {
		implRule = append(implRule, implItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "ImplementationSet", implRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryImplementationSet)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationSet", log); err != nil {
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

// ParseImplementationSet is a log parse operation binding the contract event 0x623713f72f6e427a8044bb8b3bd6834357cf285decbaa21bcc73c1d0632c4d84.
//
// Solidity: event ImplementationSet(address indexed impl, uint8 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseImplementationSet(log types.Log) (*DisputeGameFactoryImplementationSet, error) {
	event := new(DisputeGameFactoryImplementationSet)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitializedIterator struct {
	Event *DisputeGameFactoryInitialized // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryInitialized)
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
		it.Event = new(DisputeGameFactoryInitialized)
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
func (it *DisputeGameFactoryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryInitialized represents a Initialized event raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterInitialized(opts *bind.FilterOpts) (*DisputeGameFactoryInitializedIterator, error) {

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryInitializedIterator{contract: _DisputeGameFactory.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryInitialized) (event.Subscription, error) {

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryInitialized)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseInitialized(log types.Log) (*DisputeGameFactoryInitialized, error) {
	event := new(DisputeGameFactoryInitialized)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DisputeGameFactory contract.
type DisputeGameFactoryOwnershipTransferredIterator struct {
	Event *DisputeGameFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryOwnershipTransferred)
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
		it.Event = new(DisputeGameFactoryOwnershipTransferred)
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
func (it *DisputeGameFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the DisputeGameFactory contract.
type DisputeGameFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DisputeGameFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryOwnershipTransferredIterator{contract: _DisputeGameFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryOwnershipTransferred)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*DisputeGameFactoryOwnershipTransferred, error) {
	event := new(DisputeGameFactoryOwnershipTransferred)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
