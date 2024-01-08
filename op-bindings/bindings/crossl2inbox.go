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

// InboxMessages is an auto generated low-level Go binding around an user-defined struct.
type InboxMessages struct {
	Chain        [32]byte
	Output       [32]byte
	BlockNumber  *big.Int
	MessageRoots [][32]byte
}

// TypesSuperchainMessage is an auto generated low-level Go binding around an user-defined struct.
type TypesSuperchainMessage struct {
	Nonce       *big.Int
	SourceChain [32]byte
	TargetChain [32]byte
	From        common.Address
	To          common.Address
	Value       *big.Int
	GasLimit    *big.Int
	Data        []byte
}

// CrossL2InboxMetaData contains all meta data concerning the CrossL2Inbox contract.
var CrossL2InboxMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_postie_address\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"CrossL2MessageRelayed\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"chainState\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.SuperchainMessage\",\"name\":\"_msg\",\"type\":\"tuple\"}],\"name\":\"consumeMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"crossL2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"chain\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"messageRoots\",\"type\":\"bytes32[]\"}],\"internalType\":\"structInboxMessages[]\",\"name\":\"mail\",\"type\":\"tuple[]\"}],\"name\":\"deliverMessages\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageSourceChain\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"unconsumedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60a0604052600380546001600160a01b03191661dead17905534801561002457600080fd5b5060405161100238038061100283398101604081905261004391610054565b6001600160a01b0316608052610084565b60006020828403121561006657600080fd5b81516001600160a01b038116811461007d57600080fd5b9392505050565b608051610f6361009f600039600061053f0152610f636000f3fe60806040526004361061007b5760003560e01c8063c220426b1161004e578063c220426b1461019a578063db10b9a9146101da578063e5507be314610212578063e63c390c1461023457600080fd5b80632cfea9fe146100805780633d6d0dd4146100ce57806354fd4d501461012057806360a687aa14610176575b600080fd5b34801561008c57600080fd5b506100b461009b366004610b09565b6002602052600090815260409020805460019091015482565b604080519283526020830191909152015b60405180910390f35b3480156100da57600080fd5b506003546100fb9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100c5565b34801561012c57600080fd5b506101696040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b6040516100c59190610b22565b34801561018257600080fd5b5061018c60045481565b6040519081526020016100c5565b3480156101a657600080fd5b506101ca6101b5366004610b09565b60016020526000908152604090205460ff1681565b60405190151581526020016100c5565b3480156101e657600080fd5b506101ca6101f5366004610b95565b600060208181529281526040808220909352908152205460ff1681565b34801561021e57600080fd5b5061023261022d366004610ce4565b610247565b005b610232610242366004610da6565b610527565b604081015146146102df576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f43726f73734c32496e626f783a2074617267657420636861696e20646f65732060448201527f6e6f74206d61746368000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b60006102ea826108f9565b60008181526001602052604090205490915060ff16610365576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f43726f73734c32496e626f783a20756e6b6e6f776e206d65737361676500000060448201526064016102d6565b60035473ffffffffffffffffffffffffffffffffffffffff1661dead1461040e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f43726f73734c32496e626f783a2063616e206f6e6c792074726967676572206f60448201527f6e652063616c6c20706572206d6573736167650000000000000000000000000060648201526084016102d6565b8160600151600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508160200151600481905550600061047c83608001518460c001518560a001518660e00151610a8d565b600380547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead17905560006004819055838152600160205260409081902080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001690555190915082907f608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d9061051a90841515815260200190565b60405180910390a2505050565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016146105ec576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f43726f73734c32496e626f783a206f6e6c7920706f737469652063616e20646560448201527f6c69766572206d61696c0000000000000000000000000000000000000000000060648201526084016102d6565b60005b818110156108f4576002600084848481811061060d5761060d610e1b565b905060200281019061061f9190610e4a565b6000013581526020019081526020016000206001015483838381811061064757610647610e1b565b90506020028101906106599190610e4a565b60400135116106ea576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f43726f73734c32496e626f783a20626c6f636b4e756d626572206d757374206260448201527f6520696e6372656173696e67000000000000000000000000000000000000000060648201526084016102d6565b604051806040016040528084848481811061070757610707610e1b565b90506020028101906107199190610e4a565b60200135815260200184848481811061073457610734610e1b565b90506020028101906107469190610e4a565b6040013590526002600085858581811061076257610762610e1b565b90506020028101906107749190610e4a565b3581526020808201929092526040016000908120835181559290910151600192830155808585858181106107aa576107aa610e1b565b90506020028101906107bc9190610e4a565b60000135815260200190815260200160002060008585858181106107e2576107e2610e1b565b90506020028101906107f49190610e4a565b60200135815260200190815260200160002060006101000a81548160ff02191690831515021790555060005b83838381811061083257610832610e1b565b90506020028101906108449190610e4a565b610852906060810190610e88565b90508110156108e157600180600086868681811061087257610872610e1b565b90506020028101906108849190610e4a565b610892906060810190610e88565b858181106108a2576108a2610e1b565b90506020020135815260200190815260200160002060006101000a81548160ff02191690831515021790555080806108d990610ef7565b915050610820565b50806108ec81610ef7565b9150506105ef565b505050565b60e0810151805160209182018190206040805193840182905283019190915260009182906060016040516020818303038152906040528051906020012090506000846060015185608001518660a001518760c00151604051602001610992949392919073ffffffffffffffffffffffffffffffffffffffff94851681529290931660208301526040820152606081019190915260800190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152828252805160209182012081840152828201949094528051808303820181526060830182528051908501208785015197820151608084019890985260a0808401989098528151808403909801885260c08301825287519785019790972060e0830152610100808301979097528051808303909701875261012090910190525083519301929092207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001792915050565b6000806000610a9d866000610aeb565b905080610ad3576308c379a06000526020805278185361666543616c6c3a204e6f7420656e6f756768206761736058526064601cfd5b600080855160208701888b5af1979650505050505050565b600080603f83619c4001026040850201603f5a021015949350505050565b600060208284031215610b1b57600080fd5b5035919050565b600060208083528351808285015260005b81811015610b4f57858101830151858201604001528201610b33565b81811115610b61576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60008060408385031215610ba857600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051610100810167ffffffffffffffff81118282101715610c0a57610c0a610bb7565b60405290565b803573ffffffffffffffffffffffffffffffffffffffff81168114610c3457600080fd5b919050565b600082601f830112610c4a57600080fd5b813567ffffffffffffffff80821115610c6557610c65610bb7565b604051601f83017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908282118183101715610cab57610cab610bb7565b81604052838152866020858801011115610cc457600080fd5b836020870160208301376000602085830101528094505050505092915050565b600060208284031215610cf657600080fd5b813567ffffffffffffffff80821115610d0e57600080fd5b908301906101008286031215610d2357600080fd5b610d2b610be6565b823581526020830135602082015260408301356040820152610d4f60608401610c10565b6060820152610d6060808401610c10565b608082015260a083013560a082015260c083013560c082015260e083013582811115610d8b57600080fd5b610d9787828601610c39565b60e08301525095945050505050565b60008060208385031215610db957600080fd5b823567ffffffffffffffff80821115610dd157600080fd5b818501915085601f830112610de557600080fd5b813581811115610df457600080fd5b8660208260051b8501011115610e0957600080fd5b60209290920196919550909350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81833603018112610e7e57600080fd5b9190910192915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610ebd57600080fd5b83018035915067ffffffffffffffff821115610ed857600080fd5b6020019150600581901b3603821315610ef057600080fd5b9250929050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610f4f577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b506001019056fea164736f6c634300080f000a",
}

// CrossL2InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2InboxMetaData.ABI instead.
var CrossL2InboxABI = CrossL2InboxMetaData.ABI

// CrossL2InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2InboxMetaData.Bin instead.
var CrossL2InboxBin = CrossL2InboxMetaData.Bin

// DeployCrossL2Inbox deploys a new Ethereum contract, binding an instance of CrossL2Inbox to it.
func DeployCrossL2Inbox(auth *bind.TransactOpts, backend bind.ContractBackend, _postie_address common.Address) (common.Address, *types.Transaction, *CrossL2Inbox, error) {
	parsed, err := CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2InboxBin), backend, _postie_address)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// CrossL2Inbox is an auto generated Go binding around an Ethereum contract.
type CrossL2Inbox struct {
	CrossL2InboxCaller     // Read-only binding to the contract
	CrossL2InboxTransactor // Write-only binding to the contract
	CrossL2InboxFilterer   // Log filterer for contract events
}

// CrossL2InboxCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrossL2InboxCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrossL2InboxFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrossL2InboxSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrossL2InboxSession struct {
	Contract     *CrossL2Inbox     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CrossL2InboxCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrossL2InboxCallerSession struct {
	Contract *CrossL2InboxCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// CrossL2InboxTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrossL2InboxTransactorSession struct {
	Contract     *CrossL2InboxTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CrossL2InboxRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrossL2InboxRaw struct {
	Contract *CrossL2Inbox // Generic contract binding to access the raw methods on
}

// CrossL2InboxCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrossL2InboxCallerRaw struct {
	Contract *CrossL2InboxCaller // Generic read-only contract binding to access the raw methods on
}

// CrossL2InboxTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrossL2InboxTransactorRaw struct {
	Contract *CrossL2InboxTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrossL2Inbox creates a new instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2Inbox(address common.Address, backend bind.ContractBackend) (*CrossL2Inbox, error) {
	contract, err := bindCrossL2Inbox(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CrossL2Inbox{CrossL2InboxCaller: CrossL2InboxCaller{contract: contract}, CrossL2InboxTransactor: CrossL2InboxTransactor{contract: contract}, CrossL2InboxFilterer: CrossL2InboxFilterer{contract: contract}}, nil
}

// NewCrossL2InboxCaller creates a new read-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxCaller(address common.Address, caller bind.ContractCaller) (*CrossL2InboxCaller, error) {
	contract, err := bindCrossL2Inbox(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxCaller{contract: contract}, nil
}

// NewCrossL2InboxTransactor creates a new write-only instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxTransactor(address common.Address, transactor bind.ContractTransactor) (*CrossL2InboxTransactor, error) {
	contract, err := bindCrossL2Inbox(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxTransactor{contract: contract}, nil
}

// NewCrossL2InboxFilterer creates a new log filterer instance of CrossL2Inbox, bound to a specific deployed contract.
func NewCrossL2InboxFilterer(address common.Address, filterer bind.ContractFilterer) (*CrossL2InboxFilterer, error) {
	contract, err := bindCrossL2Inbox(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxFilterer{contract: contract}, nil
}

// bindCrossL2Inbox binds a generic wrapper to an already deployed contract.
func bindCrossL2Inbox(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrossL2InboxABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.CrossL2InboxCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.CrossL2InboxTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CrossL2Inbox *CrossL2InboxCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CrossL2Inbox.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CrossL2Inbox *CrossL2InboxTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.contract.Transact(opts, method, params...)
}

// ChainState is a free data retrieval call binding the contract method 0x2cfea9fe.
//
// Solidity: function chainState(bytes32 ) view returns(bytes32 output, uint256 blockNumber)
func (_CrossL2Inbox *CrossL2InboxCaller) ChainState(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Output      [32]byte
	BlockNumber *big.Int
}, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "chainState", arg0)

	outstruct := new(struct {
		Output      [32]byte
		BlockNumber *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Output = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.BlockNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ChainState is a free data retrieval call binding the contract method 0x2cfea9fe.
//
// Solidity: function chainState(bytes32 ) view returns(bytes32 output, uint256 blockNumber)
func (_CrossL2Inbox *CrossL2InboxSession) ChainState(arg0 [32]byte) (struct {
	Output      [32]byte
	BlockNumber *big.Int
}, error) {
	return _CrossL2Inbox.Contract.ChainState(&_CrossL2Inbox.CallOpts, arg0)
}

// ChainState is a free data retrieval call binding the contract method 0x2cfea9fe.
//
// Solidity: function chainState(bytes32 ) view returns(bytes32 output, uint256 blockNumber)
func (_CrossL2Inbox *CrossL2InboxCallerSession) ChainState(arg0 [32]byte) (struct {
	Output      [32]byte
	BlockNumber *big.Int
}, error) {
	return _CrossL2Inbox.Contract.ChainState(&_CrossL2Inbox.CallOpts, arg0)
}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) CrossL2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "crossL2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) CrossL2Sender() (common.Address, error) {
	return _CrossL2Inbox.Contract.CrossL2Sender(&_CrossL2Inbox.CallOpts)
}

// CrossL2Sender is a free data retrieval call binding the contract method 0x3d6d0dd4.
//
// Solidity: function crossL2Sender() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) CrossL2Sender() (common.Address, error) {
	return _CrossL2Inbox.Contract.CrossL2Sender(&_CrossL2Inbox.CallOpts)
}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCaller) MessageSourceChain(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "messageSourceChain")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxSession) MessageSourceChain() ([32]byte, error) {
	return _CrossL2Inbox.Contract.MessageSourceChain(&_CrossL2Inbox.CallOpts)
}

// MessageSourceChain is a free data retrieval call binding the contract method 0x60a687aa.
//
// Solidity: function messageSourceChain() view returns(bytes32)
func (_CrossL2Inbox *CrossL2InboxCallerSession) MessageSourceChain() ([32]byte, error) {
	return _CrossL2Inbox.Contract.MessageSourceChain(&_CrossL2Inbox.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCaller) Roots(opts *bind.CallOpts, arg0 [32]byte, arg1 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "roots", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// Roots is a free data retrieval call binding the contract method 0xdb10b9a9.
//
// Solidity: function roots(bytes32 , bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Roots(arg0 [32]byte, arg1 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.Roots(&_CrossL2Inbox.CallOpts, arg0, arg1)
}

// UnconsumedMessages is a free data retrieval call binding the contract method 0xc220426b.
//
// Solidity: function unconsumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCaller) UnconsumedMessages(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "unconsumedMessages", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// UnconsumedMessages is a free data retrieval call binding the contract method 0xc220426b.
//
// Solidity: function unconsumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxSession) UnconsumedMessages(arg0 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.UnconsumedMessages(&_CrossL2Inbox.CallOpts, arg0)
}

// UnconsumedMessages is a free data retrieval call binding the contract method 0xc220426b.
//
// Solidity: function unconsumedMessages(bytes32 ) view returns(bool)
func (_CrossL2Inbox *CrossL2InboxCallerSession) UnconsumedMessages(arg0 [32]byte) (bool, error) {
	return _CrossL2Inbox.Contract.UnconsumedMessages(&_CrossL2Inbox.CallOpts, arg0)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_CrossL2Inbox *CrossL2InboxCallerSession) Version() (string, error) {
	return _CrossL2Inbox.Contract.Version(&_CrossL2Inbox.CallOpts)
}

// ConsumeMessage is a paid mutator transaction binding the contract method 0xe5507be3.
//
// Solidity: function consumeMessage((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg) returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) ConsumeMessage(opts *bind.TransactOpts, _msg TypesSuperchainMessage) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "consumeMessage", _msg)
}

// ConsumeMessage is a paid mutator transaction binding the contract method 0xe5507be3.
//
// Solidity: function consumeMessage((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg) returns()
func (_CrossL2Inbox *CrossL2InboxSession) ConsumeMessage(_msg TypesSuperchainMessage) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.ConsumeMessage(&_CrossL2Inbox.TransactOpts, _msg)
}

// ConsumeMessage is a paid mutator transaction binding the contract method 0xe5507be3.
//
// Solidity: function consumeMessage((uint256,bytes32,bytes32,address,address,uint256,uint256,bytes) _msg) returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) ConsumeMessage(_msg TypesSuperchainMessage) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.ConsumeMessage(&_CrossL2Inbox.TransactOpts, _msg)
}

// DeliverMessages is a paid mutator transaction binding the contract method 0xe63c390c.
//
// Solidity: function deliverMessages((bytes32,bytes32,uint256,bytes32[])[] mail) payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactor) DeliverMessages(opts *bind.TransactOpts, mail []InboxMessages) (*types.Transaction, error) {
	return _CrossL2Inbox.contract.Transact(opts, "deliverMessages", mail)
}

// DeliverMessages is a paid mutator transaction binding the contract method 0xe63c390c.
//
// Solidity: function deliverMessages((bytes32,bytes32,uint256,bytes32[])[] mail) payable returns()
func (_CrossL2Inbox *CrossL2InboxSession) DeliverMessages(mail []InboxMessages) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMessages(&_CrossL2Inbox.TransactOpts, mail)
}

// DeliverMessages is a paid mutator transaction binding the contract method 0xe63c390c.
//
// Solidity: function deliverMessages((bytes32,bytes32,uint256,bytes32[])[] mail) payable returns()
func (_CrossL2Inbox *CrossL2InboxTransactorSession) DeliverMessages(mail []InboxMessages) (*types.Transaction, error) {
	return _CrossL2Inbox.Contract.DeliverMessages(&_CrossL2Inbox.TransactOpts, mail)
}

// CrossL2InboxCrossL2MessageRelayedIterator is returned from FilterCrossL2MessageRelayed and is used to iterate over the raw logs and unpacked data for CrossL2MessageRelayed events raised by the CrossL2Inbox contract.
type CrossL2InboxCrossL2MessageRelayedIterator struct {
	Event *CrossL2InboxCrossL2MessageRelayed // Event containing the contract specifics and raw log

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
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrossL2InboxCrossL2MessageRelayed)
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
		it.Event = new(CrossL2InboxCrossL2MessageRelayed)
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
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrossL2InboxCrossL2MessageRelayedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrossL2InboxCrossL2MessageRelayed represents a CrossL2MessageRelayed event raised by the CrossL2Inbox contract.
type CrossL2InboxCrossL2MessageRelayed struct {
	MessageRoot [32]byte
	Success     bool
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterCrossL2MessageRelayed is a free log retrieval operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) FilterCrossL2MessageRelayed(opts *bind.FilterOpts, messageRoot [][32]byte) (*CrossL2InboxCrossL2MessageRelayedIterator, error) {

	var messageRootRule []interface{}
	for _, messageRootItem := range messageRoot {
		messageRootRule = append(messageRootRule, messageRootItem)
	}

	logs, sub, err := _CrossL2Inbox.contract.FilterLogs(opts, "CrossL2MessageRelayed", messageRootRule)
	if err != nil {
		return nil, err
	}
	return &CrossL2InboxCrossL2MessageRelayedIterator{contract: _CrossL2Inbox.contract, event: "CrossL2MessageRelayed", logs: logs, sub: sub}, nil
}

// WatchCrossL2MessageRelayed is a free log subscription operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) WatchCrossL2MessageRelayed(opts *bind.WatchOpts, sink chan<- *CrossL2InboxCrossL2MessageRelayed, messageRoot [][32]byte) (event.Subscription, error) {

	var messageRootRule []interface{}
	for _, messageRootItem := range messageRoot {
		messageRootRule = append(messageRootRule, messageRootItem)
	}

	logs, sub, err := _CrossL2Inbox.contract.WatchLogs(opts, "CrossL2MessageRelayed", messageRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrossL2InboxCrossL2MessageRelayed)
				if err := _CrossL2Inbox.contract.UnpackLog(event, "CrossL2MessageRelayed", log); err != nil {
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

// ParseCrossL2MessageRelayed is a log parse operation binding the contract event 0x608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d.
//
// Solidity: event CrossL2MessageRelayed(bytes32 indexed messageRoot, bool success)
func (_CrossL2Inbox *CrossL2InboxFilterer) ParseCrossL2MessageRelayed(log types.Log) (*CrossL2InboxCrossL2MessageRelayed, error) {
	event := new(CrossL2InboxCrossL2MessageRelayed)
	if err := _CrossL2Inbox.contract.UnpackLog(event, "CrossL2MessageRelayed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
