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
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageRoot\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"CrossL2MessageRelayed\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEPOSITOR_ACCOUNT\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"chainState\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"sourceChain\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"targetChain\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.SuperchainMessage\",\"name\":\"_msg\",\"type\":\"tuple\"}],\"name\":\"consumeMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"crossL2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"chain\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"messageRoots\",\"type\":\"bytes32[]\"}],\"internalType\":\"structInboxMessages[]\",\"name\":\"mail\",\"type\":\"tuple[]\"}],\"name\":\"deliverMessages\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageSourceChain\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"unconsumedMessages\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6080604052600380546001600160a01b03191661dead17905534801561002457600080fd5b50610f88806100346000396000f3fe6080604052600436106100965760003560e01c8063c220426b11610069578063e5507be31161004e578063e5507be31461022d578063e591b2821461024f578063e63c390c1461027757600080fd5b8063c220426b146101b5578063db10b9a9146101f557600080fd5b80632cfea9fe1461009b5780633d6d0dd4146100e957806354fd4d501461013b57806360a687aa14610191575b600080fd5b3480156100a757600080fd5b506100cf6100b6366004610b2e565b6002602052600090815260409020805460019091015482565b604080519283526020830191909152015b60405180910390f35b3480156100f557600080fd5b506003546101169073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016100e0565b34801561014757600080fd5b506101846040518060400160405280600581526020017f302e302e3100000000000000000000000000000000000000000000000000000081525081565b6040516100e09190610b47565b34801561019d57600080fd5b506101a760045481565b6040519081526020016100e0565b3480156101c157600080fd5b506101e56101d0366004610b2e565b60016020526000908152604090205460ff1681565b60405190151581526020016100e0565b34801561020157600080fd5b506101e5610210366004610bba565b600060208181529281526040808220909352908152205460ff1681565b34801561023957600080fd5b5061024d610248366004610d09565b61028a565b005b34801561025b57600080fd5b5061011673deaddeaddeaddeaddeaddeaddeaddeaddead000281565b61024d610285366004610dcb565b61056e565b60408101514614610322576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f43726f73734c32496e626f783a2074617267657420636861696e20646f65732060448201527f6e6f74206d61746368000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b600061032d8261091e565b60008181526001602052604090205490915060ff166103a8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f43726f73734c32496e626f783a20756e6b6e6f776e206d6573736167650000006044820152606401610319565b60035473ffffffffffffffffffffffffffffffffffffffff1661dead14610451576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f43726f73734c32496e626f783a2063616e206f6e6c792074726967676572206f60448201527f6e652063616c6c20706572206d657373616765000000000000000000000000006064820152608401610319565b6060820151600380547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff909216919091179055602080830151600455600082815260019091526040812080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00169055608083015160c084015160a085015160e08601516104f793929190610ab2565b600380547fffffffffffffffffffffffff00000000000000000000000000000000000000001661dead179055600060045560405190915082907f608b51d991a28926c87c94dae8c72df6a62c5f22b359bb418bf204355b39fa7d9061056190841515815260200190565b60405180910390a2505050565b3373deaddeaddeaddeaddeaddeaddeaddeaddead000214610611576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f43726f73734c32496e626f783a206f6e6c7920706f737469652063616e20646560448201527f6c69766572206d61696c000000000000000000000000000000000000000000006064820152608401610319565b60005b81811015610919576002600084848481811061063257610632610e40565b90506020028101906106449190610e6f565b6000013581526020019081526020016000206001015483838381811061066c5761066c610e40565b905060200281019061067e9190610e6f565b604001351161070f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f43726f73734c32496e626f783a20626c6f636b4e756d626572206d757374206260448201527f6520696e6372656173696e6700000000000000000000000000000000000000006064820152608401610319565b604051806040016040528084848481811061072c5761072c610e40565b905060200281019061073e9190610e6f565b60200135815260200184848481811061075957610759610e40565b905060200281019061076b9190610e6f565b6040013590526002600085858581811061078757610787610e40565b90506020028101906107999190610e6f565b3581526020808201929092526040016000908120835181559290910151600192830155808585858181106107cf576107cf610e40565b90506020028101906107e19190610e6f565b600001358152602001908152602001600020600085858581811061080757610807610e40565b90506020028101906108199190610e6f565b60200135815260200190815260200160002060006101000a81548160ff02191690831515021790555060005b83838381811061085757610857610e40565b90506020028101906108699190610e6f565b610877906060810190610ead565b905081101561090657600180600086868681811061089757610897610e40565b90506020028101906108a99190610e6f565b6108b7906060810190610ead565b858181106108c7576108c7610e40565b90506020020135815260200190815260200160002060006101000a81548160ff02191690831515021790555080806108fe90610f1c565b915050610845565b508061091181610f1c565b915050610614565b505050565b60e0810151805160209182018190206040805193840182905283019190915260009182906060016040516020818303038152906040528051906020012090506000846060015185608001518660a001518760c001516040516020016109b7949392919073ffffffffffffffffffffffffffffffffffffffff94851681529290931660208301526040820152606081019190915260800190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0818403018152828252805160209182012081840152828201949094528051808303820181526060830182528051908501208785015197820151608084019890985260a0808401989098528151808403909801885260c08301825287519785019790972060e0830152610100808301979097528051808303909701875261012090910190525083519301929092207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001792915050565b6000806000610ac2866000610b10565b905080610af8576308c379a06000526020805278185361666543616c6c3a204e6f7420656e6f756768206761736058526064601cfd5b600080855160208701888b5af1979650505050505050565b600080603f83619c4001026040850201603f5a021015949350505050565b600060208284031215610b4057600080fd5b5035919050565b600060208083528351808285015260005b81811015610b7457858101830151858201604001528201610b58565b81811115610b86576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60008060408385031215610bcd57600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051610100810167ffffffffffffffff81118282101715610c2f57610c2f610bdc565b60405290565b803573ffffffffffffffffffffffffffffffffffffffff81168114610c5957600080fd5b919050565b600082601f830112610c6f57600080fd5b813567ffffffffffffffff80821115610c8a57610c8a610bdc565b604051601f83017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908282118183101715610cd057610cd0610bdc565b81604052838152866020858801011115610ce957600080fd5b836020870160208301376000602085830101528094505050505092915050565b600060208284031215610d1b57600080fd5b813567ffffffffffffffff80821115610d3357600080fd5b908301906101008286031215610d4857600080fd5b610d50610c0b565b823581526020830135602082015260408301356040820152610d7460608401610c35565b6060820152610d8560808401610c35565b608082015260a083013560a082015260c083013560c082015260e083013582811115610db057600080fd5b610dbc87828601610c5e565b60e08301525095945050505050565b60008060208385031215610dde57600080fd5b823567ffffffffffffffff80821115610df657600080fd5b818501915085601f830112610e0a57600080fd5b813581811115610e1957600080fd5b8660208260051b8501011115610e2e57600080fd5b60209290920196919550909350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81833603018112610ea357600080fd5b9190910192915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112610ee257600080fd5b83018035915067ffffffffffffffff821115610efd57600080fd5b6020019150600581901b3603821315610f1557600080fd5b9250929050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610f74577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b506001019056fea164736f6c634300080f000a",
}

// CrossL2InboxABI is the input ABI used to generate the binding from.
// Deprecated: Use CrossL2InboxMetaData.ABI instead.
var CrossL2InboxABI = CrossL2InboxMetaData.ABI

// CrossL2InboxBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrossL2InboxMetaData.Bin instead.
var CrossL2InboxBin = CrossL2InboxMetaData.Bin

// DeployCrossL2Inbox deploys a new Ethereum contract, binding an instance of CrossL2Inbox to it.
func DeployCrossL2Inbox(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *CrossL2Inbox, error) {
	parsed, err := CrossL2InboxMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrossL2InboxBin), backend)
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

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCaller) DEPOSITORACCOUNT(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _CrossL2Inbox.contract.Call(opts, &out, "DEPOSITOR_ACCOUNT")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_CrossL2Inbox *CrossL2InboxSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _CrossL2Inbox.Contract.DEPOSITORACCOUNT(&_CrossL2Inbox.CallOpts)
}

// DEPOSITORACCOUNT is a free data retrieval call binding the contract method 0xe591b282.
//
// Solidity: function DEPOSITOR_ACCOUNT() view returns(address)
func (_CrossL2Inbox *CrossL2InboxCallerSession) DEPOSITORACCOUNT() (common.Address, error) {
	return _CrossL2Inbox.Contract.DEPOSITORACCOUNT(&_CrossL2Inbox.CallOpts)
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
